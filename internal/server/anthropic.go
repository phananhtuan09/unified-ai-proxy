package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/tuanp-github/unified-ai-proxy/internal/apierr"
	"github.com/tuanp-github/unified-ai-proxy/internal/model"
)

// anthropicAllowed are the MVP-supported /v1/messages fields.
var anthropicAllowed = map[string]bool{
	"model": true, "messages": true, "system": true, "stream": true,
	"temperature": true, "top_p": true, "max_tokens": true,
	"stop_sequences": true, "metadata": true,
}

type anthropicContentBlock struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type anthropicMessage struct {
	Role    string                  `json:"role"`
	Content []anthropicContentBlock `json:"content"`
}

type anthropicMessagesRequest struct {
	Model         string             `json:"model"`
	Messages      []anthropicMessage `json:"messages"`
	System        json.RawMessage    `json:"system"`
	Stream        bool               `json:"stream"`
	Temperature   *float64           `json:"temperature"`
	TopP          *float64           `json:"top_p"`
	MaxTokens     int                `json:"max_tokens"`
	StopSequences []string           `json:"stop_sequences"`
	Metadata      map[string]any     `json:"metadata"`
}

func (s *Server) handleMessages(c *gin.Context) {
	body, err := c.GetRawData()
	if err != nil {
		writeAnthropicError(c, apierr.InvalidRequest("failed to read request body"))
		return
	}

	req, err := parseAnthropicMessagesRequest(body)
	if err != nil {
		writeAnthropicError(c, asAPIError(err))
		return
	}

	normalized, err := anthropicToNormalized(req)
	if err != nil {
		writeAnthropicError(c, asAPIError(err))
		return
	}
	if normalized.MaxTokens == 0 {
		normalized.MaxTokens = s.cfg.Server.DefaultMaxTokens
	}

	alias := req.Model
	if normalized.Stream {
		ch, err := s.svc.Stream(c.Request.Context(), normalized)
		if err != nil {
			writeAnthropicError(c, asAPIError(err))
			return
		}
		s.writeAnthropicStream(c, alias, ch)
		return
	}

	resp, err := s.svc.Chat(c.Request.Context(), normalized)
	if err != nil {
		writeAnthropicError(c, asAPIError(err))
		return
	}
	writeAnthropicMessageResponse(c, alias, resp)
}

func parseAnthropicMessagesRequest(body []byte) (*anthropicMessagesRequest, error) {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, apierr.InvalidRequest("request body is malformed")
	}
	for key := range raw {
		if anthropicAllowed[key] {
			continue
		}
		return nil, apierr.UnsupportedField("unsupported field: " + key)
	}
	var req anthropicMessagesRequest
	if err := json.Unmarshal(body, &req); err != nil {
		return nil, apierr.InvalidRequest("request body is malformed")
	}
	return &req, nil
}

func anthropicToNormalized(req *anthropicMessagesRequest) (*model.ChatRequest, error) {
	if strings.TrimSpace(req.Model) == "" {
		return nil, apierr.InvalidRequest("model is required")
	}
	if len(req.Messages) == 0 {
		return nil, apierr.InvalidRequest("messages must not be empty")
	}

	out := &model.ChatRequest{
		Model:         req.Model,
		Stream:        req.Stream,
		Temperature:   req.Temperature,
		TopP:          req.TopP,
		MaxTokens:     req.MaxTokens,
		StopSequences: req.StopSequences,
		Metadata:      req.Metadata,
	}

	if len(req.System) > 0 {
		sys, err := decodeAnthropicSystem(req.System)
		if err != nil {
			return nil, err
		}
		out.System = sys
	}

	for _, m := range req.Messages {
		switch m.Role {
		case "user", "assistant":
		default:
			return nil, apierr.InvalidRequest("invalid message role: " + m.Role)
		}
		var text strings.Builder
		for _, block := range m.Content {
			if block.Type != "text" {
				return nil, apierr.UnsupportedField("unsupported content block type: " + block.Type)
			}
			text.WriteString(block.Text)
		}
		out.Messages = append(out.Messages, model.Message{Role: model.Role(m.Role), Content: text.String()})
	}
	return out, nil
}

func decodeAnthropicSystem(raw json.RawMessage) (string, error) {
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		return s, nil
	}
	var blocks []anthropicContentBlock
	if err := json.Unmarshal(raw, &blocks); err != nil {
		return "", apierr.InvalidRequest("system must be a string or text blocks")
	}
	var b strings.Builder
	for _, block := range blocks {
		if block.Type != "text" {
			return "", apierr.UnsupportedField("unsupported system content block type: " + block.Type)
		}
		b.WriteString(block.Text)
	}
	return b.String(), nil
}

func writeAnthropicMessageResponse(c *gin.Context, alias string, resp *model.ChatResponse) {
	id := resp.ID
	if id == "" {
		id = "msg_" + randomID()
	}
	c.JSON(http.StatusOK, gin.H{
		"id":            id,
		"type":          "message",
		"role":          "assistant",
		"content":       []gin.H{{"type": "text", "text": resp.Content}},
		"model":         alias,
		"stop_reason":   anthropicStopReason(resp.StopReason),
		"stop_sequence": nil,
		"usage": gin.H{
			"input_tokens":  resp.Usage.InputTokens,
			"output_tokens": resp.Usage.OutputTokens,
		},
	})
}

func anthropicStopReason(reason string) string {
	switch reason {
	case "max_tokens", "max_output_tokens", "length":
		return "max_tokens"
	case "stop_sequence":
		return "stop_sequence"
	case "tool_use":
		return "tool_use"
	default:
		return "end_turn"
	}
}

func (s *Server) writeAnthropicStream(c *gin.Context, alias string, ch <-chan model.StreamEvent) {
	header := c.Writer.Header()
	header.Set("Content-Type", "text/event-stream")
	header.Set("Cache-Control", "no-cache")
	header.Set("Connection", "keep-alive")
	header.Set("X-Accel-Buffering", "no")
	c.Writer.WriteHeader(http.StatusOK)

	flusher := c.Writer.(http.Flusher)
	id := "msg_" + randomID()
	index := 0
	started := false

	writeEvent := func(event string, data gin.H) {
		fmt.Fprintf(c.Writer, "event: %s\n", event)
		payload, _ := json.Marshal(data)
		fmt.Fprintf(c.Writer, "data: %s\n\n", payload)
		flusher.Flush()
	}

	startMessage := func() {
		writeEvent("message_start", gin.H{
			"type": "message_start",
			"message": gin.H{
				"id":      id,
				"type":    "message",
				"role":    "assistant",
				"content": []gin.H{},
				"model":   alias,
			},
		})
		writeEvent("content_block_start", gin.H{
			"type":          "content_block_start",
			"index":         0,
			"content_block": gin.H{"type": "text", "text": ""},
		})
		started = true
	}

	for ev := range ch {
		select {
		case <-c.Request.Context().Done():
			return
		default:
		}

		switch ev.Type {
		case model.StreamMessageStart:
			if ev.ID != "" {
				id = ev.ID
			}
			if !started {
				startMessage()
			}
		case model.StreamContentDelta:
			if !started {
				startMessage()
			}
			writeEvent("content_block_delta", gin.H{
				"type":  "content_block_delta",
				"index": index,
				"delta": gin.H{"type": "text_delta", "text": ev.Content},
			})
		case model.StreamMessageStop:
			if !started {
				startMessage()
			}
			writeEvent("content_block_stop", gin.H{"type": "content_block_stop", "index": index})
			usage := ev.Usage
			if usage == nil {
				usage = &model.Usage{}
			}
			writeEvent("message_delta", gin.H{
				"type":  "message_delta",
				"delta": gin.H{"stop_reason": anthropicStopReason(ev.StopReason), "stop_sequence": nil},
				"usage": gin.H{"output_tokens": usage.OutputTokens},
			})
			writeEvent("message_stop", gin.H{"type": "message_stop"})
		case model.StreamError:
			if ev.Error != nil {
				writeEvent("error", gin.H{
					"type":  "error",
					"error": gin.H{"type": "api_error", "message": ev.Error.Error()},
				})
			}
			return
		}
	}
}

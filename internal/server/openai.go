package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/tuanp-github/unified-ai-proxy/internal/apierr"
	"github.com/tuanp-github/unified-ai-proxy/internal/model"
)

// openAIAllowed are the MVP-supported /v1/chat/completions fields.
var openAIAllowed = map[string]bool{
	"model": true, "messages": true, "stream": true, "temperature": true,
	"top_p": true, "max_tokens": true, "stop": true, "metadata": true,
}

// openAIIgnored are fields that can be safely ignored.
var openAIIgnored = map[string]bool{
	"user": true,
}

type openAIMessage struct {
	Role    string          `json:"role"`
	Content json.RawMessage `json:"content"`
}

type openAIChatRequest struct {
	Model       string          `json:"model"`
	Messages    []openAIMessage `json:"messages"`
	Stream      bool            `json:"stream"`
	Temperature *float64        `json:"temperature"`
	TopP        *float64        `json:"top_p"`
	MaxTokens   int             `json:"max_tokens"`
	Stop        json.RawMessage `json:"stop"`
	Metadata    map[string]any  `json:"metadata"`
}

func (s *Server) handleChatCompletions(c *gin.Context) {
	body, err := c.GetRawData()
	if err != nil {
		writeOpenAIError(c, apierr.InvalidRequest("failed to read request body"))
		return
	}

	req, err := parseOpenAIChatRequest(body)
	if err != nil {
		writeOpenAIError(c, asAPIError(err))
		return
	}

	normalized, err := openAIToNormalized(req)
	if err != nil {
		writeOpenAIError(c, asAPIError(err))
		return
	}
	if normalized.MaxTokens == 0 {
		normalized.MaxTokens = s.cfg.Server.DefaultMaxTokens
	}

	alias := req.Model
	if normalized.Stream {
		ch, err := s.svc.Stream(c.Request.Context(), normalized)
		if err != nil {
			writeOpenAIError(c, asAPIError(err))
			return
		}
		s.writeOpenAIStream(c, alias, ch)
		return
	}

	resp, err := s.svc.Chat(c.Request.Context(), normalized)
	if err != nil {
		writeOpenAIError(c, asAPIError(err))
		return
	}
	writeOpenAIChatResponse(c, alias, resp)
}

func parseOpenAIChatRequest(body []byte) (*openAIChatRequest, error) {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, apierr.InvalidRequest("request body is malformed")
	}
	for key := range raw {
		if openAIAllowed[key] || openAIIgnored[key] {
			continue
		}
		return nil, apierr.UnsupportedField("unsupported field: " + key)
	}
	var req openAIChatRequest
	if err := json.Unmarshal(body, &req); err != nil {
		return nil, apierr.InvalidRequest("request body is malformed")
	}
	return &req, nil
}

func openAIToNormalized(req *openAIChatRequest) (*model.ChatRequest, error) {
	if strings.TrimSpace(req.Model) == "" {
		return nil, apierr.InvalidRequest("model is required")
	}
	if len(req.Messages) == 0 {
		return nil, apierr.InvalidRequest("messages must not be empty")
	}

	out := &model.ChatRequest{
		Model:       req.Model,
		Stream:      req.Stream,
		Temperature: req.Temperature,
		TopP:        req.TopP,
		MaxTokens:   req.MaxTokens,
		Metadata:    req.Metadata,
	}

	for _, m := range req.Messages {
		content, err := decodeOpenAIContent(m.Content)
		if err != nil {
			return nil, err
		}
		switch m.Role {
		case "system", "developer":
			if out.System != "" {
				out.System += "\n"
			}
			out.System += content
		case "user", "assistant":
			out.Messages = append(out.Messages, model.Message{Role: model.Role(m.Role), Content: content})
		default:
			return nil, apierr.InvalidRequest("invalid message role: " + m.Role)
		}
	}

	if len(req.Stop) > 0 {
		var stop []string
		if err := json.Unmarshal(req.Stop, &stop); err != nil {
			var single string
			if err2 := json.Unmarshal(req.Stop, &single); err2 != nil {
				return nil, apierr.InvalidRequest("stop must be a string or array of strings")
			}
			stop = []string{single}
		}
		out.StopSequences = stop
	}
	return out, nil
}

func decodeOpenAIContent(raw json.RawMessage) (string, error) {
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		return s, nil
	}
	var parts []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	}
	if err := json.Unmarshal(raw, &parts); err != nil {
		return "", apierr.InvalidRequest("message content must be text")
	}
	var b strings.Builder
	for _, p := range parts {
		if p.Type != "text" {
			return "", apierr.UnsupportedField("unsupported message content type: " + p.Type)
		}
		b.WriteString(p.Text)
	}
	return b.String(), nil
}

func writeOpenAIChatResponse(c *gin.Context, alias string, resp *model.ChatResponse) {
	id := resp.ID
	if id == "" {
		id = "chatcmpl-" + randomID()
	}
	c.JSON(http.StatusOK, gin.H{
		"id":      id,
		"object":  "chat.completion",
		"created": time.Now().Unix(),
		"model":   alias,
		"choices": []gin.H{
			{
				"index":         0,
				"message":       gin.H{"role": "assistant", "content": resp.Content},
				"finish_reason": openAIFinishReason(resp.StopReason),
			},
		},
		"usage": gin.H{
			"prompt_tokens":     resp.Usage.InputTokens,
			"completion_tokens": resp.Usage.OutputTokens,
			"total_tokens":      resp.Usage.InputTokens + resp.Usage.OutputTokens,
		},
	})
}

func openAIFinishReason(reason string) string {
	switch reason {
	case "max_tokens", "max_output_tokens", "length":
		return "length"
	case "content_filter":
		return "content_filter"
	default:
		return "stop"
	}
}

func (s *Server) writeOpenAIStream(c *gin.Context, alias string, ch <-chan model.StreamEvent) {
	header := c.Writer.Header()
	header.Set("Content-Type", "text/event-stream")
	header.Set("Cache-Control", "no-cache")
	header.Set("Connection", "keep-alive")
	header.Set("X-Accel-Buffering", "no")
	c.Writer.WriteHeader(http.StatusOK)

	id := ""
	created := time.Now().Unix()
	sentRole := false
	finishSent := false
	flusher := c.Writer.(http.Flusher)

	writeChunk := func(delta gin.H, finish any) {
		chunk := gin.H{
			"id":      id,
			"object":  "chat.completion.chunk",
			"created": created,
			"model":   alias,
			"choices": []gin.H{
				{"index": 0, "delta": delta, "finish_reason": finish},
			},
		}
		data, _ := json.Marshal(chunk)
		fmt.Fprintf(c.Writer, "data: %s\n\n", data)
		flusher.Flush()
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
			if id == "" {
				id = "chatcmpl-" + randomID()
			}
			writeChunk(gin.H{"role": "assistant", "content": ""}, nil)
			sentRole = true
		case model.StreamContentDelta:
			if !sentRole {
				if id == "" {
					id = "chatcmpl-" + randomID()
				}
				writeChunk(gin.H{"role": "assistant", "content": ""}, nil)
				sentRole = true
			}
			writeChunk(gin.H{"content": ev.Content}, nil)
		case model.StreamMessageStop:
			finish := openAIFinishReason(ev.StopReason)
			writeChunk(gin.H{}, finish)
			finishSent = true
		case model.StreamError:
			if ev.Error != nil {
				errBody := gin.H{"error": gin.H{"message": ev.Error.Error(), "type": "api_error"}}
				data, _ := json.Marshal(errBody)
				fmt.Fprintf(c.Writer, "data: %s\n\n", data)
				flusher.Flush()
			}
		}
	}

	if !finishSent {
		writeChunk(gin.H{}, "stop")
	}
	fmt.Fprint(c.Writer, "data: [DONE]\n\n")
	flusher.Flush()
}

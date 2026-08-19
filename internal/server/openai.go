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

func (s *Server) handleResponses(c *gin.Context) {
	body, err := c.GetRawData()
	if err != nil {
		writeOpenAIError(c, apierr.InvalidRequest("failed to read request body"))
		return
	}

	req, alias, normalized, err := parseOpenAIResponsesRequest(body)
	if err != nil {
		writeOpenAIError(c, asAPIError(err))
		return
	}
	if normalized.MaxTokens == 0 {
		normalized.MaxTokens = s.cfg.Server.DefaultMaxTokens
	}

	if normalized.Stream {
		ch, err := s.svc.Stream(c.Request.Context(), normalized)
		if err != nil {
			writeOpenAIError(c, asAPIError(err))
			return
		}
		s.writeResponsesStream(c, req.Model, alias, ch)
		return
	}

	resp, err := s.svc.Chat(c.Request.Context(), normalized)
	if err != nil {
		writeOpenAIError(c, asAPIError(err))
		return
	}
	writeResponsesResponse(c, req.Model, alias, resp)
}

func parseOpenAIResponsesRequest(body []byte) (*responsesRequest, string, *model.ChatRequest, error) {
	var req responsesRequest
	if err := json.Unmarshal(body, &req); err != nil {
		return nil, "", nil, apierr.InvalidRequest("request body is malformed")
	}
	if strings.TrimSpace(req.Model) == "" {
		return nil, "", nil, apierr.InvalidRequest("model is required")
	}

	normalized := &model.ChatRequest{
		Model:         req.Model,
		Stream:        req.Stream,
		Temperature:   req.Temperature,
		TopP:          req.TopP,
		MaxTokens:     req.MaxOutputTokens,
		Metadata:      req.Metadata,
		StopSequences: req.StopSequences,
	}

	if err := appendResponsesInput(normalized, req.Input); err != nil {
		return nil, "", nil, err
	}
	if req.Instructions != "" {
		normalized.System = req.Instructions
	}
	for _, m := range req.Messages {
		content, err := decodeOpenAIContent(m.Content)
		if err != nil {
			return nil, "", nil, err
		}
		switch m.Role {
		case "system", "developer":
			if normalized.System != "" {
				normalized.System += "\n"
			}
			normalized.System += content
		case "user", "assistant":
			normalized.Messages = append(normalized.Messages, model.Message{Role: model.Role(m.Role), Content: content})
		default:
			return nil, "", nil, apierr.InvalidRequest("invalid message role: " + m.Role)
		}
	}
	if len(normalized.Messages) == 0 {
		return nil, "", nil, apierr.InvalidRequest("input or messages must not be empty")
	}

	return &req, req.Model, normalized, nil
}

func appendResponsesInput(normalized *model.ChatRequest, raw json.RawMessage) error {
	if len(raw) == 0 || string(raw) == "null" {
		return nil
	}
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		if s != "" {
			normalized.Messages = append(normalized.Messages, model.Message{Role: "user", Content: s})
		}
		return nil
	}
	var items []struct {
		Role    string          `json:"role"`
		Type    string          `json:"type"`
		Content json.RawMessage `json:"content"`
		Text    string          `json:"text"`
	}
	if err := json.Unmarshal(raw, &items); err != nil {
		return apierr.InvalidRequest("input must be a string or array of items")
	}
	for _, it := range items {
		if it.Role == "" && it.Type != "" {
			it.Role = "user"
		}
		if it.Role == "system" {
			if it.Text != "" {
				if normalized.System != "" {
					normalized.System += "\n"
				}
				normalized.System += it.Text
			}
			continue
		}
		if it.Role == "" {
			it.Role = "user"
		}
		content := it.Text
		if len(it.Content) > 0 {
			decoded, err := decodeOpenAIContent(it.Content)
			if err != nil {
				return err
			}
			content = decoded
		}
		if content != "" {
			normalized.Messages = append(normalized.Messages, model.Message{Role: model.Role(it.Role), Content: content})
		}
	}
	return nil
}

type responsesRequest struct {
	Model           string          `json:"model"`
	Input           json.RawMessage `json:"input"`
	Instructions    string          `json:"instructions"`
	Messages        []openAIMessage `json:"messages"`
	Stream          bool            `json:"stream"`
	Temperature     *float64        `json:"temperature"`
	TopP            *float64        `json:"top_p"`
	MaxOutputTokens int             `json:"max_output_tokens"`
	StopSequences   []string        `json:"stop_sequences"`
	Metadata        map[string]any  `json:"metadata"`
}

func writeResponsesResponse(c *gin.Context, reqModel, alias string, resp *model.ChatResponse) {
	id := resp.ID
	if id == "" {
		id = "resp_" + randomID()
	}
	c.JSON(http.StatusOK, gin.H{
		"id":      id,
		"object":  "response",
		"created": time.Now().Unix(),
		"model":   reqModel,
		"output": []gin.H{
			{
				"type":    "message",
				"status":  "completed",
				"role":    "assistant",
				"content": []gin.H{{"type": "output_text", "text": resp.Content}},
			},
		},
		"usage": gin.H{
			"input_tokens":  resp.Usage.InputTokens,
			"output_tokens": resp.Usage.OutputTokens,
			"total_tokens":  resp.Usage.InputTokens + resp.Usage.OutputTokens,
		},
	})
}

func (s *Server) writeResponsesStream(c *gin.Context, reqModel, alias string, ch <-chan model.StreamEvent) {
	header := c.Writer.Header()
	header.Set("Content-Type", "text/event-stream")
	header.Set("Cache-Control", "no-cache")
	header.Set("Connection", "keep-alive")
	header.Set("X-Accel-Buffering", "no")
	c.Writer.WriteHeader(http.StatusOK)

	respID := ""
	created := time.Now().Unix()
	contentBuilder := new(strings.Builder)
	flusher := c.Writer.(http.Flusher)
	emitStarted := false
	msgID := respID + "_msg"

	writeEvent := func(data gin.H) {
		body, _ := json.Marshal(data)
		var event string
		if t, ok := data["type"].(string); ok {
			event = t
		}
		fmt.Fprintf(c.Writer, "event: %s\ndata: %s\n\n", event, body)
		flusher.Flush()
	}

	// startMessage emits the response.created / output_item.added /
	// content_part.added prologue exactly once, so the SDK builds the
	// output item structure before any delta arrives.
	startMessage := func() {
		if emitStarted {
			return
		}
		emitStarted = true
		if respID == "" {
			respID = "resp_" + randomID()
		}
		msgID = respID + "_msg"
		writeEvent(gin.H{
			"type": "response.created",
			"response": gin.H{
				"id":      respID,
				"object":  "response",
				"created": created,
				"model":   reqModel,
				"status":  "in_progress",
				"output":  []gin.H{},
			},
		})
		writeEvent(gin.H{
			"type":         "response.output_item.added",
			"output_index": 0,
			"item": gin.H{
				"id":      msgID,
				"type":    "message",
				"status":  "in_progress",
				"role":    "assistant",
				"content": []gin.H{},
			},
		})
		writeEvent(gin.H{
			"type":          "response.content_part.added",
			"item_id":       msgID,
			"output_index":  0,
			"content_index": 0,
			"part":          gin.H{"type": "output_text", "text": ""},
		})
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
				respID = ev.ID
			}
			startMessage()
		case model.StreamContentDelta:
			startMessage()
			writeEvent(gin.H{
				"type":          "response.output_text.delta",
				"item_id":       msgID,
				"output_index":  0,
				"content_index": 0,
				"delta":         ev.Content,
			})
			contentBuilder.WriteString(ev.Content)
		case model.StreamMessageStop:
			startMessage()
			writeEvent(gin.H{
				"type": "response.completed",
				"response": gin.H{
					"id":      respID,
					"object":  "response",
					"created": created,
					"model":   reqModel,
					"status":  "completed",
					"output": []gin.H{
						{
							"id":      msgID,
							"type":    "message",
							"status":  "completed",
							"role":    "assistant",
							"content": []gin.H{{"type": "output_text", "text": contentBuilder.String()}},
						},
					},
					"usage": gin.H{
						"input_tokens":          0,
						"output_tokens":         0,
						"total_tokens":          0,
						"output_tokens_details": gin.H{"reasoning_tokens": 0},
						"input_tokens_details":  gin.H{"cached_tokens": 0},
					},
				},
			})
			return
		case model.StreamError:
			if ev.Error != nil {
				writeEvent(gin.H{
					"type":  "error",
					"error": gin.H{"message": ev.Error.Error(), "type": "api_error"},
				})
			}
			return
		}
	}
	if emitStarted {
		writeEvent(gin.H{
			"type": "response.completed",
			"response": gin.H{
				"id":      respID,
				"object":  "response",
				"created": created,
				"model":   reqModel,
				"status":  "completed",
				"output": []gin.H{
					{
						"id":      msgID,
						"type":    "message",
						"status":  "completed",
						"role":    "assistant",
						"content": []gin.H{{"type": "output_text", "text": contentBuilder.String()}},
					},
				},
				"usage": gin.H{
					"input_tokens":          0,
					"output_tokens":         0,
					"total_tokens":          0,
					"output_tokens_details": gin.H{"reasoning_tokens": 0},
					"input_tokens_details":  gin.H{"cached_tokens": 0},
				},
			},
		})
	}
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
		switch p.Type {
		case "text", "input_text", "output_text", "":
			b.WriteString(p.Text)
		default:
			return "", apierr.UnsupportedField("unsupported message content type: " + p.Type)
		}
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
			return
		}
	}

	if !finishSent {
		writeChunk(gin.H{}, "stop")
	}
	fmt.Fprint(c.Writer, "data: [DONE]\n\n")
	flusher.Flush()
}

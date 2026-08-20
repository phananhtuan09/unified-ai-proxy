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
	"stop_sequences": true, "metadata": true, "output_config": true, "tools": true, "thinking": true,
	"tool_choice": true,
}

type anthropicContentBlock struct {
	Type      string          `json:"type"`
	Text      string          `json:"text"`
	ID        string          `json:"id"`
	Name      string          `json:"name"`
	Input     json.RawMessage `json:"input"`
	ToolUseID string          `json:"tool_use_id"`
	Content   json.RawMessage `json:"content"`
}

type anthropicMessage struct {
	Role    string          `json:"role"`
	Content json.RawMessage `json:"content"`
}

type anthropicTool struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	InputSchema map[string]any `json:"input_schema"`
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
	OutputConfig  map[string]any     `json:"output_config"`
	Tools         []anthropicTool    `json:"tools"`
	Thinking      map[string]any     `json:"thinking"`
	ToolChoice    json.RawMessage    `json:"tool_choice"`
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
	for _, tool := range req.Tools {
		out.Tools = append(out.Tools, model.Tool{Name: tool.Name, Description: tool.Description, Parameters: tool.InputSchema})
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
		case "system":
			blocks, err := decodeAnthropicContent(m.Content)
			if err != nil {
				return nil, err
			}
			for _, block := range blocks {
				if block.Type != "text" {
					return nil, apierr.UnsupportedField("unsupported system content block type: " + block.Type)
				}
				if out.System != "" {
					out.System += "\n"
				}
				out.System += block.Text
			}
			continue
		case "user", "assistant":
		default:
			return nil, apierr.InvalidRequest("invalid message role: " + m.Role)
		}
		blocks, err := decodeAnthropicContent(m.Content)
		if err != nil {
			return nil, err
		}
		var text strings.Builder
		var toolCalls []model.ToolCall
		var toolResults []model.ToolResult
		for _, block := range blocks {
			switch block.Type {
			case "text":
				text.WriteString(block.Text)
			case "tool_use":
				if block.ID == "" || block.Name == "" || len(block.Input) == 0 {
					return nil, apierr.InvalidRequest("tool_use block requires id, name, and input")
				}
				toolCalls = append(toolCalls, model.ToolCall{ID: block.ID, Name: block.Name, Arguments: string(block.Input)})
			case "tool_result":
				content, err := decodeAnthropicToolResult(block.Content)
				if err != nil {
					return nil, err
				}
				if block.ToolUseID == "" {
					return nil, apierr.InvalidRequest("tool_result block requires tool_use_id")
				}
				toolResults = append(toolResults, model.ToolResult{CallID: block.ToolUseID, Content: content})
			default:
				return nil, apierr.UnsupportedField("unsupported content block type: " + block.Type)
			}
		}
		if text.Len() > 0 || len(toolCalls) > 0 || len(toolResults) == 0 {
			out.Messages = append(out.Messages, model.Message{
				Role:      model.Role(m.Role),
				Content:   text.String(),
				ToolCalls: toolCalls,
			})
		}
		for i := range toolResults {
			result := toolResults[i]
			out.Messages = append(out.Messages, model.Message{
				Role:       model.RoleTool,
				ToolResult: &result,
			})
		}
	}
	return out, nil
}

func decodeAnthropicContent(raw json.RawMessage) ([]anthropicContentBlock, error) {
	var text string
	if err := json.Unmarshal(raw, &text); err == nil {
		return []anthropicContentBlock{{Type: "text", Text: text}}, nil
	}
	var blocks []anthropicContentBlock
	if err := json.Unmarshal(raw, &blocks); err != nil {
		return nil, apierr.InvalidRequest("message content must be a string or content blocks")
	}
	return blocks, nil
}

func decodeAnthropicToolResult(raw json.RawMessage) (string, error) {
	var text string
	if err := json.Unmarshal(raw, &text); err == nil {
		return text, nil
	}
	var blocks []anthropicContentBlock
	if err := json.Unmarshal(raw, &blocks); err != nil {
		return "", apierr.InvalidRequest("tool_result content must be a string or text blocks")
	}
	var out strings.Builder
	for _, block := range blocks {
		if block.Type != "text" {
			return "", apierr.UnsupportedField("unsupported tool_result content block type: " + block.Type)
		}
		out.WriteString(block.Text)
	}
	return out.String(), nil
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
	content := []gin.H{}
	if resp.Content != "" || len(resp.ToolCalls) == 0 {
		content = append(content, gin.H{"type": "text", "text": resp.Content})
	}
	for _, call := range resp.ToolCalls {
		var input any
		if err := json.Unmarshal([]byte(call.Arguments), &input); err != nil {
			input = map[string]any{}
		}
		content = append(content, gin.H{"type": "tool_use", "id": call.ID, "name": call.Name, "input": input})
	}
	stopReason := anthropicStopReason(resp.StopReason)
	c.JSON(http.StatusOK, gin.H{
		"id":            id,
		"type":          "message",
		"role":          "assistant",
		"content":       content,
		"model":         alias,
		"stop_reason":   stopReason,
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
	case "tool-calls", "tool_call":
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
	messageStarted := false
	blockStarted := false

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
		messageStarted = true
	}
	startTextBlock := func() {
		if !messageStarted {
			startMessage()
		}
		if blockStarted {
			return
		}
		writeEvent("content_block_start", gin.H{
			"type":          "content_block_start",
			"index":         index,
			"content_block": gin.H{"type": "text", "text": ""},
		})
		blockStarted = true
	}
	closeBlock := func() {
		if !blockStarted {
			return
		}
		writeEvent("content_block_stop", gin.H{"type": "content_block_stop", "index": index})
		blockStarted = false
		index++
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
			if !messageStarted {
				startMessage()
			}
		case model.StreamContentDelta:
			startTextBlock()
			writeEvent("content_block_delta", gin.H{
				"type":  "content_block_delta",
				"index": index,
				"delta": gin.H{"type": "text_delta", "text": ev.Content},
			})
		case model.StreamToolCall:
			if ev.ToolCall == nil {
				continue
			}
			if !messageStarted {
				startMessage()
			}
			closeBlock()
			call := ev.ToolCall
			writeEvent("content_block_start", gin.H{
				"type": "content_block_start", "index": index,
				"content_block": gin.H{"type": "tool_use", "id": call.ID, "name": call.Name, "input": gin.H{}},
			})
			arguments := call.Arguments
			if arguments == "" {
				arguments = "{}"
			}
			writeEvent("content_block_delta", gin.H{
				"type": "content_block_delta", "index": index,
				"delta": gin.H{"type": "input_json_delta", "partial_json": arguments},
			})
			writeEvent("content_block_stop", gin.H{"type": "content_block_stop", "index": index})
			index++
		case model.StreamMessageStop:
			if !messageStarted {
				startMessage()
			}
			closeBlock()
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

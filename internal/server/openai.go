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

	var totalDeltas int
	var usage *model.Usage
	var toolCalls []model.ToolCall
	streamStartTime := time.Now()

	writeEvent := func(data gin.H) {
		body, _ := json.Marshal(data)
		var event string
		if t, ok := data["type"].(string); ok {
			event = t
		}
		fmt.Fprintf(c.Writer, "event: %s\ndata: %s\n\n", event, body)
		flusher.Flush()
	}

	buildOutput := func(status string) []gin.H {
		output := []gin.H{}
		if contentBuilder.Len() > 0 || len(toolCalls) == 0 {
			output = append(output, gin.H{
				"id":      msgID,
				"type":    "message",
				"status":  status,
				"role":    "assistant",
				"content": []gin.H{{"type": "output_text", "text": contentBuilder.String()}},
			})
		}
		for _, tc := range toolCalls {
			output = append(output, gin.H{
				"id":        tc.ID,
				"type":      "function_call",
				"call_id":   tc.ID,
				"name":      tc.Name,
				"arguments": tc.Arguments,
				"status":    status,
			})
		}
		return output
	}

	writeCompleted := func(status string) {
		writeEvent(gin.H{
			"type": "response.completed",
			"response": gin.H{
				"id":      respID,
				"object":  "response",
				"created": created,
				"model":   reqModel,
				"status":  status,
				"output":  buildOutput(status),
				"usage": gin.H{
					"input_tokens":          usageInputTokens(usage),
					"output_tokens":         usageOutputTokens(usage),
					"total_tokens":          usageInputTokens(usage) + usageOutputTokens(usage),
					"output_tokens_details": gin.H{"reasoning_tokens": 0},
					"input_tokens_details":  gin.H{"cached_tokens": 0},
				},
			},
		})
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
			totalDeltas++
		case model.StreamToolCall:
			if ev.ToolCall == nil {
				continue
			}
			startMessage()
			tc := *ev.ToolCall
			if tc.ID == "" {
				tc.ID = "fc_" + randomID()
			}
			toolCalls = append(toolCalls, tc)
			outputIndex := len(toolCalls)
			writeEvent(gin.H{
				"type":         "response.output_item.added",
				"output_index": outputIndex,
				"item": gin.H{
					"id":        tc.ID,
					"type":      "function_call",
					"call_id":   tc.ID,
					"name":      tc.Name,
					"arguments": "",
					"status":    "in_progress",
				},
			})
			writeEvent(gin.H{
				"type":         "response.function_call_arguments.delta",
				"item_id":      tc.ID,
				"output_index": outputIndex,
				"delta":        tc.Arguments,
			})
			writeEvent(gin.H{
				"type":         "response.output_item.done",
				"output_index": outputIndex,
				"item": gin.H{
					"id":        tc.ID,
					"type":      "function_call",
					"call_id":   tc.ID,
					"name":      tc.Name,
					"arguments": tc.Arguments,
					"status":    "completed",
				},
			})
		case model.StreamMessageStop:
			usage = ev.Usage
			startMessage()
			writeCompleted("completed")
			if s.slog != nil {
				s.slog.Info("responses.stream.complete",
					"model", alias,
					"stop_reason", ev.StopReason,
					"deltas", totalDeltas,
					"duration_ms", time.Since(streamStartTime).Milliseconds(),
					"usage_input", usageInputTokens(ev.Usage),
					"usage_output", usageOutputTokens(ev.Usage),
				)
			}
			return
		case model.StreamError:
			if ev.Error != nil {
				if s.slog != nil {
					s.slog.Error("responses.stream.error",
						"model", alias,
						"error", ev.Error.Error(),
						"deltas", totalDeltas,
					)
				}
				writeEvent(gin.H{
					"type":  "error",
					"error": gin.H{"message": ev.Error.Error(), "type": "api_error"},
				})
			}
			return
		}
	}
	if emitStarted {
		if s.slog != nil {
			s.slog.Warn("responses.stream.no_stop_event",
				"model", alias,
				"deltas", totalDeltas,
				"duration_ms", time.Since(streamStartTime).Milliseconds(),
			)
		}
		writeCompleted("incomplete")
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

	if s.slog != nil {
		s.slog.Info("openai.chat.request",
			"model", alias,
			"stream", normalized.Stream,
			"message_count", len(normalized.Messages),
		)
	}

	if normalized.Stream {
		ch, err := s.svc.Stream(c.Request.Context(), normalized)
		if err != nil {
			if s.slog != nil {
				s.slog.Error("openai.chat.stream.error",
					"model", alias,
					"error", err.Error(),
				)
			}
			writeOpenAIError(c, asAPIError(err))
			return
		}
		s.writeOpenAIStream(c, alias, ch)
		return
	}

	resp, err := s.svc.Chat(c.Request.Context(), normalized)
	if err != nil {
		if s.slog != nil {
			s.slog.Error("openai.chat.error",
				"model", alias,
				"error", err.Error(),
			)
		}
		writeOpenAIError(c, asAPIError(err))
		return
	}

	if s.slog != nil {
		s.slog.Info("openai.chat.response",
			"model", alias,
			"stop_reason", resp.StopReason,
			"usage_input", resp.Usage.InputTokens,
			"usage_output", resp.Usage.OutputTokens,
		)
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

	// Track stream stats for logging
	var totalDeltas int
	streamStartTime := time.Now()

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
			totalDeltas++
		case model.StreamMessageStop:
			finish := openAIFinishReason(ev.StopReason)
			writeChunk(gin.H{}, finish)
			finishSent = true
			if s.slog != nil {
				s.slog.Info("openai.stream.complete",
					"model", alias,
					"stop_reason", ev.StopReason,
					"deltas", totalDeltas,
					"duration_ms", time.Since(streamStartTime).Milliseconds(),
					"usage_input", usageInputTokens(ev.Usage),
					"usage_output", usageOutputTokens(ev.Usage),
				)
			}
		case model.StreamError:
			if ev.Error != nil {
				if s.slog != nil {
					s.slog.Error("openai.stream.error",
						"model", alias,
						"error", ev.Error.Error(),
						"deltas", totalDeltas,
					)
				}
				errBody := gin.H{"error": gin.H{"message": ev.Error.Error(), "type": "api_error"}}
				data, _ := json.Marshal(errBody)
				fmt.Fprintf(c.Writer, "data: %s\n\n", data)
				flusher.Flush()
			}
			return
		}
	}

	if !finishSent {
		if s.slog != nil {
			s.slog.Warn("openai.stream.no_stop_event",
				"model", alias,
				"deltas", totalDeltas,
				"duration_ms", time.Since(streamStartTime).Milliseconds(),
			)
		}
		writeChunk(gin.H{}, "stop")
	}
	fmt.Fprint(c.Writer, "data: [DONE]\n\n")
	flusher.Flush()
}

package server

import (
	"encoding/json"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/tuanp-github/unified-ai-proxy/internal/apierr"
	"github.com/tuanp-github/unified-ai-proxy/internal/model"
)

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

	if s.slog != nil {
		s.slog.Info("responses.request",
			"model", alias,
			"stream", normalized.Stream,
			"message_count", len(normalized.Messages),
		)
	}

	if normalized.Stream {
		ch, err := s.svc.Stream(c.Request.Context(), normalized)
		if err != nil {
			if s.slog != nil {
				s.slog.Error("responses.stream.error",
					"model", alias,
					"error", err.Error(),
				)
			}
			writeOpenAIError(c, asAPIError(err))
			return
		}
		s.writeResponsesStream(c, req.Model, alias, ch)
		return
	}
	resp, err := s.svc.Chat(c.Request.Context(), normalized)
	if err != nil {
		if s.slog != nil {
			s.slog.Error("responses.error",
				"model", alias,
				"error", err.Error(),
			)
		}
		writeOpenAIError(c, asAPIError(err))
		return
	}

	if s.slog != nil {
		s.slog.Info("responses.response",
			"model", alias,
			"stop_reason", resp.StopReason,
			"usage_input", resp.Usage.InputTokens,
			"usage_output", resp.Usage.OutputTokens,
			"content_length", len(resp.Content),
		)
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
	normalized := &model.ChatRequest{Model: req.Model, Stream: req.Stream, Temperature: req.Temperature, TopP: req.TopP, MaxTokens: req.MaxOutputTokens, Metadata: req.Metadata, StopSequences: req.StopSequences}
	for _, t := range req.Tools {
		normalized.Tools = append(normalized.Tools, model.Tool{Name: t.Name, Description: t.Description, Parameters: t.Parameters})
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
		Role      string          `json:"role"`
		Type      string          `json:"type"`
		Content   json.RawMessage `json:"content"`
		Text      string          `json:"text"`
		CallID    string          `json:"call_id"`
		Name      string          `json:"name"`
		Arguments string          `json:"arguments"`
		Output    json.RawMessage `json:"output"`
	}
	if err := json.Unmarshal(raw, &items); err != nil {
		return apierr.InvalidRequest("input must be a string or array of items")
	}
	for _, it := range items {
		switch it.Type {
		case "function_call":
			normalized.Messages = append(normalized.Messages, model.Message{
				Role:      model.RoleAssistant,
				ToolCalls: []model.ToolCall{{ID: it.CallID, Name: it.Name, Arguments: it.Arguments}},
			})
			continue
		case "function_call_output":
			out, err := decodeOpenAIContent(it.Output)
			if err != nil {
				return err
			}
			normalized.Messages = append(normalized.Messages, model.Message{
				Role:       model.RoleTool,
				ToolResult: &model.ToolResult{CallID: it.CallID, Content: out},
			})
			continue
		}
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
	Tools           []responsesTool `json:"tools"`
	Stream          bool            `json:"stream"`
	Temperature     *float64        `json:"temperature"`
	TopP            *float64        `json:"top_p"`
	MaxOutputTokens int             `json:"max_output_tokens"`
	StopSequences   []string        `json:"stop_sequences"`
	Metadata        map[string]any  `json:"metadata"`
}

type responsesTool struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Parameters  map[string]any `json:"parameters"`
}

package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/tuanp-github/unified-ai-proxy/internal/config"
	"github.com/tuanp-github/unified-ai-proxy/internal/model"
)

// Codex is the OpenAI Codex provider authenticated via browser OAuth.
// It speaks the OpenAI Responses API.
type Codex struct {
	base
}

// NewCodex constructs an OpenAI Codex provider.
func NewCodex(cfg config.ProviderConfig, timeout time.Duration) *Codex {
	var models []model.Model
	for _, m := range cfg.Models {
		models = append(models, model.Model{
			ID:       m.ID,
			Upstream: m.Upstream,
			Provider: "openai_codex",
		})
	}
	return &Codex{base: newBase("openai_codex", cfg, models, timeout)}
}

func (c *Codex) endpoint() string {
	return strings.TrimRight(c.cfg.API.BaseURL, "/") + "/v1/responses"
}

type codexInputItem struct {
	Role    string         `json:"role"`
	Content []codexContent `json:"content"`
}

type codexContent struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type codexRequest struct {
	Model       string           `json:"model"`
	Input       []codexInputItem `json:"input"`
	Stream      bool             `json:"stream"`
	Temperature *float64         `json:"temperature,omitempty"`
	TopP        *float64         `json:"top_p,omitempty"`
	MaxOutput   int              `json:"max_output_tokens,omitempty"`
	Stop        []string         `json:"stop,omitempty"`
	Metadata    map[string]any   `json:"metadata,omitempty"`
}

func (c *Codex) buildRequest(req *model.ChatRequest) *codexRequest {
	out := &codexRequest{
		Model:       req.Model,
		Stream:      req.Stream,
		Temperature: req.Temperature,
		TopP:        req.TopP,
		MaxOutput:   req.MaxTokens,
		Stop:        req.StopSequences,
		Metadata:    req.Metadata,
	}
	if req.System != "" {
		out.Input = append(out.Input, codexInputItem{
			Role:    "developer",
			Content: []codexContent{{Type: "input_text", Text: req.System}},
		})
	}
	for _, m := range req.Messages {
		role := string(m.Role)
		if role == "system" {
			role = "developer"
		}
		out.Input = append(out.Input, codexInputItem{
			Role:    role,
			Content: []codexContent{{Type: "input_text", Text: m.Content}},
		})
	}
	return out
}

func (c *Codex) headers(token string) map[string]string {
	return map[string]string{"Authorization": "Bearer " + token}
}

// ChatCompletion performs a non-streaming Responses API request.
func (c *Codex) ChatCompletion(ctx context.Context, account model.Account, req *model.ChatRequest) (*model.ChatResponse, error) {
	token, err := c.accessToken(ctx, account)
	if err != nil {
		return nil, err
	}
	status, body, err := c.doJSON(ctx, http.MethodPost, c.endpoint(), c.headers(token), c.buildRequest(req))
	if err != nil {
		return nil, networkError(err)
	}
	if status != http.StatusOK {
		return nil, c.upstreamError(status, body)
	}
	var resp struct {
		ID     string `json:"id"`
		Model  string `json:"model"`
		Status string `json:"status"`
		Output []struct {
			Type    string `json:"type"`
			Role    string `json:"role"`
			Content []struct {
				Type string `json:"type"`
				Text string `json:"text"`
			} `json:"content"`
		} `json:"output"`
		Usage struct {
			InputTokens  int `json:"input_tokens"`
			OutputTokens int `json:"output_tokens"`
		} `json:"usage"`
		Incomplete struct {
			Reason string `json:"reason"`
		} `json:"incomplete_details"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("parse codex response: %w", err)
	}

	var text strings.Builder
	for _, item := range resp.Output {
		if item.Type != "message" {
			continue
		}
		for _, part := range item.Content {
			if part.Type == "output_text" || part.Type == "text" {
				text.WriteString(part.Text)
			}
		}
	}

	stopReason := resp.Status
	if resp.Incomplete.Reason != "" {
		stopReason = resp.Incomplete.Reason
	}

	return &model.ChatResponse{
		ID:         resp.ID,
		Model:      resp.Model,
		Content:    text.String(),
		StopReason: stopReason,
		Usage: model.Usage{
			InputTokens:  resp.Usage.InputTokens,
			OutputTokens: resp.Usage.OutputTokens,
		},
	}, nil
}

// StreamChatCompletion streams a Responses API request.
func (c *Codex) StreamChatCompletion(ctx context.Context, account model.Account, req *model.ChatRequest) (<-chan model.StreamEvent, error) {
	token, err := c.accessToken(ctx, account)
	if err != nil {
		return nil, err
	}
	sreq := c.buildRequest(req)
	sreq.Stream = true

	data, err := json.Marshal(sreq)
	if err != nil {
		return nil, err
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint(), strings.NewReader(string(data)))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "text/event-stream")
	for k, v := range c.headers(token) {
		httpReq.Header.Set(k, v)
	}

	resp, err := c.client.Do(httpReq)
	if err != nil {
		return nil, networkError(err)
	}
	if resp.StatusCode != http.StatusOK {
		defer resp.Body.Close()
		return nil, c.upstreamError(resp.StatusCode, readBody(resp))
	}

	out := make(chan model.StreamEvent)
	go func() {
		defer close(out)
		defer resp.Body.Close()

		var messageID, modelName, stopReason string
		var usage *model.Usage
		sentStart := false

		for ev := range readSSE(resp.Body) {
			select {
			case <-ctx.Done():
				return
			default:
			}

			if strings.TrimSpace(ev.Data) == "" {
				continue
			}
			var payload struct {
				Type     string          `json:"type"`
				Delta    json.RawMessage `json:"delta"`
				Response json.RawMessage `json:"response"`
			}
			if err := json.Unmarshal([]byte(ev.Data), &payload); err != nil {
				continue
			}

			switch payload.Type {
			case "response.created":
				var r struct {
					ID    string `json:"id"`
					Model string `json:"model"`
				}
				_ = json.Unmarshal(payload.Response, &r)
				messageID = r.ID
				modelName = r.Model
				out <- model.StreamEvent{Type: model.StreamMessageStart, ID: messageID, Model: modelName}
				sentStart = true
			case "response.output_text.delta":
				var delta string
				if err := json.Unmarshal(payload.Delta, &delta); err == nil && delta != "" {
					out <- model.StreamEvent{Type: model.StreamContentDelta, Content: delta}
				}
			case "response.content_part.delta":
				var delta struct {
					Type string `json:"type"`
					Text string `json:"text"`
				}
				_ = json.Unmarshal(payload.Delta, &delta)
				if delta.Type == "text_delta" && delta.Text != "" {
					out <- model.StreamEvent{Type: model.StreamContentDelta, Content: delta.Text}
				}
			case "response.completed":
				var r struct {
					Status string `json:"status"`
					Usage  struct {
						InputTokens  int `json:"input_tokens"`
						OutputTokens int `json:"output_tokens"`
					} `json:"usage"`
					Incomplete struct {
						Reason string `json:"reason"`
					} `json:"incomplete_details"`
				}
				_ = json.Unmarshal(payload.Response, &r)
				stopReason = r.Status
				if r.Incomplete.Reason != "" {
					stopReason = r.Incomplete.Reason
				}
				usage = &model.Usage{InputTokens: r.Usage.InputTokens, OutputTokens: r.Usage.OutputTokens}
				if !sentStart {
					out <- model.StreamEvent{Type: model.StreamMessageStart, ID: messageID, Model: modelName}
				}
				out <- model.StreamEvent{Type: model.StreamMessageStop, StopReason: stopReason, Usage: usage}
			case "response.failed", "error":
				var e struct {
					Message string `json:"message"`
					Code    string `json:"code"`
				}
				_ = json.Unmarshal(payload.Response, &e)
				out <- model.StreamEvent{Type: model.StreamError, Error: fmt.Errorf("%s: %s", e.Code, e.Message)}
				return
			}
		}
	}()

	return out, nil
}

func (c *Codex) upstreamError(status int, body []byte) error {
	var oe struct {
		Error struct {
			Message string `json:"message"`
			Type    string `json:"type"`
			Code    string `json:"code"`
		} `json:"error"`
	}
	if err := json.Unmarshal(body, &oe); err == nil && oe.Error.Message != "" {
		msg := oe.Error.Message
		if oe.Error.Code != "" {
			msg = oe.Error.Code + ": " + msg
		}
		return &UpstreamError{StatusCode: status, Retryable: isRetryStatus(status), AuthFailed: isAuthStatus(status), Message: msg}
	}
	return &UpstreamError{StatusCode: status, Retryable: isRetryStatus(status), AuthFailed: isAuthStatus(status), Message: strings.TrimSpace(string(body))}
}

var _ Provider = (*Codex)(nil)

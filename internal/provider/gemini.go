package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/tuanp-github/unified-ai-proxy/internal/config"
	"github.com/tuanp-github/unified-ai-proxy/internal/model"
)

// Gemini is the Google Gemini provider authenticated via a static API key.
// It speaks the native Gemini generateContent API.
type Gemini struct {
	transport
}

// NewGemini constructs a Gemini provider.
func NewGemini(cfg config.ProviderConfig, timeout time.Duration) *Gemini {
	var models []model.Model
	for _, m := range cfg.Models {
		models = append(models, model.Model{ID: m.ID, Upstream: m.Upstream, Provider: "gemini", ContextWindow: m.ContextWindow, MaxTokens: m.MaxTokens})
	}
	return &Gemini{transport: newTransport("gemini", cfg, models, timeout)}
}

func (g *Gemini) endpoint(model, action string) string {
	base := strings.TrimRight(g.cfg.API.BaseURL, "/")
	return fmt.Sprintf("%s/v1beta/models/%s:%s", base, url.PathEscape(model), action)
}

type geminiPart struct {
	Text string `json:"text"`
}

type geminiContent struct {
	Role  string       `json:"role"`
	Parts []geminiPart `json:"parts"`
}

type geminiGenerationConfig struct {
	Temperature     *float64 `json:"temperature,omitempty"`
	TopP            *float64 `json:"topP,omitempty"`
	MaxOutputTokens int      `json:"maxOutputTokens,omitempty"`
	StopSequences   []string `json:"stopSequences,omitempty"`
}

type geminiRequest struct {
	Contents          []geminiContent         `json:"contents"`
	SystemInstruction *geminiContent          `json:"systemInstruction,omitempty"`
	GenerationConfig  *geminiGenerationConfig `json:"generationConfig,omitempty"`
}

func (g *Gemini) buildRequest(req *model.ChatRequest) *geminiRequest {
	out := &geminiRequest{}
	if req.System != "" {
		out.SystemInstruction = &geminiContent{Role: "user", Parts: []geminiPart{{Text: req.System}}}
	}
	for _, m := range req.Messages {
		role := "user"
		if m.Role == model.RoleAssistant {
			role = "model"
		}
		out.Contents = append(out.Contents, geminiContent{
			Role:  role,
			Parts: []geminiPart{{Text: m.Content}},
		})
	}
	gc := &geminiGenerationConfig{
		Temperature:     req.Temperature,
		TopP:            req.TopP,
		MaxOutputTokens: req.MaxTokens,
		StopSequences:   req.StopSequences,
	}
	if gc.Temperature != nil || gc.TopP != nil || gc.MaxOutputTokens != 0 || len(gc.StopSequences) > 0 {
		out.GenerationConfig = gc
	}
	return out
}

func (g *Gemini) headers(key string) map[string]string {
	return map[string]string{"x-goog-api-key": key}
}

// ValidateAccount verifies the account has a non-empty API key.
func (g *Gemini) ValidateAccount(ctx context.Context, account model.Account) error {
	if strings.TrimSpace(account.APIKey) == "" {
		return fmt.Errorf("account %q has no API key", account.Name)
	}
	return nil
}

// ChatCompletion performs a non-streaming generateContent request.
func (g *Gemini) ChatCompletion(ctx context.Context, account model.Account, req *model.ChatRequest) (*model.ChatResponse, error) {
	if strings.TrimSpace(account.APIKey) == "" {
		return nil, fmt.Errorf("account %q has no API key", account.Name)
	}
	status, body, err := g.doJSON(ctx, http.MethodPost, g.endpoint(req.Model, "generateContent"), g.headers(account.APIKey), g.buildRequest(req))
	if err != nil {
		return nil, networkError(err)
	}
	if status != http.StatusOK {
		return nil, g.upstreamError(status, body)
	}

	var resp struct {
		Candidates []struct {
			Content struct {
				Parts []geminiPart `json:"parts"`
			} `json:"content"`
			FinishReason string `json:"finishReason"`
		} `json:"candidates"`
		UsageMetadata struct {
			PromptTokenCount     int `json:"promptTokenCount"`
			CandidatesTokenCount int `json:"candidatesTokenCount"`
			TotalTokenCount      int `json:"totalTokenCount"`
		} `json:"usageMetadata"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("parse gemini response: %w", err)
	}

	var text strings.Builder
	finishReason := ""
	for _, cand := range resp.Candidates {
		for _, p := range cand.Content.Parts {
			text.WriteString(p.Text)
		}
		if cand.FinishReason != "" && finishReason == "" {
			finishReason = cand.FinishReason
		}
	}

	return &model.ChatResponse{
		Model:      req.Model,
		Content:    text.String(),
		StopReason: finishReason,
		Usage: model.Usage{
			InputTokens:  resp.UsageMetadata.PromptTokenCount,
			OutputTokens: resp.UsageMetadata.CandidatesTokenCount,
		},
	}, nil
}

// StreamChatCompletion streams a streamGenerateContent request.
func (g *Gemini) StreamChatCompletion(ctx context.Context, account model.Account, req *model.ChatRequest) (<-chan model.StreamEvent, error) {
	if strings.TrimSpace(account.APIKey) == "" {
		return nil, fmt.Errorf("account %q has no API key", account.Name)
	}
	data, err := json.Marshal(g.buildRequest(req))
	if err != nil {
		return nil, err
	}

	endpoint := g.endpoint(req.Model, "streamGenerateContent") + "?alt=sse"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(string(data)))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "text/event-stream")
	for k, v := range g.headers(account.APIKey) {
		httpReq.Header.Set(k, v)
	}

	resp, err := g.client.Do(httpReq)
	if err != nil {
		return nil, networkError(err)
	}
	if resp.StatusCode != http.StatusOK {
		defer resp.Body.Close()
		return nil, g.upstreamError(resp.StatusCode, readBody(resp))
	}

	out := make(chan model.StreamEvent)
	go func() {
		defer close(out)
		defer resp.Body.Close()

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
				Candidates []struct {
					Content struct {
						Parts []geminiPart `json:"parts"`
					} `json:"content"`
					FinishReason string `json:"finishReason"`
				} `json:"candidates"`
				UsageMetadata struct {
					PromptTokenCount     int `json:"promptTokenCount"`
					CandidatesTokenCount int `json:"candidatesTokenCount"`
					TotalTokenCount      int `json:"totalTokenCount"`
				} `json:"usageMetadata"`
				Error struct {
					Code    int    `json:"code"`
					Message string `json:"message"`
					Status  string `json:"status"`
				} `json:"error"`
			}
			if err := json.Unmarshal([]byte(ev.Data), &payload); err != nil {
				continue
			}

			if payload.Error.Message != "" {
				out <- model.StreamEvent{Type: model.StreamError, Error: fmt.Errorf("gemini %d (%s): %s", payload.Error.Code, payload.Error.Status, payload.Error.Message)}
				return
			}

			var text strings.Builder
			finishReason := ""
			for _, cand := range payload.Candidates {
				for _, p := range cand.Content.Parts {
					text.WriteString(p.Text)
				}
				if cand.FinishReason != "" {
					finishReason = cand.FinishReason
				}
			}

			if text.Len() > 0 {
				if !sentStart {
					out <- model.StreamEvent{Type: model.StreamMessageStart, Model: req.Model}
					sentStart = true
				}
				out <- model.StreamEvent{Type: model.StreamContentDelta, Content: text.String()}
			}
			if finishReason != "" {
				if !sentStart {
					out <- model.StreamEvent{Type: model.StreamMessageStart, Model: req.Model}
					sentStart = true
				}
				usage := &model.Usage{
					InputTokens:  payload.UsageMetadata.PromptTokenCount,
					OutputTokens: payload.UsageMetadata.CandidatesTokenCount,
				}
				out <- model.StreamEvent{Type: model.StreamMessageStop, StopReason: finishReason, Usage: usage}
				return
			}
		}
	}()

	return out, nil
}

func (g *Gemini) upstreamError(status int, body []byte) error {
	var ge struct {
		Error struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
			Status  string `json:"status"`
		} `json:"error"`
	}
	msg := strings.TrimSpace(string(body))
	if err := json.Unmarshal(body, &ge); err == nil && ge.Error.Message != "" {
		msg = ge.Error.Message
		if ge.Error.Status != "" {
			msg = ge.Error.Status + ": " + msg
		}
	}
	return &UpstreamError{
		StatusCode: status,
		Retryable:  isRetryStatus(status),
		AuthFailed: isAuthStatus(status),
		Message:    msg,
	}
}

var _ Provider = (*Gemini)(nil)

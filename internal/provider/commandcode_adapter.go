package provider

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/tuanp-github/unified-ai-proxy/internal/config"
	"github.com/tuanp-github/unified-ai-proxy/internal/model"
	"github.com/tuanp-github/unified-ai-proxy/internal/tokenstore"
)

// commandCodeVersion is the upstream CLI version reported in requests. It is
// agent-chosen and can be made configurable later if upstream requires it.
const commandCodeVersion = "0.25.7"

// maxNDJSONLine is the maximum accepted length of a single upstream NDJSON line.
const maxNDJSONLine = 1 << 20

// CommandCode is the Command Code provider authenticated via a browser
// `user_...` API key. It speaks the `/alpha/generate` endpoint and translates
// the AI SDK v5 NDJSON response into normalized stream events.
type CommandCode struct {
	transport
}

// NewCommandCode constructs a Command Code provider.
// The static registry is enriched from Command Code's authenticated catalog.
func NewCommandCode(cfg config.ProviderConfig, timeout time.Duration) *CommandCode {
	models := append([]model.Model(nil), commandCodeModels...)
	if len(cfg.Accounts) > 0 {
		account := model.Account{Provider: "command_code", Name: cfg.Accounts[0].Name, TokenFile: cfg.Accounts[0].TokenFile}
		discoveryTimeout := timeout
		if discoveryTimeout <= 0 {
			discoveryTimeout = 2 * time.Minute
		}
		ctx, cancel := context.WithTimeout(context.Background(), discoveryTimeout)
		models = discoverCommandCodeModels(ctx, cfg, account, models)
		cancel()
	}
	return &CommandCode{transport: newTransport("command_code", cfg, models, timeout)}
}

type commandCodeModelList struct {
	Data []struct {
		ID            string `json:"id"`
		ContextLength int    `json:"context_length"`
	} `json:"data"`
}

func discoverCommandCodeModels(ctx context.Context, cfg config.ProviderConfig, account model.Account, models []model.Model) []model.Model {
	token, err := tokenstore.Load(account.TokenFile)
	if err != nil || token == nil || token.AccessToken == "" {
		return models
	}
	endpoint := strings.TrimRight(cfg.API.BaseURL, "/") + "/provider/v1/models"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return models
	}
	req.Header.Set("Authorization", "Bearer "+token.AccessToken)
	req.Header.Set("Accept", "application/json")
	resp, err := (&http.Client{}).Do(req)
	if err != nil {
		return models
	}
	defer resp.Body.Close()
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return models
	}
	var list commandCodeModelList
	if err := json.NewDecoder(resp.Body).Decode(&list); err != nil {
		return models
	}
	contexts := make(map[string]int, len(list.Data))
	for _, remote := range list.Data {
		if remote.ID != "" && remote.ContextLength > 0 {
			contexts[remote.ID] = remote.ContextLength
		}
	}
	for i := range models {
		if limit := contexts[models[i].Upstream]; limit > 0 {
			models[i].ContextWindow = limit
		}
	}
	return models
}

func (c *CommandCode) endpoint() string {
	return strings.TrimRight(c.cfg.API.BaseURL, "/") + "/alpha/generate"
}

// apiKey loads and validates the account's persisted API key. It never returns
// the token file contents in an error message.
func (c *CommandCode) apiKey(account model.Account) (string, error) {
	ts, err := tokenstore.Load(account.TokenFile)
	if err != nil {
		return "", c.authError("token file unreadable")
	}
	if ts == nil || !strings.HasPrefix(ts.AccessToken, "user_") {
		return "", c.authError("missing or invalid API key")
	}
	return ts.AccessToken, nil
}

func (c *CommandCode) authError(msg string) *UpstreamError {
	return &UpstreamError{StatusCode: http.StatusUnauthorized, AuthFailed: true, Retryable: false, Message: msg}
}

// ValidateAccount checks the local token file without contacting upstream.
func (c *CommandCode) ValidateAccount(ctx context.Context, account model.Account) error {
	_, err := c.apiKey(account)
	return err
}

func (c *CommandCode) headers(sessionID, token string) map[string]string {
	return map[string]string{
		"Authorization":          "Bearer " + token,
		"x-session-id":           sessionID,
		"x-command-code-version": commandCodeVersion,
		"x-cli-environment":      "cli",
		"Accept":                 "text/event-stream",
	}
}

// ChatCompletion performs a non-streaming request by aggregating the internal
// NDJSON stream (BR-002: every upstream request uses stream=true).
func (c *CommandCode) ChatCompletion(ctx context.Context, account model.Account, req *model.ChatRequest) (*model.ChatResponse, error) {
	ch, err := c.StreamChatCompletion(ctx, account, req)
	if err != nil {
		return nil, err
	}

	resp := &model.ChatResponse{Model: req.Model}
	var content strings.Builder
	var toolCalls []model.ToolCall
	for ev := range ch {
		switch ev.Type {
		case model.StreamContentDelta:
			content.WriteString(ev.Content)
		case model.StreamToolCall:
			if ev.ToolCall != nil {
				toolCalls = append(toolCalls, *ev.ToolCall)
			}
		case model.StreamMessageStop:
			resp.Content = content.String()
			resp.ToolCalls = toolCalls
			resp.StopReason = ev.StopReason
			if ev.Usage != nil {
				resp.Usage = *ev.Usage
			}
		case model.StreamError:
			return nil, ev.Error
		}
	}
	if resp.StopReason == "" && resp.Content == "" {
		return nil, fmt.Errorf("stream ended without completion")
	}
	return resp, nil
}

// StreamChatCompletion streams an /alpha/generate request and translates the
// NDJSON response into normalized stream events.
func (c *CommandCode) StreamChatCompletion(ctx context.Context, account model.Account, req *model.ChatRequest) (<-chan model.StreamEvent, error) {
	token, err := c.apiKey(account)
	if err != nil {
		return nil, err
	}

	sessionID, err := randomUUID()
	if err != nil {
		return nil, err
	}

	data, err := json.Marshal(c.buildRequest(req, sessionID))
	if err != nil {
		return nil, err
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint(), strings.NewReader(string(data)))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	for k, v := range c.headers(sessionID, token) {
		httpReq.Header.Set(k, v)
	}

	resp, err := c.client.Do(httpReq)
	if err != nil {
		return nil, networkError(err)
	}
	if resp.StatusCode != http.StatusOK {
		defer resp.Body.Close()
		return nil, c.upstreamError(resp.StatusCode, readBody(resp), token)
	}
	stream, err := commandCodeStreamPreflight(resp.Body)
	if err != nil {
		_ = resp.Body.Close()
		return nil, err
	}

	out := make(chan model.StreamEvent)
	go c.parseNDJSON(ctx, stream, req.Model, token, out)
	return out, nil
}

// commandCodeStreamPreflight reads exactly one NDJSON frame before exposing a
// stream downstream. Command Code may return a gateway error in a 200 response;
// treating that frame as a typed error preserves proxy retry and OpenAI error
// semantics instead of starting an empty Responses stream.
func commandCodeStreamPreflight(body io.Reader) (io.Reader, error) {
	reader := bufio.NewReader(body)
	line, err := reader.ReadBytes('\n')
	if err != nil && len(line) == 0 {
		if err == io.EOF {
			return nil, NewUpstreamError(http.StatusBadGateway, "upstream stream ended before first event")
		}
		return nil, networkError(err)
	}

	trimmed := bytes.TrimSpace(line)
	if len(trimmed) > 0 {
		var event ndjsonEvent
		if json.Unmarshal(trimmed, &event) == nil && event.Type == "error" {
			message := decodeErrorMessage(event.Error, event.Message)
			if strings.Contains(strings.ToLower(message), "gateway") {
				upstreamErr := NewUpstreamError(http.StatusBadGateway, message)
				return nil, upstreamErr
			}
		}
	}
	return io.MultiReader(bytes.NewReader(line), reader), nil
}

func mustRandomHex(n int) string {
	s, err := randomHex(n)
	if err != nil {
		return fmt.Sprintf("%x", time.Now().UnixNano())
	}
	return s
}

var _ Provider = (*CommandCode)(nil)

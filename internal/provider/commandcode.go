package provider

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"runtime"
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
	base
}

// NewCommandCode constructs a Command Code provider.
// Models come from the hardcoded registry (commandCodeModels), not from config.
func NewCommandCode(cfg config.ProviderConfig, timeout time.Duration) *CommandCode {
	return &CommandCode{base: newBase("command_code", cfg, commandCodeModels, timeout)}
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

// RefreshToken returns the current API key when valid; there is no refresh flow.
func (c *CommandCode) RefreshToken(ctx context.Context, account model.Account) (*model.TokenSet, error) {
	if _, err := c.apiKey(account); err != nil {
		return nil, err
	}
	ts, err := tokenstore.Load(account.TokenFile)
	if err != nil {
		return nil, c.authError("token file unreadable")
	}
	return ts, nil
}

type commandCodeContentBlock struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type commandCodeMessage struct {
	Role    string                    `json:"role"`
	Content []commandCodeContentBlock `json:"content"`
}

type commandCodeConfig struct {
	WorkingDir    string   `json:"workingDir"`
	Date          string   `json:"date"`
	Environment   string   `json:"environment"`
	Structure     []string `json:"structure"`
	IsGitRepo     bool     `json:"isGitRepo"`
	CurrentBranch string   `json:"currentBranch"`
	MainBranch    string   `json:"mainBranch"`
	GitStatus     string   `json:"gitStatus"`
	RecentCommits []string `json:"recentCommits"`
}

type commandCodeParams struct {
	Model       string               `json:"model"`
	Messages    []commandCodeMessage `json:"messages"`
	System      string               `json:"system,omitempty"`
	Stream      bool                 `json:"stream"`
	MaxTokens   int                  `json:"max_tokens"`
	Temperature *float64             `json:"temperature,omitempty"`
	TopP        *float64             `json:"top_p,omitempty"`
}

type commandCodeRequest struct {
	ThreadID string            `json:"threadId"`
	Memory   string            `json:"memory"`
	Config   commandCodeConfig `json:"config"`
	Params   commandCodeParams `json:"params"`
}

func (c *CommandCode) buildRequest(req *model.ChatRequest, sessionID string) *commandCodeRequest {
	params := commandCodeParams{
		Model:       req.Model,
		System:      req.System,
		Stream:      true,
		MaxTokens:   req.MaxTokens,
		Temperature: req.Temperature,
		TopP:        req.TopP,
	}
	for _, m := range req.Messages {
		role := string(m.Role)
		if role == "system" || role == "developer" {
			continue
		}
		params.Messages = append(params.Messages, commandCodeMessage{
			Role:    role,
			Content: []commandCodeContentBlock{{Type: "text", Text: m.Content}},
		})
	}
	return &commandCodeRequest{
		ThreadID: sessionID,
		Memory:   "",
		Config:   c.buildConfig(),
		Params:   params,
	}
}

func (c *CommandCode) buildConfig() commandCodeConfig {
	wd, _ := os.Getwd()
	return commandCodeConfig{
		WorkingDir:    wd,
		Date:          time.Now().UTC().Format("2006-01-02"),
		Environment:   runtime.GOOS,
		Structure:     []string{},
		IsGitRepo:     false,
		CurrentBranch: "",
		MainBranch:    "",
		GitStatus:     "",
		RecentCommits: []string{},
	}
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
	for ev := range ch {
		switch ev.Type {
		case model.StreamContentDelta:
			content.WriteString(ev.Content)
		case model.StreamMessageStop:
			resp.Content = content.String()
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

	out := make(chan model.StreamEvent)
	go c.parseNDJSON(ctx, resp.Body, req.Model, token, out)
	return out, nil
}

// ndjsonEvent is a single AI SDK v5 event line. Recognized fields are decoded
// separately so a wrong-typed field can be reported as a terminal error.
type ndjsonEvent struct {
	Type         string          `json:"type"`
	Text         json.RawMessage `json:"text"`
	Delta        json.RawMessage `json:"delta"`
	FinishReason json.RawMessage `json:"finishReason"`
	Usage        json.RawMessage `json:"usage"`
	TotalUsage   json.RawMessage `json:"totalUsage"`
	Error        json.RawMessage `json:"error"`
	Message      json.RawMessage `json:"message"`
}

type ndjsonUsage struct {
	InputTokens  int `json:"inputTokens"`
	OutputTokens int `json:"outputTokens"`
	TotalTokens  int `json:"totalTokens"`
}

func (c *CommandCode) parseNDJSON(ctx context.Context, r io.Reader, upstreamModel, apiKey string, out chan<- model.StreamEvent) {
	defer close(out)
	defer func() {
		if c, ok := r.(io.Closer); ok {
			_ = c.Close()
		}
	}()

	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 64*1024), maxNDJSONLine)

	messageID := "chatcmpl-" + mustRandomHex(16)
	sentStart := false
	stopSent := false

	emitStart := func() {
		if sentStart {
			return
		}
		out <- model.StreamEvent{Type: model.StreamMessageStart, ID: messageID, Model: upstreamModel}
		sentStart = true
	}
	fail := func(err error) {
		out <- model.StreamEvent{Type: model.StreamError, Error: fmt.Errorf("%s", sanitizeMessage(err.Error(), apiKey))}
	}

	for scanner.Scan() {
		select {
		case <-ctx.Done():
			return
		default:
		}

		line := bytes.TrimSpace(scanner.Bytes())
		if len(line) == 0 {
			continue
		}

		var ev ndjsonEvent
		if err := json.Unmarshal(line, &ev); err != nil {
			continue // non-JSON garbage is ignored
		}

		switch ev.Type {
		case "start":
			emitStart()
		case "text-delta":
			text, err := decodeTextDelta(ev.Text, ev.Delta)
			if err != nil {
				fail(err)
				return
			}
			if text != "" {
				emitStart()
				out <- model.StreamEvent{Type: model.StreamContentDelta, Content: text}
			}
		case "finish-step":
			reason, ok := decodeStringField(ev.FinishReason)
			if !ok {
				fail(errors.New("finish-step event has non-string finishReason"))
				return
			}
			usage, ok := decodeUsage(ev.Usage)
			if !ok {
				fail(errors.New("finish-step event has malformed usage"))
				return
			}
			emitStart()
			if !stopSent {
				stopSent = true
				out <- model.StreamEvent{
					Type:       model.StreamMessageStop,
					StopReason: mapCommandCodeStopReason(reason),
					Usage:      usage,
				}
			}
		case "finish":
			reason, ok := decodeStringField(ev.FinishReason)
			if !ok {
				fail(errors.New("finish event has non-string finishReason"))
				return
			}
			usage, ok := decodeUsage(ev.TotalUsage)
			if !ok {
				fail(errors.New("finish event has malformed totalUsage"))
				return
			}
			emitStart()
			if !stopSent {
				stopSent = true
				out <- model.StreamEvent{
					Type:       model.StreamMessageStop,
					StopReason: mapCommandCodeStopReason(reason),
					Usage:      usage,
				}
			}
		case "error":
			fail(errors.New(decodeErrorMessage(ev.Error, ev.Message)))
			return
		default:
			// Unknown event types are ignored.
		}
	}

	if err := scanner.Err(); err != nil {
		if errors.Is(err, bufio.ErrTooLong) {
			fail(errors.New("upstream NDJSON line exceeds limit"))
		} else {
			fail(fmt.Errorf("upstream NDJSON read error: %w", err))
		}
		return
	}
	if !stopSent {
		fail(errors.New("stream ended before terminal event"))
	}
}

func mapCommandCodeStopReason(reason string) string {
	switch reason {
	case "stop":
		return "end_turn"
	case "length":
		return "max_tokens"
	default:
		return reason
	}
}

func decodeTextDelta(textRaw, deltaRaw json.RawMessage) (string, error) {
	if len(textRaw) > 0 && string(textRaw) != "null" {
		var s string
		if err := json.Unmarshal(textRaw, &s); err != nil {
			return "", errors.New("text-delta event has non-string text field")
		}
		return s, nil
	}
	if len(deltaRaw) > 0 && string(deltaRaw) != "null" {
		var s string
		if err := json.Unmarshal(deltaRaw, &s); err != nil {
			return "", errors.New("text-delta event has non-string delta field")
		}
		return s, nil
	}
	return "", nil
}

func decodeStringField(raw json.RawMessage) (string, bool) {
	if len(raw) == 0 || string(raw) == "null" {
		return "", true
	}
	var s string
	if err := json.Unmarshal(raw, &s); err != nil {
		return "", false
	}
	return s, true
}

func decodeUsage(raw json.RawMessage) (*model.Usage, bool) {
	if len(raw) == 0 || string(raw) == "null" {
		return nil, true
	}
	var u ndjsonUsage
	if err := json.Unmarshal(raw, &u); err != nil {
		return nil, false
	}
	return &model.Usage{InputTokens: u.InputTokens, OutputTokens: u.OutputTokens}, true
}

func decodeErrorMessage(errorRaw, messageRaw json.RawMessage) string {
	if len(messageRaw) > 0 && string(messageRaw) != "null" {
		var s string
		if err := json.Unmarshal(messageRaw, &s); err == nil && s != "" {
			return s
		}
	}
	if len(errorRaw) > 0 && string(errorRaw) != "null" {
		var s string
		if err := json.Unmarshal(errorRaw, &s); err == nil && s != "" {
			return s
		}
		var obj map[string]any
		if err := json.Unmarshal(errorRaw, &obj); err == nil {
			if m, ok := obj["message"].(string); ok && m != "" {
				return m
			}
		}
	}
	return "unknown upstream error"
}

// unknownModelRe matches upstream messages indicating the model is not in the
// catalog (used only to classify a 400 as unsupported_model).
var unknownModelRe = regexp.MustCompile(`(?i)model.{0,40}(not found|not supported|does not exist|unknown|unavailable|not in catalog|invalid model)`)

// isUnknownModelError reports whether a non-200 body is a structured error that
// confirms the requested model is not in the upstream catalog.
func isUnknownModelError(body []byte) bool {
	var probe struct {
		Code    string          `json:"code"`
		Message string          `json:"message"`
		Error   json.RawMessage `json:"error"`
	}
	if json.Unmarshal(body, &probe) != nil {
		return false
	}
	code, msg := probe.Code, probe.Message
	if len(probe.Error) > 0 {
		var e struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		}
		if json.Unmarshal(probe.Error, &e) == nil {
			if code == "" {
				code = e.Code
			}
			if msg == "" {
				msg = e.Message
			}
		}
	}
	switch strings.ToLower(code) {
	case "unsupported_model", "model_not_found":
		return true
	}
	return unknownModelRe.MatchString(msg)
}

// isPlanRestricted reports whether the error indicates the model is not
// available in the current plan (Command Code returns 403 FORBIDDEN with
// code "MODEL_NOT_IN_PLAN").
func isPlanRestricted(body []byte) bool {
	var probe struct {
		Code    string          `json:"code"`
		Error   json.RawMessage `json:"error"`
		Message string          `json:"message"`
	}
	if json.Unmarshal(body, &probe) != nil {
		return false
	}
	if strings.EqualFold(probe.Code, "FORBIDDEN") {
		return true
	}
	if strings.Contains(probe.Message, "MODEL_NOT_IN_PLAN") {
		return true
	}
	if len(probe.Error) > 0 {
		var e struct {
			Code string `json:"code"`
		}
		if json.Unmarshal(probe.Error, &e) == nil && strings.EqualFold(e.Code, "FORBIDDEN") {
			return true
		}
	}
	return false
}

func (c *CommandCode) upstreamError(status int, body []byte, apiKey string) error {
	msg := extractErrorMessage(body)
	msg = sanitizeMessage(msg, apiKey)

	e := &UpstreamError{
		StatusCode:       status,
		Retryable:        isRetryStatus(status),
		AuthFailed:       isAuthStatus(status),
		UnsupportedModel: isUnknownModelError(body),
		PlanRestricted:   isPlanRestricted(body),
		Message:          msg,
	}
	return e
}

// extractErrorMessage pulls a structured error message from common error shapes,
// falling back to the raw body.
func extractErrorMessage(body []byte) string {
	var probe struct {
		Message string          `json:"message"`
		Error   json.RawMessage `json:"error"`
	}
	if err := json.Unmarshal(body, &probe); err != nil {
		return strings.TrimSpace(string(body))
	}
	if probe.Message != "" {
		return probe.Message
	}
	if len(probe.Error) > 0 {
		var em struct {
			Message string `json:"message"`
		}
		if json.Unmarshal(probe.Error, &em) == nil && em.Message != "" {
			return em.Message
		}
		var es string
		if json.Unmarshal(probe.Error, &es) == nil && es != "" {
			return es
		}
	}
	return strings.TrimSpace(string(body))
}

// sanitizeMessage redacts the active API key and bearer credential from any
// upstream message before it is exposed or logged.
func sanitizeMessage(msg, apiKey string) string {
	if apiKey == "" {
		return msg
	}
	msg = strings.ReplaceAll(msg, "Bearer "+apiKey, "Bearer [REDACTED]")
	msg = strings.ReplaceAll(msg, apiKey, "[REDACTED]")
	return msg
}

func mustRandomHex(n int) string {
	s, err := randomHex(n)
	if err != nil {
		return fmt.Sprintf("%x", time.Now().UnixNano())
	}
	return s
}

var _ Provider = (*CommandCode)(nil)

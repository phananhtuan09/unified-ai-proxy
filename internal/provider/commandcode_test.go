package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/tuanp-github/unified-ai-proxy/internal/config"
	"github.com/tuanp-github/unified-ai-proxy/internal/model"
	"github.com/tuanp-github/unified-ai-proxy/internal/tokenstore"
)

func testCommandCode(baseURL string) *CommandCode {
	cfg := config.ProviderConfig{
		Auth: config.AuthConfig{
			Method:           "browser_key",
			AuthorizationURL: "https://commandcode.ai/studio/auth/cli",
			RedirectPort:     1458,
			RedirectPath:     "/callback",
		},
		API: config.APIConfig{BaseURL: baseURL},
		Models: []config.ModelConfig{
			{ID: "cc-deepseek-v4-flash", Upstream: "deepseek/deepseek-v4-flash"},
		},
	}
	return NewCommandCode(cfg, 10*time.Second)
}

func writeCCToken(t *testing.T, ts *model.TokenSet) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "cc.json")
	if err := tokenstore.Save(path, ts); err != nil {
		t.Fatalf("save token: %v", err)
	}
	return path
}

func testCCAccount(tokenFile string) model.Account {
	return model.Account{Provider: "command_code", Name: "main", TokenFile: tokenFile}
}

func testCCReq() *model.ChatRequest {
	temp := 0.7
	topP := 0.9
	return &model.ChatRequest{
		Model:         "deepseek/deepseek-v4-flash",
		System:        "be concise",
		MaxTokens:     256,
		Temperature:   &temp,
		TopP:          &topP,
		StopSequences: []string{"END"},
		Metadata:      map[string]any{"user": "x"},
		Messages: []model.Message{
			{Role: model.RoleUser, Content: "hi"},
			{Role: model.RoleAssistant, Content: "hello"},
		},
	}
}

func TestCommandCodeBuildRequest(t *testing.T) {
	c := testCommandCode("http://example.com")
	out := c.buildRequest(testCCReq(), "session-123")

	if out.ThreadID != "session-123" {
		t.Fatalf("threadId should equal session id, got %q", out.ThreadID)
	}
	if out.Memory != "" {
		t.Fatalf("memory should be empty, got %q", out.Memory)
	}
	if out.Params.Stream != true {
		t.Fatalf("params.stream must be true, got %v", out.Params.Stream)
	}
	if out.Params.Model != "deepseek/deepseek-v4-flash" {
		t.Fatalf("params.model mismatch: %q", out.Params.Model)
	}
	if out.Params.System != "be concise" {
		t.Fatalf("params.system mismatch: %q", out.Params.System)
	}
	if out.Params.MaxTokens != 256 {
		t.Fatalf("params.max_tokens mismatch: %d", out.Params.MaxTokens)
	}
	if out.Params.Temperature == nil || *out.Params.Temperature != 0.7 {
		t.Fatalf("temperature mismatch: %+v", out.Params.Temperature)
	}
	if out.Params.TopP == nil || *out.Params.TopP != 0.9 {
		t.Fatalf("top_p mismatch: %+v", out.Params.TopP)
	}
	if len(out.Params.Messages) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(out.Params.Messages))
	}
	m := out.Params.Messages[0]
	if m.Role != "user" || len(m.Content) != 1 || m.Content[0].Type != "text" || m.Content[0].Text != "hi" {
		t.Fatalf("unexpected message: %+v", m)
	}
	if out.Config.WorkingDir == "" {
		t.Fatal("config.workingDir should be set")
	}
	if out.Config.Environment == "" {
		t.Fatal("config.environment should be set")
	}
	if out.Config.Date == "" {
		t.Fatal("config.date should be set")
	}
}

func TestCommandCodeBuildRequestSkipsSystemMessages(t *testing.T) {
	c := testCommandCode("http://example.com")
	req := testCCReq()
	req.Messages = append([]model.Message{{Role: model.RoleSystem, Content: "sys"}}, req.Messages...)
	out := c.buildRequest(req, "s")
	for _, m := range out.Params.Messages {
		if m.Role == "system" || m.Role == "developer" {
			t.Fatalf("system/developer must not appear in messages: %+v", m)
		}
	}
}

func TestCommandCodeBuildRequestDropsStopAndMetadata(t *testing.T) {
	c := testCommandCode("http://example.com")
	out := c.buildRequest(testCCReq(), "s")
	data, err := json.Marshal(out)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "stop") {
		t.Fatalf("stop/stop_sequences must not be sent upstream: %s", data)
	}
	if strings.Contains(string(data), "metadata") {
		t.Fatalf("metadata must not be sent upstream: %s", data)
	}
}

func TestCommandCodeCredentialErrors(t *testing.T) {
	c := testCommandCode("http://example.com")

	cases := []struct {
		name string
		path string
	}{
		{"missing file", filepath.Join(t.TempDir(), "nope.json")},
		{"parse error", func() string {
			p := filepath.Join(t.TempDir(), "bad.json")
			if err := os.WriteFile(p, []byte("{not json"), 0o600); err != nil {
				t.Fatal(err)
			}
			return p
		}()},
		{"empty access token", writeCCToken(t, &model.TokenSet{})},
		{"wrong prefix", writeCCToken(t, &model.TokenSet{AccessToken: "sk-not-user"})},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			acc := testCCAccount(tc.path)
			if err := c.ValidateAccount(context.Background(), acc); !IsAuthFailure(err) {
				t.Fatalf("ValidateAccount: expected auth failure, got %v", err)
			}
			if _, err := c.RefreshToken(context.Background(), acc); !IsAuthFailure(err) {
				t.Fatalf("RefreshToken: expected auth failure, got %v", err)
			}
			if _, err := c.ChatCompletion(context.Background(), acc, testCCReq()); !IsAuthFailure(err) {
				t.Fatalf("ChatCompletion: expected auth failure, got %v", err)
			}
			if _, err := c.StreamChatCompletion(context.Background(), acc, testCCReq()); !IsAuthFailure(err) {
				t.Fatalf("StreamChatCompletion: expected auth failure, got %v", err)
			}
		})
	}
}

func TestCommandCodeValidateAccountValid(t *testing.T) {
	c := testCommandCode("http://example.com")
	path := writeCCToken(t, &model.TokenSet{AccessToken: "user_valid", TokenType: "Bearer"})
	if err := c.ValidateAccount(context.Background(), testCCAccount(path)); err != nil {
		t.Fatalf("expected valid, got %v", err)
	}
}

func TestCommandCodeChatCompletion(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertCommandCodeHeaders(t, r, "user_test")
		w.Header().Set("Content-Type", "application/x-ndjson")
		fmt.Fprint(w, "{\"type\":\"start\"}\n")
		fmt.Fprint(w, "{\"type\":\"text-delta\",\"text\":\"Hello\"}\n")
		fmt.Fprint(w, "{\"type\":\"text-delta\",\"text\":\" world\"}\n")
		fmt.Fprint(w, "{\"type\":\"finish-step\",\"finishReason\":\"stop\",\"usage\":{\"inputTokens\":7,\"outputTokens\":2,\"totalTokens\":9}}\n")
	}))
	defer srv.Close()

	c := testCommandCode(srv.URL)
	path := writeCCToken(t, &model.TokenSet{AccessToken: "user_test"})
	resp, err := c.ChatCompletion(context.Background(), testCCAccount(path), testCCReq())
	if err != nil {
		t.Fatalf("ChatCompletion: %v", err)
	}
	if resp.Content != "Hello world" {
		t.Fatalf("unexpected content %q", resp.Content)
	}
	if resp.StopReason != "end_turn" {
		t.Fatalf("unexpected stop reason %q", resp.StopReason)
	}
	if resp.Usage.InputTokens != 7 || resp.Usage.OutputTokens != 2 {
		t.Fatalf("unexpected usage %+v", resp.Usage)
	}
}

func TestCommandCodeChatCompletionErrorDiscardsPartial(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "{\"type\":\"text-delta\",\"text\":\"partial\"}\n")
		fmt.Fprint(w, "{\"type\":\"error\",\"error\":{\"message\":\"boom\"}}\n")
	}))
	defer srv.Close()

	c := testCommandCode(srv.URL)
	path := writeCCToken(t, &model.TokenSet{AccessToken: "user_test"})
	if _, err := c.ChatCompletion(context.Background(), testCCAccount(path), testCCReq()); err == nil {
		t.Fatal("expected error")
	}
}

func TestCommandCodeStreamSuccess(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertCommandCodeHeaders(t, r, "user_test")
		fmt.Fprint(w, "{\"type\":\"start\"}\n")
		fmt.Fprint(w, "{\"type\":\"text-delta\",\"text\":\"Hello\"}\n")
		fmt.Fprint(w, "{\"type\":\"text-delta\",\"text\":\" world\"}\n")
		fmt.Fprint(w, "{\"type\":\"finish-step\",\"finishReason\":\"stop\",\"usage\":{\"inputTokens\":7,\"outputTokens\":2}}\n")
		fmt.Fprint(w, "{\"type\":\"finish\",\"totalUsage\":{\"inputTokens\":7,\"outputTokens\":2}}\n")
	}))
	defer srv.Close()

	c := testCommandCode(srv.URL)
	path := writeCCToken(t, &model.TokenSet{AccessToken: "user_test"})
	ch, err := c.StreamChatCompletion(context.Background(), testCCAccount(path), testCCReq())
	if err != nil {
		t.Fatalf("StreamChatCompletion: %v", err)
	}

	var events []model.StreamEvent
	for ev := range ch {
		events = append(events, ev)
	}

	if len(events) != 4 {
		t.Fatalf("expected 4 events (start, delta, delta, stop), got %d: %+v", len(events), events)
	}
	if events[0].Type != model.StreamMessageStart {
		t.Fatalf("first event should be start, got %v", events[0].Type)
	}
	if !strings.HasPrefix(events[0].ID, "chatcmpl-") {
		t.Fatalf("start id should have chatcmpl prefix, got %q", events[0].ID)
	}
	if events[1].Type != model.StreamContentDelta || events[1].Content != "Hello" {
		t.Fatalf("unexpected delta: %+v", events[1])
	}
	if events[2].Type != model.StreamContentDelta || events[2].Content != " world" {
		t.Fatalf("unexpected delta: %+v", events[2])
	}
	if events[3].Type != model.StreamMessageStop || events[3].StopReason != "end_turn" {
		t.Fatalf("unexpected stop: %+v", events[3])
	}
	if events[3].Usage == nil || events[3].Usage.InputTokens != 7 || events[3].Usage.OutputTokens != 2 {
		t.Fatalf("unexpected usage: %+v", events[3].Usage)
	}
}

func TestCommandCodeStreamLengthStopReason(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "{\"type\":\"finish\",\"finishReason\":\"length\",\"totalUsage\":{\"inputTokens\":1,\"outputTokens\":1}}\n")
	}))
	defer srv.Close()

	c := testCommandCode(srv.URL)
	path := writeCCToken(t, &model.TokenSet{AccessToken: "user_test"})
	ch, err := c.StreamChatCompletion(context.Background(), testCCAccount(path), testCCReq())
	if err != nil {
		t.Fatalf("StreamChatCompletion: %v", err)
	}
	var events []model.StreamEvent
	for ev := range ch {
		events = append(events, ev)
	}
	if len(events) != 2 {
		t.Fatalf("expected start + stop, got %+v", events)
	}
	if events[1].StopReason != "max_tokens" {
		t.Fatalf("expected max_tokens stop reason, got %q", events[1].StopReason)
	}
}

func TestCommandCodeStreamErrorEvent(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "{\"type\":\"text-delta\",\"text\":\"partial\"}\n")
		fmt.Fprint(w, "{\"type\":\"error\",\"error\":{\"message\":\"upstream exploded\"}}\n")
	}))
	defer srv.Close()

	c := testCommandCode(srv.URL)
	path := writeCCToken(t, &model.TokenSet{AccessToken: "user_test"})
	ch, err := c.StreamChatCompletion(context.Background(), testCCAccount(path), testCCReq())
	if err != nil {
		t.Fatalf("StreamChatCompletion: %v", err)
	}
	var events []model.StreamEvent
	for ev := range ch {
		events = append(events, ev)
	}
	last := events[len(events)-1]
	if last.Type != model.StreamError {
		t.Fatalf("expected terminal error, got %+v", events)
	}
	if !strings.Contains(last.Error.Error(), "upstream exploded") {
		t.Fatalf("unexpected error message %q", last.Error.Error())
	}
	for i, ev := range events[:len(events)-1] {
		if ev.Type == model.StreamMessageStop {
			t.Fatalf("stop must not appear before error, got %+v at %d", ev, i)
		}
	}
}

func TestCommandCodeStreamEOFBeforeTerminal(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "{\"type\":\"text-delta\",\"text\":\"no end\"}\n")
	}))
	defer srv.Close()

	c := testCommandCode(srv.URL)
	path := writeCCToken(t, &model.TokenSet{AccessToken: "user_test"})
	ch, err := c.StreamChatCompletion(context.Background(), testCCAccount(path), testCCReq())
	if err != nil {
		t.Fatalf("StreamChatCompletion: %v", err)
	}
	var events []model.StreamEvent
	for ev := range ch {
		events = append(events, ev)
	}
	if len(events) == 0 || events[len(events)-1].Type != model.StreamError {
		t.Fatalf("expected terminal error on EOF, got %+v", events)
	}
}

func TestCommandCodeStreamLineTooLong(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, strings.Repeat("a", maxNDJSONLine+1))
		fmt.Fprint(w, "\n")
	}))
	defer srv.Close()

	c := testCommandCode(srv.URL)
	path := writeCCToken(t, &model.TokenSet{AccessToken: "user_test"})
	ch, err := c.StreamChatCompletion(context.Background(), testCCAccount(path), testCCReq())
	if err != nil {
		t.Fatalf("StreamChatCompletion: %v", err)
	}
	var events []model.StreamEvent
	for ev := range ch {
		events = append(events, ev)
	}
	if len(events) == 0 || events[len(events)-1].Type != model.StreamError {
		t.Fatalf("expected terminal error for oversized line, got %+v", events)
	}
}

func TestCommandCodeStreamIgnoresGarbageAndUnknown(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "\n")
		fmt.Fprint(w, "not-json\n")
		fmt.Fprint(w, "{\"type\":\"reasoning-delta\",\"text\":\"thinking\"}\n")
		fmt.Fprint(w, "{\"type\":\"text-delta\",\"text\":\"ok\"}\n")
		fmt.Fprint(w, "{\"type\":\"finish\",\"finishReason\":\"stop\"}\n")
	}))
	defer srv.Close()

	c := testCommandCode(srv.URL)
	path := writeCCToken(t, &model.TokenSet{AccessToken: "user_test"})
	ch, err := c.StreamChatCompletion(context.Background(), testCCAccount(path), testCCReq())
	if err != nil {
		t.Fatalf("StreamChatCompletion: %v", err)
	}
	var contents []string
	for ev := range ch {
		if ev.Type == model.StreamContentDelta {
			contents = append(contents, ev.Content)
		}
	}
	if len(contents) != 1 || contents[0] != "ok" {
		t.Fatalf("garbage/unknown events must be ignored, got %+v", contents)
	}
}

func TestCommandCodeStreamKnownEventWrongShape(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "{\"type\":\"text-delta\",\"text\":123}\n")
	}))
	defer srv.Close()

	c := testCommandCode(srv.URL)
	path := writeCCToken(t, &model.TokenSet{AccessToken: "user_test"})
	ch, err := c.StreamChatCompletion(context.Background(), testCCAccount(path), testCCReq())
	if err != nil {
		t.Fatalf("StreamChatCompletion: %v", err)
	}
	var events []model.StreamEvent
	for ev := range ch {
		events = append(events, ev)
	}
	if len(events) == 0 || events[len(events)-1].Type != model.StreamError {
		t.Fatalf("expected terminal error for wrong shape, got %+v", events)
	}
}

func TestCommandCodeUpstreamErrorMapping(t *testing.T) {
	c := testCommandCode("http://example.com")
	key := "user_secret"

	cases := []struct {
		status   int
		body     string
		wantAuth bool
		wantRetr bool
		wantUnsp bool
	}{
		{401, `{"error":{"message":"bad key"}}`, true, false, false},
		{403, `{"error":{"message":"forbidden"}}`, true, false, false},
		{429, `{"error":{"message":"slow down"}}`, false, true, false},
		{503, `{"error":{"message":"down"}}`, false, true, false},
		{400, `{"error":{"message":"bad request"}}`, false, false, false},
		{400, `{"error":{"code":"unsupported_model","message":"model not found"}}`, false, false, true},
	}

	for _, tc := range cases {
		err := c.upstreamError(tc.status, []byte(tc.body), key)
		ue, ok := err.(*UpstreamError)
		if !ok {
			t.Fatalf("expected UpstreamError, got %T", err)
		}
		if ue.AuthFailed != tc.wantAuth {
			t.Errorf("status %d: auth=%v want %v", tc.status, ue.AuthFailed, tc.wantAuth)
		}
		if ue.Retryable != tc.wantRetr {
			t.Errorf("status %d: retryable=%v want %v", tc.status, ue.Retryable, tc.wantRetr)
		}
		if ue.UnsupportedModel != tc.wantUnsp {
			t.Errorf("status %d: unsupported=%v want %v", tc.status, ue.UnsupportedModel, tc.wantUnsp)
		}
	}
}

func TestCommandCodeRedaction(t *testing.T) {
	key := "user_secret_key_123"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		fmt.Fprintf(w, `{"error":{"message":"token user_secret_key_123 is invalid"}}`)
	}))
	defer srv.Close()

	c := testCommandCode(srv.URL)
	path := writeCCToken(t, &model.TokenSet{AccessToken: key})
	_, err := c.ChatCompletion(context.Background(), testCCAccount(path), testCCReq())
	if err == nil {
		t.Fatal("expected error")
	}
	if strings.Contains(err.Error(), key) {
		t.Fatalf("api key leaked in error: %v", err)
	}
}

func TestCommandCodeStreamRedaction(t *testing.T) {
	key := "user_secret_key_456"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "{\"type\":\"error\",\"error\":{\"message\":\"Bearer %s revoked\"}}\n", key)
	}))
	defer srv.Close()

	c := testCommandCode(srv.URL)
	path := writeCCToken(t, &model.TokenSet{AccessToken: key})
	ch, err := c.StreamChatCompletion(context.Background(), testCCAccount(path), testCCReq())
	if err != nil {
		t.Fatalf("StreamChatCompletion: %v", err)
	}
	for ev := range ch {
		if ev.Type == model.StreamError && strings.Contains(ev.Error.Error(), key) {
			t.Fatalf("api key leaked in stream error: %v", ev.Error)
		}
	}
}

func assertCommandCodeHeaders(t *testing.T, r *http.Request, wantKey string) {
	t.Helper()
	if got := r.Header.Get("Authorization"); got != "Bearer "+wantKey {
		t.Errorf("Authorization mismatch: %q", got)
	}
	if got := r.Header.Get("x-session-id"); got == "" {
		t.Error("x-session-id header missing")
	}
	if got := r.Header.Get("x-command-code-version"); got != "0.25.7" {
		t.Errorf("x-command-code-version mismatch: %q", got)
	}
	if got := r.Header.Get("x-cli-environment"); got != "cli" {
		t.Errorf("x-cli-environment mismatch: %q", got)
	}
	if got := r.Header.Get("Accept"); got != "text/event-stream" {
		t.Errorf("Accept mismatch: %q", got)
	}
	var body commandCodeRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		t.Errorf("decode request body: %v", err)
		return
	}
	if body.ThreadID != r.Header.Get("x-session-id") {
		t.Errorf("threadId %q should equal x-session-id %q", body.ThreadID, r.Header.Get("x-session-id"))
	}
	if !body.Params.Stream {
		t.Errorf("params.stream must be true even for non-stream clients")
	}
}

func TestCommandCodeSessionIDIsUUID(t *testing.T) {
	id, err := randomUUID()
	if err != nil {
		t.Fatalf("randomUUID: %v", err)
	}
	if !isCanonicalUUID(id) {
		t.Fatalf("session id %q is not a canonical dashed UUID", id)
	}
}

// isCanonicalUUID reports whether s matches the RFC 4122 8-4-4-4-12 hex shape.
func isCanonicalUUID(s string) bool {
	if len(s) != 36 {
		return false
	}
	for i, c := range s {
		if i == 8 || i == 13 || i == 18 || i == 23 {
			if c != '-' {
				return false
			}
			continue
		}
		if !strings.ContainsRune("0123456789abcdef", c) {
			return false
		}
	}
	return true
}

package server

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/tuanp-github/unified-ai-proxy/internal/accounts"
	"github.com/tuanp-github/unified-ai-proxy/internal/config"
	"github.com/tuanp-github/unified-ai-proxy/internal/model"
	"github.com/tuanp-github/unified-ai-proxy/internal/provider"
	"github.com/tuanp-github/unified-ai-proxy/internal/proxy"
	"github.com/tuanp-github/unified-ai-proxy/internal/tokenstore"
)

type ccIntegration struct {
	client *http.Client
	url    string
}

func startCCIntegration(t *testing.T, upstream http.HandlerFunc) *ccIntegration {
	t.Helper()

	upstreamSrv := httptest.NewServer(upstream)
	t.Cleanup(upstreamSrv.Close)

	tokenFile := filepath.Join(t.TempDir(), "cc.json")
	if err := tokenstore.Save(tokenFile, &model.TokenSet{AccessToken: "user_integration", TokenType: "Bearer"}); err != nil {
		t.Fatalf("save token: %v", err)
	}

	cfg := &config.Config{
		Server: config.ServerConfig{APIKeys: []string{"sk-test"}, DefaultMaxTokens: 100},
		Providers: map[string]config.ProviderConfig{
			"command_code": {
				Enabled: true,
				Auth: config.AuthConfig{
					Method:           "browser_key",
					AuthorizationURL: "https://commandcode.ai/studio/auth/cli",
					RedirectPort:     1458,
					RedirectPath:     "/callback",
				},
				API:      config.APIConfig{BaseURL: upstreamSrv.URL},
				Models:   []config.ModelConfig{{ID: "cc-deepseek-v4-flash", Upstream: "deepseek/deepseek-v4-flash"}},
				Accounts: []config.AccountConfig{{Name: "main", TokenFile: tokenFile}},
			},
		},
		Routing: config.RoutingConfig{Failover: config.FailoverConfig{Enabled: false}},
	}

	mgr := accounts.New(time.Minute)
	svc, err := proxy.New(cfg, mgr, provider.Build)
	if err != nil {
		t.Fatalf("proxy.New: %v", err)
	}
	srv := New(cfg, svc, nil)
	httpSrv := httptest.NewServer(srv.Handler())
	t.Cleanup(httpSrv.Close)

	return &ccIntegration{client: httpSrv.Client(), url: httpSrv.URL}
}

func (c *ccIntegration) do(t *testing.T, method, path, body string) (*http.Response, string) {
	t.Helper()
	req, err := http.NewRequest(method, c.url+path, strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer sk-test")
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)
	return resp, string(data)
}

func ndjsonSuccess(w http.ResponseWriter) {
	fmt.Fprint(w, "{\"type\":\"start\"}\n")
	fmt.Fprint(w, "{\"type\":\"text-delta\",\"text\":\"Hello\"}\n")
	fmt.Fprint(w, "{\"type\":\"text-delta\",\"text\":\" world\"}\n")
	fmt.Fprint(w, "{\"type\":\"finish-step\",\"finishReason\":\"stop\",\"usage\":{\"inputTokens\":7,\"outputTokens\":2,\"totalTokens\":9}}\n")
}

func ndjsonError(w http.ResponseWriter) {
	fmt.Fprint(w, "{\"type\":\"text-delta\",\"text\":\"partial\"}\n")
	fmt.Fprint(w, "{\"type\":\"error\",\"error\":{\"message\":\"boom\"}}\n")
}

func TestCCIntegrationOpenAINonStream(t *testing.T) {
	c := startCCIntegration(t, func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		if params, ok := body["params"].(map[string]any); ok && params["stream"] != true {
			t.Errorf("upstream must always receive stream=true")
		}
		ndjsonSuccess(w)
	})

	resp, data := c.do(t, http.MethodPost, "/v1/chat/completions",
		`{"model":"cc-deepseek-v4-flash","messages":[{"role":"user","content":"Hi"}],"stream":false}`)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status %d: %s", resp.StatusCode, data)
	}
	if !strings.Contains(data, "Hello world") {
		t.Fatalf("missing content: %s", data)
	}
	if !strings.Contains(data, `"finish_reason":"stop"`) {
		t.Fatalf("missing finish_reason: %s", data)
	}
}

func TestCCIntegrationOpenAIStream(t *testing.T) {
	c := startCCIntegration(t, func(w http.ResponseWriter, r *http.Request) {
		ndjsonSuccess(w)
	})

	resp, data := c.do(t, http.MethodPost, "/v1/chat/completions",
		`{"model":"cc-deepseek-v4-flash","messages":[{"role":"user","content":"Hi"}],"stream":true}`)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status %d: %s", resp.StatusCode, data)
	}
	if !strings.Contains(data, "Hello") || !strings.Contains(data, " world") {
		t.Fatalf("missing streamed content: %s", data)
	}
	if !strings.Contains(data, "[DONE]") {
		t.Fatalf("missing [DONE]: %s", data)
	}
}

func TestCCIntegrationOpenAIStreamErrorTerminal(t *testing.T) {
	c := startCCIntegration(t, func(w http.ResponseWriter, r *http.Request) {
		ndjsonError(w)
	})

	resp, data := c.do(t, http.MethodPost, "/v1/chat/completions",
		`{"model":"cc-deepseek-v4-flash","messages":[{"role":"user","content":"Hi"}],"stream":true}`)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status %d: %s", resp.StatusCode, data)
	}
	if !strings.Contains(data, "boom") {
		t.Fatalf("missing error payload: %s", data)
	}
	if strings.Contains(data, "[DONE]") {
		t.Fatalf("must not emit [DONE] after error: %s", data)
	}
	if strings.Contains(data, `"finish_reason":"stop"`) {
		t.Fatalf("must not emit success finish after error: %s", data)
	}
}

func TestCCIntegrationAnthropicNonStream(t *testing.T) {
	c := startCCIntegration(t, func(w http.ResponseWriter, r *http.Request) {
		ndjsonSuccess(w)
	})

	resp, data := c.do(t, http.MethodPost, "/v1/messages",
		`{"model":"cc-deepseek-v4-flash","messages":[{"role":"user","content":[{"type":"text","text":"Hi"}]}],"max_tokens":100}`)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status %d: %s", resp.StatusCode, data)
	}
	if !strings.Contains(data, `"type":"message"`) {
		t.Fatalf("missing message type: %s", data)
	}
	if !strings.Contains(data, "Hello world") {
		t.Fatalf("missing content: %s", data)
	}
	if !strings.Contains(data, `"stop_reason":"end_turn"`) {
		t.Fatalf("missing stop_reason: %s", data)
	}
}

func TestCCIntegrationAnthropicStreamErrorTerminal(t *testing.T) {
	c := startCCIntegration(t, func(w http.ResponseWriter, r *http.Request) {
		ndjsonError(w)
	})

	resp, data := c.do(t, http.MethodPost, "/v1/messages",
		`{"model":"cc-deepseek-v4-flash","messages":[{"role":"user","content":[{"type":"text","text":"Hi"}]}],"stream":true,"max_tokens":100}`)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status %d: %s", resp.StatusCode, data)
	}
	if !strings.Contains(data, "boom") {
		t.Fatalf("missing error event: %s", data)
	}
	if strings.Contains(data, "message_stop") {
		t.Fatalf("must not emit message_stop after error: %s", data)
	}
}

func TestCCIntegrationUnknownModelAlias(t *testing.T) {
	c := startCCIntegration(t, func(w http.ResponseWriter, r *http.Request) {
		ndjsonSuccess(w)
	})

	resp, data := c.do(t, http.MethodPost, "/v1/chat/completions",
		`{"model":"cc-does-not-exist","messages":[{"role":"user","content":"Hi"}]}`)
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", resp.StatusCode, data)
	}
	if !strings.Contains(data, "model_not_found") {
		t.Fatalf("missing code: %s", data)
	}
}

func TestCCIntegrationUnknownModelUpstream(t *testing.T) {
	c := startCCIntegration(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprint(w, `{"error":{"code":"unsupported_model","message":"model not found"}}`)
	})

	resp, data := c.do(t, http.MethodPost, "/v1/chat/completions",
		`{"model":"cc-deepseek-v4-flash","messages":[{"role":"user","content":"Hi"}]}`)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", resp.StatusCode, data)
	}
	if !strings.Contains(data, "unsupported_model") {
		t.Fatalf("missing unsupported_model code: %s", data)
	}
}

func TestCCIntegrationGenericBadRequest(t *testing.T) {
	c := startCCIntegration(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprint(w, `{"error":{"message":"bad request"}}`)
	})

	resp, data := c.do(t, http.MethodPost, "/v1/chat/completions",
		`{"model":"cc-deepseek-v4-flash","messages":[{"role":"user","content":"Hi"}]}`)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", resp.StatusCode, data)
	}
	if !strings.Contains(data, "invalid_request") {
		t.Fatalf("missing invalid_request code: %s", data)
	}
	if strings.Contains(data, "unsupported_model") {
		t.Fatalf("generic 400 must not be classified as unsupported_model: %s", data)
	}
}

func TestCCIntegrationAuthErrorRedacted(t *testing.T) {
	c := startCCIntegration(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		fmt.Fprint(w, `{"error":{"message":"api key user_integration is revoked"}}`)
	})

	resp, data := c.do(t, http.MethodPost, "/v1/chat/completions",
		`{"model":"cc-deepseek-v4-flash","messages":[{"role":"user","content":"Hi"}]}`)
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", resp.StatusCode, data)
	}
	if strings.Contains(data, "user_integration") {
		t.Fatalf("api key leaked: %s", data)
	}
	if !strings.Contains(data, "provider_auth_failed") {
		t.Fatalf("missing provider_auth_failed code: %s", data)
	}
}

func TestCCIntegrationRedactionInStream(t *testing.T) {
	c := startCCIntegration(t, func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "{\"type\":\"error\",\"error\":{\"message\":\"Bearer user_integration revoked\"}}\n")
	})

	resp, data := c.do(t, http.MethodPost, "/v1/chat/completions",
		`{"model":"cc-deepseek-v4-flash","messages":[{"role":"user","content":"Hi"}],"stream":true}`)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status %d: %s", resp.StatusCode, data)
	}
	if strings.Contains(data, "user_integration") {
		t.Fatalf("api key leaked in stream: %s", data)
	}
}

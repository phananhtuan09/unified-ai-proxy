package provider

import (
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/tuanp-github/unified-ai-proxy/internal/config"
	"github.com/tuanp-github/unified-ai-proxy/internal/model"
)

func TestBuildBrowserKeyURL(t *testing.T) {
	cfg := config.ProviderConfig{
		Auth: config.AuthConfig{
			AuthorizationURL: "https://commandcode.ai/studio/auth/cli",
			RedirectHost:     "localhost",
			RedirectPort:     1458,
			RedirectPath:     "/callback",
		},
	}
	u := buildBrowserKeyURL(cfg, "http://localhost:1458/callback", "abc123")
	parsed, err := url.Parse(u)
	if err != nil {
		t.Fatalf("parse url: %v", err)
	}
	if parsed.Scheme != "https" || parsed.Host != "commandcode.ai" || parsed.Path != "/studio/auth/cli" {
		t.Fatalf("unexpected base url %q", u)
	}
	if got := parsed.Query().Get("callback"); got != "http://localhost:1458/callback" {
		t.Fatalf("callback mismatch: %q", got)
	}
	if got := parsed.Query().Get("state"); got != "abc123" {
		t.Fatalf("state mismatch: %q", got)
	}
}

func TestReadCallbackCredentialsQuery(t *testing.T) {
	r := httptest.NewRequest("GET", "/callback?state=abc&apiKey=user_x", nil)
	state, key := readCallbackCredentials(r)
	if state != "abc" || key != "user_x" {
		t.Fatalf("unexpected (%q, %q)", state, key)
	}
}

func TestReadCallbackCredentialsBody(t *testing.T) {
	r := httptest.NewRequest("POST", "/callback", strings.NewReader(`{"state":"abc","apiKey":"user_x","userId":"u"}`))
	state, key := readCallbackCredentials(r)
	if state != "abc" || key != "user_x" {
		t.Fatalf("unexpected (%q, %q)", state, key)
	}
}

func newCallbackSession(state string) *browserKeySession {
	return &browserKeySession{
		state: state,
		key:   make(chan *model.TokenSet, 1),
		err:   make(chan error, 1),
	}
}

func TestBrowserKeyCallbackValid(t *testing.T) {
	sess := newCallbackSession("abc123")
	r := httptest.NewRequest("POST", "/callback", strings.NewReader(`{"state":"abc123","apiKey":"user_valid"}`))
	w := httptest.NewRecorder()
	sess.handleCallback()(w, r)

	select {
	case ts := <-sess.key:
		if ts.AccessToken != "user_valid" {
			t.Fatalf("unexpected api key %q", ts.AccessToken)
		}
	default:
		t.Fatal("expected key to be delivered")
	}
	if strings.Contains(w.Body.String(), "user_valid") {
		t.Fatalf("response must not echo the api key: %q", w.Body.String())
	}
}

func TestBrowserKeyCallbackStateMismatch(t *testing.T) {
	sess := newCallbackSession("abc123")
	r := httptest.NewRequest("POST", "/callback", strings.NewReader(`{"state":"wrong","apiKey":"user_valid"}`))
	w := httptest.NewRecorder()
	sess.handleCallback()(w, r)

	if w.Code != 400 {
		t.Fatalf("expected 400, got %d", w.Code)
	}
	select {
	case <-sess.key:
		t.Fatal("must not deliver key on state mismatch")
	case err := <-sess.err:
		if !strings.Contains(err.Error(), "state mismatch") {
			t.Fatalf("unexpected error %v", err)
		}
	default:
		t.Fatal("expected an error")
	}
}

func TestBrowserKeyCallbackInvalidKey(t *testing.T) {
	sess := newCallbackSession("abc123")
	r := httptest.NewRequest("POST", "/callback", strings.NewReader(`{"state":"abc123","apiKey":"not_user"}`))
	w := httptest.NewRecorder()
	sess.handleCallback()(w, r)

	if w.Code != 400 {
		t.Fatalf("expected 400, got %d", w.Code)
	}
	select {
	case <-sess.key:
		t.Fatal("must not deliver key on invalid api key")
	case err := <-sess.err:
		if !strings.Contains(err.Error(), "invalid api key") {
			t.Fatalf("unexpected error %v", err)
		}
	default:
		t.Fatal("expected an error")
	}
}

func TestBrowserKeyCallbackMissingKey(t *testing.T) {
	sess := newCallbackSession("abc123")
	r := httptest.NewRequest("POST", "/callback", strings.NewReader(`{"state":"abc123"}`))
	w := httptest.NewRecorder()
	sess.handleCallback()(w, r)

	if w.Code != 400 {
		t.Fatalf("expected 400, got %d", w.Code)
	}
	select {
	case <-sess.key:
		t.Fatal("must not deliver key when api key missing")
	default:
	}
}

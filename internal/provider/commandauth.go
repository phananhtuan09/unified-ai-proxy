package provider

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/tuanp-github/unified-ai-proxy/internal/config"
	"github.com/tuanp-github/unified-ai-proxy/internal/model"
	"github.com/tuanp-github/unified-ai-proxy/internal/tokenstore"
)

// browserKeySession holds the in-flight browser login state for the
// command_code `browser_key` auth flow.
type browserKeySession struct {
	provider string
	account  string
	state    string
	key      chan *model.TokenSet
	err      chan error
}

// RunBrowserKeyLogin performs the Command Code browser login flow. It opens the
// studio auth page and waits for the studio to POST the `user_...` API key back
// to a loopback callback server. Unlike OAuth, no token exchange occurs: the
// API key returned by the callback is persisted directly.
func RunBrowserKeyLogin(ctx context.Context, provider, account, tokenFile string, cfg config.ProviderConfig) (*model.TokenSet, error) {
	if !strings.EqualFold(cfg.Auth.Method, "browser_key") {
		return nil, fmt.Errorf("provider %q auth method %q is not browser_key", provider, cfg.Auth.Method)
	}

	state, err := randomHex(32)
	if err != nil {
		return nil, err
	}

	sess := &browserKeySession{
		provider: provider,
		account:  account,
		state:    state,
		key:      make(chan *model.TokenSet, 1),
		err:      make(chan error, 1),
	}

	redirectURI := redirectURI(cfg)

	mux := http.NewServeMux()
	mux.HandleFunc(cfg.Auth.RedirectPath, sess.handleCallback())

	// Bind only to loopback regardless of the advertised redirect host.
	server := &http.Server{
		Addr:    fmt.Sprintf("127.0.0.1:%d", cfg.Auth.RedirectPort),
		Handler: mux,
	}

	errCh := make(chan error, 1)
	go func() {
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- fmt.Errorf("callback server: %w", err)
		}
	}()

	// Give the listener a moment to bind.
	time.Sleep(200 * time.Millisecond)
	select {
	case err := <-errCh:
		return nil, err
	default:
	}

	authURL := buildBrowserKeyURL(cfg, redirectURI, state)
	fmt.Printf("Opening browser to authorize provider %q account %q...\n", provider, account)
	fmt.Printf("If your browser does not open, visit:\n  %s\n", authURL)
	if err := openBrowser(authURL); err != nil {
		fmt.Printf("Could not open browser automatically: %v\n", err)
	}

	timeout := time.NewTimer(5 * time.Minute)
	defer timeout.Stop()

	select {
	case ts := <-sess.key:
		shutdown(server)
		ts.Provider = provider
		ts.Account = account
		ts.TokenType = orDefault(ts.TokenType, "Bearer")
		if err := tokenstore.Save(tokenFile, ts); err != nil {
			return nil, err
		}
		return ts, nil
	case err := <-sess.err:
		shutdown(server)
		return nil, err
	case err := <-errCh:
		shutdown(server)
		return nil, err
	case <-timeout.C:
		shutdown(server)
		return nil, fmt.Errorf("authorization timed out")
	case <-ctx.Done():
		shutdown(server)
		return nil, ctx.Err()
	}
}

func buildBrowserKeyURL(cfg config.ProviderConfig, redirectURI, state string) string {
	u, _ := url.Parse(cfg.Auth.AuthorizationURL)
	q := u.Query()
	q.Set("callback", redirectURI)
	q.Set("state", state)
	u.RawQuery = q.Encode()
	return u.String()
}

func (s *browserKeySession) handleCallback() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// The studio page POSTs from a public origin to a loopback server.
		// Answer Private Network Access preflight so the browser allows it.
		origin := "https://commandcode.ai"
		if o := r.Header.Get("Origin"); strings.HasSuffix(o, "commandcode.ai") {
			origin = o
		}
		w.Header().Set("Access-Control-Allow-Origin", origin)
		w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		w.Header().Set("Access-Control-Allow-Private-Network", "true")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		state, apiKey := readCallbackCredentials(r)

		if subtle.ConstantTimeCompare([]byte(state), []byte(s.state)) != 1 {
			http.Error(w, "state mismatch", http.StatusBadRequest)
			s.fail(fmt.Errorf("login callback state mismatch"))
			return
		}
		if !strings.HasPrefix(apiKey, "user_") {
			http.Error(w, "invalid api key", http.StatusBadRequest)
			s.fail(fmt.Errorf("login callback returned invalid api key"))
			return
		}

		fmt.Fprintf(w, "<html><body>Authorization complete. You can close this window.</body></html>")
		s.send(&model.TokenSet{AccessToken: apiKey})
	}
}

// readCallbackCredentials reads the state and apiKey from the callback query or
// a JSON body, matching the studio/auth/cli callback shape.
func readCallbackCredentials(r *http.Request) (state, apiKey string) {
	q := r.URL.Query()
	state = q.Get("state")
	apiKey = q.Get("apiKey")

	if state != "" && apiKey != "" {
		return state, apiKey
	}

	body, _ := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	var payload struct {
		State  string `json:"state"`
		APIKey string `json:"apiKey"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return state, apiKey
	}
	if state == "" {
		state = payload.State
	}
	if apiKey == "" {
		apiKey = payload.APIKey
	}
	return state, apiKey
}

// send delivers the key without blocking on a repeated callback after the
// session has already completed.
func (s *browserKeySession) send(ts *model.TokenSet) {
	select {
	case s.key <- ts:
	default:
	}
}

func (s *browserKeySession) fail(err error) {
	select {
	case s.err <- err:
	default:
	}
}

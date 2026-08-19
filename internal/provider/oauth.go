package provider

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/tuanp-github/unified-ai-proxy/internal/config"
	"github.com/tuanp-github/unified-ai-proxy/internal/model"
	"github.com/tuanp-github/unified-ai-proxy/internal/tokenstore"
)

// oauthSession holds the in-flight authorization state.
type oauthSession struct {
	cfg      config.ProviderConfig
	provider string
	account  string
	state    string
	verifier string
	token    chan *model.TokenSet
	err      chan error
}

// RunOAuthLogin performs the browser OAuth flow and persists the token.
// It runs a temporary callback server bound only to 127.0.0.1.
func RunOAuthLogin(ctx context.Context, provider, account, tokenFile string, cfg config.ProviderConfig) (*model.TokenSet, error) {
	if !strings.EqualFold(cfg.Auth.Method, "oauth") {
		return nil, fmt.Errorf("provider %q auth method %q is not oauth", provider, cfg.Auth.Method)
	}

	verifier, challenge, err := newPKCE()
	if err != nil {
		return nil, err
	}
	state, err := randomHex(32)
	if err != nil {
		return nil, err
	}

	sess := &oauthSession{
		cfg:      cfg,
		provider: provider,
		account:  account,
		state:    state,
		verifier: verifier,
		token:    make(chan *model.TokenSet, 1),
		err:      make(chan error, 1),
	}

	redirectURI := redirectURI(cfg)

	mux := http.NewServeMux()
	mux.HandleFunc(cfg.Auth.RedirectPath, sess.handleCallback(redirectURI))

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

	authURL := buildAuthURL(cfg, provider, state, challenge, redirectURI)
	fmt.Printf("Opening browser to authorize provider %q account %q...\n", provider, account)
	fmt.Printf("If your browser does not open, visit:\n  %s\n", authURL)
	if err := openBrowser(authURL); err != nil {
		fmt.Printf("Could not open browser automatically: %v\n", err)
	}

	timeout := time.NewTimer(5 * time.Minute)
	defer timeout.Stop()

	select {
	case ts := <-sess.token:
		shutdown(server)
		ts.Provider = provider
		ts.Account = account
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

func shutdown(server *http.Server) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	_ = server.Shutdown(ctx)
}

func redirectURI(cfg config.ProviderConfig) string {
	path := cfg.Auth.RedirectPath
	if path == "" {
		path = "/callback"
	}
	return fmt.Sprintf("http://%s:%d%s", cfg.Auth.RedirectHost, cfg.Auth.RedirectPort, path)
}

func (s *oauthSession) handleCallback(redirectURI string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()

		if got := query.Get("state"); got != s.state {
			http.Error(w, "state mismatch", http.StatusBadRequest)
			s.err <- fmt.Errorf("OAuth state mismatch")
			return
		}
		if e := query.Get("error"); e != "" {
			desc := query.Get("error_description")
			http.Error(w, "authorization failed", http.StatusBadRequest)
			s.err <- fmt.Errorf("authorization error: %s (%s)", e, desc)
			return
		}
		code := query.Get("code")
		if code == "" {
			http.Error(w, "missing authorization code", http.StatusBadRequest)
			s.err <- fmt.Errorf("missing authorization code")
			return
		}

		ts, err := exchangeCode(r.Context(), s.cfg, code, s.verifier, s.state, redirectURI)
		if err != nil {
			http.Error(w, "token exchange failed", http.StatusInternalServerError)
			s.err <- err
			return
		}
		fmt.Fprintf(w, "<html><body>Authorization complete. You can close this window.</body></html>")
		s.token <- ts
	}
}

func buildAuthURL(cfg config.ProviderConfig, provider, state, challenge, redirectURI string) string {
	u, _ := url.Parse(cfg.Auth.AuthorizationURL)
	q := u.Query()
	q.Set("response_type", "code")
	q.Set("client_id", cfg.Auth.ClientID)
	q.Set("redirect_uri", redirectURI)
	q.Set("state", state)
	if len(cfg.Auth.Scopes) > 0 {
		q.Set("scope", strings.Join(cfg.Auth.Scopes, " "))
	}
	if cfg.Auth.PKCE {
		q.Set("code_challenge", challenge)
		q.Set("code_challenge_method", "S256")
	}
	for k, v := range extraAuthParams(provider) {
		q.Set(k, v)
	}
	u.RawQuery = q.Encode()
	return u.String()
}

// extraAuthParams are provider-specific query parameters the official OAuth
// authorization endpoints expect.
func extraAuthParams(provider string) map[string]string {
	switch provider {
	case "openai_codex":
		return map[string]string{
			"prompt":                     "login",
			"id_token_add_organizations": "true",
			"codex_cli_simplified_flow":  "true",
		}
	default:
		return nil
	}
}

func exchangeCode(ctx context.Context, cfg config.ProviderConfig, code, verifier, state, redirectURI string) (*model.TokenSet, error) {
	req, err := buildTokenRequest(ctx, cfg, code, verifier, state, redirectURI)
	if err != nil {
		return nil, err
	}

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("token exchange: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("token exchange HTTP %d: %s", resp.StatusCode, string(body))
	}

	var payload struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		TokenType    string `json:"token_type"`
		ExpiresIn    int64  `json:"expires_in"`
		Scope        string `json:"scope"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, fmt.Errorf("parse token response: %w", err)
	}
	if payload.AccessToken == "" {
		return nil, fmt.Errorf("token response missing access_token")
	}

	now := time.Now().UTC()
	ts := &model.TokenSet{
		AccessToken:  payload.AccessToken,
		RefreshToken: payload.RefreshToken,
		TokenType:    orDefault(payload.TokenType, "Bearer"),
		Scope:        payload.Scope,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	if payload.ExpiresIn > 0 {
		ts.ExpiresAt = now.Add(time.Duration(payload.ExpiresIn) * time.Second)
	}
	return ts, nil
}

// buildTokenRequest builds the authorization-code exchange request, using a
// JSON body for providers that require it (Claude) and form encoding otherwise.
func buildTokenRequest(ctx context.Context, cfg config.ProviderConfig, code, verifier, state, redirectURI string) (*http.Request, error) {
	if strings.EqualFold(cfg.Auth.ExchangeFormat, "json") {
		body := map[string]string{
			"grant_type":    "authorization_code",
			"code":          code,
			"redirect_uri":  redirectURI,
			"client_id":     cfg.Auth.ClientID,
			"code_verifier": verifier,
			"state":         state,
		}
		data, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, cfg.Auth.TokenURL, strings.NewReader(string(data)))
		if err != nil {
			return nil, err
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "application/json")
		return req, nil
	}

	form := url.Values{}
	form.Set("grant_type", "authorization_code")
	form.Set("code", code)
	form.Set("client_id", cfg.Auth.ClientID)
	form.Set("redirect_uri", redirectURI)
	if cfg.Auth.PKCE {
		form.Set("code_verifier", verifier)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, cfg.Auth.TokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")
	return req, nil
}

func newPKCE() (verifier, challenge string, err error) {
	raw := make([]byte, 32)
	if _, err = rand.Read(raw); err != nil {
		return "", "", err
	}
	verifier = base64.RawURLEncoding.EncodeToString(raw)
	sum := sha256.Sum256([]byte(verifier))
	challenge = base64.RawURLEncoding.EncodeToString(sum[:])
	return verifier, challenge, nil
}

func randomHex(n int) (string, error) {
	raw := make([]byte, n)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", raw), nil
}

// randomUUID returns a canonical RFC 4122 version 4 UUID string.
func randomUUID() (string, error) {
	raw := make([]byte, 16)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	raw[6] = (raw[6] & 0x0f) | 0x40 // version 4
	raw[8] = (raw[8] & 0x3f) | 0x80 // variant 10
	return fmt.Sprintf("%x-%x-%x-%x-%x", raw[0:4], raw[4:6], raw[6:8], raw[8:10], raw[10:16]), nil
}

func orDefault(s, def string) string {
	if s == "" {
		return def
	}
	return s
}

func openBrowser(target string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "linux":
		cmd = exec.Command("xdg-open", target)
	case "darwin":
		cmd = exec.Command("open", target)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", target)
	default:
		return fmt.Errorf("unsupported platform %q", runtime.GOOS)
	}
	return cmd.Start()
}

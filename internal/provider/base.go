package provider

import (
	"context"
	"encoding/json"
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

// base holds shared behavior for OAuth-backed upstream providers.
type base struct {
	name    string
	cfg     config.ProviderConfig
	models  []model.Model
	client  *http.Client
	timeout time.Duration
}

func newBase(name string, cfg config.ProviderConfig, models []model.Model, timeout time.Duration) base {
	if timeout <= 0 {
		timeout = 2 * time.Minute
	}
	return base{
		name:    name,
		cfg:     cfg,
		models:  models,
		client:  &http.Client{Timeout: timeout},
		timeout: timeout,
	}
}

func (b *base) Name() string          { return b.name }
func (b *base) Models() []model.Model { return b.models }

// accessToken returns a valid access token, refreshing when necessary.
func (b *base) accessToken(ctx context.Context, account model.Account) (string, error) {
	ts, err := tokenstore.Load(account.TokenFile)
	if err != nil {
		return "", err
	}
	if ts == nil {
		return "", fmt.Errorf("no token file for account %q", account.Name)
	}
	if ts.NeedsRefresh(time.Now()) {
		ts, err = b.refresh(ctx, account, ts)
		if err != nil {
			return "", err
		}
	}
	return ts.AccessToken, nil
}

// RefreshToken refreshes an account's token and persists the result.
func (b *base) RefreshToken(ctx context.Context, account model.Account) (*model.TokenSet, error) {
	ts, err := tokenstore.Load(account.TokenFile)
	if err != nil {
		return nil, err
	}
	if ts == nil {
		return nil, fmt.Errorf("no token file for account %q", account.Name)
	}
	return b.refresh(ctx, account, ts)
}

// ValidateAccount verifies an account has a usable token, refreshing if needed.
func (b *base) ValidateAccount(ctx context.Context, account model.Account) error {
	_, err := b.accessToken(ctx, account)
	return err
}

func (b *base) refresh(ctx context.Context, account model.Account, ts *model.TokenSet) (*model.TokenSet, error) {
	if ts.RefreshToken == "" {
		return nil, fmt.Errorf("account %q has no refresh token", account.Name)
	}

	req, err := b.buildRefreshRequest(ctx, ts.RefreshToken)
	if err != nil {
		return nil, err
	}

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("token refresh: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("token refresh HTTP %d: %s", resp.StatusCode, string(body))
	}

	var payload struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		TokenType    string `json:"token_type"`
		ExpiresIn    int64  `json:"expires_in"`
		Scope        string `json:"scope"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, fmt.Errorf("parse refresh response: %w", err)
	}
	if payload.AccessToken == "" {
		return nil, fmt.Errorf("refresh response missing access_token")
	}

	now := time.Now().UTC()
	ts.AccessToken = payload.AccessToken
	if payload.RefreshToken != "" {
		ts.RefreshToken = payload.RefreshToken
	}
	if payload.TokenType != "" {
		ts.TokenType = payload.TokenType
	}
	if payload.Scope != "" {
		ts.Scope = payload.Scope
	}
	ts.UpdatedAt = now
	if payload.ExpiresIn > 0 {
		ts.ExpiresAt = now.Add(time.Duration(payload.ExpiresIn) * time.Second)
	} else {
		ts.ExpiresAt = time.Time{}
	}
	if err := tokenstore.Save(account.TokenFile, ts); err != nil {
		return nil, err
	}
	return ts, nil
}

// buildRefreshRequest builds the refresh request, using a JSON body for
// providers that require it (Claude) and form encoding otherwise.
func (b *base) buildRefreshRequest(ctx context.Context, refreshToken string) (*http.Request, error) {
	scope := strings.Join(b.cfg.Auth.Scopes, " ")

	if strings.EqualFold(b.cfg.Auth.ExchangeFormat, "json") {
		body := map[string]string{
			"client_id":     b.cfg.Auth.ClientID,
			"grant_type":    "refresh_token",
			"refresh_token": refreshToken,
			"scope":         scope,
		}
		data, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, b.cfg.Auth.TokenURL, strings.NewReader(string(data)))
		if err != nil {
			return nil, err
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "application/json")
		return req, nil
	}

	form := url.Values{}
	form.Set("grant_type", "refresh_token")
	form.Set("refresh_token", refreshToken)
	form.Set("client_id", b.cfg.Auth.ClientID)
	if scope != "" {
		form.Set("scope", scope)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, b.cfg.Auth.TokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")
	return req, nil
}

// doJSON performs a JSON request and returns the raw status and body.
func (b *base) doJSON(ctx context.Context, method, endpoint string, headers map[string]string, body any) (int, []byte, error) {
	var reader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return 0, nil, err
		}
		reader = strings.NewReader(string(data))
	}
	req, err := http.NewRequestWithContext(ctx, method, endpoint, reader)
	if err != nil {
		return 0, nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	resp, err := b.client.Do(req)
	if err != nil {
		return 0, nil, err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(io.LimitReader(resp.Body, 32<<20))
	if err != nil {
		return 0, nil, err
	}
	return resp.StatusCode, data, nil
}

// readBody drains an error response body up to a bounded size.
func readBody(resp *http.Response) []byte {
	if resp.Body == nil {
		return nil
	}
	b := make([]byte, 0, 4096)
	tmp := make([]byte, 4096)
	for {
		n, err := resp.Body.Read(tmp)
		b = append(b, tmp[:n]...)
		if err != nil || len(b) > 1<<20 {
			break
		}
	}
	return b
}

func isRetryStatus(status int) bool {
	switch status {
	case 429, 500, 502, 503, 504:
		return true
	}
	return false
}

func isAuthStatus(status int) bool {
	return status == http.StatusUnauthorized || status == http.StatusForbidden
}

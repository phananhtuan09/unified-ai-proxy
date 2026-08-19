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

	"github.com/tuanp-github/unified-ai-proxy/internal/model"
	"github.com/tuanp-github/unified-ai-proxy/internal/tokenstore"
)

// oauthCapability owns OAuth token persistence and refresh for Codex.
type oauthCapability struct {
	transport
}

func (o *oauthCapability) accessToken(ctx context.Context, account model.Account) (string, error) {
	ts, err := tokenstore.Load(account.TokenFile)
	if err != nil {
		return "", err
	}
	if ts == nil {
		return "", fmt.Errorf("no token file for account %q", account.Name)
	}
	if ts.NeedsRefresh(time.Now()) {
		ts, err = o.refresh(ctx, account, ts)
		if err != nil {
			return "", err
		}
	}
	return ts.AccessToken, nil
}

func (o *oauthCapability) RefreshToken(ctx context.Context, account model.Account) (*model.TokenSet, error) {
	ts, err := tokenstore.Load(account.TokenFile)
	if err != nil {
		return nil, err
	}
	if ts == nil {
		return nil, fmt.Errorf("no token file for account %q", account.Name)
	}
	return o.refresh(ctx, account, ts)
}

func (o *oauthCapability) ValidateAccount(ctx context.Context, account model.Account) error {
	_, err := o.accessToken(ctx, account)
	return err
}

func (o *oauthCapability) refresh(ctx context.Context, account model.Account, ts *model.TokenSet) (*model.TokenSet, error) {
	if ts.RefreshToken == "" {
		return nil, fmt.Errorf("account %q has no refresh token", account.Name)
	}
	req, err := o.buildRefreshRequest(ctx, ts.RefreshToken)
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

func (o *oauthCapability) buildRefreshRequest(ctx context.Context, refreshToken string) (*http.Request, error) {
	scope := strings.Join(o.cfg.Auth.Scopes, " ")
	if strings.EqualFold(o.cfg.Auth.ExchangeFormat, "json") {
		body := map[string]string{"client_id": o.cfg.Auth.ClientID, "grant_type": "refresh_token", "refresh_token": refreshToken, "scope": scope}
		data, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, o.cfg.Auth.TokenURL, strings.NewReader(string(data)))
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
	form.Set("client_id", o.cfg.Auth.ClientID)
	if scope != "" {
		form.Set("scope", scope)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, o.cfg.Auth.TokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")
	return req, nil
}

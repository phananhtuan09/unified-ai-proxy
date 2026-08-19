package config

import (
	"fmt"
	"strings"
)

const tbd = "TBD"

// Validate enforces startup validation rules.
func (c *Config) Validate() error {
	if len(c.Server.APIKeys) == 0 {
		return fmt.Errorf("no local API keys configured")
	}
	enabled := c.EnabledProviders()
	if len(enabled) == 0 {
		return fmt.Errorf("no provider is enabled")
	}
	seenModels := map[string]string{}
	for name, p := range enabled {
		if err := c.validateProvider(name, p, seenModels); err != nil {
			return err
		}
	}
	return nil
}

func (c *Config) validateProvider(name string, p ProviderConfig, seenModels map[string]string) error {
	if len(p.Accounts) == 0 {
		return fmt.Errorf("provider %q has no accounts", name)
	}
	method := normalizeMethod(p.Auth.Method)
	switch method {
	case "oauth":
		if strings.TrimSpace(p.Auth.ClientID) == "" || strings.EqualFold(p.Auth.ClientID, tbd) {
			return fmt.Errorf("provider %q auth.client_id is unresolved (TBD)", name)
		}
		if strings.TrimSpace(p.Auth.AuthorizationURL) == "" || strings.EqualFold(p.Auth.AuthorizationURL, tbd) {
			return fmt.Errorf("provider %q auth.authorization_url is unresolved (TBD)", name)
		}
		if strings.TrimSpace(p.Auth.TokenURL) == "" || strings.EqualFold(p.Auth.TokenURL, tbd) {
			return fmt.Errorf("provider %q auth.token_url is unresolved (TBD)", name)
		}
		if p.Auth.RedirectPort <= 0 {
			return fmt.Errorf("provider %q auth.redirect_port must be a positive integer", name)
		}
	case "api_key":
	case "browser_key":
		if strings.TrimSpace(p.Auth.AuthorizationURL) == "" || strings.EqualFold(p.Auth.AuthorizationURL, tbd) {
			return fmt.Errorf("provider %q auth.authorization_url is unresolved (TBD)", name)
		}
		if p.Auth.RedirectPort <= 0 {
			return fmt.Errorf("provider %q auth.redirect_port must be a positive integer", name)
		}
	default:
		return fmt.Errorf("provider %q has unsupported auth method %q", name, method)
	}
	if strings.TrimSpace(p.API.BaseURL) == "" || strings.EqualFold(p.API.BaseURL, tbd) {
		return fmt.Errorf("provider %q api.base_url is unresolved (TBD)", name)
	}
	if name != "command_code" {
		providerModels := map[string]struct{}{}
		for _, m := range p.Models {
			if strings.TrimSpace(m.ID) == "" {
				return fmt.Errorf("provider %q has a model with empty id", name)
			}
			if strings.TrimSpace(m.Upstream) == "" || strings.EqualFold(m.Upstream, tbd) {
				return fmt.Errorf("provider %q model %q has unresolved upstream (TBD)", name, m.ID)
			}
			if _, dup := providerModels[m.ID]; dup {
				return fmt.Errorf("provider %q configures model alias %q more than once", name, m.ID)
			}
			providerModels[m.ID] = struct{}{}
			if other, exists := seenModels[m.ID]; exists {
				return fmt.Errorf("model alias %q is configured by both %q and %q", m.ID, other, name)
			}
			seenModels[m.ID] = name
		}
	}
	for _, a := range p.Accounts {
		if strings.TrimSpace(a.Name) == "" {
			return fmt.Errorf("provider %q has an account with empty name", name)
		}
		if method == "oauth" || method == "browser_key" {
			if strings.TrimSpace(a.TokenFile) == "" {
				return fmt.Errorf("provider %q account %q has no token file", name, a.Name)
			}
		} else if strings.TrimSpace(a.APIKey) == "" {
			return fmt.Errorf("provider %q account %q has no api_key", name, a.Name)
		}
	}
	return nil
}

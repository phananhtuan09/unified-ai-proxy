package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// DefaultPath is the default config location.
func DefaultPath() string {
	return filepath.Join(homeDir(), ".config", "unified-ai-proxy", "config.yaml")
}

func homeDir() string {
	if h, err := os.UserHomeDir(); err == nil && h != "" {
		return h
	}
	return "~"
}

// ExpandPath expands a leading ~ to the user's home directory.
func ExpandPath(p string) string {
	if p == "~" {
		return homeDir()
	}
	if strings.HasPrefix(p, "~/") {
		return filepath.Join(homeDir(), p[2:])
	}
	return p
}

// Duration is a time.Duration that unmarshals from strings like "5m".
type Duration time.Duration

func (d *Duration) UnmarshalYAML(value *yaml.Node) error {
	var s string
	if err := value.Decode(&s); err != nil {
		return err
	}
	v, err := time.ParseDuration(s)
	if err != nil {
		return fmt.Errorf("invalid duration %q: %w", s, err)
	}
	*d = Duration(v)
	return nil
}

func (d Duration) Duration() time.Duration { return time.Duration(d) }

// MarshalYAML renders the duration as a human-readable string like "5m".
func (d Duration) MarshalYAML() (interface{}, error) {
	s := time.Duration(d).String()
	if strings.HasSuffix(s, "m0s") {
		s = strings.TrimSuffix(s, "0s")
	}
	return s, nil
}

func (c *Config) applyDefaults() {
	if c.Server.Host == "" {
		c.Server.Host = "127.0.0.1"
	}
	if c.Server.Port == 0 {
		c.Server.Port = 8080
	}
	if c.Server.DefaultMaxTokens == 0 {
		c.Server.DefaultMaxTokens = 4096
	}
	if c.Routing.Strategy == "" {
		c.Routing.Strategy = "round-robin"
	}
	if c.Routing.Failover.MaxRetries == 0 {
		c.Routing.Failover.MaxRetries = 3
	}
	if c.Routing.Failover.UnhealthyCooldown.Duration() == 0 {
		c.Routing.Failover.UnhealthyCooldown = Duration(5 * time.Minute)
	}
	if c.Routing.Failover.RequestTimeout.Duration() == 0 {
		c.Routing.Failover.RequestTimeout = Duration(2 * time.Minute)
	}
	for name, p := range c.Providers {
		if def, ok := providerDefaults[name]; ok {
			p = mergeProviderDefaults(p, def)
		}
		if p.Auth.RedirectHost == "" {
			p.Auth.RedirectHost = "localhost"
		}
		p.Auth.Method = normalizeMethod(p.Auth.Method)
		if p.Auth.ExchangeFormat == "" {
			p.Auth.ExchangeFormat = "form"
		}
		c.Providers[name] = p
	}
}

// providerDefaults are the well-known OAuth constants for the official
// Codex CLI client. They are applied only when the config leaves a field
// empty, so a minimal config works out of the box while explicit values
// (and "TBD") are still respected.
var providerDefaults = map[string]ProviderConfig{
	"openai_codex": {
		Auth: AuthConfig{
			Method:           "oauth",
			ClientID:         "app_EMoamEEZ73f0CkXaXp7hrann",
			AuthorizationURL: "https://auth.openai.com/oauth/authorize",
			TokenURL:         "https://auth.openai.com/oauth/token",
			Scopes:           []string{"openid", "email", "profile", "offline_access"},
			RedirectHost:     "localhost",
			RedirectPort:     1455,
			RedirectPath:     "/auth/callback",
			ExchangeFormat:   "form",
			PKCE:             true,
		},
		API: APIConfig{BaseURL: "https://api.openai.com/v1"},
	},
	"gemini": {
		Auth: AuthConfig{
			Method: "api_key",
		},
		API: APIConfig{BaseURL: "https://generativelanguage.googleapis.com"},
	},
	"command_code": {
		Auth: AuthConfig{
			Method:           "browser_key",
			AuthorizationURL: "https://commandcode.ai/studio/auth/cli",
			RedirectHost:     "localhost",
			RedirectPort:     1458,
			RedirectPath:     "/callback",
		},
		API: APIConfig{BaseURL: "https://api.commandcode.ai"},
	},
}

func mergeProviderDefaults(p, def ProviderConfig) ProviderConfig {
	if p.Auth.Method == "" {
		p.Auth.Method = def.Auth.Method
	}
	if p.Auth.ClientID == "" {
		p.Auth.ClientID = def.Auth.ClientID
	}
	if p.Auth.AuthorizationURL == "" {
		p.Auth.AuthorizationURL = def.Auth.AuthorizationURL
	}
	if p.Auth.TokenURL == "" {
		p.Auth.TokenURL = def.Auth.TokenURL
	}
	if len(p.Auth.Scopes) == 0 {
		p.Auth.Scopes = def.Auth.Scopes
	}
	if p.Auth.RedirectHost == "" {
		p.Auth.RedirectHost = def.Auth.RedirectHost
	}
	if p.Auth.RedirectPort == 0 {
		p.Auth.RedirectPort = def.Auth.RedirectPort
	}
	if p.Auth.RedirectPath == "" {
		p.Auth.RedirectPath = def.Auth.RedirectPath
	}
	if p.Auth.ExchangeFormat == "" {
		p.Auth.ExchangeFormat = def.Auth.ExchangeFormat
	}
	if !p.Auth.PKCE {
		p.Auth.PKCE = def.Auth.PKCE
	}
	if p.API.BaseURL == "" {
		p.API.BaseURL = def.API.BaseURL
	}
	return p
}

func normalizeMethod(m string) string {
	if m == "" {
		return "oauth"
	}
	return m
}

// EnabledProviders returns enabled provider configs keyed by provider name.
func (c *Config) EnabledProviders() map[string]ProviderConfig {
	out := make(map[string]ProviderConfig)
	for name, p := range c.Providers {
		if p.Enabled {
			out[name] = p
		}
	}
	return out
}

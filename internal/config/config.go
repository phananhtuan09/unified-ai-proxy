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

// Config is the root configuration document.
type Config struct {
	Server    ServerConfig              `yaml:"server"`
	Providers map[string]ProviderConfig `yaml:"providers"`
	Routing   RoutingConfig             `yaml:"routing"`
}

type ServerConfig struct {
	Host             string   `yaml:"host,omitempty"`
	Port             int      `yaml:"port,omitempty"`
	APIKeys          []string `yaml:"api_keys,omitempty"`
	DefaultMaxTokens int      `yaml:"default_max_tokens,omitempty"`
}

type ProviderConfig struct {
	Enabled  bool            `yaml:"enabled,omitempty"`
	Auth     AuthConfig      `yaml:"auth,omitempty"`
	API      APIConfig       `yaml:"api,omitempty"`
	Models   []ModelConfig   `yaml:"models,omitempty"`
	Accounts []AccountConfig `yaml:"accounts,omitempty"`
}

type AuthConfig struct {
	Method           string   `yaml:"method,omitempty"`
	ClientID         string   `yaml:"client_id,omitempty"`
	AuthorizationURL string   `yaml:"authorization_url,omitempty"`
	TokenURL         string   `yaml:"token_url,omitempty"`
	Scopes           []string `yaml:"scopes,omitempty"`
	RedirectHost     string   `yaml:"redirect_host,omitempty"`
	RedirectPort     int      `yaml:"redirect_port,omitempty"`
	RedirectPath     string   `yaml:"redirect_path,omitempty"`
	ExchangeFormat   string   `yaml:"exchange_format,omitempty"` // "json" or "form"
	PKCE             bool     `yaml:"pkce,omitempty"`
}

type APIConfig struct {
	BaseURL string `yaml:"base_url,omitempty"`
}

type ModelConfig struct {
	ID       string `yaml:"id,omitempty"`
	Upstream string `yaml:"upstream,omitempty"`
}

type AccountConfig struct {
	Name      string `yaml:"name,omitempty"`
	TokenFile string `yaml:"token_file,omitempty"`
	APIKey    string `yaml:"api_key,omitempty"`
}

type RoutingConfig struct {
	Strategy string         `yaml:"strategy,omitempty"`
	Failover FailoverConfig `yaml:"failover,omitempty"`
}

type FailoverConfig struct {
	Enabled           bool     `yaml:"enabled,omitempty"`
	MaxRetries        int      `yaml:"max_retries,omitempty"`
	UnhealthyCooldown Duration `yaml:"unhealthy_cooldown,omitempty"`
	RequestTimeout    Duration `yaml:"request_timeout,omitempty"`
}

// Load reads and parses the config file at path.
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}
	cfg := &Config{}
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}
	cfg.applyDefaults()
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	return cfg, nil
}

// Save writes the config back to path as YAML with 0600 permissions.
func Save(path string, cfg *Config) error {
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		return fmt.Errorf("write config: %w", err)
	}
	return nil
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
		// No OAuth fields required.
	default:
		return fmt.Errorf("provider %q has unsupported auth method %q", name, method)
	}

	if strings.TrimSpace(p.API.BaseURL) == "" || strings.EqualFold(p.API.BaseURL, tbd) {
		return fmt.Errorf("provider %q api.base_url is unresolved (TBD)", name)
	}

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

	for _, a := range p.Accounts {
		if strings.TrimSpace(a.Name) == "" {
			return fmt.Errorf("provider %q has an account with empty name", name)
		}
		if method == "oauth" {
			if strings.TrimSpace(a.TokenFile) == "" {
				return fmt.Errorf("provider %q account %q has no token file", name, a.Name)
			}
		} else if strings.TrimSpace(a.APIKey) == "" {
			return fmt.Errorf("provider %q account %q has no api_key", name, a.Name)
		}
	}
	return nil
}

package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestValidateGeminiAPIKey(t *testing.T) {
	cfg := &Config{
		Server: ServerConfig{APIKeys: []string{"sk-1"}},
		Providers: map[string]ProviderConfig{
			"gemini": {
				Enabled:  true,
				Auth:     AuthConfig{Method: "api_key"},
				API:      APIConfig{BaseURL: "https://generativelanguage.googleapis.com"},
				Models:   []ModelConfig{{ID: "gemini-2.5-flash", Upstream: "gemini-2.5-flash"}},
				Accounts: []AccountConfig{{Name: "main", APIKey: "AIza-test"}},
			},
		},
	}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("expected valid config, got %v", err)
	}
}

func TestValidateGeminiMissingAPIKey(t *testing.T) {
	cfg := &Config{
		Server: ServerConfig{APIKeys: []string{"sk-1"}},
		Providers: map[string]ProviderConfig{
			"gemini": {
				Enabled:  true,
				Auth:     AuthConfig{Method: "api_key"},
				API:      APIConfig{BaseURL: "https://generativelanguage.googleapis.com"},
				Models:   []ModelConfig{{ID: "gemini-2.5-flash", Upstream: "gemini-2.5-flash"}},
				Accounts: []AccountConfig{{Name: "main"}},
			},
		},
	}
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected error for missing api_key")
	}
}

func TestValidateGeminiUnsupportedMethod(t *testing.T) {
	cfg := &Config{
		Server: ServerConfig{APIKeys: []string{"sk-1"}},
		Providers: map[string]ProviderConfig{
			"gemini": {
				Enabled:  true,
				Auth:     AuthConfig{Method: "bogus"},
				API:      APIConfig{BaseURL: "https://generativelanguage.googleapis.com"},
				Models:   []ModelConfig{{ID: "gemini-2.5-flash", Upstream: "gemini-2.5-flash"}},
				Accounts: []AccountConfig{{Name: "main", APIKey: "AIza-test"}},
			},
		},
	}
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected error for unsupported auth method")
	}
}

func TestValidateOAuthRequiresTokenFile(t *testing.T) {
	cfg := &Config{
		Server: ServerConfig{APIKeys: []string{"sk-1"}},
		Providers: map[string]ProviderConfig{
			"openai_codex": {
				Enabled: true,
				Auth: AuthConfig{
					Method:           "oauth",
					ClientID:         "client",
					AuthorizationURL: "https://example.com/authorize",
					TokenURL:         "https://example.com/token",
					RedirectPort:     1455,
				},
				API:      APIConfig{BaseURL: "https://api.example.com"},
				Models:   []ModelConfig{{ID: "gpt-5-codex", Upstream: "gpt-5-codex"}},
				Accounts: []AccountConfig{{Name: "main"}},
			},
		},
	}
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected error for oauth account missing token file")
	}
}

func TestValidateDuplicateModelAlias(t *testing.T) {
	cfg := &Config{
		Server: ServerConfig{APIKeys: []string{"sk-1"}},
		Providers: map[string]ProviderConfig{
			"gemini": {
				Enabled:  true,
				Auth:     AuthConfig{Method: "api_key"},
				API:      APIConfig{BaseURL: "https://generativelanguage.googleapis.com"},
				Models:   []ModelConfig{{ID: "shared", Upstream: "m1"}},
				Accounts: []AccountConfig{{Name: "main", APIKey: "AIza-test"}},
			},
			"openai_codex": {
				Enabled: true,
				Auth: AuthConfig{
					Method:           "oauth",
					ClientID:         "client",
					AuthorizationURL: "https://example.com/authorize",
					TokenURL:         "https://example.com/token",
					RedirectPort:     1455,
				},
				API:      APIConfig{BaseURL: "https://api.example.com"},
				Models:   []ModelConfig{{ID: "shared", Upstream: "m2"}},
				Accounts: []AccountConfig{{Name: "main", TokenFile: "/tmp/t.json"}},
			},
		},
	}
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected error for duplicate model alias")
	}
}

func TestValidateCommandCodeDefaults(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	yaml := `server:
  api_keys: [sk-1]
providers:
  command_code:
    enabled: true
    accounts:
      - name: main
        token_file: /tmp/cc.json
`
	if err := os.WriteFile(path, []byte(yaml), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("minimal config should pass with defaults: %v", err)
	}
	pc := cfg.Providers["command_code"]
	if pc.Auth.Method != "browser_key" {
		t.Fatalf("expected method browser_key, got %q", pc.Auth.Method)
	}
	if pc.Auth.AuthorizationURL != "https://commandcode.ai/studio/auth/cli" {
		t.Fatalf("unexpected authorization_url %q", pc.Auth.AuthorizationURL)
	}
	if pc.Auth.RedirectPort != 1458 {
		t.Fatalf("unexpected redirect_port %d", pc.Auth.RedirectPort)
	}
	if pc.API.BaseURL != "https://api.commandcode.ai" {
		t.Fatalf("unexpected base_url %q", pc.API.BaseURL)
	}
}

func TestValidateCommandCodeMissingAuthorizationURL(t *testing.T) {
	cfg := &Config{
		Server: ServerConfig{APIKeys: []string{"sk-1"}},
		Providers: map[string]ProviderConfig{
			"command_code": {
				Enabled:  true,
				Auth:     AuthConfig{Method: "browser_key", RedirectPort: 1458},
				API:      APIConfig{BaseURL: "https://api.commandcode.ai"},
				Accounts: []AccountConfig{{Name: "main", TokenFile: "/tmp/cc.json"}},
			},
		},
	}
	err := cfg.Validate()
	if err == nil || !strings.Contains(err.Error(), "command_code") {
		t.Fatalf("expected error naming provider, got %v", err)
	}
}

func TestValidateCommandCodeMissingRedirectPort(t *testing.T) {
	cfg := &Config{
		Server: ServerConfig{APIKeys: []string{"sk-1"}},
		Providers: map[string]ProviderConfig{
			"command_code": {
				Enabled:  true,
				Auth:     AuthConfig{Method: "browser_key", AuthorizationURL: "https://commandcode.ai/studio/auth/cli"},
				API:      APIConfig{BaseURL: "https://api.commandcode.ai"},
				Accounts: []AccountConfig{{Name: "main", TokenFile: "/tmp/cc.json"}},
			},
		},
	}
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected error for missing redirect_port")
	}
}

func TestValidateCommandCodeMissingTokenFile(t *testing.T) {
	cfg := &Config{
		Server: ServerConfig{APIKeys: []string{"sk-1"}},
		Providers: map[string]ProviderConfig{
			"command_code": {
				Enabled:  true,
				Auth:     AuthConfig{Method: "browser_key", AuthorizationURL: "https://commandcode.ai/studio/auth/cli", RedirectPort: 1458},
				API:      APIConfig{BaseURL: "https://api.commandcode.ai"},
				Accounts: []AccountConfig{{Name: "main"}},
			},
		},
	}
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected error for missing token file")
	}
}

func TestSaveRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")

	cfg := &Config{
		Server: ServerConfig{
			Host:             "127.0.0.1",
			Port:             8787,
			APIKeys:          []string{"sk-1"},
			DefaultMaxTokens: 4096,
		},
		Providers: map[string]ProviderConfig{
			"gemini": {
				Enabled:  true,
				Auth:     AuthConfig{Method: "api_key"},
				API:      APIConfig{BaseURL: "https://generativelanguage.googleapis.com"},
				Models:   []ModelConfig{{ID: "gemini-2.5-flash", Upstream: "gemini-2.5-flash"}},
				Accounts: []AccountConfig{{Name: "main", APIKey: "AIza-test"}},
			},
		},
		Routing: RoutingConfig{
			Strategy: "round-robin",
			Failover: FailoverConfig{
				Enabled:           true,
				MaxRetries:        3,
				UnhealthyCooldown: Duration(5 * time.Minute),
				RequestTimeout:    Duration(2 * time.Minute),
			},
		},
	}

	if err := Save(path, cfg); err != nil {
		t.Fatalf("save: %v", err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("expected 0600 permissions, got %o", info.Mode().Perm())
	}

	reloaded, err := Load(path)
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	if reloaded.Server.Port != 8787 {
		t.Fatalf("expected port 8787, got %d", reloaded.Server.Port)
	}
	if got := reloaded.Providers["gemini"].Accounts[0].APIKey; got != "AIza-test" {
		t.Fatalf("expected api key preserved, got %q", got)
	}
	if got := reloaded.Routing.Failover.UnhealthyCooldown.Duration(); got != 5*time.Minute {
		t.Fatalf("expected unhealthy_cooldown 5m, got %s", got)
	}
}

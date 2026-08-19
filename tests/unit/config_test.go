package unit

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/tuanp-github/unified-ai-proxy/internal/config"
)

func validGeminiConfig() *config.Config {
	return &config.Config{Server: config.ServerConfig{APIKeys: []string{"sk-test"}}, Providers: map[string]config.ProviderConfig{"gemini": {Enabled: true, Auth: config.AuthConfig{Method: "api_key"}, API: config.APIConfig{BaseURL: "https://example.test"}, Models: []config.ModelConfig{{ID: "model", Upstream: "upstream"}}, Accounts: []config.AccountConfig{{Name: "main", APIKey: "AIza-test"}}}}}
}

func TestConfigValidationMatrix(t *testing.T) {
	cases := []struct {
		name   string
		mutate func(*config.Config)
		want   string
	}{
		{"valid api key", func(c *config.Config) {}, ""},
		{"missing api key", func(c *config.Config) { c.Providers["gemini"].Accounts[0].APIKey = "" }, "api_key"},
		{"unsupported method", func(c *config.Config) { p := c.Providers["gemini"]; p.Auth.Method = "bogus"; c.Providers["gemini"] = p }, "unsupported auth method"},
		{"missing account", func(c *config.Config) { p := c.Providers["gemini"]; p.Accounts = nil; c.Providers["gemini"] = p }, "no accounts"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c := validGeminiConfig()
			tc.mutate(c)
			err := c.Validate()
			if tc.want == "" && err != nil {
				t.Fatal(err)
			}
			if tc.want != "" && (err == nil || !strings.Contains(err.Error(), tc.want)) {
				t.Fatalf("expected %q, got %v", tc.want, err)
			}
		})
	}
}

func TestConfigSaveLoadPermissionAndRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	c := validGeminiConfig()
	c.Routing.Failover.UnhealthyCooldown = config.Duration(5 * time.Minute)
	if err := config.Save(path, c); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("permission=%o", info.Mode().Perm())
	}
	reloaded, err := config.Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if reloaded.Providers["gemini"].Accounts[0].APIKey != "AIza-test" {
		t.Fatal("api key not preserved")
	}
}

package tui

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/tuanp-github/unified-ai-proxy/internal/config"
)

const testConfig = `server:
  host: "127.0.0.1"
  port: 18080
  api_keys:
    - "sk-local-key-1"
  default_max_tokens: 4096

providers:
  gemini:
    enabled: true
    auth:
      method: api_key
    api:
      base_url: "https://generativelanguage.googleapis.com"
    models:
      - id: "gemini-2.5-flash"
        upstream: "gemini-2.5-flash"
    accounts:
      - name: "gemini-main"
        api_key: "AIza-old"

routing:
  strategy: "round-robin"
  failover:
    enabled: true
    max_retries: 3
    unhealthy_cooldown: "5m"
    request_timeout: "2m"
`

func writeTestConfig(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte(testConfig), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestSetGeminiAPIKey(t *testing.T) {
	path := writeTestConfig(t)
	r := NewRuntime(path)
	if err := r.SetGeminiAPIKey("gemini-main", "AIza-new"); err != nil {
		t.Fatalf("SetGeminiAPIKey: %v", err)
	}
	cfg, err := config.Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got := cfg.Providers["gemini"].Accounts[0].APIKey; got != "AIza-new" {
		t.Fatalf("expected key updated, got %q", got)
	}
}

func TestSetGeminiAPIKeyMissingAccount(t *testing.T) {
	path := writeTestConfig(t)
	r := NewRuntime(path)
	if err := r.SetGeminiAPIKey("nope", "AIza-x"); err == nil {
		t.Fatal("expected error for missing account")
	}
}

func TestAddGeminiAccount(t *testing.T) {
	path := writeTestConfig(t)
	r := NewRuntime(path)
	if err := r.AddGeminiAccount("gemini-backup", "AIza-backup"); err != nil {
		t.Fatalf("AddGeminiAccount: %v", err)
	}
	cfg, err := config.Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	accounts := cfg.Providers["gemini"].Accounts
	if len(accounts) != 2 {
		t.Fatalf("expected 2 accounts, got %d", len(accounts))
	}
	if accounts[1].Name != "gemini-backup" || accounts[1].APIKey != "AIza-backup" {
		t.Fatalf("unexpected account: %+v", accounts[1])
	}
}

func TestAddGeminiAccountDuplicate(t *testing.T) {
	path := writeTestConfig(t)
	r := NewRuntime(path)
	if err := r.AddGeminiAccount("gemini-main", "AIza-dup"); err == nil {
		t.Fatal("expected error for duplicate account")
	}
}

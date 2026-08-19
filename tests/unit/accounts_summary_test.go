package unit

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	accountsapi "github.com/tuanp-github/unified-ai-proxy/internal/accounts"
	"github.com/tuanp-github/unified-ai-proxy/internal/config"
	"github.com/tuanp-github/unified-ai-proxy/internal/model"
	"github.com/tuanp-github/unified-ai-proxy/internal/tokenstore"
)

type Manager = accountsapi.Manager

var newAccountsManager = accountsapi.New
var summarizeAccounts = accountsapi.Summarize

func newManager(provider string, accs []config.AccountConfig) *Manager {
	m := newAccountsManager(time.Minute)
	m.Register(provider, accs)
	return m
}

func TestSummarizeReauthWhenTokenMissing(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nope.json")
	m := newManager("command_code", []config.AccountConfig{{Name: "main", TokenFile: path}})
	sums := summarizeAccounts(m)
	if len(sums) != 1 {
		t.Fatalf("expected 1 summary, got %d", len(sums))
	}
	if sums[0].Status != "reauth_required" {
		t.Fatalf("expected reauth_required, got %q", sums[0].Status)
	}
	if sums[0].Expiry != "missing" {
		t.Fatalf("expected missing expiry, got %q", sums[0].Expiry)
	}
}

func TestSummarizeReauthWhenTokenUnparseable(t *testing.T) {
	path := filepath.Join(t.TempDir(), "bad.json")
	if err := os.WriteFile(path, []byte("{not json"), 0o600); err != nil {
		t.Fatal(err)
	}
	m := newManager("command_code", []config.AccountConfig{{Name: "main", TokenFile: path}})
	sums := summarizeAccounts(m)
	if sums[0].Status != "reauth_required" {
		t.Fatalf("expected reauth_required, got %q", sums[0].Status)
	}
}

func TestSummarizeReauthWhenAccessTokenEmpty(t *testing.T) {
	path := filepath.Join(t.TempDir(), "empty.json")
	if err := tokenstore.Save(path, &model.TokenSet{}); err != nil {
		t.Fatal(err)
	}
	m := newManager("command_code", []config.AccountConfig{{Name: "main", TokenFile: path}})
	sums := summarizeAccounts(m)
	if sums[0].Status != "reauth_required" {
		t.Fatalf("expected reauth_required, got %q", sums[0].Status)
	}
}

func TestSummarizeValidTokenNeverExpires(t *testing.T) {
	path := filepath.Join(t.TempDir(), "ok.json")
	if err := tokenstore.Save(path, &model.TokenSet{AccessToken: "user_x", TokenType: "Bearer"}); err != nil {
		t.Fatal(err)
	}
	m := newManager("command_code", []config.AccountConfig{{Name: "main", TokenFile: path}})
	sums := summarizeAccounts(m)
	if sums[0].Status != "ok" {
		t.Fatalf("expected ok, got %q", sums[0].Status)
	}
	if sums[0].Expiry != "never" {
		t.Fatalf("expected never expiry, got %q", sums[0].Expiry)
	}
}

func TestSummarizeValidTokenWithExpiry(t *testing.T) {
	path := filepath.Join(t.TempDir(), "ok.json")
	exp := time.Now().Add(24 * time.Hour).UTC().Truncate(time.Second)
	if err := tokenstore.Save(path, &model.TokenSet{AccessToken: "user_x", ExpiresAt: exp}); err != nil {
		t.Fatal(err)
	}
	m := newManager("command_code", []config.AccountConfig{{Name: "main", TokenFile: path}})
	sums := summarizeAccounts(m)
	if sums[0].Status != "ok" {
		t.Fatalf("expected ok, got %q", sums[0].Status)
	}
	if sums[0].Expiry != exp.Format(time.RFC3339) {
		t.Fatalf("unexpected expiry %q", sums[0].Expiry)
	}
}

func TestSummarizeAPIKeyAccountUnaffected(t *testing.T) {
	m := newManager("gemini", []config.AccountConfig{{Name: "main", APIKey: "AIza-test"}})
	sums := summarizeAccounts(m)
	if sums[0].Status != "ok" {
		t.Fatalf("expected ok, got %q", sums[0].Status)
	}
	if !sums[0].HasAPIKey {
		t.Fatal("expected HasAPIKey true")
	}
	if sums[0].Expiry != "n/a" {
		t.Fatalf("expected n/a expiry, got %q", sums[0].Expiry)
	}
}

func TestSummarizeRuntimeReauthWins(t *testing.T) {
	path := filepath.Join(t.TempDir(), "ok.json")
	if err := tokenstore.Save(path, &model.TokenSet{AccessToken: "user_x"}); err != nil {
		t.Fatal(err)
	}
	m := newManager("command_code", []config.AccountConfig{{Name: "main", TokenFile: path}})
	m.MarkReauth("command_code", "main")
	sums := summarizeAccounts(m)
	if sums[0].Status != "reauth_required" {
		t.Fatalf("expected reauth_required from runtime flag, got %q", sums[0].Status)
	}
}

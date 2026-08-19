package unit

import (
	"testing"
	"time"

	"github.com/tuanp-github/unified-ai-proxy/internal/accounts"
	"github.com/tuanp-github/unified-ai-proxy/internal/config"
)

func TestAccountSelectionStates(t *testing.T) {
	m := accounts.New(time.Minute)
	m.Register("test", []config.AccountConfig{{Name: "main"}})
	if _, err := m.Next("missing"); err != accounts.ErrNoAccounts {
		t.Fatalf("expected no accounts, got %v", err)
	}
	m.MarkReauth("test", "main")
	if _, err := m.Next("test"); err != accounts.ErrAllReauth {
		t.Fatalf("expected reauth, got %v", err)
	}
}

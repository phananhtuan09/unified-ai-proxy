package accounts

import (
	"time"

	"github.com/tuanp-github/unified-ai-proxy/internal/model"
	"github.com/tuanp-github/unified-ai-proxy/internal/tokenstore"
)

// Summary is a display-ready view of a single account's runtime state.
type Summary struct {
	Provider  string
	Account   string
	Status    string
	Expiry    string
	HasAPIKey bool
	TokenFile string
}

// Summarize builds display-ready account summaries from the manager.
func Summarize(m *Manager) []Summary {
	var out []Summary
	for _, s := range m.Snapshot() {
		sum := Summary{
			Provider: s.Provider,
			Account:  s.Account,
			Status:   statusText(s),
		}
		if acc, ok := m.Lookup(s.Provider, s.Account); ok {
			sum.HasAPIKey = acc.APIKey != ""
			sum.TokenFile = acc.TokenFile
			sum.Expiry = expiryText(acc)
		} else {
			sum.Expiry = "missing"
		}
		out = append(out, sum)
	}
	return out
}

func statusText(s Status) string {
	switch {
	case s.Disabled:
		return "disabled"
	case s.ReauthRequired:
		return "reauth_required"
	case !s.UnhealthyUntil.IsZero():
		return "unhealthy until " + s.UnhealthyUntil.Format(time.RFC3339)
	default:
		return "ok"
	}
}

func expiryText(acc model.Account) string {
	if acc.APIKey != "" {
		return "n/a"
	}
	ts, err := tokenstore.Load(acc.TokenFile)
	if err != nil || ts == nil {
		return "missing"
	}
	if ts.ExpiresAt.IsZero() {
		return "never"
	}
	return ts.ExpiresAt.Format(time.RFC3339)
}

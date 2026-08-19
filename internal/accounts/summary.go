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
			expiry, needsReauth := tokenFileStatus(acc)
			sum.Expiry = expiry
			// A token-file account with a missing, unparseable, or empty token
			// is not logged in, regardless of the runtime reauth flag.
			if acc.APIKey == "" && needsReauth {
				sum.Status = "reauth_required"
				sum.Expiry = "missing"
			}
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

// tokenFileStatus returns the display expiry and whether the account requires
// (re)authentication based on its token file.
func tokenFileStatus(acc model.Account) (expiry string, needsReauth bool) {
	if acc.APIKey != "" {
		return "n/a", false
	}
	ts, err := tokenstore.Load(acc.TokenFile)
	if err != nil || ts == nil {
		return "missing", true
	}
	if ts.AccessToken == "" {
		return "missing", true
	}
	if ts.ExpiresAt.IsZero() {
		return "never", false
	}
	return ts.ExpiresAt.Format(time.RFC3339), false
}

package accounts

import (
	"errors"
	"sync"
	"time"

	"github.com/tuanp-github/unified-ai-proxy/internal/config"
	"github.com/tuanp-github/unified-ai-proxy/internal/model"
)

// Sentinel selection failures used by the proxy to map to error codes.
var (
	ErrNoAccounts      = errors.New("no accounts configured for provider")
	ErrAllReauth       = errors.New("all accounts require reauthentication")
	ErrAllUnhealthy    = errors.New("all accounts are unhealthy")
	ErrAccountNotFound = errors.New("account not registered")
)

// Status summarizes the runtime state of a single account.
type Status struct {
	Provider       string    `json:"provider"`
	Account        string    `json:"account"`
	Disabled       bool      `json:"disabled"`
	ReauthRequired bool      `json:"reauth_required"`
	UnhealthyUntil time.Time `json:"unhealthy_until,omitempty"`
}

// State is the mutable health state of one account.
type State struct {
	Disabled       bool
	ReauthRequired bool
	UnhealthyUntil time.Time
}

// Manager tracks per-provider accounts and performs round-robin selection.
type Manager struct {
	mu       sync.Mutex
	accounts map[string][]model.Account // provider -> accounts
	state    map[string]*State          // "provider/account" -> state
	rr       map[string]int             // provider -> next index
	cooldown time.Duration
}

// New creates an account manager with the given unhealthy cooldown.
func New(cooldown time.Duration) *Manager {
	return &Manager{
		accounts: make(map[string][]model.Account),
		state:    make(map[string]*State),
		rr:       make(map[string]int),
		cooldown: cooldown,
	}
}

func key(provider, account string) string { return provider + "/" + account }

// Register adds the configured accounts for a provider.
func (m *Manager) Register(provider string, cfgs []config.AccountConfig) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, c := range cfgs {
		acc := model.Account{
			Provider:  provider,
			Name:      c.Name,
			TokenFile: config.ExpandPath(c.TokenFile),
			APIKey:    c.APIKey,
		}
		m.accounts[provider] = append(m.accounts[provider], acc)
		if _, ok := m.state[key(provider, c.Name)]; !ok {
			m.state[key(provider, c.Name)] = &State{}
		}
	}
}

// SetDisabled marks an account disabled/enabled.
func (m *Manager) SetDisabled(provider, account string, disabled bool) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	st, ok := m.state[key(provider, account)]
	if !ok {
		return ErrAccountNotFound
	}
	st.Disabled = disabled
	return nil
}

// MarkReauth flags an account as requiring browser login again.
func (m *Manager) MarkReauth(provider, account string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if st, ok := m.state[key(provider, account)]; ok {
		st.ReauthRequired = true
	}
}

// ClearReauth clears the reauthentication flag (e.g. after `auth`).
func (m *Manager) ClearReauth(provider, account string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if st, ok := m.state[key(provider, account)]; ok {
		st.ReauthRequired = false
	}
}

// MarkUnhealthy marks an account unhealthy until the cooldown expires.
func (m *Manager) MarkUnhealthy(provider, account string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if st, ok := m.state[key(provider, account)]; ok {
		st.UnhealthyUntil = time.Now().Add(m.cooldown)
	}
}

// ClearUnhealthy immediately clears the unhealthy flag.
func (m *Manager) ClearUnhealthy(provider, account string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if st, ok := m.state[key(provider, account)]; ok {
		st.UnhealthyUntil = time.Time{}
	}
}

// Next returns the next healthy account for a provider using round-robin.
func (m *Manager) Next(provider string) (model.Account, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	list := m.accounts[provider]
	if len(list) == 0 {
		return model.Account{}, ErrNoAccounts
	}

	now := time.Now()
	start := m.rr[provider] % len(list)
	seenReauth := 0
	seenUnhealthy := 0
	seenDisabled := 0

	for i := 0; i < len(list); i++ {
		idx := (start + i) % len(list)
		acc := list[idx]
		st := m.state[key(provider, acc.Name)]
		if st == nil {
			st = &State{}
		}
		switch {
		case st.Disabled:
			seenDisabled++
		case st.ReauthRequired:
			seenReauth++
		case st.UnhealthyUntil.After(now):
			seenUnhealthy++
		default:
			m.rr[provider] = (idx + 1) % len(list)
			return acc, nil
		}
	}

	m.rr[provider] = (start + 1) % len(list)
	if seenDisabled == len(list) {
		return model.Account{}, ErrAllUnhealthy
	}
	if seenReauth > 0 && seenReauth+seenDisabled == len(list) {
		return model.Account{}, ErrAllReauth
	}
	return model.Account{}, ErrAllUnhealthy
}

// Snapshot returns the current status of every registered account.
func (m *Manager) Snapshot() []Status {
	m.mu.Lock()
	defer m.mu.Unlock()
	var out []Status
	for provider, list := range m.accounts {
		for _, acc := range list {
			st := m.state[key(provider, acc.Name)]
			if st == nil {
				st = &State{}
			}
			out = append(out, Status{
				Provider:       provider,
				Account:        acc.Name,
				Disabled:       st.Disabled,
				ReauthRequired: st.ReauthRequired,
				UnhealthyUntil: st.UnhealthyUntil,
			})
		}
	}
	return out
}

// Lookup returns a registered account by name.
func (m *Manager) Lookup(provider, account string) (model.Account, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, acc := range m.accounts[provider] {
		if acc.Name == account {
			return acc, true
		}
	}
	return model.Account{}, false
}

package tui

import (
	"context"
	"fmt"
	"sync"

	"github.com/tuanp-github/unified-ai-proxy/internal/accounts"
	"github.com/tuanp-github/unified-ai-proxy/internal/app"
	"github.com/tuanp-github/unified-ai-proxy/internal/config"
	"github.com/tuanp-github/unified-ai-proxy/internal/logs"
	"github.com/tuanp-github/unified-ai-proxy/internal/model"
	"github.com/tuanp-github/unified-ai-proxy/internal/provider"
	"github.com/tuanp-github/unified-ai-proxy/internal/proxy"
	"github.com/tuanp-github/unified-ai-proxy/internal/server"
)

const maxLogEntries = 500

// Runtime owns the proxy lifecycle and exposes read-only state to the UI.
type Runtime struct {
	configPath string

	mu      sync.Mutex
	cfg     *config.Config
	mgr     *accounts.Manager
	svc     *proxy.Service
	srv     *server.Server
	running bool
	cancel  context.CancelFunc
	errCh   chan error

	logger *logs.Logger
}

// NewRuntime creates a runtime bound to a config path.
func NewRuntime(configPath string) *Runtime {
	cfg := logs.DefaultLoggerConfig()
	logger, err := logs.NewLogger(cfg)
	if err != nil {
		panic(fmt.Sprintf("failed to initialize logger: %v", err))
	}
	return &Runtime{
		configPath: configPath,
		logger:     logger,
	}
}

// Load reads and validates the config and rebuilds the proxy services.
// It must not be called while the proxy is running.
func (r *Runtime) Load() error {
	runtime, err := app.Build(r.configPath, r.logger)
	if err != nil {
		return err
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	r.cfg = runtime.Config
	r.mgr = runtime.Accounts
	r.svc = runtime.Proxy
	r.srv = runtime.Server
	return nil
}

// Start launches the HTTP server in the background.
func (r *Runtime) Start() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.running {
		return nil
	}
	if r.srv == nil {
		return fmt.Errorf("config not loaded")
	}
	ctx, cancel := context.WithCancel(context.Background())
	errCh := make(chan error, 1)
	go func() {
		errCh <- r.srv.Run(ctx)
	}()
	r.cancel = cancel
	r.errCh = errCh
	r.running = true
	r.logger.Slog().Info("proxy started")
	return nil
}

// Stop shuts the HTTP server down and waits for it to finish.
func (r *Runtime) Stop() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if !r.running {
		return nil
	}
	r.cancel()
	err := <-r.errCh
	r.running = false
	r.cancel = nil
	r.errCh = nil
	r.logger.Slog().Info("proxy stopped")
	return err
}

// Auth runs the browser OAuth login for an OAuth-backed account.
func (r *Runtime) Auth(providerName, accountName string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.cfg == nil {
		return fmt.Errorf("config not loaded")
	}
	pc, ok := r.cfg.Providers[providerName]
	if !ok {
		return fmt.Errorf("provider %q not found", providerName)
	}
	var target config.AccountConfig
	found := false
	for _, a := range pc.Accounts {
		if a.Name == accountName {
			target = a
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("account %q not found", accountName)
	}
	if target.APIKey != "" {
		return fmt.Errorf("account %q uses an API key; no browser login required", accountName)
	}
	_, err := provider.RunOAuthLogin(context.Background(), providerName, accountName, config.ExpandPath(target.TokenFile), pc)
	if err == nil {
		r.mgr.ClearReauth(providerName, accountName)
	}
	return err
}

// Config returns the loaded config, or nil when not loaded.
func (r *Runtime) Config() *config.Config {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.cfg
}

// Running reports whether the proxy server is currently running.
func (r *Runtime) Running() bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.running
}

// Models returns the configured model aliases from enabled providers.
func (r *Runtime) Models() []model.Model {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.svc == nil {
		return nil
	}
	return r.svc.Models()
}

// AccountSummaries returns display-ready account state.
func (r *Runtime) AccountSummaries() []accounts.Summary {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.mgr == nil {
		return nil
	}
	return accounts.Summarize(r.mgr)
}

// LogEntries returns a snapshot of the request log.
func (r *Runtime) LogEntries() []logs.Entry {
	return r.logger.Ring().Entries()
}

// DefaultMaxTokens returns the configured default max tokens for test requests.
func (r *Runtime) DefaultMaxTokens() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.cfg == nil || r.cfg.Server.DefaultMaxTokens == 0 {
		return 4096
	}
	return r.cfg.Server.DefaultMaxTokens
}

// Chat sends a non-streaming chat request through the routing service.
func (r *Runtime) Chat(ctx context.Context, req *model.ChatRequest) (*model.ChatResponse, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.svc == nil {
		return nil, fmt.Errorf("config not loaded")
	}
	return r.svc.Chat(ctx, req)
}

// SetGeminiAPIKey updates an existing gemini account's api_key and saves it.
func (r *Runtime) SetGeminiAPIKey(accountName, key string) error {
	cfg, err := config.Load(r.configPath)
	if err != nil {
		return err
	}
	pc, ok := cfg.Providers["gemini"]
	if !ok {
		return fmt.Errorf("gemini provider is not configured; add it via the config editor (e) first")
	}
	found := false
	for i := range pc.Accounts {
		if pc.Accounts[i].Name == accountName {
			pc.Accounts[i].APIKey = key
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("gemini account %q not found", accountName)
	}
	cfg.Providers["gemini"] = pc
	return config.Save(r.configPath, cfg)
}

// AddGeminiAccount appends a new gemini account with an api_key and saves it.
func (r *Runtime) AddGeminiAccount(name, key string) error {
	cfg, err := config.Load(r.configPath)
	if err != nil {
		return err
	}
	pc, ok := cfg.Providers["gemini"]
	if !ok {
		return fmt.Errorf("gemini provider is not configured; add it via the config editor (e) first")
	}
	for _, a := range pc.Accounts {
		if a.Name == name {
			return fmt.Errorf("gemini account %q already exists", name)
		}
	}
	pc.Accounts = append(pc.Accounts, config.AccountConfig{Name: name, APIKey: key})
	cfg.Providers["gemini"] = pc
	return config.Save(r.configPath, cfg)
}

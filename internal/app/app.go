package app

import (
	"fmt"

	"github.com/tuanp-github/unified-ai-proxy/internal/accounts"
	"github.com/tuanp-github/unified-ai-proxy/internal/config"
	"github.com/tuanp-github/unified-ai-proxy/internal/logs"
	"github.com/tuanp-github/unified-ai-proxy/internal/provider"
	"github.com/tuanp-github/unified-ai-proxy/internal/proxy"
	"github.com/tuanp-github/unified-ai-proxy/internal/server"
)

// App is the shared application object graph used by entry adapters.
type App struct {
	Config   *config.Config
	Accounts *accounts.Manager
	Proxy    *proxy.Service
	Server   *server.Server
}

// Build loads configuration and constructs the complete application graph.
func Build(configPath string, logger *logs.Logger) (*App, error) {
	cfg, err := config.Load(configPath)
	if err != nil {
		return nil, err
	}
	mgr := accounts.New(cfg.Routing.Failover.UnhealthyCooldown.Duration())
	svc, err := proxy.New(cfg, mgr, provider.Build, logger.Slog())
	if err != nil {
		return nil, fmt.Errorf("build proxy: %w", err)
	}
	return &App{
		Config:   cfg,
		Accounts: mgr,
		Proxy:    svc,
		Server:   server.New(cfg, svc, logger),
	}, nil
}

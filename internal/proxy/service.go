package proxy

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/tuanp-github/unified-ai-proxy/internal/accounts"
	"github.com/tuanp-github/unified-ai-proxy/internal/apierr"
	"github.com/tuanp-github/unified-ai-proxy/internal/config"
	"github.com/tuanp-github/unified-ai-proxy/internal/model"
	"github.com/tuanp-github/unified-ai-proxy/internal/provider"
)

// resolved maps a model alias to its provider and upstream model id.
type resolved struct {
	provider provider.Provider
	upstream string
}

// Service resolves models to providers and applies account failover.
type Service struct {
	providers map[string]provider.Provider
	byModel   map[string]resolved
	accounts  *accounts.Manager
	failover  config.FailoverConfig
	logger    *slog.Logger
}

// New builds the routing service from config and registers accounts.
func New(cfg *config.Config, mgr *accounts.Manager, build func(name string, p config.ProviderConfig, timeout time.Duration) (provider.Provider, error), logger *slog.Logger) (*Service, error) {
	s := &Service{
		providers: make(map[string]provider.Provider),
		byModel:   make(map[string]resolved),
		accounts:  mgr,
		failover:  cfg.Routing.Failover,
		logger:    logger,
	}

	// Debug: log logger status
	if logger == nil {
		fmt.Println("[DEBUG] proxy.New: logger is NIL")
	} else {
		fmt.Println("[DEBUG] proxy.New: logger is SET")
	}

	timeout := cfg.Routing.Failover.RequestTimeout.Duration()
	for name, p := range cfg.EnabledProviders() {
		prov, err := build(name, p, timeout)
		if err != nil {
			return nil, err
		}
		s.providers[name] = prov
		mgr.Register(name, p.Accounts)
		for _, m := range prov.Models() {
			s.byModel[m.ID] = resolved{provider: prov, upstream: m.Upstream}
		}
	}
	return s, nil
}

// ResolveModel returns the provider and upstream id for a model alias.
func (s *Service) ResolveModel(alias string) (provider.Provider, string, error) {
	r, ok := s.byModel[alias]
	if !ok {
		return nil, "", apierr.ModelNotFound("unknown model: " + alias)
	}
	return r.provider, r.upstream, nil
}

// Models returns every configured model alias from enabled providers.
func (s *Service) Models() []model.Model {
	var out []model.Model
	for _, p := range s.providers {
		out = append(out, p.Models()...)
	}
	return out
}

// Chat routes a non-streaming request with pre-stream failover.
func (s *Service) Chat(ctx context.Context, req *model.ChatRequest) (*model.ChatResponse, error) {
	p, upstream, err := s.ResolveModel(req.Model)
	if err != nil {
		return nil, err
	}
	routed := *req
	routed.Model = upstream
	routed.Provider = p.Name()

	retries := s.maxRetries()
	var lastErr error
	for attempt := 0; attempt <= retries; attempt++ {
		acc, err := s.accounts.Next(p.Name())
		if err != nil {
			return nil, s.accountError(err)
		}
		if s.logger != nil {
			s.logger.Info("proxy.chat.attempt",
				"provider", p.Name(),
				"account", acc.Name,
				"model", req.Model,
				"upstream_model", upstream,
				"attempt", attempt+1,
				"max_retries", retries,
			)
		}
		resp, err := p.ChatCompletion(ctx, acc, &routed)
		if err == nil {
			if s.logger != nil {
				s.logger.Info("proxy.chat.success",
					"provider", p.Name(),
					"account", acc.Name,
					"model", req.Model,
					"usage_input", resp.Usage.InputTokens,
					"usage_output", resp.Usage.OutputTokens,
					"stop_reason", resp.StopReason,
				)
			}
			return resp, nil
		}
		lastErr = err
		if s.logger != nil {
			s.logger.Warn("proxy.chat.error",
				"provider", p.Name(),
				"account", acc.Name,
				"model", req.Model,
				"attempt", attempt+1,
				"error", err.Error(),
			)
		}
		if provider.IsAuthFailure(err) {
			s.accounts.MarkReauth(p.Name(), acc.Name)
			return nil, apierr.ProviderAuthFailed(err.Error())
		}
		if !provider.IsRetryable(err) {
			return nil, s.mapUpstreamError(err)
		}
		s.accounts.MarkUnhealthy(p.Name(), acc.Name)
	}
	return nil, s.mapExhausted(lastErr)
}

// Stream routes a streaming request. Failover happens only before any
// downstream bytes are produced.
func (s *Service) Stream(ctx context.Context, req *model.ChatRequest) (<-chan model.StreamEvent, error) {
	p, upstream, err := s.ResolveModel(req.Model)
	if err != nil {
		return nil, err
	}
	routed := *req
	routed.Model = upstream
	routed.Provider = p.Name()
	routed.Stream = true

	retries := s.maxRetries()
	var lastErr error
	for attempt := 0; attempt <= retries; attempt++ {
		acc, err := s.accounts.Next(p.Name())
		if err != nil {
			return nil, s.accountError(err)
		}
		if s.logger != nil {
			s.logger.Info("proxy.stream.attempt",
				"provider", p.Name(),
				"account", acc.Name,
				"model", req.Model,
				"upstream_model", upstream,
				"attempt", attempt+1,
				"max_retries", retries,
			)
		}
		ch, err := p.StreamChatCompletion(ctx, acc, &routed)
		if err == nil {
			if s.logger != nil {
				s.logger.Info("proxy.stream.success",
					"provider", p.Name(),
					"account", acc.Name,
					"model", req.Model,
				)
			}
			return ch, nil
		}
		lastErr = err
		if s.logger != nil {
			s.logger.Warn("proxy.stream.error",
				"provider", p.Name(),
				"account", acc.Name,
				"model", req.Model,
				"attempt", attempt+1,
				"error", err.Error(),
			)
		}
		if provider.IsAuthFailure(err) {
			s.accounts.MarkReauth(p.Name(), acc.Name)
			return nil, apierr.ProviderAuthFailed(err.Error())
		}
		if !provider.IsRetryable(err) {
			return nil, s.mapUpstreamError(err)
		}
		s.accounts.MarkUnhealthy(p.Name(), acc.Name)
	}
	return nil, s.mapExhausted(lastErr)
}

func (s *Service) maxRetries() int {
	if !s.failover.Enabled {
		return 0
	}
	return s.failover.MaxRetries
}

func (s *Service) accountError(err error) *apierr.APIError {
	switch err {
	case accounts.ErrNoAccounts:
		return apierr.ProviderUnavailable("no accounts configured for provider")
	case accounts.ErrAllReauth:
		return apierr.ReauthRequired("account requires browser login again")
	default:
		return apierr.ProviderUnavailable("no healthy provider account is available")
	}
}

func (s *Service) mapUpstreamError(err error) *apierr.APIError {
	if ue, ok := err.(*provider.UpstreamError); ok {
		if ue.PlanRestricted {
			return apierr.PlanRestricted(ue.Message)
		}
		if ue.UnsupportedModel {
			return apierr.UnsupportedModel(ue.Message)
		}
		if ue.StatusCode == 429 {
			return apierr.RateLimited(ue.Message)
		}
		if ue.StatusCode == 400 {
			return apierr.InvalidRequest(ue.Message)
		}
		if ue.Timeout {
			return apierr.UpstreamTimeout(ue.Message)
		}
	}
	return apierr.ProviderUnavailable(err.Error())
}

func (s *Service) mapExhausted(lastErr error) *apierr.APIError {
	if lastErr == nil {
		return apierr.ProviderUnavailable("no healthy provider account is available")
	}
	if provider.IsTimeout(lastErr) {
		return apierr.UpstreamTimeout(lastErr.Error())
	}
	if ue, ok := lastErr.(*provider.UpstreamError); ok && ue.StatusCode == 429 {
		return apierr.RateLimited("all available accounts are rate-limited")
	}
	return apierr.ProviderUnavailable("no healthy provider account is available")
}

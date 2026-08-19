package proxy

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/tuanp-github/unified-ai-proxy/internal/accounts"
	"github.com/tuanp-github/unified-ai-proxy/internal/apierr"
	"github.com/tuanp-github/unified-ai-proxy/internal/config"
	"github.com/tuanp-github/unified-ai-proxy/internal/model"
	"github.com/tuanp-github/unified-ai-proxy/internal/provider"
)

type captureProvider struct {
	request *model.ChatRequest
	models  []model.Model
	err     error
}

func TestChatErrorMatrix(t *testing.T) {
	cases := []struct {
		name string
		err  error
		code string
	}{
		{name: "non-retryable", err: &provider.UpstreamError{StatusCode: 400, Message: "bad request"}, code: "invalid_request"},
		{name: "rate-limit", err: &provider.UpstreamError{StatusCode: 429, Message: "busy"}, code: "rate_limited"},
		{name: "timeout", err: &provider.UpstreamError{Timeout: true, Message: "deadline"}, code: "upstream_timeout"},
		{name: "unsupported-model", err: &provider.UpstreamError{UnsupportedModel: true, Message: "unknown"}, code: "unsupported_model"},
		{name: "plan", err: &provider.UpstreamError{PlanRestricted: true, Message: "plan"}, code: "plan_restricted"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			p := &captureProvider{models: []model.Model{{ID: "alias", Upstream: "upstream"}}, err: tc.err}
			s := testService(p)
			_, got := s.Chat(context.Background(), &model.ChatRequest{Model: "alias"})
			apiGot, ok := got.(*apierr.APIError)
			if !ok || apiGot.Code != tc.code {
				t.Fatalf("expected %q, got %v", tc.code, got)
			}
		})
	}
}

func TestChatUnknownModelAndNoAccount(t *testing.T) {
	p := &captureProvider{models: []model.Model{{ID: "alias", Upstream: "upstream"}}}
	s := testService(p)
	if _, err := s.Chat(context.Background(), &model.ChatRequest{Model: "missing"}); err == nil || err.(*apierr.APIError).Code != "model_not_found" {
		t.Fatalf("expected model_not_found, got %v", err)
	}
	noAccounts := accounts.New(time.Minute)
	s.accounts = noAccounts
	if _, err := s.Chat(context.Background(), &model.ChatRequest{Model: "alias"}); err == nil || err.(*apierr.APIError).Code != "provider_unavailable" {
		t.Fatalf("expected provider_unavailable, got %v", err)
	}
}

func TestChatAuthFailureMarksReauthAndStops(t *testing.T) {
	p := &captureProvider{models: []model.Model{{ID: "alias", Upstream: "upstream"}}, err: &provider.UpstreamError{StatusCode: 401, AuthFailed: true, Message: "expired"}}
	s := testService(p)
	_, err := s.Chat(context.Background(), &model.ChatRequest{Model: "alias"})
	if err == nil || err.(*apierr.APIError).Code != "provider_auth_failed" {
		t.Fatalf("expected provider_auth_failed, got %v", err)
	}
	if _, nextErr := s.accounts.Next("test"); nextErr != accounts.ErrAllReauth {
		t.Fatalf("expected account reauth state, got %v", nextErr)
	}
}

func (p *captureProvider) Name() string          { return "test" }
func (p *captureProvider) Models() []model.Model { return p.models }
func (p *captureProvider) ChatCompletion(_ context.Context, _ model.Account, req *model.ChatRequest) (*model.ChatResponse, error) {
	routed := *req
	p.request = &routed
	return nil, p.err
}
func (p *captureProvider) StreamChatCompletion(_ context.Context, _ model.Account, req *model.ChatRequest) (<-chan model.StreamEvent, error) {
	routed := *req
	p.request = &routed
	return nil, p.err
}

func testService(p provider.Provider) *Service {
	mgr := accounts.New(time.Minute)
	mgr.Register("test", []config.AccountConfig{{Name: "account"}})
	return &Service{
		providers: map[string]provider.Provider{"test": p},
		byModel:   map[string]resolved{"alias": {provider: p, upstream: "upstream"}},
		accounts:  mgr,
		logger:    nil,
	}
}

func TestChatDoesNotMutateCallerRequest(t *testing.T) {
	p := &captureProvider{models: []model.Model{{ID: "alias", Upstream: "upstream"}}}
	s := testService(p)
	req := &model.ChatRequest{Model: "alias", Provider: "caller", Stream: true}
	before := *req
	_, _ = s.Chat(context.Background(), req)
	if !reflect.DeepEqual(*req, before) {
		t.Fatalf("Chat mutated caller request: before=%+v after=%+v", before, *req)
	}
	if p.request.Model != "upstream" || p.request.Provider != "test" || p.request.Stream != before.Stream {
		t.Fatalf("provider received incorrect routed request: %+v", p.request)
	}
}

func TestStreamDoesNotMutateCallerRequest(t *testing.T) {
	p := &captureProvider{models: []model.Model{{ID: "alias", Upstream: "upstream"}}, err: errors.New("setup failed")}
	s := testService(p)
	req := &model.ChatRequest{Model: "alias", Provider: "caller"}
	before := *req
	_, _ = s.Stream(context.Background(), req)
	if !reflect.DeepEqual(*req, before) {
		t.Fatalf("Stream mutated caller request: before=%+v after=%+v", before, *req)
	}
	if p.request.Model != "upstream" || p.request.Provider != "test" || !p.request.Stream {
		t.Fatalf("provider received incorrect routed request: %+v", p.request)
	}
}

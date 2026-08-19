package provider

import (
	"context"
	"errors"
	"fmt"
	"net"
	"time"

	"github.com/tuanp-github/unified-ai-proxy/internal/config"
	"github.com/tuanp-github/unified-ai-proxy/internal/model"
)

// Provider is the seam every upstream provider must implement.
type Provider interface {
	Name() string
	Models() []model.Model
	ChatCompletion(ctx context.Context, account model.Account, req *model.ChatRequest) (*model.ChatResponse, error)
	StreamChatCompletion(ctx context.Context, account model.Account, req *model.ChatRequest) (<-chan model.StreamEvent, error)
}

// RefreshOAuthToken keeps OAuth lifecycle separate from the chat provider contract.
func RefreshOAuthToken(ctx context.Context, name string, cfg config.ProviderConfig, account model.Account, timeout time.Duration) (*model.TokenSet, error) {
	if cfg.Auth.Method != "oauth" {
		return nil, fmt.Errorf("provider %q does not use OAuth", name)
	}
	return NewCodex(cfg, timeout).RefreshToken(ctx, account)
}

// UpstreamError is a typed error returned by provider HTTP calls.
type UpstreamError struct {
	StatusCode int
	Retryable  bool
	AuthFailed bool
	Timeout    bool
	// UnsupportedModel reports that the upstream rejected the model id as
	// unknown, independent of the HTTP status code.
	UnsupportedModel bool
	// PlanRestricted reports that the model is not available in the current plan.
	PlanRestricted bool
	Message        string
}

func (e *UpstreamError) Error() string {
	if e.StatusCode > 0 {
		return fmt.Sprintf("upstream HTTP %d: %s", e.StatusCode, e.Message)
	}
	return e.Message
}

// NewUpstreamError builds an UpstreamError from a status code.
func NewUpstreamError(status int, body string) *UpstreamError {
	e := &UpstreamError{StatusCode: status, Message: body}
	switch status {
	case 429, 500, 502, 503, 504:
		e.Retryable = true
	case 401, 403:
		e.AuthFailed = true
	}
	return e
}

// IsRetryable reports whether an error is eligible for pre-stream failover.
func IsRetryable(err error) bool {
	if ue, ok := err.(*UpstreamError); ok {
		return ue.Retryable
	}
	return false
}

// IsAuthFailure reports whether an error indicates an invalid token.
func IsAuthFailure(err error) bool {
	if ue, ok := err.(*UpstreamError); ok {
		return ue.AuthFailed
	}
	return false
}

// IsTimeout reports whether an error indicates an upstream timeout.
func IsTimeout(err error) bool {
	if ue, ok := err.(*UpstreamError); ok {
		return ue.Timeout
	}
	return false
}

// networkError wraps a transport-level error as a retryable UpstreamError.
func networkError(err error) *UpstreamError {
	e := &UpstreamError{Retryable: true, Message: err.Error()}
	if errors.Is(err, context.DeadlineExceeded) {
		e.Timeout = true
	}
	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		e.Timeout = true
	}
	return e
}

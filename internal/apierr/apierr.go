package apierr

import "net/http"

// APIError is a proxy error with a stable code and HTTP status.
type APIError struct {
	Code    string
	Message string
	Status  int
}

func (e *APIError) Error() string { return e.Message }

// New builds an APIError.
func New(code, message string, status int) *APIError {
	return &APIError{Code: code, Message: message, Status: status}
}

// Convenience constructors matching the spec's required error codes.
func Unauthorized(msg string) *APIError {
	return New("unauthorized", msg, http.StatusUnauthorized)
}

func InvalidRequest(msg string) *APIError {
	return New("invalid_request", msg, http.StatusBadRequest)
}

func UnsupportedField(msg string) *APIError {
	return New("unsupported_field", msg, http.StatusBadRequest)
}

func ModelNotFound(msg string) *APIError {
	return New("model_not_found", msg, http.StatusNotFound)
}

func ProviderAuthFailed(msg string) *APIError {
	return New("provider_auth_failed", msg, http.StatusUnauthorized)
}

func ReauthRequired(msg string) *APIError {
	return New("reauth_required", msg, http.StatusUnauthorized)
}

func RateLimited(msg string) *APIError {
	return New("rate_limited", msg, http.StatusTooManyRequests)
}

func ProviderUnavailable(msg string) *APIError {
	return New("provider_unavailable", msg, http.StatusServiceUnavailable)
}

func UpstreamTimeout(msg string) *APIError {
	return New("upstream_timeout", msg, http.StatusGatewayTimeout)
}

// ErrorType maps an APIError to a protocol-compatible error type.
func (e *APIError) ErrorType() string {
	switch e.Status {
	case http.StatusUnauthorized:
		return "authentication_error"
	case http.StatusTooManyRequests:
		return "rate_limit_error"
	case http.StatusNotFound:
		return "invalid_request_error"
	case http.StatusBadRequest:
		return "invalid_request_error"
	default:
		return "api_error"
	}
}

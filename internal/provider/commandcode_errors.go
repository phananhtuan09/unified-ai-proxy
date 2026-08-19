package provider

import (
	"encoding/json"
	"regexp"
	"strings"
)

var unknownModelRe = regexp.MustCompile(`(?i)model.{0,40}(not found|not supported|does not exist|unknown|unavailable|not in catalog|invalid model)`)

func isUnknownModelError(body []byte) bool {
	var probe struct {
		Code    string          `json:"code"`
		Message string          `json:"message"`
		Error   json.RawMessage `json:"error"`
	}
	if json.Unmarshal(body, &probe) != nil {
		return false
	}
	code, msg := probe.Code, probe.Message
	if len(probe.Error) > 0 {
		var e struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		}
		if json.Unmarshal(probe.Error, &e) == nil {
			if code == "" {
				code = e.Code
			}
			if msg == "" {
				msg = e.Message
			}
		}
	}
	if strings.EqualFold(code, "unsupported_model") || strings.EqualFold(code, "model_not_found") {
		return true
	}
	return unknownModelRe.MatchString(msg)
}

func isPlanRestricted(body []byte) bool {
	var probe struct {
		Code    string          `json:"code"`
		Error   json.RawMessage `json:"error"`
		Message string          `json:"message"`
	}
	if json.Unmarshal(body, &probe) != nil {
		return false
	}
	if strings.EqualFold(probe.Code, "FORBIDDEN") || strings.Contains(probe.Message, "MODEL_NOT_IN_PLAN") {
		return true
	}
	if len(probe.Error) > 0 {
		var e struct {
			Code string `json:"code"`
		}
		if json.Unmarshal(probe.Error, &e) == nil && strings.EqualFold(e.Code, "FORBIDDEN") {
			return true
		}
	}
	return false
}

func (c *CommandCode) upstreamError(status int, body []byte, apiKey string) error {
	return &UpstreamError{StatusCode: status, Retryable: isRetryStatus(status), AuthFailed: isAuthStatus(status), UnsupportedModel: isUnknownModelError(body), PlanRestricted: isPlanRestricted(body), Message: sanitizeMessage(extractErrorMessage(body), apiKey)}
}

func extractErrorMessage(body []byte) string {
	var probe struct {
		Message string          `json:"message"`
		Error   json.RawMessage `json:"error"`
	}
	if json.Unmarshal(body, &probe) != nil {
		return strings.TrimSpace(string(body))
	}
	if probe.Message != "" {
		return probe.Message
	}
	if len(probe.Error) > 0 {
		var em struct {
			Message string `json:"message"`
		}
		if json.Unmarshal(probe.Error, &em) == nil && em.Message != "" {
			return em.Message
		}
		var es string
		if json.Unmarshal(probe.Error, &es) == nil && es != "" {
			return es
		}
	}
	return strings.TrimSpace(string(body))
}

func sanitizeMessage(msg, apiKey string) string {
	if apiKey == "" {
		return msg
	}
	msg = strings.ReplaceAll(msg, "Bearer "+apiKey, "Bearer [REDACTED]")
	return strings.ReplaceAll(msg, apiKey, "[REDACTED]")
}

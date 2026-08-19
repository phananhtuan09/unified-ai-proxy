package server

import (
	"encoding/json"
	"strings"

	"github.com/tuanp-github/unified-ai-proxy/internal/apierr"
	"github.com/tuanp-github/unified-ai-proxy/internal/model"
)

// decodeOpenAIContent is shared by Chat Completions and Responses input
// normalization; protocol-specific streaming remains in each renderer.
func decodeOpenAIContent(raw json.RawMessage) (string, error) {
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		return s, nil
	}
	var parts []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	}
	if err := json.Unmarshal(raw, &parts); err != nil {
		return "", apierr.InvalidRequest("message content must be text")
	}
	var b strings.Builder
	for _, p := range parts {
		switch p.Type {
		case "text", "input_text", "output_text", "":
			b.WriteString(p.Text)
		default:
			return "", apierr.UnsupportedField("unsupported message content type: " + p.Type)
		}
	}
	return b.String(), nil
}

func openAIFinishReason(reason string) string {
	switch reason {
	case "max_tokens", "max_output_tokens", "length":
		return "length"
	case "content_filter":
		return "content_filter"
	default:
		return "stop"
	}
}

func usageInputTokens(u *model.Usage) int {
	if u == nil {
		return 0
	}
	return u.InputTokens
}

func usageOutputTokens(u *model.Usage) int {
	if u == nil {
		return 0
	}
	return u.OutputTokens
}

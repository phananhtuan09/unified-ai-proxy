package provider

import (
	"fmt"
	"time"

	"github.com/tuanp-github/unified-ai-proxy/internal/config"
)

// Build constructs a provider from its name and config.
func Build(name string, cfg config.ProviderConfig, timeout time.Duration) (Provider, error) {
	switch name {
	case "openai_codex":
		return NewCodex(cfg, timeout), nil
	case "gemini":
		return NewGemini(cfg, timeout), nil
	default:
		return nil, fmt.Errorf("unsupported provider %q", name)
	}
}

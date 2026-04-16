package ai

import (
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/jcornudella/hotbrew/internal/config"
)

// ProviderFromConfig returns a Provider built from the user's AI
// config, or nil when AI is disabled (empty config, unknown
// provider, or missing API key). A nil return is the explicit
// contract for "use the fallback" — callers pass it straight to
// NewSummarizer without branching.
func ProviderFromConfig(cfg *config.AIConfig) Provider {
	if cfg == nil {
		return nil
	}
	provider := strings.ToLower(strings.TrimSpace(cfg.Provider))
	if provider == "" || provider == "none" {
		return nil
	}

	apiKey := resolveAPIKey(cfg)
	if apiKey == "" {
		return nil
	}

	switch provider {
	case "anthropic":
		p := NewAnthropicProvider(apiKey)
		if cfg.Model != "" {
			p.Model = cfg.Model
		}
		if cfg.MaxTokens > 0 {
			p.MaxTokens = cfg.MaxTokens
		}
		p.Client = &http.Client{Timeout: 15 * time.Second}
		return p
	default:
		return nil
	}
}

// SummarizerFromConfig is the convenience the call site actually
// wants: "give me a Summarizer, don't make me think about whether
// AI is on." Internally it always wraps with the fallback so the
// returned Summarizer is safe to call even when config disables AI.
func SummarizerFromConfig(cfg *config.AIConfig) Summarizer {
	return NewSummarizer(ProviderFromConfig(cfg))
}

// resolveAPIKey prefers the literal key in config, falls back to
// the named env var, and finally tries the conventional
// ANTHROPIC_API_KEY / OPENAI_API_KEY style vars for known
// providers. Empty return means "no key available".
func resolveAPIKey(cfg *config.AIConfig) string {
	if cfg.APIKey != "" {
		return cfg.APIKey
	}
	if cfg.APIKeyEnv != "" {
		if v := os.Getenv(cfg.APIKeyEnv); v != "" {
			return v
		}
	}
	switch strings.ToLower(cfg.Provider) {
	case "anthropic":
		return os.Getenv("ANTHROPIC_API_KEY")
	}
	return ""
}

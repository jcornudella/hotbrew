package ai

import (
	"context"
	"testing"

	"github.com/jcornudella/hotbrew/internal/config"
	"github.com/jcornudella/hotbrew/internal/intel"
)

func TestProviderFromConfigNilReturnsNil(t *testing.T) {
	if got := ProviderFromConfig(nil); got != nil {
		t.Errorf("nil config should produce nil provider, got %T", got)
	}
}

func TestProviderFromConfigEmptyProviderReturnsNil(t *testing.T) {
	if got := ProviderFromConfig(&config.AIConfig{}); got != nil {
		t.Errorf("empty provider should produce nil, got %T", got)
	}
}

func TestProviderFromConfigNoneReturnsNil(t *testing.T) {
	if got := ProviderFromConfig(&config.AIConfig{Provider: "none"}); got != nil {
		t.Errorf("provider=none should produce nil, got %T", got)
	}
}

func TestProviderFromConfigMissingKeyReturnsNil(t *testing.T) {
	// Explicit env var not set → no key → nil provider so the
	// summarizer falls back transparently. Regression guard for the
	// "user enabled provider but forgot the key" footgun.
	t.Setenv("ANTHROPIC_API_KEY", "")
	cfg := &config.AIConfig{Provider: "anthropic"}
	if got := ProviderFromConfig(cfg); got != nil {
		t.Errorf("missing api key should produce nil, got %T", got)
	}
}

func TestProviderFromConfigUnknownProviderReturnsNil(t *testing.T) {
	cfg := &config.AIConfig{Provider: "made-up", APIKey: "x"}
	if got := ProviderFromConfig(cfg); got != nil {
		t.Errorf("unknown provider should produce nil, got %T", got)
	}
}

func TestProviderFromConfigAnthropicUsesLiteralKey(t *testing.T) {
	cfg := &config.AIConfig{
		Provider: "anthropic",
		APIKey:   "literal-key",
		Model:    "claude-foo",
	}
	p, ok := ProviderFromConfig(cfg).(*AnthropicProvider)
	if !ok {
		t.Fatalf("expected *AnthropicProvider, got %T", ProviderFromConfig(cfg))
	}
	if p.APIKey != "literal-key" {
		t.Errorf("APIKey = %q", p.APIKey)
	}
	if p.Model != "claude-foo" {
		t.Errorf("Model override lost: %q", p.Model)
	}
}

func TestProviderFromConfigAnthropicUsesEnvFallback(t *testing.T) {
	t.Setenv("ANTHROPIC_API_KEY", "env-key")
	cfg := &config.AIConfig{Provider: "anthropic"}
	p, ok := ProviderFromConfig(cfg).(*AnthropicProvider)
	if !ok {
		t.Fatalf("expected *AnthropicProvider")
	}
	if p.APIKey != "env-key" {
		t.Errorf("env fallback failed: APIKey=%q", p.APIKey)
	}
}

func TestProviderFromConfigUsesCustomEnvName(t *testing.T) {
	t.Setenv("CUSTOM_AI_KEY", "custom-key")
	cfg := &config.AIConfig{Provider: "anthropic", APIKeyEnv: "CUSTOM_AI_KEY"}
	p, ok := ProviderFromConfig(cfg).(*AnthropicProvider)
	if !ok {
		t.Fatalf("expected *AnthropicProvider")
	}
	if p.APIKey != "custom-key" {
		t.Errorf("custom env lookup failed: APIKey=%q", p.APIKey)
	}
}

// SummarizerFromConfig should always hand back a usable summarizer,
// even with a disabled/nil config — the fallback kicks in.
func TestSummarizerFromConfigDisabledStillRuns(t *testing.T) {
	s := SummarizerFromConfig(nil)
	got, err := s.SummarizeItem(context.Background(), intel.IntelItem{Title: "hello"})
	if err != nil {
		t.Fatalf("SummarizeItem: %v", err)
	}
	if got != "hello" {
		t.Errorf("fallback should return title; got %q", got)
	}
}

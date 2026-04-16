package ai

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/jcornudella/hotbrew/internal/intel"
)

func TestSummarizerUsesProviderOutput(t *testing.T) {
	provider := ProviderFunc(func(ctx context.Context, prompt string) (string, error) {
		return "  AI-written sentence.  ", nil
	})
	s := NewSummarizer(provider)

	got, err := s.SummarizeItem(context.Background(), intel.IntelItem{
		Title: "Something happened",
	})
	if err != nil {
		t.Fatalf("SummarizeItem: %v", err)
	}
	if got != "AI-written sentence." {
		t.Errorf("got %q, want trimmed AI output", got)
	}
}

func TestSummarizerFallsBackOnProviderError(t *testing.T) {
	provider := ProviderFunc(func(ctx context.Context, prompt string) (string, error) {
		return "", errors.New("network down")
	})
	s := NewSummarizer(provider)

	item := intel.IntelItem{Title: "bare title"}
	got, err := s.SummarizeItem(context.Background(), item)
	if err != nil {
		t.Fatalf("SummarizeItem returned error despite fallback: %v", err)
	}
	if got != "bare title" {
		t.Errorf("fallback output mismatch: got %q", got)
	}
}

func TestSummarizerFallsBackOnEmptyCompletion(t *testing.T) {
	provider := ProviderFunc(func(ctx context.Context, prompt string) (string, error) {
		return "   ", nil
	})
	s := NewSummarizer(provider)

	item := intel.IntelItem{Title: "fallback"}
	got, _ := s.SummarizeItem(context.Background(), item)
	if got != "fallback" {
		t.Errorf("empty completion should yield fallback; got %q", got)
	}
}

func TestSummarizerNilProviderUsesFallback(t *testing.T) {
	s := NewSummarizer(nil)
	got, _ := s.SummarizeItem(context.Background(), intel.IntelItem{Title: "t"})
	if got != "t" {
		t.Errorf("nil provider should route to fallback; got %q", got)
	}
}

func TestSummarizerClusterUsesProvider(t *testing.T) {
	var capturedPrompt string
	provider := ProviderFunc(func(ctx context.Context, prompt string) (string, error) {
		capturedPrompt = prompt
		return "Cluster blurb.", nil
	})
	s := NewSummarizer(provider)

	items := []intel.IntelItem{
		{ID: "lead", Title: "The Lead", SourceName: "HN"},
		{ID: "a", Title: "A side story", SourceName: "Lobsters"},
	}
	cluster := intel.ThemeCluster{
		Label:          "AI",
		Representative: "lead",
		ItemIDs:        []string{"lead", "a"},
	}

	got, err := s.SummarizeCluster(context.Background(), cluster, items)
	if err != nil {
		t.Fatalf("SummarizeCluster: %v", err)
	}
	if got != "Cluster blurb." {
		t.Errorf("got %q", got)
	}
	if !strings.Contains(capturedPrompt, "The Lead") {
		t.Errorf("prompt missing lead title: %q", capturedPrompt)
	}
	if !strings.Contains(capturedPrompt, "A side story") {
		t.Errorf("prompt missing supporting title: %q", capturedPrompt)
	}
}

func TestSummarizerClusterFallbackOnError(t *testing.T) {
	provider := ProviderFunc(func(ctx context.Context, prompt string) (string, error) {
		return "", errors.New("rate limited")
	})
	s := NewSummarizer(provider)

	items := []intel.IntelItem{{ID: "lead", Title: "Solo", SourceName: "HN"}}
	cluster := intel.ThemeCluster{
		Label:          "Devtools",
		Representative: "lead",
		ItemIDs:        []string{"lead"},
	}
	got, _ := s.SummarizeCluster(context.Background(), cluster, items)
	if got != "Devtools: Solo" {
		t.Errorf("fallback mismatch: got %q", got)
	}
}

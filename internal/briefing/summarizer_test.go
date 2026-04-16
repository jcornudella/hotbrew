package briefing

import (
	"context"
	"strings"
	"testing"

	"github.com/jcornudella/hotbrew/internal/intel"
)

func TestFallbackSummarizerItemPrefersSummary(t *testing.T) {
	item := intel.IntelItem{
		Title:   "headline only",
		Summary: "First sentence here. Second sentence that should not appear.",
		Body:    "longer body that should not be used when a summary exists",
	}
	got, err := FallbackSummarizer{}.SummarizeItem(context.Background(), item)
	if err != nil {
		t.Fatalf("SummarizeItem: %v", err)
	}
	if got != "First sentence here." {
		t.Errorf("got %q, want %q", got, "First sentence here.")
	}
}

func TestFallbackSummarizerItemFallsBackToBody(t *testing.T) {
	item := intel.IntelItem{
		Title: "some title",
		Body:  "The body describes the thing. And then more detail.",
	}
	got, _ := FallbackSummarizer{}.SummarizeItem(context.Background(), item)
	if got != "The body describes the thing." {
		t.Errorf("got %q", got)
	}
}

func TestFallbackSummarizerItemFallsBackToTitle(t *testing.T) {
	item := intel.IntelItem{Title: "bare title"}
	got, _ := FallbackSummarizer{}.SummarizeItem(context.Background(), item)
	if got != "bare title" {
		t.Errorf("got %q", got)
	}
}

func TestFallbackSummarizerClusterLeadAndSupporting(t *testing.T) {
	items := []intel.IntelItem{
		{ID: "lead", Title: "The Main Story", SourceName: "Hacker News"},
		{ID: "a", Title: "ignored", SourceName: "Lobsters"},
		{ID: "b", Title: "ignored", SourceName: "Reddit"},
	}
	c := intel.ThemeCluster{
		Label:          "AI",
		Representative: "lead",
		ItemIDs:        []string{"lead", "a", "b"},
	}
	got, err := FallbackSummarizer{}.SummarizeCluster(context.Background(), c, items)
	if err != nil {
		t.Fatalf("SummarizeCluster: %v", err)
	}
	want := "AI: The Main Story — also covered by Lobsters and Reddit"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestFallbackSummarizerClusterWithoutSupportingMembers(t *testing.T) {
	items := []intel.IntelItem{{ID: "lead", Title: "Solo Story"}}
	c := intel.ThemeCluster{
		Label:          "Devtools",
		Representative: "lead",
		ItemIDs:        []string{"lead"},
	}
	got, _ := FallbackSummarizer{}.SummarizeCluster(context.Background(), c, items)
	if got != "Devtools: Solo Story" {
		t.Errorf("got %q", got)
	}
}

func TestFallbackSummarizerClusterDedupsAndSortsSources(t *testing.T) {
	items := []intel.IntelItem{
		{ID: "lead", Title: "The Thing", SourceName: "Hacker News"},
		{ID: "a", Title: "x", SourceName: "Reddit"},
		{ID: "b", Title: "y", SourceName: "Lobsters"},
		{ID: "c", Title: "z", SourceName: "Reddit"}, // duplicate source
		{ID: "d", Title: "w", SourceName: "ArXiv"},
		{ID: "e", Title: "v", SourceName: "Zulip"}, // 4th distinct — should be clipped
	}
	c := intel.ThemeCluster{
		Label:          "AI",
		Representative: "lead",
		ItemIDs:        []string{"lead", "a", "b", "c", "d", "e"},
	}
	got, _ := FallbackSummarizer{}.SummarizeCluster(context.Background(), c, items)
	// Sources sorted alphabetically, capped at 3: ArXiv, Lobsters, Reddit.
	if !strings.Contains(got, "ArXiv, Lobsters and Reddit") {
		t.Errorf("expected sorted/clipped source list; got %q", got)
	}
	if strings.Contains(got, "Zulip") {
		t.Errorf("fourth source should be clipped; got %q", got)
	}
}

func TestFallbackSummarizerIsDeterministic(t *testing.T) {
	items := []intel.IntelItem{
		{ID: "lead", Title: "X", SourceName: "Hacker News"},
		{ID: "a", Title: "y", SourceName: "Lobsters"},
		{ID: "b", Title: "z", SourceName: "Reddit"},
	}
	c := intel.ThemeCluster{
		Label:          "AI",
		Representative: "lead",
		ItemIDs:        []string{"lead", "a", "b"},
	}
	s := FallbackSummarizer{}
	one, _ := s.SummarizeCluster(context.Background(), c, items)
	two, _ := s.SummarizeCluster(context.Background(), c, items)
	if one != two {
		t.Errorf("not deterministic:\n  one: %q\n  two: %q", one, two)
	}
}

// DefaultSummarizer should return something satisfying the interface.
// Keeps the constructor honest — without this test, a later refactor
// could easily break the no-provider contract without noticing.
func TestDefaultSummarizerSatisfiesInterface(t *testing.T) {
	var s Summarizer = DefaultSummarizer() //nolint:staticcheck // explicit interface assertion documents intent
	got, err := s.SummarizeItem(context.Background(), intel.IntelItem{Title: "hello"})
	if err != nil {
		t.Fatalf("SummarizeItem: %v", err)
	}
	if got != "hello" {
		t.Errorf("got %q, want %q", got, "hello")
	}
}

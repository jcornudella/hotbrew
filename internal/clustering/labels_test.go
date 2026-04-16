package clustering

import (
	"testing"

	"github.com/jcornudella/hotbrew/pkg/trss"
)

func TestLabelForItemsPicksAIWhenKeywordsDominate(t *testing.T) {
	items := []trss.Item{
		{Title: "Claude Opus 4.8 launches today", Summary: "New LLM release"},
		{Title: "Agents finally work well", Summary: "LLM frontier"},
	}
	got := LabelForItems(items)
	if got.Slug != "ai" {
		t.Fatalf("expected ai, got %q", got.Slug)
	}
	if got.Display != "AI" {
		t.Fatalf("expected AI display, got %q", got.Display)
	}
}

func TestLabelForItemsPicksReposWhenAllGitHubTrending(t *testing.T) {
	items := []trss.Item{
		{Title: "karpathy/autoresearch", Source: trss.ItemSource{Via: "github-trending"}},
		{Title: "anthropics/claude-cookbooks", Source: trss.ItemSource{Via: "github-trending"}},
	}
	got := LabelForItems(items)
	if got.Slug != "repo" {
		t.Fatalf("expected repo, got %q", got.Slug)
	}
}

func TestLabelForItemsPicksInfraKeywords(t *testing.T) {
	items := []trss.Item{
		{Title: "Supabase ships edge inference on Cloudflare", Summary: "infra moves closer"},
	}
	got := LabelForItems(items)
	if got.Slug != "infra" {
		t.Fatalf("expected infra, got %q", got.Slug)
	}
}

func TestLabelForItemsFallsBackToGeneral(t *testing.T) {
	items := []trss.Item{
		{Title: "A quiet thought on mornings", Summary: "nothing technical"},
	}
	got := LabelForItems(items)
	if got.Slug != "general" {
		t.Fatalf("expected general fallback, got %q", got.Slug)
	}
}

func TestLabelForItemsDoesNotMatchSubstringsAcrossWordBoundaries(t *testing.T) {
	// "ai" must not match inside "renaissance"; "ml" must not match
	// inside "mainline". A real AI item would stay AI; this test
	// makes sure we don't route Compiler articles to AI by accident.
	items := []trss.Item{
		{Title: "Compiler renaissance", Summary: "Why compilers will matter again on the mainline."},
	}
	got := LabelForItems(items)
	if got.Slug == "ai" {
		t.Fatalf("compiler article should not label as ai, got %q", got.Slug)
	}
	if got.Slug != "devtools" {
		t.Fatalf("compiler article should label as devtools, got %q", got.Slug)
	}
}

func TestLabelForItemsEmptyReturnsGeneral(t *testing.T) {
	got := LabelForItems(nil)
	if got.Slug != "general" {
		t.Fatalf("expected general, got %q", got.Slug)
	}
}

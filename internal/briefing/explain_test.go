package briefing

import (
	"strings"
	"testing"
	"time"

	"github.com/jcornudella/hotbrew/internal/intel"
)

func TestExplainReturnsFalseForMissingID(t *testing.T) {
	b := &intel.Briefing{
		Items: []intel.ScoredItem{
			{Item: intel.IntelItem{ID: "a"}},
		},
	}
	if _, ok := Explain(b, "missing"); ok {
		t.Fatal("expected Explain to report missing id")
	}
	if _, ok := Explain(nil, "a"); ok {
		t.Fatal("nil briefing should not yield an explanation")
	}
	if _, ok := Explain(b, ""); ok {
		t.Fatal("empty id should not yield an explanation")
	}
}

func TestExplainPopulatesThemeAndSourcesFromCluster(t *testing.T) {
	b := &intel.Briefing{
		Items: []intel.ScoredItem{
			{Item: intel.IntelItem{ID: "hn", SourceName: "Hacker News", Title: "Big launch"}, Score: 9},
			{Item: intel.IntelItem{ID: "tldr", SourceName: "TLDR", Title: "Big launch"}, Score: 5},
		},
		Clusters: []intel.ThemeCluster{
			{ID: "c1", Slug: "ai", Label: "AI", Representative: "hn", ItemIDs: []string{"hn", "tldr"}, Score: 9},
		},
	}

	exp, ok := Explain(b, "hn")
	if !ok {
		t.Fatal("expected explanation")
	}
	if exp.Theme != "AI" {
		t.Fatalf("theme mismatch: got %q want AI", exp.Theme)
	}
	if len(exp.Sources) != 2 || exp.Sources[0] != "Hacker News" || exp.Sources[1] != "TLDR" {
		t.Fatalf("sources mismatch: got %v", exp.Sources)
	}
	if !strings.Contains(exp.WhyItMatters, "AI") {
		t.Fatalf("why-it-matters should mention theme: %q", exp.WhyItMatters)
	}
	if !strings.Contains(exp.WhyItMatters, "2 sources") {
		t.Fatalf("why-it-matters should note cross-source: %q", exp.WhyItMatters)
	}
}

func TestExplainFallsBackToSingletonForUnclusteredItem(t *testing.T) {
	b := &intel.Briefing{
		Items: []intel.ScoredItem{
			{Item: intel.IntelItem{ID: "lonely", SourceName: "HN", Title: "Solo"}, Score: 3},
		},
	}
	exp, ok := Explain(b, "lonely")
	if !ok {
		t.Fatal("expected explanation")
	}
	if len(exp.Sources) != 1 || exp.Sources[0] != "HN" {
		t.Fatalf("singleton sources: got %v", exp.Sources)
	}
	if exp.WhyItMatters == "" {
		t.Fatal("why-it-matters should always be non-empty")
	}
}

func TestExplainRanksFactorsByDistanceFromNeutral(t *testing.T) {
	b := &intel.Briefing{
		Items: []intel.ScoredItem{
			{
				Item: intel.IntelItem{ID: "x", Title: "x"},
				Breakdown: intel.ScoreBreakdown{
					Freshness:     1.8,
					Authority:     1.0,
					Engagement:    1.05,
					Resonance:     1.6,
					TopicMatch:    1.0,
					RepeatPenalty: 1.0,
				},
			},
		},
	}
	exp, _ := Explain(b, "x")
	if len(exp.Factors) < 2 {
		t.Fatalf("expected at least 2 non-neutral factors, got %d", len(exp.Factors))
	}
	// Freshness (1.8, distance 0.8) should outrank resonance (1.6, 0.6).
	if exp.Factors[0].Name != "freshness" {
		t.Fatalf("biggest mover should be freshness, got %q", exp.Factors[0].Name)
	}
	if exp.Factors[1].Name != "resonance" {
		t.Fatalf("second should be resonance, got %q", exp.Factors[1].Name)
	}
	// Neutral (1.0) and near-neutral (~1.05) are filtered out.
	for _, f := range exp.Factors {
		if f.Name == "authority" || f.Name == "topic match" || f.Name == "repeat penalty" {
			t.Fatalf("neutral factor leaked into output: %q", f.Name)
		}
	}
}

func TestExplainRanksPenaltiesByDistanceToo(t *testing.T) {
	// A strong penalty (0.4) must show ahead of a mild boost (1.1).
	b := &intel.Briefing{
		Items: []intel.ScoredItem{
			{
				Item: intel.IntelItem{ID: "penalized", Title: "overshare"},
				Breakdown: intel.ScoreBreakdown{
					Freshness:     1.1,
					RepeatPenalty: 0.4,
				},
			},
		},
	}
	exp, _ := Explain(b, "penalized")
	if len(exp.Factors) == 0 || exp.Factors[0].Name != "repeat penalty" {
		t.Fatalf("penalty should rank first by magnitude: %+v", exp.Factors)
	}
}

func TestExplainFreshnessCopyUsesPublishedAt(t *testing.T) {
	now := time.Now()
	b := &intel.Briefing{
		Items: []intel.ScoredItem{
			{
				Item: intel.IntelItem{
					ID:          "fresh",
					Title:       "Fresh",
					PublishedAt: now.Add(-90 * time.Minute),
				},
				Breakdown: intel.ScoreBreakdown{Freshness: 1.4},
			},
		},
	}
	exp, _ := Explain(b, "fresh")
	if !strings.Contains(exp.WhyItMatters, "Fresh") {
		t.Fatalf("published recently should produce a Fresh note: %q", exp.WhyItMatters)
	}
}

func TestExplainDeterministicForSameBriefing(t *testing.T) {
	b := &intel.Briefing{
		Items: []intel.ScoredItem{
			{Item: intel.IntelItem{ID: "a", SourceName: "HN", Title: "a"}, Score: 1, Breakdown: intel.ScoreBreakdown{Freshness: 1.5, Resonance: 1.3}},
			{Item: intel.IntelItem{ID: "b", SourceName: "TLDR", Title: "a"}, Score: 1, Breakdown: intel.ScoreBreakdown{Freshness: 1.5}},
		},
		Clusters: []intel.ThemeCluster{
			{ID: "c", Slug: "ai", Label: "AI", Representative: "a", ItemIDs: []string{"a", "b"}, Score: 2},
		},
	}
	first, _ := Explain(b, "a")
	second, _ := Explain(b, "a")
	if first.WhyItMatters != second.WhyItMatters || first.WhyYouSee != second.WhyYouSee {
		t.Fatalf("explain output should be deterministic:\n1: %+v\n2: %+v", first, second)
	}
	if len(first.Factors) != len(second.Factors) {
		t.Fatalf("factor count drifted: %d vs %d", len(first.Factors), len(second.Factors))
	}
	for i := range first.Factors {
		if first.Factors[i] != second.Factors[i] {
			t.Fatalf("factor %d drifted: %+v vs %+v", i, first.Factors[i], second.Factors[i])
		}
	}
}

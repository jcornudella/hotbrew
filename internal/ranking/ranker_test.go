package ranking

import (
	"math"
	"testing"
	"time"

	"github.com/jcornudella/hotbrew/pkg/trss"
)

func TestRecencyScoreUsesExponentialDecayWithFloor(t *testing.T) {
	now := time.Date(2026, 3, 9, 12, 0, 0, 0, time.UTC)

	fresh := RecencyScore(now.Add(-24*time.Hour), now)
	wantFresh := math.Exp(-1)
	if math.Abs(fresh-wantFresh) > 0.000001 {
		t.Fatalf("fresh recency mismatch: got %f want %f", fresh, wantFresh)
	}

	old := RecencyScore(now.Add(-240*time.Hour), now)
	if old != 0.1 {
		t.Fatalf("old recency floor mismatch: got %f want 0.1", old)
	}
}

func TestEngagementScoreMatchesCurrentNormalization(t *testing.T) {
	score := EngagementScore(map[string]any{
		"points":   200,
		"comments": 100,
	})
	want := math.Log(1+250) / math.Log(1+500)
	if math.Abs(score-want) > 0.000001 {
		t.Fatalf("engagement mismatch: got %f want %f", score, want)
	}
}

func TestSourceWeightDefaultsToOne(t *testing.T) {
	if got := SourceWeight("Hacker News", nil); got != 1.0 {
		t.Fatalf("nil weights mismatch: got %f want 1.0", got)
	}
	if got := SourceWeight("Hacker News", map[string]float64{"GitHub": 2}); got != 1.0 {
		t.Fatalf("missing weight mismatch: got %f want 1.0", got)
	}
}

func TestUserBoostPrefersMatchingTagThenSource(t *testing.T) {
	item := trss.Item{
		Tags:   []string{"ai", "agents"},
		Source: trss.ItemSource{Name: "Hacker News"},
	}
	boosts := map[string]float64{
		"ai":          2.0,
		"Hacker News": 1.5,
	}
	if got := UserBoost(item, boosts); got != 2.0 {
		t.Fatalf("boost mismatch: got %f want 2.0", got)
	}
}

func TestRankItemsMatchesLegacyFinalScore(t *testing.T) {
	now := time.Date(2026, 3, 9, 12, 0, 0, 0, time.UTC)
	published := now.Add(-6 * time.Hour)
	item := trss.Item{
		ID:          "hn-1",
		PublishedAt: published,
		Tags:        []string{"ai"},
		Source:      trss.ItemSource{Name: "Hacker News"},
		Engagement: map[string]any{
			"points":   120,
			"comments": 30,
		},
	}

	ranked := RankItems([]trss.Item{item}, map[string]float64{"Hacker News": 1.2}, map[string]float64{"ai": 2.0}, now)
	if len(ranked) != 1 {
		t.Fatalf("ranked count mismatch: got %d want 1", len(ranked))
	}

	want := RecencyScore(published, now) * 1.2 * EngagementScore(item.Engagement) * 2.0
	if math.Abs(ranked[0].Score-want) > 0.000001 {
		t.Fatalf("final score mismatch: got %f want %f", ranked[0].Score, want)
	}
	if math.Abs(ranked[0].Breakdown.Final-want) > 0.000001 {
		t.Fatalf("breakdown final mismatch: got %f want %f", ranked[0].Breakdown.Final, want)
	}
}

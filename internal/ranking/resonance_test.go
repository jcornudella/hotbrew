package ranking

import (
	"testing"
	"time"

	"github.com/jcornudella/hotbrew/pkg/trss"
)

func TestComputeResonanceBoostsMultiSourceStories(t *testing.T) {
	items := []trss.Item{
		{ID: "hn-1", URLCanonical: "https://example.com/big-launch", Title: "Anthropic launches Opus 4.8", Source: trss.ItemSource{Name: "Hacker News"}},
		{ID: "tldr-1", URLCanonical: "https://example.com/big-launch", Title: "Anthropic launches Opus 4.8", Source: trss.ItemSource{Name: "TLDR"}},
		{ID: "lobs-1", URLCanonical: "https://example.com/big-launch", Title: "Anthropic launches Opus 4.8", Source: trss.ItemSource{Name: "Lobsters"}},
		{ID: "solo-1", URLCanonical: "https://other.com/noise", Title: "An uninteresting post", Source: trss.ItemSource{Name: "Hacker News"}},
	}

	scores := ComputeResonance(items)
	if scores["hn-1"] <= scores["solo-1"] {
		t.Fatalf("multi-source story should beat solo: %+v", scores)
	}
	want := 1.0 + resonanceStep*2 // 3 distinct sources share the URL
	if scores["hn-1"] != want {
		t.Fatalf("hn-1 resonance mismatch: got %f want %f", scores["hn-1"], want)
	}
	if scores["solo-1"] != 1.0 {
		t.Fatalf("solo item should have neutral resonance, got %f", scores["solo-1"])
	}
}

func TestComputeResonanceIgnoresSameSourceDuplicates(t *testing.T) {
	items := []trss.Item{
		{ID: "hn-a", URLCanonical: "https://example.com/post", Title: "Identical title here", Source: trss.ItemSource{Name: "Hacker News"}},
		{ID: "hn-b", URLCanonical: "https://example.com/post", Title: "Identical title here", Source: trss.ItemSource{Name: "Hacker News"}},
	}

	scores := ComputeResonance(items)
	if scores["hn-a"] != 1.0 {
		t.Fatalf("same-source duplicates should not boost, got %f", scores["hn-a"])
	}
}

func TestComputeResonanceMatchesByNormalizedTitleWhenURLsDiffer(t *testing.T) {
	items := []trss.Item{
		{ID: "a", URL: "https://a.example/1", Title: "Agents take over DevTools!", Source: trss.ItemSource{Name: "HN"}},
		{ID: "b", URL: "https://b.example/2", Title: "agents take over devtools?", Source: trss.ItemSource{Name: "TLDR"}},
	}

	scores := ComputeResonance(items)
	if scores["a"] <= 1.0 {
		t.Fatalf("normalized-title match should boost: %+v", scores)
	}
}

func TestComputeResonanceCapsAtMax(t *testing.T) {
	items := make([]trss.Item, 0, 10)
	for i := 0; i < 10; i++ {
		items = append(items, trss.Item{
			ID:           string(rune('a' + i)),
			URLCanonical: "https://example.com/hot",
			Title:        "The only story",
			Source:       trss.ItemSource{Name: "Source" + string(rune('A'+i))},
		})
	}
	scores := ComputeResonance(items)
	if scores["a"] != resonanceCap {
		t.Fatalf("expected cap %f, got %f", resonanceCap, scores["a"])
	}
}

func TestComputeRepeatPenaltyDemotesDominantDomain(t *testing.T) {
	items := []trss.Item{}
	for i := 0; i < 8; i++ {
		items = append(items, trss.Item{
			ID:     "noise-" + string(rune('0'+i)),
			URL:    "https://noisy.example/post" + string(rune('0'+i)),
			Source: trss.ItemSource{Name: "Source" + string(rune('A'+i))},
		})
	}
	items = append(items, trss.Item{
		ID:     "signal",
		URL:    "https://quiet.example/one",
		Source: trss.ItemSource{Name: "Quiet"},
	})

	scores := ComputeRepeatPenalty(items)
	if scores["noise-0"] >= 1.0 {
		t.Fatalf("overrepresented domain should be penalized, got %f", scores["noise-0"])
	}
	if scores["signal"] != 1.0 {
		t.Fatalf("novel item should not be penalized, got %f", scores["signal"])
	}
}

func TestComputeRepeatPenaltyDemotesDominantSource(t *testing.T) {
	items := []trss.Item{}
	for i := 0; i < 7; i++ {
		items = append(items, trss.Item{
			ID:     "hn-" + string(rune('0'+i)),
			URL:    "https://host" + string(rune('0'+i)) + ".example/p",
			Source: trss.ItemSource{Name: "Hacker News"},
		})
	}
	items = append(items, trss.Item{
		ID:     "solo",
		URL:    "https://elsewhere.example/p",
		Source: trss.ItemSource{Name: "TLDR"},
	})

	scores := ComputeRepeatPenalty(items)
	if scores["hn-0"] >= 1.0 {
		t.Fatalf("overrepresented source should be penalized, got %f", scores["hn-0"])
	}
	if scores["solo"] != 1.0 {
		t.Fatalf("non-dominant source should not be penalized, got %f", scores["solo"])
	}
}

func TestComputeRepeatPenaltyFloorsAtMin(t *testing.T) {
	items := []trss.Item{}
	for i := 0; i < 100; i++ {
		items = append(items, trss.Item{
			ID:     "x-" + timeTag(i),
			URL:    "https://spam.example/" + timeTag(i),
			Source: trss.ItemSource{Name: "Spam"},
		})
	}
	scores := ComputeRepeatPenalty(items)
	if scores["x-"+timeTag(0)] != repeatPenaltyMin {
		t.Fatalf("expected floor %f, got %f", repeatPenaltyMin, scores["x-"+timeTag(0)])
	}
}

func TestRankItemsBoostsResonantStoryAboveSolo(t *testing.T) {
	now := time.Date(2026, 4, 1, 12, 0, 0, 0, time.UTC)
	published := now.Add(-2 * time.Hour)

	items := []trss.Item{
		{ID: "solo", Title: "Interesting but niche essay on compilers", URL: "https://niche.example/essay", PublishedAt: published, Source: trss.ItemSource{Name: "HN"}, Engagement: map[string]any{"points": 400.0}},
		{ID: "res-a", Title: "Big launch everyone is talking about", URLCanonical: "https://buzz.example/launch", PublishedAt: published, Source: trss.ItemSource{Name: "HN"}, Engagement: map[string]any{"points": 200.0}},
		{ID: "res-b", Title: "Big launch everyone is talking about", URLCanonical: "https://buzz.example/launch", PublishedAt: published, Source: trss.ItemSource{Name: "TLDR"}, Engagement: map[string]any{"points": 200.0}},
		{ID: "res-c", Title: "Big launch everyone is talking about", URLCanonical: "https://buzz.example/launch", PublishedAt: published, Source: trss.ItemSource{Name: "Lobsters"}, Engagement: map[string]any{"points": 200.0}},
	}

	ranked := RankItems(items, nil, nil, now)
	scoreByID := map[string]float64{}
	for _, r := range ranked {
		scoreByID[r.Item.ID] = r.Score
	}

	if scoreByID["res-a"] <= scoreByID["solo"] {
		t.Fatalf("resonant story should outrank solo despite lower engagement: %+v", scoreByID)
	}
}

func timeTag(i int) string {
	// deterministic short tag for test IDs
	return time.Date(2026, 1, 1, 0, 0, i, 0, time.UTC).Format("150405")
}

package ranking

import (
	"testing"
	"time"

	"github.com/jcornudella/hotbrew/pkg/trss"
)

func TestThemeMultiplierFollowBoosts(t *testing.T) {
	got := ThemeMultiplier("ai", map[string]string{"ai": "follow"})
	if got != FollowedThemeBoost {
		t.Errorf("follow multiplier = %v, want %v", got, FollowedThemeBoost)
	}
}

func TestThemeMultiplierMuteDemotesToZero(t *testing.T) {
	got := ThemeMultiplier("ai", map[string]string{"ai": "mute"})
	if got != MutedThemeBoost {
		t.Errorf("mute multiplier = %v, want %v", got, MutedThemeBoost)
	}
}

func TestThemeMultiplierNeutralByDefault(t *testing.T) {
	cases := []struct {
		name  string
		slug  string
		prefs map[string]string
	}{
		{"empty slug", "", map[string]string{"ai": "follow"}},
		{"empty prefs", "ai", nil},
		{"unknown slug", "papers", map[string]string{"ai": "follow"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := ThemeMultiplier(tc.slug, tc.prefs); got != 1.0 {
				t.Errorf("got %v, want 1.0", got)
			}
		})
	}
}

// RankItemsWith should fold the per-item theme multiplier directly
// into TopicMatch, so a followed-theme item outranks an identical
// unfollowed-theme item even when every other signal matches.
func TestRankItemsWithAppliesTopicBoosts(t *testing.T) {
	now := time.Now().UTC()
	published := now.Add(-1 * time.Hour)
	a := trss.Item{
		ID:          "a",
		Fingerprint: "fpa",
		Title:       "story a",
		URL:         "https://example.com/a",
		Source:      trss.ItemSource{Name: "HN"},
		PublishedAt: published,
	}
	b := trss.Item{
		ID:          "b",
		Fingerprint: "fpb",
		Title:       "story b",
		URL:         "https://example.com/b",
		Source:      trss.ItemSource{Name: "HN"},
		PublishedAt: published,
	}

	ranked := RankItemsWith(
		[]trss.Item{a, b},
		nil, nil, now, nil, nil,
		map[string]float64{"a": FollowedThemeBoost, "b": 1.0},
	)
	if len(ranked) != 2 {
		t.Fatalf("ranked len = %d, want 2", len(ranked))
	}
	byID := map[string]float64{}
	for _, r := range ranked {
		byID[r.Item.ID] = r.Score
	}
	if byID["a"] <= byID["b"] {
		t.Errorf("followed item should outrank neutral: a=%v b=%v", byID["a"], byID["b"])
	}
}

func TestRankItemsWithMutedThemeZeroesScore(t *testing.T) {
	now := time.Now().UTC()
	item := trss.Item{
		ID:          "x",
		Fingerprint: "fpx",
		Title:       "story x",
		URL:         "https://example.com/x",
		Source:      trss.ItemSource{Name: "HN"},
		PublishedAt: now.Add(-1 * time.Hour),
	}
	ranked := RankItemsWith(
		[]trss.Item{item},
		nil, nil, now, nil, nil,
		map[string]float64{"x": MutedThemeBoost},
	)
	if ranked[0].Score != 0 {
		t.Errorf("muted item should score 0, got %v", ranked[0].Score)
	}
}

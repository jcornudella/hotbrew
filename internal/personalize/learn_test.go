package personalize

import (
	"math"
	"testing"
	"time"
)

func TestDecayFactor(t *testing.T) {
	half := 14 * 24 * time.Hour
	cases := []struct {
		age  time.Duration
		want float64
	}{
		{0, 1.0},
		{half, 0.5},
		{2 * half, 0.25},
		{4 * half, 0.0625},
	}
	for _, c := range cases {
		got := decayFactor(c.age, half)
		if math.Abs(got-c.want) > 1e-6 {
			t.Errorf("age=%v: got %v, want %v", c.age, got, c.want)
		}
	}
}

func TestNormalizeRescalesToCapAndPreservesSign(t *testing.T) {
	in := map[string]float64{
		"ai":      6.0,
		"papers":  3.0,
		"general": -2.0,
	}
	got := normalize(in, 1.5)

	scores := map[string]float64{}
	for _, r := range got {
		scores[r.Key] = r.Score
	}

	if math.Abs(scores["ai"]-1.5) > 1e-6 {
		t.Errorf("ai should rescale to cap, got %v", scores["ai"])
	}
	if scores["papers"] <= 0 || scores["papers"] >= 1.5 {
		t.Errorf("papers should be positive and below cap, got %v", scores["papers"])
	}
	if scores["general"] >= 0 {
		t.Errorf("general should keep negative sign, got %v", scores["general"])
	}
}

func TestNormalizeEmpty(t *testing.T) {
	if got := normalize(nil, 1.0); got != nil {
		t.Errorf("expected nil, got %v", got)
	}
}

func TestNormalizeNoCapPreservesAbsoluteValues(t *testing.T) {
	in := map[string]float64{"spam.com": -10.0}
	got := normalize(in, 0)
	if len(got) != 1 || got[0].Score != -10.0 {
		t.Errorf("uncapped negative should pass through, got %+v", got)
	}
}

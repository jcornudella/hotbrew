package ranking

import (
	"math"
	"time"

	"github.com/jcornudella/hotbrew/internal/intel"
	"github.com/jcornudella/hotbrew/pkg/trss"
)

// ComputeSignals derives the phase-1 ranking features for an item:
// freshness, source authority, and engagement. Later signals (resonance,
// novelty, topic match, etc.) will be filled in by higher layers with
// access to corpus-wide context.
func ComputeSignals(item trss.Item, sourceWeights map[string]float64, now time.Time) intel.ItemSignals {
	if now.IsZero() {
		now = time.Now()
	}
	return intel.ItemSignals{
		Freshness:       RecencyScore(item.PublishedAt, now),
		SourceAuthority: SourceWeight(item.Source.Name, sourceWeights),
		Engagement:      EngagementScore(item.Engagement),
	}
}

// RecencyScore decays exponentially with age and floors at 0.1.
func RecencyScore(published time.Time, now time.Time) float64 {
	ageHours := now.Sub(published).Hours()
	if ageHours < 0 {
		ageHours = 0
	}
	score := math.Exp(-ageHours / 24.0)
	if score < 0.1 {
		return 0.1
	}
	return score
}

// SourceWeight returns the configured weight for a source, defaulting to 1.0.
func SourceWeight(sourceName string, weights map[string]float64) float64 {
	if weights == nil {
		return 1.0
	}
	if w, ok := weights[sourceName]; ok {
		return w
	}
	return 1.0
}

// EngagementScore normalizes available engagement signals.
func EngagementScore(engagement map[string]any) float64 {
	if engagement == nil {
		return 1.0
	}

	points := extractFloat(engagement, "points")
	stars := extractFloat(engagement, "stars")
	comments := extractFloat(engagement, "comments")

	signal := points
	if stars > signal {
		signal = stars
	}
	signal += comments * 0.5

	if signal <= 0 {
		return 1.0
	}

	score := math.Log(1+signal) / math.Log(1+500)
	if score > 2.0 {
		return 2.0
	}
	return score
}

// UserBoost applies tag/source boosts to a TRSS item.
func UserBoost(item trss.Item, boosts map[string]float64) float64 {
	if boosts == nil {
		return 1.0
	}
	for _, tag := range item.Tags {
		if b, ok := boosts[tag]; ok {
			return b
		}
	}
	if b, ok := boosts[item.Source.Name]; ok {
		return b
	}
	return 1.0
}

func extractFloat(m map[string]any, key string) float64 {
	v, ok := m[key]
	if !ok {
		return 0
	}
	switch n := v.(type) {
	case float64:
		return n
	case int:
		return float64(n)
	case int64:
		return float64(n)
	default:
		return 0
	}
}

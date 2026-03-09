package ranking

import (
	"math"
	"time"

	"github.com/jcornudella/hotbrew/internal/intel"
	"github.com/jcornudella/hotbrew/pkg/trss"
)

// RankItems computes scores and breakdowns for TRSS items.
func RankItems(items []trss.Item, sourceWeights map[string]float64, boosts map[string]float64, now time.Time) []intel.ScoredItem {
	if now.IsZero() {
		now = time.Now()
	}

	ranked := make([]intel.ScoredItem, 0, len(items))
	for _, item := range items {
		freshness := RecencyScore(item.PublishedAt, now)
		authority := SourceWeight(item.Source.Name, sourceWeights)
		engagement := EngagementScore(item.Engagement)
		topicMatch := UserBoost(item, boosts)
		final := freshness * authority * engagement * topicMatch

		ranked = append(ranked, intel.ScoredItem{
			Item:  intel.FromTRSSItem(item),
			Score: final,
			Breakdown: intel.ScoreBreakdown{
				Freshness:  freshness,
				Authority:  authority,
				Engagement: engagement,
				TopicMatch: topicMatch,
				Final:      final,
			},
		})
	}
	return ranked
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
	case jsonNumber:
		f, _ := n.Float64()
		return f
	default:
		return 0
	}
}

type jsonNumber interface {
	Float64() (float64, error)
}

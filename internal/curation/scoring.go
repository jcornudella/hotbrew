package curation

import (
	"math"
	"time"

	"github.com/jcornudella/hotbrew/pkg/trss"
)

// ScoreItems computes the final score for each item.
// score = recency * source_weight * engagement * user_boost
func ScoreItems(items []trss.Item, sourceWeights map[string]float64, boosts map[string]float64) []trss.Item {
	now := time.Now()

	for i := range items {
		recency := recencyScore(items[i].PublishedAt, now)
		weight := getSourceWeight(items[i].Source.Name, sourceWeights)
		engagement := engagementScore(items[i].Engagement)
		boost := getUserBoost(items[i], boosts)

		items[i].Score = recency * weight * engagement * boost
	}

	return items
}

// recencyScore decays exponentially with age.
// exp(-age_hours / 24), floor 0.1
func recencyScore(published time.Time, now time.Time) float64 {
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

// engagementScore normalizes engagement signals.
// log(1 + points) / log(1 + 500), cap 2.0
func engagementScore(engagement map[string]any) float64 {
	if engagement == nil {
		return 1.0
	}

	points := extractFloat(engagement, "points")
	stars := extractFloat(engagement, "stars")
	comments := extractFloat(engagement, "comments")

	// Use the highest signal
	signal := points
	if stars > signal {
		signal = stars
	}
	// Comments contribute at half weight
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

// getSourceWeight returns the weight for a source (default 1.0).
func getSourceWeight(sourceName string, weights map[string]float64) float64 {
	if weights == nil {
		return 1.0
	}
	if w, ok := weights[sourceName]; ok {
		return w
	}
	return 1.0
}

// getUserBoost applies user boost/mute rules.
// 2.0 for boosted, 0.0 for muted, 1.0 for default.
func getUserBoost(item trss.Item, boosts map[string]float64) float64 {
	if boosts == nil {
		return 1.0
	}

	// Check tags
	for _, tag := range item.Tags {
		if b, ok := boosts[tag]; ok {
			return b
		}
	}

	// Check source name
	if b, ok := boosts[item.Source.Name]; ok {
		return b
	}

	return 1.0
}

// extractFloat tries to get a numeric value from a map.
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
	case json_number:
		f, _ := n.Float64()
		return f
	default:
		return 0
	}
}

// json_number is an interface matching encoding/json.Number.
type json_number interface {
	Float64() (float64, error)
}

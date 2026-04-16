package ranking

import (
	"time"

	"github.com/jcornudella/hotbrew/internal/intel"
	"github.com/jcornudella/hotbrew/pkg/trss"
)

// RankItems scores items using resonance/repeat computed from the same
// item set. Convenient for tests and simple callers. Use RankItemsWith
// when the corpus for resonance differs from the scoring set (e.g.
// pre-dedup resonance, post-dedup scoring).
func RankItems(items []trss.Item, sourceWeights map[string]float64, boosts map[string]float64, now time.Time) []intel.ScoredItem {
	return RankItemsWith(items, sourceWeights, boosts, now, ComputeResonance(items), ComputeRepeatPenalty(items), nil)
}

// RankItemsWith scores items against externally-provided corpus
// signals. Missing keys default to a neutral 1.0 so items outside the
// resonance/repeat corpus are neither rewarded nor penalized.
//
// topicBoosts is an optional per-item override applied on top of
// UserBoost — used for theme follow/mute preferences that the caller
// (curation) computes ahead of time via clustering labels.
func RankItemsWith(
	items []trss.Item,
	sourceWeights map[string]float64,
	boosts map[string]float64,
	now time.Time,
	resonance map[string]float64,
	repeat map[string]float64,
	topicBoosts map[string]float64,
) []intel.ScoredItem {
	if now.IsZero() {
		now = time.Now()
	}

	ranked := make([]intel.ScoredItem, 0, len(items))
	for _, item := range items {
		signals := ComputeSignals(item, sourceWeights, now)
		signals.Resonance = lookupMultiplier(resonance, item.ID)
		signals.RepeatPenalty = lookupMultiplier(repeat, item.ID)
		signals.TopicMatch = UserBoost(item, boosts) * lookupMultiplier(topicBoosts, item.ID)

		final := signals.Freshness *
			signals.SourceAuthority *
			signals.Engagement *
			signals.TopicMatch *
			signals.Resonance *
			signals.RepeatPenalty

		ranked = append(ranked, intel.ScoredItem{
			Item:  intel.FromTRSSItem(item),
			Score: final,
			Breakdown: intel.ScoreBreakdown{
				Freshness:     signals.Freshness,
				Authority:     signals.SourceAuthority,
				Engagement:    signals.Engagement,
				Resonance:     signals.Resonance,
				RepeatPenalty: signals.RepeatPenalty,
				TopicMatch:    signals.TopicMatch,
				Final:         final,
			},
		})
	}
	return ranked
}

func lookupMultiplier(m map[string]float64, id string) float64 {
	if v, ok := m[id]; ok {
		return v
	}
	return 1.0
}

package ranking

import (
	"time"

	"github.com/jcornudella/hotbrew/internal/intel"
	"github.com/jcornudella/hotbrew/pkg/trss"
)

// RankItems computes scores and breakdowns for TRSS items. Per-item
// signals (freshness, authority, engagement) are combined with
// corpus-wide signals (resonance, repeat penalty) and user boosts to
// produce a final multiplicative score.
func RankItems(items []trss.Item, sourceWeights map[string]float64, boosts map[string]float64, now time.Time) []intel.ScoredItem {
	if now.IsZero() {
		now = time.Now()
	}

	resonance := ComputeResonance(items)
	repeat := ComputeRepeatPenalty(items)

	ranked := make([]intel.ScoredItem, 0, len(items))
	for _, item := range items {
		signals := ComputeSignals(item, sourceWeights, now)
		signals.Resonance = resonance[item.ID]
		signals.RepeatPenalty = repeat[item.ID]
		signals.TopicMatch = UserBoost(item, boosts)

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

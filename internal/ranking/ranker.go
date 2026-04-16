package ranking

import (
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
		signals := ComputeSignals(item, sourceWeights, now)
		topicMatch := UserBoost(item, boosts)
		final := signals.Freshness * signals.SourceAuthority * signals.Engagement * topicMatch

		ranked = append(ranked, intel.ScoredItem{
			Item:  intel.FromTRSSItem(item),
			Score: final,
			Breakdown: intel.ScoreBreakdown{
				Freshness:  signals.Freshness,
				Authority:  signals.SourceAuthority,
				Engagement: signals.Engagement,
				TopicMatch: topicMatch,
				Final:      final,
			},
		})
	}
	return ranked
}

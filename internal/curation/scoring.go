package curation

import (
	"time"

	"github.com/jcornudella/hotbrew/internal/ranking"
	"github.com/jcornudella/hotbrew/pkg/trss"
)

// ScoreItems computes final scores for items via the ranking package while
// preserving the legacy curation interface.
func ScoreItems(items []trss.Item, sourceWeights map[string]float64, boosts map[string]float64) []trss.Item {
	ranked := ranking.RankItems(items, sourceWeights, boosts, time.Now())
	scored := make([]trss.Item, 0, len(ranked))
	for i, rankedItem := range ranked {
		item := items[i]
		item.Score = rankedItem.Score
		scored = append(scored, item)
	}
	return scored
}

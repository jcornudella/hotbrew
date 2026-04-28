package personalize

import (
	"sort"

	"github.com/jcornudella/hotbrew/internal/store"
)

func sortRowsDesc(rows []store.AffinityRow) {
	sort.SliceStable(rows, func(i, j int) bool {
		if rows[i].Score != rows[j].Score {
			return rows[i].Score > rows[j].Score
		}
		return rows[i].Key < rows[j].Key
	})
}

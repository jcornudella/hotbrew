package clustering

import (
	"crypto/sha1"
	"encoding/hex"
	"sort"

	"github.com/jcornudella/hotbrew/internal/intel"
	"github.com/jcornudella/hotbrew/pkg/trss"
)

// Cluster groups items into deterministic theme clusters. Two items
// end up in the same cluster when they share any of:
//   - canonical URL
//   - normalized title (long enough to carry signal)
//   - GitHub owner/repo signature
//
// Input order is preserved in the representative selection: highest
// score wins, ties broken by earlier position.
func Cluster(items []trss.Item) []intel.ThemeCluster {
	if len(items) == 0 {
		return nil
	}

	uf := newUnionFind(len(items))
	indexByID := make(map[string]int, len(items))
	for i, item := range items {
		indexByID[item.ID] = i
	}

	unionBy(uf, items, func(it trss.Item) string { return CanonicalKey(it) })
	unionBy(uf, items, func(it trss.Item) string {
		title := NormalizeTitle(it.Title)
		if len(title) < titleMatchMinimum {
			return ""
		}
		return "title:" + title
	})
	unionBy(uf, items, func(it trss.Item) string {
		sig := RepoSignature(it)
		if sig == "" {
			return ""
		}
		return "repo:" + sig
	})

	groups := map[int][]int{}
	for i := range items {
		root := uf.find(i)
		groups[root] = append(groups[root], i)
	}

	clusters := make([]intel.ThemeCluster, 0, len(groups))
	for _, indices := range groups {
		members := make([]trss.Item, 0, len(indices))
		ids := make([]string, 0, len(indices))
		for _, i := range indices {
			members = append(members, items[i])
			ids = append(ids, items[i].ID)
		}
		sort.Strings(ids)

		rep, score := pickRepresentative(members)
		label := LabelForItems(members)

		clusters = append(clusters, intel.ThemeCluster{
			ID:             clusterID(label.Slug, ids),
			Label:          label.Display,
			Slug:           label.Slug,
			Representative: rep,
			ItemIDs:        ids,
			Themes:         []string{label.Slug},
			Score:          score,
		})
	}

	sort.Slice(clusters, func(i, j int) bool {
		if clusters[i].Score != clusters[j].Score {
			return clusters[i].Score > clusters[j].Score
		}
		return clusters[i].ID < clusters[j].ID
	})
	return clusters
}

// unionBy unions every pair of items sharing a non-empty key produced
// by keyFn. Empty keys are skipped so items without URL/title signal
// stay in their own cluster.
func unionBy(uf *unionFind, items []trss.Item, keyFn func(trss.Item) string) {
	byKey := map[string]int{}
	for i, item := range items {
		key := keyFn(item)
		if key == "" {
			continue
		}
		if first, ok := byKey[key]; ok {
			uf.union(first, i)
		} else {
			byKey[key] = i
		}
	}
}

func pickRepresentative(items []trss.Item) (string, float64) {
	if len(items) == 0 {
		return "", 0
	}
	best := items[0]
	bestScore := items[0].Score
	for _, it := range items[1:] {
		if it.Score > bestScore {
			best = it
			bestScore = it.Score
		}
	}
	return best.ID, bestScore
}

func clusterID(slug string, memberIDs []string) string {
	h := sha1.New()
	h.Write([]byte(slug))
	for _, id := range memberIDs {
		h.Write([]byte{0})
		h.Write([]byte(id))
	}
	return "c_" + hex.EncodeToString(h.Sum(nil))[:16]
}

type unionFind struct {
	parent []int
	rank   []int
}

func newUnionFind(n int) *unionFind {
	uf := &unionFind{parent: make([]int, n), rank: make([]int, n)}
	for i := range uf.parent {
		uf.parent[i] = i
	}
	return uf
}

func (uf *unionFind) find(i int) int {
	for uf.parent[i] != i {
		uf.parent[i] = uf.parent[uf.parent[i]]
		i = uf.parent[i]
	}
	return i
}

func (uf *unionFind) union(a, b int) {
	ra, rb := uf.find(a), uf.find(b)
	if ra == rb {
		return
	}
	if uf.rank[ra] < uf.rank[rb] {
		ra, rb = rb, ra
	}
	uf.parent[rb] = ra
	if uf.rank[ra] == uf.rank[rb] {
		uf.rank[ra]++
	}
}

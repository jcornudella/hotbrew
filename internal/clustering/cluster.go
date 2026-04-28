package clustering

import (
	"context"
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
//
// Labels come from the keyword matcher in labels.go. For LLM-based
// labeling, use ClusterWith with a non-nil Labeler.
func Cluster(items []trss.Item) []intel.ThemeCluster {
	return ClusterWith(context.Background(), items, nil)
}

// ClusterWith is the labeler-aware variant. When labeler is non-nil,
// the formed clusters are batched through Labeler.LabelClusters in a
// single call. On any labeler error — network failure, parse error,
// timeout — we fall back to the keyword matcher for *every* cluster.
// Per-cluster fallback would mean making a second API call to relabel
// only the failed ones, which complicates the cost model without
// improving the worst case.
//
// The keyword matcher's "repo" override (clusters of pure
// github-trending items) is applied AFTER the labeler runs, so the
// LLM never has to learn that special case — labeler's slug for a
// pure-github cluster is overwritten with "repo" deterministically.
func ClusterWith(ctx context.Context, items []trss.Item, labeler Labeler) []intel.ThemeCluster {
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

	// Phase 1: build clusters with keyword labels. We stamp keyword
	// labels first so that if the labeler bails out (or is nil), every
	// cluster already has a usable slug — the LLM pass becomes a
	// strict upgrade rather than a hard dependency.
	type built struct {
		members []trss.Item
		ids     []string
		rep     string
		score   float64
		keyword Label
	}

	groupKeys := make([]int, 0, len(groups))
	for k := range groups {
		groupKeys = append(groupKeys, k)
	}
	sort.Ints(groupKeys) // deterministic order for batches

	prepared := make([]built, 0, len(groupKeys))
	for _, k := range groupKeys {
		indices := groups[k]
		members := make([]trss.Item, 0, len(indices))
		ids := make([]string, 0, len(indices))
		for _, i := range indices {
			members = append(members, items[i])
			ids = append(ids, items[i].ID)
		}
		sort.Strings(ids)

		rep, score := pickRepresentative(members)
		prepared = append(prepared, built{
			members: members,
			ids:     ids,
			rep:     rep,
			score:   score,
			keyword: LabelForItems(members),
		})
	}

	// Phase 2: optionally upgrade labels via the LLM labeler. The
	// keyword pass already stamped a fallback, so a labeler error
	// degrades silently to keyword-only labeling — the briefing still
	// renders, just without the semantic boost.
	batches := make([]ClusterBatch, len(prepared))
	for i, p := range prepared {
		batches[i] = ClusterBatch{ID: batchID(i, p.ids), Items: p.members}
	}
	llmLabels := tryLLMLabel(ctx, labeler, batches)

	clusters := make([]intel.ThemeCluster, 0, len(prepared))
	for i, p := range prepared {
		// "repo" override: a cluster made entirely of github-trending
		// items is always "Repo of the Day" regardless of what the
		// keyword matcher or LLM said. Keyword matcher already encodes
		// this in labelFromSourceType; we re-apply here to make sure
		// LLM output can't pollute a pure-github cluster with a wrong
		// slug like "devtools".
		var label Label
		if forced, ok := labelFromSourceType(p.members); ok {
			label = forced
		} else if slug, ok := llmLabels[batchID(i, p.ids)]; ok {
			label = labelBySlug(slug)
		} else {
			label = p.keyword
		}

		clusters = append(clusters, intel.ThemeCluster{
			ID:             clusterID(label.Slug, p.ids),
			Label:          label.Display,
			Slug:           label.Slug,
			Representative: p.rep,
			ItemIDs:        p.ids,
			Themes:         []string{label.Slug},
			Score:          p.score,
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

// tryLLMLabel runs the optional batched labeling step. Returns a map
// from batch ID → slug for the clusters the labeler successfully
// classified. On any error or nil labeler, returns an empty map and
// the caller falls back to keyword labels.
func tryLLMLabel(ctx context.Context, labeler Labeler, batches []ClusterBatch) map[string]string {
	if labeler == nil || len(batches) == 0 {
		return nil
	}
	labels, err := labeler.LabelClusters(ctx, batches)
	if err != nil || len(labels) == 0 {
		return nil
	}
	out := make(map[string]string, len(labels))
	for _, l := range labels {
		out[l.ID] = l.Slug
	}
	return out
}

// batchID is the opaque identifier we hand to the labeler. Using
// position + first member ID rather than the final cluster ID, because
// cluster ID depends on the slug — and we don't know the slug until
// the labeler has answered.
func batchID(position int, memberIDs []string) string {
	first := ""
	if len(memberIDs) > 0 {
		first = memberIDs[0]
	}
	return "b_" + first + "_" + intToHex(position)
}

func intToHex(i int) string {
	const hexDigits = "0123456789abcdef"
	if i == 0 {
		return "0"
	}
	var buf [16]byte
	pos := len(buf)
	for i > 0 {
		pos--
		buf[pos] = hexDigits[i&0xf]
		i >>= 4
	}
	return string(buf[pos:])
}

// labelBySlug looks up the canonical Label record for a slug. Used to
// translate LLM output (which is just a slug string) back into the
// {Slug, Display} pair the briefing layer expects.
func labelBySlug(slug string) Label {
	for _, l := range KnownLabels() {
		if l.Slug == slug {
			return l
		}
	}
	return Label{Slug: "general", Display: "General"}
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

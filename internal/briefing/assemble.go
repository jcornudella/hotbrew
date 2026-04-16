package briefing

import (
	"sort"

	"github.com/jcornudella/hotbrew/internal/intel"
)

// Assemble groups a briefing's clusters into theme-based sections and
// reorders the clusters to match section priority. It expects
// Clusters to already be populated by the clustering stage; Items and
// Meta pass through untouched. Clusters fold duplicates and carry
// scores, so assembly is a pure rearrangement — no diversity or
// selection logic lives here, keeping this stage deterministic.
//
// After Assemble runs:
//   - Sections is ordered by sectionOrder (canonical theme priority)
//   - Clusters is reordered to match, highest-scoring first within
//     each section, stable across equal scores
//   - Each section's ClusterIDs lists its clusters in the same order
func Assemble(b *intel.Briefing) {
	if b == nil || len(b.Clusters) == 0 {
		b.Sections = nil
		return
	}

	bySlug := map[string][]intel.ThemeCluster{}
	for _, c := range b.Clusters {
		slug := c.Slug
		if slug == "" {
			slug = "general"
		}
		bySlug[slug] = append(bySlug[slug], c)
	}

	idsBySlug := make(map[string][]string, len(bySlug))
	for slug, clusters := range bySlug {
		sort.SliceStable(clusters, func(i, j int) bool {
			if clusters[i].Score != clusters[j].Score {
				return clusters[i].Score > clusters[j].Score
			}
			return clusters[i].ID < clusters[j].ID
		})
		bySlug[slug] = clusters
		ids := make([]string, len(clusters))
		for i, c := range clusters {
			ids[i] = c.ID
		}
		idsBySlug[slug] = ids
	}

	slugs := orderedSlugs(idsBySlug)

	sections := make([]intel.BriefingSection, 0, len(slugs))
	orderedClusters := make([]intel.ThemeCluster, 0, len(b.Clusters))
	for _, slug := range slugs {
		sections = append(sections, intel.BriefingSection{
			Name:       displayName(slug),
			Kind:       slug,
			ClusterIDs: idsBySlug[slug],
		})
		orderedClusters = append(orderedClusters, bySlug[slug]...)
	}

	b.Sections = sections
	b.Clusters = orderedClusters
}

// SupportingItems returns the item IDs in a cluster that are not the
// representative, sorted by score descending using the provided score
// map. Ties break by ID to stay deterministic across runs. Use this
// at render time to list "also covered by" links under a lead.
func SupportingItems(c intel.ThemeCluster, scores map[string]float64) []string {
	if len(c.ItemIDs) <= 1 {
		return nil
	}
	supporting := make([]string, 0, len(c.ItemIDs)-1)
	for _, id := range c.ItemIDs {
		if id == c.Representative {
			continue
		}
		supporting = append(supporting, id)
	}
	sort.SliceStable(supporting, func(i, j int) bool {
		si, sj := scores[supporting[i]], scores[supporting[j]]
		if si != sj {
			return si > sj
		}
		return supporting[i] < supporting[j]
	})
	return supporting
}

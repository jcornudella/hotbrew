package sinks

import (
	"github.com/jcornudella/hotbrew/internal/intel"
	"github.com/jcornudella/hotbrew/pkg/source"
)

// BriefingToSections flattens a theme-assembled briefing into the
// source.Section shape the TUI already renders. One section per
// theme; within each section, one Item per cluster — the cluster's
// representative, with supporting members stashed under
// Metadata["supporting"] so the expanded card can surface the "also
// covered by" list without the component layer learning about
// clusters directly.
//
// Keeping the TUI data model source-flavoured lets cluster rendering
// piggyback on every existing affordance (priority badges, nav,
// expand, open) instead of forking a parallel pipeline.
func BriefingToSections(b *intel.Briefing) []*source.Section {
	if b == nil || len(b.Sections) == 0 {
		return nil
	}

	items := indexScoredItems(b.Items)
	clusters := indexThemeClusters(b.Clusters)
	scores := scoresForItems(b.Items)

	sections := make([]*source.Section, 0, len(b.Sections))
	for _, sec := range b.Sections {
		out := &source.Section{
			Name: sec.Name,
			Icon: sectionIcon(sec.Kind),
		}
		for _, clusterID := range sec.ClusterIDs {
			cluster, ok := clusters[clusterID]
			if !ok {
				continue
			}
			lead, ok := items[cluster.Representative]
			if !ok {
				continue
			}
			item := intelToSourceItem(lead)
			item.Metadata["cluster_id"] = cluster.ID
			item.Metadata["cluster_label"] = cluster.Label
			item.Metadata["supporting"] = supportingLinks(cluster, items, scores)
			out.Items = append(out.Items, item)
		}
		if len(out.Items) == 0 {
			continue
		}
		sections = append(sections, out)
	}
	return sections
}

// sectionIcon picks a theme glyph. These are deliberately low-key —
// the theme header carries the word, the glyph just anchors the eye.
func sectionIcon(slug string) string {
	switch slug {
	case "ai":
		return "🤖"
	case "devtools":
		return "🛠"
	case "infra":
		return "🧱"
	case "startups":
		return "🚀"
	case "papers":
		return "📄"
	case "deep-read":
		return "📖"
	case "debate":
		return "💬"
	case "repo":
		return "⭐"
	default:
		return "•"
	}
}

func indexScoredItems(items []intel.ScoredItem) map[string]intel.ScoredItem {
	out := make(map[string]intel.ScoredItem, len(items))
	for _, item := range items {
		out[item.Item.ID] = item
	}
	return out
}

func indexThemeClusters(clusters []intel.ThemeCluster) map[string]intel.ThemeCluster {
	out := make(map[string]intel.ThemeCluster, len(clusters))
	for _, c := range clusters {
		out[c.ID] = c
	}
	return out
}

func scoresForItems(items []intel.ScoredItem) map[string]float64 {
	out := make(map[string]float64, len(items))
	for _, item := range items {
		out[item.Item.ID] = item.Score
	}
	return out
}

// supportingLinks returns a metadata-friendly list of the cluster's
// non-rep members, ordered by score. []map[string]any keeps the
// shape compatible with the rest of Item.Metadata (which is decoded
// into any-typed values when round-tripping through trss meta).
func supportingLinks(c intel.ThemeCluster, items map[string]intel.ScoredItem, scores map[string]float64) []map[string]any {
	if len(c.ItemIDs) <= 1 {
		return nil
	}
	supporting := make([]map[string]any, 0, len(c.ItemIDs)-1)
	for _, id := range orderSupportingIDs(c, scores) {
		item, ok := items[id]
		if !ok {
			continue
		}
		supporting = append(supporting, map[string]any{
			"title":  item.Item.Title,
			"source": item.Item.SourceName,
			"url":    item.Item.CanonicalURL,
		})
	}
	return supporting
}

// orderSupportingIDs mirrors briefing.SupportingItems but is inlined
// here to avoid a cross-package dependency from sinks up into
// briefing, which already depends on intel.
func orderSupportingIDs(c intel.ThemeCluster, scores map[string]float64) []string {
	if len(c.ItemIDs) <= 1 {
		return nil
	}
	ids := make([]string, 0, len(c.ItemIDs)-1)
	for _, id := range c.ItemIDs {
		if id == c.Representative {
			continue
		}
		ids = append(ids, id)
	}
	// Insertion sort keeps determinism: score desc, id asc on tie.
	for i := 1; i < len(ids); i++ {
		j := i
		for j > 0 {
			a, b := ids[j-1], ids[j]
			if scores[a] > scores[b] {
				break
			}
			if scores[a] == scores[b] && a < b {
				break
			}
			ids[j-1], ids[j] = b, a
			j--
		}
	}
	return ids
}

// intelToSourceItem maps a ranked intel item to the TUI's source
// shape. Mirrors trssItemToSourceItem so the two rendering paths
// stay visually consistent.
func intelToSourceItem(scored intel.ScoredItem) source.Item {
	item := scored.Item

	priority := source.Low
	switch {
	case scored.Score >= 7:
		priority = source.Urgent
	case scored.Score >= 5:
		priority = source.High
	case scored.Score >= 3:
		priority = source.Medium
	}

	meta := map[string]any{}
	for k, v := range item.Metadata {
		meta[k] = v
	}
	meta["trss_id"] = item.ID
	meta["trss_score"] = scored.Score
	meta["source_name"] = item.SourceName

	url := item.CanonicalURL
	return source.Item{
		ID:        item.ID,
		Title:     item.Title,
		Subtitle:  item.Summary,
		Body:      item.Body,
		URL:       url,
		Priority:  priority,
		Timestamp: item.PublishedAt,
		Icon:      item.SourceIcon,
		Metadata:  meta,
		Actions: []source.Action{
			{Key: "o", Label: "open", Command: url},
		},
	}
}

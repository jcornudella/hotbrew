// Package curation provides the digest curation pipeline:
// dedup, scoring, diversity enforcement, and rule application.
package curation

import (
	"strings"

	"github.com/jcornudella/hotbrew/internal/store"
	"github.com/jcornudella/hotbrew/pkg/trss"
)

// Dedup removes duplicate items using fingerprint-based exact matching
// and title similarity fuzzy matching. It records dedup edges in the store.
func Dedup(items []trss.Item, st *store.Store) []trss.Item {
	seen := map[string]int{}     // fingerprint → index of kept item
	urlSeen := map[string]int{}  // canonical URL → index
	var result []trss.Item

	for _, item := range items {
		// Exact fingerprint match
		if idx, ok := seen[item.Fingerprint]; ok {
			if st != nil {
				st.InsertDedupEdge(result[idx].ID, item.ID, 1.0)
			}
			continue
		}

		// Exact canonical URL match
		if item.URLCanonical != "" {
			if idx, ok := urlSeen[item.URLCanonical]; ok {
				if st != nil {
					st.InsertDedupEdge(result[idx].ID, item.ID, 0.95)
				}
				continue
			}
		}

		// Fuzzy title match
		if idx, dup := fuzzyTitleMatch(item.Title, result); dup {
			if st != nil {
				st.InsertDedupEdge(result[idx].ID, item.ID, 0.8)
			}
			continue
		}

		idx := len(result)
		seen[item.Fingerprint] = idx
		if item.URLCanonical != "" {
			urlSeen[item.URLCanonical] = idx
		}
		result = append(result, item)
	}

	return result
}

// fuzzyTitleMatch checks if a title is very similar to any existing item.
// Returns the index of the match and whether a match was found.
func fuzzyTitleMatch(title string, items []trss.Item) (int, bool) {
	normalized := normalizeTitle(title)
	if normalized == "" {
		return 0, false
	}

	for i, item := range items {
		existing := normalizeTitle(item.Title)
		if existing == "" {
			continue
		}

		// Exact normalized match
		if normalized == existing {
			return i, true
		}

		// One title contains the other (common with repost titles)
		if strings.Contains(normalized, existing) || strings.Contains(existing, normalized) {
			shorter := len(normalized)
			longer := len(existing)
			if shorter > longer {
				shorter, longer = longer, shorter
			}
			// Only match if the shorter string is at least 70% of the longer
			if float64(shorter)/float64(longer) > 0.7 {
				return i, true
			}
		}
	}

	return 0, false
}

// normalizeTitle lowercases and strips common noise from titles.
func normalizeTitle(t string) string {
	t = strings.ToLower(strings.TrimSpace(t))
	// Strip common prefixes/suffixes
	for _, prefix := range []string{"show hn: ", "ask hn: ", "tell hn: ", "[p] ", "[d] "} {
		t = strings.TrimPrefix(t, prefix)
	}
	return t
}

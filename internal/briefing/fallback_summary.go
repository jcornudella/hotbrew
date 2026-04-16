package briefing

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/jcornudella/hotbrew/internal/intel"
)

// FallbackSummarizer is the deterministic-template Summarizer used
// when no AI provider is configured. Same input always yields the
// same output — making it safe for tests, golden files, and any
// caller that wants stable behaviour offline.
//
// Design choices:
//   - Item summaries reuse whatever the source already supplied,
//     trimmed to the first full sentence when it runs long. The
//     body is a last-resort fallback; the title is the floor.
//   - Cluster summaries read "<Label>: <lead title>" with an "also
//     covered by" tail listing up to three distinct source names.
//     Three is a compromise between informative and terse — more
//     than that turns into a run-on sentence when rendered.
type FallbackSummarizer struct{}

// SummarizeItem returns a terse rendering of the item. Never returns
// an error — the template always has something to fall back on
// (the title is always present on a valid item).
func (FallbackSummarizer) SummarizeItem(ctx context.Context, item intel.IntelItem) (string, error) {
	if s := firstSentence(item.Summary); s != "" {
		return s, nil
	}
	if s := firstSentence(item.Body); s != "" {
		return s, nil
	}
	return strings.TrimSpace(item.Title), nil
}

// SummarizeCluster returns a single-line description of the cluster.
// Items is the caller's lookup slice — the lead is resolved via
// cluster.Representative, and supporting sources are the distinct
// SourceNames of the remaining members.
func (FallbackSummarizer) SummarizeCluster(ctx context.Context, cluster intel.ThemeCluster, items []intel.IntelItem) (string, error) {
	index := indexIntelItems(items)
	lead, ok := index[cluster.Representative]
	if !ok {
		// Fall back to the first item we can resolve. A cluster with
		// an unresolvable representative is a data bug, but we'd
		// rather print something than panic at render time.
		for _, id := range cluster.ItemIDs {
			if it, ok := index[id]; ok {
				lead = it
				break
			}
		}
	}

	label := cluster.Label
	if label == "" {
		label = "Coverage"
	}
	title := strings.TrimSpace(lead.Title)
	if title == "" {
		return label, nil
	}

	base := fmt.Sprintf("%s: %s", label, title)

	sources := supportingSourceNames(cluster, index)
	if len(sources) == 0 {
		return base, nil
	}
	return fmt.Sprintf("%s — also covered by %s", base, joinWithAnd(sources)), nil
}

// firstSentence returns the first sentence of s, trimming trailing
// punctuation. Falls back to the whole string when there's no
// sentence terminator — typical of headline-only summaries.
func firstSentence(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	for i, r := range s {
		if r == '.' || r == '!' || r == '?' {
			end := i + 1
			// Include a trailing quote or bracket so we don't cut
			// "He said "hi."" in a weird spot.
			if end < len(s) && (s[end] == '"' || s[end] == ')') {
				end++
			}
			return strings.TrimSpace(s[:end])
		}
	}
	return s
}

// supportingSourceNames collects up to three distinct source names
// from non-representative cluster members, sorted alphabetically so
// output is stable across runs. Empty names are skipped.
func supportingSourceNames(cluster intel.ThemeCluster, index map[string]intel.IntelItem) []string {
	seen := map[string]struct{}{}
	var sources []string
	for _, id := range cluster.ItemIDs {
		if id == cluster.Representative {
			continue
		}
		item, ok := index[id]
		if !ok {
			continue
		}
		name := strings.TrimSpace(item.SourceName)
		if name == "" {
			continue
		}
		if _, dupe := seen[name]; dupe {
			continue
		}
		seen[name] = struct{}{}
		sources = append(sources, name)
	}
	sort.Strings(sources)
	if len(sources) > 3 {
		sources = sources[:3]
	}
	return sources
}

// joinWithAnd writes ["a", "b", "c"] as "a, b and c". Two items
// drop the comma ("a and b"); one item passes through unchanged.
func joinWithAnd(parts []string) string {
	switch len(parts) {
	case 0:
		return ""
	case 1:
		return parts[0]
	case 2:
		return parts[0] + " and " + parts[1]
	default:
		return strings.Join(parts[:len(parts)-1], ", ") + " and " + parts[len(parts)-1]
	}
}

func indexIntelItems(items []intel.IntelItem) map[string]intel.IntelItem {
	out := make(map[string]intel.IntelItem, len(items))
	for _, item := range items {
		out[item.ID] = item
	}
	return out
}

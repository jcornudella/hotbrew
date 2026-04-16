package briefing

import (
	"net/url"
	"strings"

	"github.com/jcornudella/hotbrew/internal/intel"
)

// BalanceLimits tunes how aggressively Balance trims the briefing.
// Zero values disable the corresponding rule rather than acting as
// "no allowed" — the briefing should still render if every limit is
// unset, so a partially-configured caller can opt into one rule at
// a time without accidentally erasing their digest.
type BalanceLimits struct {
	// MaxClustersPerTheme is the hard cap on how many clusters any
	// single theme section can carry. Drops the lowest-scoring
	// clusters first. 0 disables.
	MaxClustersPerTheme int

	// MaxLeadsPerDomain prevents one domain (e.g. a prolific blog)
	// from dominating the briefing via multiple clusters whose lead
	// items all point at the same host. 0 disables.
	MaxLeadsPerDomain int

	// MaxTotalClusters caps the final number of clusters shown. Used
	// to prevent a long-tail sprawl on high-volume days. 0 disables.
	MaxTotalClusters int

	// EnsureDeepRead guarantees that a "deep-read" cluster appears in
	// the output when one exists, even if theme/domain caps would
	// have dropped it. Same idea for EnsureRepo.
	EnsureDeepRead bool
	EnsureRepo     bool
}

// DefaultBalanceLimits are the phase-1 numbers — chosen to match the
// briefing's target feel: a handful of themes, not more than a pair
// of clusters from any single domain, and a guaranteed slot for a
// long-read and a repo-of-the-day when the pipeline produced them.
func DefaultBalanceLimits() BalanceLimits {
	return BalanceLimits{
		MaxClustersPerTheme: 3,
		MaxLeadsPerDomain:   2,
		MaxTotalClusters:    10,
		EnsureDeepRead:      true,
		EnsureRepo:          true,
	}
}

// Balance trims a briefing's clusters according to limits. It runs
// after Assemble (which sorts clusters by section/score) and before
// any rendering — Balance rebuilds Sections to match the trimmed
// cluster list by delegating to Assemble again at the end, so
// callers never see a sections/clusters mismatch.
//
// The algorithm is a single priority-ordered filter pass:
//  1. Move required-theme clusters (deep-read, repo) to the front of
//     the queue so they survive the theme/domain caps.
//  2. Walk the queue; keep a cluster if every cap still has room.
//  3. Drop otherwise — silently, since this is editorial pruning,
//     not an error condition.
//
// The pass is deterministic because the queue order is deterministic
// (Assemble's output + stable required-slug promotion).
func Balance(b *intel.Briefing, limits BalanceLimits) {
	if b == nil || len(b.Clusters) == 0 {
		return
	}

	queue := prioritizeQueue(b.Clusters, limits)
	items := indexItemsByID(b.Items)

	themeCount := map[string]int{}
	domainCount := map[string]int{}
	kept := make([]intel.ThemeCluster, 0, len(queue))

	for _, c := range queue {
		if limits.MaxTotalClusters > 0 && len(kept) >= limits.MaxTotalClusters {
			break
		}
		if limits.MaxClustersPerTheme > 0 && themeCount[c.Slug] >= limits.MaxClustersPerTheme {
			continue
		}
		domain := leadDomain(c, items)
		if limits.MaxLeadsPerDomain > 0 && domain != "" && domainCount[domain] >= limits.MaxLeadsPerDomain {
			continue
		}
		kept = append(kept, c)
		themeCount[c.Slug]++
		if domain != "" {
			domainCount[domain]++
		}
	}

	b.Clusters = kept
	// Re-run Assemble so Sections reflects the pruned cluster set.
	// Scores didn't change, so the section order is identical — we
	// just rebuild the cluster ID lists within each section.
	Assemble(b)
}

// prioritizeQueue lifts required-theme clusters to the front so the
// cap pass doesn't accidentally evict them. Within required slugs
// we take the highest-scoring cluster (whatever Assemble surfaced
// first). Everything else keeps the order Assemble produced.
func prioritizeQueue(clusters []intel.ThemeCluster, limits BalanceLimits) []intel.ThemeCluster {
	required := []string{}
	if limits.EnsureDeepRead {
		required = append(required, "deep-read")
	}
	if limits.EnsureRepo {
		required = append(required, "repo")
	}

	queue := make([]intel.ThemeCluster, 0, len(clusters))
	taken := map[string]bool{}
	for _, slug := range required {
		for _, c := range clusters {
			if c.Slug == slug && !taken[c.ID] {
				queue = append(queue, c)
				taken[c.ID] = true
				break
			}
		}
	}
	for _, c := range clusters {
		if !taken[c.ID] {
			queue = append(queue, c)
		}
	}
	return queue
}

func indexItemsByID(items []intel.ScoredItem) map[string]intel.IntelItem {
	out := make(map[string]intel.IntelItem, len(items))
	for _, item := range items {
		out[item.Item.ID] = item.Item
	}
	return out
}

// leadDomain extracts the host of a cluster's representative item.
// Used for the max-leads-per-domain cap. Empty string is returned
// when the representative has no parseable URL — in that case the
// domain cap is skipped, which is deliberate: an item without a
// domain can't reasonably be said to "dominate" one.
func leadDomain(c intel.ThemeCluster, items map[string]intel.IntelItem) string {
	lead, ok := items[c.Representative]
	if !ok {
		return ""
	}
	raw := strings.TrimSpace(lead.CanonicalURL)
	if raw == "" {
		return ""
	}
	parsed, err := url.Parse(raw)
	if err != nil {
		return ""
	}
	host := strings.ToLower(parsed.Hostname())
	return strings.TrimPrefix(host, "www.")
}

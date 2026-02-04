package curation

import (
	"net/url"
	"strings"

	"github.com/jcornudella/hotbrew/pkg/trss"
)

// DiversityLimits controls the maximum concentration from any single dimension.
type DiversityLimits struct {
	MaxPerDomain       int     // Max items from one domain (default 3)
	MaxSourcePercent   float64 // Max % from one source (default 0.4)
	MaxPerTagCluster   int     // Max items sharing a dominant tag (default 3)
}

// DefaultLimits returns sensible diversity defaults.
func DefaultLimits() DiversityLimits {
	return DiversityLimits{
		MaxPerDomain:     3,
		MaxSourcePercent: 0.4,
		MaxPerTagCluster: 3,
	}
}

// EnforceDiversity filters items to ensure no single domain, source, or tag
// cluster dominates the digest. Items must be pre-sorted by score (highest first).
func EnforceDiversity(items []trss.Item, limits DiversityLimits, maxItems int) []trss.Item {
	if maxItems <= 0 {
		maxItems = 25
	}

	domainCount := map[string]int{}
	sourceCount := map[string]int{}
	tagCount := map[string]int{}

	maxFromSource := int(float64(maxItems) * limits.MaxSourcePercent)
	if maxFromSource < 1 {
		maxFromSource = 1
	}

	var result []trss.Item

	for _, item := range items {
		if len(result) >= maxItems {
			break
		}

		domain := extractDomain(item.URL)
		sourceName := item.Source.Name

		// Check domain limit
		if limits.MaxPerDomain > 0 && domain != "" && domainCount[domain] >= limits.MaxPerDomain {
			continue
		}

		// Check source limit
		if sourceCount[sourceName] >= maxFromSource {
			continue
		}

		// Check tag cluster limit
		if limits.MaxPerTagCluster > 0 && isTagSaturated(item.Tags, tagCount, limits.MaxPerTagCluster) {
			continue
		}

		// Accept the item
		result = append(result, item)
		if domain != "" {
			domainCount[domain]++
		}
		sourceCount[sourceName]++
		for _, tag := range item.Tags {
			tagCount[tag]++
		}
	}

	return result
}

// extractDomain gets the hostname from a URL.
func extractDomain(rawURL string) string {
	if rawURL == "" {
		return ""
	}
	u, err := url.Parse(rawURL)
	if err != nil {
		return ""
	}
	host := u.Hostname()
	// Strip www. prefix
	host = strings.TrimPrefix(host, "www.")
	return host
}

// isTagSaturated checks if accepting this item would exceed the tag cluster limit.
func isTagSaturated(tags []string, counts map[string]int, limit int) bool {
	for _, tag := range tags {
		if counts[tag] >= limit {
			return true
		}
	}
	return false
}

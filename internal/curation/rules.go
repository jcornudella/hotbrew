package curation

import (
	"strings"

	"github.com/jcornudella/hotbrew/internal/store"
	"github.com/jcornudella/hotbrew/pkg/trss"
)

// ApplyRules filters and boosts items according to user-defined rules.
// Returns filtered items and a boost map for scoring.
func ApplyRules(items []trss.Item, rules []store.Rule) ([]trss.Item, map[string]float64) {
	boosts := map[string]float64{}

	// Build lookup maps from rules.
	muteDomains := map[string]bool{}
	muteSources := map[string]bool{}

	for _, r := range rules {
		if !r.Enabled {
			continue
		}
		pattern := strings.ToLower(r.Pattern)

		switch r.Kind {
		case "mute_domain":
			muteDomains[pattern] = true
		case "mute_source":
			muteSources[pattern] = true
		case "boost_tag":
			boosts[pattern] = 2.0
		case "boost_domain":
			boosts[pattern] = 2.0
		}
	}

	// Filter muted items.
	var filtered []trss.Item
	for _, item := range items {
		domain := strings.ToLower(extractDomain(item.URL))
		sourceName := strings.ToLower(item.Source.Name)

		if muteDomains[domain] {
			continue
		}
		if muteSources[sourceName] {
			continue
		}

		filtered = append(filtered, item)
	}

	return filtered, boosts
}

// CountAppliedRules returns how many rules actually affected the result.
func CountAppliedRules(original, filtered int, boosts map[string]float64) int {
	count := original - filtered // muted items
	count += len(boosts)        // boost rules
	return count
}

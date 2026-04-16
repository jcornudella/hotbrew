// Package briefing assembles scored clusters into the theme-organized
// shape that CLI and TUI surfaces render. It is the final pipeline
// stage: clustering folds duplicates, ranking orders items, and
// assembly turns the result into labelled sections with leads and
// supporting links.
package briefing

import "sort"

// sectionOrder is the canonical display order for theme sections.
// Cluster slugs in this list render in the exact order given. Slugs
// not listed here fall to the tail, alphabetised, so new label slugs
// still appear in the briefing without code changes — but they only
// get priority positioning once explicitly added here.
var sectionOrder = []string{
	"ai",
	"devtools",
	"infra",
	"startups",
	"repo",
	"papers",
	"deep-read",
	"debate",
	"general",
}

// sectionDisplay maps a slug to the user-facing label. Keep in sync
// with clustering.labels — slugs are the contract between the two
// packages.
var sectionDisplay = map[string]string{
	"ai":        "AI",
	"devtools":  "Devtools",
	"infra":     "Infra",
	"startups":  "Startups",
	"papers":    "Papers",
	"deep-read": "Deep Read",
	"debate":    "Debate",
	"repo":      "Repo of the Day",
	"general":   "General",
}

// displayName returns the friendly label for a slug, falling back to
// the slug itself when we haven't explicitly named it.
func displayName(slug string) string {
	if name, ok := sectionDisplay[slug]; ok {
		return name
	}
	return slug
}

// orderedSlugs returns the slugs present in bySlug, sorted by the
// canonical sectionOrder first and then alphabetically for any
// unknown slug, so output is deterministic regardless of iteration.
func orderedSlugs(bySlug map[string][]string) []string {
	priority := make(map[string]int, len(sectionOrder))
	for i, slug := range sectionOrder {
		priority[slug] = i
	}

	slugs := make([]string, 0, len(bySlug))
	for slug := range bySlug {
		slugs = append(slugs, slug)
	}
	sort.Slice(slugs, func(i, j int) bool {
		pi, iKnown := priority[slugs[i]]
		pj, jKnown := priority[slugs[j]]
		if iKnown && jKnown {
			return pi < pj
		}
		if iKnown != jKnown {
			return iKnown
		}
		return slugs[i] < slugs[j]
	})
	return slugs
}

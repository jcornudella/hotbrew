// Package clustering groups items into deterministic themes using
// canonical URL, normalized title, and repo/domain heuristics. All
// membership decisions are explainable and free of network or LLM
// dependencies.
package clustering

import (
	"net/url"
	"strings"
	"unicode"

	"github.com/jcornudella/hotbrew/pkg/trss"
)

// titleMatchMinimum keeps short titles (e.g. "TIL", "Ask HN") from
// collapsing into one cluster just because they share a label.
const titleMatchMinimum = 10

// CanonicalKey returns a lowercase URL used to identify the same story
// across sources. Falls back to the non-canonical URL when necessary.
func CanonicalKey(item trss.Item) string {
	key := strings.TrimSpace(item.URLCanonical)
	if key == "" {
		key = strings.TrimSpace(item.URL)
	}
	return strings.ToLower(key)
}

// NormalizeTitle lowercases, drops punctuation, and collapses
// whitespace so that "Agents Take Over DevTools!" and
// "agents take over devtools?" land on the same cluster key.
func NormalizeTitle(title string) string {
	var b strings.Builder
	lastSpace := true
	for _, r := range title {
		switch {
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			b.WriteRune(unicode.ToLower(r))
			lastSpace = false
		case !lastSpace:
			b.WriteByte(' ')
			lastSpace = true
		}
	}
	return strings.TrimSpace(b.String())
}

// Domain returns the registrable host (sans www.) for an item's URL.
func Domain(item trss.Item) string {
	key := CanonicalKey(item)
	if key == "" {
		return ""
	}
	parsed, err := url.Parse(key)
	if err != nil {
		return ""
	}
	return strings.TrimPrefix(parsed.Hostname(), "www.")
}

// RepoSignature extracts "owner/repo" for GitHub items so that
// commentary threads discussing the same repo cluster together.
// Returns "" when the item is not a GitHub link.
func RepoSignature(item trss.Item) string {
	key := CanonicalKey(item)
	if key == "" || !strings.Contains(key, "github.com") {
		return ""
	}
	parsed, err := url.Parse(key)
	if err != nil {
		return ""
	}
	parts := strings.Split(strings.Trim(parsed.Path, "/"), "/")
	if len(parts) < 2 {
		return ""
	}
	owner, repo := parts[0], parts[1]
	if owner == "" || repo == "" {
		return ""
	}
	return "github.com/" + owner + "/" + repo
}

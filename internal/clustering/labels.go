package clustering

import (
	"sort"
	"strings"

	"github.com/jcornudella/hotbrew/pkg/trss"
)

// Label identifies a deterministic theme label.
type Label struct {
	Slug     string
	Display  string
	keywords []string
}

// Known phase-1 labels. Keyword order does not matter; whichever label
// has the most keyword hits across a cluster wins. Ties break by the
// order of this slice, so earlier entries are preferred.
var labels = []Label{
	{Slug: "ai", Display: "AI", keywords: []string{
		"ai", "ml", "llm", "gpt", "claude", "anthropic", "opus", "sonnet",
		"haiku", "openai", "gemini", "mistral", "agent", "agents", "rag",
		"inference", "embedding", "prompt", "transformer", "fine tune",
	}},
	{Slug: "devtools", Display: "Devtools", keywords: []string{
		"ide", "editor", "vscode", "neovim", "vim", "debugger", "linter",
		"cli", "framework", "compiler", "typescript", "rustc", "build",
		"package manager", "runtime",
	}},
	{Slug: "infra", Display: "Infra", keywords: []string{
		"kubernetes", "k8s", "docker", "postgres", "postgresql", "mysql",
		"redis", "kafka", "sqlite", "terraform", "aws", "gcp", "azure",
		"cloudflare", "supabase", "vercel", "edge", "cdn", "serverless",
	}},
	{Slug: "startups", Display: "Startups", keywords: []string{
		"startup", "yc", "y combinator", "seed", "series a", "series b",
		"funding", "raises", "raised", "acquired", "acquires", "ipo",
		"valuation",
	}},
	{Slug: "papers", Display: "Papers", keywords: []string{
		"paper", "arxiv", "preprint", "study", "benchmark", "evaluation",
	}},
	{Slug: "deep-read", Display: "Deep Read", keywords: []string{
		"essay", "long read", "deep dive", "postmortem", "retrospective",
		"manifesto",
	}},
	{Slug: "debate", Display: "Debate", keywords: []string{
		"hot take", "controversial", "vs", "versus", "debate", "critique",
	}},
}

// KnownLabels returns the phase-1 keyword-based labels plus the
// source-type-derived "repo" shortcut. Callers (CLI listings,
// preference pickers) use this to show users the set of slugs they
// can follow or mute — keeping the source of truth in one place so
// adding a label here automatically flows through.
func KnownLabels() []Label {
	out := make([]Label, 0, len(labels)+2)
	out = append(out, labels...)
	out = append(out, Label{Slug: "repo", Display: "Repo of the Day"})
	out = append(out, Label{Slug: "general", Display: "General"})
	return out
}

// IsKnownLabel reports whether slug matches one of the canonical
// theme slugs. Used by CLI commands to reject typos before writing
// a preference row the ranker would silently ignore.
func IsKnownLabel(slug string) bool {
	for _, label := range KnownLabels() {
		if label.Slug == slug {
			return true
		}
	}
	return false
}

// LabelForItems picks the strongest theme label for a cluster by
// scanning title, summary, tags, and domain across its items. Returns
// ("general", "General") when no rule matches — keeping every cluster
// labeled so downstream surfaces never have to special-case "unlabeled".
func LabelForItems(items []trss.Item) Label {
	if len(items) == 0 {
		return Label{Slug: "general", Display: "General"}
	}

	if label, ok := labelFromSourceType(items); ok {
		return label
	}

	scores := make(map[string]int, len(labels))
	for _, item := range items {
		text := labelText(item)
		for _, label := range labels {
			scores[label.Slug] += countKeywords(text, label.keywords)
		}
	}

	best := ""
	bestScore := 0
	for _, label := range labels {
		if scores[label.Slug] > bestScore {
			best = label.Slug
			bestScore = scores[label.Slug]
		}
	}
	if best == "" {
		return Label{Slug: "general", Display: "General"}
	}
	for _, label := range labels {
		if label.Slug == best {
			return label
		}
	}
	return Label{Slug: "general", Display: "General"}
}

// labelFromSourceType handles the "repo of the day" shortcut: a cluster
// composed of GitHub-trending items always reads as a Repo cluster,
// regardless of title keywords.
func labelFromSourceType(items []trss.Item) (Label, bool) {
	for _, item := range items {
		via := strings.ToLower(item.Source.Via)
		if via != "github-trending" && !strings.Contains(via, "github") {
			return Label{}, false
		}
	}
	return Label{Slug: "repo", Display: "Repo of the Day"}, true
}

func labelText(item trss.Item) string {
	parts := []string{item.Title, item.Summary}
	parts = append(parts, item.Tags...)
	parts = append(parts, Domain(item))
	return strings.ToLower(strings.Join(parts, " \n "))
}

func countKeywords(text string, keywords []string) int {
	hits := 0
	for _, kw := range keywords {
		if kw == "" {
			continue
		}
		if containsKeyword(text, kw) {
			hits++
		}
	}
	return hits
}

// containsKeyword looks for kw in text. Multi-word keywords match as
// substrings (they're distinctive enough not to collide). Single-word
// keywords require a word boundary so short tokens like "ai" don't
// match inside unrelated words like "renaissance" or "mainline".
func containsKeyword(text, kw string) bool {
	if strings.Contains(kw, " ") {
		return strings.Contains(text, kw)
	}
	start := 0
	for {
		rel := strings.Index(text[start:], kw)
		if rel == -1 {
			return false
		}
		lo := start + rel
		hi := lo + len(kw)
		if !isWordChar(text, lo-1) && !isWordChar(text, hi) {
			return true
		}
		start = lo + 1
	}
}

func isWordChar(text string, i int) bool {
	if i < 0 || i >= len(text) {
		return false
	}
	c := text[i]
	return (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9')
}

// SortLabels returns labels sorted alphabetically by slug — used for
// stable test output and CLI listings.
func SortLabels(in []Label) []Label {
	out := append([]Label(nil), in...)
	sort.Slice(out, func(i, j int) bool { return out[i].Slug < out[j].Slug })
	return out
}

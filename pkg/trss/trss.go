// Package trss defines the Terminal RSS interchange format.
//
// TRSS is an open format for curated, programmable digests delivered
// to developer surfaces (terminal, logs, editor panes).
package trss

import "time"

// Item represents a normalized content item in TRSS format.
type Item struct {
	ID          string         `json:"id"`
	Title       string         `json:"title"`
	URL         string         `json:"url,omitempty"`
	URLCanonical string        `json:"url_canonical,omitempty"`
	Source      ItemSource     `json:"source"`
	PublishedAt time.Time      `json:"published_at"`
	FetchedAt   time.Time      `json:"fetched_at"`
	Summary     string         `json:"summary,omitempty"`
	Body        string         `json:"body,omitempty"`
	Tags        []string       `json:"tags,omitempty"`
	Score       float64        `json:"score"`
	Engagement  map[string]any `json:"engagement,omitempty"`
	Fingerprint string         `json:"fingerprint"`
	Meta        map[string]any `json:"meta,omitempty"`
}

// ItemSource identifies where an item came from.
type ItemSource struct {
	Name string `json:"name"`
	Icon string `json:"icon,omitempty"`
	Via  string `json:"via,omitempty"`
}

// Digest represents a curated collection of items.
type Digest struct {
	Type        string          `json:"type"`
	Version     string          `json:"version"`
	GeneratedAt time.Time       `json:"generated_at"`
	Title       string          `json:"title"`
	Window      string          `json:"window"`
	MaxItems    int             `json:"max_items"`
	ItemCount   int             `json:"item_count"`
	Items       []Item          `json:"items"`
	Sections    []DigestSection `json:"sections,omitempty"`
	Meta        DigestMeta      `json:"meta"`
}

// DigestSection groups items by topic or source.
type DigestSection struct {
	Name    string   `json:"name"`
	Icon    string   `json:"icon"`
	ItemIDs []string `json:"item_ids"`
}

// DigestMeta holds statistics about digest generation.
type DigestMeta struct {
	SourcesSynced   int `json:"sources_synced"`
	ItemsConsidered int `json:"items_considered"`
	ItemsDeduped    int `json:"items_deduped"`
	RulesApplied    int `json:"rules_applied"`
}

// NewDigest creates a new digest envelope.
func NewDigest(title, window string, maxItems int) *Digest {
	return &Digest{
		Type:        "trss-digest",
		Version:     "1",
		GeneratedAt: time.Now(),
		Title:       title,
		Window:      window,
		MaxItems:    maxItems,
	}
}

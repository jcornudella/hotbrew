package intel

import (
	"testing"
	"time"

	"github.com/jcornudella/hotbrew/pkg/source"
	"github.com/jcornudella/hotbrew/pkg/trss"
)

func TestFromTRSSItemPreservesCoreFields(t *testing.T) {
	publishedAt := time.Date(2026, 3, 9, 6, 0, 0, 0, time.UTC)
	fetchedAt := publishedAt.Add(15 * time.Minute)

	item := trss.Item{
		ID:           "item-1",
		Title:        "Karpathy launches autoresearch",
		URL:          "https://github.com/karpathy/autoresearch",
		URLCanonical: "https://github.com/karpathy/autoresearch",
		Source: trss.ItemSource{
			Name: "GitHub Trending",
			Icon: "🐙",
			Via:  "github-trending",
		},
		PublishedAt: publishedAt,
		FetchedAt:   fetchedAt,
		Summary:     "A repo for automated research workflows.",
		Body:        "Longer body",
		Tags:        []string{"ai", "agents"},
		Engagement: map[string]any{
			"stars":    1200,
			"forks":    90,
			"comments": 14,
		},
		Meta: map[string]any{
			"language": "Python",
			"author":   "karpathy",
		},
	}

	got := FromTRSSItem(item)

	if got.ID != item.ID {
		t.Fatalf("ID mismatch: got %q want %q", got.ID, item.ID)
	}
	if got.Title != item.Title {
		t.Fatalf("Title mismatch: got %q want %q", got.Title, item.Title)
	}
	if got.CanonicalURL != item.URLCanonical {
		t.Fatalf("CanonicalURL mismatch: got %q want %q", got.CanonicalURL, item.URLCanonical)
	}
	if got.SourceName != item.Source.Name {
		t.Fatalf("SourceName mismatch: got %q want %q", got.SourceName, item.Source.Name)
	}
	if got.SourceKey != item.Source.Via {
		t.Fatalf("SourceKey mismatch: got %q want %q", got.SourceKey, item.Source.Via)
	}
	if !got.PublishedAt.Equal(item.PublishedAt) {
		t.Fatalf("PublishedAt mismatch: got %v want %v", got.PublishedAt, item.PublishedAt)
	}
	if !got.FetchedAt.Equal(item.FetchedAt) {
		t.Fatalf("FetchedAt mismatch: got %v want %v", got.FetchedAt, item.FetchedAt)
	}
	if got.Engagement.Stars != 1200 {
		t.Fatalf("Stars mismatch: got %v want 1200", got.Engagement.Stars)
	}
	if got.Engagement.Forks != 90 {
		t.Fatalf("Forks mismatch: got %v want 90", got.Engagement.Forks)
	}
	if got.Engagement.Comments != 14 {
		t.Fatalf("Comments mismatch: got %v want 14", got.Engagement.Comments)
	}
	if got.Metadata["language"] != "Python" {
		t.Fatalf("Metadata language mismatch: got %q want %q", got.Metadata["language"], "Python")
	}
	if got.Author != "karpathy" {
		t.Fatalf("Author mismatch: got %q want %q", got.Author, "karpathy")
	}
}

func TestFromSourceItemPreservesCoreFields(t *testing.T) {
	publishedAt := time.Date(2026, 3, 9, 7, 0, 0, 0, time.UTC)

	item := source.Item{
		ID:        "hn-1",
		Title:     "Nvidia leaks next GPU architecture",
		Subtitle:  "520 points by pg",
		Body:      "Leaked details from a conference talk.",
		URL:       "https://example.com/gpu-leak",
		Timestamp: publishedAt,
		Category:  "hackernews",
		Metadata: map[string]any{
			"score":    520,
			"comments": 102,
			"author":   "pg",
		},
	}

	got := FromSourceItem(item, "Hacker News", "🔶")

	if got.ID != item.ID {
		t.Fatalf("ID mismatch: got %q want %q", got.ID, item.ID)
	}
	if got.Title != item.Title {
		t.Fatalf("Title mismatch: got %q want %q", got.Title, item.Title)
	}
	if got.Summary != item.Subtitle {
		t.Fatalf("Summary mismatch: got %q want %q", got.Summary, item.Subtitle)
	}
	if got.Body != item.Body {
		t.Fatalf("Body mismatch: got %q want %q", got.Body, item.Body)
	}
	if got.SourceName != "Hacker News" {
		t.Fatalf("SourceName mismatch: got %q want %q", got.SourceName, "Hacker News")
	}
	if got.SourceKey != "hackernews" {
		t.Fatalf("SourceKey mismatch: got %q want %q", got.SourceKey, "hackernews")
	}
	if got.CanonicalURL != item.URL {
		t.Fatalf("CanonicalURL mismatch: got %q want %q", got.CanonicalURL, item.URL)
	}
	if got.Engagement.Points != 520 {
		t.Fatalf("Points mismatch: got %v want 520", got.Engagement.Points)
	}
	if got.Engagement.Comments != 102 {
		t.Fatalf("Comments mismatch: got %v want 102", got.Engagement.Comments)
	}
	if got.Author != "pg" {
		t.Fatalf("Author mismatch: got %q want %q", got.Author, "pg")
	}
}

func TestTRSSRoundTripPreservesTitleURLAndSourceIdentity(t *testing.T) {
	original := trss.Item{
		ID:           "item-2",
		Title:        "Rust framework exploding on HN",
		URL:          "https://example.com/rust-framework",
		URLCanonical: "https://example.com/rust-framework",
		Source: trss.ItemSource{
			Name: "Lobste.rs",
			Icon: "🦞",
			Via:  "lobsters",
		},
		PublishedAt: time.Date(2026, 3, 9, 8, 0, 0, 0, time.UTC),
		FetchedAt:   time.Date(2026, 3, 9, 8, 10, 0, 0, time.UTC),
		Summary:     "A discussion about a fast-growing Rust web framework.",
		Tags:        []string{"rust", "web"},
	}

	roundTrip := ToTRSSItem(FromTRSSItem(original))

	if roundTrip.Title != original.Title {
		t.Fatalf("Title mismatch: got %q want %q", roundTrip.Title, original.Title)
	}
	if roundTrip.URL != original.URL {
		t.Fatalf("URL mismatch: got %q want %q", roundTrip.URL, original.URL)
	}
	if roundTrip.URLCanonical != original.URLCanonical {
		t.Fatalf("URLCanonical mismatch: got %q want %q", roundTrip.URLCanonical, original.URLCanonical)
	}
	if roundTrip.Source.Name != original.Source.Name {
		t.Fatalf("Source.Name mismatch: got %q want %q", roundTrip.Source.Name, original.Source.Name)
	}
	if roundTrip.Source.Via != original.Source.Via {
		t.Fatalf("Source.Via mismatch: got %q want %q", roundTrip.Source.Via, original.Source.Via)
	}
}

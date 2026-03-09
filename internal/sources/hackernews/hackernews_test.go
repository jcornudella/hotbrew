package hackernews

import (
	"context"
	"net/http"
	"testing"

	"github.com/jcornudella/hotbrew/pkg/source"
)

func TestFetchNormalizesTopStoriesFromFixtures(t *testing.T) {
	withFixtureTransport(t, map[string]string{
		topStories: "testdata/topstories.json",
		"https://hacker-news.firebaseio.com/v0/item/101.json": "testdata/item_101.json",
		"https://hacker-news.firebaseio.com/v0/item/102.json": "testdata/item_102.json",
	})

	section, err := New().Fetch(context.Background(), source.Config{
		Enabled:  true,
		Settings: map[string]any{"max": 2},
	})
	if err != nil {
		t.Fatalf("Fetch: %v", err)
	}

	if section == nil {
		t.Fatal("expected section, got nil")
	}
	if section.Name != "Hacker News" {
		t.Fatalf("section name mismatch: got %q", section.Name)
	}
	if len(section.Items) != 2 {
		t.Fatalf("item count mismatch: got %d want 2", len(section.Items))
	}
	if section.Items[0].ID != "hn-101" {
		t.Fatalf("first item id mismatch: got %q", section.Items[0].ID)
	}
	if section.Items[0].Metadata["hn_url"] != "https://news.ycombinator.com/item?id=101" {
		t.Fatalf("hn_url mismatch: got %v", section.Items[0].Metadata["hn_url"])
	}
	if section.Items[1].URL != "https://news.ycombinator.com/item?id=102" {
		t.Fatalf("expected fallback comments url, got %q", section.Items[1].URL)
	}
}

func withFixtureTransport(t *testing.T, fixtures map[string]string) {
	t.Helper()
	original := http.DefaultClient.Transport
	http.DefaultClient.Transport = &fixtureRoundTripper{t: t, fixtures: fixtures}
	t.Cleanup(func() {
		http.DefaultClient.Transport = original
	})
}

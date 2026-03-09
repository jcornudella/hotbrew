package github

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/jcornudella/hotbrew/pkg/source"
)

func TestFetchNormalizesTrendingReposFromFixtures(t *testing.T) {
	withFixtureTransport(t, map[string]string{
		searchURL: "testdata/search_repositories.json",
	})

	section, err := New("GitHub Trending", []string{"ai", "llm"}, "🐙").Fetch(context.Background(), source.Config{
		Enabled:  true,
		Settings: map[string]any{"max": 2},
	})
	if err != nil {
		t.Fatalf("Fetch: %v", err)
	}

	if section == nil {
		t.Fatal("expected section, got nil")
	}
	if section.Name != "GitHub Trending" {
		t.Fatalf("section name mismatch: got %q", section.Name)
	}
	if len(section.Items) != 2 {
		t.Fatalf("item count mismatch: got %d want 2", len(section.Items))
	}
	if section.Items[0].ID != "gh-2001" {
		t.Fatalf("first item id mismatch: got %q", section.Items[0].ID)
	}
	if !strings.Contains(section.Items[0].Subtitle, "1.5k") {
		t.Fatalf("expected star count in subtitle, got %q", section.Items[0].Subtitle)
	}
	if section.Items[0].Icon != "🐹" {
		t.Fatalf("expected Go icon, got %q", section.Items[0].Icon)
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

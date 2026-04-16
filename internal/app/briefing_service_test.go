package app

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/jcornudella/hotbrew/internal/config"
	"github.com/jcornudella/hotbrew/internal/store"
	"github.com/jcornudella/hotbrew/pkg/trss"
)

func TestBriefingServiceBuildReturnsBriefingFromPopulatedStore(t *testing.T) {
	st := openTestStore(t)
	cfg := config.Default()
	cfg.DigestWindow = "24h"
	cfg.DigestMax = 10

	sourceID, err := st.GetOrCreateSource("Hacker News", "hackernews", "", "🔶")
	if err != nil {
		t.Fatalf("GetOrCreateSource: %v", err)
	}

	now := time.Now().UTC()
	items := []trss.Item{
		{
			ID:           "hn-1",
			Fingerprint:  "fp-1",
			Title:        "Agent Safehouse",
			URL:          "https://example.com/agent-safehouse",
			URLCanonical: "https://example.com/agent-safehouse",
			Source:       trss.ItemSource{Name: "Hacker News", Icon: "🔶", Via: "hackernews"},
			PublishedAt:  now.Add(-1 * time.Hour),
			FetchedAt:    now,
			Summary:      "A sandbox for local agents.",
			Tags:         []string{"ai", "agents"},
		},
		{
			ID:           "hn-2",
			Fingerprint:  "fp-2",
			Title:        "Compiler renaissance",
			URL:          "https://example.com/compiler-essay",
			URLCanonical: "https://example.com/compiler-essay",
			Source:       trss.ItemSource{Name: "Hacker News", Icon: "🔶", Via: "hackernews"},
			PublishedAt:  now.Add(-2 * time.Hour),
			FetchedAt:    now,
			Summary:      "Why compilers will matter again.",
			Tags:         []string{"compilers"},
		},
	}

	for _, item := range items {
		if err := st.InsertItem(item, sourceID); err != nil {
			t.Fatalf("InsertItem(%s): %v", item.ID, err)
		}
	}

	svc := NewBriefingService(st, cfg)
	briefing, err := svc.Build(context.Background(), BuildOptions{})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}

	if briefing == nil {
		t.Fatal("Build returned nil briefing")
	}
	if briefing.Title != "Hotbrew Digest" {
		t.Fatalf("Title mismatch: got %q want %q", briefing.Title, "Hotbrew Digest")
	}
	if len(briefing.Items) != 2 {
		t.Fatalf("Item count mismatch: got %d want 2", len(briefing.Items))
	}
	if len(briefing.Sections) != 1 {
		t.Fatalf("Section count mismatch: got %d want 1", len(briefing.Sections))
	}
	if briefing.Sections[0].Name != "Hacker News" {
		t.Fatalf("Section name mismatch: got %q want %q", briefing.Sections[0].Name, "Hacker News")
	}
	if briefing.Meta.SourcesSynced != 1 {
		t.Fatalf("SourcesSynced mismatch: got %d want 1", briefing.Meta.SourcesSynced)
	}
}

func TestBriefingServiceBuildHandlesEmptyStoreGracefully(t *testing.T) {
	st := openTestStore(t)
	cfg := config.Default()

	svc := NewBriefingService(st, cfg)
	briefing, err := svc.Build(context.Background(), BuildOptions{})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}

	if briefing == nil {
		t.Fatal("Build returned nil briefing")
	}
	if len(briefing.Items) != 0 {
		t.Fatalf("expected 0 items, got %d", len(briefing.Items))
	}
	if len(briefing.Sections) != 0 {
		t.Fatalf("expected 0 sections, got %d", len(briefing.Sections))
	}
}

func TestBriefingServiceBuildSyncsWhenStoreEmptyAndSyncRequested(t *testing.T) {
	st := openTestStore(t)
	cfg := config.Default()
	cfg.DigestWindow = "24h"
	cfg.DigestMax = 10

	svc := NewBriefingService(st, cfg)
	svc.syncFunc = func(ctx context.Context) error {
		_ = ctx
		sourceID, err := st.GetOrCreateSource("Hacker News", "hackernews", "", "🔶")
		if err != nil {
			return err
		}
		now := time.Now().UTC()
		return st.InsertItem(trss.Item{
			ID:           "synced-1",
			Fingerprint:  "synced-fp-1",
			Title:        "Freshly synced item",
			URL:          "https://example.com/fresh",
			URLCanonical: "https://example.com/fresh",
			Source:       trss.ItemSource{Name: "Hacker News", Icon: "🔶", Via: "hackernews"},
			PublishedAt:  now,
			FetchedAt:    now,
			Summary:      "Loaded via sync",
		}, sourceID)
	}

	briefing, err := svc.Build(context.Background(), BuildOptions{SyncIfEmpty: true})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}

	if briefing == nil {
		t.Fatal("Build returned nil briefing")
	}
	if len(briefing.Items) != 1 {
		t.Fatalf("expected 1 item after sync, got %d", len(briefing.Items))
	}
	if briefing.Items[0].Item.Title != "Freshly synced item" {
		t.Fatalf("unexpected item title: got %q", briefing.Items[0].Item.Title)
	}
}

func TestBriefingServiceBuildPersistsItemFeatures(t *testing.T) {
	st := openTestStore(t)
	cfg := config.Default()
	cfg.DigestWindow = "24h"
	cfg.DigestMax = 10

	sourceID, err := st.GetOrCreateSource("Hacker News", "hackernews", "", "🔶")
	if err != nil {
		t.Fatalf("GetOrCreateSource: %v", err)
	}
	now := time.Now().UTC()
	item := trss.Item{
		ID:          "hn-features",
		Fingerprint: "fp-features",
		Title:       "With engagement",
		URL:         "https://example.com/features",
		Source:      trss.ItemSource{Name: "Hacker News", Icon: "🔶", Via: "hackernews"},
		PublishedAt: now.Add(-30 * time.Minute),
		FetchedAt:   now,
		Engagement:  map[string]any{"points": 300.0, "comments": 50.0},
	}
	if err := st.InsertItem(item, sourceID); err != nil {
		t.Fatalf("InsertItem: %v", err)
	}

	svc := NewBriefingService(st, cfg)
	if _, err := svc.Build(context.Background(), BuildOptions{}); err != nil {
		t.Fatalf("Build: %v", err)
	}

	features, err := st.GetItemFeatures("hn-features")
	if err != nil {
		t.Fatalf("GetItemFeatures: %v", err)
	}
	if features == nil {
		t.Fatal("expected features to be persisted")
	}
	if features.Signals.Freshness <= 0 {
		t.Fatalf("expected positive freshness, got %f", features.Signals.Freshness)
	}
	if features.Signals.Engagement <= 0 {
		t.Fatalf("expected positive engagement, got %f", features.Signals.Engagement)
	}
	if features.Signals.SourceAuthority == 0 {
		t.Fatalf("expected non-zero authority, got %f", features.Signals.SourceAuthority)
	}
}

func TestBriefingServiceBuildPopulatesAndPersistsClusters(t *testing.T) {
	st := openTestStore(t)
	cfg := config.Default()
	cfg.DigestWindow = "24h"
	cfg.DigestMax = 10

	sourceID, err := st.GetOrCreateSource("Hacker News", "hackernews", "", "🔶")
	if err != nil {
		t.Fatalf("GetOrCreateSource: %v", err)
	}
	tldrID, err := st.GetOrCreateSource("TLDR", "tldr", "", "📰")
	if err != nil {
		t.Fatalf("GetOrCreateSource TLDR: %v", err)
	}

	now := time.Now().UTC()
	// Items survive dedup (different titles + URLs) but share a GitHub
	// repo signature, so clustering should fold them together via repo.
	items := []struct {
		sourceID int
		item     trss.Item
	}{
		{sourceID, trss.Item{ID: "gh-repo", Fingerprint: "fp-repo", Title: "karpathy/autoresearch on GitHub", URL: "https://github.com/karpathy/autoresearch", URLCanonical: "https://github.com/karpathy/autoresearch", Source: trss.ItemSource{Name: "GitHub Trending", Via: "github-trending"}, PublishedAt: now.Add(-1 * time.Hour), FetchedAt: now}},
		{tldrID, trss.Item{ID: "hn-discuss", Fingerprint: "fp-discuss", Title: "Discussion on Karpathy's new literature review tool", URL: "https://github.com/karpathy/autoresearch/issues/12", URLCanonical: "https://github.com/karpathy/autoresearch/issues/12", Source: trss.ItemSource{Name: "TLDR", Via: "tldr"}, PublishedAt: now.Add(-2 * time.Hour), FetchedAt: now}},
		{sourceID, trss.Item{ID: "hn-solo", Fingerprint: "fp-solo", Title: "An essay about something else", URL: "https://other.example/essay", URLCanonical: "https://other.example/essay", Source: trss.ItemSource{Name: "Hacker News", Via: "hackernews"}, PublishedAt: now.Add(-3 * time.Hour), FetchedAt: now}},
	}
	for _, entry := range items {
		if err := st.InsertItem(entry.item, entry.sourceID); err != nil {
			t.Fatalf("InsertItem(%s): %v", entry.item.ID, err)
		}
	}

	svc := NewBriefingService(st, cfg)
	briefing, err := svc.Build(context.Background(), BuildOptions{})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}

	if len(briefing.Clusters) == 0 {
		t.Fatal("briefing should carry clusters")
	}
	sharedFound := false
	for _, c := range briefing.Clusters {
		if len(c.ItemIDs) == 2 {
			sharedFound = true
		}
	}
	if !sharedFound {
		t.Fatalf("expected a 2-member cluster for shared URL, got %+v", briefing.Clusters)
	}

	persisted, err := st.ListClusters()
	if err != nil {
		t.Fatalf("ListClusters: %v", err)
	}
	if len(persisted) != len(briefing.Clusters) {
		t.Fatalf("persisted cluster count mismatch: %d vs %d", len(persisted), len(briefing.Clusters))
	}
}

func openTestStore(t *testing.T) *store.Store {
	t.Helper()

	dbPath := filepath.Join(t.TempDir(), "hotbrew-test.db")
	st, err := store.Open(dbPath)
	if err != nil {
		t.Fatalf("store.Open: %v", err)
	}
	t.Cleanup(func() {
		_ = st.Close()
	})
	return st
}

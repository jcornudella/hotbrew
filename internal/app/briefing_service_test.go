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

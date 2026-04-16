package store

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/jcornudella/hotbrew/internal/intel"
	"github.com/jcornudella/hotbrew/pkg/trss"
)

func TestUpsertItemFeaturesRoundTrips(t *testing.T) {
	st := openTestStore(t)
	itemID := seedItem(t, st, "item-1")

	signals := intel.ItemSignals{
		Freshness:       0.91,
		SourceAuthority: 1.25,
		Engagement:      0.77,
		TopicMatch:      2.0,
	}
	if err := st.UpsertItemFeatures(itemID, signals); err != nil {
		t.Fatalf("UpsertItemFeatures: %v", err)
	}

	got, err := st.GetItemFeatures(itemID)
	if err != nil {
		t.Fatalf("GetItemFeatures: %v", err)
	}
	if got == nil {
		t.Fatal("GetItemFeatures returned nil")
	}
	if got.Signals != signals {
		t.Fatalf("signals mismatch: got %+v want %+v", got.Signals, signals)
	}
	if got.ComputedAt.IsZero() {
		t.Fatal("ComputedAt should be set")
	}
}

func TestUpsertItemFeaturesOverwritesExistingRow(t *testing.T) {
	st := openTestStore(t)
	itemID := seedItem(t, st, "item-2")

	first := intel.ItemSignals{Freshness: 0.5, SourceAuthority: 1.0, Engagement: 0.5}
	second := intel.ItemSignals{Freshness: 0.8, SourceAuthority: 1.5, Engagement: 1.2}

	if err := st.UpsertItemFeatures(itemID, first); err != nil {
		t.Fatalf("upsert first: %v", err)
	}
	if err := st.UpsertItemFeatures(itemID, second); err != nil {
		t.Fatalf("upsert second: %v", err)
	}

	got, err := st.GetItemFeatures(itemID)
	if err != nil {
		t.Fatalf("GetItemFeatures: %v", err)
	}
	if got.Signals != second {
		t.Fatalf("expected overwrite with %+v, got %+v", second, got.Signals)
	}
}

func TestListItemFeaturesReturnsOnlyKnownIDs(t *testing.T) {
	st := openTestStore(t)
	idA := seedItem(t, st, "item-a")
	idB := seedItem(t, st, "item-b")
	_ = seedItem(t, st, "item-c")

	if err := st.UpsertItemFeatures(idA, intel.ItemSignals{Freshness: 0.4}); err != nil {
		t.Fatalf("upsert a: %v", err)
	}
	if err := st.UpsertItemFeatures(idB, intel.ItemSignals{Freshness: 0.6}); err != nil {
		t.Fatalf("upsert b: %v", err)
	}

	result, err := st.ListItemFeatures([]string{idA, idB, "item-c", "missing"})
	if err != nil {
		t.Fatalf("ListItemFeatures: %v", err)
	}
	if len(result) != 2 {
		t.Fatalf("expected 2 results, got %d", len(result))
	}
	if _, ok := result["missing"]; ok {
		t.Fatal("unexpected entry for missing id")
	}
	if result[idA].Signals.Freshness != 0.4 {
		t.Fatalf("idA freshness mismatch: %+v", result[idA])
	}
	if result[idB].Signals.Freshness != 0.6 {
		t.Fatalf("idB freshness mismatch: %+v", result[idB])
	}
}

func TestGetItemFeaturesReturnsErrorWhenAbsent(t *testing.T) {
	st := openTestStore(t)
	if _, err := st.GetItemFeatures("nonexistent"); err == nil {
		t.Fatal("expected error for missing item")
	}
}

func openTestStore(t *testing.T) *Store {
	t.Helper()
	st, err := Open(filepath.Join(t.TempDir(), "features-test.db"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })
	return st
}

func seedItem(t *testing.T, st *Store, id string) string {
	t.Helper()
	sourceID, err := st.GetOrCreateSource("Hacker News", "hackernews", "", "🔶")
	if err != nil {
		t.Fatalf("GetOrCreateSource: %v", err)
	}
	now := time.Now().UTC()
	item := trss.Item{
		ID:          id,
		Fingerprint: "fp-" + id,
		Title:       "seed " + id,
		URL:         "https://example.com/" + id,
		Source:      trss.ItemSource{Name: "Hacker News", Via: "hackernews"},
		PublishedAt: now,
		FetchedAt:   now,
	}
	if err := st.InsertItem(item, sourceID); err != nil {
		t.Fatalf("InsertItem: %v", err)
	}
	return id
}

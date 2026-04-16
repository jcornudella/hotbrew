package store

import (
	"testing"

	"github.com/jcornudella/hotbrew/internal/intel"
)

func TestReplaceClustersRoundTripsShapeAndMembership(t *testing.T) {
	st := openTestStore(t)
	seedItem(t, st, "item-a")
	seedItem(t, st, "item-b")
	seedItem(t, st, "item-c")

	clusters := []intel.ThemeCluster{
		{
			ID:             "c_ai_1",
			Label:          "AI",
			Slug:           "ai",
			Representative: "item-a",
			ItemIDs:        []string{"item-a", "item-b"},
			Score:          1.7,
			WhyItMatters:   "Multiple sources reporting an Opus launch.",
		},
		{
			ID:             "c_infra_1",
			Label:          "Infra",
			Slug:           "infra",
			Representative: "item-c",
			ItemIDs:        []string{"item-c"},
			Score:          1.1,
		},
	}

	if err := st.ReplaceClusters(clusters); err != nil {
		t.Fatalf("ReplaceClusters: %v", err)
	}

	got, err := st.ListClusters()
	if err != nil {
		t.Fatalf("ListClusters: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 clusters, got %d", len(got))
	}
	if got[0].ID != "c_ai_1" {
		t.Fatalf("expected AI first by score, got %q", got[0].ID)
	}
	if len(got[0].ItemIDs) != 2 {
		t.Fatalf("expected 2 members on AI cluster, got %v", got[0].ItemIDs)
	}
	if got[0].WhyItMatters == "" {
		t.Fatal("why_it_matters should round-trip")
	}
}

func TestReplaceClustersIsIdempotentAndWipesPrior(t *testing.T) {
	st := openTestStore(t)
	seedItem(t, st, "x")
	seedItem(t, st, "y")

	first := []intel.ThemeCluster{{ID: "c1", Label: "AI", Slug: "ai", ItemIDs: []string{"x"}, Score: 0.5}}
	second := []intel.ThemeCluster{{ID: "c2", Label: "Infra", Slug: "infra", ItemIDs: []string{"y"}, Score: 0.9}}

	if err := st.ReplaceClusters(first); err != nil {
		t.Fatalf("first replace: %v", err)
	}
	if err := st.ReplaceClusters(second); err != nil {
		t.Fatalf("second replace: %v", err)
	}

	got, err := st.ListClusters()
	if err != nil {
		t.Fatalf("ListClusters: %v", err)
	}
	if len(got) != 1 || got[0].ID != "c2" {
		t.Fatalf("expected second generation only, got %+v", got)
	}
}

func TestListClustersEmptyStoreReturnsNoRows(t *testing.T) {
	st := openTestStore(t)
	got, err := st.ListClusters()
	if err != nil {
		t.Fatalf("ListClusters: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("expected empty, got %+v", got)
	}
}

package briefing

import (
	"reflect"
	"testing"

	"github.com/jcornudella/hotbrew/internal/intel"
)

func TestAssembleGroupsClustersByCanonicalSectionOrder(t *testing.T) {
	b := &intel.Briefing{
		Clusters: []intel.ThemeCluster{
			{ID: "c_debate", Slug: "debate", Label: "Debate", Score: 2.0},
			{ID: "c_ai", Slug: "ai", Label: "AI", Score: 1.0},
			{ID: "c_infra", Slug: "infra", Label: "Infra", Score: 3.0},
		},
	}

	Assemble(b)

	if len(b.Sections) != 3 {
		t.Fatalf("want 3 sections, got %d: %+v", len(b.Sections), b.Sections)
	}
	gotKinds := []string{b.Sections[0].Kind, b.Sections[1].Kind, b.Sections[2].Kind}
	wantKinds := []string{"ai", "infra", "debate"}
	if !reflect.DeepEqual(gotKinds, wantKinds) {
		t.Fatalf("section order: got %v want %v", gotKinds, wantKinds)
	}
	if b.Sections[0].Name != "AI" || b.Sections[2].Name != "Debate" {
		t.Fatalf("display name mismatch: %+v", b.Sections)
	}
}

func TestAssembleGroupsMultipleClustersUnderSameSectionByScore(t *testing.T) {
	b := &intel.Briefing{
		Clusters: []intel.ThemeCluster{
			{ID: "c_ai_low", Slug: "ai", Score: 1.0},
			{ID: "c_ai_high", Slug: "ai", Score: 5.0},
			{ID: "c_ai_mid", Slug: "ai", Score: 3.0},
		},
	}

	Assemble(b)

	if len(b.Sections) != 1 {
		t.Fatalf("want 1 section, got %d", len(b.Sections))
	}
	gotIDs := b.Sections[0].ClusterIDs
	wantIDs := []string{"c_ai_high", "c_ai_mid", "c_ai_low"}
	if !reflect.DeepEqual(gotIDs, wantIDs) {
		t.Fatalf("cluster order: got %v want %v", gotIDs, wantIDs)
	}
	// Briefing.Clusters should be reshuffled to match section order.
	gotClusterOrder := []string{b.Clusters[0].ID, b.Clusters[1].ID, b.Clusters[2].ID}
	if !reflect.DeepEqual(gotClusterOrder, wantIDs) {
		t.Fatalf("clusters slice not realigned: got %v want %v", gotClusterOrder, wantIDs)
	}
}

func TestAssembleRoutesUnknownSlugsToTailAlphabetically(t *testing.T) {
	b := &intel.Briefing{
		Clusters: []intel.ThemeCluster{
			{ID: "c_ai", Slug: "ai", Score: 1.0},
			{ID: "c_zebra", Slug: "zebra", Score: 9.0},
			{ID: "c_apex", Slug: "apex", Score: 9.0},
		},
	}

	Assemble(b)

	kinds := make([]string, len(b.Sections))
	for i, s := range b.Sections {
		kinds[i] = s.Kind
	}
	want := []string{"ai", "apex", "zebra"}
	if !reflect.DeepEqual(kinds, want) {
		t.Fatalf("unknown slugs should sort to tail alphabetically: got %v want %v", kinds, want)
	}
}

func TestAssembleDefaultsEmptySlugToGeneral(t *testing.T) {
	b := &intel.Briefing{
		Clusters: []intel.ThemeCluster{
			{ID: "c_blank", Slug: "", Score: 1.0},
		},
	}

	Assemble(b)

	if len(b.Sections) != 1 || b.Sections[0].Kind != "general" {
		t.Fatalf("blank slug should become general, got %+v", b.Sections)
	}
	if b.Sections[0].Name != "General" {
		t.Fatalf("display name should be General, got %q", b.Sections[0].Name)
	}
}

func TestAssembleOnEmptyBriefingClearsSections(t *testing.T) {
	b := &intel.Briefing{
		Sections: []intel.BriefingSection{{Kind: "stale"}},
	}

	Assemble(b)

	if len(b.Sections) != 0 {
		t.Fatalf("empty clusters should yield no sections, got %+v", b.Sections)
	}
}

func TestSupportingItemsExcludesRepAndSortsByScore(t *testing.T) {
	cluster := intel.ThemeCluster{
		Representative: "b",
		ItemIDs:        []string{"a", "b", "c", "d"},
	}
	scores := map[string]float64{"a": 2.0, "b": 5.0, "c": 1.0, "d": 3.0}

	got := SupportingItems(cluster, scores)
	want := []string{"d", "a", "c"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("supporting order: got %v want %v", got, want)
	}
}

func TestSupportingItemsBreaksTiesByID(t *testing.T) {
	cluster := intel.ThemeCluster{
		Representative: "rep",
		ItemIDs:        []string{"rep", "zeta", "alpha", "middle"},
	}
	scores := map[string]float64{"rep": 9, "zeta": 1, "alpha": 1, "middle": 1}

	got := SupportingItems(cluster, scores)
	want := []string{"alpha", "middle", "zeta"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("tie-break order: got %v want %v", got, want)
	}
}

func TestSupportingItemsReturnsNilForSingletonCluster(t *testing.T) {
	cluster := intel.ThemeCluster{Representative: "only", ItemIDs: []string{"only"}}
	if got := SupportingItems(cluster, nil); got != nil {
		t.Fatalf("singleton cluster should have no supporting items, got %v", got)
	}
}

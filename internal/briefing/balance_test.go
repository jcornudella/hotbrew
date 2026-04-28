package briefing

import (
	"fmt"
	"testing"

	"github.com/jcornudella/hotbrew/internal/intel"
)

// Helpers — keep the fixtures declarative so each scenario reads as a
// short story. Every cluster is built with a single item and its own
// domain, since Balance only consults the representative's URL.

func scored(id, url string) intel.ScoredItem {
	return intel.ScoredItem{
		Item: intel.IntelItem{ID: id, CanonicalURL: url},
	}
}

func cluster(id, slug string, score float64, repID string) intel.ThemeCluster {
	return intel.ThemeCluster{
		ID:             id,
		Slug:           slug,
		Label:          slug,
		Score:          score,
		Representative: repID,
		ItemIDs:        []string{repID},
	}
}

func TestBalanceTrimsRepoHeavyDay(t *testing.T) {
	// Eight repo clusters from five different domains — the theme
	// cap should knock it down to three.
	b := &intel.Briefing{}
	for i := 0; i < 8; i++ {
		id := fmt.Sprintf("repo-%d", i)
		// Distinct hosts so the theme cap (not the domain cap) is the
		// rule under test here.
		url := fmt.Sprintf("https://host%d.dev/project", i)
		b.Items = append(b.Items, scored(id, url))
		b.Clusters = append(b.Clusters, cluster("c_"+id, "repo", float64(8-i), id))
	}
	Assemble(b)

	limits := DefaultBalanceLimits()
	Balance(b, limits)

	count := 0
	for _, c := range b.Clusters {
		if c.Slug == "repo" {
			count++
		}
	}
	if count != limits.MaxClustersPerTheme {
		t.Fatalf("repo clusters after balance = %d, want %d", count, limits.MaxClustersPerTheme)
	}
}

func TestBalanceTrimsPaperHeavyDay(t *testing.T) {
	b := &intel.Briefing{}
	for i := 0; i < 6; i++ {
		id := fmt.Sprintf("paper-%d", i)
		// Distinct journals per cluster so the theme cap is what bites.
		url := fmt.Sprintf("https://journal%d.science/abs", i)
		b.Items = append(b.Items, scored(id, url))
		b.Clusters = append(b.Clusters, cluster("c_"+id, "papers", float64(6-i), id))
	}
	// Seed a non-papers cluster so we verify Balance doesn't bleed
	// across themes.
	b.Items = append(b.Items, scored("ai-1", "https://news.ycombinator.com/item?id=1"))
	b.Clusters = append(b.Clusters, cluster("c_ai", "ai", 10.0, "ai-1"))
	Assemble(b)

	// Pin limits explicitly so the test describes behavior at a known
	// per-theme cap, independent of the package default.
	Balance(b, BalanceLimits{
		MaxClustersPerTheme: 3,
		MaxLeadsPerDomain:   2,
		MaxTotalClusters:    10,
	})

	papers := 0
	ai := 0
	for _, c := range b.Clusters {
		switch c.Slug {
		case "papers":
			papers++
		case "ai":
			ai++
		}
	}
	if papers != 3 {
		t.Errorf("papers kept = %d, want 3", papers)
	}
	if ai != 1 {
		t.Errorf("ai kept = %d, want 1", ai)
	}
}

func TestBalanceCapsHNDominatedDay(t *testing.T) {
	// Five ai clusters all leading with an HN domain — domain cap
	// should cut them to two regardless of theme cap, because each
	// cluster is a different theme wouldn't apply; here we also test
	// domain cap by making them all ai. Theme cap (3) still bites,
	// but domain cap (2) bites earlier.
	b := &intel.Briefing{}
	for i := 0; i < 5; i++ {
		id := fmt.Sprintf("hn-%d", i)
		url := fmt.Sprintf("https://news.ycombinator.com/item?id=%d", i)
		b.Items = append(b.Items, scored(id, url))
		b.Clusters = append(b.Clusters, cluster("c_"+id, "ai", float64(5-i), id))
	}
	Assemble(b)

	Balance(b, BalanceLimits{
		MaxClustersPerTheme: 3,
		MaxLeadsPerDomain:   2,
		MaxTotalClusters:    10,
	})

	domainLeads := 0
	for _, c := range b.Clusters {
		lead := b.Items[clusterItemIndex(b, c.Representative)].Item
		if lead.CanonicalURL != "" {
			domainLeads++
		}
	}
	if domainLeads > 2 {
		t.Errorf("expected at most 2 HN clusters, got %d", domainLeads)
	}
}

func TestBalanceLowSignalDayLeavesSmallInputAlone(t *testing.T) {
	// Three clusters, spread across themes, one domain each — nothing
	// should trip caps, and nothing should be dropped.
	b := &intel.Briefing{}
	b.Items = append(b.Items,
		scored("a", "https://example.com/a"),
		scored("b", "https://other.org/b"),
		scored("c", "https://third.io/c"),
	)
	b.Clusters = append(b.Clusters,
		cluster("c_a", "ai", 3.0, "a"),
		cluster("c_b", "devtools", 2.0, "b"),
		cluster("c_c", "infra", 1.0, "c"),
	)
	Assemble(b)
	before := len(b.Clusters)

	Balance(b, DefaultBalanceLimits())

	if len(b.Clusters) != before {
		t.Errorf("low-signal day lost clusters: before=%d after=%d", before, len(b.Clusters))
	}
}

func TestBalanceEnsuresDeepReadSurvivesTrim(t *testing.T) {
	// Pack the top of the briefing with ai clusters that would fill
	// the total cap; the single deep-read cluster scores last but
	// should still make it because EnsureDeepRead is on.
	b := &intel.Briefing{}
	for i := 0; i < 10; i++ {
		id := fmt.Sprintf("ai-%d", i)
		url := fmt.Sprintf("https://site%d.com/", i)
		b.Items = append(b.Items, scored(id, url))
		b.Clusters = append(b.Clusters, cluster("c_"+id, "ai", float64(20-i), id))
	}
	b.Items = append(b.Items, scored("dr-1", "https://long.blog/post"))
	b.Clusters = append(b.Clusters, cluster("c_dr", "deep-read", 0.1, "dr-1"))
	Assemble(b)

	Balance(b, BalanceLimits{
		MaxTotalClusters: 4,
		EnsureDeepRead:   true,
	})

	found := false
	for _, c := range b.Clusters {
		if c.Slug == "deep-read" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("deep-read cluster was dropped despite EnsureDeepRead")
	}
}

func TestBalanceEnsuresRepoSurvivesTrim(t *testing.T) {
	b := &intel.Briefing{}
	for i := 0; i < 6; i++ {
		id := fmt.Sprintf("ai-%d", i)
		b.Items = append(b.Items, scored(id, fmt.Sprintf("https://s%d.com/", i)))
		b.Clusters = append(b.Clusters, cluster("c_"+id, "ai", float64(10-i), id))
	}
	b.Items = append(b.Items, scored("repo-1", "https://github.com/foo/bar"))
	b.Clusters = append(b.Clusters, cluster("c_repo", "repo", 0.1, "repo-1"))
	Assemble(b)

	Balance(b, BalanceLimits{
		MaxTotalClusters: 3,
		EnsureRepo:       true,
	})

	found := false
	for _, c := range b.Clusters {
		if c.Slug == "repo" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("repo cluster was dropped despite EnsureRepo")
	}
}

func TestBalanceIsDeterministic(t *testing.T) {
	build := func() *intel.Briefing {
		b := &intel.Briefing{}
		for i := 0; i < 6; i++ {
			id := fmt.Sprintf("r-%d", i)
			b.Items = append(b.Items, scored(id, fmt.Sprintf("https://github.com/o%d/p", i)))
			b.Clusters = append(b.Clusters, cluster("c_"+id, "repo", float64(6-i), id))
		}
		Assemble(b)
		return b
	}

	a := build()
	Balance(a, DefaultBalanceLimits())
	b := build()
	Balance(b, DefaultBalanceLimits())

	if len(a.Clusters) != len(b.Clusters) {
		t.Fatalf("non-deterministic: %d vs %d", len(a.Clusters), len(b.Clusters))
	}
	for i := range a.Clusters {
		if a.Clusters[i].ID != b.Clusters[i].ID {
			t.Errorf("cluster %d order drift: %s vs %s", i, a.Clusters[i].ID, b.Clusters[i].ID)
		}
	}
}

func TestBalanceRebuildsSectionsFromTrimmedClusters(t *testing.T) {
	b := &intel.Briefing{}
	for i := 0; i < 5; i++ {
		id := fmt.Sprintf("r-%d", i)
		b.Items = append(b.Items, scored(id, fmt.Sprintf("https://g%d.com/r", i)))
		b.Clusters = append(b.Clusters, cluster("c_"+id, "repo", float64(5-i), id))
	}
	Assemble(b)
	Balance(b, BalanceLimits{MaxClustersPerTheme: 2})

	// Exactly one section (repo), exactly two cluster IDs listed.
	if len(b.Sections) != 1 || b.Sections[0].Kind != "repo" {
		t.Fatalf("sections after balance: %+v", b.Sections)
	}
	if len(b.Sections[0].ClusterIDs) != len(b.Clusters) {
		t.Errorf("section ClusterIDs drifted from Clusters: %d vs %d",
			len(b.Sections[0].ClusterIDs), len(b.Clusters))
	}
}

// clusterItemIndex finds the position in b.Items of a representative
// id — only used for assertions; real code uses indexItemsByID.
func clusterItemIndex(b *intel.Briefing, id string) int {
	for i, it := range b.Items {
		if it.Item.ID == id {
			return i
		}
	}
	return -1
}

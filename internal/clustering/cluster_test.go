package clustering

import (
	"strings"
	"testing"

	"github.com/jcornudella/hotbrew/internal/intel"
	"github.com/jcornudella/hotbrew/pkg/trss"
)

func TestClusterGroupsItemsSharingCanonicalURL(t *testing.T) {
	items := []trss.Item{
		{ID: "hn-1", URLCanonical: "https://example.com/opus-launch", Title: "Anthropic launches Opus 4.8", Source: trss.ItemSource{Name: "HN"}},
		{ID: "tldr-1", URLCanonical: "https://example.com/opus-launch", Title: "Opus launch", Source: trss.ItemSource{Name: "TLDR"}},
		{ID: "solo", URLCanonical: "https://other.com/unrelated", Title: "Something completely different", Source: trss.ItemSource{Name: "HN"}},
	}

	clusters := Cluster(items)
	if len(clusters) != 2 {
		t.Fatalf("expected 2 clusters, got %d", len(clusters))
	}
	clusterOf := indexByItem(clusters)
	if clusterOf["hn-1"] != clusterOf["tldr-1"] {
		t.Fatalf("shared URL should cluster: %+v", clusterOf)
	}
	if clusterOf["solo"] == clusterOf["hn-1"] {
		t.Fatal("unrelated item should not join URL cluster")
	}
}

func TestClusterGroupsItemsSharingGitHubRepo(t *testing.T) {
	items := []trss.Item{
		{ID: "gh-1", URL: "https://github.com/anthropics/claude-cookbooks", Title: "claude-cookbooks", Source: trss.ItemSource{Name: "GitHub Trending", Via: "github-trending"}},
		{ID: "hn-1", URL: "https://github.com/anthropics/claude-cookbooks/pull/42", Title: "Discussion about claude cookbooks", Source: trss.ItemSource{Name: "HN"}},
	}

	clusters := Cluster(items)
	if len(clusters) != 1 {
		t.Fatalf("expected 1 cluster via shared repo, got %d", len(clusters))
	}
	if len(clusters[0].ItemIDs) != 2 {
		t.Fatalf("expected both items in cluster, got %v", clusters[0].ItemIDs)
	}
}

func TestClusterSelectsHighestScoringRepresentative(t *testing.T) {
	items := []trss.Item{
		{ID: "lo", URLCanonical: "https://example.com/story", Title: "Shared story across sources", Score: 0.2, Source: trss.ItemSource{Name: "HN"}},
		{ID: "hi", URLCanonical: "https://example.com/story", Title: "Shared story across sources", Score: 0.9, Source: trss.ItemSource{Name: "TLDR"}},
		{ID: "mid", URLCanonical: "https://example.com/story", Title: "Shared story across sources", Score: 0.5, Source: trss.ItemSource{Name: "Lobsters"}},
	}

	clusters := Cluster(items)
	if len(clusters) != 1 {
		t.Fatalf("expected 1 cluster, got %d", len(clusters))
	}
	if clusters[0].Representative != "hi" {
		t.Fatalf("expected representative hi, got %s", clusters[0].Representative)
	}
	if clusters[0].Score != 0.9 {
		t.Fatalf("expected cluster score 0.9, got %f", clusters[0].Score)
	}
}

func TestClusterIsDeterministicAcrossRuns(t *testing.T) {
	items := []trss.Item{
		{ID: "a", URLCanonical: "https://example.com/1", Title: "LLM agents take over the world today", Source: trss.ItemSource{Name: "HN"}, Score: 1.0},
		{ID: "b", URLCanonical: "https://example.com/2", Title: "LLM agents take over the world today", Source: trss.ItemSource{Name: "TLDR"}, Score: 1.2},
		{ID: "c", URLCanonical: "https://example.com/3", Title: "Different headline entirely here", Source: trss.ItemSource{Name: "Lobsters"}, Score: 0.8},
	}

	first := Cluster(items)
	second := Cluster(items)

	if len(first) != len(second) {
		t.Fatalf("cluster counts diverge: %d vs %d", len(first), len(second))
	}
	for i := range first {
		if first[i].ID != second[i].ID || strings.Join(first[i].ItemIDs, ",") != strings.Join(second[i].ItemIDs, ",") {
			t.Fatalf("cluster %d unstable between runs: %+v vs %+v", i, first[i], second[i])
		}
	}
}

func TestClusterSortsByScoreDescending(t *testing.T) {
	items := []trss.Item{
		{ID: "low", Title: "Solo low score", URL: "https://a.example/x", Score: 0.2, Source: trss.ItemSource{Name: "HN"}},
		{ID: "high", Title: "Solo high score", URL: "https://b.example/x", Score: 2.0, Source: trss.ItemSource{Name: "HN"}},
	}
	clusters := Cluster(items)
	if len(clusters) != 2 {
		t.Fatalf("expected 2 clusters, got %d", len(clusters))
	}
	if clusters[0].Representative != "high" {
		t.Fatalf("expected high first, got %s", clusters[0].Representative)
	}
}

func indexByItem(clusters []intel.ThemeCluster) map[string]string {
	index := map[string]string{}
	for _, c := range clusters {
		for _, id := range c.ItemIDs {
			index[id] = c.ID
		}
	}
	return index
}

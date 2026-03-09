package app

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jcornudella/hotbrew/internal/config"
	"github.com/jcornudella/hotbrew/internal/intel"
	"github.com/jcornudella/hotbrew/pkg/trss"
)

func TestBriefingBuildMatchesGolden(t *testing.T) {
	st := openTestStore(t)
	cfg := config.Default()
	cfg.DigestWindow = "24h"
	cfg.DigestMax = 10

	hnSourceID, err := st.GetOrCreateSource("Hacker News", "hackernews", "", "🔶")
	if err != nil {
		t.Fatalf("GetOrCreateSource HN: %v", err)
	}
	ghSourceID, err := st.GetOrCreateSource("GitHub Trending", "github-trending", "", "🐙")
	if err != nil {
		t.Fatalf("GetOrCreateSource GitHub: %v", err)
	}

	now := time.Date(2026, 3, 9, 8, 0, 0, 0, time.UTC)
	seed := []struct {
		sourceID int
		item     trss.Item
	}{
		{hnSourceID, trss.Item{ID: "hn-1", Fingerprint: "fp-hn-1", Title: "Agent Safehouse", URL: "https://example.com/agent-safehouse", URLCanonical: "https://example.com/agent-safehouse", Source: trss.ItemSource{Name: "Hacker News", Icon: "🔶", Via: "hackernews"}, PublishedAt: now.Add(-1 * time.Hour), FetchedAt: now, Summary: "Sandbox for local agents.", Engagement: map[string]any{"points": 457, "comments": 103}}},
		{hnSourceID, trss.Item{ID: "hn-2", Fingerprint: "fp-hn-2", Title: "Compiler renaissance", URL: "https://example.com/compiler-renaissance", URLCanonical: "https://example.com/compiler-renaissance", Source: trss.ItemSource{Name: "Hacker News", Icon: "🔶", Via: "hackernews"}, PublishedAt: now.Add(-2 * time.Hour), FetchedAt: now, Summary: "Why compilers matter again.", Engagement: map[string]any{"points": 187, "comments": 107}}},
		{ghSourceID, trss.Item{ID: "gh-1", Fingerprint: "fp-gh-1", Title: "karpathy/autoresearch", URL: "https://github.com/karpathy/autoresearch", URLCanonical: "https://github.com/karpathy/autoresearch", Source: trss.ItemSource{Name: "GitHub Trending", Icon: "🐙", Via: "github-trending"}, PublishedAt: now.Add(-90 * time.Minute), FetchedAt: now, Summary: "Automated literature reviews.", Engagement: map[string]any{"stars": 1500, "forks": 120}}},
	}

	for _, entry := range seed {
		if err := st.InsertItem(entry.item, entry.sourceID); err != nil {
			t.Fatalf("InsertItem(%s): %v", entry.item.ID, err)
		}
	}

	svc := NewBriefingService(st, cfg)
	briefing, err := svc.Build(context.Background(), BuildOptions{})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}

	got := renderBriefing(briefing)
	goldenPath := filepath.Join("testdata", "basic_digest.golden")
	want, err := os.ReadFile(goldenPath)
	if err != nil {
		t.Fatalf("ReadFile(%s): %v", goldenPath, err)
	}

	if got != string(want) {
		t.Fatalf("golden mismatch\n--- got ---\n%s\n--- want ---\n%s", got, string(want))
	}
}

func renderBriefing(briefing *intel.Briefing) string {
	if briefing == nil {
		return "<nil>\n"
	}

	var b strings.Builder
	fmt.Fprintf(&b, "Title: %s\n", briefing.Title)
	fmt.Fprintf(&b, "Items: %d\n", len(briefing.Items))
	fmt.Fprintf(&b, "Sections: %d\n", len(briefing.Sections))
	fmt.Fprintf(&b, "Meta: sources=%d considered=%d deduped=%d rules=%d\n", briefing.Meta.SourcesSynced, briefing.Meta.ItemsConsidered, briefing.Meta.ItemsDeduped, briefing.Meta.RulesApplied)

	for i, section := range briefing.Sections {
		fmt.Fprintf(&b, "Section %d: %s [%s] %v\n", i+1, section.Name, section.Kind, section.ClusterIDs)
	}
	for i, item := range briefing.Items {
		fmt.Fprintf(&b, "Item %d: %s | %s | %s\n", i+1, item.Item.SourceName, item.Item.Title, item.Item.CanonicalURL)
	}
	return b.String()
}

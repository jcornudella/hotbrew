package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/jcornudella/hotbrew/internal/app"
	"github.com/jcornudella/hotbrew/internal/briefing"
	"github.com/jcornudella/hotbrew/internal/config"
	"github.com/jcornudella/hotbrew/internal/intel"
	"github.com/jcornudella/hotbrew/internal/sanitize"
	"github.com/jcornudella/hotbrew/internal/store"
	"github.com/jcornudella/hotbrew/internal/store/repo"
	"github.com/jcornudella/hotbrew/pkg/trss"
)

func (r *Root) cmdDigest(args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	st, err := store.Open(cfg.GetDBPath())
	if err != nil {
		return fmt.Errorf("open store: %w", err)
	}
	repo := repo.New(st)
	defer repo.Close()

	service := app.NewBriefingService(st, cfg)

	// --json still serves the raw digest for now; theme assembly is a
	// render concern. A dedicated briefing --json command lands later.
	if len(args) > 0 && args[0] == "--json" {
		digest, err := service.BuildDigest(context.Background(), app.BuildOptions{})
		if err != nil {
			return fmt.Errorf("generate digest: %w", err)
		}
		return trss.EncodeDigest(os.Stdout, digest)
	}

	b, err := service.Build(context.Background(), app.BuildOptions{})
	if err != nil {
		return fmt.Errorf("generate briefing: %w", err)
	}

	printBriefing(b)
	promptIssueRating()
	return nil
}

func printBriefing(b *intel.Briefing) {
	if b == nil {
		return
	}

	fmt.Printf("\n☕ %s\n", b.Title)
	fmt.Printf("   %s | %d items | %d sources\n\n",
		b.Date.Local().Format("Jan 2, 3:04 PM"),
		len(b.Items), b.Meta.SourcesSynced)

	items := indexItems(b.Items)
	clusters := indexClusters(b.Clusters)
	scores := scoresByID(b.Items)

	for _, section := range b.Sections {
		fmt.Printf("── %s ──\n\n", section.Name)
		for _, clusterID := range section.ClusterIDs {
			cluster, ok := clusters[clusterID]
			if !ok {
				continue
			}
			printCluster(cluster, items, scores)
		}
	}

	if b.Meta.ItemsDeduped > 0 || b.Meta.RulesApplied > 0 {
		fmt.Printf("  --- %d deduped, %d rules applied ---\n\n",
			b.Meta.ItemsDeduped, b.Meta.RulesApplied)
	}
}

func printCluster(c intel.ThemeCluster, items map[string]intel.ScoredItem, scores map[string]float64) {
	lead, ok := items[c.Representative]
	if !ok {
		return
	}

	badge := scoreBadge(lead.Score)
	title := sanitize.Text(lead.Item.Title)
	summary := sanitize.Text(lead.Item.Summary)
	url := sanitize.Text(lead.Item.CanonicalURL)
	source := sanitize.Text(lead.Item.SourceName)

	fmt.Printf("  %s %s\n", badge, title)
	if summary != "" {
		if len(summary) > 120 {
			summary = summary[:117] + "..."
		}
		fmt.Printf("       %s\n", summary)
	}
	fmt.Printf("       %s %s", lead.Item.SourceIcon, source)
	if url != "" {
		fmt.Printf(" · %s", url)
	}
	fmt.Println()

	for _, id := range briefing.SupportingItems(c, scores) {
		sup, ok := items[id]
		if !ok {
			continue
		}
		supTitle := sanitize.Text(sup.Item.Title)
		supSource := sanitize.Text(sup.Item.SourceName)
		fmt.Printf("         ↳ %s · %s\n", supTitle, supSource)
	}
	fmt.Println()
}

func scoreBadge(score float64) string {
	switch {
	case score >= 7:
		return "🔥"
	case score >= 4:
		return "⭐"
	default:
		return "  "
	}
}

func indexItems(items []intel.ScoredItem) map[string]intel.ScoredItem {
	out := make(map[string]intel.ScoredItem, len(items))
	for _, item := range items {
		out[item.Item.ID] = item
	}
	return out
}

func indexClusters(clusters []intel.ThemeCluster) map[string]intel.ThemeCluster {
	out := make(map[string]intel.ThemeCluster, len(clusters))
	for _, c := range clusters {
		out[c.ID] = c
	}
	return out
}

func scoresByID(items []intel.ScoredItem) map[string]float64 {
	out := make(map[string]float64, len(items))
	for _, item := range items {
		out[item.Item.ID] = item.Score
	}
	return out
}

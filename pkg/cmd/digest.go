package cmd

import (
	"fmt"
	"os"

	"github.com/jcornudella/hotbrew/internal/config"
	"github.com/jcornudella/hotbrew/internal/curation"
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

	engine := curation.NewEngine(st)
	window := cfg.GetDigestWindow()
	maxItems := cfg.GetDigestMax()

	digest, err := engine.GenerateDigest(window, maxItems, "Hotbrew Digest")
	if err != nil {
		return fmt.Errorf("generate digest: %w", err)
	}

	if len(args) > 0 && args[0] == "--json" {
		return trss.EncodeDigest(os.Stdout, digest)
	}

	printDigest(digest)
	promptIssueRating()
	return nil
}

func printDigest(d *trss.Digest) {
	fmt.Printf("\n☕ %s\n", d.Title)
	fmt.Printf("   %s | %d items | %d sources\n\n",
		d.GeneratedAt.Local().Format("Jan 2, 3:04 PM"),
		d.ItemCount, d.Meta.SourcesSynced)

	for i, item := range d.Items {
		score := "  "
		if item.Score >= 7 {
			score = "🔥"
		} else if item.Score >= 4 {
			score = "⭐"
		}

		title := sanitize.Text(item.Title)
		summary := sanitize.Text(item.Summary)
		url := sanitize.Text(item.URL)
		sourceName := sanitize.Text(item.Source.Name)
		fmt.Printf("  %s %2d. %s\n", score, i+1, title)
		if summary != "" {
			if len(summary) > 120 {
				summary = summary[:117] + "..."
			}
			fmt.Printf("       %s\n", summary)
		}
		fmt.Printf("       %s %s", item.Source.Icon, sourceName)
		if url != "" {
			fmt.Printf(" · %s", url)
		}
		fmt.Println()
		fmt.Println()
	}

	if d.Meta.ItemsDeduped > 0 || d.Meta.RulesApplied > 0 {
		fmt.Printf("  --- %d deduped, %d rules applied ---\n\n",
			d.Meta.ItemsDeduped, d.Meta.RulesApplied)
	}
}

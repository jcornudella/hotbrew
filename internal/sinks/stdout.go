package sinks

import (
	"fmt"

	"github.com/jcornudella/hotbrew/pkg/trss"
)

// Stdout renders a digest as pretty terminal output.
type Stdout struct{}

func (s *Stdout) Deliver(d *trss.Digest) error {
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

		fmt.Printf("  %s %2d. %s\n", score, i+1, item.Title)
		if item.Summary != "" {
			summary := item.Summary
			if len(summary) > 120 {
				summary = summary[:117] + "..."
			}
			fmt.Printf("       %s\n", summary)
		}
		fmt.Printf("       %s %s", item.Source.Icon, item.Source.Name)
		if item.URL != "" {
			fmt.Printf(" · %s", item.URL)
		}
		fmt.Println()
		fmt.Println()
	}

	return nil
}

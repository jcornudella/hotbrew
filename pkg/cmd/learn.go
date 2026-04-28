package cmd

import (
	"fmt"

	"github.com/jcornudella/hotbrew/internal/personalize"
	"github.com/jcornudella/hotbrew/internal/store"
)

// cmdLearn runs the personalization pass on demand and prints a
// summary of what changed. The same logic also runs at the end of
// `sync`, so `learn` is mostly an introspection tool — useful when
// the user wants to see "why does the digest think I'm into X?".
func (r *Root) cmdLearn(args []string) error {
	return withStore(func(st *store.Store) error {
		diff, err := personalize.Learn(st, personalize.Options{})
		if err != nil {
			return fmt.Errorf("learn: %w", err)
		}
		printDiff(diff)
		return nil
	})
}

func printDiff(d personalize.Diff) {
	if d.Events == 0 {
		fmt.Println("☕ No interactions yet. Open, save, or mute a few items, then try again.")
		return
	}
	fmt.Printf("☕ Learned from %d interaction%s.\n", d.Events, plural(d.Events))
	printKindDiff("Themes", d.Themes)
	printKindDiff("Sources", d.Sources)
	printKindDiff("Domains", d.Domains)
}

func printKindDiff(label string, kd personalize.KindDiff) {
	if kd.Total == 0 {
		return
	}
	fmt.Printf("\n%s\n", label)
	for _, r := range kd.Top {
		if r.Score <= 0 {
			break
		}
		fmt.Printf("  ↑ %-32s %+.2f\n", r.Key, r.Score)
	}
	for _, r := range kd.Bottom {
		fmt.Printf("  ↓ %-32s %+.2f\n", r.Key, r.Score)
	}
}

func plural(n int) string {
	if n == 1 {
		return ""
	}
	return "s"
}

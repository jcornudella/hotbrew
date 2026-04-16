package cmd

import (
	"fmt"
	"sort"

	"github.com/jcornudella/hotbrew/internal/clustering"
	"github.com/jcornudella/hotbrew/internal/store"
)

// cmdThemes lists every known content theme alongside the user's
// current preference (follow / mute / neutral). This is the content
// counterpart to `hotbrew theme`, which still handles the TUI
// color palette.
func (r *Root) cmdThemes(args []string) error {
	return withStore(func(st *store.Store) error {
		prefs, err := st.ListThemePreferences()
		if err != nil {
			return fmt.Errorf("list theme preferences: %w", err)
		}

		labels := clustering.KnownLabels()
		sort.Slice(labels, func(i, j int) bool { return labels[i].Slug < labels[j].Slug })

		fmt.Println("☕ Content themes:")
		fmt.Println()
		for _, label := range labels {
			state := prefs[label.Slug]
			fmt.Printf("  %s %-10s %s\n", stateGlyph(state), label.Slug, label.Display)
		}
		fmt.Println()
		fmt.Println("Follow:   hotbrew follow <theme>")
		fmt.Println("Mute:     hotbrew mute-theme <theme>")
		fmt.Println("Reset:    hotbrew unfollow <theme>")
		return nil
	})
}

// stateGlyph renders the same three states everywhere:
// followed (★), muted (🔇), neutral (·). Keeps the list scannable
// without a legend line for each row.
func stateGlyph(state string) string {
	switch state {
	case store.ThemeStateFollow:
		return "★"
	case store.ThemeStateMute:
		return "🔇"
	default:
		return "·"
	}
}

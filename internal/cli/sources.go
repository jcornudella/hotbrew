package cli

import (
	"fmt"
	"os"

	"github.com/jcornudella/hotbrew/internal/store"
)

// Sources handles `hotbrew sources` — lists all registered sources.
func Sources(st *store.Store) {
	sources, err := st.ListSources()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error listing sources: %v\n", err)
		os.Exit(1)
	}

	if len(sources) == 0 {
		fmt.Println("No sources registered.")
		fmt.Print("\nUse 'hotbrew add <url>' to add an RSS feed.\n")
		fmt.Println("Or run 'hotbrew sync' to register built-in sources.")
		return
	}

	fmt.Println("☕ Sources:")
	fmt.Println()
	for _, s := range sources {
		status := "✓"
		if !s.Enabled {
			status = "✗"
		}
		if s.SyncErrors > 0 {
			status = fmt.Sprintf("⚠ (%d errors)", s.SyncErrors)
		}

		lastSync := "never"
		if s.LastSync != nil {
			lastSync = formatAge(*s.LastSync)
		}

		fmt.Printf("  %s %s #%d %s (%s)\n", status, s.Icon, s.ID, s.Name, s.Kind)
		fmt.Printf("      Last sync: %s\n", lastSync)
		if s.URL != "" {
			fmt.Printf("      URL: %s\n", s.URL)
		}
		fmt.Println()
	}
}

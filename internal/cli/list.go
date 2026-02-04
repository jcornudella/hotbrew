package cli

import (
	"fmt"
	"time"

	"github.com/jcornudella/hotbrew/internal/store"
)

// ListOptions controls list output.
type ListOptions struct {
	Unread     bool
	SourceName string
	Top        int
	Since      time.Duration
}

// List handles `hotbrew list`.
func List(st *store.Store, opts ListOptions) {
	if opts.Top <= 0 {
		opts.Top = 20
	}

	items, err := st.ListItems(store.ItemFilter{
		Unread:     opts.Unread,
		SourceName: opts.SourceName,
		Since:      opts.Since,
		Limit:      opts.Top,
	})
	if err != nil {
		fmt.Printf("Error listing items: %v\n", err)
		return
	}

	if len(items) == 0 {
		fmt.Println("No items found. Run 'hotbrew sync' to fetch content.")
		return
	}

	// Header
	counts := st.CountByState()
	fmt.Printf("☕ %d unread · %d read · %d saved\n\n",
		counts["unread"], counts["read"], counts["saved"])

	for i, item := range items {
		state := ""
		if meta, ok := item.Meta["state"].(string); ok {
			switch meta {
			case "read":
				state = " ✓"
			case "saved":
				state = " ★"
			}
		}

		// Score indicator
		score := "  "
		if item.Score >= 7 {
			score = "🔥"
		} else if item.Score >= 4 {
			score = "⭐"
		}

		age := formatAge(item.PublishedAt)

		fmt.Printf("  %s %2d. %s%s\n", score, i+1, item.Title, state)
		fmt.Printf("       %s %s · %s · %s\n",
			item.Source.Icon, item.Source.Name, age, shortID(item.ID))

		if item.Summary != "" {
			summary := item.Summary
			if len(summary) > 100 {
				summary = summary[:97] + "..."
			}
			fmt.Printf("       %s\n", summary)
		}
		fmt.Println()
	}
}

// formatAge returns a human-readable age string.
func formatAge(t time.Time) string {
	d := time.Since(t)
	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		return fmt.Sprintf("%dm ago", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh ago", int(d.Hours()))
	default:
		return fmt.Sprintf("%dd ago", int(d.Hours()/24))
	}
}

// shortID returns a truncated item ID for display.
func shortID(id string) string {
	if len(id) > 13 {
		return id[:13] // "sha256:abcdef" (12 hex + prefix)
	}
	return id
}

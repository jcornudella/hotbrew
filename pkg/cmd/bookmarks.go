package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/jcornudella/hotbrew/internal/config"
	"github.com/jcornudella/hotbrew/internal/sources/xbookmarks"
	"github.com/jcornudella/hotbrew/pkg/source"
)

func (r *Root) cmdBookmarks(args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	var cid string
	if cfg.X != nil {
		cid = cfg.X.ClientID
	}
	src := xbookmarks.New("", "").WithClientID(xbookmarks.ClientID(cid))

	settings := map[string]any{"max": 20}
	if sc, ok := cfg.Sources["xbookmarks"]; ok {
		if sc.Settings != nil {
			settings = sc.Settings
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	section, err := src.Fetch(ctx, source.Config{Enabled: true, Settings: settings})
	if err != nil {
		return fmt.Errorf("fetch bookmarks: %w", err)
	}
	if section == nil {
		fmt.Println("No bookmarks.")
		return nil
	}

	fmt.Printf("\n%s %s\n\n", section.Icon, section.Name)
	if len(section.Items) == 0 {
		fmt.Println("  (no new bookmarks since last sync)")
		return nil
	}
	for _, item := range section.Items {
		fmt.Printf("  %s %s\n", engagementBadge(item), item.Title)
		if item.Subtitle != "" {
			fmt.Printf("    %s\n", item.Subtitle)
		}
		if item.URL != "" {
			fmt.Printf("    %s\n", item.URL)
		}
		if link, ok := item.Metadata["link"].(string); ok && link != "" {
			fmt.Printf("    → %s\n", link)
		}
		fmt.Println()
	}
	return nil
}

func engagementBadge(it source.Item) string {
	switch it.Priority {
	case source.Urgent:
		return "🔥"
	case source.High:
		return "⭐"
	case source.Medium:
		return "•"
	default:
		return " "
	}
}

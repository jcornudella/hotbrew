package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/jcornudella/hotbrew/internal/sources/xbookmarks"
	"github.com/jcornudella/hotbrew/pkg/source"
)

func (r *Root) cmdBookmarks(args []string) error {
	src := xbookmarks.New("", "")

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	section, err := src.Fetch(ctx, source.Config{Enabled: true})
	if err != nil {
		return fmt.Errorf("fetch bookmarks: %w", err)
	}
	if section == nil {
		fmt.Println("No bookmarks.")
		return nil
	}

	fmt.Printf("\n%s %s\n\n", section.Icon, section.Name)
	if len(section.Items) == 0 {
		fmt.Println("  (empty)")
		return nil
	}
	for _, item := range section.Items {
		fmt.Printf("  • %s\n", item.Title)
		if item.Subtitle != "" {
			fmt.Printf("    %s\n", item.Subtitle)
		}
		if item.URL != "" {
			fmt.Printf("    %s\n", item.URL)
		}
	}
	fmt.Println()
	return nil
}

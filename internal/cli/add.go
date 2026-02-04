// Package cli implements hotbrew's CLI commands.
package cli

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/jcornudella/hotbrew/internal/store"
	hsync "github.com/jcornudella/hotbrew/internal/sync"
	"github.com/jcornudella/hotbrew/pkg/source"

	// RSS source for feed auto-detection
	"github.com/jcornudella/hotbrew/internal/sources/rss"
)

// Add handles `hotbrew add <url>`.
// Auto-detects RSS feed, inserts source, and runs initial sync.
func Add(st *store.Store, args []string) {
	if len(args) < 1 {
		fmt.Println("Usage: hotbrew add <feed-url> [name]")
		fmt.Println("\nExamples:")
		fmt.Println("  hotbrew add https://blog.golang.org/feed.atom")
		fmt.Println("  hotbrew add https://simonwillison.net/atom/everything/ \"Simon Willison\"")
		os.Exit(1)
	}

	feedURL := args[0]
	name := feedURL
	if len(args) > 1 {
		name = args[1]
	}

	// Create the source in the store.
	sourceID, err := st.InsertSource(name, "rss", feedURL, "📰", nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error adding source: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✓ Added source #%d: %s\n", sourceID, name)
	fmt.Println("  Fetching initial items...")

	// Do an initial sync for this source.
	src := rss.New(name, feedURL, "📰")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	result := hsync.SyncSource(ctx, st, "rss", src, source.Config{
		Enabled: true,
		Settings: map[string]any{"max": 20},
	})

	if result.Err != nil {
		fmt.Fprintf(os.Stderr, "  Warning: initial sync failed: %v\n", result.Err)
		fmt.Println("  Source saved. It will be fetched on next sync.")
	} else {
		fmt.Printf("  ✓ Fetched %d items\n", result.ItemCount)
	}
}

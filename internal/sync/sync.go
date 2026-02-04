package sync

import (
	"context"
	"fmt"
	"log"

	"github.com/jcornudella/hotbrew/internal/store"
	"github.com/jcornudella/hotbrew/pkg/source"
)

// Result holds the outcome of a sync operation.
type Result struct {
	SourceName string
	ItemCount  int
	Err        error
}

// SyncAll fetches all sources in the registry and stores them.
func SyncAll(ctx context.Context, st *store.Store, registry *source.Registry) []Result {
	var results []Result

	for name, src := range registry.All() {
		cfg := source.Config{Enabled: true}
		res := syncSource(ctx, st, name, src, cfg)
		results = append(results, res)
	}

	return results
}

// SyncSource fetches a single source and stores its items.
func SyncSource(ctx context.Context, st *store.Store, name string, src source.Source, cfg source.Config) Result {
	return syncSource(ctx, st, name, src, cfg)
}

func syncSource(ctx context.Context, st *store.Store, name string, src source.Source, cfg source.Config) Result {
	section, err := src.Fetch(ctx, cfg)
	if err != nil {
		// Track the error in the store if we have a source ID.
		sourceID, _ := st.GetOrCreateSource(src.Name(), name, "", src.Icon())
		if sourceID > 0 {
			st.IncrSyncErrors(sourceID)
		}
		return Result{SourceName: name, Err: fmt.Errorf("fetch %s: %w", name, err)}
	}

	if section == nil || len(section.Items) == 0 {
		return Result{SourceName: name, ItemCount: 0}
	}

	// Ensure the source exists in the store.
	sourceID, err := st.GetOrCreateSource(src.Name(), name, "", src.Icon())
	if err != nil {
		return Result{SourceName: name, Err: fmt.Errorf("register source %s: %w", name, err)}
	}

	// Convert and insert items.
	items := ConvertSection(section, src)
	inserted := 0
	for _, item := range items {
		if err := st.InsertItem(item, sourceID); err != nil {
			log.Printf("sync: insert item %s: %v", item.ID, err)
			continue
		}
		inserted++
	}

	// Update sync timestamp.
	st.UpdateLastSync(sourceID)

	return Result{SourceName: name, ItemCount: inserted}
}

// PrintResults logs sync results to stdout.
func PrintResults(results []Result) {
	total := 0
	errs := 0
	for _, r := range results {
		if r.Err != nil {
			fmt.Printf("  ✗ %s: %v\n", r.SourceName, r.Err)
			errs++
		} else {
			fmt.Printf("  ✓ %s: %d items\n", r.SourceName, r.ItemCount)
			total += r.ItemCount
		}
	}
	fmt.Printf("\nSynced %d items from %d sources", total, len(results)-errs)
	if errs > 0 {
		fmt.Printf(" (%d errors)", errs)
	}
	fmt.Println()
}

package cmd

import (
	"context"
	"fmt"
	"strings"

	"github.com/jcornudella/hotbrew/internal/app"
	"github.com/jcornudella/hotbrew/internal/briefing"
	"github.com/jcornudella/hotbrew/internal/config"
	"github.com/jcornudella/hotbrew/internal/intel"
	"github.com/jcornudella/hotbrew/internal/store"
)

func (r *Root) cmdExplain(args []string) error {
	return runExplanation(args, formatUserFacing)
}

func (r *Root) cmdWhy(args []string) error {
	return runExplanation(args, formatSystemFacing)
}

func runExplanation(args []string, format func(briefing.Explanation) string) error {
	if len(args) == 0 {
		return fmt.Errorf("item id required")
	}
	idPrefix := args[0]

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}
	st, err := store.Open(cfg.GetDBPath())
	if err != nil {
		return fmt.Errorf("open store: %w", err)
	}
	defer func() { _ = st.Close() }()

	service := app.NewBriefingService(st, cfg)
	b, err := service.Build(context.Background(), app.BuildOptions{})
	if err != nil {
		return fmt.Errorf("build briefing: %w", err)
	}

	id, ok := resolveItemID(b.Items, idPrefix)
	if !ok {
		return fmt.Errorf("no item matches %q in the current briefing", idPrefix)
	}

	exp, ok := briefing.Explain(b, id)
	if !ok {
		return fmt.Errorf("could not explain %q", id)
	}

	fmt.Print(format(exp))
	return nil
}

// resolveItemID lets users pass a short prefix instead of the full
// item id. An exact match wins; otherwise we accept a unique prefix.
func resolveItemID(items []intel.ScoredItem, prefix string) (string, bool) {
	var match string
	var count int
	for _, item := range items {
		id := item.Item.ID
		if id == prefix {
			return id, true
		}
		if strings.HasPrefix(id, prefix) {
			match = id
			count++
		}
	}
	if count == 1 {
		return match, true
	}
	return "", false
}

func formatUserFacing(exp briefing.Explanation) string {
	var b strings.Builder
	fmt.Fprintf(&b, "\n☕ %s\n\n", exp.Title)
	if exp.Theme != "" {
		fmt.Fprintf(&b, "   Theme: %s\n", exp.Theme)
	}
	if len(exp.Sources) > 0 {
		fmt.Fprintf(&b, "   Sources: %s\n", strings.Join(exp.Sources, ", "))
	}
	fmt.Fprintf(&b, "\n   Why it matters\n   %s\n\n", exp.WhyItMatters)
	return b.String()
}

func formatSystemFacing(exp briefing.Explanation) string {
	var b strings.Builder
	fmt.Fprintf(&b, "\n☕ %s\n\n", exp.Title)
	fmt.Fprintf(&b, "   %s\n\n", exp.WhyYouSee)
	for _, f := range exp.Factors {
		fmt.Fprintf(&b, "   • %-14s x%.2f — %s\n", f.Name, f.Multiplier, f.Description)
	}
	fmt.Fprintln(&b)
	return b.String()
}

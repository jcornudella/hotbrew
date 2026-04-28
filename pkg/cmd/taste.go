package cmd

import (
	"fmt"
	"math"
	"sort"
	"strings"

	"github.com/jcornudella/hotbrew/internal/store"
)

// cmdTaste renders a snapshot of what hotbrew has inferred about the
// user: behavioral affinity per theme/source/domain plus any explicit
// follow/mute preferences. Read-only — `hotbrew learn` recomputes
// affinity, `hotbrew follow|mute` adjust explicit preferences.
//
// The view is the public face of the personalization loop: if a user
// can't see what was learned about them, they can't trust or steer
// it. So we surface raw scores, mark explicit overrides, and link to
// the commands that change them.
func (r *Root) cmdTaste(args []string) error {
	return withStore(func(st *store.Store) error {
		themes, err := st.ListAffinity(store.AffinityKindTheme)
		if err != nil {
			return fmt.Errorf("list theme affinity: %w", err)
		}
		sources, err := st.ListAffinity(store.AffinityKindSource)
		if err != nil {
			return fmt.Errorf("list source affinity: %w", err)
		}
		domains, err := st.ListAffinity(store.AffinityKindDomain)
		if err != nil {
			return fmt.Errorf("list domain affinity: %w", err)
		}
		prefs, err := st.ListThemePreferences()
		if err != nil {
			return fmt.Errorf("list theme preferences: %w", err)
		}
		events, err := st.ListFeedbackEvents(0)
		if err != nil {
			return fmt.Errorf("list feedback events: %w", err)
		}

		if len(themes)+len(sources)+len(domains)+len(prefs) == 0 {
			fmt.Println("☕ No taste profile yet.")
			fmt.Println("   Open, save, or mute items to teach hotbrew what you like —")
			fmt.Println("   then `hotbrew learn` to refresh, or just run `hotbrew sync`.")
			return nil
		}

		fmt.Printf("☕ Your taste — based on %d interaction%s.\n", len(events), plural(len(events)))

		renderTasteSection("Themes", themes, prefs)
		renderTasteSection("Sources", sources, nil)
		renderTasteSection("Domains", domains, nil)

		fmt.Println()
		fmt.Println("Explicit overrides always win. Learned scores tilt within them.")
		fmt.Println("  Adjust themes:    hotbrew follow <theme>  /  hotbrew mute-theme <theme>")
		fmt.Println("  Adjust domains:   hotbrew mute <domain>")
		fmt.Println("  Refresh learning: hotbrew learn")
		return nil
	})
}

// renderTasteSection prints one (kind, scores) block. Themes pass in
// the explicit preferences map so followed/muted entries get a glyph
// and a label. Sources and domains pass nil; their explicit overrides
// live in different tables (sources.weight, rules) and are out of
// scope for the v1 dashboard.
func renderTasteSection(label string, scores map[string]float64, prefs map[string]string) {
	if len(scores) == 0 && len(prefs) == 0 {
		return
	}

	type row struct {
		key   string
		score float64
		state string
	}

	rows := make([]row, 0, len(scores)+len(prefs))
	seen := map[string]bool{}
	for k, v := range scores {
		rows = append(rows, row{key: k, score: v, state: prefs[k]})
		seen[k] = true
	}
	// Surface explicit follow/mute even when there's no behavioral
	// signal yet — otherwise a freshly-followed theme would be
	// invisible until the user generated events on it.
	for slug, state := range prefs {
		if seen[slug] {
			continue
		}
		rows = append(rows, row{key: slug, score: 0, state: state})
	}

	sort.SliceStable(rows, func(i, j int) bool {
		// Explicit follow > positive score > zero > negative score > muted
		ri, rj := rows[i], rows[j]
		if ri.state != rj.state {
			return statePriority(ri.state) > statePriority(rj.state)
		}
		return ri.score > rj.score
	})

	maxAbs := 0.0
	for _, r := range rows {
		if a := math.Abs(r.score); a > maxAbs {
			maxAbs = a
		}
	}

	fmt.Printf("\n%s\n", label)
	for _, r := range rows {
		glyph := stateGlyph(r.state)
		bar := proportionalBar(r.score, maxAbs, 20)
		fmt.Printf("  %s %-20s %+6.2f  %s\n", glyph, truncate(r.key, 20), r.score, bar)
	}
}

// statePriority orders rows so explicit follow rises to the top and
// explicit mute sinks to the bottom — keeps the user's intent visible
// regardless of how strong the learned score happens to be.
func statePriority(state string) int {
	switch state {
	case store.ThemeStateFollow:
		return 2
	case "":
		return 1
	case store.ThemeStateMute:
		return 0
	}
	return 1
}

// proportionalBar renders a bar of width up to `width` chars. Positive
// scores use `█`, negative use `░` so the sign reads at a glance even
// in monochrome terminals. maxAbs=0 means everything is zero — return
// blanks to keep alignment consistent.
func proportionalBar(score, maxAbs float64, width int) string {
	if maxAbs <= 0 {
		return ""
	}
	frac := math.Abs(score) / maxAbs
	n := int(math.Round(frac * float64(width)))
	if n < 0 {
		n = 0
	}
	if n > width {
		n = width
	}
	if score < 0 {
		return strings.Repeat("░", n)
	}
	return strings.Repeat("█", n)
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	if n <= 1 {
		return s[:n]
	}
	return s[:n-1] + "…"
}

package cmd

import (
	"fmt"
	"strings"

	"github.com/jcornudella/hotbrew/internal/clustering"
	"github.com/jcornudella/hotbrew/internal/store"
)

// cmdFollow marks a theme as preferred. Ranking then multiplies items
// in that theme by ranking.FollowedThemeBoost on the next digest build.
func (r *Root) cmdFollow(args []string) error {
	slug, err := requireThemeArg(args, "follow")
	if err != nil {
		return err
	}
	return withStore(func(st *store.Store) error {
		if err := st.SetThemePreference(slug, store.ThemeStateFollow); err != nil {
			return fmt.Errorf("follow theme: %w", err)
		}
		fmt.Printf("☕ Following %s\n", slug)
		return nil
	})
}

// cmdUnfollow clears a preference row. Removal, not a third "neutral"
// state, so the default-case branch in ranking stays simple.
func (r *Root) cmdUnfollow(args []string) error {
	slug, err := requireThemeArg(args, "unfollow")
	if err != nil {
		return err
	}
	return withStore(func(st *store.Store) error {
		if err := st.DeleteThemePreference(slug); err != nil {
			return fmt.Errorf("unfollow theme: %w", err)
		}
		fmt.Printf("○ Unfollowed %s\n", slug)
		return nil
	})
}

// cmdMuteTheme hides a theme from future briefings.
func (r *Root) cmdMuteTheme(args []string) error {
	slug, err := requireThemeArg(args, "mute-theme")
	if err != nil {
		return err
	}
	return withStore(func(st *store.Store) error {
		if err := st.SetThemePreference(slug, store.ThemeStateMute); err != nil {
			return fmt.Errorf("mute theme: %w", err)
		}
		fmt.Printf("🔇 Muted %s\n", slug)
		return nil
	})
}

func requireThemeArg(args []string, verb string) (string, error) {
	if len(args) == 0 {
		return "", fmt.Errorf("usage: hotbrew %s <theme>\n\n%s", verb, knownThemeList())
	}
	slug := strings.ToLower(strings.TrimSpace(args[0]))
	if !clustering.IsKnownLabel(slug) {
		return "", fmt.Errorf("unknown theme %q\n\n%s", slug, knownThemeList())
	}
	return slug, nil
}

func knownThemeList() string {
	var b strings.Builder
	b.WriteString("Known themes:\n")
	for _, label := range clustering.KnownLabels() {
		fmt.Fprintf(&b, "  %-10s %s\n", label.Slug, label.Display)
	}
	return b.String()
}

package components

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/jcornudella/digest/internal/ui/theme"
)

// KeyBinding represents a keyboard shortcut
type KeyBinding struct {
	Key  string
	Help string
}

// Footer renders the help bar at the bottom
func Footer(t theme.Theme, width int) string {
	bindings := []KeyBinding{
		{Key: "j/k", Help: "navigate"},
		{Key: "enter", Help: "expand"},
		{Key: "o", Help: "open"},
		{Key: "r", Help: "refresh"},
		{Key: "q", Help: "quit"},
	}

	var parts []string
	for _, b := range bindings {
		key := lipgloss.NewStyle().
			Foreground(t.Accent()).
			Bold(true).
			Render(b.Key)
		help := t.MutedStyle().Render(b.Help)
		parts = append(parts, key+" "+help)
	}

	content := strings.Join(parts, "  │  ")

	// Center it
	contentWidth := lipgloss.Width(content)
	padding := (width - contentWidth) / 2
	if padding < 0 {
		padding = 0
	}

	separator := t.MutedStyle().Render(strings.Repeat("─", width))

	return separator + "\n" + strings.Repeat(" ", padding) + content
}

// LoadingFooter shows loading state
func LoadingFooter(t theme.Theme, width int) string {
	content := t.AccentStyle().Render("⠋ Loading...")
	separator := t.MutedStyle().Render(strings.Repeat("─", width))
	return separator + "\n" + content
}

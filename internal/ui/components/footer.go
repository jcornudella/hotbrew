package components

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/jcornudella/hotbrew/internal/ui/theme"
)

// KeyBinding represents a keyboard shortcut
type KeyBinding struct {
	Key  string
	Help string
}

// Footer renders the help bar at the bottom
func Footer(t theme.Theme, width int, expanded bool, current int, total int) string {
	bindings := collapsedBindings()
	if expanded {
		bindings = expandedBindings()
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
	progress := renderProgress(t, current, total)

	line := content
	if progress != "" {
		space := width - lipgloss.Width(content) - lipgloss.Width(progress)
		if space < 1 {
			space = 1
		}
		line = content + strings.Repeat(" ", space) + progress
	}

	separator := t.MutedStyle().Render(strings.Repeat("─", width))

	return separator + "\n" + line
}

func collapsedBindings() []KeyBinding {
	return []KeyBinding{
		{Key: "j/k", Help: "navigate"},
		{Key: "tab", Help: "next section"},
		{Key: "1-9", Help: "jump"},
		{Key: "enter", Help: "expand"},
		{Key: "s", Help: "save"},
		{Key: "p", Help: "profiles"},
		{Key: "t", Help: "theme"},
	}
}

func expandedBindings() []KeyBinding {
	return []KeyBinding{
		{Key: "esc", Help: "collapse"},
		{Key: "o", Help: "open"},
		{Key: "c", Help: "comments"},
		{Key: "s", Help: "save"},
		{Key: "p", Help: "profiles"},
	}
}

func renderProgress(t theme.Theme, current, total int) string {
	if total <= 0 {
		return ""
	}
	if current <= 0 {
		current = 1
	}
	if current > total {
		current = total
	}
	barWidth := 10
	ratio := float64(current) / float64(total)
	filled := int(ratio * float64(barWidth))
	if filled > barWidth {
		filled = barWidth
	}
	bar := strings.Repeat("━", filled) + strings.Repeat("─", barWidth-filled)
	barStyled := lipgloss.NewStyle().Foreground(t.Accent()).Render(bar)
	count := t.MutedStyle().Render(fmt.Sprintf("%d/%d", current, total))
	return count + " " + barStyled
}

// LoadingFooter shows loading state
func LoadingFooter(t theme.Theme, width int) string {
	content := t.AccentStyle().Render("⠋ Loading...")
	separator := t.MutedStyle().Render(strings.Repeat("─", width))
	return separator + "\n" + content
}

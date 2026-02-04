package components

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/jcornudella/hotbrew/internal/ui/gradient"
	"github.com/jcornudella/hotbrew/internal/ui/theme"
	"github.com/jcornudella/hotbrew/pkg/source"
)

// SectionHeader renders a section header with icon and gradient title
func SectionHeader(s *source.Section, t theme.Theme, width int) string {
	// Icon + gradient name
	title := fmt.Sprintf("%s %s", s.Icon, gradient.Text(strings.ToUpper(s.Name), t.HeaderGradient()))

	// Item count
	count := t.MutedStyle().Render(fmt.Sprintf("(%d)", len(s.Items)))

	// Separator line
	titleWidth := lipgloss.Width(title) + lipgloss.Width(count) + 1
	separatorWidth := width - titleWidth - 2
	if separatorWidth < 3 {
		separatorWidth = 3
	}
	separator := t.MutedStyle().Render(strings.Repeat(t.Separator(), separatorWidth))

	return fmt.Sprintf("%s %s %s", title, count, separator)
}

// Section renders a complete section with header and items
func Section(s *source.Section, t theme.Theme, width int, selectedIdx int, isSectionSelected bool, expanded bool) string {
	if s == nil || len(s.Items) == 0 {
		return ""
	}

	var lines []string

	// Section header
	lines = append(lines, SectionHeader(s, t, width))

	// Items
	for i, item := range s.Items {
		isSelected := isSectionSelected && i == selectedIdx
		if isSelected && expanded {
			lines = append(lines, ItemExpanded(item, t, width))
		} else {
			lines = append(lines, Item(item, t, isSelected))
		}
	}

	// Add spacing
	lines = append(lines, "")

	return strings.Join(lines, "\n")
}

// EmptySection renders a message when a section has no items
func EmptySection(name, icon string, t theme.Theme, width int) string {
	title := fmt.Sprintf("%s %s", icon, gradient.Text(strings.ToUpper(name), t.HeaderGradient()))
	separator := t.MutedStyle().Render(strings.Repeat(t.Separator(), width-lipgloss.Width(title)-2))
	header := fmt.Sprintf("%s %s", title, separator)

	empty := t.MutedStyle().Render("  No items")

	return fmt.Sprintf("%s\n%s\n", header, empty)
}

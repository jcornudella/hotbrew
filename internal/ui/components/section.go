package components

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/jcornudella/hotbrew/internal/ui/gradient"
	"github.com/jcornudella/hotbrew/internal/ui/theme"
	"github.com/jcornudella/hotbrew/pkg/source"
)

// sectionAccentColumnWidth is kept at 0 — the lateral accent bar is removed
// for a cleaner look. Section headers use inline color instead.
const sectionAccentColumnWidth = 0

var sourceColorHints = []struct {
	match string
	color string
}{
	{match: "hacker news", color: "#ff7a18"},
	{match: "reddit", color: "#5dade2"},
	{match: "research", color: "#34d399"},
	{match: "github", color: "#8be9fd"},
	{match: "lobsters", color: "#ff6b6b"},
}

// SectionHeader renders a section header with icon and gradient title.
func SectionHeader(s *source.Section, t theme.Theme, width int, accent lipgloss.Color) string {
	icon := s.Icon
	if icon == "" {
		icon = "•"
	}
	iconStyled := lipgloss.NewStyle().Foreground(accent).Render(icon)
	title := fmt.Sprintf("%s %s", iconStyled, gradient.Text(strings.ToUpper(s.Name), t.HeaderGradient()))

	count := t.MutedStyle().Render(fmt.Sprintf("(%d)", len(s.Items)))

	used := lipgloss.Width(title) + lipgloss.Width(count) + 2
	separatorWidth := width - used
	if separatorWidth < 3 {
		separatorWidth = 3
	}
	separator := t.MutedStyle().Render(strings.Repeat(t.Separator(), separatorWidth))

	return fmt.Sprintf("%s %s %s", title, count, separator)
}

// Section renders a complete section with header and items.
func Section(s *source.Section, t theme.Theme, width int, selectedIdx int, isSectionSelected bool, expanded bool) string {
	if s == nil || len(s.Items) == 0 {
		return ""
	}

	accent := sectionAccentColor(s, t)
	tag := sourceTag(s)

	contentWidth := width - sectionAccentColumnWidth
	if contentWidth < 20 {
		contentWidth = width
	}

	var lines []string
	lines = append(lines, "")
	lines = append(lines, SectionHeader(s, t, contentWidth, accent))

	for i, item := range s.Items {
		isSelected := isSectionSelected && i == selectedIdx
		isFirst := i == 0
		if isSelected && expanded {
			lines = append(lines, ItemExpanded(item, t, contentWidth, isSelected, isFirst, tag, accent))
		} else {
			lines = append(lines, Item(item, t, contentWidth, isSelected, isFirst, tag, accent))
		}
	}

	lines = append(lines, "")

	return strings.Join(lines, "\n")
}

// EmptySection renders a message when a section has no items.
func EmptySection(name, icon string, t theme.Theme, width int) string {
	section := &source.Section{Name: name, Icon: icon, Items: nil}
	accent := sectionAccentColor(section, t)

	contentWidth := width
	if contentWidth < 10 {
		contentWidth = width
	}

	lines := []string{""}
	lines = append(lines, SectionHeader(section, t, contentWidth, accent))
	lines = append(lines, t.MutedStyle().Render("  No items"))
	lines = append(lines, "")

	return strings.Join(lines, "\n")
}

func sectionAccentColor(s *source.Section, t theme.Theme) lipgloss.Color {
	name := ""
	if s != nil {
		name = strings.ToLower(s.Name)
	}
	for _, hint := range sourceColorHints {
		if strings.Contains(name, hint.match) {
			return lipgloss.Color(hint.color)
		}
	}

	palette := t.HeaderGradient()
	if len(palette) == 0 {
		return t.Accent()
	}

	sum := 0
	for _, r := range name {
		sum += int(r)
	}
	idx := 0
	if len(name) > 0 {
		idx = sum % len(palette)
	}
	return lipgloss.Color(palette[idx])
}

func sourceTag(s *source.Section) string {
	if s == nil {
		return ""
	}
	words := strings.Fields(s.Name)
	if len(words) == 0 {
		return ""
	}
	if len(words) == 1 {
		runes := []rune(words[0])
		if len(runes) >= 2 {
			return strings.ToUpper(string(runes[:2]))
		}
		return strings.ToUpper(words[0])
	}
	first := []rune(words[0])
	second := []rune(words[1])
	return strings.ToUpper(string(first[0]) + string(second[0]))
}

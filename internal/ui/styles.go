package ui

import "github.com/charmbracelet/lipgloss"

// Common styles used across the app

// Box creates a bordered box
func Box(content string, color lipgloss.Color, width int) string {
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(color).
		Padding(1, 2).
		Width(width).
		Render(content)
}

// DoubleBox creates a double-bordered box
func DoubleBox(content string, color lipgloss.Color, width int) string {
	return lipgloss.NewStyle().
		Border(lipgloss.DoubleBorder()).
		BorderForeground(color).
		Padding(1, 2).
		Width(width).
		Render(content)
}

// Card creates a card-style container
func Card(title, content string, color lipgloss.Color, width int) string {
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(color).
		MarginBottom(1)

	contentStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#ffffff"))

	inner := lipgloss.JoinVertical(
		lipgloss.Left,
		titleStyle.Render(title),
		contentStyle.Render(content),
	)

	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(color).
		Padding(1, 2).
		Width(width).
		Render(inner)
}

// Badge creates a small badge/tag
func Badge(text string, fg, bg lipgloss.Color) string {
	return lipgloss.NewStyle().
		Foreground(fg).
		Background(bg).
		Padding(0, 1).
		Bold(true).
		Render(text)
}

// Divider creates a horizontal divider
func Divider(width int, color lipgloss.Color) string {
	line := ""
	for i := 0; i < width; i++ {
		line += "─"
	}
	return lipgloss.NewStyle().Foreground(color).Render(line)
}

// DividerWithLabel creates a divider with centered text
func DividerWithLabel(label string, width int, color lipgloss.Color) string {
	labelLen := len(label) + 2 // padding
	sideLen := (width - labelLen) / 2

	if sideLen < 1 {
		sideLen = 1
	}

	side := ""
	for i := 0; i < sideLen; i++ {
		side += "─"
	}

	style := lipgloss.NewStyle().Foreground(color)
	labelStyle := lipgloss.NewStyle().Foreground(color).Bold(true)

	return style.Render(side) + " " + labelStyle.Render(label) + " " + style.Render(side)
}

// ProgressBar creates a simple progress bar
func ProgressBar(percent float64, width int, filled, empty lipgloss.Color) string {
	if percent < 0 {
		percent = 0
	}
	if percent > 1 {
		percent = 1
	}

	filledWidth := int(float64(width) * percent)
	emptyWidth := width - filledWidth

	filledStyle := lipgloss.NewStyle().Foreground(filled)
	emptyStyle := lipgloss.NewStyle().Foreground(empty)

	bar := ""
	for i := 0; i < filledWidth; i++ {
		bar += filledStyle.Render("█")
	}
	for i := 0; i < emptyWidth; i++ {
		bar += emptyStyle.Render("░")
	}

	return bar
}

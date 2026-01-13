package components

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/jcornudella/digest/internal/ui"
	"github.com/jcornudella/digest/internal/ui/theme"
)

// Header renders the main digest header with date and greeting
func Header(t theme.Theme, width int) string {
	now := time.Now()

	// Greeting based on time of day
	greeting := getGreeting(now)

	// Format date nicely
	date := now.Format("Monday, January 2")

	// Create the gradient title
	title := ui.GradientBold("DIGEST", t.HeaderGradient())

	// Time with icon
	timeStr := now.Format("3:04 PM")

	// Build header content
	left := fmt.Sprintf("%s  %s", title, t.MutedStyle().Render(date))
	right := fmt.Sprintf("%s  %s", greeting, t.AccentStyle().Render(timeStr))

	// Calculate spacing
	leftWidth := lipgloss.Width(left)
	rightWidth := lipgloss.Width(right)
	spacing := width - leftWidth - rightWidth - 4

	if spacing < 1 {
		spacing = 1
	}

	content := left + strings.Repeat(" ", spacing) + right

	// Wrap in a styled box
	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(t.Primary()).
		Padding(0, 1).
		Width(width - 2)

	return boxStyle.Render(content)
}

// getGreeting returns a greeting based on time of day
func getGreeting(t time.Time) string {
	hour := t.Hour()
	switch {
	case hour < 6:
		return "🌙 Night owl"
	case hour < 12:
		return "☀️  Good morning"
	case hour < 17:
		return "🌤  Good afternoon"
	case hour < 21:
		return "🌅 Good evening"
	default:
		return "🌙 Good night"
	}
}

// CompactHeader renders a smaller header for narrow terminals
func CompactHeader(t theme.Theme) string {
	now := time.Now()
	title := ui.GradientBold("DIGEST", t.HeaderGradient())
	date := t.MutedStyle().Render(now.Format("Jan 2, 3:04 PM"))

	return fmt.Sprintf("%s %s", title, date)
}

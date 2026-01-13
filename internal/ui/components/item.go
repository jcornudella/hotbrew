package components

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/jcornudella/hotbrew/internal/ui/theme"
	"github.com/jcornudella/hotbrew/pkg/source"
)

// Item renders a single item in the digest
func Item(item source.Item, t theme.Theme, selected bool) string {
	// Choose style based on selection
	var bullet string
	var titleStyle lipgloss.Style

	if selected {
		bullet = t.BulletSelected()
		titleStyle = t.ItemSelectedStyle()
	} else {
		bullet = t.Bullet()
		titleStyle = t.ItemStyle()
	}

	// Priority indicator
	priority := priorityIndicator(item.Priority, t)

	// Title (truncate if too long)
	title := truncate(item.Title, 60)

	// Time ago
	timeAgo := formatTimeAgo(item.Timestamp, t)

	// Build the line
	line := fmt.Sprintf("%s%s %s %s", bullet, priority, titleStyle.Render(title), timeAgo)

	return line
}

// ItemExpanded renders an item with its body/subtitle visible
func ItemExpanded(item source.Item, t theme.Theme, width int) string {
	lines := []string{Item(item, t, true)}

	if item.Subtitle != "" {
		subtitle := t.SubtitleStyle().Render("    " + truncate(item.Subtitle, width-6))
		lines = append(lines, subtitle)
	}

	if item.Body != "" {
		body := t.MutedStyle().Render("    " + truncate(item.Body, width-6))
		lines = append(lines, body)
	}

	if item.URL != "" {
		url := t.AccentStyle().Render("    " + item.URL)
		lines = append(lines, url)
	}

	return strings.Join(lines, "\n")
}

// priorityIndicator returns a colored indicator based on priority
func priorityIndicator(p source.Priority, t theme.Theme) string {
	var color lipgloss.Color
	var icon string

	switch p {
	case source.Urgent:
		color = t.PriorityUrgent()
		icon = "●"
	case source.High:
		color = t.PriorityHigh()
		icon = "●"
	case source.Medium:
		color = t.PriorityMedium()
		icon = "○"
	default:
		color = t.PriorityLow()
		icon = "·"
	}

	return lipgloss.NewStyle().Foreground(color).Render(icon)
}

// formatTimeAgo returns a human-readable time difference
func formatTimeAgo(timestamp time.Time, t theme.Theme) string {
	if timestamp.IsZero() {
		return ""
	}

	diff := time.Since(timestamp)
	var text string

	switch {
	case diff < time.Minute:
		text = "now"
	case diff < time.Hour:
		mins := int(diff.Minutes())
		text = fmt.Sprintf("%dm", mins)
	case diff < 24*time.Hour:
		hours := int(diff.Hours())
		text = fmt.Sprintf("%dh", hours)
	case diff < 7*24*time.Hour:
		days := int(diff.Hours() / 24)
		text = fmt.Sprintf("%dd", days)
	default:
		text = timestamp.Format("Jan 2")
	}

	return t.MutedStyle().Render(text)
}

// truncate shortens a string to max length with ellipsis
func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	if max <= 3 {
		return s[:max]
	}
	return s[:max-3] + "..."
}

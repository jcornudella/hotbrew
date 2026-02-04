package components

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/jcornudella/hotbrew/internal/ui/gradient"
	"github.com/jcornudella/hotbrew/internal/ui/theme"
)

// Steam animation frames for the coffee cup
var steamFrames = [][]string{
	{"    ) )  ", "   ( (   ", "    ) )  "},
	{"   ( (   ", "    ) )  ", "   ( (   "},
	{"    ) )  ", "   ) )   ", "    ( (  "},
	{"   ( (   ", "    ( (  ", "   ) )   "},
}

// Coffee cup ASCII art
var coffeeCup = []string{
	"   ______  ",
	"  |      |]",
	"  |      | ",
	"   \\____/  ",
}

// HOTBREW ASCII art title
var hotbrewTitle = []string{
	"██╗  ██╗ ██████╗ ████████╗██████╗ ██████╗ ███████╗██╗    ██╗",
	"██║  ██║██╔═══██╗╚══██╔══╝██╔══██╗██╔══██╗██╔════╝██║    ██║",
	"███████║██║   ██║   ██║   ██████╔╝██████╔╝█████╗  ██║ █╗ ██║",
	"██╔══██║██║   ██║   ██║   ██╔══██╗██╔══██╗██╔══╝  ██║███╗██║",
	"██║  ██║╚██████╔╝   ██║   ██████╔╝██║  ██║███████╗╚███╔███╔╝",
	"╚═╝  ╚═╝ ╚═════╝    ╚═╝   ╚═════╝ ╚═╝  ╚═╝╚══════╝ ╚══╝╚══╝ ",
}

// AnimatedHeader renders the header with animated steam
func AnimatedHeader(t theme.Theme, width int, frame int) string {
	now := time.Now()

	// Build animated coffee cup with steam
	steamFrame := steamFrames[frame%len(steamFrames)]
	var cupLines []string

	// Add steam lines
	for _, line := range steamFrame {
		cupLines = append(cupLines, lipgloss.NewStyle().Foreground(t.Accent()).Render(line))
	}
	// Add cup lines
	for _, line := range coffeeCup {
		cupLines = append(cupLines, lipgloss.NewStyle().Foreground(t.Secondary()).Bold(true).Render(line))
	}

	cupArt := strings.Join(cupLines, "\n")

	// Build gradient title
	var titleLines []string
	colors := t.HeaderGradient()
	for _, line := range hotbrewTitle {
		titleLines = append(titleLines, gradient.Text(line, colors))
	}
	titleArt := strings.Join(titleLines, "\n")

	// Tagline
	tagline := t.MutedStyle().Render("  Your morning, piping hot.")

	// Combine cup and title
	leftBlock := lipgloss.JoinHorizontal(
		lipgloss.Top,
		cupArt,
		"  ",
		lipgloss.JoinVertical(lipgloss.Left, titleArt, "", tagline),
	)

	// Right side: greeting + time
	greeting := getGreeting(now)
	date := t.MutedStyle().Render(now.Format("Monday, January 2"))
	timeStr := t.AccentStyle().Render(now.Format("3:04 PM"))

	rightContent := lipgloss.JoinVertical(
		lipgloss.Right,
		"",
		"",
		fmt.Sprintf("%s  %s", greeting, timeStr),
		date,
	)

	// Calculate spacing
	leftWidth := lipgloss.Width(leftBlock)
	rightWidth := lipgloss.Width(rightContent)
	spacing := width - leftWidth - rightWidth - 8

	if spacing < 1 {
		spacing = 1
	}

	content := lipgloss.JoinHorizontal(
		lipgloss.Top,
		leftBlock,
		strings.Repeat(" ", spacing),
		rightContent,
	)

	// Wrap in a styled box
	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(t.Primary()).
		Padding(1, 2).
		Width(width - 2)

	return boxStyle.Render(content)
}

// Header renders the main digest header with date and greeting (non-animated fallback)
func Header(t theme.Theme, width int) string {
	return AnimatedHeader(t, width, 0)
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
	title := gradient.Bold("DIGEST", t.HeaderGradient())
	date := t.MutedStyle().Render(now.Format("Jan 2, 3:04 PM"))

	return fmt.Sprintf("%s %s", title, date)
}

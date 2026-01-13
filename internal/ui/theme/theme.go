// Package theme provides theming support for the digest UI
package theme

import "github.com/charmbracelet/lipgloss"

// Theme defines the interface for digest themes
type Theme interface {
	// Name returns the theme identifier
	Name() string

	// Colors
	Primary() lipgloss.Color
	Secondary() lipgloss.Color
	Accent() lipgloss.Color
	Muted() lipgloss.Color
	Background() lipgloss.Color
	Text() lipgloss.Color
	TextMuted() lipgloss.Color

	// Gradient colors for headers and emphasis
	HeaderGradient() []string

	// Priority colors
	PriorityUrgent() lipgloss.Color
	PriorityHigh() lipgloss.Color
	PriorityMedium() lipgloss.Color
	PriorityLow() lipgloss.Color

	// Pre-built styles
	HeaderStyle() lipgloss.Style
	SectionHeaderStyle() lipgloss.Style
	ItemStyle() lipgloss.Style
	ItemSelectedStyle() lipgloss.Style
	SubtitleStyle() lipgloss.Style
	MutedStyle() lipgloss.Style
	AccentStyle() lipgloss.Style
	BorderStyle() lipgloss.Style

	// Decorations
	Bullet() string
	BulletSelected() string
	Separator() string
}

// Available themes
var themes = map[string]Theme{
	"synthwave": &Synthwave{},
	"nord":      &Nord{},
	"dracula":   &Dracula{},
}

// Get returns a theme by name, defaults to synthwave
func Get(name string) Theme {
	if t, ok := themes[name]; ok {
		return t
	}
	return &Synthwave{}
}

// List returns all available theme names
func List() []string {
	names := make([]string, 0, len(themes))
	for name := range themes {
		names = append(names, name)
	}
	return names
}

package theme

import "github.com/charmbracelet/lipgloss"

// Synthwave is a vibrant, retro-futuristic theme
type Synthwave struct{}

func (s *Synthwave) Name() string { return "synthwave" }

// Colors - neon pinks, purples, and cyans
func (s *Synthwave) Primary() lipgloss.Color    { return lipgloss.Color("#ff6ad5") }
func (s *Synthwave) Secondary() lipgloss.Color  { return lipgloss.Color("#c774e8") }
func (s *Synthwave) Accent() lipgloss.Color     { return lipgloss.Color("#94d0ff") }
func (s *Synthwave) Muted() lipgloss.Color      { return lipgloss.Color("#6a6a8a") }
func (s *Synthwave) Background() lipgloss.Color { return lipgloss.Color("#1a1a2e") }
func (s *Synthwave) Text() lipgloss.Color       { return lipgloss.Color("#ffffff") }
func (s *Synthwave) TextMuted() lipgloss.Color  { return lipgloss.Color("#a0a0c0") }

// Gradient for headers - pink to cyan
func (s *Synthwave) HeaderGradient() []string {
	return []string{
		"#ff6ad5", // Hot pink
		"#c774e8", // Purple
		"#ad8cff", // Lavender
		"#8795e8", // Periwinkle
		"#94d0ff", // Cyan
	}
}

// Priority colors
func (s *Synthwave) PriorityUrgent() lipgloss.Color { return lipgloss.Color("#ff2a6d") }
func (s *Synthwave) PriorityHigh() lipgloss.Color   { return lipgloss.Color("#ff6ad5") }
func (s *Synthwave) PriorityMedium() lipgloss.Color { return lipgloss.Color("#c774e8") }
func (s *Synthwave) PriorityLow() lipgloss.Color    { return lipgloss.Color("#6a6a8a") }

// Styles
func (s *Synthwave) HeaderStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Bold(true).
		Foreground(s.Primary()).
		Padding(0, 1)
}

func (s *Synthwave) SectionHeaderStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Bold(true).
		Foreground(s.Accent()).
		BorderStyle(lipgloss.NormalBorder()).
		BorderBottom(true).
		BorderForeground(s.Muted()).
		MarginTop(1).
		MarginBottom(1)
}

func (s *Synthwave) ItemStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(s.Text()).
		PaddingLeft(2)
}

func (s *Synthwave) ItemSelectedStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(s.Primary()).
		Bold(true).
		PaddingLeft(2)
}

func (s *Synthwave) SubtitleStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(s.TextMuted()).
		Italic(true)
}

func (s *Synthwave) MutedStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(s.Muted())
}

func (s *Synthwave) AccentStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(s.Accent())
}

func (s *Synthwave) BorderStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(s.Primary()).
		Padding(1, 2)
}

// Decorations
func (s *Synthwave) Bullet() string         { return "│ " }
func (s *Synthwave) BulletSelected() string { return "▶ " }
func (s *Synthwave) Separator() string      { return "─" }

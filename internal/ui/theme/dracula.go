package theme

import "github.com/charmbracelet/lipgloss"

// Dracula is a dark, vampire-inspired theme
type Dracula struct{}

func (d *Dracula) Name() string { return "dracula" }

// Colors - purples, pinks, and dark backgrounds
func (d *Dracula) Primary() lipgloss.Color    { return lipgloss.Color("#bd93f9") }
func (d *Dracula) Secondary() lipgloss.Color  { return lipgloss.Color("#ff79c6") }
func (d *Dracula) Accent() lipgloss.Color     { return lipgloss.Color("#8be9fd") }
func (d *Dracula) Muted() lipgloss.Color      { return lipgloss.Color("#6272a4") }
func (d *Dracula) Background() lipgloss.Color { return lipgloss.Color("#282a36") }
func (d *Dracula) Text() lipgloss.Color       { return lipgloss.Color("#f8f8f2") }
func (d *Dracula) TextMuted() lipgloss.Color  { return lipgloss.Color("#bfbfbf") }

func (d *Dracula) HeaderGradient() []string {
	return []string{
		"#ff79c6", // Pink
		"#bd93f9", // Purple
		"#8be9fd", // Cyan
		"#50fa7b", // Green
	}
}

func (d *Dracula) PriorityUrgent() lipgloss.Color { return lipgloss.Color("#ff5555") }
func (d *Dracula) PriorityHigh() lipgloss.Color   { return lipgloss.Color("#ffb86c") }
func (d *Dracula) PriorityMedium() lipgloss.Color { return lipgloss.Color("#f1fa8c") }
func (d *Dracula) PriorityLow() lipgloss.Color    { return lipgloss.Color("#50fa7b") }

func (d *Dracula) HeaderStyle() lipgloss.Style {
	return lipgloss.NewStyle().Bold(true).Foreground(d.Primary()).Padding(0, 1)
}

func (d *Dracula) SectionHeaderStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Bold(true).
		Foreground(d.Accent()).
		BorderStyle(lipgloss.NormalBorder()).
		BorderBottom(true).
		BorderForeground(d.Muted()).
		MarginTop(1).
		MarginBottom(1)
}

func (d *Dracula) ItemStyle() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(d.Text()).PaddingLeft(2)
}

func (d *Dracula) ItemSelectedStyle() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(d.Secondary()).Bold(true).PaddingLeft(2)
}

func (d *Dracula) SubtitleStyle() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(d.TextMuted()).Italic(true)
}

func (d *Dracula) MutedStyle() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(d.Muted())
}

func (d *Dracula) AccentStyle() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(d.Accent())
}

func (d *Dracula) BorderStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(d.Primary()).
		Padding(1, 2)
}

func (d *Dracula) Bullet() string         { return "│ " }
func (d *Dracula) BulletSelected() string { return "▶ " }
func (d *Dracula) Separator() string      { return "─" }

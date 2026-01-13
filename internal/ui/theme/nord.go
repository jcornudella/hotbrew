package theme

import "github.com/charmbracelet/lipgloss"

// Nord is a cool, arctic theme
type Nord struct{}

func (n *Nord) Name() string { return "nord" }

// Colors - arctic blues and frost
func (n *Nord) Primary() lipgloss.Color    { return lipgloss.Color("#88c0d0") }
func (n *Nord) Secondary() lipgloss.Color  { return lipgloss.Color("#81a1c1") }
func (n *Nord) Accent() lipgloss.Color     { return lipgloss.Color("#8fbcbb") }
func (n *Nord) Muted() lipgloss.Color      { return lipgloss.Color("#4c566a") }
func (n *Nord) Background() lipgloss.Color { return lipgloss.Color("#2e3440") }
func (n *Nord) Text() lipgloss.Color       { return lipgloss.Color("#eceff4") }
func (n *Nord) TextMuted() lipgloss.Color  { return lipgloss.Color("#d8dee9") }

func (n *Nord) HeaderGradient() []string {
	return []string{
		"#8fbcbb",
		"#88c0d0",
		"#81a1c1",
		"#5e81ac",
	}
}

func (n *Nord) PriorityUrgent() lipgloss.Color { return lipgloss.Color("#bf616a") }
func (n *Nord) PriorityHigh() lipgloss.Color   { return lipgloss.Color("#d08770") }
func (n *Nord) PriorityMedium() lipgloss.Color { return lipgloss.Color("#ebcb8b") }
func (n *Nord) PriorityLow() lipgloss.Color    { return lipgloss.Color("#a3be8c") }

func (n *Nord) HeaderStyle() lipgloss.Style {
	return lipgloss.NewStyle().Bold(true).Foreground(n.Primary()).Padding(0, 1)
}

func (n *Nord) SectionHeaderStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Bold(true).
		Foreground(n.Accent()).
		BorderStyle(lipgloss.NormalBorder()).
		BorderBottom(true).
		BorderForeground(n.Muted()).
		MarginTop(1).
		MarginBottom(1)
}

func (n *Nord) ItemStyle() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(n.Text()).PaddingLeft(2)
}

func (n *Nord) ItemSelectedStyle() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(n.Primary()).Bold(true).PaddingLeft(2)
}

func (n *Nord) SubtitleStyle() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(n.TextMuted()).Italic(true)
}

func (n *Nord) MutedStyle() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(n.Muted())
}

func (n *Nord) AccentStyle() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(n.Accent())
}

func (n *Nord) BorderStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(n.Primary()).
		Padding(1, 2)
}

func (n *Nord) Bullet() string         { return "│ " }
func (n *Nord) BulletSelected() string { return "▶ " }
func (n *Nord) Separator() string      { return "─" }

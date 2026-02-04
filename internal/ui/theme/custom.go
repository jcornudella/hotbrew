package theme

import "github.com/charmbracelet/lipgloss"

// CustomColors holds user-defined theme colors
type CustomColors struct {
	Primary        string   `yaml:"primary"`
	Secondary      string   `yaml:"secondary"`
	Accent         string   `yaml:"accent"`
	Muted          string   `yaml:"muted"`
	Background     string   `yaml:"background"`
	Text           string   `yaml:"text"`
	TextMuted      string   `yaml:"text_muted"`
	HeaderGradient []string `yaml:"header_gradient"`
	PriorityUrgent string   `yaml:"priority_urgent"`
	PriorityHigh   string   `yaml:"priority_high"`
	PriorityMedium string   `yaml:"priority_medium"`
	PriorityLow    string   `yaml:"priority_low"`
}

// Custom is a user-defined theme
type Custom struct {
	name   string
	colors CustomColors
}

// NewCustom creates a custom theme from colors
func NewCustom(name string, colors CustomColors) *Custom {
	// Fill in defaults from synthwave for any missing colors
	if colors.Primary == "" {
		colors.Primary = "#ff6ad5"
	}
	if colors.Secondary == "" {
		colors.Secondary = "#c774e8"
	}
	if colors.Accent == "" {
		colors.Accent = "#94d0ff"
	}
	if colors.Muted == "" {
		colors.Muted = "#6a6a8a"
	}
	if colors.Background == "" {
		colors.Background = "#1a1a2e"
	}
	if colors.Text == "" {
		colors.Text = "#ffffff"
	}
	if colors.TextMuted == "" {
		colors.TextMuted = "#a0a0c0"
	}
	if len(colors.HeaderGradient) == 0 {
		colors.HeaderGradient = []string{colors.Primary, colors.Secondary, colors.Accent}
	}
	if colors.PriorityUrgent == "" {
		colors.PriorityUrgent = "#ff2a6d"
	}
	if colors.PriorityHigh == "" {
		colors.PriorityHigh = colors.Primary
	}
	if colors.PriorityMedium == "" {
		colors.PriorityMedium = colors.Secondary
	}
	if colors.PriorityLow == "" {
		colors.PriorityLow = colors.Muted
	}

	return &Custom{name: name, colors: colors}
}

func (c *Custom) Name() string { return c.name }

// Colors
func (c *Custom) Primary() lipgloss.Color    { return lipgloss.Color(c.colors.Primary) }
func (c *Custom) Secondary() lipgloss.Color  { return lipgloss.Color(c.colors.Secondary) }
func (c *Custom) Accent() lipgloss.Color     { return lipgloss.Color(c.colors.Accent) }
func (c *Custom) Muted() lipgloss.Color      { return lipgloss.Color(c.colors.Muted) }
func (c *Custom) Background() lipgloss.Color { return lipgloss.Color(c.colors.Background) }
func (c *Custom) Text() lipgloss.Color       { return lipgloss.Color(c.colors.Text) }
func (c *Custom) TextMuted() lipgloss.Color  { return lipgloss.Color(c.colors.TextMuted) }

func (c *Custom) HeaderGradient() []string { return c.colors.HeaderGradient }

// Priority colors
func (c *Custom) PriorityUrgent() lipgloss.Color { return lipgloss.Color(c.colors.PriorityUrgent) }
func (c *Custom) PriorityHigh() lipgloss.Color   { return lipgloss.Color(c.colors.PriorityHigh) }
func (c *Custom) PriorityMedium() lipgloss.Color { return lipgloss.Color(c.colors.PriorityMedium) }
func (c *Custom) PriorityLow() lipgloss.Color    { return lipgloss.Color(c.colors.PriorityLow) }

// Styles
func (c *Custom) HeaderStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Bold(true).
		Foreground(c.Primary()).
		Padding(0, 1)
}

func (c *Custom) SectionHeaderStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Bold(true).
		Foreground(c.Accent()).
		BorderStyle(lipgloss.NormalBorder()).
		BorderBottom(true).
		BorderForeground(c.Muted()).
		MarginTop(1).
		MarginBottom(1)
}

func (c *Custom) ItemStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(c.Text()).
		PaddingLeft(2)
}

func (c *Custom) ItemSelectedStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(c.Primary()).
		Bold(true).
		PaddingLeft(2)
}

func (c *Custom) SubtitleStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(c.TextMuted()).
		Italic(true)
}

func (c *Custom) MutedStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(c.Muted())
}

func (c *Custom) AccentStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(c.Accent())
}

func (c *Custom) BorderStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(c.Primary()).
		Padding(1, 2)
}

// Decorations
func (c *Custom) Bullet() string         { return "│ " }
func (c *Custom) BulletSelected() string { return "▶ " }
func (c *Custom) Separator() string      { return "─" }

// RegisterCustom adds a custom theme to the available themes
func RegisterCustom(name string, colors CustomColors) {
	themes[name] = NewCustom(name, colors)
}

// Presets for quick access
var Presets = map[string]CustomColors{
	"mocha": {
		Primary:        "#d4a574",
		Secondary:      "#a67c52",
		Accent:         "#e8c39e",
		Muted:          "#6b5344",
		Background:     "#1c1410",
		Text:           "#f5e6d3",
		TextMuted:      "#b8a089",
		HeaderGradient: []string{"#d4a574", "#c4956a", "#a67c52", "#8b6545", "#e8c39e"},
		PriorityUrgent: "#ff6b6b",
		PriorityHigh:   "#d4a574",
		PriorityMedium: "#a67c52",
		PriorityLow:    "#6b5344",
	},
	"ocean": {
		Primary:        "#00d9ff",
		Secondary:      "#0099cc",
		Accent:         "#66ffcc",
		Muted:          "#4a6670",
		Background:     "#0a1628",
		Text:           "#e0f7ff",
		TextMuted:      "#8ab4c4",
		HeaderGradient: []string{"#00d9ff", "#00b8d4", "#0099cc", "#00796b", "#66ffcc"},
		PriorityUrgent: "#ff5252",
		PriorityHigh:   "#00d9ff",
		PriorityMedium: "#0099cc",
		PriorityLow:    "#4a6670",
	},
	"forest": {
		Primary:        "#7cb342",
		Secondary:      "#558b2f",
		Accent:         "#c5e1a5",
		Muted:          "#4a5a40",
		Background:     "#1a1f16",
		Text:           "#e8f5e9",
		TextMuted:      "#a5c49a",
		HeaderGradient: []string{"#c5e1a5", "#aed581", "#9ccc65", "#7cb342", "#558b2f"},
		PriorityUrgent: "#ff7043",
		PriorityHigh:   "#7cb342",
		PriorityMedium: "#558b2f",
		PriorityLow:    "#4a5a40",
	},
	"sunset": {
		Primary:        "#ff7043",
		Secondary:      "#ff5722",
		Accent:         "#ffcc80",
		Muted:          "#6d5a4a",
		Background:     "#1f1410",
		Text:           "#fff3e0",
		TextMuted:      "#c9a88a",
		HeaderGradient: []string{"#ffcc80", "#ffb74d", "#ffa726", "#ff9800", "#ff7043", "#ff5722"},
		PriorityUrgent: "#f44336",
		PriorityHigh:   "#ff7043",
		PriorityMedium: "#ff5722",
		PriorityLow:    "#6d5a4a",
	},
	"midnight": {
		Primary:        "#bb86fc",
		Secondary:      "#985eff",
		Accent:         "#03dac6",
		Muted:          "#4a4458",
		Background:     "#121212",
		Text:           "#e1e1e1",
		TextMuted:      "#a0a0a0",
		HeaderGradient: []string{"#bb86fc", "#a66efa", "#985eff", "#7c4dff", "#03dac6"},
		PriorityUrgent: "#cf6679",
		PriorityHigh:   "#bb86fc",
		PriorityMedium: "#985eff",
		PriorityLow:    "#4a4458",
	},
}

func init() {
	// Register preset themes
	for name, colors := range Presets {
		RegisterCustom(name, colors)
	}
}

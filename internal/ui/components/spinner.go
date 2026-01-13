package components

import (
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/lipgloss"
	"github.com/jcornudella/hotbrew/internal/ui/theme"
)

// Spinner styles for different states
type SpinnerStyle int

const (
	SpinnerDot SpinnerStyle = iota
	SpinnerLine
	SpinnerPulse
	SpinnerGlobe
	SpinnerMeter
)

// NewSpinner creates a themed spinner
func NewSpinner(t theme.Theme, style SpinnerStyle) spinner.Model {
	s := spinner.New()

	switch style {
	case SpinnerLine:
		s.Spinner = spinner.Line
	case SpinnerPulse:
		s.Spinner = spinner.Pulse
	case SpinnerGlobe:
		s.Spinner = spinner.Globe
	case SpinnerMeter:
		s.Spinner = spinner.Meter
	default:
		s.Spinner = spinner.Dot
	}

	s.Style = lipgloss.NewStyle().Foreground(t.Primary())
	return s
}

// Custom spinners for extra flair
var (
	// Neon spinner frames
	NeonSpinner = spinner.Spinner{
		Frames: []string{
			"▓▒░",
			"░▓▒",
			"▒░▓",
			"▓▒░",
		},
		FPS: 10,
	}

	// Retro wave spinner
	WaveSpinner = spinner.Spinner{
		Frames: []string{
			"∙∙∙∙∙",
			"●∙∙∙∙",
			"∙●∙∙∙",
			"∙∙●∙∙",
			"∙∙∙●∙",
			"∙∙∙∙●",
			"∙∙∙●∙",
			"∙∙●∙∙",
			"∙●∙∙∙",
			"●∙∙∙∙",
		},
		FPS: 12,
	}

	// Blocks spinner
	BlocksSpinner = spinner.Spinner{
		Frames: []string{
			"█▓▒░",
			"░█▓▒",
			"▒░█▓",
			"▓▒░█",
		},
		FPS: 8,
	}

	// Gradient dots
	GradientSpinner = spinner.Spinner{
		Frames: []string{
			"⠋",
			"⠙",
			"⠹",
			"⠸",
			"⠼",
			"⠴",
			"⠦",
			"⠧",
			"⠇",
			"⠏",
		},
		FPS: 10,
	}
)

// NewCustomSpinner creates a spinner with a custom animation
func NewCustomSpinner(t theme.Theme, s spinner.Spinner) spinner.Model {
	sp := spinner.New()
	sp.Spinner = s
	sp.Style = lipgloss.NewStyle().Foreground(t.Primary())
	return sp
}

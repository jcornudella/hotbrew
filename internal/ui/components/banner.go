package components

import (
	"strings"

	"github.com/jcornudella/hotbrew/internal/ui/gradient"
	"github.com/jcornudella/hotbrew/internal/ui/theme"
)

// ASCII art banner
const bannerArt = `
     ██████╗ ██╗ ██████╗ ███████╗███████╗████████╗
     ██╔══██╗██║██╔════╝ ██╔════╝██╔════╝╚══██╔══╝
     ██║  ██║██║██║  ███╗█████╗  ███████╗   ██║
     ██║  ██║██║██║   ██║██╔══╝  ╚════██║   ██║
     ██████╔╝██║╚██████╔╝███████╗███████║   ██║
     ╚═════╝ ╚═╝ ╚═════╝ ╚══════╝╚══════╝   ╚═╝
`

// Banner renders the ASCII art banner with gradient colors
func Banner(t theme.Theme) string {
	lines := strings.Split(bannerArt, "\n")
	colors := t.HeaderGradient()

	var result []string
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			result = append(result, "")
			continue
		}
		result = append(result, gradient.Text(line, colors))
	}

	return strings.Join(result, "\n")
}

// SmallBanner renders a smaller text banner
func SmallBanner(t theme.Theme) string {
	text := "░▒▓ DIGEST ▓▒░"
	return gradient.Bold(text, t.HeaderGradient())
}

// Tagline renders the app tagline
func Tagline(t theme.Theme) string {
	return t.MutedStyle().Render("Your personalized terminal newsletter")
}

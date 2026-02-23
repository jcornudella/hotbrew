package components

import (
	"strings"

	"github.com/jcornudella/hotbrew/internal/ui/gradient"
	"github.com/jcornudella/hotbrew/internal/ui/theme"
)

// hotbrewTitle is defined in header.go and shared across banner functions.

// Banner renders the HOTBREW wordmark with gradient colors
func Banner(t theme.Theme) string {
	wordmark := strings.Join(hotbrewTitle, "\n")
	return gradient.LineGradient(wordmark, t.HeaderGradient())
}

// SmallBanner renders a smaller text banner
func SmallBanner(t theme.Theme) string {
	text := "░▒▓ HOTBREW ▓▒░"
	return gradient.Bold(text, t.HeaderGradient())
}

// Tagline renders the app tagline
func Tagline(t theme.Theme) string {
	return t.MutedStyle().Render("Your morning, piping hot.")
}

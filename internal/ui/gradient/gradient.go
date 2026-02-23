package gradient

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// hexToRGB converts a hex color string to RGB values
func hexToRGB(hex string) (r, g, b int) {
	if len(hex) > 0 && hex[0] == '#' {
		hex = hex[1:]
	}
	if len(hex) != 6 {
		return 255, 255, 255
	}
	rVal, _ := strconv.ParseInt(hex[0:2], 16, 64)
	gVal, _ := strconv.ParseInt(hex[2:4], 16, 64)
	bVal, _ := strconv.ParseInt(hex[4:6], 16, 64)
	return int(rVal), int(gVal), int(bVal)
}

// rgbToHex converts RGB values to a hex color string
func rgbToHex(r, g, b int) string {
	return fmt.Sprintf("#%02x%02x%02x", r, g, b)
}

// interpolateColor blends between two colors based on t (0.0 to 1.0)
func interpolateColor(c1, c2 string, t float64) string {
	r1, g1, b1 := hexToRGB(c1)
	r2, g2, b2 := hexToRGB(c2)

	r := int(float64(r1) + t*(float64(r2)-float64(r1)))
	g := int(float64(g1) + t*(float64(g2)-float64(g1)))
	b := int(float64(b1) + t*(float64(b2)-float64(b1)))

	return rgbToHex(r, g, b)
}

// Text renders text with a gradient effect across characters
func Text(text string, colors []string) string {
	return applyGradient(text, colors, lipgloss.NewStyle())
}

// Bold renders bold text with a gradient effect
func Bold(text string, colors []string) string {
	base := lipgloss.NewStyle().Bold(true)
	return applyGradient(text, colors, base)
}

// LineGradient colors each line using the provided gradient stops.
func LineGradient(text string, colors []string) string {
	if len(colors) == 0 {
		return text
	}
	lines := strings.Split(text, "\n")
	if len(lines) == 0 {
		return ""
	}
	if len(lines) == 1 {
		color := gradientColorAt(colors, 0)
		return lipgloss.NewStyle().Foreground(lipgloss.Color(color)).Render(lines[0])
	}
	var rendered []string
	for i, line := range lines {
		pos := float64(i) / float64(len(lines)-1)
		color := gradientColorAt(colors, pos)
		style := lipgloss.NewStyle().Foreground(lipgloss.Color(color))
		rendered = append(rendered, style.Render(line))
	}
	return strings.Join(rendered, "\n")
}

func applyGradient(text string, colors []string, base lipgloss.Style) string {
	if len(colors) == 0 {
		return base.Render(text)
	}
	runes := []rune(text)
	if len(runes) == 0 {
		return ""
	}
	if len(colors) == 1 || len(runes) == 1 {
		return base.Foreground(lipgloss.Color(colors[0])).Render(text)
	}

	var builder strings.Builder
	currentColor := ""
	chunk := make([]rune, 0, len(runes))
	flush := func(color string) {
		if color == "" || len(chunk) == 0 {
			return
		}
		style := base.Foreground(lipgloss.Color(color))
		builder.WriteString(style.Render(string(chunk)))
		chunk = chunk[:0]
	}

	total := len(runes) - 1
	for i, r := range runes {
		pos := float64(i) / float64(total)
		color := gradientColorAt(colors, pos)
		if currentColor == "" {
			currentColor = color
		}
		if color != currentColor {
			flush(currentColor)
			currentColor = color
		}
		chunk = append(chunk, r)
	}
	flush(currentColor)

	return builder.String()
}

// ColorAt returns the interpolated color at a position (0.0 to 1.0) along the gradient.
func ColorAt(colors []string, position float64) string {
	return gradientColorAt(colors, position)
}

func gradientColorAt(colors []string, position float64) string {
	if len(colors) == 1 {
		return colors[0]
	}
	if position <= 0 {
		return colors[0]
	}
	if position >= 1 {
		return colors[len(colors)-1]
	}
	segments := len(colors) - 1
	segmentFloat := position * float64(segments)
	segment := int(segmentFloat)
	if segment >= segments {
		segment = segments - 1
	}
	segmentT := segmentFloat - float64(segment)
	return interpolateColor(colors[segment], colors[segment+1], segmentT)
}

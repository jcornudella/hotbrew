package gradient

import (
	"fmt"
	"strconv"

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
	if len(colors) == 0 {
		return text
	}
	if len(colors) == 1 {
		return lipgloss.NewStyle().Foreground(lipgloss.Color(colors[0])).Render(text)
	}

	runes := []rune(text)
	if len(runes) == 0 {
		return ""
	}
	if len(runes) == 1 {
		return lipgloss.NewStyle().Foreground(lipgloss.Color(colors[0])).Render(text)
	}

	result := ""
	segments := len(colors) - 1

	for i, r := range runes {
		// Calculate position in gradient (0.0 to 1.0)
		t := float64(i) / float64(len(runes)-1)

		// Find which color segment we're in
		segmentFloat := t * float64(segments)
		segment := int(segmentFloat)
		if segment >= segments {
			segment = segments - 1
		}

		// Calculate position within this segment
		segmentT := segmentFloat - float64(segment)

		// Interpolate between the two colors in this segment
		color := interpolateColor(colors[segment], colors[segment+1], segmentT)

		style := lipgloss.NewStyle().Foreground(lipgloss.Color(color))
		result += style.Render(string(r))
	}

	return result
}

// Bold renders bold text with a gradient effect
func Bold(text string, colors []string) string {
	if len(colors) == 0 {
		return lipgloss.NewStyle().Bold(true).Render(text)
	}
	if len(colors) == 1 {
		return lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(colors[0])).Render(text)
	}

	runes := []rune(text)
	if len(runes) == 0 {
		return ""
	}
	if len(runes) == 1 {
		return lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(colors[0])).Render(text)
	}

	result := ""
	segments := len(colors) - 1

	for i, r := range runes {
		t := float64(i) / float64(len(runes)-1)
		segmentFloat := t * float64(segments)
		segment := int(segmentFloat)
		if segment >= segments {
			segment = segments - 1
		}
		segmentT := segmentFloat - float64(segment)
		color := interpolateColor(colors[segment], colors[segment+1], segmentT)

		style := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(color))
		result += style.Render(string(r))
	}

	return result
}

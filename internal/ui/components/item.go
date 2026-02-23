package components

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/jcornudella/hotbrew/internal/sanitize"
	"github.com/jcornudella/hotbrew/internal/ui/theme"
	"github.com/jcornudella/hotbrew/pkg/source"
)

const itemGutterWidth = 4

var markdownLinkRegex = regexp.MustCompile(`\[([^\]]+)\]\(([^)]+)\)`)

// Item renders a single item in the digest with right-aligned timestamps.
func Item(item source.Item, t theme.Theme, width int, selected bool, isFirst bool, tag string, accent lipgloss.Color) string {
	prefix, _ := itemPadding(selected, accent)
	prefixWidth := lipgloss.Width(prefix)
	contentWidth := width - prefixWidth
	if contentWidth < 20 {
		contentWidth = width
	}

	timeText := formatTimeAgo(item.Timestamp)
	timestamp := ""
	timeWidth := 0
	if timeText != "" {
		timestamp = t.MutedStyle().Render(timeText)
		timeWidth = lipgloss.Width(timeText)
	}

	lineWidth := contentWidth
	if lineWidth < 0 {
		lineWidth = contentWidth
	}

	leftWidth := lineWidth - timeWidth - 2
	if timestamp == "" {
		leftWidth = lineWidth
	}
	if leftWidth < 12 {
		timestamp = ""
		leftWidth = lineWidth
		timeWidth = 0
	}
	if leftWidth < 4 {
		leftWidth = 4
	}

	cleanTag := sanitize.Text(tag)
	tagSegment := renderSourceTag(cleanTag, accent)
	tagWidth := 0
	if tagSegment != "" {
		tagWidth = lipgloss.Width(tagSegment) + 1
	}

	badge := priorityBadge(item.Priority, t)
	badgeWidth := 0
	if badge != "" {
		badgeWidth = lipgloss.Width(badge) + 1
	}

	titleMax := leftWidth - tagWidth - badgeWidth
	if titleMax < 8 {
		titleMax = leftWidth
	}
	if titleMax < 4 {
		titleMax = 4
	}

	cleanTitle := sanitize.Text(item.Title)
	titleText := truncate(cleanTitle, titleMax)
	titleStyle := lipgloss.NewStyle().Foreground(t.Text())
	if isFirst && !selected {
		titleStyle = titleStyle.Foreground(lipgloss.Color("#ffffff")).Bold(true)
	}
	if selected {
		titleStyle = lipgloss.NewStyle().Foreground(accent).Bold(true)
	}

	var segments []string
	if tagSegment != "" {
		segments = append(segments, tagSegment)
	}
	segments = append(segments, titleStyle.Render(titleText))
	if badge != "" {
		segments = append(segments, badge)
	}

	leftBlock := strings.Join(segments, " ")
	leftBlockWidth := lipgloss.Width(leftBlock)
	if timestamp != "" && leftBlockWidth >= leftWidth {
		timestamp = ""
	}
	spacing := leftWidth - leftBlockWidth
	if spacing < 1 {
		spacing = 1
	}

	line := leftBlock
	if timestamp != "" {
		paddingSpaces := spacing
		if paddingSpaces < 1 {
			paddingSpaces = 1
		}
		line = leftBlock + strings.Repeat(" ", paddingSpaces) + timestamp
	}

	return prefix + line
}

// ItemExpanded renders an item with its details inside a card.
func ItemExpanded(item source.Item, t theme.Theme, width int, selected bool, isFirst bool, tag string, accent lipgloss.Color) string {
	lines := []string{Item(item, t, width, selected, isFirst, tag, accent)}

	_, indent := itemPadding(false, accent)
	indentWidth := lipgloss.Width(indent)
	available := width - indentWidth
	if available < 20 {
		available = width
	}

	cardWidth := available
	innerWidth := cardWidth - 6
	if innerWidth < 12 {
		innerWidth = cardWidth - 4
	}

	cleanedBody, markdownLinks := stripMarkdownLinks(item.Body)
	cleanedBody = sanitize.Text(cleanedBody)
	for i, link := range markdownLinks {
		markdownLinks[i] = sanitize.Text(link)
	}

	var cardSections []string
	if subtitle := sanitize.Text(item.Subtitle); subtitle != "" {
		cardSections = append(cardSections, t.SubtitleStyle().Render(wrapText(subtitle, innerWidth)))
	}

	meta := buildMetadataLine(item)
	if meta != "" {
		cardSections = append(cardSections, t.MutedStyle().Render(meta))
	}

	if cleanedBody != "" {
		cardSections = append(cardSections, t.MutedStyle().Render(wrapText(cleanedBody, innerWidth)))
	}

	for _, link := range markdownLinks {
		cardSections = append(cardSections, t.AccentStyle().Render(link))
	}

	if url := sanitize.Text(item.URL); url != "" {
		cardSections = append(cardSections, t.AccentStyle().Render(url))
	}

	if len(cardSections) == 0 {
		return strings.Join(lines, "\n")
	}

	cardBody := strings.Join(cardSections, "\n\n")
	card := lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(accent).
		Padding(1, 2).
		Width(cardWidth).
		Render(cardBody)

	for _, line := range strings.Split(card, "\n") {
		lines = append(lines, indent+line)
	}

	return strings.Join(lines, "\n")
}

// priorityBadge returns a colored indicator based on priority
func priorityBadge(p source.Priority, t theme.Theme) string {
	var color lipgloss.Color
	var glyph string
	switch p {
	case source.Urgent:
		color = t.PriorityUrgent()
		glyph = "●"
	case source.High:
		color = t.PriorityHigh()
		glyph = "●"
	case source.Medium:
		color = t.PriorityMedium()
		glyph = "○"
	default:
		color = t.PriorityLow()
		glyph = "·"
	}
	return lipgloss.NewStyle().Foreground(color).Render(glyph)
}

// formatTimeAgo returns a human-readable time difference
func formatTimeAgo(timestamp time.Time) string {
	if timestamp.IsZero() {
		return ""
	}

	diff := time.Since(timestamp)
	switch {
	case diff < time.Minute:
		return "now"
	case diff < time.Hour:
		mins := int(diff.Minutes())
		return fmt.Sprintf("%dm", mins)
	case diff < 24*time.Hour:
		hours := int(diff.Hours())
		return fmt.Sprintf("%dh", hours)
	case diff < 7*24*time.Hour:
		days := int(diff.Hours() / 24)
		return fmt.Sprintf("%dd", days)
	default:
		return timestamp.Format("Jan 2")
	}
}

// truncate shortens a string to max length with ellipsis
func truncate(s string, max int) string {
	if max <= 0 {
		return ""
	}
	runes := []rune(s)
	if len(runes) <= max {
		return s
	}
	if max <= 3 {
		return string(runes[:max])
	}
	return string(runes[:max-3]) + "..."
}

func wrapText(text string, width int) string {
	if width <= 0 {
		return text
	}
	var lines []string
	paragraphs := strings.Split(text, "\n")
	for _, para := range paragraphs {
		para = strings.TrimSpace(para)
		if para == "" {
			lines = append(lines, "")
			continue
		}
		var current string
		for _, word := range strings.Fields(para) {
			if current == "" {
				current = word
				continue
			}
			if lipgloss.Width(current)+1+lipgloss.Width(word) > width {
				lines = append(lines, current)
				current = word
				continue
			}
			current += " " + word
		}
		if current != "" {
			lines = append(lines, current)
		}
	}
	return strings.Join(lines, "\n")
}

func stripMarkdownLinks(body string) (string, []string) {
	if body == "" {
		return body, nil
	}
	var links []string
	cleaned := markdownLinkRegex.ReplaceAllStringFunc(body, func(match string) string {
		parts := markdownLinkRegex.FindStringSubmatch(match)
		if len(parts) < 3 {
			return match
		}
		links = append(links, parts[2])
		return parts[1]
	})
	return cleaned, links
}

func buildMetadataLine(item source.Item) string {
	if item.Metadata == nil {
		return ""
	}

	meta := item.Metadata
	category := strings.ToLower(item.Category)
	switch category {
	case "hackernews":
		score := intFromAny(meta["score"])
		comments := intFromAny(meta["comments"])
		if score == 0 && comments == 0 {
			return ""
		}
		return sanitize.Text(fmt.Sprintf("▲ %s  •  💬 %s", formatCompact(score), formatCompact(comments)))
	case "discussion":
		points := intFromAny(meta["points"])
		comments := intFromAny(meta["comments"])
		author, _ := meta["author"].(string)
		var parts []string
		if points > 0 {
			parts = append(parts, fmt.Sprintf("▲ %s", formatCompact(points)))
		}
		if comments > 0 {
			parts = append(parts, fmt.Sprintf("💬 %s", formatCompact(comments)))
		}
		if author != "" {
			parts = append(parts, fmt.Sprintf("by %s", sanitize.Text(author)))
		}
		return sanitize.Text(strings.Join(parts, "  •  "))
	case "research":
		authors := stringSlice(meta["authors"])
		if len(authors) > 0 {
			if len(authors) > 3 {
				authors = append(authors[:3], "et al.")
			}
			return sanitize.Text("Authors: " + strings.Join(authors, ", "))
		}
	case "github":
		stars := intFromAny(meta["stars"])
		language, _ := meta["language"].(string)
		var parts []string
		if stars > 0 {
			parts = append(parts, fmt.Sprintf("⭐ %s", formatCompact(stars)))
		}
		if language != "" {
			parts = append(parts, sanitize.Text(language))
		}
		return sanitize.Text(strings.Join(parts, "  •  "))
	case "lobsters":
		score := intFromAny(meta["score"])
		comments := intFromAny(meta["comments"])
		var parts []string
		if score > 0 {
			parts = append(parts, fmt.Sprintf("▲ %s", formatCompact(score)))
		}
		if comments > 0 {
			parts = append(parts, fmt.Sprintf("💬 %s", formatCompact(comments)))
		}
		return sanitize.Text(strings.Join(parts, "  •  "))
	}

	if points := intFromAny(meta["points"]); points > 0 {
		parts := []string{fmt.Sprintf("▲ %s", formatCompact(points))}
		if comments := intFromAny(meta["comments"]); comments > 0 {
			parts = append(parts, fmt.Sprintf("💬 %s", formatCompact(comments)))
		}
		return sanitize.Text(strings.Join(parts, "  •  "))
	}

	return ""
}

func stringSlice(v any) []string {
	switch val := v.(type) {
	case []string:
		return val
	case []any:
		var out []string
		for _, entry := range val {
			if s, ok := entry.(string); ok {
				out = append(out, s)
			}
		}
		return out
	default:
		return nil
	}
}

func formatCompact(n int) string {
	switch {
	case n >= 1_000_000:
		return fmt.Sprintf("%.1fm", float64(n)/1_000_000)
	case n >= 1_000:
		return fmt.Sprintf("%.1fk", float64(n)/1_000)
	default:
		return fmt.Sprintf("%d", n)
	}
}

func intFromAny(v any) int {
	switch val := v.(type) {
	case int:
		return val
	case int64:
		return int(val)
	case float64:
		return int(val)
	case float32:
		return int(val)
	default:
		return 0
	}
}

func itemPadding(selected bool, accent lipgloss.Color) (string, string) {
	indent := strings.Repeat(" ", itemGutterWidth)
	if !selected {
		return indent, indent
	}
	marker := "  " + lipgloss.NewStyle().Foreground(accent).Render("▍") + " "
	marker = lipgloss.NewStyle().Width(itemGutterWidth).Render(marker)
	return marker, indent
}

func renderSourceTag(tag string, accent lipgloss.Color) string {
	if tag == "" {
		return ""
	}
	label := fmt.Sprintf("[%s]", sanitize.Text(tag))
	return lipgloss.NewStyle().Foreground(accent).Bold(true).Render(label)
}

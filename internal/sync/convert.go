// Package sync orchestrates fetching sources and storing TRSS items.
package sync

import (
	"time"

	"github.com/jcornudella/hotbrew/pkg/source"
	"github.com/jcornudella/hotbrew/pkg/trss"
)

// ConvertItem transforms a source.Item into a trss.Item.
func ConvertItem(item source.Item, src source.Source) trss.Item {
	canonical := trss.CanonicalURL(item.URL)

	var fingerprint string
	if canonical != "" {
		fingerprint = trss.Fingerprint(canonical)
	} else {
		fingerprint = trss.Fingerprint(trss.FallbackKey(item.Title, src.Name()))
	}

	id := trss.GenerateID(canonical)
	if canonical == "" {
		id = trss.GenerateID(trss.FallbackKey(item.Title, src.Name()))
	}

	// Map priority to a base score (0-10 range).
	score := priorityToScore(item.Priority)

	// Build engagement map from metadata if available.
	engagement := map[string]any{}
	if item.Metadata != nil {
		if v, ok := item.Metadata["points"]; ok {
			engagement["points"] = v
		}
		if v, ok := item.Metadata["comments"]; ok {
			engagement["comments"] = v
		}
		if v, ok := item.Metadata["stars"]; ok {
			engagement["stars"] = v
		}
	}

	// Extract tags from metadata.
	var tags []string
	if item.Metadata != nil {
		if v, ok := item.Metadata["tags"]; ok {
			if t, ok := v.([]string); ok {
				tags = t
			}
		}
		if v, ok := item.Metadata["language"]; ok {
			if lang, ok := v.(string); ok && lang != "" {
				tags = append(tags, lang)
			}
		}
		if item.Category != "" {
			tags = append(tags, item.Category)
		}
	}

	publishedAt := item.Timestamp
	if publishedAt.IsZero() {
		publishedAt = time.Now()
	}

	return trss.Item{
		ID:           id,
		Title:        item.Title,
		URL:          item.URL,
		URLCanonical: canonical,
		Source: trss.ItemSource{
			Name: src.Name(),
			Icon: src.Icon(),
		},
		PublishedAt:  publishedAt,
		FetchedAt:    time.Now(),
		Summary:      item.Subtitle,
		Body:         item.Body,
		Tags:         tags,
		Score:        score,
		Engagement:   engagement,
		Fingerprint:  fingerprint,
		Meta:         item.Metadata,
	}
}

// ConvertSection converts all items in a source.Section to trss.Items.
func ConvertSection(section *source.Section, src source.Source) []trss.Item {
	if section == nil {
		return nil
	}

	items := make([]trss.Item, 0, len(section.Items))
	for _, item := range section.Items {
		items = append(items, ConvertItem(item, src))
	}
	return items
}

// priorityToScore maps source.Priority to a numeric score.
func priorityToScore(p source.Priority) float64 {
	switch p {
	case source.Urgent:
		return 9.0
	case source.High:
		return 7.0
	case source.Medium:
		return 5.0
	default:
		return 3.0
	}
}

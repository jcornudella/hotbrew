package intel

import (
	"fmt"
	"strings"

	"github.com/jcornudella/hotbrew/pkg/source"
	"github.com/jcornudella/hotbrew/pkg/trss"
)

// FromTRSSItem converts a TRSS item into the internal intel representation.
func FromTRSSItem(item trss.Item) IntelItem {
	metadata := stringifyMap(item.Meta)
	author := metadata["author"]
	language := metadata["language"]

	return IntelItem{
		ID:           item.ID,
		CanonicalURL: firstNonEmpty(item.URLCanonical, item.URL),
		SourceKey:    item.Source.Via,
		SourceName:   item.Source.Name,
		SourceIcon:   item.Source.Icon,
		Title:        item.Title,
		Summary:      item.Summary,
		Body:         item.Body,
		Author:       author,
		PublishedAt:  item.PublishedAt,
		FetchedAt:    item.FetchedAt,
		Tags:         copyStrings(item.Tags),
		Language:     language,
		Engagement: EngagementSignals{
			Points:   getFloat(item.Engagement, "points"),
			Comments: getFloat(item.Engagement, "comments"),
			Stars:    getFloat(item.Engagement, "stars"),
			Forks:    getFloat(item.Engagement, "forks"),
			Velocity: getFloat(item.Engagement, "velocity"),
		},
		Metadata: metadata,
	}
}

// ToTRSSItem converts an intel item back into a TRSS item.
func ToTRSSItem(item IntelItem) trss.Item {
	meta := anyMap(item.Metadata)
	if item.Author != "" {
		meta["author"] = item.Author
	}
	if item.Language != "" {
		meta["language"] = item.Language
	}

	engagement := map[string]any{}
	if item.Engagement.Points != 0 {
		engagement["points"] = item.Engagement.Points
	}
	if item.Engagement.Comments != 0 {
		engagement["comments"] = item.Engagement.Comments
	}
	if item.Engagement.Stars != 0 {
		engagement["stars"] = item.Engagement.Stars
	}
	if item.Engagement.Forks != 0 {
		engagement["forks"] = item.Engagement.Forks
	}
	if item.Engagement.Velocity != 0 {
		engagement["velocity"] = item.Engagement.Velocity
	}
	if len(engagement) == 0 {
		engagement = nil
	}

	return trss.Item{
		ID:           item.ID,
		Title:        item.Title,
		URL:          item.CanonicalURL,
		URLCanonical: item.CanonicalURL,
		Source: trss.ItemSource{
			Name: item.SourceName,
			Icon: item.SourceIcon,
			Via:  item.SourceKey,
		},
		PublishedAt: item.PublishedAt,
		FetchedAt:   item.FetchedAt,
		Summary:     item.Summary,
		Body:        item.Body,
		Tags:        copyStrings(item.Tags),
		Engagement:  engagement,
		Meta:        meta,
	}
}

// FromSourceItem converts a source item into the internal intel representation.
func FromSourceItem(item source.Item, sourceName string, sourceIcon string) IntelItem {
	metadata := stringifyMap(item.Metadata)
	author := metadata["author"]

	return IntelItem{
		ID:           item.ID,
		CanonicalURL: item.URL,
		SourceKey:    strings.ToLower(item.Category),
		SourceName:   sourceName,
		SourceIcon:   sourceIcon,
		Title:        item.Title,
		Summary:      item.Subtitle,
		Body:         item.Body,
		Author:       author,
		PublishedAt:  item.Timestamp,
		Tags:         nil,
		Engagement: EngagementSignals{
			Points:   getFloat(item.Metadata, "score"),
			Comments: getFloat(item.Metadata, "comments"),
			Stars:    getFloat(item.Metadata, "stars"),
			Forks:    getFloat(item.Metadata, "forks"),
			Velocity: getFloat(item.Metadata, "velocity"),
		},
		Metadata: metadata,
	}
}

func stringifyMap(m map[string]any) map[string]string {
	if len(m) == 0 {
		return map[string]string{}
	}
	result := make(map[string]string, len(m))
	for k, v := range m {
		result[k] = fmt.Sprint(v)
	}
	return result
}

func anyMap(m map[string]string) map[string]any {
	if len(m) == 0 {
		return map[string]any{}
	}
	result := make(map[string]any, len(m))
	for k, v := range m {
		result[k] = v
	}
	return result
}

func getFloat(m map[string]any, key string) float64 {
	if len(m) == 0 {
		return 0
	}
	v, ok := m[key]
	if !ok {
		return 0
	}
	return toFloat(v)
}

func toFloat(v any) float64 {
	switch n := v.(type) {
	case float64:
		return n
	case float32:
		return float64(n)
	case int:
		return float64(n)
	case int8:
		return float64(n)
	case int16:
		return float64(n)
	case int32:
		return float64(n)
	case int64:
		return float64(n)
	case uint:
		return float64(n)
	case uint8:
		return float64(n)
	case uint16:
		return float64(n)
	case uint32:
		return float64(n)
	case uint64:
		return float64(n)
	default:
		return 0
	}
}

func copyStrings(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	copied := make([]string, len(values))
	copy(copied, values)
	return copied
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

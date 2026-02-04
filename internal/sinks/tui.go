package sinks

import (
	"github.com/jcornudella/hotbrew/pkg/source"
	"github.com/jcornudella/hotbrew/pkg/trss"
)

// DigestToSections converts a trss.Digest into source.Section slices
// for the existing TUI to consume.
func DigestToSections(d *trss.Digest) []*source.Section {
	if d == nil || len(d.Items) == 0 {
		return nil
	}

	// Group items by source
	sectionMap := map[string]*source.Section{}
	var order []string

	for _, item := range d.Items {
		name := item.Source.Name
		sec, ok := sectionMap[name]
		if !ok {
			sec = &source.Section{
				Name: name,
				Icon: item.Source.Icon,
			}
			sectionMap[name] = sec
			order = append(order, name)
		}

		sec.Items = append(sec.Items, trssItemToSourceItem(item))
	}

	sections := make([]*source.Section, 0, len(order))
	for _, name := range order {
		sections = append(sections, sectionMap[name])
	}
	return sections
}

// trssItemToSourceItem converts a trss.Item back to a source.Item for the TUI.
func trssItemToSourceItem(item trss.Item) source.Item {
	priority := source.Low
	switch {
	case item.Score >= 7:
		priority = source.Urgent
	case item.Score >= 5:
		priority = source.High
	case item.Score >= 3:
		priority = source.Medium
	}

	meta := item.Meta
	if meta == nil {
		meta = map[string]any{}
	}
	meta["trss_id"] = item.ID
	meta["trss_score"] = item.Score

	return source.Item{
		ID:        item.ID,
		Title:     item.Title,
		Subtitle:  item.Summary,
		Body:      item.Body,
		URL:       item.URL,
		Priority:  priority,
		Timestamp: item.PublishedAt,
		Icon:      item.Source.Icon,
		Metadata:  meta,
		Actions: []source.Action{
			{Key: "o", Label: "open", Command: item.URL},
		},
	}
}

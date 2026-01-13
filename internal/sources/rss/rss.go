// Package rss provides an RSS/Atom feed source for digest
package rss

import (
	"context"
	"time"

	"github.com/jcornudella/digest/pkg/source"
	"github.com/mmcdole/gofeed"
)

// Source fetches items from RSS/Atom feeds
type Source struct {
	name string
	url  string
	icon string
}

// New creates a new RSS source
func New(name, url, icon string) *Source {
	if icon == "" {
		icon = "📰"
	}
	return &Source{
		name: name,
		url:  url,
		icon: icon,
	}
}

func (s *Source) Name() string { return s.name }
func (s *Source) Icon() string { return s.icon }
func (s *Source) TTL() time.Duration { return 15 * time.Minute }

func (s *Source) Fetch(ctx context.Context, cfg source.Config) (*source.Section, error) {
	parser := gofeed.NewParser()

	feed, err := parser.ParseURLWithContext(s.url, ctx)
	if err != nil {
		return nil, err
	}

	// Get max items from config, default to 5
	maxItems := 5
	if max, ok := cfg.Settings["max"].(int); ok {
		maxItems = max
	}

	items := make([]source.Item, 0, maxItems)
	for i, entry := range feed.Items {
		if i >= maxItems {
			break
		}

		timestamp := time.Now()
		if entry.PublishedParsed != nil {
			timestamp = *entry.PublishedParsed
		} else if entry.UpdatedParsed != nil {
			timestamp = *entry.UpdatedParsed
		}

		// Determine priority based on recency
		priority := source.Low
		age := time.Since(timestamp)
		switch {
		case age < 1*time.Hour:
			priority = source.High
		case age < 6*time.Hour:
			priority = source.Medium
		}

		items = append(items, source.Item{
			ID:        entry.GUID,
			Title:     entry.Title,
			Subtitle:  entry.Description,
			URL:       entry.Link,
			Timestamp: timestamp,
			Priority:  priority,
			Category:  "news",
			Icon:      s.icon,
			Actions: []source.Action{
				{Key: "o", Label: "open", Command: entry.Link},
			},
		})
	}

	return &source.Section{
		Name:     s.name,
		Icon:     s.icon,
		Priority: 50,
		Items:    items,
	}, nil
}

// Package tldr provides a TLDR newsletter RSS source connector.
package tldr

import (
	"context"
	"time"

	"github.com/jcornudella/hotbrew/pkg/source"
	"github.com/mmcdole/gofeed"
)

// Feed represents a TLDR newsletter feed.
type Feed struct {
	Name string
	URL  string
	Icon string
}

// AvailableFeeds lists the TLDR newsletters with RSS feeds.
var AvailableFeeds = []Feed{
	{Name: "TLDR AI", URL: "https://tldr.tech/api/rss/ai", Icon: "🧠"},
	{Name: "TLDR Tech", URL: "https://tldr.tech/api/rss/tech", Icon: "💻"},
	{Name: "TLDR Web Dev", URL: "https://tldr.tech/api/rss/webdev", Icon: "🌐"},
}

// Source fetches items from a TLDR newsletter RSS feed.
type Source struct {
	name string
	url  string
	icon string
}

// New creates a TLDR source for a specific newsletter.
func New(name, url, icon string) *Source {
	return &Source{name: name, url: url, icon: icon}
}

// NewAI creates the TLDR AI source.
func NewAI() *Source {
	return New("TLDR AI", "https://tldr.tech/api/rss/ai", "🧠")
}

// NewTech creates the TLDR Tech source.
func NewTech() *Source {
	return New("TLDR Tech", "https://tldr.tech/api/rss/tech", "💻")
}

func (s *Source) Name() string        { return s.name }
func (s *Source) Icon() string        { return s.icon }
func (s *Source) TTL() time.Duration  { return 30 * time.Minute }

func (s *Source) Fetch(ctx context.Context, cfg source.Config) (*source.Section, error) {
	maxItems := 8
	if max, ok := cfg.Settings["max"].(int); ok {
		maxItems = max
	}

	parser := gofeed.NewParser()
	feed, err := parser.ParseURLWithContext(s.url, ctx)
	if err != nil {
		return nil, err
	}

	var items []source.Item
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

		priority := source.Medium
		age := time.Since(timestamp)
		if age < 6*time.Hour {
			priority = source.High
		}

		// TLDR titles often contain emoji category indicators.
		items = append(items, source.Item{
			ID:        entry.GUID,
			Title:     entry.Title,
			Subtitle:  entry.Description,
			URL:       entry.Link,
			Priority:  priority,
			Timestamp: timestamp,
			Category:  "newsletter",
			Icon:      s.icon,
			Actions: []source.Action{
				{Key: "o", Label: "open", Command: entry.Link},
			},
			Metadata: map[string]any{
				"via": "TLDR",
			},
		})
	}

	return &source.Section{
		Name:     s.name,
		Icon:     s.icon,
		Priority: 35,
		Items:    items,
	}, nil
}

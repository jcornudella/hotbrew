// Package hnsearch provides a Hacker News search source for digest
package hnsearch

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/jcornudella/hotbrew/pkg/source"
)

const searchURL = "https://hn.algolia.com/api/v1/search"

// SearchResult represents the Algolia search response
type SearchResult struct {
	Hits []Hit `json:"hits"`
}

// Hit represents a single search result
type Hit struct {
	ObjectID    string `json:"objectID"`
	Title       string `json:"title"`
	URL         string `json:"url"`
	Points      int    `json:"points"`
	Author      string `json:"author"`
	CreatedAt   string `json:"created_at"`
	NumComments int    `json:"num_comments"`
	StoryID     int    `json:"story_id"`
}

// Source searches Hacker News for specific topics
type Source struct {
	queries []string
	name    string
	icon    string
}

// New creates a new HN Search source
func New(name string, queries []string, icon string) *Source {
	if icon == "" {
		icon = "🔍"
	}
	return &Source{
		name:    name,
		queries: queries,
		icon:    icon,
	}
}

func (s *Source) Name() string       { return s.name }
func (s *Source) Icon() string       { return s.icon }
func (s *Source) TTL() time.Duration { return 15 * time.Minute }

func (s *Source) Fetch(ctx context.Context, cfg source.Config) (*source.Section, error) {
	maxItems := 5
	if max, ok := cfg.Settings["max"].(int); ok {
		maxItems = max
	}

	// Search for each query and combine results
	seen := make(map[string]bool)
	var allItems []source.Item

	for _, query := range s.queries {
		hits, err := search(ctx, query, maxItems)
		if err != nil {
			continue
		}

		for _, hit := range hits {
			// Dedupe by object ID
			if seen[hit.ObjectID] {
				continue
			}
			seen[hit.ObjectID] = true

			timestamp, _ := time.Parse(time.RFC3339, hit.CreatedAt)
			if timestamp.IsZero() {
				timestamp = time.Now()
			}

			// Priority based on points
			priority := source.Low
			switch {
			case hit.Points > 200:
				priority = source.Urgent
			case hit.Points > 100:
				priority = source.High
			case hit.Points > 30:
				priority = source.Medium
			}

			storyID := hit.StoryID
			if storyID == 0 {
				// Parse from objectID if story_id not set
				fmt.Sscanf(hit.ObjectID, "%d", &storyID)
			}

			hnURL := fmt.Sprintf("https://news.ycombinator.com/item?id=%s", hit.ObjectID)
			articleURL := hit.URL
			if articleURL == "" {
				articleURL = hnURL
			}

			subtitle := fmt.Sprintf("%d points by %s • %d comments", hit.Points, hit.Author, hit.NumComments)

			allItems = append(allItems, source.Item{
				ID:        fmt.Sprintf("hns-%s", hit.ObjectID),
				Title:     hit.Title,
				Subtitle:  subtitle,
				URL:       articleURL,
				Timestamp: timestamp,
				Priority:  priority,
				Category:  "hackernews",
				Icon:      s.icon,
				Actions: []source.Action{
					{Key: "o", Label: "open article", Command: articleURL},
					{Key: "c", Label: "open comments", Command: hnURL},
				},
				Metadata: map[string]any{
					"points":   hit.Points,
					"comments": hit.NumComments,
					"hn_url":   hnURL,
					"query":    query,
				},
			})
		}
	}

	// Sort by points (highest first) and limit
	sortByPoints(allItems)
	if len(allItems) > maxItems {
		allItems = allItems[:maxItems]
	}

	return &source.Section{
		Name:     s.name,
		Icon:     s.icon,
		Priority: 20,
		Items:    allItems,
	}, nil
}

func search(ctx context.Context, query string, limit int) ([]Hit, error) {
	params := url.Values{}
	params.Set("query", query)
	params.Set("tags", "story")
	params.Set("hitsPerPage", fmt.Sprintf("%d", limit*2)) // Fetch extra for deduping
	params.Set("numericFilters", "points>10")             // Only stories with some traction

	reqURL := searchURL + "?" + params.Encode()
	req, err := http.NewRequestWithContext(ctx, "GET", reqURL, nil)
	if err != nil {
		return nil, err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result SearchResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	// Filter out empty titles
	var filtered []Hit
	for _, hit := range result.Hits {
		if strings.TrimSpace(hit.Title) != "" {
			filtered = append(filtered, hit)
		}
	}

	return filtered, nil
}

func sortByPoints(items []source.Item) {
	for i := 0; i < len(items)-1; i++ {
		for j := i + 1; j < len(items); j++ {
			pi, _ := items[i].Metadata["points"].(int)
			pj, _ := items[j].Metadata["points"].(int)
			if pj > pi {
				items[i], items[j] = items[j], items[i]
			}
		}
	}
}

// Package hackernews provides a Hacker News source for digest
package hackernews

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/jcornudella/hotbrew/pkg/source"
)

const (
	baseURL     = "https://hacker-news.firebaseio.com/v0"
	topStories  = baseURL + "/topstories.json"
	itemURL     = baseURL + "/item/%d.json"
)

// Story represents a HN story
type Story struct {
	ID          int    `json:"id"`
	Title       string `json:"title"`
	URL         string `json:"url"`
	Score       int    `json:"score"`
	By          string `json:"by"`
	Time        int64  `json:"time"`
	Descendants int    `json:"descendants"` // comment count
	Type        string `json:"type"`
}

// Source fetches top stories from Hacker News
type Source struct{}

func New() *Source {
	return &Source{}
}

func (s *Source) Name() string          { return "Hacker News" }
func (s *Source) Icon() string          { return "🔶" }
func (s *Source) TTL() time.Duration    { return 10 * time.Minute }

func (s *Source) Fetch(ctx context.Context, cfg source.Config) (*source.Section, error) {
	// Get max items from config
	maxItems := 8
	if max, ok := cfg.Settings["max"].(int); ok {
		maxItems = max
	}

	// Fetch top story IDs
	ids, err := fetchTopStoryIDs(ctx, maxItems)
	if err != nil {
		return nil, err
	}

	// Fetch stories in parallel
	stories := fetchStories(ctx, ids)

	// Convert to source.Items
	items := make([]source.Item, 0, len(stories))
	for _, story := range stories {
		if story == nil {
			continue
		}

		timestamp := time.Unix(story.Time, 0)

		// Priority based on score
		priority := source.Low
		switch {
		case story.Score > 500:
			priority = source.Urgent
		case story.Score > 200:
			priority = source.High
		case story.Score > 50:
			priority = source.Medium
		}

		// HN discussion URL
		hnURL := fmt.Sprintf("https://news.ycombinator.com/item?id=%d", story.ID)

		subtitle := fmt.Sprintf("%d points by %s • %d comments", story.Score, story.By, story.Descendants)

		url := story.URL
		if url == "" {
			url = hnURL
		}

		items = append(items, source.Item{
			ID:        fmt.Sprintf("hn-%d", story.ID),
			Title:     story.Title,
			Subtitle:  subtitle,
			URL:       url,
			Timestamp: timestamp,
			Priority:  priority,
			Category:  "hackernews",
			Icon:      "🔶",
			Actions: []source.Action{
				{Key: "o", Label: "open article", Command: url},
				{Key: "c", Label: "open comments", Command: hnURL},
			},
			Metadata: map[string]any{
				"score":    story.Score,
				"comments": story.Descendants,
				"hn_url":   hnURL,
			},
		})
	}

	return &source.Section{
		Name:     "Hacker News",
		Icon:     "🔶",
		Priority: 30,
		Items:    items,
	}, nil
}

func fetchTopStoryIDs(ctx context.Context, limit int) ([]int, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", topStories, nil)
	if err != nil {
		return nil, err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var ids []int
	if err := json.NewDecoder(resp.Body).Decode(&ids); err != nil {
		return nil, err
	}

	if len(ids) > limit {
		ids = ids[:limit]
	}

	return ids, nil
}

func fetchStories(ctx context.Context, ids []int) []*Story {
	stories := make([]*Story, len(ids))
	var wg sync.WaitGroup

	for i, id := range ids {
		wg.Add(1)
		go func(idx, storyID int) {
			defer wg.Done()
			story, err := fetchStory(ctx, storyID)
			if err == nil {
				stories[idx] = story
			}
		}(i, id)
	}

	wg.Wait()
	return stories
}

func fetchStory(ctx context.Context, id int) (*Story, error) {
	url := fmt.Sprintf(itemURL, id)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var story Story
	if err := json.NewDecoder(resp.Body).Decode(&story); err != nil {
		return nil, err
	}

	return &story, nil
}

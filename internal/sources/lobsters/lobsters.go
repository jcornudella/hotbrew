// Package lobsters provides a Lobste.rs source connector.
package lobsters

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/jcornudella/hotbrew/pkg/source"
)

type story struct {
	ShortID      string   `json:"short_id"`
	Title        string   `json:"title"`
	URL          string   `json:"url"`
	Score        int      `json:"score"`
	Flags        int      `json:"flags"`
	CommentCount int      `json:"comment_count"`
	Description  string   `json:"description_plain"`
	Submitter    string   `json:"submitter_user"`
	Tags         []string `json:"tags"`
	CreatedAt    string   `json:"created_at"`
	ShortIDURL   string   `json:"short_id_url"`
	CommentsURL  string   `json:"comments_url"`
}

// Source fetches stories from Lobste.rs.
type Source struct {
	name     string
	icon     string
	tags     []string // optional tag filter
	endpoint string
}

// New creates a Lobste.rs source.
// If tags are provided, only stories matching those tags are included.
func New(name string, tags []string, icon string) *Source {
	if icon == "" {
		icon = "🦞"
	}
	return &Source{
		name:     name,
		icon:     icon,
		tags:     tags,
		endpoint: "https://lobste.rs/hottest.json",
	}
}

func (s *Source) Name() string        { return s.name }
func (s *Source) Icon() string        { return s.icon }
func (s *Source) TTL() time.Duration  { return 15 * time.Minute }

func (s *Source) Fetch(ctx context.Context, cfg source.Config) (*source.Section, error) {
	maxItems := 10
	if max, ok := cfg.Settings["max"].(int); ok {
		maxItems = max
	}

	req, err := http.NewRequestWithContext(ctx, "GET", s.endpoint, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "hotbrew/1.0")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("lobsters: status %d", resp.StatusCode)
	}

	var stories []story
	if err := json.NewDecoder(resp.Body).Decode(&stories); err != nil {
		return nil, err
	}

	tagSet := map[string]bool{}
	for _, t := range s.tags {
		tagSet[t] = true
	}

	var items []source.Item
	for _, st := range stories {
		if len(items) >= maxItems {
			break
		}

		// Tag filter: if tags specified, story must match at least one.
		if len(tagSet) > 0 && !matchesAnyTag(st.Tags, tagSet) {
			continue
		}

		// Skip heavily flagged stories.
		if st.Flags > 2 {
			continue
		}

		timestamp, _ := time.Parse(time.RFC3339, st.CreatedAt)
		if timestamp.IsZero() {
			timestamp = time.Now()
		}

		priority := source.Low
		switch {
		case st.Score >= 30:
			priority = source.Urgent
		case st.Score >= 15:
			priority = source.High
		case st.Score >= 5:
			priority = source.Medium
		}

		items = append(items, source.Item{
			ID:       st.ShortID,
			Title:    st.Title,
			Subtitle: st.Description,
			URL:      st.URL,
			Priority: priority,
			Timestamp: timestamp,
			Category: "tech",
			Icon:     s.icon,
			Actions: []source.Action{
				{Key: "o", Label: "open", Command: st.URL},
				{Key: "c", Label: "comments", Command: st.CommentsURL},
			},
			Metadata: map[string]any{
				"points":       st.Score,
				"comments":     st.CommentCount,
				"tags":         st.Tags,
				"submitter":    st.Submitter,
				"comments_url": st.CommentsURL,
				"lobsters_url": st.ShortIDURL,
			},
		})
	}

	return &source.Section{
		Name:     s.name,
		Icon:     s.icon,
		Priority: 40,
		Items:    items,
	}, nil
}

func matchesAnyTag(storyTags []string, filter map[string]bool) bool {
	for _, t := range storyTags {
		if filter[t] {
			return true
		}
	}
	return false
}

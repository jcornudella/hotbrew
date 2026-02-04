// Package reddit provides a Reddit source connector using the public JSON API.
package reddit

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/jcornudella/hotbrew/pkg/source"
)

// redditListing is the top-level Reddit API response.
type redditListing struct {
	Data struct {
		Children []struct {
			Data redditPost `json:"data"`
		} `json:"children"`
	} `json:"data"`
}

type redditPost struct {
	ID            string  `json:"id"`
	Title         string  `json:"title"`
	URL           string  `json:"url"`
	Permalink     string  `json:"permalink"`
	Selftext      string  `json:"selftext"`
	Score         int     `json:"score"`
	NumComments   int     `json:"num_comments"`
	Subreddit     string  `json:"subreddit"`
	Author        string  `json:"author"`
	CreatedUTC    float64 `json:"created_utc"`
	LinkFlairText string  `json:"link_flair_text"`
	IsSelf        bool    `json:"is_self"`
	Domain        string  `json:"domain"`
	Ups           int     `json:"ups"`
}

// Source fetches posts from one or more subreddits.
type Source struct {
	name       string
	icon       string
	subreddits []string
	sort       string // "hot", "top", "new"
}

// New creates a Reddit source for the given subreddits.
func New(name string, subreddits []string, icon string) *Source {
	if icon == "" {
		icon = "🤖"
	}
	return &Source{
		name:       name,
		icon:       icon,
		subreddits: subreddits,
		sort:       "hot",
	}
}

func (s *Source) Name() string        { return s.name }
func (s *Source) Icon() string        { return s.icon }
func (s *Source) TTL() time.Duration  { return 15 * time.Minute }

func (s *Source) Fetch(ctx context.Context, cfg source.Config) (*source.Section, error) {
	maxItems := 8
	if max, ok := cfg.Settings["max"].(int); ok {
		maxItems = max
	}

	maxPerSub := maxItems
	if len(s.subreddits) > 1 {
		maxPerSub = (maxItems / len(s.subreddits)) + 1
	}

	var allItems []source.Item

	for _, sub := range s.subreddits {
		items, err := s.fetchSubreddit(ctx, sub, maxPerSub)
		if err != nil {
			continue // skip failed subs, don't fail the whole source
		}
		allItems = append(allItems, items...)
	}

	// Trim to max.
	if len(allItems) > maxItems {
		allItems = allItems[:maxItems]
	}

	return &source.Section{
		Name:     s.name,
		Icon:     s.icon,
		Priority: 45,
		Items:    allItems,
	}, nil
}

func (s *Source) fetchSubreddit(ctx context.Context, subreddit string, limit int) ([]source.Item, error) {
	url := fmt.Sprintf("https://www.reddit.com/r/%s/%s.json?limit=%d&raw_json=1",
		subreddit, s.sort, limit)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}
	// Reddit requires a descriptive User-Agent, blocks generic ones.
	req.Header.Set("User-Agent", "hotbrew:v1.0 (terminal-rss)")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("reddit r/%s: status %d", subreddit, resp.StatusCode)
	}

	var listing redditListing
	if err := json.NewDecoder(resp.Body).Decode(&listing); err != nil {
		return nil, err
	}

	var items []source.Item
	for _, child := range listing.Data.Children {
		post := child.Data

		// Skip stickied/pinned mod posts (they tend to be low-value).
		if post.Score < 2 {
			continue
		}

		timestamp := time.Unix(int64(post.CreatedUTC), 0)

		priority := source.Low
		switch {
		case post.Score >= 500:
			priority = source.Urgent
		case post.Score >= 100:
			priority = source.High
		case post.Score >= 20:
			priority = source.Medium
		}

		// For self posts, link to the Reddit thread.
		itemURL := post.URL
		commentsURL := "https://www.reddit.com" + post.Permalink
		if post.IsSelf {
			itemURL = commentsURL
		}

		subtitle := ""
		if post.Selftext != "" {
			subtitle = post.Selftext
			if len(subtitle) > 300 {
				subtitle = subtitle[:297] + "..."
			}
		}

		tags := []string{post.Subreddit}
		if post.LinkFlairText != "" {
			tags = append(tags, post.LinkFlairText)
		}

		items = append(items, source.Item{
			ID:       post.ID,
			Title:    post.Title,
			Subtitle: subtitle,
			URL:      itemURL,
			Priority: priority,
			Timestamp: timestamp,
			Category: "discussion",
			Icon:     s.icon,
			Actions: []source.Action{
				{Key: "o", Label: "open", Command: itemURL},
				{Key: "c", Label: "comments", Command: commentsURL},
			},
			Metadata: map[string]any{
				"points":       post.Score,
				"comments":     post.NumComments,
				"author":       post.Author,
				"subreddit":    post.Subreddit,
				"domain":       post.Domain,
				"tags":         tags,
				"comments_url": commentsURL,
			},
		})
	}

	return items, nil
}

// Package github provides a GitHub trending source for digest
package github

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"time"

	"github.com/jcornudella/hotbrew/pkg/source"
)

const searchURL = "https://api.github.com/search/repositories"

// SearchResult represents the GitHub search response
type SearchResult struct {
	Items []Repo `json:"items"`
}

// Repo represents a GitHub repository
type Repo struct {
	ID          int64    `json:"id"`
	Name        string   `json:"name"`
	FullName    string   `json:"full_name"`
	Description string   `json:"description"`
	HTMLURL     string   `json:"html_url"`
	Stars       int      `json:"stargazers_count"`
	Forks       int      `json:"forks_count"`
	Language    string   `json:"language"`
	CreatedAt   string   `json:"created_at"`
	UpdatedAt   string   `json:"updated_at"`
	Topics      []string `json:"topics"`
	Owner       Owner    `json:"owner"`
}

// Owner represents a repo owner
type Owner struct {
	Login     string `json:"login"`
	AvatarURL string `json:"avatar_url"`
}

// Source fetches trending repos from GitHub
type Source struct {
	name   string
	topics []string
	icon   string
}

// New creates a new GitHub trending source
func New(name string, topics []string, icon string) *Source {
	if icon == "" {
		icon = "⭐"
	}
	return &Source{
		name:   name,
		topics: topics,
		icon:   icon,
	}
}

func (s *Source) Name() string       { return s.name }
func (s *Source) Icon() string       { return s.icon }
func (s *Source) TTL() time.Duration { return 30 * time.Minute }

func (s *Source) Fetch(ctx context.Context, cfg source.Config) (*source.Section, error) {
	maxItems := 8
	if max, ok := cfg.Settings["max"].(int); ok {
		maxItems = max
	}

	// Time range: repos created or pushed in the last week
	since := time.Now().AddDate(0, 0, -7).Format("2006-01-02")

	// GitHub's Search API does not parse `(topic:a OR topic:b ...)` the
	// way you'd expect — a grouped OR across topic qualifiers returns
	// zero results even when each clause individually matches
	// thousands. We fan out one request per topic and merge, which
	// also gives us a cap of len(topics) requests (≤ unauth 60/hr).
	baseQuery := fmt.Sprintf("pushed:>%s stars:>50", since)
	queries := []string{baseQuery}
	if len(s.topics) > 0 {
		queries = queries[:0]
		for _, topic := range s.topics {
			queries = append(queries, fmt.Sprintf("topic:%s %s", topic, baseQuery))
		}
	}

	repos, err := searchReposMerged(ctx, queries, maxItems)
	if err != nil {
		return nil, err
	}

	items := make([]source.Item, 0, len(repos))
	for _, repo := range repos {
		timestamp, _ := time.Parse(time.RFC3339, repo.UpdatedAt)
		if timestamp.IsZero() {
			timestamp = time.Now()
		}

		// Priority based on stars
		priority := source.Low
		switch {
		case repo.Stars > 5000:
			priority = source.Urgent
		case repo.Stars > 1000:
			priority = source.High
		case repo.Stars > 200:
			priority = source.Medium
		}

		// Language emoji
		langIcon := getLanguageIcon(repo.Language)

		subtitle := fmt.Sprintf("%s ⭐ %s  •  %s", langIcon, formatNumber(repo.Stars), repo.Owner.Login)

		// Truncate description
		desc := repo.Description
		if len(desc) > 80 {
			desc = desc[:77] + "..."
		}

		items = append(items, source.Item{
			ID:        fmt.Sprintf("gh-%d", repo.ID),
			Title:     repo.FullName,
			Subtitle:  subtitle,
			Body:      desc,
			URL:       repo.HTMLURL,
			Timestamp: timestamp,
			Priority:  priority,
			Category:  "github",
			Icon:      langIcon,
			Actions: []source.Action{
				{Key: "o", Label: "open repo", Command: repo.HTMLURL},
			},
			Metadata: map[string]any{
				"stars":    repo.Stars,
				"forks":    repo.Forks,
				"language": repo.Language,
				"topics":   repo.Topics,
			},
		})
	}

	return &source.Section{
		Name:     s.name,
		Icon:     s.icon,
		Priority: 25,
		Items:    items,
	}, nil
}

// searchReposMerged runs each query in sequence, dedupes by repo ID,
// sorts by stars descending, and trims to limit. Keeps the stars-sort
// contract the Fetch code assumes, even when results span topics.
func searchReposMerged(ctx context.Context, queries []string, limit int) ([]Repo, error) {
	seen := make(map[int64]struct{})
	merged := make([]Repo, 0, limit*len(queries))
	for _, q := range queries {
		repos, err := searchRepos(ctx, q, limit)
		if err != nil {
			return nil, err
		}
		for _, r := range repos {
			if _, dup := seen[r.ID]; dup {
				continue
			}
			seen[r.ID] = struct{}{}
			merged = append(merged, r)
		}
	}
	sort.SliceStable(merged, func(i, j int) bool {
		return merged[i].Stars > merged[j].Stars
	})
	if len(merged) > limit {
		merged = merged[:limit]
	}
	return merged, nil
}

func searchRepos(ctx context.Context, query string, limit int) ([]Repo, error) {
	params := url.Values{}
	params.Set("q", query)
	params.Set("sort", "stars")
	params.Set("order", "desc")
	params.Set("per_page", fmt.Sprintf("%d", limit))

	reqURL := searchURL + "?" + params.Encode()
	req, err := http.NewRequestWithContext(ctx, "GET", reqURL, nil)
	if err != nil {
		return nil, err
	}

	// GitHub API requires User-Agent
	req.Header.Set("User-Agent", "hotbrew-cli")
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("GitHub API returned %d", resp.StatusCode)
	}

	var result SearchResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return result.Items, nil
}

func getLanguageIcon(lang string) string {
	icons := map[string]string{
		"Go":         "🐹",
		"Python":     "🐍",
		"JavaScript": "💛",
		"TypeScript": "💙",
		"Rust":       "🦀",
		"Java":       "☕",
		"C++":        "⚡",
		"C":          "⚙️",
		"Ruby":       "💎",
		"Swift":      "🍎",
		"Kotlin":     "🟣",
		"Zig":        "⚡",
		"Shell":      "🐚",
		"Lua":        "🌙",
	}
	if icon, ok := icons[lang]; ok {
		return icon
	}
	return "📦"
}

func formatNumber(n int) string {
	if n >= 1000 {
		return fmt.Sprintf("%.1fk", float64(n)/1000)
	}
	return fmt.Sprintf("%d", n)
}

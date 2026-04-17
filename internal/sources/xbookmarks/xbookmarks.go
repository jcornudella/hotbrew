// Package xbookmarks provides a source connector for X (Twitter) bookmarks.
package xbookmarks

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/jcornudella/hotbrew/pkg/source"
)

const (
	tokenFilename = "x_token.json"
	defaultMax    = 20
	pageSize      = 100
)

// Source fetches a user's X bookmarks.
type Source struct {
	name     string
	icon     string
	clientID string
}

// New creates an X bookmarks source.
func New(name, icon string) *Source {
	return &Source{
		name: displayName(name),
		icon: displayIcon(icon),
	}
}

// WithClientID lets the caller inject the OAuth client id for refresh
// flows. Empty is fine — refresh is skipped and the user is asked to
// re-run `hotbrew x-auth` on 401.
func (s *Source) WithClientID(id string) *Source {
	s.clientID = id
	return s
}

func displayName(name string) string {
	if name == "" {
		return "X Bookmarks"
	}
	return name
}

func displayIcon(icon string) string {
	if icon == "" {
		return "🔖"
	}
	return icon
}

// Name implements source.Source.
func (s *Source) Name() string { return s.name }

// Icon implements source.Source.
func (s *Source) Icon() string { return s.icon }

// TTL implements source.Source.
func (s *Source) TTL() time.Duration { return 30 * time.Minute }

// Fetch returns the user's recent bookmarks. Returns a "needs auth"
// item when no token is on disk.
func (s *Source) Fetch(ctx context.Context, cfg source.Config) (*source.Section, error) {
	tok, err := LoadToken()
	if err != nil {
		return nil, fmt.Errorf("load token: %w", err)
	}
	if tok == nil {
		return s.needsAuthSection(), nil
	}

	maxItems := defaultMax
	if m, ok := cfg.Settings["max"].(int); ok && m > 0 {
		maxItems = m
	}

	accessToken, err := s.ensureFreshToken(ctx, tok)
	if err != nil {
		return nil, fmt.Errorf("refresh token: %w", err)
	}

	cur, err := loadCursor()
	if err != nil {
		return nil, fmt.Errorf("load cursor: %w", err)
	}

	if cur.UserID == "" {
		id, err := fetchMe(ctx, accessToken)
		if err != nil {
			if isUnauthorized(err) {
				return s.needsAuthSection(), nil
			}
			return nil, fmt.Errorf("identify user: %w", err)
		}
		cur.UserID = id
	}

	tweets, users, newestSeen, err := s.drainBookmarks(ctx, accessToken, cur, maxItems)
	if err != nil {
		if isUnauthorized(err) {
			return s.needsAuthSection(), nil
		}
		return nil, err
	}

	if newestSeen != "" {
		cur.NewestID = newestSeen
	}
	if err := saveCursor(cur); err != nil {
		return nil, fmt.Errorf("save cursor: %w", err)
	}

	return s.buildSection(tweets, users), nil
}

// drainBookmarks pulls pages until we find a tweet whose id matches the
// persisted NewestID, hit maxItems, or run out of pages.
func (s *Source) drainBookmarks(ctx context.Context, accessToken string, cur *cursor, maxItems int) ([]tweet, map[string]userRef, string, error) {
	var collected []tweet
	userIdx := map[string]userRef{}
	var newestSeen string
	var pageToken string

	for {
		resp, err := fetchBookmarksPage(ctx, accessToken, cur.UserID, pageToken, pageSize)
		if err != nil {
			return nil, nil, "", err
		}
		for _, u := range resp.Includes.Users {
			userIdx[u.ID] = u
		}
		for _, tw := range resp.Data {
			if newestSeen == "" {
				newestSeen = tw.ID
			}
			if cur.NewestID != "" && tw.ID == cur.NewestID {
				return collected, userIdx, newestSeen, nil
			}
			collected = append(collected, tw)
			if len(collected) >= maxItems {
				return collected, userIdx, newestSeen, nil
			}
		}
		if resp.Meta.NextToken == "" {
			return collected, userIdx, newestSeen, nil
		}
		pageToken = resp.Meta.NextToken
	}
}

func (s *Source) buildSection(tweets []tweet, users map[string]userRef) *source.Section {
	items := make([]source.Item, 0, len(tweets))
	for _, tw := range tweets {
		items = append(items, tweetToItem(tw, users[tw.AuthorID], s.icon))
	}
	return &source.Section{
		Name:  s.name,
		Icon:  s.icon,
		Items: items,
	}
}

func tweetToItem(tw tweet, author userRef, icon string) source.Item {
	title := collapseWhitespace(tw.Text)
	if len(title) > 140 {
		title = title[:137] + "…"
	}

	tweetURL := fmt.Sprintf("https://x.com/%s/status/%s", fallbackUsername(author), tw.ID)
	linkedURL := firstExternalURL(tw)

	subtitle := ""
	if author.Username != "" {
		subtitle = "@" + author.Username
	}

	return source.Item{
		ID:        "x:" + tw.ID,
		Title:     title,
		Subtitle:  subtitle,
		URL:       tweetURL,
		Category:  "bookmark",
		Timestamp: tw.CreatedAt,
		Icon:      icon,
		Priority:  priorityFor(tw.PublicMetrics.LikeCount + tw.PublicMetrics.RetweetCount),
		Metadata: map[string]any{
			"likes":    tw.PublicMetrics.LikeCount,
			"retweets": tw.PublicMetrics.RetweetCount,
			"replies":  tw.PublicMetrics.ReplyCount,
			"link":     linkedURL,
			"author":   author.Name,
		},
	}
}

func priorityFor(engagement int) source.Priority {
	switch {
	case engagement >= 5000:
		return source.Urgent
	case engagement >= 500:
		return source.High
	case engagement >= 50:
		return source.Medium
	default:
		return source.Low
	}
}

func firstExternalURL(tw tweet) string {
	for _, u := range tw.Entities.URLs {
		if u.ExpandedURL != "" && !strings.Contains(u.ExpandedURL, "x.com/") && !strings.Contains(u.ExpandedURL, "twitter.com/") {
			return u.ExpandedURL
		}
	}
	return ""
}

func fallbackUsername(u userRef) string {
	if u.Username != "" {
		return u.Username
	}
	return "i"
}

func collapseWhitespace(s string) string {
	return strings.Join(strings.Fields(s), " ")
}

// ensureFreshToken refreshes the access token if it's expired/near expiry
// and the refresh token + client id are available.
func (s *Source) ensureFreshToken(ctx context.Context, tok *Token) (string, error) {
	if tok.Valid() {
		return tok.AccessToken, nil
	}
	if tok.RefreshToken == "" || s.clientID == "" {
		return tok.AccessToken, nil
	}
	fresh, err := RefreshToken(ctx, s.clientID, tok.RefreshToken)
	if err != nil {
		return "", err
	}
	if err := SaveToken(fresh); err != nil {
		return "", err
	}
	return fresh.AccessToken, nil
}

func isUnauthorized(err error) bool {
	var apiErr *apiError
	if errors.As(err, &apiErr) {
		return apiErr.Status == 401
	}
	return false
}

func (s *Source) needsAuthSection() *source.Section {
	return &source.Section{
		Name: s.name,
		Icon: s.icon,
		Items: []source.Item{
			{
				ID:       "xbookmarks-needs-auth",
				Title:    "Connect your X account",
				Subtitle: "Run `hotbrew x-auth` to authorize hotbrew to read your bookmarks",
				Priority: source.Medium,
				Category: "auth",
				Icon:     s.icon,
			},
		},
	}
}

// TokenPath returns the on-disk location of the stored OAuth token.
func TokenPath() string {
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		return filepath.Join(xdg, "hotbrew", tokenFilename)
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "hotbrew", tokenFilename)
}

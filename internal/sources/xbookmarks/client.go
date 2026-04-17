package xbookmarks

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

const (
	apiBase    = "https://api.x.com/2"
	meEndpoint = apiBase + "/users/me"
)

// tweet is the subset of the /2 tweet payload we care about.
type tweet struct {
	ID            string    `json:"id"`
	Text          string    `json:"text"`
	AuthorID      string    `json:"author_id"`
	CreatedAt     time.Time `json:"created_at"`
	PublicMetrics struct {
		LikeCount    int `json:"like_count"`
		RetweetCount int `json:"retweet_count"`
		ReplyCount   int `json:"reply_count"`
		QuoteCount   int `json:"quote_count"`
	} `json:"public_metrics"`
	Entities struct {
		URLs []struct {
			ExpandedURL string `json:"expanded_url"`
		} `json:"urls"`
	} `json:"entities"`
}

type userRef struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Username string `json:"username"`
}

type bookmarksResponse struct {
	Data     []tweet `json:"data"`
	Includes struct {
		Users []userRef `json:"users"`
	} `json:"includes"`
	Meta struct {
		ResultCount   int    `json:"result_count"`
		NextToken     string `json:"next_token"`
		PreviousToken string `json:"previous_token"`
	} `json:"meta"`
}

type meResponse struct {
	Data userRef `json:"data"`
}

// fetchMe returns the authenticated user's id. Cached by the caller so
// we don't re-hit this endpoint on every sync.
func fetchMe(ctx context.Context, accessToken string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, meEndpoint, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	body, err := doRequest(req)
	if err != nil {
		return "", err
	}
	var out meResponse
	if err := json.Unmarshal(body, &out); err != nil {
		return "", fmt.Errorf("decode /users/me: %w", err)
	}
	return out.Data.ID, nil
}

// fetchBookmarksPage pulls a single page. paginationToken may be empty.
func fetchBookmarksPage(ctx context.Context, accessToken, userID, paginationToken string, pageSize int) (*bookmarksResponse, error) {
	q := url.Values{}
	q.Set("max_results", strconv.Itoa(pageSize))
	q.Set("tweet.fields", "created_at,public_metrics,entities,author_id")
	q.Set("expansions", "author_id")
	q.Set("user.fields", "name,username")
	if paginationToken != "" {
		q.Set("pagination_token", paginationToken)
	}
	endpoint := fmt.Sprintf("%s/users/%s/bookmarks?%s", apiBase, userID, q.Encode())

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	body, err := doRequest(req)
	if err != nil {
		return nil, err
	}
	var out bookmarksResponse
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, fmt.Errorf("decode bookmarks: %w", err)
	}
	return &out, nil
}

func doRequest(req *http.Request) ([]byte, error) {
	client := &http.Client{Timeout: 20 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, &apiError{Status: resp.StatusCode, Body: string(body)}
	}
	return body, nil
}

// apiError captures non-200 responses so the driver can distinguish
// 401 (expired token → refresh) from other failures.
type apiError struct {
	Status int
	Body   string
}

func (e *apiError) Error() string {
	return fmt.Sprintf("x api %d: %s", e.Status, e.Body)
}

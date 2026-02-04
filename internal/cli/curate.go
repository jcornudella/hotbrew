package cli

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/jcornudella/hotbrew/internal/store"
	"github.com/jcornudella/hotbrew/pkg/trss"
)

// CurateOptions holds options for the curate command.
type CurateOptions struct {
	URL   string
	Title string
	Tags  []string
	Note  string
}

// Curate handles `hotbrew curate <url>` — manually save a link.
func Curate(st *store.Store, opts CurateOptions) {
	if opts.URL == "" {
		fmt.Println("Usage: hotbrew curate <url> [--title \"...\"] [--tags ai,coding] [--note \"...\"]")
		fmt.Println("\nExamples:")
		fmt.Println("  hotbrew curate https://example.com/great-article")
		fmt.Println("  hotbrew curate https://x.com/user/status/123 --title \"Great thread on AI\"")
		fmt.Println("  hotbrew curate https://arxiv.org/abs/2401.00001 --tags ai,paper")
		os.Exit(1)
	}

	// Auto-fetch title if not provided.
	if opts.Title == "" {
		fmt.Print("  Fetching title... ")
		opts.Title = fetchTitle(opts.URL)
		if opts.Title != "" {
			fmt.Printf("%s\n", opts.Title)
		} else {
			fmt.Println("(could not fetch, using URL)")
			opts.Title = opts.URL
		}
	}

	canonical := trss.CanonicalURL(opts.URL)
	fingerprint := trss.Fingerprint(canonical)
	id := trss.GenerateID(canonical)

	item := trss.Item{
		ID:           id,
		Title:        opts.Title,
		URL:          opts.URL,
		URLCanonical: canonical,
		Source: trss.ItemSource{
			Name: "Curated",
			Icon: "📌",
		},
		PublishedAt: time.Now(),
		FetchedAt:   time.Now(),
		Summary:     opts.Note,
		Tags:        opts.Tags,
		Score:       8.0, // Manually curated items get a high base score.
		Fingerprint: fingerprint,
		Meta: map[string]any{
			"curated": true,
		},
	}

	sourceID, err := st.GetOrCreateSource("Curated", "manual", "", "📌")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating source: %v\n", err)
		os.Exit(1)
	}

	if err := st.InsertItem(item, sourceID); err != nil {
		fmt.Fprintf(os.Stderr, "Error saving item: %v\n", err)
		os.Exit(1)
	}

	// Also mark it as saved.
	st.MarkSaved(item.ID)

	fmt.Printf("📌 Curated: %s\n", opts.Title)
	fmt.Printf("   ID: %s\n", id)
	if len(opts.Tags) > 0 {
		fmt.Printf("   Tags: %s\n", strings.Join(opts.Tags, ", "))
	}
}

// fetchTitle tries to extract the <title> from a URL.
func fetchTitle(url string) string {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return ""
	}
	req.Header.Set("User-Agent", "hotbrew/1.0")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()

	// Read up to 64KB to find the title tag.
	body, err := io.ReadAll(io.LimitReader(resp.Body, 64*1024))
	if err != nil {
		return ""
	}

	return extractTitle(string(body))
}

var titleRe = regexp.MustCompile(`(?i)<title[^>]*>([^<]+)</title>`)

func extractTitle(html string) string {
	matches := titleRe.FindStringSubmatch(html)
	if len(matches) < 2 {
		return ""
	}
	title := strings.TrimSpace(matches[1])
	// Decode common HTML entities.
	title = strings.ReplaceAll(title, "&amp;", "&")
	title = strings.ReplaceAll(title, "&lt;", "<")
	title = strings.ReplaceAll(title, "&gt;", ">")
	title = strings.ReplaceAll(title, "&#39;", "'")
	title = strings.ReplaceAll(title, "&quot;", "\"")
	return title
}

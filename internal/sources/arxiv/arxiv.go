// Package arxiv provides an arXiv paper source connector.
// Inspired by github.com/jcornudella/llm-research-digest — fetches recent
// papers from key CS categories and ranks by keyword relevance.
package arxiv

import (
	"context"
	"encoding/xml"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/jcornudella/hotbrew/pkg/source"
)

// Default categories matching llm-research-digest.
var DefaultCategories = []string{"cs.CL", "cs.AI", "cs.LG", "cs.MA"}

// Relevance keywords — expanded from llm-research-digest with more
// practical LLM-building terms.
var relevancePatterns = []string{
	// Core LLM terms
	`\bllm\b`, `\blarge language model`, `\bfoundation model`,
	`\btransformer\b`, `\battention\b`, `\bpre.?train`,

	// Agent/agentic
	`\bagent\b`, `\bagentic\b`, `\bmulti.?agent\b`,
	`\btool\s*(use|call|ing)\b`, `\bfunction call`,
	`\bplanning\b`, `\borchestrat`, `\bworkflow\b`,

	// Reasoning
	`\breason(ing)?\b`, `chain.of.thought`, `\bcot\b`,
	`\bthink(ing)?\b`, `\bself.?reflect`, `\bverif`,

	// RAG + retrieval
	`\brag\b`, `\bretrieval`, `\bvector`, `\bembedding`,
	`\bknowledge.?(graph|base)\b`, `\bsemantic.?search`,

	// Prompting + alignment
	`\bprompt`, `\binstruct`, `\bfine.?tun`, `\balign`,
	`\brlhf\b`, `\bdpo\b`, `\breinforcement`,
	`\bin.?context.?learn`, `\bfew.?shot`, `\bzero.?shot`,

	// Model names (signal relevance)
	`\bgpt\b`, `\bclaude\b`, `\bllama\b`, `\bgemini\b`,
	`\bmistral\b`, `\bqwen\b`, `\bdeepseek\b`,
	`\banthropic\b`, `\bopen.?ai\b`,

	// Practical building
	`\binference\b`, `\bserving\b`, `\blatency\b`,
	`\bquantiz`, `\bdistill`, `\bprun`,
	`\bcontext.?window\b`, `\blong.?context`,
	`\bscaling\b`, `\befficien`,
	`\bbenchmark`, `\bevaluat`,

	// Code + coding
	`\bcode.?gen`, `\bcoding\b`, `\bprogram.?synth`,
	`\bsoftware.?eng`, `\bdebug`,

	// Safety
	`\bhallucin`, `\bground(ing|ed)\b`, `\bfaithful`,
	`\bsafety\b`, `\bjailbreak\b`, `\bred.?team`,

	// Memory + context
	`\bmemory\b`, `\bchat\b`, `\bconversat`,
	`\bsummariz`, `\bcompress`,

	// Multimodal
	`\bmultimodal\b`, `\bvision.?language\b`, `\bvlm\b`,

	// Deployment
	`\bapi\b`, `\bdeployment\b`, `\bproduction\b`,
	`\bcost\b`, `\boptimiz`, `\bcach`,
	`\btokeniz`, `\btoken\b`,
}

var relevanceRegex *regexp.Regexp

func init() {
	combined := strings.Join(relevancePatterns, "|")
	relevanceRegex = regexp.MustCompile("(?i)" + combined)
}

// arXiv Atom feed types.
type atomFeed struct {
	XMLName xml.Name    `xml:"feed"`
	Entries []atomEntry `xml:"entry"`
}

type atomEntry struct {
	Title     string       `xml:"title"`
	ID        string       `xml:"id"`
	Summary   string       `xml:"summary"`
	Published string       `xml:"published"`
	Updated   string       `xml:"updated"`
	Authors   []atomAuthor `xml:"author"`
	Links     []atomLink   `xml:"link"`
	Category  []atomCat    `xml:"category"`
}

type atomAuthor struct {
	Name string `xml:"name"`
}

type atomLink struct {
	Href string `xml:"href,attr"`
	Rel  string `xml:"rel,attr"`
	Type string `xml:"type,attr"`
}

type atomCat struct {
	Term string `xml:"term,attr"`
}

// Source fetches papers from arXiv.
type Source struct {
	name       string
	icon       string
	categories []string
}

// New creates an arXiv source with specified categories.
func New(name string, categories []string, icon string) *Source {
	if icon == "" {
		icon = "📄"
	}
	if len(categories) == 0 {
		categories = DefaultCategories
	}
	return &Source{name: name, icon: icon, categories: categories}
}

func (s *Source) Name() string        { return s.name }
func (s *Source) Icon() string        { return s.icon }
func (s *Source) TTL() time.Duration  { return 1 * time.Hour }

func (s *Source) Fetch(ctx context.Context, cfg source.Config) (*source.Section, error) {
	maxItems := 5
	if max, ok := cfg.Settings["max"].(int); ok {
		maxItems = max
	}

	// Build query: cat:cs.CL+OR+cat:cs.AI+OR+...
	// Note: arXiv API uses + for spaces and +OR+ for boolean OR.
	// Do NOT url.QueryEscape this — arXiv expects the raw query string.
	catClauses := make([]string, len(s.categories))
	for i, cat := range s.categories {
		catClauses[i] = "cat:" + cat
	}
	query := strings.Join(catClauses, "+OR+")

	// Fetch more than needed so keyword filter has enough to work with.
	fetchCount := maxItems * 10
	if fetchCount < 50 {
		fetchCount = 50
	}

	apiURL := fmt.Sprintf(
		"https://export.arxiv.org/api/query?search_query=%s&sortBy=submittedDate&sortOrder=descending&max_results=%d",
		query, fetchCount,
	)

	req, err := http.NewRequestWithContext(ctx, "GET", apiURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "hotbrew/1.0 (terminal-rss)")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("arxiv: status %d", resp.StatusCode)
	}

	var feed atomFeed
	if err := xml.NewDecoder(resp.Body).Decode(&feed); err != nil {
		return nil, fmt.Errorf("arxiv: parse: %w", err)
	}

	// Score by keyword relevance and take top N.
	type scored struct {
		entry atomEntry
		score int
	}
	var candidates []scored

	for _, entry := range feed.Entries {
		text := entry.Title + " " + entry.Summary
		matches := relevanceRegex.FindAllStringIndex(text, -1)
		candidates = append(candidates, scored{entry: entry, score: len(matches)})
	}

	// Sort by score descending (keyword matches first, then recency).
	for i := 0; i < len(candidates); i++ {
		for j := i + 1; j < len(candidates); j++ {
			if candidates[j].score > candidates[i].score {
				candidates[i], candidates[j] = candidates[j], candidates[i]
			}
		}
	}

	// Take top maxItems — even if some have 0 keyword matches,
	// they're still from the right categories and recently published.
	var items []source.Item
	for _, c := range candidates {
		if len(items) >= maxItems {
			break
		}

		entry := c.entry
		published, _ := time.Parse(time.RFC3339, entry.Published)
		if published.IsZero() {
			published, _ = time.Parse(time.RFC3339, entry.Updated)
		}

		// Get the abstract link (HTML version).
		paperURL := entry.ID
		for _, link := range entry.Links {
			if link.Rel == "alternate" {
				paperURL = link.Href
				break
			}
		}

		// Authors.
		var authors []string
		for _, a := range entry.Authors {
			if a.Name != "" {
				authors = append(authors, a.Name)
			}
		}
		authorStr := strings.Join(authors, ", ")
		if len(authors) > 3 {
			authorStr = strings.Join(authors[:3], ", ") + " et al."
		}

		// Categories as tags.
		var tags []string
		for _, cat := range entry.Category {
			tags = append(tags, cat.Term)
		}

		priority := source.Medium
		if c.score >= 8 {
			priority = source.Urgent
		} else if c.score >= 4 {
			priority = source.High
		}

		// Clean up abstract for subtitle.
		abstract := strings.TrimSpace(entry.Summary)
		abstract = strings.ReplaceAll(abstract, "\n", " ")
		if len(abstract) > 250 {
			abstract = abstract[:247] + "..."
		}

		items = append(items, source.Item{
			ID:        entry.ID,
			Title:     strings.TrimSpace(entry.Title),
			Subtitle:  abstract,
			Body:      fmt.Sprintf("Authors: %s\n\n%s", authorStr, entry.Summary),
			URL:       paperURL,
			Priority:  priority,
			Timestamp: published,
			Category:  "research",
			Icon:      s.icon,
			Actions: []source.Action{
				{Key: "o", Label: "open", Command: paperURL},
			},
			Metadata: map[string]any{
				"authors":         authors,
				"tags":            tags,
				"relevance_score": c.score,
			},
		})
	}

	return &source.Section{
		Name:     s.name,
		Icon:     s.icon,
		Priority: 30,
		Items:    items,
	}, nil
}

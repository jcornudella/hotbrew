package ai

import (
	"context"
	"fmt"
	"strings"

	"github.com/jcornudella/hotbrew/internal/briefing"
	"github.com/jcornudella/hotbrew/internal/intel"
)

// Summarizer wraps a Provider to satisfy briefing.Summarizer. Any
// provider error (including "not configured" — a nil provider) is
// caught here and the deterministic fallback takes over, so the
// caller never has to decide between "AI on" and "AI off" code
// paths. That's the whole point of the interface: AI is optional
// and invisible when it fails.
type Summarizer struct {
	Provider Provider
	Fallback briefing.Summarizer
}

// NewSummarizer builds a Summarizer pairing provider with the
// default fallback template. Pass a nil provider to get a
// deterministic-only summarizer — handy when the config layer has
// decided AI is disabled but you still want the same call site.
func NewSummarizer(provider Provider) Summarizer {
	return Summarizer{
		Provider: provider,
		Fallback: briefing.DefaultSummarizer(),
	}
}

// SummarizeItem asks the provider for a one-line summary of the
// item. On any error (including a nil provider) we degrade to the
// template path silently — the briefing must render.
func (s Summarizer) SummarizeItem(ctx context.Context, item intel.IntelItem) (string, error) {
	if s.Provider == nil {
		return s.fallback().SummarizeItem(ctx, item)
	}
	prompt := buildItemPrompt(item)
	out, err := s.Provider.Complete(ctx, prompt)
	if err != nil || strings.TrimSpace(out) == "" {
		return s.fallback().SummarizeItem(ctx, item)
	}
	return strings.TrimSpace(out), nil
}

// SummarizeCluster asks the provider for a short cluster blurb.
// Same fallback contract as SummarizeItem.
func (s Summarizer) SummarizeCluster(ctx context.Context, cluster intel.ThemeCluster, items []intel.IntelItem) (string, error) {
	if s.Provider == nil {
		return s.fallback().SummarizeCluster(ctx, cluster, items)
	}
	prompt := buildClusterPrompt(cluster, items)
	out, err := s.Provider.Complete(ctx, prompt)
	if err != nil || strings.TrimSpace(out) == "" {
		return s.fallback().SummarizeCluster(ctx, cluster, items)
	}
	return strings.TrimSpace(out), nil
}

func (s Summarizer) fallback() briefing.Summarizer {
	if s.Fallback == nil {
		return briefing.DefaultSummarizer()
	}
	return s.Fallback
}

// buildItemPrompt is a short instruction + the raw item fields. We
// keep the prompt itself deterministic and readable so a diff in
// output can always be traced to a model change, not a prompt
// regression introduced accidentally.
func buildItemPrompt(item intel.IntelItem) string {
	var b strings.Builder
	b.WriteString("Summarize this item in one sentence (max 30 words). ")
	b.WriteString("No preamble, no quotation marks. Return only the sentence.\n\n")
	fmt.Fprintf(&b, "Title: %s\n", item.Title)
	if item.Summary != "" {
		fmt.Fprintf(&b, "Summary: %s\n", item.Summary)
	}
	if item.Body != "" {
		body := item.Body
		if len(body) > 1500 {
			body = body[:1500]
		}
		fmt.Fprintf(&b, "Body: %s\n", body)
	}
	return b.String()
}

func buildClusterPrompt(cluster intel.ThemeCluster, items []intel.IntelItem) string {
	index := map[string]intel.IntelItem{}
	for _, it := range items {
		index[it.ID] = it
	}

	var b strings.Builder
	b.WriteString("Summarize this news cluster in one sentence (max 30 words). ")
	b.WriteString("Lead with what's happening; mention the theme only if it adds clarity. ")
	b.WriteString("No preamble, no quotation marks.\n\n")
	if cluster.Label != "" {
		fmt.Fprintf(&b, "Theme: %s\n", cluster.Label)
	}
	if lead, ok := index[cluster.Representative]; ok {
		fmt.Fprintf(&b, "Lead title: %s\n", lead.Title)
		if lead.Summary != "" {
			fmt.Fprintf(&b, "Lead summary: %s\n", lead.Summary)
		}
	}
	if len(cluster.ItemIDs) > 1 {
		b.WriteString("Other coverage:\n")
		for _, id := range cluster.ItemIDs {
			if id == cluster.Representative {
				continue
			}
			if it, ok := index[id]; ok {
				fmt.Fprintf(&b, "- %s (%s)\n", it.Title, it.SourceName)
			}
		}
	}
	return b.String()
}

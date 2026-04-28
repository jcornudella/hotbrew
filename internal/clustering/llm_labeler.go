// LLM-based theme labeling. The keyword matcher in labels.go is fast,
// deterministic, and cheap, but suffers from coverage gaps — a tweet
// like "@pmarca Catechism for Robots" or a post titled "DeepSeek v4"
// has no AI-keyword overlap, so it falls into GENERAL even though a
// human would call it AI immediately. The LLM labeler closes that gap
// by sending the clusters' titles + sources to Claude and getting back
// a slug per cluster.
//
// Cost discipline: the keyword matcher remains the default. The LLM
// labeler is opt-in (NewLLMLabeler returns nil when AI isn't
// configured), runs in batch (one API call per Cluster() invocation),
// and caches the >4K-token rubric so 99% of input tokens come back
// cached after the first call within the TTL window.

package clustering

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/jcornudella/hotbrew/internal/ai"
	"github.com/jcornudella/hotbrew/internal/config"
	"github.com/jcornudella/hotbrew/pkg/trss"
)

// Labeler is an optional batch labeling strategy used by Cluster.
// Implementations classify each ClusterBatch into one of the known
// theme slugs (KnownLabels). Returning a slug not in KnownLabels for
// any cluster is treated as a cluster-level failure and the keyword
// matcher fills in.
type Labeler interface {
	LabelClusters(ctx context.Context, batches []ClusterBatch) ([]ClusterLabel, error)
}

// ClusterBatch is one cluster passed to a Labeler. ID is opaque to
// the implementation — it's the matching key against ClusterLabel.ID.
type ClusterBatch struct {
	ID    string
	Items []trss.Item
}

// ClusterLabel is the labeler's verdict for one ClusterBatch.
type ClusterLabel struct {
	ID   string
	Slug string
}

// LLMLabeler is a Labeler that calls the Anthropic Messages API. It
// builds a >4K-token cacheable system prompt once at construction and
// reuses it across calls — cache_read_input_tokens should dominate
// input_tokens after the first request within the TTL window.
type LLMLabeler struct {
	provider     *ai.AnthropicProvider
	systemPrompt string
	schema       map[string]any
	allowedSlugs map[string]struct{}
}

// NewLLMLabeler returns an LLMLabeler when AI is configured, or nil
// otherwise. Returning nil rather than a no-op stub keeps the
// "should I use LLM labeling?" decision at the caller — Cluster()
// checks for nil and falls back to the keyword matcher cleanly.
func NewLLMLabeler(cfg *config.Config) *LLMLabeler {
	if cfg == nil || cfg.AI == nil {
		return nil
	}
	apiKey := resolveAPIKey(cfg.AI)
	if apiKey == "" {
		return nil
	}

	provider := ai.NewAnthropicProvider(apiKey)
	if cfg.AI.Model != "" {
		provider.Model = cfg.AI.Model
	}
	if cfg.AI.MaxTokens > 0 {
		provider.MaxTokens = cfg.AI.MaxTokens
	}

	allowed := allowedSlugSet()
	return &LLMLabeler{
		provider:     provider,
		systemPrompt: buildSystemPrompt(),
		schema:       buildSchema(allowed),
		allowedSlugs: allowed,
	}
}

// LabelClusters batches every cluster into a single API call. We do
// not split into per-cluster requests because that defeats both the
// batch latency win and the cache (each call would be a fresh prefix
// match attempt with the full rubric). If batching fails, the caller
// falls back to keyword labeling for every cluster — partial
// degradation rather than per-cluster retry.
func (l *LLMLabeler) LabelClusters(ctx context.Context, batches []ClusterBatch) ([]ClusterLabel, error) {
	if l == nil || len(batches) == 0 {
		return nil, nil
	}

	userPrompt, err := encodeBatches(batches)
	if err != nil {
		return nil, fmt.Errorf("llm labeler: encode batches: %w", err)
	}

	resp, err := l.provider.CompleteStructured(ctx, ai.StructuredRequest{
		SystemPrompt: l.systemPrompt,
		UserPrompt:   userPrompt,
		Schema:       l.schema,
		MaxTokens:    1024, // ~30 tokens per cluster × up to 30 clusters
		CacheTTL:     "1h",
	})
	if err != nil {
		return nil, fmt.Errorf("llm labeler: %w", err)
	}

	labels, err := parseLabels(resp.Text, l.allowedSlugs)
	if err != nil {
		return nil, fmt.Errorf("llm labeler: parse: %w (raw=%q)", err, resp.Text)
	}
	return labels, nil
}

// resolveAPIKey honors the env-var indirection. Reading the literal
// APIKey first lets tests inject a key without touching env state;
// APIKeyEnv is the production path that keeps the secret out of
// hotbrew.yaml.
func resolveAPIKey(cfg *config.AIConfig) string {
	if cfg.APIKey != "" {
		return cfg.APIKey
	}
	if cfg.APIKeyEnv != "" {
		return getenv(cfg.APIKeyEnv)
	}
	return getenv("ANTHROPIC_API_KEY")
}

// allowedSlugSet returns the set of slugs the LLM is allowed to
// produce. Sourcing from KnownLabels keeps the LLM and keyword
// matcher in agreement on the legal vocabulary — adding a label in
// labels.go automatically flows here.
func allowedSlugSet() map[string]struct{} {
	known := KnownLabels()
	out := make(map[string]struct{}, len(known))
	for _, l := range known {
		out[l.Slug] = struct{}{}
	}
	return out
}

// buildSchema constrains the model to {labels: [{id, slug}, ...]}
// where slug must be one of the known theme slugs. JSON Schema enum
// is sorted for cache stability — non-deterministic key/element
// order would invalidate the prefix cache between requests.
func buildSchema(allowed map[string]struct{}) map[string]any {
	enum := make([]string, 0, len(allowed))
	for slug := range allowed {
		enum = append(enum, slug)
	}
	sort.Strings(enum)

	enumAny := make([]any, len(enum))
	for i, s := range enum {
		enumAny[i] = s
	}

	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"labels": map[string]any{
				"type": "array",
				"items": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"id":   map[string]any{"type": "string"},
						"slug": map[string]any{"type": "string", "enum": enumAny},
					},
					"required":             []string{"id", "slug"},
					"additionalProperties": false,
				},
			},
		},
		"required":             []string{"labels"},
		"additionalProperties": false,
	}
}

// encodeBatches serializes the per-call inputs deterministically.
// Order of clusters is preserved; titles are joined with newlines
// inside each cluster object so the model sees the cluster as one
// chunk rather than as a list of strings (mildly better classification
// quality on multi-item clusters in our tests).
func encodeBatches(batches []ClusterBatch) (string, error) {
	type itemView struct {
		Title  string `json:"title"`
		Source string `json:"source"`
	}
	type clusterView struct {
		ID    string     `json:"id"`
		Items []itemView `json:"items"`
	}

	out := make([]clusterView, len(batches))
	for i, b := range batches {
		view := clusterView{ID: b.ID, Items: make([]itemView, 0, len(b.Items))}
		for _, it := range b.Items {
			view.Items = append(view.Items, itemView{
				Title:  truncateForPrompt(it.Title, 200),
				Source: it.Source.Name,
			})
		}
		out[i] = view
	}

	encoded, err := json.Marshal(map[string]any{"clusters": out})
	if err != nil {
		return "", err
	}
	return string(encoded), nil
}

func truncateForPrompt(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-1] + "…"
}

// parseLabels validates the model's output. We trust the schema
// constraint to enforce shape, but we also defensively check each
// slug against the allowed set — a future model or schema bug
// shouldn't be able to inject an unknown theme into the briefing.
func parseLabels(raw string, allowed map[string]struct{}) ([]ClusterLabel, error) {
	var parsed struct {
		Labels []ClusterLabel `json:"labels"`
	}
	if err := json.Unmarshal([]byte(strings.TrimSpace(raw)), &parsed); err != nil {
		return nil, err
	}
	for _, l := range parsed.Labels {
		if _, ok := allowed[l.Slug]; !ok {
			return nil, errors.New("unknown slug: " + l.Slug)
		}
	}
	return parsed.Labels, nil
}

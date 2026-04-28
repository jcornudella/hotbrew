package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"
)

// DefaultAnthropicModel is a cheap, fast model that's a reasonable
// default for short summary completions. Callers can override via
// config — we keep the default here rather than baking it into
// config so the config layer doesn't need to know about AI internals.
const DefaultAnthropicModel = "claude-haiku-4-5"

const (
	anthropicEndpoint = "https://api.anthropic.com/v1/messages"
	anthropicVersion  = "2023-06-01"
)

// AnthropicProvider talks to the Anthropic Messages API. It is safe
// to copy by value — the underlying http.Client has its own
// concurrency guarantees.
//
// Instances without an APIKey return an error on Complete rather
// than silently succeeding, so the summarizer wrapper can fall
// back to the deterministic path. We intentionally do not read
// ANTHROPIC_API_KEY from the environment here; the config layer
// owns credential resolution so env access stays auditable.
type AnthropicProvider struct {
	APIKey    string
	Model     string
	MaxTokens int
	Client    *http.Client
}

// NewAnthropicProvider returns a provider with reasonable defaults
// filled in. Caller still needs to supply an APIKey.
func NewAnthropicProvider(apiKey string) *AnthropicProvider {
	return &AnthropicProvider{
		APIKey:    apiKey,
		Model:     DefaultAnthropicModel,
		MaxTokens: 256,
		Client:    &http.Client{Timeout: 15 * time.Second},
	}
}

type anthropicRequest struct {
	Model     string             `json:"model"`
	MaxTokens int                `json:"max_tokens"`
	Messages  []anthropicMessage `json:"messages"`
}

type anthropicMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type anthropicResponse struct {
	Content []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content"`
	Error *struct {
		Type    string `json:"type"`
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

// Complete sends a single-turn user message and returns the model's
// first text block. Non-text blocks (tool use, images) are ignored
// — summary prompts don't ask for them.
func (p *AnthropicProvider) Complete(ctx context.Context, prompt string) (string, error) {
	if p.APIKey == "" {
		return "", errors.New("anthropic: missing API key")
	}

	model := p.Model
	if model == "" {
		model = DefaultAnthropicModel
	}
	maxTokens := p.MaxTokens
	if maxTokens <= 0 {
		maxTokens = 256
	}

	body, err := json.Marshal(anthropicRequest{
		Model:     model,
		MaxTokens: maxTokens,
		Messages:  []anthropicMessage{{Role: "user", Content: prompt}},
	})
	if err != nil {
		return "", fmt.Errorf("anthropic: encode request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, anthropicEndpoint, bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("anthropic: build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", p.APIKey)
	req.Header.Set("anthropic-version", anthropicVersion)

	client := p.Client
	if client == nil {
		client = &http.Client{Timeout: 15 * time.Second}
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("anthropic: transport: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("anthropic: read response: %w", err)
	}

	var parsed anthropicResponse
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return "", fmt.Errorf("anthropic: decode response: %w", err)
	}
	if parsed.Error != nil {
		return "", fmt.Errorf("anthropic: %s: %s", parsed.Error.Type, parsed.Error.Message)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("anthropic: status %d", resp.StatusCode)
	}

	for _, block := range parsed.Content {
		if block.Type == "text" && block.Text != "" {
			return block.Text, nil
		}
	}
	return "", errors.New("anthropic: empty response")
}

// StructuredRequest is the input shape for CompleteStructured. It
// keeps the cacheable system prompt and the per-call user prompt
// separate so callers don't have to think about block ordering or
// cache_control placement — the wire shape is constructed correctly
// by the provider.
//
// SystemPrompt is the stable, cacheable prefix (theme rubric,
// few-shot examples, instructions). UserPrompt is the volatile
// suffix (the actual classification target). Schema, if non-nil,
// constrains the model to JSON matching that JSON Schema object.
type StructuredRequest struct {
	SystemPrompt string
	UserPrompt   string
	Schema       map[string]any // JSON Schema; nil = free-form text
	MaxTokens    int            // 0 → uses provider default
	CacheTTL     string         // "" or "5m" → default ephemeral; "1h" → extended
}

// StructuredResponse carries the parsed text plus cache telemetry so
// callers can verify caching is actually working — a silent
// invalidator (timestamp in the prompt, non-deterministic key
// ordering) shows up as a permanently zero CacheReadTokens.
type StructuredResponse struct {
	Text             string
	InputTokens      int
	OutputTokens     int
	CacheReadTokens  int
	CacheWriteTokens int
}

type anthropicSystemBlock struct {
	Type         string                 `json:"type"`
	Text         string                 `json:"text"`
	CacheControl *anthropicCacheControl `json:"cache_control,omitempty"`
}

type anthropicCacheControl struct {
	Type string `json:"type"`
	TTL  string `json:"ttl,omitempty"`
}

type anthropicJSONFormat struct {
	Type   string         `json:"type"`
	Schema map[string]any `json:"schema"`
}

type anthropicOutputConfig struct {
	Format anthropicJSONFormat `json:"format"`
}

type structuredRequest struct {
	Model        string                 `json:"model"`
	MaxTokens    int                    `json:"max_tokens"`
	System       []anthropicSystemBlock `json:"system,omitempty"`
	Messages     []anthropicMessage     `json:"messages"`
	OutputConfig *anthropicOutputConfig `json:"output_config,omitempty"`
}

type structuredResponseEnvelope struct {
	Content []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content"`
	Usage struct {
		InputTokens              int `json:"input_tokens"`
		OutputTokens             int `json:"output_tokens"`
		CacheReadInputTokens     int `json:"cache_read_input_tokens"`
		CacheCreationInputTokens int `json:"cache_creation_input_tokens"`
	} `json:"usage"`
	Error *struct {
		Type    string `json:"type"`
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

// CompleteStructured sends a request with a cacheable system prompt
// and an optional JSON-Schema response constraint. Use this for
// closed-set classification, structured extraction, and any other
// task where the prompt has a stable rubric and a varying input.
//
// The system prompt is wrapped in a single text block with
// cache_control set, so a prefix at or above the model's minimum
// cacheable length (4096 tokens for Opus 4.7 / Haiku 4.5) is
// served from cache on subsequent calls.
func (p *AnthropicProvider) CompleteStructured(ctx context.Context, in StructuredRequest) (StructuredResponse, error) {
	if p.APIKey == "" {
		return StructuredResponse{}, errors.New("anthropic: missing API key")
	}

	model := p.Model
	if model == "" {
		model = DefaultAnthropicModel
	}
	maxTokens := in.MaxTokens
	if maxTokens <= 0 {
		maxTokens = p.MaxTokens
	}
	if maxTokens <= 0 {
		maxTokens = 1024
	}

	cache := &anthropicCacheControl{Type: "ephemeral"}
	if in.CacheTTL == "1h" {
		cache.TTL = "1h"
	}

	body := structuredRequest{
		Model:     model,
		MaxTokens: maxTokens,
		Messages:  []anthropicMessage{{Role: "user", Content: in.UserPrompt}},
	}
	if in.SystemPrompt != "" {
		body.System = []anthropicSystemBlock{{
			Type:         "text",
			Text:         in.SystemPrompt,
			CacheControl: cache,
		}}
	}
	if in.Schema != nil {
		body.OutputConfig = &anthropicOutputConfig{
			Format: anthropicJSONFormat{Type: "json_schema", Schema: in.Schema},
		}
	}

	encoded, err := json.Marshal(body)
	if err != nil {
		return StructuredResponse{}, fmt.Errorf("anthropic: encode request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, anthropicEndpoint, bytes.NewReader(encoded))
	if err != nil {
		return StructuredResponse{}, fmt.Errorf("anthropic: build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", p.APIKey)
	req.Header.Set("anthropic-version", anthropicVersion)

	client := p.Client
	if client == nil {
		client = &http.Client{Timeout: 30 * time.Second}
	}
	resp, err := client.Do(req)
	if err != nil {
		return StructuredResponse{}, fmt.Errorf("anthropic: transport: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return StructuredResponse{}, fmt.Errorf("anthropic: read response: %w", err)
	}

	var parsed structuredResponseEnvelope
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return StructuredResponse{}, fmt.Errorf("anthropic: decode response: %w", err)
	}
	if parsed.Error != nil {
		return StructuredResponse{}, fmt.Errorf("anthropic: %s: %s", parsed.Error.Type, parsed.Error.Message)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return StructuredResponse{}, fmt.Errorf("anthropic: status %d: %s", resp.StatusCode, string(raw))
	}

	var text string
	for _, block := range parsed.Content {
		if block.Type == "text" && block.Text != "" {
			text = block.Text
			break
		}
	}
	if text == "" {
		return StructuredResponse{}, errors.New("anthropic: empty response")
	}

	return StructuredResponse{
		Text:             text,
		InputTokens:      parsed.Usage.InputTokens,
		OutputTokens:     parsed.Usage.OutputTokens,
		CacheReadTokens:  parsed.Usage.CacheReadInputTokens,
		CacheWriteTokens: parsed.Usage.CacheCreationInputTokens,
	}, nil
}

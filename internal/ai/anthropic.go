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

// Package ai hosts AI-backed enrichments for the briefing pipeline.
// The package is deliberately narrow: a Provider abstracts "give me
// a short completion for a prompt", and a Summarizer wraps that
// into the briefing.Summarizer interface with a deterministic
// fallback. Ranking, clustering, and selection never call anything
// here — AI strictly enriches copy, it doesn't decide what appears.
package ai

import "context"

// Provider is the minimal contract a backend must satisfy to power
// AI-driven summaries. Keeping it one method makes mocking trivial
// for tests and makes swapping backends (Anthropic, OpenAI, local
// models) a matter of writing one adapter.
type Provider interface {
	Complete(ctx context.Context, prompt string) (string, error)
}

// ProviderFunc lets any free function satisfy Provider. Mostly
// useful in tests so you don't have to declare a struct just to
// stub one response.
type ProviderFunc func(ctx context.Context, prompt string) (string, error)

// Complete implements Provider for ProviderFunc.
func (f ProviderFunc) Complete(ctx context.Context, prompt string) (string, error) {
	return f(ctx, prompt)
}

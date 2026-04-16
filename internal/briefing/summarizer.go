package briefing

import (
	"context"

	"github.com/jcornudella/hotbrew/internal/intel"
)

// Summarizer produces short, user-facing summaries for items and
// clusters. It sits at the boundary where briefing assembly hands
// off to rendering — extracting this interface now (with a
// deterministic fallback) lets us swap in an LLM-backed provider
// later without the rest of the pipeline having to know.
//
// Implementations must be safe to call with a cancelled context: a
// fallback can ignore it, but real providers should honor it.
// Errors should be reserved for genuine failures; an unavailable
// provider should return a graceful placeholder string rather than
// propagating an error that stalls the whole briefing.
type Summarizer interface {
	SummarizeItem(ctx context.Context, item intel.IntelItem) (string, error)
	SummarizeCluster(ctx context.Context, cluster intel.ThemeCluster, items []intel.IntelItem) (string, error)
}

// DefaultSummarizer is the process-wide fallback used when no
// provider is configured. It's a pure template summarizer — no IO,
// no randomness — so the rest of the system stays deterministic
// unless a caller explicitly plugs in an LLM-backed implementation.
func DefaultSummarizer() Summarizer {
	return FallbackSummarizer{}
}

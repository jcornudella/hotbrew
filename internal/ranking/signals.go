package ranking

import (
	"math"
	"net/url"
	"strings"
	"time"
	"unicode"

	"github.com/jcornudella/hotbrew/internal/intel"
	"github.com/jcornudella/hotbrew/pkg/trss"
)

// Phase-1 tuning constants. Kept small and explicit so ranking stays
// deterministic and inspectable. Tune by reading fixture test output.
const (
	resonanceStep     = 0.3
	resonanceCap      = 2.0
	sourceRepeatCap   = 5
	domainRepeatCap   = 3
	repeatPenaltyMin  = 0.3
	titleMatchMinimum = 10 // normalized title must carry enough signal
)

// ComputeSignals derives per-item ranking features. Corpus-wide signals
// (resonance, repeat penalty) are computed separately over the candidate
// set and folded in by the ranker.
func ComputeSignals(item trss.Item, sourceWeights map[string]float64, now time.Time) intel.ItemSignals {
	if now.IsZero() {
		now = time.Now()
	}
	return intel.ItemSignals{
		Freshness:       RecencyScore(item.PublishedAt, now),
		SourceAuthority: SourceWeight(item.Source.Name, sourceWeights),
		Engagement:      EngagementScore(item.Engagement),
	}
}

// ComputeResonance boosts items that surface from multiple distinct
// sources via the same canonical URL or normalized title — strong signal
// that the story matters beyond one community.
func ComputeResonance(items []trss.Item) map[string]float64 {
	urlSources := map[string]map[string]struct{}{}
	titleSources := map[string]map[string]struct{}{}

	for _, item := range items {
		if key := canonicalKey(item); key != "" {
			addSource(urlSources, key, item.Source.Name)
		}
		if title := normalizeTitle(item.Title); len(title) >= titleMatchMinimum {
			addSource(titleSources, title, item.Source.Name)
		}
	}

	scores := make(map[string]float64, len(items))
	for _, item := range items {
		matches := 1
		if key := canonicalKey(item); key != "" {
			if n := len(urlSources[key]); n > matches {
				matches = n
			}
		}
		if title := normalizeTitle(item.Title); len(title) >= titleMatchMinimum {
			if n := len(titleSources[title]); n > matches {
				matches = n
			}
		}
		score := 1.0 + resonanceStep*float64(matches-1)
		if score > resonanceCap {
			score = resonanceCap
		}
		scores[item.ID] = score
	}
	return scores
}

// ComputeRepeatPenalty demotes items whose source or domain is
// over-represented in the current candidate set, so one loud source
// can't hog the briefing.
func ComputeRepeatPenalty(items []trss.Item) map[string]float64 {
	sourceCount := map[string]int{}
	domainCount := map[string]int{}
	for _, item := range items {
		sourceCount[item.Source.Name]++
		if d := extractDomain(item); d != "" {
			domainCount[d]++
		}
	}

	scores := make(map[string]float64, len(items))
	for _, item := range items {
		penalty := 1.0
		if n := sourceCount[item.Source.Name]; n > sourceRepeatCap {
			penalty *= float64(sourceRepeatCap) / float64(n)
		}
		if d := extractDomain(item); d != "" {
			if n := domainCount[d]; n > domainRepeatCap {
				penalty *= float64(domainRepeatCap) / float64(n)
			}
		}
		if penalty < repeatPenaltyMin {
			penalty = repeatPenaltyMin
		}
		scores[item.ID] = penalty
	}
	return scores
}

// RecencyScore decays exponentially with age and floors at 0.1.
func RecencyScore(published time.Time, now time.Time) float64 {
	ageHours := now.Sub(published).Hours()
	if ageHours < 0 {
		ageHours = 0
	}
	score := math.Exp(-ageHours / 24.0)
	if score < 0.1 {
		return 0.1
	}
	return score
}

// SourceWeight returns the configured weight for a source, defaulting to 1.0.
func SourceWeight(sourceName string, weights map[string]float64) float64 {
	if weights == nil {
		return 1.0
	}
	if w, ok := weights[sourceName]; ok {
		return w
	}
	return 1.0
}

// EngagementScore normalizes available engagement signals.
func EngagementScore(engagement map[string]any) float64 {
	if engagement == nil {
		return 1.0
	}

	points := extractFloat(engagement, "points")
	stars := extractFloat(engagement, "stars")
	comments := extractFloat(engagement, "comments")

	signal := points
	if stars > signal {
		signal = stars
	}
	signal += comments * 0.5

	if signal <= 0 {
		return 1.0
	}

	score := math.Log(1+signal) / math.Log(1+500)
	if score > 2.0 {
		return 2.0
	}
	return score
}

// Theme preference multipliers. Numbers are deliberately loud so
// the signal survives the stack of other multiplicative factors —
// a followed theme nudges above mid-tier freshness, and a muted
// theme multiplies to zero so those items sort to the bottom
// regardless of how strong their other signals are.
const (
	FollowedThemeBoost = 1.35
	MutedThemeBoost    = 0.0
)

// ThemeMultiplier maps a theme slug and the user's preference map
// to a multiplicative factor applied on top of TopicMatch.
// Unknown or absent preferences are neutral (1.0) so the ranker
// behaves identically for users who never followed anything.
func ThemeMultiplier(slug string, preferences map[string]string) float64 {
	if slug == "" || len(preferences) == 0 {
		return 1.0
	}
	switch preferences[slug] {
	case "follow":
		return FollowedThemeBoost
	case "mute":
		return MutedThemeBoost
	default:
		return 1.0
	}
}

// AffinityFactor converts a behavioral affinity score (typically in
// [-1.5, 1.5] for themes/sources, unbounded negative for muted
// domains) into a multiplicative factor centered at 1.0. The slope
// is intentionally gentler than explicit FollowedThemeBoost so a
// followed-theme always feels stronger than learned preference, but
// behavior still nudges the briefing toward what the user engages
// with. Output is clamped to [0.5, 1.5] so a single noisy session
// can't swing the ranker.
func AffinityFactor(score float64) float64 {
	const slope = 0.2
	f := 1.0 + score*slope
	if f < 0.5 {
		return 0.5
	}
	if f > 1.5 {
		return 1.5
	}
	return f
}

// UserBoost applies tag/source boosts to a TRSS item.
func UserBoost(item trss.Item, boosts map[string]float64) float64 {
	if boosts == nil {
		return 1.0
	}
	for _, tag := range item.Tags {
		if b, ok := boosts[tag]; ok {
			return b
		}
	}
	if b, ok := boosts[item.Source.Name]; ok {
		return b
	}
	return 1.0
}

func addSource(index map[string]map[string]struct{}, key, source string) {
	bucket, ok := index[key]
	if !ok {
		bucket = map[string]struct{}{}
		index[key] = bucket
	}
	bucket[source] = struct{}{}
}

func canonicalKey(item trss.Item) string {
	key := strings.TrimSpace(item.URLCanonical)
	if key == "" {
		key = strings.TrimSpace(item.URL)
	}
	return strings.ToLower(key)
}

func normalizeTitle(title string) string {
	var b strings.Builder
	lastSpace := true
	for _, r := range title {
		switch {
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			b.WriteRune(unicode.ToLower(r))
			lastSpace = false
		case !lastSpace:
			b.WriteByte(' ')
			lastSpace = true
		}
	}
	return strings.TrimSpace(b.String())
}

func extractDomain(item trss.Item) string {
	key := canonicalKey(item)
	if key == "" {
		return ""
	}
	parsed, err := url.Parse(key)
	if err != nil {
		return ""
	}
	return strings.TrimPrefix(parsed.Hostname(), "www.")
}

func extractFloat(m map[string]any, key string) float64 {
	v, ok := m[key]
	if !ok {
		return 0
	}
	switch n := v.(type) {
	case float64:
		return n
	case int:
		return float64(n)
	case int64:
		return float64(n)
	default:
		return 0
	}
}

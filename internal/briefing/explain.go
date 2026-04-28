package briefing

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/jcornudella/hotbrew/internal/intel"
)

// Explanation is the structured output backing both the `explain`
// (user-facing) and `why` (system-facing) commands. It carries
// everything a renderer needs; formatting decisions live at the
// command layer. Structured data also keeps the output testable — a
// prose-only explanation would drift with every string tweak.
type Explanation struct {
	ItemID       string
	ClusterID    string
	Title        string
	Theme        string
	Sources      []string
	WhyItMatters string
	WhyYouSee    string
	Factors      []Factor
}

// Factor is one signal that contributed to the item's ranking,
// surfaced in `why` output. Factors are sorted by Multiplier desc so
// the biggest movers show first.
type Factor struct {
	Name        string
	Multiplier  float64
	Description string
}

// ExplainOptions carries personalization context that the briefing
// itself doesn't store on each item — affinity scores and explicit
// preferences live in the store, and we read them at explain time so
// users always see the *current* learned signal, not a snapshot from
// when the digest was scored.
type ExplainOptions struct {
	ThemeAffinity  map[string]float64 // theme slug → behavioral score
	SourceAffinity map[string]float64 // source name → behavioral score
	ThemePrefs     map[string]string  // theme slug → "follow"|"mute"
}

// Explain assembles the explanation for itemID from a briefing. It
// returns false when the id doesn't appear in the briefing so
// callers can surface a clean "not found" rather than empty output.
//
// This is the no-personalization wrapper. Use ExplainWith when the
// caller has access to the store and wants affinity factors mixed in.
func Explain(b *intel.Briefing, itemID string) (Explanation, bool) {
	return ExplainWith(b, itemID, ExplainOptions{})
}

// ExplainWith is the personalization-aware variant. When opts carries
// theme/source affinity scores and explicit theme prefs, the
// explanation grows additional factors and the "why it matters" line
// pulls in personal context ("you've been engaging with AI items").
func ExplainWith(b *intel.Briefing, itemID string, opts ExplainOptions) (Explanation, bool) {
	if b == nil || itemID == "" {
		return Explanation{}, false
	}

	item, ok := findItem(b.Items, itemID)
	if !ok {
		return Explanation{}, false
	}

	cluster, sources := enclosingCluster(b, item)

	exp := Explanation{
		ItemID:    item.Item.ID,
		ClusterID: cluster.ID,
		Title:     item.Item.Title,
		Theme:     clusterLabel(cluster),
		Sources:   sources,
		Factors:   rankFactors(item.Breakdown),
	}
	exp.Factors = appendPersonalFactors(exp.Factors, item, cluster, opts)
	exp.WhyItMatters = whyItMattersWithPersonal(item, cluster, sources, opts)
	exp.WhyYouSee = whyYouSee(exp.Factors)
	return exp, true
}

func findItem(items []intel.ScoredItem, id string) (intel.ScoredItem, bool) {
	for _, item := range items {
		if item.Item.ID == id {
			return item, true
		}
	}
	return intel.ScoredItem{}, false
}

// enclosingCluster returns the cluster that contains item and the
// deduped list of source names represented in it. When no cluster
// holds the item we fall back to a singleton so downstream
// formatting stays uniform.
func enclosingCluster(b *intel.Briefing, item intel.ScoredItem) (intel.ThemeCluster, []string) {
	for _, c := range b.Clusters {
		for _, id := range c.ItemIDs {
			if id == item.Item.ID {
				return c, sourcesForCluster(b, c)
			}
		}
	}
	singleton := intel.ThemeCluster{
		ID:             "",
		Representative: item.Item.ID,
		ItemIDs:        []string{item.Item.ID},
	}
	return singleton, []string{item.Item.SourceName}
}

func sourcesForCluster(b *intel.Briefing, c intel.ThemeCluster) []string {
	seen := map[string]bool{}
	var names []string
	for _, id := range c.ItemIDs {
		for _, item := range b.Items {
			if item.Item.ID != id {
				continue
			}
			name := item.Item.SourceName
			if name == "" || seen[name] {
				continue
			}
			seen[name] = true
			names = append(names, name)
		}
	}
	sort.Strings(names)
	return names
}

func clusterLabel(c intel.ThemeCluster) string {
	if c.Label != "" {
		return c.Label
	}
	if c.Slug != "" {
		return displayName(c.Slug)
	}
	return "General"
}

// rankFactors turns the score breakdown into an ordered factor list.
// We only surface factors that carry a non-neutral multiplier so the
// output focuses on what actually moved the rank; resonance at 1.0
// isn't a story worth telling.
func rankFactors(bd intel.ScoreBreakdown) []Factor {
	raw := []Factor{
		{Name: "freshness", Multiplier: bd.Freshness, Description: describeFreshness(bd.Freshness)},
		{Name: "authority", Multiplier: bd.Authority, Description: describeAuthority(bd.Authority)},
		{Name: "engagement", Multiplier: bd.Engagement, Description: describeEngagement(bd.Engagement)},
		{Name: "resonance", Multiplier: bd.Resonance, Description: describeResonance(bd.Resonance)},
		{Name: "topic match", Multiplier: bd.TopicMatch, Description: describeTopicMatch(bd.TopicMatch)},
		{Name: "repeat penalty", Multiplier: bd.RepeatPenalty, Description: describeRepeatPenalty(bd.RepeatPenalty)},
	}
	factors := make([]Factor, 0, len(raw))
	for _, f := range raw {
		if f.Multiplier == 0 || isNeutral(f.Multiplier) {
			continue
		}
		factors = append(factors, f)
	}
	sort.SliceStable(factors, func(i, j int) bool {
		return distanceFromNeutral(factors[i].Multiplier) > distanceFromNeutral(factors[j].Multiplier)
	})
	return factors
}

func isNeutral(m float64) bool {
	return m > 0.999 && m < 1.001
}

func distanceFromNeutral(m float64) float64 {
	if m >= 1 {
		return m - 1
	}
	return 1 - m
}

func describeFreshness(m float64) string {
	switch {
	case m >= 1.5:
		return "very fresh — published within the last few hours"
	case m >= 1.2:
		return "fresh — published today"
	case m <= 0.5:
		return "older — pushed toward the tail of the window"
	default:
		return fmt.Sprintf("moderately fresh (x%.2f)", m)
	}
}

func describeAuthority(m float64) string {
	switch {
	case m >= 1.3:
		return "from a source you trust"
	case m <= 0.7:
		return "from a low-weight source"
	default:
		return fmt.Sprintf("source weight x%.2f", m)
	}
}

func describeEngagement(m float64) string {
	switch {
	case m >= 1.5:
		return "heavy engagement (points, comments, stars)"
	case m >= 1.2:
		return "above-average engagement"
	default:
		return fmt.Sprintf("engagement x%.2f", m)
	}
}

func describeResonance(m float64) string {
	if m >= 1.6 {
		return "echoed across several sources"
	}
	if m >= 1.2 {
		return "covered by more than one source"
	}
	return fmt.Sprintf("cross-source resonance x%.2f", m)
}

func describeTopicMatch(m float64) string {
	if m > 1 {
		return "matches a topic you boosted"
	}
	if m < 1 {
		return "matches a topic you muted"
	}
	return fmt.Sprintf("topic match x%.2f", m)
}

func describeRepeatPenalty(m float64) string {
	if m < 0.7 {
		return "demoted — same source or domain dominates the window"
	}
	return fmt.Sprintf("repeat penalty x%.2f", m)
}

// whyItMatters writes the user-facing line. It leans on cluster
// composition (how many sources carry the story) and freshness
// because those are the most reliable signals without an LLM.
// Prose-heavy summaries come later via the Summarizer interface.
func whyItMatters(item intel.ScoredItem, cluster intel.ThemeCluster, sources []string) string {
	var parts []string

	theme := clusterLabel(cluster)
	if theme != "" && theme != "General" {
		parts = append(parts, fmt.Sprintf("Sits in the %s track.", theme))
	}

	if len(sources) > 1 {
		parts = append(parts, fmt.Sprintf("Covered by %d sources (%s) — cross-source agreement.",
			len(sources), strings.Join(sources, ", ")))
	}

	if !item.Item.PublishedAt.IsZero() {
		age := time.Since(item.Item.PublishedAt)
		if age > 0 && age < 24*time.Hour {
			parts = append(parts, fmt.Sprintf("Fresh — published %s ago.", humanDuration(age)))
		}
	}

	if len(parts) == 0 {
		return "Surfaced because it cleared the ranking threshold for this window."
	}
	return strings.Join(parts, " ")
}

// whyYouSee writes the system-facing line. It names the top factors
// explicitly so users can trace a score to its drivers.
func whyYouSee(factors []Factor) string {
	if len(factors) == 0 {
		return "All ranking signals were neutral for this item."
	}
	names := make([]string, 0, 3)
	for i, f := range factors {
		if i >= 3 {
			break
		}
		names = append(names, f.Name)
	}
	return "Top signals: " + strings.Join(names, ", ") + "."
}

func humanDuration(d time.Duration) string {
	if d < time.Minute {
		return "under a minute"
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm", int(d.Minutes()))
	}
	return fmt.Sprintf("%dh", int(d.Hours()))
}

// appendPersonalFactors adds theme/source affinity factors derived
// from the personalize package's snapshot. We render them as
// "factor multipliers" on the same axis as the standard signals so
// users see them in one ordered list — distance-from-neutral keeps
// the strongest movers (positive or negative) at the top regardless
// of factor type.
func appendPersonalFactors(base []Factor, item intel.ScoredItem, cluster intel.ThemeCluster, opts ExplainOptions) []Factor {
	out := append([]Factor{}, base...)

	if score, ok := opts.ThemeAffinity[cluster.Slug]; ok && score != 0 {
		mult := affinityToMultiplier(score)
		out = append(out, Factor{
			Name:        "theme affinity",
			Multiplier:  mult,
			Description: describeThemeAffinity(cluster.Slug, score),
		})
	}

	if name := item.Item.SourceName; name != "" {
		if score, ok := opts.SourceAffinity[name]; ok && score != 0 {
			mult := affinityToMultiplier(score)
			out = append(out, Factor{
				Name:        "source affinity",
				Multiplier:  mult,
				Description: describeSourceAffinity(name, score),
			})
		}
	}

	if state, ok := opts.ThemePrefs[cluster.Slug]; ok && state != "" {
		out = append(out, Factor{
			Name:        "theme preference",
			Multiplier:  prefMultiplier(state),
			Description: describeThemePref(cluster.Slug, state),
		})
	}

	sort.SliceStable(out, func(i, j int) bool {
		return distanceFromNeutral(out[i].Multiplier) > distanceFromNeutral(out[j].Multiplier)
	})
	return out
}

// affinityToMultiplier maps an affinity score (≈ ±1.5 for themes/
// sources, larger negative for muted domains) into a multiplier
// centered at 1.0. Mirrors ranking.AffinityFactor's slope so the
// number a user sees here matches what the ranker actually applied,
// without importing the ranking package (which would create a cycle).
func affinityToMultiplier(score float64) float64 {
	const slope = 0.2
	m := 1.0 + score*slope
	if m < 0.5 {
		return 0.5
	}
	if m > 1.5 {
		return 1.5
	}
	return m
}

func prefMultiplier(state string) float64 {
	switch state {
	case "follow":
		return 1.35
	case "mute":
		return 0.0
	}
	return 1.0
}

func describeThemeAffinity(slug string, score float64) string {
	switch {
	case score >= 1.0:
		return fmt.Sprintf("you've been engaging with %s items lately", slug)
	case score >= 0.3:
		return fmt.Sprintf("you've shown interest in %s recently", slug)
	case score <= -1.0:
		return fmt.Sprintf("you've been ignoring %s items", slug)
	case score <= -0.3:
		return fmt.Sprintf("you've shown less interest in %s lately", slug)
	default:
		return fmt.Sprintf("learned %s affinity %+.2f", slug, score)
	}
}

func describeSourceAffinity(name string, score float64) string {
	switch {
	case score >= 1.0:
		return fmt.Sprintf("you favor %s — frequent opens and saves", name)
	case score >= 0.3:
		return fmt.Sprintf("you've been reading %s recently", name)
	case score <= -1.0:
		return fmt.Sprintf("you've been skipping %s", name)
	default:
		return fmt.Sprintf("source affinity %+.2f", score)
	}
}

func describeThemePref(slug, state string) string {
	switch state {
	case "follow":
		return fmt.Sprintf("you explicitly follow %s", slug)
	case "mute":
		return fmt.Sprintf("you explicitly muted %s — score zeroed", slug)
	}
	return fmt.Sprintf("theme %s state=%s", slug, state)
}

// whyItMattersWithPersonal layers personal context onto the standard
// "why it matters" line. We only add a personal sentence when the
// signal is strong enough to be meaningful — small positive scores
// would feel performative ("we noticed you opened a thing") and
// undermine trust in stronger signals.
func whyItMattersWithPersonal(item intel.ScoredItem, cluster intel.ThemeCluster, sources []string, opts ExplainOptions) string {
	base := whyItMatters(item, cluster, sources)

	if state := opts.ThemePrefs[cluster.Slug]; state == "follow" {
		return fmt.Sprintf("%s You follow %s.", base, cluster.Slug)
	}

	if score, ok := opts.ThemeAffinity[cluster.Slug]; ok && score >= 1.0 {
		return fmt.Sprintf("%s You've been deep in %s lately.", base, cluster.Slug)
	}

	return base
}

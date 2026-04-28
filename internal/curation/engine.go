package curation

import (
	"sort"
	"time"

	"github.com/jcornudella/hotbrew/internal/clustering"
	"github.com/jcornudella/hotbrew/internal/intel"
	"github.com/jcornudella/hotbrew/internal/ranking"
	"github.com/jcornudella/hotbrew/internal/store"
	"github.com/jcornudella/hotbrew/pkg/trss"
)

// Engine orchestrates the curation pipeline.
type Engine struct {
	Store  *store.Store
	Limits DiversityLimits
}

// NewEngine creates a curation engine with default settings.
func NewEngine(st *store.Store) *Engine {
	return &Engine{
		Store:  st,
		Limits: DefaultLimits(),
	}
}

// GenerateDigest runs the full curation pipeline:
// 1. Load items from store within the time window
// 2. Apply user rules (mute/boost)
// 3. Dedup (fingerprint + fuzzy title)
// 4. Score via ranking package and persist derived features
// 5. Sort by score
// 6. Enforce diversity limits
// 7. Package as trss.Digest
func (e *Engine) GenerateDigest(window time.Duration, maxItems int, title string) (*trss.Digest, error) {
	items, err := e.Store.ListItems(store.ItemFilter{Since: window})
	if err != nil {
		return nil, err
	}
	totalConsidered := len(items)

	rules, _ := e.Store.ListRules()
	filtered, boosts := ApplyRules(items, rules)
	rulesApplied := CountAppliedRules(totalConsidered, len(filtered), boosts)

	// Resonance measures cross-source agreement, so it must see the
	// pre-dedup corpus — after dedup, only one representative per URL
	// or near-identical title survives, which would hide the signal.
	resonance := ranking.ComputeResonance(filtered)

	deduped := Dedup(filtered, e.Store)
	itemsDeduped := len(filtered) - len(deduped)

	// Repeat penalty is about balancing the actual briefing output, so
	// it runs on the survivors — inflating counts with duplicates would
	// over-penalize and hide otherwise strong stories.
	repeat := ranking.ComputeRepeatPenalty(deduped)

	sourceWeights := e.loadSourceWeights()
	topicBoosts := e.computeThemeBoosts(deduped, e.loadThemePreferences())
	ranked := ranking.RankItemsWith(deduped, sourceWeights, boosts, time.Now(), resonance, repeat, topicBoosts)
	e.persistFeatures(ranked)

	scored := applyScores(deduped, ranked)
	sort.Slice(scored, func(i, j int) bool { return scored[i].Score > scored[j].Score })

	diverse := EnforceDiversity(scored, e.Limits, maxItems)
	for _, item := range diverse {
		_ = e.Store.UpdateScore(item.ID, item.Score)
	}

	digest := trss.NewDigest(title, window.String(), maxItems)
	digest.Items = diverse
	digest.ItemCount = len(diverse)
	digest.Meta = trss.DigestMeta{
		SourcesSynced:   e.countSources(diverse),
		ItemsConsidered: totalConsidered,
		ItemsDeduped:    itemsDeduped,
		RulesApplied:    rulesApplied,
	}
	digest.Sections = e.buildSections(diverse)
	return digest, nil
}

// persistFeatures upserts the derived signals for every ranked item so that
// later stages (and future briefings) can inspect them without recomputing.
func (e *Engine) persistFeatures(ranked []intel.ScoredItem) {
	if e.Store == nil {
		return
	}
	for _, r := range ranked {
		signals := intel.ItemSignals{
			Freshness:       r.Breakdown.Freshness,
			SourceAuthority: r.Breakdown.Authority,
			Engagement:      r.Breakdown.Engagement,
			Resonance:       r.Breakdown.Resonance,
			TopicMatch:      r.Breakdown.TopicMatch,
			RepeatPenalty:   r.Breakdown.RepeatPenalty,
		}
		_ = e.Store.UpsertItemFeatures(r.Item.ID, signals)
	}
}

// applyScores zips ranking scores back onto the source trss items used by
// the downstream diversity/section steps that still operate on trss.Item.
func applyScores(items []trss.Item, ranked []intel.ScoredItem) []trss.Item {
	scored := make([]trss.Item, len(items))
	for i, item := range items {
		if i < len(ranked) {
			item.Score = ranked[i].Score
		}
		scored[i] = item
	}
	return scored
}

// loadThemePreferences reads the user's follow/mute dictionary.
// A nil/empty result is treated as neutral by downstream ranking.
func (e *Engine) loadThemePreferences() map[string]string {
	if e.Store == nil {
		return nil
	}
	prefs, err := e.Store.ListThemePreferences()
	if err != nil {
		return nil
	}
	return prefs
}

// computeThemeBoosts labels each candidate item and combines the
// user's explicit preferences with learned behavioral affinity into
// a per-item multiplier. Explicit follow/mute always wins — affinity
// only nudges within the explicit envelope (skipped when a theme is
// muted, since 0 * anything = 0).
//
// Labels are computed on single-item "clusters" here — the real
// clustering pass runs later inside briefing assembly, but for
// ranking we only need each item's own dominant theme.
func (e *Engine) computeThemeBoosts(items []trss.Item, preferences map[string]string) map[string]float64 {
	if len(items) == 0 {
		return nil
	}
	affinity := e.loadThemeAffinity()
	if len(preferences) == 0 && len(affinity) == 0 {
		return nil
	}
	out := make(map[string]float64, len(items))
	for _, item := range items {
		slug := clustering.LabelForItems([]trss.Item{item}).Slug
		explicit := ranking.ThemeMultiplier(slug, preferences)
		if explicit == 0 {
			out[item.ID] = 0 // muted theme: stop here so behavior can't override
			continue
		}
		out[item.ID] = explicit * ranking.AffinityFactor(affinity[slug])
	}
	return out
}

// loadThemeAffinity reads the behavioral theme scores written by
// personalize.Learn. Empty/error returns an empty map; downstream
// treats absent keys as zero (neutral).
func (e *Engine) loadThemeAffinity() map[string]float64 {
	if e.Store == nil {
		return nil
	}
	a, err := e.Store.ListAffinity(store.AffinityKindTheme)
	if err != nil {
		return nil
	}
	return a
}

// loadSourceWeights retrieves configured weights from the sources
// table and blends in behavioral source affinity. Affinity acts as
// a multiplier on top of the configured weight so an explicit
// weight=0 still suppresses the source (admin override beats
// learned signal), while sources the user opens often get a gentle
// lift.
func (e *Engine) loadSourceWeights() map[string]float64 {
	weights := map[string]float64{}
	sources, err := e.Store.ListSources()
	if err != nil {
		return weights
	}
	for _, s := range sources {
		if s.Weight != 0 {
			weights[s.Name] = s.Weight
		}
	}

	affinity, err := e.Store.ListAffinity(store.AffinityKindSource)
	if err != nil || len(affinity) == 0 {
		return weights
	}
	for name, score := range affinity {
		base := weights[name]
		if base == 0 {
			base = 1.0
		}
		weights[name] = base * ranking.AffinityFactor(score)
	}
	return weights
}

// countSources counts distinct sources in items.
func (e *Engine) countSources(items []trss.Item) int {
	seen := map[string]bool{}
	for _, item := range items {
		seen[item.Source.Name] = true
	}
	return len(seen)
}

// buildSections groups items by source for the digest.
func (e *Engine) buildSections(items []trss.Item) []trss.DigestSection {
	sectionMap := map[string]*trss.DigestSection{}
	var order []string

	for _, item := range items {
		name := item.Source.Name
		sec, ok := sectionMap[name]
		if !ok {
			sec = &trss.DigestSection{Name: name, Icon: item.Source.Icon}
			sectionMap[name] = sec
			order = append(order, name)
		}
		sec.ItemIDs = append(sec.ItemIDs, item.ID)
	}

	sections := make([]trss.DigestSection, 0, len(order))
	for _, name := range order {
		sections = append(sections, *sectionMap[name])
	}
	return sections
}

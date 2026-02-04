package curation

import (
	"sort"
	"time"

	"github.com/jcornudella/hotbrew/internal/store"
	"github.com/jcornudella/hotbrew/pkg/trss"
)

// Engine orchestrates the curation pipeline.
type Engine struct {
	Store    *store.Store
	Limits   DiversityLimits
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
// 4. Score (recency * source_weight * engagement * boost)
// 5. Sort by score
// 6. Enforce diversity limits
// 7. Package as trss.Digest
func (e *Engine) GenerateDigest(window time.Duration, maxItems int, title string) (*trss.Digest, error) {
	// 1. Load items
	items, err := e.Store.ListItems(store.ItemFilter{
		Since: window,
	})
	if err != nil {
		return nil, err
	}
	totalConsidered := len(items)

	// 2. Load and apply rules
	rules, _ := e.Store.ListRules()
	filtered, boosts := ApplyRules(items, rules)
	rulesApplied := CountAppliedRules(totalConsidered, len(filtered), boosts)

	// 3. Dedup
	deduped := Dedup(filtered, e.Store)
	itemsDeduped := len(filtered) - len(deduped)

	// 4. Get source weights from store
	sourceWeights := e.loadSourceWeights()

	// 5. Score
	scored := ScoreItems(deduped, sourceWeights, boosts)

	// 6. Sort by score descending
	sort.Slice(scored, func(i, j int) bool {
		return scored[i].Score > scored[j].Score
	})

	// 7. Enforce diversity
	diverse := EnforceDiversity(scored, e.Limits, maxItems)

	// Update computed scores in store
	for _, item := range diverse {
		e.Store.UpdateScore(item.ID, item.Score)
	}

	// Build digest
	windowStr := window.String()
	digest := trss.NewDigest(title, windowStr, maxItems)
	digest.Items = diverse
	digest.ItemCount = len(diverse)
	digest.Meta = trss.DigestMeta{
		SourcesSynced:   e.countSources(diverse),
		ItemsConsidered: totalConsidered,
		ItemsDeduped:    itemsDeduped,
		RulesApplied:    rulesApplied,
	}

	// Build sections by source
	digest.Sections = e.buildSections(diverse)

	return digest, nil
}

// loadSourceWeights retrieves weights from the sources table.
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
			sec = &trss.DigestSection{
				Name: name,
				Icon: item.Source.Icon,
			}
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

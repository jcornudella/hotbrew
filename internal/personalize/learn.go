// Package personalize closes the loop between captured user signals
// (feedback_events, item_state) and the ranker. It reads the recent
// event log, scores each interaction, decays older events, aggregates
// by theme/source/domain, and writes the result to the affinity
// table. The ranker reads from affinity each time it builds a
// digest, so the next briefing tilts toward the topics, sources, and
// domains the user actually engages with.
//
// Design choices:
//   - Recompute from scratch each run (idempotent, debuggable).
//   - Keep behavioral scores in a separate table from explicit
//     theme_preferences so user follow/mute always wins.
//   - Decay is exponential with a 14-day half-life: a one-month-old
//     save still counts but is roughly a quarter of yesterday's.
//   - Mute-domain events use a large negative weight so a single
//     mute reliably suppresses the domain even if older saves
//     pulled it up.
package personalize

import (
	"math"
	"net/url"
	"strings"
	"time"

	"github.com/jcornudella/hotbrew/internal/clustering"
	"github.com/jcornudella/hotbrew/internal/store"
	"github.com/jcornudella/hotbrew/pkg/trss"
)

// Defaults that callers can override via Options. The values are
// chosen so a single save outweighs ~3 opens, a mute reliably blocks,
// and signal from a month ago contributes a quarter of today's.
const (
	DefaultLookback  = 90 * 24 * time.Hour
	DefaultHalfLife  = 14 * 24 * time.Hour
	DefaultThemeCap  = 1.5
	DefaultSourceCap = 1.5
)

// Options tunes the learn pass. Zero values fall through to defaults
// so callers only set the knobs they care about (typically: nothing).
type Options struct {
	Lookback time.Duration
	HalfLife time.Duration
	Now      time.Time
}

// Diff reports what Learn changed in the affinity table — the small
// data structure behind explainable personalization. Top/Bottom
// surface the strongest positive and negative signals so a `taste`
// view can render "you've been into X, less so Y" without rerunning
// the math.
type Diff struct {
	Themes  KindDiff
	Sources KindDiff
	Domains KindDiff
	Events  int
}

// KindDiff is the per-kind portion of Diff. Top/Bottom are sorted
// descending/ascending by score and trimmed to a small N.
type KindDiff struct {
	Total  int
	Top    []store.AffinityRow
	Bottom []store.AffinityRow
}

// Per-event-type weights. Save dominates because it's the most
// deliberate signal; open is mid-strength; read is weakest because
// it can be passive (auto-mark-read on open). Mute is large negative.
var actionWeights = map[string]float64{
	store.FeedbackActionSave:          3.0,
	store.FeedbackActionOpen:          1.0,
	store.FeedbackActionRead:          0.5,
	store.FeedbackActionExplainViewed: 0.3,
	store.FeedbackActionUnread:        -0.3,
	store.FeedbackActionMuteDomain:    -10.0,
}

// Learn reads recent feedback_events, aggregates by theme/source/domain,
// applies time decay, normalizes, and writes the result to the
// affinity table. Returns a Diff describing the new state.
func Learn(st *store.Store, opts Options) (Diff, error) {
	if opts.Lookback == 0 {
		opts.Lookback = DefaultLookback
	}
	if opts.HalfLife == 0 {
		opts.HalfLife = DefaultHalfLife
	}
	if opts.Now.IsZero() {
		opts.Now = time.Now()
	}

	events, err := st.ListFeedbackEvents(0)
	if err != nil {
		return Diff{}, err
	}

	cutoff := opts.Now.Add(-opts.Lookback)
	itemIDs := collectItemIDs(events, cutoff)
	items, err := loadItems(st, itemIDs)
	if err != nil {
		return Diff{}, err
	}

	themeScores := map[string]float64{}
	sourceScores := map[string]float64{}
	domainScores := map[string]float64{}
	usedEvents := 0

	for _, ev := range events {
		if ev.CreatedAt.Before(cutoff) {
			continue
		}
		w, ok := actionWeights[ev.Action]
		if !ok {
			continue
		}
		decay := decayFactor(opts.Now.Sub(ev.CreatedAt), opts.HalfLife)
		weighted := w * decay
		usedEvents++

		// mute_domain carries the domain in `target`, not via item lookup.
		if ev.Action == store.FeedbackActionMuteDomain && ev.Target != "" {
			domainScores[normalizeDomain(ev.Target)] += weighted
			continue
		}

		item, ok := items[ev.ItemID]
		if !ok {
			continue
		}
		themeScores[clustering.LabelForItems([]trss.Item{*item}).Slug] += weighted
		if name := item.Source.Name; name != "" {
			sourceScores[name] += weighted
		}
		if d := domainOf(*item); d != "" {
			domainScores[d] += weighted
		}
	}

	themes := normalize(themeScores, DefaultThemeCap)
	sources := normalize(sourceScores, DefaultSourceCap)
	domains := normalize(domainScores, 0) // no cap — mute should bite hard

	if err := st.ReplaceAffinity(store.AffinityKindTheme, themes); err != nil {
		return Diff{}, err
	}
	if err := st.ReplaceAffinity(store.AffinityKindSource, sources); err != nil {
		return Diff{}, err
	}
	if err := st.ReplaceAffinity(store.AffinityKindDomain, domains); err != nil {
		return Diff{}, err
	}

	return Diff{
		Themes:  summarize(themes, 5),
		Sources: summarize(sources, 5),
		Domains: summarize(domains, 5),
		Events:  usedEvents,
	}, nil
}

func collectItemIDs(events []store.FeedbackEvent, cutoff time.Time) []string {
	seen := map[string]struct{}{}
	out := []string{}
	for _, ev := range events {
		if ev.CreatedAt.Before(cutoff) {
			continue
		}
		if ev.ItemID == "" {
			continue
		}
		if _, ok := seen[ev.ItemID]; ok {
			continue
		}
		seen[ev.ItemID] = struct{}{}
		out = append(out, ev.ItemID)
	}
	return out
}

// loadItems hydrates a set of items by ID. We GetItem one at a time;
// the count is bounded by event volume (tens to low hundreds in
// realistic personal use), so a batch SELECT IN (?...) isn't worth
// the SQL plumbing here.
func loadItems(st *store.Store, ids []string) (map[string]*trss.Item, error) {
	out := make(map[string]*trss.Item, len(ids))
	for _, id := range ids {
		item, err := st.GetItem(id)
		if err != nil {
			// Item may have been pruned since the event was logged.
			// Don't fail the whole learn pass — skip and continue.
			continue
		}
		out[id] = item
	}
	return out, nil
}

func decayFactor(age, halfLife time.Duration) float64 {
	if age <= 0 || halfLife <= 0 {
		return 1
	}
	return math.Exp(-math.Ln2 * float64(age) / float64(halfLife))
}

// normalize converts raw additive scores into a comparable [-cap, cap]
// range by dividing by the max absolute value. cap=0 means no rescale.
// Empty input returns nil.
func normalize(scores map[string]float64, cap float64) []store.AffinityRow {
	if len(scores) == 0 {
		return nil
	}
	out := make([]store.AffinityRow, 0, len(scores))
	maxAbs := 0.0
	for _, v := range scores {
		if a := math.Abs(v); a > maxAbs {
			maxAbs = a
		}
	}
	for k, v := range scores {
		score := v
		if cap > 0 && maxAbs > 0 {
			score = (v / maxAbs) * cap
		}
		out = append(out, store.AffinityRow{Key: k, Score: score})
	}
	return out
}

func summarize(rows []store.AffinityRow, n int) KindDiff {
	if len(rows) == 0 {
		return KindDiff{}
	}
	sorted := make([]store.AffinityRow, len(rows))
	copy(sorted, rows)
	// Sort descending without importing sort here would be silly;
	// the caller is on the cold path, so use the stdlib sort.
	sortRowsDesc(sorted)

	d := KindDiff{Total: len(sorted)}
	if len(sorted) <= n {
		d.Top = sorted
	} else {
		d.Top = sorted[:n]
	}
	for _, r := range sorted {
		if r.Score >= 0 {
			break
		}
		d.Bottom = append([]store.AffinityRow{r}, d.Bottom...)
		if len(d.Bottom) >= n {
			break
		}
	}
	return d
}

func domainOf(item trss.Item) string {
	raw := item.URLCanonical
	if raw == "" {
		raw = item.URL
	}
	if raw == "" {
		return ""
	}
	parsed, err := url.Parse(raw)
	if err != nil {
		return ""
	}
	return normalizeDomain(parsed.Hostname())
}

func normalizeDomain(host string) string {
	return strings.TrimPrefix(strings.ToLower(host), "www.")
}

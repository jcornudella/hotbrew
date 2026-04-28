package app

import (
	"context"

	"github.com/jcornudella/hotbrew/internal/briefing"
	"github.com/jcornudella/hotbrew/internal/clustering"
	"github.com/jcornudella/hotbrew/internal/config"
	"github.com/jcornudella/hotbrew/internal/curation"
	"github.com/jcornudella/hotbrew/internal/intel"
	"github.com/jcornudella/hotbrew/internal/store"
	"github.com/jcornudella/hotbrew/pkg/trss"
)

// BriefingService builds a canonical briefing for CLI and TUI surfaces.
type BriefingService struct {
	store    *store.Store
	config   *config.Config
	syncFunc func(context.Context) error
}

// BuildOptions controls how the briefing should be built.
type BuildOptions struct {
	SyncIfEmpty bool
	ForceSync   bool
}

// NewBriefingService creates a new service instance.
func NewBriefingService(st *store.Store, cfg *config.Config) *BriefingService {
	service := &BriefingService{store: st, config: cfg}
	service.syncFunc = func(ctx context.Context) error {
		return syncStore(ctx, st, cfg)
	}
	return service
}

// Build creates a briefing from the local store.
func (s *BriefingService) Build(ctx context.Context, opts BuildOptions) (*intel.Briefing, error) {
	digest, err := s.BuildDigest(ctx, opts)
	if err != nil {
		return nil, err
	}

	b := briefingFromDigest(digest)
	if digest != nil && len(digest.Items) > 0 {
		b.Clusters = clustering.ClusterWith(ctx, digest.Items, clustering.NewLLMLabeler(s.config))
		briefing.Assemble(b)
		briefing.Balance(b, balanceLimitsFromConfig(s.config))
		if s.store != nil {
			_ = s.store.ReplaceClusters(b.Clusters)
			s.hydrateBreakdowns(b)
		}
	}
	return b, nil
}

// hydrateBreakdowns fills ScoreBreakdown on each briefing item from
// the persisted features snapshot. Without this the explain/why
// commands would only see the final score — not the per-signal
// reasoning that makes explanations useful.
func (s *BriefingService) hydrateBreakdowns(b *intel.Briefing) {
	ids := make([]string, len(b.Items))
	for i, item := range b.Items {
		ids[i] = item.Item.ID
	}
	features, err := s.store.ListItemFeatures(ids)
	if err != nil || len(features) == 0 {
		return
	}
	for i, item := range b.Items {
		f, ok := features[item.Item.ID]
		if !ok {
			continue
		}
		b.Items[i].Breakdown = intel.ScoreBreakdown{
			Freshness:     f.Signals.Freshness,
			Authority:     f.Signals.SourceAuthority,
			Engagement:    f.Signals.Engagement,
			Resonance:     f.Signals.Resonance,
			Novelty:       f.Signals.Novelty,
			TopicMatch:    f.Signals.TopicMatch,
			RepeatPenalty: f.Signals.RepeatPenalty,
			ContentFit:    f.Signals.ContentFit,
			Final:         item.Score,
		}
	}
}

// BuildDigest generates the current curated digest via the canonical service.
func (s *BriefingService) BuildDigest(ctx context.Context, opts BuildOptions) (*trss.Digest, error) {
	if s == nil || s.store == nil || s.config == nil {
		return trss.NewDigest("Hotbrew Digest", "24h", 25), nil
	}

	if opts.ForceSync {
		if err := s.runSync(ctx); err != nil {
			return nil, err
		}
	}

	digest, err := s.generateDigest()
	if err != nil {
		return nil, err
	}
	if !opts.SyncIfEmpty || (digest != nil && len(digest.Items) > 0) {
		return digest, nil
	}

	if err := s.runSync(ctx); err != nil {
		return nil, err
	}
	return s.generateDigest()
}

func (s *BriefingService) runSync(ctx context.Context) error {
	if s.syncFunc == nil {
		return nil
	}
	return s.syncFunc(ctx)
}

func (s *BriefingService) generateDigest() (*trss.Digest, error) {
	engine := curation.NewEngine(s.store)
	return engine.GenerateDigest(s.config.GetDigestWindow(), s.config.GetDigestMax(), "Hotbrew Digest")
}

// briefingFromDigest copies non-assembly fields off the raw digest.
// Sections are intentionally skipped here — they are rebuilt by
// briefing.Assemble once clusters exist, so the theme-based output
// always wins over the digest's source-based grouping.
func briefingFromDigest(digest *trss.Digest) *intel.Briefing {
	b := &intel.Briefing{Title: "Hotbrew Digest"}
	if digest == nil {
		return b
	}

	b.Title = digest.Title
	b.Date = digest.GeneratedAt
	b.Meta = intel.BriefingMeta{
		SourcesSynced:   digest.Meta.SourcesSynced,
		ItemsConsidered: digest.Meta.ItemsConsidered,
		ItemsDeduped:    digest.Meta.ItemsDeduped,
		RulesApplied:    digest.Meta.RulesApplied,
	}

	for _, item := range digest.Items {
		b.Items = append(b.Items, intel.ScoredItem{
			Item:      intel.FromTRSSItem(item),
			Score:     item.Score,
			Breakdown: intel.ScoreBreakdown{Final: item.Score},
		})
	}
	return b
}

func balanceLimitsFromConfig(cfg *config.Config) briefing.BalanceLimits {
	limits := briefing.DefaultBalanceLimits()
	if cfg == nil || cfg.Briefing == nil {
		return limits
	}
	if cfg.Briefing.MaxClustersPerTheme > 0 {
		limits.MaxClustersPerTheme = cfg.Briefing.MaxClustersPerTheme
	}
	if cfg.Briefing.MaxLeadsPerDomain > 0 {
		limits.MaxLeadsPerDomain = cfg.Briefing.MaxLeadsPerDomain
	}
	if cfg.Briefing.MaxTotalClusters > 0 {
		limits.MaxTotalClusters = cfg.Briefing.MaxTotalClusters
	}
	return limits
}

package app

import (
	"context"

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

	briefing := briefingFromDigest(digest)
	if digest != nil && len(digest.Items) > 0 {
		clusters := clustering.Cluster(digest.Items)
		briefing.Clusters = clusters
		if s.store != nil {
			_ = s.store.ReplaceClusters(clusters)
		}
	}
	return briefing, nil
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

func briefingFromDigest(digest *trss.Digest) *intel.Briefing {
	briefing := &intel.Briefing{Title: "Hotbrew Digest"}
	if digest == nil {
		return briefing
	}

	briefing.Title = digest.Title
	briefing.Date = digest.GeneratedAt
	briefing.Meta = intel.BriefingMeta{
		SourcesSynced:   digest.Meta.SourcesSynced,
		ItemsConsidered: digest.Meta.ItemsConsidered,
		ItemsDeduped:    digest.Meta.ItemsDeduped,
		RulesApplied:    digest.Meta.RulesApplied,
	}

	for _, item := range digest.Items {
		briefing.Items = append(briefing.Items, intel.ScoredItem{
			Item:      intel.FromTRSSItem(item),
			Score:     item.Score,
			Breakdown: intel.ScoreBreakdown{Final: item.Score},
		})
	}

	for _, section := range digest.Sections {
		briefing.Sections = append(briefing.Sections, intel.BriefingSection{
			Name:       section.Name,
			Kind:       section.Name,
			ClusterIDs: append([]string(nil), section.ItemIDs...),
		})
	}

	return briefing
}

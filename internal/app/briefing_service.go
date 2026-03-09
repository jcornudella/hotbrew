package app

import (
	"context"

	"github.com/jcornudella/hotbrew/internal/config"
	"github.com/jcornudella/hotbrew/internal/curation"
	"github.com/jcornudella/hotbrew/internal/intel"
	"github.com/jcornudella/hotbrew/internal/store"
	"github.com/jcornudella/hotbrew/pkg/trss"
)

// BriefingService builds a canonical briefing for CLI and TUI surfaces.
type BriefingService struct {
	store  *store.Store
	config *config.Config
}

// BuildOptions controls how the briefing should be built.
type BuildOptions struct {
	SyncIfEmpty bool
	ForceSync   bool
}

// NewBriefingService creates a new service instance.
func NewBriefingService(st *store.Store, cfg *config.Config) *BriefingService {
	return &BriefingService{store: st, config: cfg}
}

// Build creates a briefing from the local store.
func (s *BriefingService) Build(ctx context.Context, opts BuildOptions) (*intel.Briefing, error) {
	digest, err := s.BuildDigest(ctx, opts)
	if err != nil {
		return nil, err
	}
	return briefingFromDigest(digest), nil
}

// BuildDigest generates the current curated digest via the canonical service.
func (s *BriefingService) BuildDigest(_ context.Context, _ BuildOptions) (*trss.Digest, error) {
	if s == nil || s.store == nil || s.config == nil {
		return trss.NewDigest("Hotbrew Digest", "24h", 25), nil
	}

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

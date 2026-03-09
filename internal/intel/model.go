package intel

import "time"

// IntelItem is the normalized durable content unit used by briefing logic.
type IntelItem struct {
	ID           string
	CanonicalURL string
	SourceKey    string
	SourceName   string
	SourceType   string
	SourceIcon   string
	Title        string
	Summary      string
	Body         string
	Author       string
	PublishedAt  time.Time
	FetchedAt    time.Time
	Tags         []string
	Topics       []string
	Language     string
	ContentType  string
	Engagement   EngagementSignals
	Signals      ItemSignals
	Metadata     map[string]string
}

// EngagementSignals captures source-specific engagement metrics in a normalized form.
type EngagementSignals struct {
	Points   float64
	Comments float64
	Stars    float64
	Forks    float64
	Velocity float64
}

// ItemSignals stores derived ranking features for an item.
type ItemSignals struct {
	Freshness       float64
	SourceAuthority float64
	Engagement      float64
	Resonance       float64
	Novelty         float64
	TopicMatch      float64
	RepeatPenalty   float64
	ContentFit      float64
}

// ScoreBreakdown stores explainable ranking primitives and the final score.
type ScoreBreakdown struct {
	Freshness     float64
	Authority     float64
	Engagement    float64
	Resonance     float64
	Novelty       float64
	TopicMatch    float64
	RepeatPenalty float64
	ContentFit    float64
	Final         float64
}

// ScoredItem wraps an item with its final score and ranking breakdown.
type ScoredItem struct {
	Item      IntelItem
	Score     float64
	Breakdown ScoreBreakdown
}

// ThemeCluster groups related items under a shared topic.
type ThemeCluster struct {
	ID             string
	Label          string
	Slug           string
	Representative string
	ItemIDs        []string
	Themes         []string
	Score          float64
	WhyItMatters   string
}

// BriefingSection groups clusters under a presentation section.
type BriefingSection struct {
	Name       string
	Kind       string
	ClusterIDs []string
}

// BriefingMeta holds summary metadata for a generated briefing.
type BriefingMeta struct {
	SourcesSynced   int
	ItemsConsidered int
	ItemsDeduped    int
	RulesApplied    int
}

// Briefing is the canonical output consumed by TUI/CLI surfaces.
type Briefing struct {
	Date     time.Time
	Title    string
	Sections []BriefingSection
	Clusters []ThemeCluster
	Items    []ScoredItem
	Meta     BriefingMeta
}

// FeedbackEvent records a user interaction for later personalization.
type FeedbackEvent struct {
	ID        string
	ItemID    string
	ClusterID string
	Action    string
	Value     string
	CreatedAt time.Time
}

// PreferenceProfile stores durable user preferences used by ranking.
type PreferenceProfile struct {
	FollowedThemes []string
	MutedThemes    []string
	MutedDomains   []string
	PreferredTypes []string
}

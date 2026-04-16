package store

import (
	"strings"
	"time"

	"github.com/jcornudella/hotbrew/internal/intel"
)

// ItemFeatures is a persisted snapshot of derived ranking signals for one item.
type ItemFeatures struct {
	ItemID     string
	Signals    intel.ItemSignals
	ComputedAt time.Time
}

// UpsertItemFeatures inserts or replaces the derived features for an item.
func (s *Store) UpsertItemFeatures(itemID string, signals intel.ItemSignals) error {
	_, err := s.db.Exec(`
		INSERT INTO item_features
			(item_id, freshness, authority, engagement, resonance, novelty,
			 topic_match, repeat_penalty, content_fit, computed_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, datetime('now'))
		ON CONFLICT(item_id) DO UPDATE SET
			freshness      = excluded.freshness,
			authority      = excluded.authority,
			engagement     = excluded.engagement,
			resonance      = excluded.resonance,
			novelty        = excluded.novelty,
			topic_match    = excluded.topic_match,
			repeat_penalty = excluded.repeat_penalty,
			content_fit    = excluded.content_fit,
			computed_at    = excluded.computed_at`,
		itemID,
		signals.Freshness, signals.SourceAuthority, signals.Engagement,
		signals.Resonance, signals.Novelty, signals.TopicMatch,
		signals.RepeatPenalty, signals.ContentFit,
	)
	return err
}

// GetItemFeatures returns the persisted features for a single item, or nil if absent.
func (s *Store) GetItemFeatures(itemID string) (*ItemFeatures, error) {
	row := s.db.QueryRow(`
		SELECT item_id, freshness, authority, engagement, resonance, novelty,
			topic_match, repeat_penalty, content_fit, computed_at
		FROM item_features WHERE item_id = ?`, itemID)
	return scanFeaturesRow(row)
}

// ListItemFeatures returns persisted features for the given item ids.
// Items without features are omitted.
func (s *Store) ListItemFeatures(ids []string) (map[string]ItemFeatures, error) {
	result := map[string]ItemFeatures{}
	if len(ids) == 0 {
		return result, nil
	}

	placeholders := strings.Repeat("?,", len(ids)-1) + "?"
	args := make([]any, len(ids))
	for i, id := range ids {
		args[i] = id
	}

	rows, err := s.db.Query(`
		SELECT item_id, freshness, authority, engagement, resonance, novelty,
			topic_match, repeat_penalty, content_fit, computed_at
		FROM item_features WHERE item_id IN (`+placeholders+`)`, args...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	for rows.Next() {
		features, err := scanFeaturesRow(rows)
		if err != nil {
			return nil, err
		}
		if features != nil {
			result[features.ItemID] = *features
		}
	}
	return result, rows.Err()
}

type scanner interface {
	Scan(dest ...any) error
}

func scanFeaturesRow(r scanner) (*ItemFeatures, error) {
	var f ItemFeatures
	var computedAt string
	err := r.Scan(
		&f.ItemID,
		&f.Signals.Freshness, &f.Signals.SourceAuthority, &f.Signals.Engagement,
		&f.Signals.Resonance, &f.Signals.Novelty, &f.Signals.TopicMatch,
		&f.Signals.RepeatPenalty, &f.Signals.ContentFit,
		&computedAt,
	)
	if err != nil {
		return nil, err
	}
	f.ComputedAt, _ = time.Parse(time.RFC3339, computedAt)
	if f.ComputedAt.IsZero() {
		// SQLite datetime('now') returns "YYYY-MM-DD HH:MM:SS"; accept that too.
		f.ComputedAt, _ = time.Parse("2006-01-02 15:04:05", computedAt)
	}
	return &f, nil
}

package store

// Affinity kinds. The personalize package writes one row per
// (kind, key) tuple; ranking reads a snapshot per kind.
const (
	AffinityKindTheme  = "theme"
	AffinityKindSource = "source"
	AffinityKindDomain = "domain"
)

// AffinityRow is one inferred preference. Score is unbounded but is
// produced by personalize.Learn in a normalized range — typically
// roughly [-1, 1] for themes and sources, with stronger negatives for
// muted domains.
type AffinityRow struct {
	Kind  string
	Key   string
	Score float64
}

// UpsertAffinity replaces the score for (kind, key). The personalize
// package recomputes scores from scratch each run, so an upsert is
// correct — there is no incremental signal to preserve.
func (s *Store) UpsertAffinity(kind, key string, score float64) error {
	if kind == "" || key == "" {
		return nil
	}
	_, err := s.db.Exec(`
		INSERT INTO affinity (kind, key, score, updated_at)
		VALUES (?, ?, ?, datetime('now'))
		ON CONFLICT(kind, key) DO UPDATE SET
			score      = excluded.score,
			updated_at = excluded.updated_at`,
		kind, key, score,
	)
	return err
}

// ReplaceAffinity wipes every row of a given kind and writes the new
// set in one transaction. personalize.Learn uses this to publish a
// fresh snapshot — partial updates would leave stale themes lingering
// after a user's interests shifted.
func (s *Store) ReplaceAffinity(kind string, rows []AffinityRow) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.Exec(`DELETE FROM affinity WHERE kind = ?`, kind); err != nil {
		return err
	}
	for _, r := range rows {
		if r.Key == "" {
			continue
		}
		if _, err := tx.Exec(`
			INSERT INTO affinity (kind, key, score, updated_at)
			VALUES (?, ?, ?, datetime('now'))`,
			kind, r.Key, r.Score,
		); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// ListAffinity returns key → score for one kind. Empty result is
// fine — callers treat absent keys as zero (neutral).
func (s *Store) ListAffinity(kind string) (map[string]float64, error) {
	rows, err := s.db.Query(`SELECT key, score FROM affinity WHERE kind = ?`, kind)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	out := map[string]float64{}
	for rows.Next() {
		var key string
		var score float64
		if err := rows.Scan(&key, &score); err != nil {
			return nil, err
		}
		out[key] = score
	}
	return out, rows.Err()
}

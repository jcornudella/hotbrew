package store

import (
	"strings"
	"time"

	"github.com/jcornudella/hotbrew/internal/intel"
)

// ReplaceClusters atomically wipes and rewrites the cluster tables.
// Clusters are recomputed each briefing cycle, so incremental updates
// would add complexity without value.
func (s *Store) ReplaceClusters(clusters []intel.ThemeCluster) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.Exec(`DELETE FROM cluster_items`); err != nil {
		return err
	}
	if _, err := tx.Exec(`DELETE FROM clusters`); err != nil {
		return err
	}

	for _, c := range clusters {
		if _, err := tx.Exec(`
			INSERT INTO clusters
				(id, label, slug, representative_item_id, score, why_it_matters, generated_at)
			VALUES (?, ?, ?, ?, ?, ?, datetime('now'))`,
			c.ID, c.Label, c.Slug, c.Representative, c.Score, c.WhyItMatters,
		); err != nil {
			return err
		}
		for _, itemID := range c.ItemIDs {
			if _, err := tx.Exec(
				`INSERT INTO cluster_items (cluster_id, item_id) VALUES (?, ?)`,
				c.ID, itemID,
			); err != nil {
				return err
			}
		}
	}

	return tx.Commit()
}

// ListClusters returns every persisted cluster with its member ids,
// ordered by score desc then id asc for a stable surface.
func (s *Store) ListClusters() ([]intel.ThemeCluster, error) {
	rows, err := s.db.Query(`
		SELECT id, label, slug, COALESCE(representative_item_id, ''),
			score, COALESCE(why_it_matters, ''), generated_at
		FROM clusters
		ORDER BY score DESC, id ASC`)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var clusters []intel.ThemeCluster
	for rows.Next() {
		var c intel.ThemeCluster
		var generatedAt string
		if err := rows.Scan(
			&c.ID, &c.Label, &c.Slug, &c.Representative,
			&c.Score, &c.WhyItMatters, &generatedAt,
		); err != nil {
			return nil, err
		}
		c.Themes = []string{c.Slug}
		_ = generatedAt // reserved for future display
		clusters = append(clusters, c)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	if len(clusters) == 0 {
		return nil, nil
	}

	ids := make([]string, len(clusters))
	for i, c := range clusters {
		ids[i] = c.ID
	}
	members, err := s.listClusterMembers(ids)
	if err != nil {
		return nil, err
	}
	for i := range clusters {
		clusters[i].ItemIDs = members[clusters[i].ID]
	}
	return clusters, nil
}

func (s *Store) listClusterMembers(clusterIDs []string) (map[string][]string, error) {
	placeholders := strings.Repeat("?,", len(clusterIDs)-1) + "?"
	args := make([]any, len(clusterIDs))
	for i, id := range clusterIDs {
		args[i] = id
	}

	rows, err := s.db.Query(
		`SELECT cluster_id, item_id FROM cluster_items WHERE cluster_id IN (`+placeholders+`)
		 ORDER BY cluster_id, item_id`,
		args...,
	)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	result := make(map[string][]string, len(clusterIDs))
	for rows.Next() {
		var clusterID, itemID string
		if err := rows.Scan(&clusterID, &itemID); err != nil {
			return nil, err
		}
		result[clusterID] = append(result[clusterID], itemID)
	}
	return result, rows.Err()
}

// GeneratedAtFor returns the time a cluster was persisted, primarily
// for diagnostics. Returns the zero time if not found.
func (s *Store) GeneratedAtFor(clusterID string) time.Time {
	var generatedAt string
	if err := s.db.QueryRow(
		`SELECT generated_at FROM clusters WHERE id = ?`, clusterID,
	).Scan(&generatedAt); err != nil {
		return time.Time{}
	}
	if t, err := time.Parse(time.RFC3339, generatedAt); err == nil {
		return t
	}
	t, _ := time.Parse("2006-01-02 15:04:05", generatedAt)
	return t
}

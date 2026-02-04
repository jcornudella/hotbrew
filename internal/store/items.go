package store

import (
	"encoding/json"
	"time"

	"github.com/jcornudella/hotbrew/pkg/trss"
)

// InsertItem stores a TRSS item, skipping duplicates.
func (s *Store) InsertItem(item trss.Item, sourceID int) error {
	tags, _ := json.Marshal(item.Tags)
	engagement, _ := json.Marshal(item.Engagement)
	meta, _ := json.Marshal(item.Meta)

	_, err := s.db.Exec(`
		INSERT OR IGNORE INTO items
			(id, fingerprint, title, url, url_canonical, source_id, source_name,
			 published_at, fetched_at, summary, body, tags, score_raw, engagement, meta)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		item.ID, item.Fingerprint, item.Title, item.URL, item.URLCanonical,
		sourceID, item.Source.Name,
		item.PublishedAt.UTC().Format(time.RFC3339),
		item.FetchedAt.UTC().Format(time.RFC3339),
		item.Summary, item.Body, string(tags),
		item.Score, string(engagement), string(meta),
	)
	return err
}

// ItemFilter holds query parameters for listing items.
type ItemFilter struct {
	Unread     bool
	SourceName string
	Since      time.Duration
	Limit      int
}

// ListItems queries items with filters, ordered by score.
func (s *Store) ListItems(f ItemFilter) ([]trss.Item, error) {
	query := `SELECT i.id, i.fingerprint, i.title, i.url, i.url_canonical,
		i.source_name, i.published_at, i.fetched_at, i.summary, i.body,
		i.tags, i.score_raw, i.score_computed, i.engagement, i.meta,
		COALESCE(s.state, 'unread') as state
		FROM items i
		LEFT JOIN item_state s ON i.id = s.item_id
		WHERE 1=1`

	var args []any

	if f.Unread {
		query += " AND (s.state IS NULL OR s.state = 'unread')"
	}
	if f.SourceName != "" {
		query += " AND i.source_name = ?"
		args = append(args, f.SourceName)
	}
	if f.Since > 0 {
		cutoff := time.Now().Add(-f.Since).UTC().Format(time.RFC3339)
		query += " AND i.fetched_at >= ?"
		args = append(args, cutoff)
	}

	query += " ORDER BY i.score_computed DESC, i.published_at DESC"

	if f.Limit > 0 {
		query += " LIMIT ?"
		args = append(args, f.Limit)
	}

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []trss.Item
	for rows.Next() {
		var item trss.Item
		var pubAt, fetchAt string
		var tagsJSON, engJSON, metaJSON string
		var state string

		err := rows.Scan(
			&item.ID, &item.Fingerprint, &item.Title, &item.URL, &item.URLCanonical,
			&item.Source.Name, &pubAt, &fetchAt,
			&item.Summary, &item.Body, &tagsJSON,
			&item.Score, &item.Score, &engJSON, &metaJSON, &state,
		)
		if err != nil {
			continue
		}

		item.PublishedAt, _ = time.Parse(time.RFC3339, pubAt)
		item.FetchedAt, _ = time.Parse(time.RFC3339, fetchAt)
		json.Unmarshal([]byte(tagsJSON), &item.Tags)
		json.Unmarshal([]byte(engJSON), &item.Engagement)
		json.Unmarshal([]byte(metaJSON), &item.Meta)

		if item.Meta == nil {
			item.Meta = map[string]any{}
		}
		item.Meta["state"] = state

		items = append(items, item)
	}

	return items, rows.Err()
}

// GetItem retrieves a single item by ID or ID prefix.
func (s *Store) GetItem(idPrefix string) (*trss.Item, error) {
	items, err := s.ListItems(ItemFilter{Limit: 1})
	if err != nil {
		return nil, err
	}

	// Search by prefix
	row := s.db.QueryRow(`
		SELECT id, fingerprint, title, url, url_canonical, source_name,
			published_at, fetched_at, summary, body, tags, score_raw, engagement, meta
		FROM items WHERE id LIKE ? LIMIT 1`,
		idPrefix+"%",
	)

	var item trss.Item
	var pubAt, fetchAt string
	var tagsJSON, engJSON, metaJSON string

	err = row.Scan(
		&item.ID, &item.Fingerprint, &item.Title, &item.URL, &item.URLCanonical,
		&item.Source.Name, &pubAt, &fetchAt,
		&item.Summary, &item.Body, &tagsJSON,
		&item.Score, &engJSON, &metaJSON,
	)
	if err != nil {
		return nil, err
	}

	item.PublishedAt, _ = time.Parse(time.RFC3339, pubAt)
	item.FetchedAt, _ = time.Parse(time.RFC3339, fetchAt)
	json.Unmarshal([]byte(tagsJSON), &item.Tags)
	json.Unmarshal([]byte(engJSON), &item.Engagement)
	json.Unmarshal([]byte(metaJSON), &item.Meta)

	_ = items // suppress unused
	return &item, nil
}

// UpdateScore updates the computed score for an item.
func (s *Store) UpdateScore(id string, score float64) error {
	_, err := s.db.Exec("UPDATE items SET score_computed = ? WHERE id = ?", score, id)
	return err
}

// HasRecentItems checks if there are items fetched within the given duration.
func (s *Store) HasRecentItems(d time.Duration) bool {
	cutoff := time.Now().Add(-d).UTC().Format(time.RFC3339)
	var count int
	s.db.QueryRow("SELECT COUNT(*) FROM items WHERE fetched_at >= ?", cutoff).Scan(&count)
	return count > 0
}

// ItemCount returns the total number of items.
func (s *Store) ItemCount() int {
	var count int
	s.db.QueryRow("SELECT COUNT(*) FROM items").Scan(&count)
	return count
}

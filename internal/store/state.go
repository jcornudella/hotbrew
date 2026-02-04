package store

import "time"

// MarkRead marks an item as read.
func (s *Store) MarkRead(itemID string) error {
	_, err := s.db.Exec(`
		INSERT INTO item_state (item_id, state, opened_at, updated_at)
		VALUES (?, 'read', datetime('now'), datetime('now'))
		ON CONFLICT(item_id) DO UPDATE SET
			state = 'read', opened_at = datetime('now'), updated_at = datetime('now')`,
		itemID,
	)
	return err
}

// MarkSaved marks an item as saved.
func (s *Store) MarkSaved(itemID string) error {
	_, err := s.db.Exec(`
		INSERT INTO item_state (item_id, state, saved_at, updated_at)
		VALUES (?, 'saved', datetime('now'), datetime('now'))
		ON CONFLICT(item_id) DO UPDATE SET
			state = 'saved', saved_at = datetime('now'), updated_at = datetime('now')`,
		itemID,
	)
	return err
}

// MarkUnread marks an item as unread.
func (s *Store) MarkUnread(itemID string) error {
	_, err := s.db.Exec(
		"DELETE FROM item_state WHERE item_id = ?", itemID,
	)
	return err
}

// GetState returns the state of an item.
func (s *Store) GetState(itemID string) string {
	var state string
	err := s.db.QueryRow(
		"SELECT state FROM item_state WHERE item_id = ?", itemID,
	).Scan(&state)
	if err != nil {
		return "unread"
	}
	return state
}

// UnreadCount returns the number of unread items.
func (s *Store) UnreadCount() int {
	total := s.ItemCount()

	var readCount int
	s.db.QueryRow("SELECT COUNT(*) FROM item_state WHERE state != 'unread'").Scan(&readCount)

	return total - readCount
}

// CountByState returns item counts grouped by state.
func (s *Store) CountByState() map[string]int {
	counts := map[string]int{"unread": 0, "read": 0, "saved": 0}

	rows, err := s.db.Query("SELECT state, COUNT(*) FROM item_state GROUP BY state")
	if err != nil {
		return counts
	}
	defer rows.Close()

	for rows.Next() {
		var state string
		var count int
		if rows.Scan(&state, &count) == nil {
			counts[state] = count
		}
	}

	total := s.ItemCount()
	stateTotal := 0
	for _, c := range counts {
		stateTotal += c
	}
	counts["unread"] = total - stateTotal

	return counts
}

// InsertDedupEdge records a dedup relationship between two items.
func (s *Store) InsertDedupEdge(idA, idB string, confidence float64) error {
	// Ensure consistent ordering
	if idA > idB {
		idA, idB = idB, idA
	}
	_, err := s.db.Exec(`
		INSERT OR IGNORE INTO dedup_edges (item_id_a, item_id_b, confidence)
		VALUES (?, ?, ?)`,
		idA, idB, confidence,
	)
	return err
}

// GetDedupedIDs returns item IDs that are duplicates of the given item.
func (s *Store) GetDedupedIDs(itemID string) []string {
	rows, err := s.db.Query(`
		SELECT item_id_b FROM dedup_edges WHERE item_id_a = ?
		UNION
		SELECT item_id_a FROM dedup_edges WHERE item_id_b = ?`,
		itemID, itemID,
	)
	if err != nil {
		return nil
	}
	defer rows.Close()

	var ids []string
	for rows.Next() {
		var id string
		if rows.Scan(&id) == nil {
			ids = append(ids, id)
		}
	}
	return ids
}

// RecentItemsSince returns items fetched after the given time.
func (s *Store) RecentItemsSince(since time.Time) int {
	var count int
	s.db.QueryRow("SELECT COUNT(*) FROM items WHERE fetched_at >= ?",
		since.UTC().Format(time.RFC3339)).Scan(&count)
	return count
}

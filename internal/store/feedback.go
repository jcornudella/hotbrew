package store

// InsertFeedback stores a user rating (1-4) for digest issues.
func (s *Store) InsertFeedback(rating int, note string) error {
	_, err := s.db.Exec(
		`INSERT INTO feedback (rating, note) VALUES (?, ?)`,
		rating, note,
	)
	return err
}

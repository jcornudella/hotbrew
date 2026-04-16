package store

// Theme preference states. Constants are exported so callers at the
// CLI and ranking boundaries don't have to repeat magic strings —
// the SQLite CHECK constraint would catch typos eventually, but the
// compile-time guardrail here catches them instantly.
const (
	ThemeStateFollow = "follow"
	ThemeStateMute   = "mute"
)

// SetThemePreference upserts the preference for a theme slug.
// An existing row is overwritten — the table is a dictionary of
// current intent, not an event log (that role belongs to
// feedback_events).
func (s *Store) SetThemePreference(slug, state string) error {
	if slug == "" {
		return nil
	}
	_, err := s.db.Exec(`
		INSERT INTO theme_preferences (slug, state, updated_at)
		VALUES (?, ?, datetime('now'))
		ON CONFLICT(slug) DO UPDATE SET
			state      = excluded.state,
			updated_at = excluded.updated_at`,
		slug, state,
	)
	return err
}

// DeleteThemePreference clears the row for a slug (if any). Used by
// `hotbrew unfollow` to revert to neutral ranking, rather than by
// writing a third "neutral" state that confuses downstream queries.
func (s *Store) DeleteThemePreference(slug string) error {
	if slug == "" {
		return nil
	}
	_, err := s.db.Exec(`DELETE FROM theme_preferences WHERE slug = ?`, slug)
	return err
}

// ListThemePreferences returns slug → state for every configured
// theme. Ranking calls this once per digest build; the row count is
// bounded by the number of known slugs, so a plain map is fine.
func (s *Store) ListThemePreferences() (map[string]string, error) {
	rows, err := s.db.Query(`SELECT slug, state FROM theme_preferences`)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	out := map[string]string{}
	for rows.Next() {
		var slug, state string
		if err := rows.Scan(&slug, &state); err != nil {
			return nil, err
		}
		out[slug] = state
	}
	return out, rows.Err()
}

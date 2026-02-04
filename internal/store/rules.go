package store

// Rule represents a user-defined rule.
type Rule struct {
	ID      int
	Kind    string // mute_domain, mute_source, boost_tag, boost_domain, tag
	Pattern string
	Value   string
	Enabled bool
}

// AddRule creates a new rule.
func (s *Store) AddRule(kind, pattern, value string) error {
	_, err := s.db.Exec(`
		INSERT INTO rules (kind, pattern, value) VALUES (?, ?, ?)`,
		kind, pattern, value,
	)
	return err
}

// ListRules returns all enabled rules.
func (s *Store) ListRules() ([]Rule, error) {
	rows, err := s.db.Query(`
		SELECT id, kind, pattern, COALESCE(value,''), enabled
		FROM rules WHERE enabled = 1 ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var rules []Rule
	for rows.Next() {
		var r Rule
		var enabled int
		if rows.Scan(&r.ID, &r.Kind, &r.Pattern, &r.Value, &enabled) == nil {
			r.Enabled = enabled == 1
			rules = append(rules, r)
		}
	}
	return rules, rows.Err()
}

// DeleteRule removes a rule by ID.
func (s *Store) DeleteRule(id int) error {
	_, err := s.db.Exec("DELETE FROM rules WHERE id = ?", id)
	return err
}

// HasMuteRule checks if a domain or source is muted.
func (s *Store) HasMuteRule(domain string) bool {
	var count int
	s.db.QueryRow(`
		SELECT COUNT(*) FROM rules
		WHERE enabled = 1 AND kind IN ('mute_domain', 'mute_source') AND pattern = ?`,
		domain,
	).Scan(&count)
	return count > 0
}

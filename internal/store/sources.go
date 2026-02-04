package store

import (
	"encoding/json"
	"time"
)

// SourceRecord represents a source in the database.
type SourceRecord struct {
	ID         int
	Name       string
	Kind       string
	URL        string
	Icon       string
	Weight     float64
	Enabled    bool
	Settings   map[string]any
	AddedAt    time.Time
	LastSync   *time.Time
	SyncErrors int
}

// InsertSource adds a new source and returns its ID.
func (s *Store) InsertSource(name, kind, url, icon string, settings map[string]any) (int, error) {
	settingsJSON, _ := json.Marshal(settings)
	if icon == "" {
		icon = "📰"
	}

	result, err := s.db.Exec(`
		INSERT INTO sources (name, kind, url, icon, settings)
		VALUES (?, ?, ?, ?, ?)`,
		name, kind, url, icon, string(settingsJSON),
	)
	if err != nil {
		return 0, err
	}

	id, err := result.LastInsertId()
	return int(id), err
}

// GetOrCreateSource finds a source by name+kind, or creates it.
func (s *Store) GetOrCreateSource(name, kind, url, icon string) (int, error) {
	var id int
	err := s.db.QueryRow(
		"SELECT id FROM sources WHERE name = ? AND kind = ?", name, kind,
	).Scan(&id)
	if err == nil {
		return id, nil
	}

	return s.InsertSource(name, kind, url, icon, nil)
}

// ListSources returns all sources.
func (s *Store) ListSources() ([]SourceRecord, error) {
	rows, err := s.db.Query(`
		SELECT id, name, kind, COALESCE(url,''), icon, weight, enabled,
			COALESCE(settings,'{}'), added_at, last_sync, sync_errors
		FROM sources ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sources []SourceRecord
	for rows.Next() {
		var src SourceRecord
		var settingsJSON, addedAt string
		var lastSync *string
		var enabled int

		err := rows.Scan(
			&src.ID, &src.Name, &src.Kind, &src.URL, &src.Icon,
			&src.Weight, &enabled, &settingsJSON, &addedAt, &lastSync, &src.SyncErrors,
		)
		if err != nil {
			continue
		}

		src.Enabled = enabled == 1
		src.AddedAt, _ = time.Parse(time.RFC3339, addedAt)
		json.Unmarshal([]byte(settingsJSON), &src.Settings)
		if lastSync != nil {
			t, _ := time.Parse(time.RFC3339, *lastSync)
			src.LastSync = &t
		}

		sources = append(sources, src)
	}

	return sources, rows.Err()
}

// UpdateLastSync updates the last sync time for a source.
func (s *Store) UpdateLastSync(sourceID int) error {
	_, err := s.db.Exec(
		"UPDATE sources SET last_sync = datetime('now'), sync_errors = 0 WHERE id = ?",
		sourceID,
	)
	return err
}

// IncrSyncErrors increments the error count for a source.
func (s *Store) IncrSyncErrors(sourceID int) error {
	_, err := s.db.Exec(
		"UPDATE sources SET sync_errors = sync_errors + 1 WHERE id = ?",
		sourceID,
	)
	return err
}

// SetSourceEnabled enables or disables a source.
func (s *Store) SetSourceEnabled(sourceID int, enabled bool) error {
	e := 0
	if enabled {
		e = 1
	}
	_, err := s.db.Exec("UPDATE sources SET enabled = ? WHERE id = ?", e, sourceID)
	return err
}

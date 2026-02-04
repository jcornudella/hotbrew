package store

import (
	"encoding/json"
	"time"

	"github.com/jcornudella/hotbrew/pkg/trss"
)

// SaveDigest stores a generated digest.
func (s *Store) SaveDigest(d *trss.Digest) error {
	data, err := json.Marshal(d)
	if err != nil {
		return err
	}

	_, err = s.db.Exec(`
		INSERT INTO digests (title, window, item_count, data)
		VALUES (?, ?, ?, ?)`,
		d.Title, d.Window, d.ItemCount, string(data),
	)
	return err
}

// GetLatestDigest returns the most recent digest.
func (s *Store) GetLatestDigest() (*trss.Digest, error) {
	var data string
	err := s.db.QueryRow(`
		SELECT data FROM digests ORDER BY generated_at DESC LIMIT 1`,
	).Scan(&data)
	if err != nil {
		return nil, err
	}

	var d trss.Digest
	if err := json.Unmarshal([]byte(data), &d); err != nil {
		return nil, err
	}
	return &d, nil
}

// GetDigestsSince returns digests generated after the given time.
func (s *Store) GetDigestsSince(since time.Time) ([]*trss.Digest, error) {
	rows, err := s.db.Query(`
		SELECT data FROM digests WHERE generated_at >= ? ORDER BY generated_at DESC`,
		since.UTC().Format(time.RFC3339),
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var digests []*trss.Digest
	for rows.Next() {
		var data string
		if rows.Scan(&data) != nil {
			continue
		}
		var d trss.Digest
		if json.Unmarshal([]byte(data), &d) == nil {
			digests = append(digests, &d)
		}
	}
	return digests, rows.Err()
}

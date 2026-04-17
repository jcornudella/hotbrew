package xbookmarks

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
)

// cursor persists across runs so we only pull bookmarks newer than
// NewestID. UserID is cached to avoid hitting /users/me every sync.
type cursor struct {
	UserID   string `json:"user_id,omitempty"`
	NewestID string `json:"newest_id,omitempty"`
}

func cursorPath() string {
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		return filepath.Join(xdg, "hotbrew", "x_cursor.json")
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "hotbrew", "x_cursor.json")
}

func loadCursor() (*cursor, error) {
	data, err := os.ReadFile(cursorPath())
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return &cursor{}, nil
		}
		return nil, err
	}
	var c cursor
	if err := json.Unmarshal(data, &c); err != nil {
		return nil, err
	}
	return &c, nil
}

func saveCursor(c *cursor) error {
	path := cursorPath()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o600)
}

package store

import "fmt"

const currentVersion = 6

var migrations = []string{
	// Version 1: initial schema
	`
	CREATE TABLE IF NOT EXISTS sources (
		id          INTEGER PRIMARY KEY AUTOINCREMENT,
		name        TEXT NOT NULL,
		kind        TEXT NOT NULL,
		url         TEXT,
		icon        TEXT DEFAULT '📰',
		weight      REAL DEFAULT 1.0,
		enabled     INTEGER DEFAULT 1,
		settings    TEXT,
		added_at    TEXT DEFAULT (datetime('now')),
		last_sync   TEXT,
		sync_errors INTEGER DEFAULT 0
	);

	CREATE TABLE IF NOT EXISTS items (
		id              TEXT PRIMARY KEY,
		fingerprint     TEXT NOT NULL,
		title           TEXT NOT NULL,
		url             TEXT,
		url_canonical   TEXT,
		source_id       INTEGER NOT NULL REFERENCES sources(id),
		source_name     TEXT NOT NULL,
		published_at    TEXT,
		fetched_at      TEXT NOT NULL DEFAULT (datetime('now')),
		summary         TEXT,
		body            TEXT,
		tags            TEXT,
		score_raw       REAL DEFAULT 0,
		score_computed  REAL DEFAULT 0,
		engagement      TEXT,
		meta            TEXT,
		UNIQUE(fingerprint, source_id)
	);

	CREATE INDEX IF NOT EXISTS idx_items_fingerprint ON items(fingerprint);
	CREATE INDEX IF NOT EXISTS idx_items_published ON items(published_at);
	CREATE INDEX IF NOT EXISTS idx_items_source ON items(source_id);
	CREATE INDEX IF NOT EXISTS idx_items_score ON items(score_computed);

	CREATE TABLE IF NOT EXISTS dedup_edges (
		item_id_a   TEXT NOT NULL,
		item_id_b   TEXT NOT NULL,
		confidence  REAL DEFAULT 1.0,
		created_at  TEXT DEFAULT (datetime('now')),
		PRIMARY KEY (item_id_a, item_id_b)
	);

	CREATE TABLE IF NOT EXISTS item_state (
		item_id     TEXT PRIMARY KEY,
		state       TEXT NOT NULL DEFAULT 'unread',
		opened_at   TEXT,
		saved_at    TEXT,
		updated_at  TEXT DEFAULT (datetime('now'))
	);

	CREATE TABLE IF NOT EXISTS rules (
		id          INTEGER PRIMARY KEY AUTOINCREMENT,
		kind        TEXT NOT NULL,
		pattern     TEXT NOT NULL,
		value       TEXT,
		enabled     INTEGER DEFAULT 1,
		created_at  TEXT DEFAULT (datetime('now'))
	);

	CREATE INDEX IF NOT EXISTS idx_rules_kind ON rules(kind);

	CREATE TABLE IF NOT EXISTS digests (
		id           INTEGER PRIMARY KEY AUTOINCREMENT,
		title        TEXT NOT NULL,
		window       TEXT NOT NULL,
		generated_at TEXT NOT NULL DEFAULT (datetime('now')),
		item_count   INTEGER,
		data         TEXT
	);
	`,
	// Version 2: feedback table for issue ratings
	`
	CREATE TABLE IF NOT EXISTS feedback (
		id         INTEGER PRIMARY KEY AUTOINCREMENT,
		rating     INTEGER NOT NULL,
		note       TEXT,
		created_at TEXT NOT NULL DEFAULT (datetime('now'))
	);
	`,
	// Version 3: derived ranking features persisted per item
	`
	CREATE TABLE IF NOT EXISTS item_features (
		item_id        TEXT PRIMARY KEY REFERENCES items(id) ON DELETE CASCADE,
		freshness      REAL NOT NULL DEFAULT 0,
		authority      REAL NOT NULL DEFAULT 0,
		engagement     REAL NOT NULL DEFAULT 0,
		resonance      REAL NOT NULL DEFAULT 0,
		novelty        REAL NOT NULL DEFAULT 0,
		topic_match    REAL NOT NULL DEFAULT 0,
		repeat_penalty REAL NOT NULL DEFAULT 0,
		content_fit    REAL NOT NULL DEFAULT 0,
		computed_at    TEXT NOT NULL DEFAULT (datetime('now'))
	);
	`,
	// Version 4: deterministic theme clusters + cluster membership
	`
	CREATE TABLE IF NOT EXISTS clusters (
		id                     TEXT PRIMARY KEY,
		label                  TEXT NOT NULL,
		slug                   TEXT NOT NULL,
		representative_item_id TEXT,
		score                  REAL NOT NULL DEFAULT 0,
		why_it_matters         TEXT,
		generated_at           TEXT NOT NULL DEFAULT (datetime('now'))
	);

	CREATE INDEX IF NOT EXISTS idx_clusters_slug ON clusters(slug);
	CREATE INDEX IF NOT EXISTS idx_clusters_generated_at ON clusters(generated_at);

	CREATE TABLE IF NOT EXISTS cluster_items (
		cluster_id TEXT NOT NULL REFERENCES clusters(id) ON DELETE CASCADE,
		item_id    TEXT NOT NULL REFERENCES items(id) ON DELETE CASCADE,
		PRIMARY KEY (cluster_id, item_id)
	);

	CREATE INDEX IF NOT EXISTS idx_cluster_items_item ON cluster_items(item_id);
	`,
	// Version 5: event-sourced feedback log — one row per user
	// interaction (open, save, read, mute, explain_viewed). Ranking
	// and preference-learning later read this log to bias future
	// briefings without touching the per-item state rows.
	`
	CREATE TABLE IF NOT EXISTS feedback_events (
		id         INTEGER PRIMARY KEY AUTOINCREMENT,
		action     TEXT NOT NULL,
		item_id    TEXT,
		target     TEXT,
		created_at TEXT NOT NULL DEFAULT (datetime('now'))
	);

	CREATE INDEX IF NOT EXISTS idx_feedback_events_action ON feedback_events(action);
	CREATE INDEX IF NOT EXISTS idx_feedback_events_item ON feedback_events(item_id);
	CREATE INDEX IF NOT EXISTS idx_feedback_events_created_at ON feedback_events(created_at);
	`,
	// Version 6: theme preferences — first explicit personalization
	// knob. One row per theme slug with a 'follow' or 'mute' state;
	// ranking consults the table via store.ListThemePreferences() to
	// adjust TopicMatch in downstream briefings.
	`
	CREATE TABLE IF NOT EXISTS theme_preferences (
		slug       TEXT PRIMARY KEY,
		state      TEXT NOT NULL CHECK(state IN ('follow','mute')),
		updated_at TEXT NOT NULL DEFAULT (datetime('now'))
	);
	`,
}

func (s *Store) migrate() error {
	// Create version table if needed
	if _, err := s.db.Exec(`CREATE TABLE IF NOT EXISTS schema_version (version INTEGER NOT NULL)`); err != nil {
		return fmt.Errorf("create schema_version: %w", err)
	}

	// Get current version
	var version int
	row := s.db.QueryRow("SELECT COALESCE(MAX(version), 0) FROM schema_version")
	if err := row.Scan(&version); err != nil {
		return fmt.Errorf("read schema version: %w", err)
	}

	// Run pending migrations
	for i := version; i < len(migrations); i++ {
		if _, err := s.db.Exec(migrations[i]); err != nil {
			return fmt.Errorf("migration %d: %w", i+1, err)
		}
	}

	// Update version
	if version < currentVersion {
		if _, err := s.db.Exec("DELETE FROM schema_version"); err != nil {
			return err
		}
		if _, err := s.db.Exec("INSERT INTO schema_version (version) VALUES (?)", currentVersion); err != nil {
			return err
		}
	}

	return nil
}

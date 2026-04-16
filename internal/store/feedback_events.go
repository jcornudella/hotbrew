package store

import "time"

// Feedback-event action tags. Commands and the TUI import these so
// the string values stay consistent across call sites — ranking will
// later filter by action, so typos here would silently break signal.
const (
	FeedbackActionOpen          = "open"
	FeedbackActionSave          = "save"
	FeedbackActionRead          = "read"
	FeedbackActionUnread        = "unread"
	FeedbackActionMuteDomain    = "mute_domain"
	FeedbackActionBoost         = "boost"
	FeedbackActionExplainViewed = "explain_viewed"
)

// FeedbackEvent is one user interaction captured as learning input.
// Target is optional context — for mute it's the domain, for explain
// it records whether the "why it matters" or factor view was shown.
type FeedbackEvent struct {
	ID        int64
	Action    string
	ItemID    string
	Target    string
	CreatedAt time.Time
}

// RecordFeedbackEvent appends one interaction to the event log.
// Nothing else in the system reads-then-writes this table, so we
// take the insert directly rather than routing through a queue.
func (s *Store) RecordFeedbackEvent(action, itemID, target string) error {
	_, err := s.db.Exec(
		`INSERT INTO feedback_events (action, item_id, target) VALUES (?, ?, ?)`,
		action, nullIfEmpty(itemID), nullIfEmpty(target),
	)
	return err
}

// ListFeedbackEvents returns the most recent events first, capped
// at limit (0 = unbounded). Intended for introspection and future
// ranking backfill — not a hot path, so no pagination yet.
func (s *Store) ListFeedbackEvents(limit int) ([]FeedbackEvent, error) {
	query := `SELECT id, action, COALESCE(item_id, ''), COALESCE(target, ''), created_at
		FROM feedback_events ORDER BY id DESC`
	args := []any{}
	if limit > 0 {
		query += " LIMIT ?"
		args = append(args, limit)
	}
	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	return scanFeedbackEventRows(rows)
}

// ListFeedbackEventsByAction filters the log by action tag. Useful
// for "give me every save in the last 30 days" style queries.
func (s *Store) ListFeedbackEventsByAction(action string, limit int) ([]FeedbackEvent, error) {
	query := `SELECT id, action, COALESCE(item_id, ''), COALESCE(target, ''), created_at
		FROM feedback_events WHERE action = ? ORDER BY id DESC`
	args := []any{action}
	if limit > 0 {
		query += " LIMIT ?"
		args = append(args, limit)
	}
	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	return scanFeedbackEventRows(rows)
}

func scanFeedbackEventRows(rows interface {
	Next() bool
	Scan(dest ...any) error
	Err() error
}) ([]FeedbackEvent, error) {
	var events []FeedbackEvent
	for rows.Next() {
		var ev FeedbackEvent
		var createdAt string
		if err := rows.Scan(&ev.ID, &ev.Action, &ev.ItemID, &ev.Target, &createdAt); err != nil {
			return nil, err
		}
		ev.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
		if ev.CreatedAt.IsZero() {
			ev.CreatedAt, _ = time.Parse("2006-01-02 15:04:05", createdAt)
		}
		events = append(events, ev)
	}
	return events, rows.Err()
}

func nullIfEmpty(s string) any {
	if s == "" {
		return nil
	}
	return s
}

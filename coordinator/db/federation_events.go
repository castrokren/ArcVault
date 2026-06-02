package db

import (
	"fmt"
	"time"
)

// FederationEvent represents an append-only log entry for federation state changes.
type FederationEvent struct {
	ID          int64     `json:"id"`
	Seq         int64     `json:"seq"`
	Coordinator string    `json:"coordinator"`
	EventType   string    `json:"event_type"`
	Payload     string    `json:"payload"`
	CreatedAt   time.Time `json:"created_at"`
}

// AppendFederationEvent appends a new event to the federation_events log.
// Returns the assigned sequence number and any error.
func (d *DB) AppendFederationEvent(coordinatorID, eventType, payload string) (seq int64, err error) {
	// Get the next sequence number for this coordinator (max seq + 1).
	// If no prior events, start at 1.
	var maxSeq int64
	err = d.conn.QueryRow(
		`SELECT COALESCE(MAX(seq), 0) FROM federation_events WHERE coordinator = ?`,
		coordinatorID,
	).Scan(&maxSeq)
	if err != nil {
		return 0, fmt.Errorf("failed to get max seq: %w", err)
	}
	seq = maxSeq + 1

	// Insert the event.
	result, err := d.conn.Exec(
		`INSERT INTO federation_events (seq, coordinator, event_type, payload, created_at)
		 VALUES (?, ?, ?, ?, CURRENT_TIMESTAMP)`,
		seq, coordinatorID, eventType, payload,
	)
	if err != nil {
		return 0, fmt.Errorf("failed to insert event: %w", err)
	}

	// Verify insertion (optional, but good for sanity).
	id, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("failed to get last insert id: %w", err)
	}

	_ = id // Suppress unused warning if needed.
	return seq, nil
}

// GetFederationEventsSince returns all events for a coordinator since the given sequence number.
// Events are ordered by sequence number.
func (d *DB) GetFederationEventsSince(coordinatorID string, sinceSeq int64) ([]FederationEvent, error) {
	rows, err := d.conn.Query(
		`SELECT id, seq, coordinator, event_type, payload, created_at
		 FROM federation_events
		 WHERE coordinator = ? AND seq > ?
		 ORDER BY seq ASC`,
		coordinatorID, sinceSeq,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to query events: %w", err)
	}
	defer rows.Close()

	var events []FederationEvent
	for rows.Next() {
		var e FederationEvent
		if err := rows.Scan(&e.ID, &e.Seq, &e.Coordinator, &e.EventType, &e.Payload, &e.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan event: %w", err)
		}
		events = append(events, e)
	}

	if events == nil {
		events = []FederationEvent{}
	}

	return events, rows.Err()
}

// PruneFederationEvents deletes events older than the given number of days.
// Returns the number of rows deleted and any error.
func (d *DB) PruneFederationEvents(olderThanDays int) (rowsDeleted int64, err error) {
	result, err := d.conn.Exec(
		`DELETE FROM federation_events
		 WHERE created_at < datetime('now', '-' || ? || ' days')`,
		olderThanDays,
	)
	if err != nil {
		return 0, fmt.Errorf("failed to prune events: %w", err)
	}

	rowsDeleted, err = result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to get rows affected: %w", err)
	}

	return rowsDeleted, nil
}

// GetMaxEventSeq returns the highest sequence number for a given coordinator.
// Returns 0 if no events exist.
func (d *DB) GetMaxEventSeq(coordinatorID string) (int64, error) {
	var maxSeq int64
	err := d.conn.QueryRow(
		`SELECT COALESCE(MAX(seq), 0) FROM federation_events WHERE coordinator = ?`,
		coordinatorID,
	).Scan(&maxSeq)
	if err != nil {
		return 0, fmt.Errorf("failed to get max seq: %w", err)
	}
	return maxSeq, nil
}

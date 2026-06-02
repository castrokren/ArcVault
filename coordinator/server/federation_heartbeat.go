package server

import (
	"fmt"
	"log"
	"time"
)

// StartHeartbeatDetector runs a background loop that marks federation coordinators
// offline when their last_seen timestamp exceeds the 30s threshold.
// Called as a goroutine from Start(). Runs until the process exits.
func (s *Server) StartHeartbeatDetector() {
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()
	for range ticker.C {
		s.checkHeartbeatTimeouts(30 * time.Second)
	}
}

// checkHeartbeatTimeouts is the inner logic — exported for direct use in tests
// without waiting for the ticker.
func (s *Server) checkHeartbeatTimeouts(threshold time.Duration) {
	cutoff := time.Now().UTC().Add(-threshold)

	rows, err := s.db.Conn().Query(`
		SELECT id FROM federation
		WHERE last_seen < ?
		  AND status != 'offline'
		  AND last_seen IS NOT NULL
	`, cutoff)
	if err != nil {
		log.Printf("[federation] heartbeat detector query error: %v", err)
		return
	}

	var timedOut []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err == nil {
			timedOut = append(timedOut, id)
		}
	}
	rows.Close()

	for _, coordID := range timedOut {
		if _, err := s.db.Conn().Exec(
			`UPDATE federation SET status = 'offline' WHERE id = ?`, coordID,
		); err != nil {
			log.Printf("[federation] failed to mark %s offline: %v", coordID, err)
			continue
		}

		payload := fmt.Sprintf(`{"coordinator_id":%q,"reason":"heartbeat_timeout"}`, coordID)
		if _, err := s.db.AppendFederationEvent(s.coordinatorID, "coordinator_offline", payload); err != nil {
			log.Printf("[federation] failed to append offline event for %s: %v", coordID, err)
		}

		log.Printf("[federation] coordinator %s marked offline (heartbeat timeout)", coordID)
	}
}

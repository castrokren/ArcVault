package server

import (
	"log"
	"time"
)

// detectOfflineAgents marks any agent whose last_seen is older than
// the given threshold as offline, and broadcasts an agent.updated event
// for each one. Safe to call directly in tests.
func (s *Server) detectOfflineAgents(threshold time.Duration) {
	cutoff := time.Now().UTC().Add(-threshold).Format(time.RFC3339)

	rows, err := s.db.Conn().Query(
		`SELECT id FROM agents WHERE status = 'online' AND last_seen < ?`, cutoff,
	)
	if err != nil {
		log.Printf("OfflineDetector: query failed: %v", err)
		return
	}
	defer rows.Close()

	var staleIDs []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err == nil {
			staleIDs = append(staleIDs, id)
		}
	}

	for _, id := range staleIDs {
		_, err := s.db.Conn().Exec(
			`UPDATE agents SET status = 'offline' WHERE id = ?`, id,
		)
		if err != nil {
			log.Printf("OfflineDetector: failed to mark %s offline: %v", id, err)
			continue
		}
		log.Printf("OfflineDetector: marked %s offline", id)
		s.hub.Broadcast(Event{
			Type:    "agent.updated",
			Payload: map[string]string{"id": id, "status": "offline"},
		})
	}
}

// StartOfflineDetector runs detectOfflineAgents on a ticker in the background.
// Call this from coordinator Start().
func (s *Server) StartOfflineDetector(interval, threshold time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for range ticker.C {
			s.detectOfflineAgents(threshold)
		}
	}()
}

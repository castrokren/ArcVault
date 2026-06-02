package server

import (
	"encoding/json"
	"net/http"
	"time"
)

// CoordinatorHealth represents the health status of a federation peer
type CoordinatorHealth struct {
	ID              string     `json:"id"`
	Name            string     `json:"name"`
	Status          string     `json:"status"`           // online, offline, reconnecting
	LastSeen        *time.Time `json:"last_seen"`
	LagEvents       int        `json:"lag_events"`       // Number of unsynced events
	AgentCount      int        `json:"agent_count"`      // Number of agents homed to this coordinator
	LastSeq         int64      `json:"last_seq"`         // Last sequence number acknowledged by this coordinator
	MaxSeq          int64      `json:"max_seq"`          // Current max sequence number
	Version         *string    `json:"version,omitempty"`
}

// handleFederationHealth handles GET /api/federation/health
// Returns health status of all federation peers.
func (s *Server) handleFederationHealth(w http.ResponseWriter, r *http.Request) {
	// Get all federation peers
	peers, err := s.db.ListFederation()
	if err != nil {
		http.Error(w, "failed to fetch federation peers", http.StatusInternalServerError)
		return
	}

	healthList := []CoordinatorHealth{}
	now := time.Now()

	for _, peer := range peers {
		status := "offline"
		if peer.LastSeen != nil && now.Sub(*peer.LastSeen) < 30*time.Second {
			status = "online"
		}

		// Calculate lag: maxSeq - lastSeq
		maxSeq, err := s.db.GetMaxEventSeq(peer.ID)
		if err != nil {
			maxSeq = 0
		}
		lag := maxSeq - peer.LastSeq
		if lag < 0 {
			lag = 0
		}

		// Count agents homed to this coordinator
		var agentCount int
		s.db.Conn().QueryRow(
			`SELECT COUNT(*) FROM agents WHERE home_coordinator = ? AND status = 'online'`,
			peer.ID,
		).Scan(&agentCount)

		health := CoordinatorHealth{
			ID:         peer.ID,
			Name:       peer.Name,
			Status:     status,
			LastSeen:   peer.LastSeen,
			LagEvents:  int(lag),
			AgentCount: agentCount,
			LastSeq:    peer.LastSeq,
			MaxSeq:     maxSeq,
			Version:    &peer.Version,
		}

		healthList = append(healthList, health)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(healthList)
}

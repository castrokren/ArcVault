package server

import (
	"encoding/json"
	"net/http"
	"strconv"
)

// FederationSyncResponse represents the response from GET /api/federation/sync
type FederationSyncResponse struct {
	Events    []SyncEvent `json:"events"`
	LatestSeq int64       `json:"latest_seq"`
}

// SyncEvent represents a single event in the federation_events log
type SyncEvent struct {
	Seq       int64  `json:"seq"`
	EventType string `json:"event_type"`
	Payload   string `json:"payload"`
}

// SyncAckRequest represents the request body for POST /api/federation/sync/ack
type SyncAckRequest struct {
	Seq int64 `json:"seq"`
}

// handleFederationSync handles GET /api/federation/sync?since=<seq>&coordinator=<id>
// Admin only. Root coordinator only. Returns all events since the given sequence number.
func (s *Server) handleFederationSync(w http.ResponseWriter, r *http.Request) {
	// Parse query parameters
	sinceStr := r.URL.Query().Get("since")
	coordinatorID := r.URL.Query().Get("coordinator")

	if sinceStr == "" || coordinatorID == "" {
		http.Error(w, "missing required query parameters: since, coordinator", http.StatusBadRequest)
		return
	}

	since, err := strconv.ParseInt(sinceStr, 10, 64)
	if err != nil {
		http.Error(w, "invalid since parameter", http.StatusBadRequest)
		return
	}

	// Get events since the given sequence
	events, err := s.db.GetFederationEventsSince(coordinatorID, since)
	if err != nil {
		http.Error(w, "failed to fetch events", http.StatusInternalServerError)
		return
	}

	// Get the latest sequence number for this coordinator
	latestSeq, err := s.db.GetMaxEventSeq(coordinatorID)
	if err != nil {
		http.Error(w, "failed to get latest sequence", http.StatusInternalServerError)
		return
	}

	// Convert to SyncEvent format
	syncEvents := []SyncEvent{}
	for _, e := range events {
		syncEvents = append(syncEvents, SyncEvent{
			Seq:       e.Seq,
			EventType: e.EventType,
			Payload:   e.Payload,
		})
	}

	response := FederationSyncResponse{
		Events:    syncEvents,
		LatestSeq: latestSeq,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleFederationSyncAck handles POST /api/federation/sync/ack
// Spoke coordinator acknowledges that it has applied events up to the given sequence.
// Updates the federation table's last_seq field.
func (s *Server) handleFederationSyncAck(w http.ResponseWriter, r *http.Request) {
	var req SyncAckRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if req.Seq < 0 {
		http.Error(w, "seq must be non-negative", http.StatusBadRequest)
		return
	}

	// Get the coordinator ID from the request context (should be from JWT token or URL)
	// For now, we'll update the federation entry that initiated the sync
	// In a full implementation, this would come from the JWT subject claim
	coordinatorID := r.URL.Query().Get("coordinator")
	if coordinatorID == "" {
		http.Error(w, "missing coordinator parameter", http.StatusBadRequest)
		return
	}

	// Update the federation table's last_seq for this coordinator
	_, err := s.db.Conn().Exec(
		`UPDATE federation SET last_seq = ? WHERE id = ?`,
		req.Seq, coordinatorID,
	)
	if err != nil {
		http.Error(w, "failed to update coordinator last_seq", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "ack",
		"seq":    req.Seq,
	})
}

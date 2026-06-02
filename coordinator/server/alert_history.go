package server

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"

	"arcvault/coordinator/db"
)

// handleListAlertHistory lists recent alert history (viewer+)
func (s *Server) handleListAlertHistory(w http.ResponseWriter, r *http.Request) {
	const limit = 100
	history, err := s.db.ListAlertHistory(limit)
	if err != nil {
		log.Printf("[alert_history] list failed: %v", err)
		http.Error(w, "failed to list history", http.StatusInternalServerError)
		return
	}

	if history == nil {
		history = []db.AlertHistory{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(history)
}

// handleRetryAlert manually retries sending an alert (admin)
func (s *Server) handleRetryAlert(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	// For now, a simple implementation that acknowledges the retry request
	// In a full implementation, you would:
	// 1. Look up the alert history row
	// 2. Get the associated rule and job details
	// 3. Re-fire the alert using RetryDispatch
	// 4. Update the history status

	log.Printf("[alert_history] retry requested for alert %d", id)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"id":     id,
		"status": "retry_requested",
	})
}

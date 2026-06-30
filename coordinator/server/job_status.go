package server

import (
	"encoding/json"
	"log"
	"net/http"
)

var validStatuses = map[string]bool{
	"pending":   true,
	"running":   true,
	"completed": true,
	"failed":    true,
	"canceling": true,
	"cancelled": true,
}

// handleUpdateJobStatus handles PATCH /api/jobs/{id}/status
func (s *Server) handleUpdateJobStatus(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	var input struct {
		Status string `json:"status"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}
	if !validStatuses[input.Status] {
		http.Error(w, "invalid status: must be pending, running, completed, failed, canceling, or cancelled", http.StatusBadRequest)
		return
	}

	exists, err := s.db.JobExists(id)
	if err != nil {
		http.Error(w, "failed to query job", http.StatusInternalServerError)
		return
	}
	if !exists {
		http.Error(w, "job not found", http.StatusNotFound)
		return
	}

	if err := s.jobService.UpdateJobStatus(id, input.Status); err != nil {
		log.Printf("ERROR handleUpdateJobStatus: job=%s status=%s err=%v", id, input.Status, err)
		http.Error(w, "failed to update job", http.StatusInternalServerError)
		return
	}

	job, err := s.jobService.GetJob(id)
	if err != nil {
		log.Printf("ERROR handleUpdateJobStatus: job=%s GetJob err=%v", id, err)
		http.Error(w, "failed to fetch updated job", http.StatusInternalServerError)
		return
	}

	// broadcast to WebSocket clients
	s.hub.Broadcast(Event{
		Type:    "job.updated",
		Payload: map[string]string{"id": id, "status": input.Status},
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(job)
}

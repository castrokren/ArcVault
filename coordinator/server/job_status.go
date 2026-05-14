package server

import (
	"database/sql"
	"encoding/json"
	"net/http"
)

var validStatuses = map[string]bool{
	"pending":   true,
	"running":   true,
	"completed": true,
	"failed":    true,
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
		http.Error(w, "invalid status: must be pending, running, completed, or failed", http.StatusBadRequest)
		return
	}

	result, err := s.db.Conn().Exec(`UPDATE jobs SET status = ? WHERE id = ?`, input.Status, id)
	if err != nil {
		http.Error(w, "failed to update job", http.StatusInternalServerError)
		return
	}
	n, _ := result.RowsAffected()
	if n == 0 {
		http.Error(w, "job not found", http.StatusNotFound)
		return
	}

	// return updated job
	var j Job
	var schedule sql.NullString
	err = s.db.Conn().QueryRow(
		`SELECT id, agent_id, name, source_path, dest_path, schedule, status, created_at FROM jobs WHERE id = ?`, id,
	).Scan(&j.ID, &j.AgentID, &j.Name, &j.SourcePath, &j.DestPath, &schedule, &j.Status, &j.CreatedAt)
	if err != nil {
		http.Error(w, "failed to fetch updated job", http.StatusInternalServerError)
		return
	}
	if schedule.Valid {
		j.Schedule = &schedule.String
	}

	// broadcast to WebSocket clients
	s.hub.Broadcast(Event{
		Type:    "job.updated",
		Payload: map[string]string{"id": j.ID, "status": j.Status},
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(j)
}

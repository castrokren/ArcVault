package server

import (
	"encoding/json"
	"net/http"
	"time"
)

// ProgressRequest is the request body for POST /api/jobs/{id}/progress
type ProgressRequest struct {
	Percentage int      `json:"percentage"`
	Logs       []string `json:"logs"`
	Status     string   `json:"status"`
}

// ProgressResponse is the response body for progress endpoints
type ProgressResponse struct {
	Success bool   `json:"success"`
	Error   string `json:"error,omitempty"`
}

// handleProgress handles POST /api/jobs/{id}/progress
// Stores job progress and log lines from the agent
func (s *Server) handleProgress(w http.ResponseWriter, r *http.Request) {
	jobID := r.PathValue("id")
	if jobID == "" {
		http.Error(w, "missing job ID", http.StatusBadRequest)
		return
	}

	// Parse request body
	var req ProgressRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	// Validate percentage
	if req.Percentage < 0 || req.Percentage > 100 {
		http.Error(w, "percentage must be 0-100", http.StatusBadRequest)
		return
	}

	// Validate status
	validStatuses := map[string]bool{
		"running":   true,
		"completed": true,
		"cancelled": true,
		"error":     true,
	}
	if !validStatuses[req.Status] {
		http.Error(w, "invalid status", http.StatusBadRequest)
		return
	}

	// Check if job exists
	exists, err := s.db.JobExists(jobID)
	if err != nil || !exists {
		http.Error(w, "job not found", http.StatusNotFound)
		return
	}

	// Store progress and logs
	if err := s.db.UpdateProgressAndLogs(jobID, req.Percentage, req.Logs, req.Status); err != nil {
		http.Error(w, "failed to store progress", http.StatusInternalServerError)
		return
	}

	// Broadcast progress update to all connected dashboards
	s.hub.Broadcast(Event{
		Type: "progress",
		Payload: map[string]interface{}{
			"job_id":     jobID,
			"percentage": req.Percentage,
			"status":     req.Status,
			"timestamp":  time.Now(),
		},
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(ProgressResponse{Success: true})
}

// handleGetProgress handles GET /api/jobs/{id}/progress
// Returns current progress percentage, logs, and stalled status
func (s *Server) handleGetProgress(w http.ResponseWriter, r *http.Request) {
	jobID := r.PathValue("id")
	if jobID == "" {
		http.Error(w, "missing job ID", http.StatusBadRequest)
		return
	}

	// Fetch progress data
	progress, err := s.db.GetProgress(jobID)
	if err != nil {
		http.Error(w, "job not found", http.StatusNotFound)
		return
	}

	// Compute stalled (no update for 5+ minutes)
	stalled := false
	if progress.LastProgressAt != nil {
		stalled = time.Since(*progress.LastProgressAt) > 5*time.Minute
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"job_id":           jobID,
		"percentage":       progress.Percentage,
		"status":           progress.Status,
		"last_progress_at": progress.LastProgressAt,
		"logs":             progress.Logs,
		"log_count":        progress.LogCount,
		"stalled":          stalled,
	})
}

// handleGetJobLogs handles GET /api/jobs/{id}/logs
// Returns paginated logs with optional ?page= and ?limit= query params
func (s *Server) handleGetJobLogs(w http.ResponseWriter, r *http.Request) {
	jobID := r.PathValue("id")
	if jobID == "" {
		http.Error(w, "missing job ID", http.StatusBadRequest)
		return
	}

	// Parse pagination parameters
	p := ParsePagination(r)

	// Fetch paginated logs
	page, err := s.db.GetJobLogsWithPagination(jobID, p.Page, p.Limit)
	if err != nil {
		http.Error(w, "failed to retrieve logs", http.StatusInternalServerError)
		return
	}

	// Return paginated response
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(NewPaginatedResponse(page.Logs, page.Total, p.Page, p.Limit))
}

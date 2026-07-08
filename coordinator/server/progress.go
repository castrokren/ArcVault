package server

import (
	"encoding/json"
	"net/http"
	"time"
)

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

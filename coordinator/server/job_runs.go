package server

import (
	"database/sql"
	"encoding/json"
	"net/http"
)

// handleGetJobRuns handles GET /api/jobs/{id}/runs
func (s *Server) handleGetJobRuns(w http.ResponseWriter, r *http.Request) {
	jobID := r.PathValue("id")

	// verify job exists
	var exists string
	err := s.db.Conn().QueryRow(`SELECT id FROM jobs WHERE id = ?`, jobID).Scan(&exists)
	if err == sql.ErrNoRows {
		http.Error(w, "job not found", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, "failed to query job", http.StatusInternalServerError)
		return
	}

	rows, err := s.db.Conn().Query(
		`SELECT id, job_id, exit_code, output, finished_at
		 FROM job_runs WHERE job_id = ? ORDER BY finished_at DESC`,
		jobID,
	)
	if err != nil {
		http.Error(w, "failed to query runs", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	runs := []JobRun{}
	for rows.Next() {
		var run JobRun
		var output sql.NullString
		var finishedAt sql.NullString
		if err := rows.Scan(&run.ID, &run.JobID, &run.ExitCode, &output, &finishedAt); err != nil {
			http.Error(w, "failed to scan run", http.StatusInternalServerError)
			return
		}
		if output.Valid {
			run.Output = output.String
		}
		if finishedAt.Valid {
			run.FinishedAt = finishedAt.String
		}
		runs = append(runs, run)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(runs)
}

package server

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"time"
)

// JobRun represents a single execution result for a job.
type JobRun struct {
	ID         string `json:"id"`
	JobID      string `json:"job_id"`
	ExitCode   int    `json:"exit_code"`
	Output     string `json:"output"`
	FinishedAt string `json:"finished_at"`
}

func newRunID() string {
	b := make([]byte, 8)
	rand.Read(b)
	return "run-" + hex.EncodeToString(b)
}

// handlePostJobResults handles POST /api/jobs/{id}/results
func (s *Server) handlePostJobResults(w http.ResponseWriter, r *http.Request) {
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

	var input struct {
		ExitCode int    `json:"exit_code"`
		Output   string `json:"output"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}

	run := JobRun{
		ID:         newRunID(),
		JobID:      jobID,
		ExitCode:   input.ExitCode,
		Output:     input.Output,
		FinishedAt: time.Now().UTC().Format(time.RFC3339),
	}

	_, err = s.db.Conn().Exec(
		`INSERT INTO job_runs (id, job_id, exit_code, output, finished_at)
		 VALUES (?, ?, ?, ?, ?)`,
		run.ID, run.JobID, run.ExitCode, run.Output, run.FinishedAt,
	)
	if err != nil {
		http.Error(w, "failed to store result", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(run)
}

package server

import (
	"database/sql"
	"encoding/json"
	"net/http"
)

// handleGetJobRuns handles GET /api/jobs/{id}/runs with optional pagination.
func (s *Server) handleGetJobRuns(w http.ResponseWriter, r *http.Request) {
	jobID := r.PathValue("id")

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

	p := ParsePagination(r)
	offset := (p.Page - 1) * p.Limit

	var total int
	if err := s.db.Conn().QueryRow(`SELECT COUNT(*) FROM job_runs WHERE job_id = ?`, jobID).Scan(&total); err != nil {
		http.Error(w, "failed to count runs", http.StatusInternalServerError)
		return
	}

	rows, err := s.db.Conn().Query(
		`SELECT id, job_id, exit_code, output, finished_at
		 FROM job_runs WHERE job_id = ? ORDER BY finished_at DESC LIMIT ? OFFSET ?`,
		jobID, p.Limit, offset,
	)
	if err != nil {
		http.Error(w, "failed to query runs", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	runs := []JobRun{}
	for rows.Next() {
		var run JobRun
		var exitCode sql.NullInt64
		var output sql.NullString
		var finishedAt sql.NullString
		if err := rows.Scan(&run.ID, &run.JobID, &exitCode, &output, &finishedAt); err != nil {
			http.Error(w, "failed to scan run", http.StatusInternalServerError)
			return
		}
		if exitCode.Valid {
			run.ExitCode = int(exitCode.Int64)
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
	json.NewEncoder(w).Encode(NewPaginatedResponse(runs, total, p.Page, p.Limit))
}

// handleListAllJobRuns handles GET /api/job-runs
// Optional: ?job_id=, ?agent_id=, ?page=, ?limit=
func (s *Server) handleListAllJobRuns(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	jobID := q.Get("job_id")
	agentID := q.Get("agent_id")
	p := ParsePagination(r)

	args := []any{}
	join := ""
	where := " WHERE 1=1"

	if agentID != "" {
		join = " JOIN jobs j ON job_runs.job_id = j.id"
		where += " AND j.agent_id = ?"
		args = append(args, agentID)
	}
	if jobID != "" {
		where += " AND job_runs.job_id = ?"
		args = append(args, jobID)
	}

	var total int
	countArgs := append([]any{}, args...)
	if err := s.db.Conn().QueryRow("SELECT COUNT(*) FROM job_runs"+join+where, countArgs...).Scan(&total); err != nil {
		http.Error(w, "failed to count runs", http.StatusInternalServerError)
		return
	}

	offset := (p.Page - 1) * p.Limit
	queryArgs := append(args, p.Limit, offset)
	rows, err := s.db.Conn().Query(
		"SELECT job_runs.id, job_runs.job_id, job_runs.exit_code, job_runs.output, job_runs.finished_at FROM job_runs"+join+where+
			" ORDER BY job_runs.finished_at DESC LIMIT ? OFFSET ?",
		queryArgs...,
	)
	if err != nil {
		http.Error(w, "failed to query runs", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	runs := []JobRun{}
	for rows.Next() {
		var run JobRun
		var exitCode sql.NullInt64
		var output sql.NullString
		var finishedAt sql.NullString
		if err := rows.Scan(&run.ID, &run.JobID, &exitCode, &output, &finishedAt); err != nil {
			http.Error(w, "failed to scan run", http.StatusInternalServerError)
			return
		}
		if exitCode.Valid {
			run.ExitCode = int(exitCode.Int64)
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
	json.NewEncoder(w).Encode(NewPaginatedResponse(runs, total, p.Page, p.Limit))
}

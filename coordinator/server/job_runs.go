package server

import (
	"encoding/json"
	"net/http"
)

// handleGetJobRuns handles GET /api/jobs/{id}/runs with optional pagination.
func (s *Server) handleGetJobRuns(w http.ResponseWriter, r *http.Request) {
	jobID := r.PathValue("id")

	exists, err := s.db.JobExists(jobID)
	if err != nil {
		http.Error(w, "failed to query job", http.StatusInternalServerError)
		return
	}
	if !exists {
		http.Error(w, "job not found", http.StatusNotFound)
		return
	}

	p := ParsePagination(r)
	offset := (p.Page - 1) * p.Limit

	dbRuns, total, err := s.db.ListJobRuns(jobID, p.Limit, offset)
	if err != nil {
		http.Error(w, "failed to query runs", http.StatusInternalServerError)
		return
	}

	runs := make([]JobRun, 0, len(dbRuns))
	for _, dbRun := range dbRuns {
		run := JobRun{ID: dbRun.ID, JobID: dbRun.JobID, StartedAt: dbRun.StartedAt}
		if dbRun.ExitCode != nil {
			run.ExitCode = *dbRun.ExitCode
		}
		if dbRun.Output != nil {
			run.Output = *dbRun.Output
		}
		if dbRun.FinishedAt != nil {
			run.FinishedAt = *dbRun.FinishedAt
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
	p := ParsePagination(r)

	filters := map[string]string{}
	if jobID := q.Get("job_id"); jobID != "" {
		filters["job_id"] = jobID
	}
	if agentID := q.Get("agent_id"); agentID != "" {
		filters["agent_id"] = agentID
	}
	if status := q.Get("status"); status != "" {
		filters["status"] = status
	}

	offset := (p.Page - 1) * p.Limit
	dbRuns, total, err := s.db.ListAllJobRuns(filters, p.Limit, offset)
	if err != nil {
		http.Error(w, "failed to query runs", http.StatusInternalServerError)
		return
	}

	runs := make([]JobRun, 0, len(dbRuns))
	for _, dbRun := range dbRuns {
		run := JobRun{
			ID:            dbRun.ID,
			JobID:         dbRun.JobID,
			JobName:       dbRun.JobName,
			SourcePath:    dbRun.SourcePath,
			DestPath:      dbRun.DestPath,
			AgentID:       dbRun.AgentID,
			AgentHostname: dbRun.AgentHostname,
			Status:        dbRun.Status,
			StartedAt:     dbRun.StartedAt,
		}
		if dbRun.ExitCode != nil {
			run.ExitCode = *dbRun.ExitCode
		}
		if dbRun.Output != nil {
			run.Output = *dbRun.Output
		}
		if dbRun.FinishedAt != nil {
			run.FinishedAt = *dbRun.FinishedAt
		}
		runs = append(runs, run)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(NewPaginatedResponse(runs, total, p.Page, p.Limit))
}

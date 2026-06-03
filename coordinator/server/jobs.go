package server

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"time"
)

// APIContract: matches dashboard/src/types/api.ts ProgressData interface
// Last synced: 2026-06-03
type ProgressData struct {
	FilesProcessed   int   `json:"files_processed"`
	BytesTransferred int64 `json:"bytes_transferred"`
	TotalFiles       int   `json:"total_files"`
	TotalBytes       int64 `json:"total_bytes"`
}

// APIContract: matches dashboard/src/types/api.ts Job interface
// Last synced: 2026-06-03
// Job is the domain type returned by all job endpoints.
type Job struct {
	ID         string                 `json:"id"`
	AgentID    string                 `json:"agent_id"`
	Name       string                 `json:"name"`
	SourcePath string                 `json:"source_path"`
	DestPath   string                 `json:"dest_path"`
	Schedule   *string                `json:"schedule,omitempty"`
	SyncFlags  map[string]interface{} `json:"sync_flags,omitempty"`
	Status     string                 `json:"status"`
	Progress   *ProgressData          `json:"progress,omitempty"`
	CreatedAt  string                 `json:"created_at"`
}

func newJobID() string {
	b := make([]byte, 8)
	rand.Read(b)
	return "job-" + hex.EncodeToString(b)
}

// handleCreateJob handles POST /api/jobs
// Supports single agent or group dispatch:
// - If agent_id is provided: creates single job
// - If group_id is provided: creates one job per group member with shared dispatch_id
// - Both cannot be provided (validation required)
func (s *Server) handleCreateJob(w http.ResponseWriter, r *http.Request) {
	var input struct {
		AgentID    string                 `json:"agent_id"`
		GroupID    *int                   `json:"group_id"`
		Name       string                 `json:"name"`
		SourcePath string                 `json:"source_path"`
		DestPath   string                 `json:"dest_path"`
		Schedule   *string                `json:"schedule"`
		SyncFlags  map[string]interface{} `json:"sync_flags"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}

	// Validation: exactly one of agent_id or group_id must be provided
	if (input.AgentID == "") == (input.GroupID == nil) {
		http.Error(w, "must provide either agent_id or group_id, not both", http.StatusBadRequest)
		return
	}

	if input.Name == "" {
		http.Error(w, "name is required", http.StatusBadRequest)
		return
	}
	if input.SourcePath == "" {
		http.Error(w, "source_path is required", http.StatusBadRequest)
		return
	}
	if input.DestPath == "" {
		http.Error(w, "dest_path is required", http.StatusBadRequest)
		return
	}

	// Single agent dispatch
	if input.AgentID != "" {
		job := Job{
			ID:         newJobID(),
			AgentID:    input.AgentID,
			Name:       input.Name,
			SourcePath: input.SourcePath,
			DestPath:   input.DestPath,
			Schedule:   input.Schedule,
			SyncFlags:  input.SyncFlags,
			Status:     "pending",
			CreatedAt:  time.Now().UTC().Format(time.RFC3339),
		}

		// Serialize sync_flags to JSON if present
		var syncFlagsJSON *string
		if job.SyncFlags != nil {
			data, err := json.Marshal(job.SyncFlags)
			if err != nil {
				http.Error(w, "invalid sync_flags JSON", http.StatusBadRequest)
				return
			}
			jsonStr := string(data)
			syncFlagsJSON = &jsonStr
		}

		_, err := s.db.Conn().Exec(
			`INSERT INTO jobs (id, agent_id, name, source_path, dest_path, schedule, sync_flags, status, created_at)
			 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			job.ID, job.AgentID, job.Name, job.SourcePath, job.DestPath, job.Schedule, syncFlagsJSON, job.Status, job.CreatedAt,
		)
		if err != nil {
			http.Error(w, "failed to create job", http.StatusInternalServerError)
			return
		}

		// Append to federation_events log for state sync.
		payload, _ := json.Marshal(job)
		s.db.AppendFederationEvent(s.coordinatorID, "job_created", string(payload))

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(job)
		return
	}

	// Group dispatch: create one job per group member with shared dispatch_id
	groupID := *input.GroupID

	// Verify group exists
	group, err := s.db.GetGroup(groupID)
	if err != nil {
		http.Error(w, "failed to fetch group", http.StatusInternalServerError)
		return
	}
	if group == nil {
		http.Error(w, "group not found", http.StatusNotFound)
		return
	}

	// Fetch group members
	agentIDs, err := s.db.GetGroupMembers(groupID)
	if err != nil {
		http.Error(w, "failed to fetch group members", http.StatusInternalServerError)
		return
	}

	if len(agentIDs) == 0 {
		http.Error(w, "group has no members", http.StatusBadRequest)
		return
	}

	// Generate shared dispatch_id for the batch
	dispatchID := "dispatch-" + newJobID()[4:] // reuse job ID generator, strip "job-" prefix

	// Create one job per group member
	jobs := []Job{}
	createdAt := time.Now().UTC().Format(time.RFC3339)

	for _, agentID := range agentIDs {
		// Serialize sync_flags to JSON if present
		var syncFlagsJSON *string
		if input.SyncFlags != nil {
			data, err := json.Marshal(input.SyncFlags)
			if err != nil {
				http.Error(w, "invalid sync_flags JSON", http.StatusBadRequest)
				return
			}
			jsonStr := string(data)
			syncFlagsJSON = &jsonStr
		}

		job := Job{
			ID:         newJobID(),
			AgentID:    agentID,
			Name:       input.Name,
			SourcePath: input.SourcePath,
			DestPath:   input.DestPath,
			Schedule:   input.Schedule,
			SyncFlags:  input.SyncFlags,
			Status:     "pending",
			CreatedAt:  createdAt,
		}

		_, err := s.db.Conn().Exec(
			`INSERT INTO jobs (id, agent_id, name, source_path, dest_path, schedule, sync_flags, status, created_at, group_id, dispatch_id)
			 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			job.ID, job.AgentID, job.Name, job.SourcePath, job.DestPath, job.Schedule, syncFlagsJSON, job.Status, job.CreatedAt, groupID, dispatchID,
		)
		if err != nil {
			http.Error(w, "failed to create job", http.StatusInternalServerError)
			return
		}
		jobs = append(jobs, job)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"dispatch_id": dispatchID,
		"group_id":    groupID,
		"jobs":        jobs,
	})
}

// handleListJobs handles GET /api/jobs
// Optional query params: ?agent_id=, ?search=, ?status=, ?page=, ?limit=
func (s *Server) handleListJobs(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	agentID := q.Get("agent_id")
	search := q.Get("search")
	status := q.Get("status")
	p := ParsePagination(r)

	args := []any{}
	where := " WHERE 1=1"
	if agentID != "" {
		where += " AND agent_id = ?"
		args = append(args, agentID)
	}
	if search != "" {
		where += " AND (name LIKE ? OR agent_id LIKE ?)"
		like := "%" + search + "%"
		args = append(args, like, like)
	}
	if status != "" {
		where += " AND status = ?"
		args = append(args, status)
	}

	var total int
	countArgs := append([]any{}, args...)
	if err := s.db.Conn().QueryRow("SELECT COUNT(*) FROM jobs"+where, countArgs...).Scan(&total); err != nil {
		http.Error(w, "failed to count jobs", http.StatusInternalServerError)
		return
	}

	offset := (p.Page - 1) * p.Limit
	queryArgs := append(args, p.Limit, offset)
	rows, err := s.db.Conn().Query(
		"SELECT id, agent_id, name, source_path, dest_path, schedule, sync_flags, status, progress, created_at FROM jobs"+where+
			" ORDER BY created_at DESC LIMIT ? OFFSET ?",
		queryArgs...,
	)
	if err != nil {
		http.Error(w, "failed to query jobs", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	jobs := []Job{}
	for rows.Next() {
		var j Job
		var schedule *string
		var syncFlagsJSON *string
		var progressJSON *string
		if err := rows.Scan(&j.ID, &j.AgentID, &j.Name, &j.SourcePath, &j.DestPath, &schedule, &syncFlagsJSON, &j.Status, &progressJSON, &j.CreatedAt); err != nil {
			http.Error(w, "failed to scan job", http.StatusInternalServerError)
			return
		}
		j.Schedule = schedule

		// Deserialize sync_flags JSON if present
		if syncFlagsJSON != nil {
			var syncFlags map[string]interface{}
			if err := json.Unmarshal([]byte(*syncFlagsJSON), &syncFlags); err == nil {
				j.SyncFlags = syncFlags
			}
		}

		// Deserialize progress JSON if present
		if progressJSON != nil {
			var p ProgressData
			if err := json.Unmarshal([]byte(*progressJSON), &p); err == nil {
				j.Progress = &p
			}
		}

		jobs = append(jobs, j)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(NewPaginatedResponse(jobs, total, p.Page, p.Limit))
}

// handleGetJob handles GET /api/jobs/{id}
func (s *Server) handleGetJob(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	var j Job
	var schedule *string
	var progressJSON *string
	err := s.db.Conn().QueryRow(
		`SELECT id, agent_id, name, source_path, dest_path, schedule, status, progress, created_at
		 FROM jobs WHERE id = ?`, id,
	).Scan(&j.ID, &j.AgentID, &j.Name, &j.SourcePath, &j.DestPath, &schedule, &j.Status, &progressJSON, &j.CreatedAt)

	if err == sql.ErrNoRows {
		http.Error(w, "job not found", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, "failed to query job", http.StatusInternalServerError)
		return
	}
	j.Schedule = schedule

	// Deserialize progress JSON if present
	if progressJSON != nil {
		var p ProgressData
		if err := json.Unmarshal([]byte(*progressJSON), &p); err == nil {
			j.Progress = &p
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(j)
}

// handleDeleteJob handles DELETE /api/jobs/{id}
func (s *Server) handleDeleteJob(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	result, err := s.db.Conn().Exec(`DELETE FROM jobs WHERE id = ?`, id)
	if err != nil {
		http.Error(w, "failed to delete job", http.StatusInternalServerError)
		return
	}

	n, _ := result.RowsAffected()
	if n == 0 {
		http.Error(w, "job not found", http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// handleCancelJob handles POST /api/jobs/{id}/cancel (Phase 20)
// Cancels a running job by marking it as "canceling" and notifying the agent
func (s *Server) handleCancelJob(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	// Fetch the job to check its current status
	var j Job
	var schedule *string
	var progressJSON *string
	err := s.db.Conn().QueryRow(
		`SELECT id, agent_id, name, source_path, dest_path, schedule, status, progress, created_at
		 FROM jobs WHERE id = ?`, id,
	).Scan(&j.ID, &j.AgentID, &j.Name, &j.SourcePath, &j.DestPath, &schedule, &j.Status, &progressJSON, &j.CreatedAt)

	if err == sql.ErrNoRows {
		http.Error(w, "job not found", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, "failed to query job", http.StatusInternalServerError)
		return
	}
	j.Schedule = schedule

	// Deserialize progress JSON if present
	if progressJSON != nil {
		var p ProgressData
		if err := json.Unmarshal([]byte(*progressJSON), &p); err == nil {
			j.Progress = &p
		}
	}

	// Only allow canceling running jobs
	if j.Status != "running" {
		http.Error(w, "job is not running", http.StatusBadRequest)
		return
	}

	// Update job status to "canceling"
	result, err := s.db.Conn().Exec(`UPDATE jobs SET status = ? WHERE id = ?`, "canceling", id)
	if err != nil {
		http.Error(w, "failed to update job status", http.StatusInternalServerError)
		return
	}

	n, _ := result.RowsAffected()
	if n == 0 {
		http.Error(w, "job not found", http.StatusNotFound)
		return
	}

	// Send cancel message to agent via WebSocket
	// TODO: Implement WebSocket message sending in Phase 20
	// For now, just return the updated job with status="canceling"

	// Return updated job
	j.Status = "canceling"
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(j)
}

// handlePostJobProgress handles POST /api/jobs/{id}/progress (Phase 20)
// Receives progress updates from agents and stores them in the job record
func (s *Server) handlePostJobProgress(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	// Parse progress data from request body
	var input ProgressData
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}

	// Serialize progress to JSON for storage
	progressJSON, err := json.Marshal(input)
	if err != nil {
		http.Error(w, "failed to serialize progress", http.StatusInternalServerError)
		return
	}
	progressJSONStr := string(progressJSON)

	// Update job progress in database
	result, err := s.db.Conn().Exec(`UPDATE jobs SET progress = ? WHERE id = ?`, progressJSONStr, id)
	if err != nil {
		http.Error(w, "failed to update progress", http.StatusInternalServerError)
		return
	}

	n, _ := result.RowsAffected()
	if n == 0 {
		http.Error(w, "job not found", http.StatusNotFound)
		return
	}

	// Fetch and return the updated job
	var j Job
	var schedule *string
	var progressJSONFromDB *string
	err = s.db.Conn().QueryRow(
		`SELECT id, agent_id, name, source_path, dest_path, schedule, status, progress, created_at
		 FROM jobs WHERE id = ?`, id,
	).Scan(&j.ID, &j.AgentID, &j.Name, &j.SourcePath, &j.DestPath, &schedule, &j.Status, &progressJSONFromDB, &j.CreatedAt)

	if err != nil {
		http.Error(w, "failed to fetch updated job", http.StatusInternalServerError)
		return
	}
	j.Schedule = schedule

	// Deserialize progress JSON if present
	if progressJSONFromDB != nil {
		var p ProgressData
		if err := json.Unmarshal([]byte(*progressJSONFromDB), &p); err == nil {
			j.Progress = &p
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(j)
}

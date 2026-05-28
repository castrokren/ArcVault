package server

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"time"
)

// Job is the domain type returned by all job endpoints.
type Job struct {
	ID         string  `json:"id"`
	AgentID    string  `json:"agent_id"`
	Name       string  `json:"name"`
	SourcePath string  `json:"source_path"`
	DestPath   string  `json:"dest_path"`
	Schedule   *string `json:"schedule,omitempty"`
	Status     string  `json:"status"`
	CreatedAt  string  `json:"created_at"`
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
		AgentID    string  `json:"agent_id"`
		GroupID    *int    `json:"group_id"`
		Name       string  `json:"name"`
		SourcePath string  `json:"source_path"`
		DestPath   string  `json:"dest_path"`
		Schedule   *string `json:"schedule"`
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
			Status:     "pending",
			CreatedAt:  time.Now().UTC().Format(time.RFC3339),
		}

		_, err := s.db.Conn().Exec(
			`INSERT INTO jobs (id, agent_id, name, source_path, dest_path, schedule, status, created_at)
			 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
			job.ID, job.AgentID, job.Name, job.SourcePath, job.DestPath, job.Schedule, job.Status, job.CreatedAt,
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
		job := Job{
			ID:         newJobID(),
			AgentID:    agentID,
			Name:       input.Name,
			SourcePath: input.SourcePath,
			DestPath:   input.DestPath,
			Schedule:   input.Schedule,
			Status:     "pending",
			CreatedAt:  createdAt,
		}

		_, err := s.db.Conn().Exec(
			`INSERT INTO jobs (id, agent_id, name, source_path, dest_path, schedule, status, created_at, group_id, dispatch_id)
			 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			job.ID, job.AgentID, job.Name, job.SourcePath, job.DestPath, job.Schedule, job.Status, job.CreatedAt, groupID, dispatchID,
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
		"SELECT id, agent_id, name, source_path, dest_path, schedule, status, created_at FROM jobs"+where+
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
		if err := rows.Scan(&j.ID, &j.AgentID, &j.Name, &j.SourcePath, &j.DestPath, &schedule, &j.Status, &j.CreatedAt); err != nil {
			http.Error(w, "failed to scan job", http.StatusInternalServerError)
			return
		}
		j.Schedule = schedule
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
	err := s.db.Conn().QueryRow(
		`SELECT id, agent_id, name, source_path, dest_path, schedule, status, created_at
		 FROM jobs WHERE id = ?`, id,
	).Scan(&j.ID, &j.AgentID, &j.Name, &j.SourcePath, &j.DestPath, &schedule, &j.Status, &j.CreatedAt)

	if err == sql.ErrNoRows {
		http.Error(w, "job not found", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, "failed to query job", http.StatusInternalServerError)
		return
	}
	j.Schedule = schedule

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

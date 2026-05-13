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
func (s *Server) handleCreateJob(w http.ResponseWriter, r *http.Request) {
	var input struct {
		AgentID    string  `json:"agent_id"`
		Name       string  `json:"name"`
		SourcePath string  `json:"source_path"`
		DestPath   string  `json:"dest_path"`
		Schedule   *string `json:"schedule"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}
	if input.AgentID == "" {
		http.Error(w, "agent_id is required", http.StatusBadRequest)
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

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(job)
}

// handleListJobs handles GET /api/jobs
// Optional query param: ?agent_id=<id> to filter by agent
func (s *Server) handleListJobs(w http.ResponseWriter, r *http.Request) {
	agentID := r.URL.Query().Get("agent_id")

	var (
		rows *sql.Rows
		err  error
	)
	if agentID != "" {
		rows, err = s.db.Conn().Query(
			`SELECT id, agent_id, name, source_path, dest_path, schedule, status, created_at
			 FROM jobs WHERE agent_id = ? ORDER BY created_at DESC`,
			agentID,
		)
	} else {
		rows, err = s.db.Conn().Query(
			`SELECT id, agent_id, name, source_path, dest_path, schedule, status, created_at
			 FROM jobs ORDER BY created_at DESC`,
		)
	}
	if err != nil {
		http.Error(w, "failed to query jobs", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	jobs := []Job{}
	for rows.Next() {
		var j Job
		var schedule sql.NullString
		if err := rows.Scan(&j.ID, &j.AgentID, &j.Name, &j.SourcePath, &j.DestPath, &schedule, &j.Status, &j.CreatedAt); err != nil {
			http.Error(w, "failed to scan job", http.StatusInternalServerError)
			return
		}
		if schedule.Valid {
			j.Schedule = &schedule.String
		}
		jobs = append(jobs, j)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(jobs)
}

// handleGetJob handles GET /api/jobs/{id}
func (s *Server) handleGetJob(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	var j Job
	var schedule sql.NullString
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
	if schedule.Valid {
		j.Schedule = &schedule.String
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

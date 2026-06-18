package server

import (
	"encoding/json"
	"net/http"
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
	ID          string                 `json:"id"`
	AgentID     string                 `json:"agent_id"`
	Name        string                 `json:"name"`
	SourcePath  string                 `json:"source_path"`
	DestPath    string                 `json:"dest_path"`
	Schedule    *string                `json:"schedule,omitempty"`
	SyncFlags   map[string]interface{} `json:"sync_flags,omitempty"`
	Status      string                 `json:"status"`
	Progress    *ProgressData          `json:"progress,omitempty"`
	Credentials map[string]interface{} `json:"credentials,omitempty"`
	CreatedAt   string                 `json:"created_at"`
}

// handleCreateJob handles POST /api/jobs
// Supports single agent or group dispatch:
// - If agent_id is provided: creates single job
// - If group_id is provided: creates one job per group member with shared dispatch_id
// - Both cannot be provided (validation required)
func (s *Server) handleCreateJob(w http.ResponseWriter, r *http.Request) {
	var input struct {
		AgentID              string                 `json:"agent_id"`
		GroupID              *int                   `json:"group_id"`
		Name                 string                 `json:"name"`
		SourcePath           string                 `json:"source_path"`
		DestPath             string                 `json:"dest_path"`
		Schedule             *string                `json:"schedule"`
		SyncFlags            map[string]interface{} `json:"sync_flags"`
		CredentialProfileID  string                 `json:"credential_profile_id,omitempty"`
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

	// Single agent dispatch
	if input.AgentID != "" {
		jobDTO, err := s.jobService.CreateJob(input.AgentID, input.Name, input.SourcePath, input.DestPath, input.Schedule, input.SyncFlags)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		// Validate and assign credential profile if provided
		if input.CredentialProfileID != "" {
			profile, err := s.db.GetCredentialProfile(input.CredentialProfileID)
			if err != nil {
				http.Error(w, "failed to validate credential profile", http.StatusInternalServerError)
				return
			}
			if profile == nil {
				http.Error(w, "credential profile not found", http.StatusNotFound)
				return
			}

			// Get agent to check OS compatibility
			agent, err := s.db.GetAgent(input.AgentID)
			if err != nil {
				http.Error(w, "agent not found", http.StatusNotFound)
				return
			}

			// Validate credential type matches agent OS
			if !s.validateCredentialTypeForAgent(profile.Type, agent.OS) {
				http.Error(w, "credential profile type incompatible with agent OS", http.StatusUnprocessableEntity)
				return
			}

			// Assign profile to job
			if err := s.db.UpdateJobCredentialProfile(jobDTO.ID, input.CredentialProfileID); err != nil {
				http.Error(w, "failed to assign credential profile to job", http.StatusInternalServerError)
				return
			}
		}

		// Convert DTO to response and append to federation_events log
		job := Job{
			ID:         jobDTO.ID,
			AgentID:    jobDTO.AgentID,
			Name:       jobDTO.Name,
			SourcePath: jobDTO.SourcePath,
			DestPath:   jobDTO.DestPath,
			Schedule:   jobDTO.Schedule,
			SyncFlags:  jobDTO.SyncFlags,
			Status:     jobDTO.Status,
			CreatedAt:  jobDTO.CreatedAt,
		}

		// Log to federation_events for state sync
		payload, _ := json.Marshal(job)
		s.db.AppendFederationEvent(s.coordinatorID, "job_created", string(payload))

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(job)
		return
	}

	// Group dispatch
	groupID := *input.GroupID
	response, err := s.jobService.CreateJobForGroup(groupID, input.Name, input.SourcePath, input.DestPath, input.Schedule, input.SyncFlags)
	if err != nil {
		// Determine HTTP status based on error
		statusCode := http.StatusInternalServerError
		if err.Error() == "group not found" {
			statusCode = http.StatusNotFound
		} else if err.Error() == "group has no members" {
			statusCode = http.StatusBadRequest
		}
		http.Error(w, err.Error(), statusCode)
		return
	}

	// Convert service response to API response
	jobs := make([]Job, len(response.Jobs))
	for i, jobDTO := range response.Jobs {
		jobs[i] = Job{
			ID:         jobDTO.ID,
			AgentID:    jobDTO.AgentID,
			Name:       jobDTO.Name,
			SourcePath: jobDTO.SourcePath,
			DestPath:   jobDTO.DestPath,
			Schedule:   jobDTO.Schedule,
			SyncFlags:  jobDTO.SyncFlags,
			Status:     jobDTO.Status,
			CreatedAt:  jobDTO.CreatedAt,
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"dispatch_id": response.DispatchID,
		"group_id":    response.GroupID,
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

	offset := (p.Page - 1) * p.Limit
	result, err := s.jobService.ListJobs(search, status, agentID, p.Limit, offset)
	if err != nil {
		http.Error(w, "failed to list jobs", http.StatusInternalServerError)
		return
	}

	// Convert service DTOs to response DTOs
	jobs := make([]Job, len(result.Jobs))
	isAgentRequest := isAgentTokenRequest(r)

	for i, jobDTO := range result.Jobs {
		jobs[i] = Job{
			ID:         jobDTO.ID,
			AgentID:    jobDTO.AgentID,
			Name:       jobDTO.Name,
			SourcePath: jobDTO.SourcePath,
			DestPath:   jobDTO.DestPath,
			Schedule:   jobDTO.Schedule,
			SyncFlags:  jobDTO.SyncFlags,
			Status:     jobDTO.Status,
			CreatedAt:  jobDTO.CreatedAt,
		}

		// Inject credentials for agent token requests
		if isAgentRequest {
			credProfileID, err := s.db.GetJobCredentialProfileID(jobDTO.ID)
			if err == nil && credProfileID != "" {
				credentials := s.decryptCredentials(credProfileID)
				if credentials != nil {
					jobs[i].Credentials = credentials
				}
			}
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(NewPaginatedResponse(jobs, result.Total, result.Page, result.Limit))
}

// handleGetJob handles GET /api/jobs/{id}
func (s *Server) handleGetJob(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	jobDTO, err := s.jobService.GetJob(id)
	if err != nil {
		http.Error(w, "job not found", http.StatusNotFound)
		return
	}

	j := Job{
		ID:         jobDTO.ID,
		AgentID:    jobDTO.AgentID,
		Name:       jobDTO.Name,
		SourcePath: jobDTO.SourcePath,
		DestPath:   jobDTO.DestPath,
		Schedule:   jobDTO.Schedule,
		SyncFlags:  jobDTO.SyncFlags,
		Status:     jobDTO.Status,
		CreatedAt:  jobDTO.CreatedAt,
	}

	// Inject credentials for agent token requests
	if isAgentTokenRequest(r) {
		credProfileID, err := s.db.GetJobCredentialProfileID(id)
		if err == nil && credProfileID != "" {
			credentials := s.decryptCredentials(credProfileID)
			if credentials != nil {
				j.Credentials = credentials
			}
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(j)
}

// handleDeleteJob handles DELETE /api/jobs/{id}
func (s *Server) handleDeleteJob(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	// Verify job exists before deleting
	if exists, err := s.db.JobExists(id); err != nil || !exists {
		http.Error(w, "job not found", http.StatusNotFound)
		return
	}

	if err := s.jobService.DeleteJob(id); err != nil {
		http.Error(w, "failed to delete job", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// handleCancelJob handles POST /api/jobs/{id}/cancel (Phase 20)
// Cancels a running job by marking it as "canceling" and notifying the agent
func (s *Server) handleCancelJob(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	// Fetch the job to check its current status
	jobDTO, err := s.jobService.GetJob(id)
	if err != nil {
		http.Error(w, "job not found", http.StatusNotFound)
		return
	}

	// Only allow canceling running jobs
	if jobDTO.Status != "running" {
		http.Error(w, "job is not running", http.StatusBadRequest)
		return
	}

	// Update job status to "canceling"
	if err := s.jobService.UpdateJobStatus(id, "canceling"); err != nil {
		http.Error(w, "failed to update job status", http.StatusInternalServerError)
		return
	}

	// Send cancel message to agent via WebSocket
	// TODO: Implement WebSocket message sending in Phase 20
	// For now, just return the updated job with status="canceling"

	// Return updated job
	j := Job{
		ID:         jobDTO.ID,
		AgentID:    jobDTO.AgentID,
		Name:       jobDTO.Name,
		SourcePath: jobDTO.SourcePath,
		DestPath:   jobDTO.DestPath,
		Schedule:   jobDTO.Schedule,
		SyncFlags:  jobDTO.SyncFlags,
		Status:     "canceling",
		CreatedAt:  jobDTO.CreatedAt,
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(j)
}

// handlePostJobProgress handles POST /api/jobs/{id}/progress (Phase 20)
// Receives progress updates from agents and stores them in the job record
// ProgressUpdate is the request body for POST /api/jobs/{id}/progress
type ProgressUpdate struct {
	Percentage int      `json:"percentage"`
	Logs       []string `json:"logs"`
	Status     string   `json:"status"`
}

func (s *Server) handlePostJobProgress(w http.ResponseWriter, r *http.Request) {
	jobID := r.PathValue("id")
	if jobID == "" {
		http.Error(w, "missing job ID", http.StatusBadRequest)
		return
	}

	// Verify job exists
	var jobExists bool
	err := s.db.Conn().QueryRow(`SELECT EXISTS(SELECT 1 FROM jobs WHERE id = ?)`, jobID).Scan(&jobExists)
	if err != nil || !jobExists {
		http.Error(w, "job not found", http.StatusNotFound)
		return
	}

	// Parse progress update from request body
	var update ProgressUpdate
	if err := json.NewDecoder(r.Body).Decode(&update); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}

	// Validate percentage (0-100)
	if update.Percentage < 0 || update.Percentage > 100 {
		http.Error(w, "percentage must be between 0 and 100", http.StatusBadRequest)
		return
	}

	// Validate status
	validStatuses := map[string]bool{"running": true, "completed": true, "cancelled": true, "error": true}
	if !validStatuses[update.Status] {
		http.Error(w, "invalid status", http.StatusBadRequest)
		return
	}

	// Get the latest job run for this job
	var runID string
	err = s.db.Conn().QueryRow(
		`SELECT id FROM job_runs WHERE job_id = ? ORDER BY started_at DESC LIMIT 1`,
		jobID,
	).Scan(&runID)
	if err != nil {
		http.Error(w, "no job run found", http.StatusInternalServerError)
		return
	}

	// Update job_runs with progress and status
	_, err = s.db.Conn().Exec(
		`UPDATE job_runs SET progress = ?, status = ? WHERE id = ?`,
		update.Percentage, update.Status, runID,
	)
	if err != nil {
		http.Error(w, "failed to update progress", http.StatusInternalServerError)
		return
	}

	// Insert logs into job_logs table
	for _, logLine := range update.Logs {
		_, err := s.db.Conn().Exec(
			`INSERT INTO job_logs (job_id, line) VALUES (?, ?)`,
			jobID, logLine,
		)
		if err != nil {
			http.Error(w, "failed to insert logs", http.StatusInternalServerError)
			return
		}
	}

	// Return success response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":    true,
		"job_id":     jobID,
		"percentage": update.Percentage,
		"status":     update.Status,
	})
}

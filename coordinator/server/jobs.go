package server

import (
	"encoding/json"
	"fmt"
	"log"
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
	claims := GetUserClaims(r)

	var input struct {
		AgentID             string                 `json:"agent_id"`
		GroupID             *int                   `json:"group_id"`
		Name                string                 `json:"name"`
		SourcePath          string                 `json:"source_path"`
		DestPath            string                 `json:"dest_path"`
		Schedule            *string                `json:"schedule"`
		SyncFlags           map[string]interface{} `json:"sync_flags"`
		CredentialProfileID string                 `json:"credential_profile_id,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}

	// Validation: exactly one of agent_id or group_id must be provided
	if (input.AgentID == "") == (input.GroupID == nil) {
		http.Error(w, "must provide either agent_id or group_id, not both", http.StatusBadRequest)
		s.logAudit(r, claims, "job.create", false, nil, nil)
		return
	}

	// Single agent dispatch
	if input.AgentID != "" {
		jobDTO, err := s.jobService.CreateJob(input.AgentID, input.Name, input.SourcePath, input.DestPath, input.Schedule, input.SyncFlags)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			s.logAudit(r, claims, "job.create", false, nil, nil)
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

		// Notify agent to poll immediately
		if err := s.hub.SendToAgent(input.AgentID, map[string]string{"type": "poll_now"}); err != nil {
			log.Printf("Job %s: agent %s not connected via WS, will pick up on next poll cycle: %v", jobDTO.ID, input.AgentID, err)
		}

		s.logAudit(r, claims, "job.create", true, strPtr("job"), strPtr(jobDTO.ID))

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
		s.logAudit(r, claims, "job.create", false, nil, nil)
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

	// Notify all agents in the group to poll immediately
	for _, j := range jobs {
		if err := s.hub.SendToAgent(j.AgentID, map[string]string{"type": "poll_now"}); err != nil {
			log.Printf("Job %s: agent %s not connected via WS: %v", j.ID, j.AgentID, err)
		}
	}

	s.logAudit(r, claims, "job.create", true, strPtr("job"), strPtr(response.DispatchID))

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
	requestingAgentID := GetAgentID(r)

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

		// Inject credentials only when the requesting agent owns this job.
		if requestingAgentID != "" && jobDTO.AgentID == requestingAgentID {
			credProfileID, err := s.db.GetJobCredentialProfileID(jobDTO.ID)
			if err != nil {
				http.Error(w, "failed to look up job credentials", http.StatusInternalServerError)
				return
			}
			if credProfileID != "" {
				credentials, err := s.decryptCredentials(credProfileID)
				if err != nil {
					http.Error(w, "failed to decrypt job credentials", http.StatusInternalServerError)
					return
				}
				jobs[i].Credentials = credentials
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

	// Inject credentials only when the requesting agent owns this job.
	if agentID := GetAgentID(r); agentID != "" && jobDTO.AgentID == agentID {
		credProfileID, err := s.db.GetJobCredentialProfileID(id)
		if err != nil {
			http.Error(w, "failed to look up job credentials", http.StatusInternalServerError)
			return
		}
		if credProfileID != "" {
			credentials, err := s.decryptCredentials(credProfileID)
			if err != nil {
				http.Error(w, "failed to decrypt job credentials", http.StatusInternalServerError)
				return
			}
			j.Credentials = credentials
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(j)
}

// handleDeleteJob handles DELETE /api/jobs/{id}
func (s *Server) handleDeleteJob(w http.ResponseWriter, r *http.Request) {
	claims := GetUserClaims(r)
	id := r.PathValue("id")

	// Verify job exists before deleting
	if exists, err := s.db.JobExists(id); err != nil || !exists {
		http.Error(w, "job not found", http.StatusNotFound)
		s.logAudit(r, claims, "job.delete", false, strPtr("job"), strPtr(id))
		return
	}

	if err := s.jobService.DeleteJob(id); err != nil {
		http.Error(w, "failed to delete job", http.StatusInternalServerError)
		s.logAudit(r, claims, "job.delete", false, strPtr("job"), strPtr(id))
		return
	}

	s.logAudit(r, claims, "job.delete", true, strPtr("job"), strPtr(id))

	w.WriteHeader(http.StatusNoContent)
}

// handleCancelJob handles POST /api/jobs/{id}/cancel
// Cancels a pending job immediately (status → cancelled) or signals a running
// job by setting status to "canceling" and sending a WebSocket message to the agent.
func (s *Server) handleCancelJob(w http.ResponseWriter, r *http.Request) {
	claims := GetUserClaims(r)
	id := r.PathValue("id")

	// Fetch the job to check its current status
	jobDTO, err := s.jobService.GetJob(id)
	if err != nil {
		http.Error(w, "job not found", http.StatusNotFound)
		s.logAudit(r, claims, "job.cancel", false, strPtr("job"), strPtr(id))
		return
	}

	switch jobDTO.Status {
	case "pending":
		// Pending job — agent hasn't picked it up yet, cancel immediately
		if err := s.jobService.UpdateJobStatus(id, "cancelled"); err != nil {
			http.Error(w, "failed to cancel job", http.StatusInternalServerError)
			s.logAudit(r, claims, "job.cancel", false, strPtr("job"), strPtr(id))
			return
		}

	case "running":
		// Running job — mark as canceling and notify agent via WebSocket
		if err := s.jobService.UpdateJobStatus(id, "canceling"); err != nil {
			http.Error(w, "failed to update job status", http.StatusInternalServerError)
			s.logAudit(r, claims, "job.cancel", false, strPtr("job"), strPtr(id))
			return
		}

		// Send cancel message to agent via WebSocket
		if err := s.hub.SendToAgent(jobDTO.AgentID, map[string]interface{}{
			"type":   "cancel_command",
			"job_id": id,
		}); err != nil {
			// Agent not connected — job is still marked as "canceling".
			// The agent will pick it up on next poll or reconnect.
			log.Printf("Warning: agent %s not connected for cancel of job %s: %v", jobDTO.AgentID, id, err)
		}

	default:
		// Already completed, failed, or cancelled — reject
		http.Error(w, fmt.Sprintf("job is already %s", jobDTO.Status), http.StatusBadRequest)
		s.logAudit(r, claims, "job.cancel", false, strPtr("job"), strPtr(id))
		return
	}

	// Broadcast cancellation event to dashboard clients
	s.hub.Broadcast(Event{
		Type:    "job.updated",
		Payload: map[string]string{"id": id, "status": jobDTO.Status},
	})

	// Build response with the new status
	newStatus := "cancelled"
	if jobDTO.Status == "running" {
		newStatus = "canceling"
	}

	j := Job{
		ID:         jobDTO.ID,
		AgentID:    jobDTO.AgentID,
		Name:       jobDTO.Name,
		SourcePath: jobDTO.SourcePath,
		DestPath:   jobDTO.DestPath,
		Schedule:   jobDTO.Schedule,
		SyncFlags:  jobDTO.SyncFlags,
		Status:     newStatus,
		CreatedAt:  jobDTO.CreatedAt,
	}

	s.logAudit(r, claims, "job.cancel", true, strPtr("job"), strPtr(id))

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

// CreateJobRequest defines the request to create a new job
type CreateJobRequest struct {
	Name       string  `json:"name"`
	AgentID    *string `json:"agent_id"`
	GroupID    *int    `json:"group_id"`
	SourcePath string  `json:"source_path"`
	DestPath   string  `json:"dest_path"`
	Schedule   *string `json:"schedule"`
	SyncFlags  *string `json:"sync_flags"`
}

// Validate checks if CreateJobRequest is valid
func (r *CreateJobRequest) Validate() error {
	if r.Name == "" {
		return fmt.Errorf("name is required")
	}
	if len(r.Name) < 1 || len(r.Name) > 255 {
		return fmt.Errorf("name must be 1-255 characters")
	}
	if r.SourcePath == "" {
		return fmt.Errorf("source_path is required")
	}
	if len(r.SourcePath) < 1 || len(r.SourcePath) > 4096 {
		return fmt.Errorf("source_path must be 1-4096 characters")
	}
	if r.DestPath == "" {
		return fmt.Errorf("dest_path is required")
	}
	if len(r.DestPath) < 1 || len(r.DestPath) > 4096 {
		return fmt.Errorf("dest_path must be 1-4096 characters")
	}

	// Must provide either agent_id or group_id, not both
	hasAgent := r.AgentID != nil && *r.AgentID != ""
	hasGroup := r.GroupID != nil && *r.GroupID > 0

	if hasAgent && hasGroup {
		return fmt.Errorf("cannot provide both agent_id and group_id")
	}
	if !hasAgent && !hasGroup {
		return fmt.Errorf("must provide either agent_id or group_id")
	}

	// If agent_id provided, validate it's a UUID
	if hasAgent && !isValidUUID(*r.AgentID) {
		return fmt.Errorf("agent_id must be a valid UUID")
	}

	// Schedule and sync_flags are optional, no validation on format here
	return nil
}

// JobResponse defines the job response
type JobResponse struct {
	JobID      string  `json:"job_id"`
	Name       string  `json:"name"`
	AgentID    *string `json:"agent_id"`
	GroupID    *int    `json:"group_id"`
	DispatchID *string `json:"dispatch_id"`
	SourcePath string  `json:"source_path"`
	DestPath   string  `json:"dest_path"`
	Schedule   *string `json:"schedule"`
	SyncFlags  *string `json:"sync_flags"`
	Status     string  `json:"status"`
	CreatedAt  string  `json:"created_at"`
}

// PaginatedJobsResponse wraps paginated jobs list
type PaginatedJobsResponse struct {
	Data       []JobResponse  `json:"data"`
	Pagination PaginationMeta `json:"pagination"`
}

// PostJobProgressRequest defines the request to post job progress
type PostJobProgressRequest struct {
	Percentage int     `json:"percentage"`
	Status     *string `json:"status"`
}

// Validate checks if PostJobProgressRequest is valid
func (r *PostJobProgressRequest) Validate() error {
	if r.Percentage < 0 || r.Percentage > 100 {
		return fmt.Errorf("percentage must be between 0 and 100")
	}
	if r.Status != nil {
		validStatuses := []string{"pending", "in_progress", "completed", "failed"}
		isValid := false
		for _, s := range validStatuses {
			if *r.Status == s {
				isValid = true
				break
			}
		}
		if !isValid {
			return fmt.Errorf("status must be 'pending', 'in_progress', 'completed', or 'failed'")
		}
	}
	return nil
}

// JobProgressResponse defines the response after posting progress
type JobProgressResponse struct {
	Percentage int    `json:"percentage"`
	Status     string `json:"status"`
	UpdatedAt  string `json:"updated_at"`
}

// PostJobResultsRequest defines the request to post job execution results
type PostJobResultsRequest struct {
	RunID    string  `json:"run_id"`
	ExitCode int     `json:"exit_code"`
	Error    *string `json:"error"`
}

// Validate checks if PostJobResultsRequest is valid
func (r *PostJobResultsRequest) Validate() error {
	if r.RunID == "" {
		return fmt.Errorf("run_id is required")
	}
	if !isValidUUID(r.RunID) {
		return fmt.Errorf("run_id must be a valid UUID")
	}
	if r.ExitCode < 0 || r.ExitCode > 255 {
		return fmt.Errorf("exit_code must be between 0 and 255")
	}
	return nil
}

// PostJobResultsResponse defines the response after posting results
type PostJobResultsResponse struct {
	RunID    string  `json:"run_id"`
	JobID    string  `json:"job_id"`
	Status   string  `json:"status"`
	ExitCode int     `json:"exit_code"`
	Error    *string `json:"error"`
}

// JobRunResponse defines a single job run (execution)
type JobRunResponse struct {
	RunID    string  `json:"run_id"`
	JobID    string  `json:"job_id"`
	RunStart string  `json:"run_start"`
	RunEnd   *string `json:"run_end"`
	Status   string  `json:"status"`
	ExitCode *int    `json:"exit_code"`
	Error    *string `json:"error"`
}

// PaginatedJobRunsResponse wraps paginated job runs list
type PaginatedJobRunsResponse struct {
	Data       []JobRunResponse `json:"data"`
	Pagination PaginationMeta   `json:"pagination"`
}

// JobLogEntry defines a single progress log entry
type JobLogEntry struct {
	Timestamp  string `json:"timestamp"`
	Percentage int    `json:"percentage"`
	Status     string `json:"status"`
}

// JobProgressGetResponse defines the response for getting job progress
type JobProgressGetResponse struct {
	JobID      string        `json:"job_id"`
	Percentage int           `json:"percentage"`
	Status     string        `json:"status"`
	RecentLogs []JobLogEntry `json:"recent_logs"`
	UpdatedAt  string        `json:"updated_at"`
}

// PaginatedJobLogsResponse wraps paginated job logs
type PaginatedJobLogsResponse struct {
	Data       []JobLogEntry  `json:"data"`
	Pagination PaginationMeta `json:"pagination"`
}

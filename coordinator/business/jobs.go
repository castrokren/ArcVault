package business

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"arcvault/coordinator/db"
)

// JobService handles job-related business logic.
// Accepts db.AllQueries which includes JobQueries, JobRunQueries, GroupQueries, and federation event logging.
//
// Key design principles:
// - All database operations go through typed interfaces (JobQueries, JobRunQueries, GroupQueries)
// - Service layer returns clean DTOs, not raw database models
// - Validation and business logic lives here, not in handlers
// - Handlers remain focused on HTTP concerns (parsing, status codes, serialization)
type JobService struct {
	db db.AllQueries
}

// NewJobService creates a new job service with access to DB operations.
func NewJobService(database db.AllQueries) *JobService {
	return &JobService{
		db: database,
	}
}

// JobDTO is the data transfer object for jobs (API response).
type JobDTO struct {
	ID         string                 `json:"id"`
	AgentID    string                 `json:"agent_id"`
	Name       string                 `json:"name"`
	SourcePath string                 `json:"source_path"`
	DestPath   string                 `json:"dest_path"`
	Schedule   *string                `json:"schedule,omitempty"`
	SyncFlags  map[string]interface{} `json:"sync_flags,omitempty"`
	Status     string                 `json:"status"`
	CreatedAt  string                 `json:"created_at"`
}

// ListJobsDTO returns jobs with pagination.
type ListJobsDTO struct {
	Jobs  []JobDTO `json:"data"`
	Total int      `json:"total"`
	Page  int      `json:"page"`
	Pages int      `json:"pages"`
	Limit int      `json:"limit"`
}

// newJobID generates a unique job ID.
func newJobID() string {
	b := make([]byte, 8)
	rand.Read(b)
	return "job-" + hex.EncodeToString(b)
}

// CreateJob creates a single job for a specific agent.
func (s *JobService) CreateJob(agentID, name, sourcePath, destPath string, schedule *string, syncFlags map[string]interface{}) (*JobDTO, error) {
	if name == "" {
		return nil, fmt.Errorf("name is required")
	}
	if sourcePath == "" {
		return nil, fmt.Errorf("source_path is required")
	}
	if destPath == "" {
		return nil, fmt.Errorf("dest_path is required")
	}

	jobID := newJobID()
	createdAt := time.Now().UTC().Format(time.RFC3339)

	// Serialize sync_flags to JSON if present
	var syncFlagsJSON *string
	if syncFlags != nil {
		data, err := json.Marshal(syncFlags)
		if err != nil {
			return nil, fmt.Errorf("invalid sync_flags JSON: %w", err)
		}
		jsonStr := string(data)
		syncFlagsJSON = &jsonStr
	}

	if err := s.db.CreateJob(jobID, agentID, name, sourcePath, destPath, schedule, syncFlagsJSON, "pending", createdAt); err != nil {
		return nil, fmt.Errorf("failed to create job: %w", err)
	}

	return &JobDTO{
		ID:         jobID,
		AgentID:    agentID,
		Name:       name,
		SourcePath: sourcePath,
		DestPath:   destPath,
		Schedule:   schedule,
		SyncFlags:  syncFlags,
		Status:     "pending",
		CreatedAt:  createdAt,
	}, nil
}

// ListJobs returns jobs with optional filters and pagination.
func (s *JobService) ListJobs(search, status, agentID string, limit, offset int) (*ListJobsDTO, error) {
	jobs, total, err := s.db.ListJobs(search, status, agentID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to list jobs: %w", err)
	}

	dtos := make([]JobDTO, len(jobs))
	for i, job := range jobs {
		// Parse sync_flags from JSON if present
		var syncFlags map[string]interface{}
		if job.SyncFlags != nil {
			if err := json.Unmarshal([]byte(*job.SyncFlags), &syncFlags); err != nil {
				// If unmarshaling fails, just leave syncFlags empty
				syncFlags = map[string]interface{}{}
			}
		}

		dtos[i] = JobDTO{
			ID:         job.ID,
			AgentID:    job.AgentID,
			Name:       job.Name,
			SourcePath: job.SourcePath,
			DestPath:   job.DestPath,
			Schedule:   job.Schedule,
			SyncFlags:  syncFlags,
			Status:     job.Status,
			CreatedAt:  job.CreatedAt,
		}
	}

	pages := (total + limit - 1) / limit
	if pages == 0 {
		pages = 1
	}

	return &ListJobsDTO{
		Jobs:  dtos,
		Total: total,
		Page:  (offset / limit) + 1,
		Pages: pages,
		Limit: limit,
	}, nil
}

// GetJob returns a single job by ID.
func (s *JobService) GetJob(jobID string) (*JobDTO, error) {
	job, err := s.db.GetJob(jobID)
	if err != nil {
		return nil, fmt.Errorf("job not found")
	}

	// Parse sync_flags from JSON if present
	var syncFlags map[string]interface{}
	if job.SyncFlags != nil {
		if err := json.Unmarshal([]byte(*job.SyncFlags), &syncFlags); err != nil {
			syncFlags = map[string]interface{}{}
		}
	}

	return &JobDTO{
		ID:         job.ID,
		AgentID:    job.AgentID,
		Name:       job.Name,
		SourcePath: job.SourcePath,
		DestPath:   job.DestPath,
		Schedule:   job.Schedule,
		SyncFlags:  syncFlags,
		Status:     job.Status,
		CreatedAt:  job.CreatedAt,
	}, nil
}

// UpdateJobStatus updates the status of a job.
func (s *JobService) UpdateJobStatus(jobID, status string) error {
	if err := s.db.UpdateJobStatus(jobID, status); err != nil {
		return fmt.Errorf("failed to update job status: %w", err)
	}
	return nil
}

// CancelJob cancels a job (sets status to 'cancelled').
func (s *JobService) CancelJob(jobID string) error {
	return s.UpdateJobStatus(jobID, "cancelled")
}

// DeleteJob removes a job by ID.
func (s *JobService) DeleteJob(jobID string) error {
	if err := s.db.DeleteJob(jobID); err != nil {
		return fmt.Errorf("failed to delete job: %w", err)
	}
	return nil
}

// GroupDispatchResponse is the response returned when creating a job for a group.
type GroupDispatchResponse struct {
	DispatchID string   `json:"dispatch_id"`
	GroupID    int      `json:"group_id"`
	Jobs       []JobDTO `json:"jobs"`
}

// CreateJobForGroup creates one job per group member with a shared dispatch_id.
// This enables parallel execution of the same job configuration across multiple agents.
func (s *JobService) CreateJobForGroup(groupID int, name, sourcePath, destPath string, schedule *string, syncFlags map[string]interface{}) (*GroupDispatchResponse, error) {
	// Validation
	if name == "" {
		return nil, fmt.Errorf("name is required")
	}
	if sourcePath == "" {
		return nil, fmt.Errorf("source_path is required")
	}
	if destPath == "" {
		return nil, fmt.Errorf("dest_path is required")
	}

	// Verify group exists
	group, err := s.db.GetGroup(groupID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch group: %w", err)
	}
	if group == nil {
		return nil, fmt.Errorf("group not found")
	}

	// Fetch group members
	agentIDs, err := s.db.GetGroupMembers(groupID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch group members: %w", err)
	}

	if len(agentIDs) == 0 {
		return nil, fmt.Errorf("group has no members")
	}

	// Generate shared dispatch_id for the batch
	dispatchID := "dispatch-" + newJobID()[4:] // reuse job ID generator, strip "job-" prefix

	// Create one job per group member
	jobs := []JobDTO{}
	createdAt := time.Now().UTC().Format(time.RFC3339)

	for _, agentID := range agentIDs {
		// Serialize sync_flags to JSON if present
		var syncFlagsJSON *string
		if syncFlags != nil {
			data, err := json.Marshal(syncFlags)
			if err != nil {
				return nil, fmt.Errorf("invalid sync_flags JSON: %w", err)
			}
			jsonStr := string(data)
			syncFlagsJSON = &jsonStr
		}

		jobID := newJobID()

		// Create the job in database (with group_id and dispatch_id tracked for federation events)
		if err := s.createGroupJob(jobID, agentID, name, sourcePath, destPath, schedule, syncFlagsJSON, groupID, dispatchID, createdAt); err != nil {
			return nil, fmt.Errorf("failed to create job: %w", err)
		}

		jobs = append(jobs, JobDTO{
			ID:         jobID,
			AgentID:    agentID,
			Name:       name,
			SourcePath: sourcePath,
			DestPath:   destPath,
			Schedule:   schedule,
			SyncFlags:  syncFlags,
			Status:     "pending",
			CreatedAt:  createdAt,
		})
	}

	return &GroupDispatchResponse{
		DispatchID: dispatchID,
		GroupID:    groupID,
		Jobs:       jobs,
	}, nil
}

// createGroupJob is a helper to create a job with group dispatch metadata.
func (s *JobService) createGroupJob(jobID, agentID, name, sourcePath, destPath string, schedule, syncFlags *string, groupID int, dispatchID, createdAt string) error {
	if err := s.db.CreateGroupJob(jobID, agentID, name, sourcePath, destPath, schedule, syncFlags, "pending", createdAt, groupID, dispatchID); err != nil {
		return fmt.Errorf("failed to create group job: %w", err)
	}
	return nil
}

// JobResultsDTO is the data returned when posting job results.
type JobResultsDTO struct {
	JobName string // For notification context
	AgentID string // For notification context
}

// PostJobResults stores job execution results (exit code, output, timestamps).
// Returns job metadata needed for notifications.
func (s *JobService) PostJobResults(jobID string, exitCode int, output, startedAt, finishedAt string) (*JobResultsDTO, error) {
	// Fetch job name and agent_id for notification context
	jobName, agentID, err := s.db.GetJobName(jobID)
	if err != nil {
		return nil, fmt.Errorf("job not found")
	}

	// Get the first job_run (created by trigger), or create one if it doesn't exist
	runID, err := s.db.GetFirstJobRun(jobID)
	if err != nil {
		return nil, fmt.Errorf("failed to query run: %w", err)
	}

	if runID == "" {
		// No run exists yet (trigger may not have created one), create a new one
		newRunID := "run-" + newJobID()[4:] // reuse ID generator, strip "job-" prefix
		if err := s.db.CreateJobRun(newRunID, jobID, exitCode, output, startedAt, finishedAt); err != nil {
			return nil, fmt.Errorf("failed to store result: %w", err)
		}
	} else {
		// Run exists (created by trigger), update it with the result
		if err := s.db.UpdateJobRun(runID, exitCode, output, startedAt, finishedAt); err != nil {
			return nil, fmt.Errorf("failed to update result: %w", err)
		}
	}

	return &JobResultsDTO{
		JobName: jobName,
		AgentID: agentID,
	}, nil
}

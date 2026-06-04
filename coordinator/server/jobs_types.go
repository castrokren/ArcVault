package server

import "fmt"

// CreateJobRequest defines the request to create a new job
type CreateJobRequest struct {
	Name       string `json:"name"`
	AgentID    *string `json:"agent_id"`
	GroupID    *int `json:"group_id"`
	SourcePath string `json:"source_path"`
	DestPath   string `json:"dest_path"`
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
	GroupID    *int `json:"group_id"`
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
	Percentage int    `json:"percentage"`
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
	RunID    string `json:"run_id"`
	ExitCode int    `json:"exit_code"`
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
	RunID    string `json:"run_id"`
	JobID    string `json:"job_id"`
	Status   string `json:"status"`
	ExitCode int    `json:"exit_code"`
	Error    *string `json:"error"`
}

// JobRunResponse defines a single job run (execution)
type JobRunResponse struct {
	RunID    string `json:"run_id"`
	JobID    string `json:"job_id"`
	RunStart string `json:"run_start"`
	RunEnd   *string `json:"run_end"`
	Status   string `json:"status"`
	ExitCode *int `json:"exit_code"`
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
	JobID       string         `json:"job_id"`
	Percentage  int            `json:"percentage"`
	Status      string         `json:"status"`
	RecentLogs  []JobLogEntry  `json:"recent_logs"`
	UpdatedAt   string         `json:"updated_at"`
}

// PaginatedJobLogsResponse wraps paginated job logs
type PaginatedJobLogsResponse struct {
	Data       []JobLogEntry  `json:"data"`
	Pagination PaginationMeta `json:"pagination"`
}

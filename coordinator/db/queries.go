package db

// AgentQueries defines all agent-related database operations.
type AgentQueries interface {
	// RegisterAgent inserts or updates an agent registration.
	RegisterAgent(agentID, hostname, os, arch, version, coordinatorID string) error

	// UpdateAgentHeartbeat updates agent status, last_seen, and rollback_available.
	UpdateAgentHeartbeat(agentID, coordinatorID string, rollbackAvailable bool) error

	// GetAgent returns a single agent by ID, or sql.ErrNoRows if not found.
	GetAgent(agentID string) (Agent, error)

	// ListAgents returns agents with optional search and status filters, with pagination.
	ListAgents(search, status string, limit, offset int) ([]Agent, int, error)

	// CountRunningJobs returns the number of jobs with status='running' for the given agent.
	CountRunningJobs(agentID string) (int, error)

	// DeleteAgent removes an agent and its tokens + group memberships.
	DeleteAgent(agentID string) error

	// DeleteAgentTokens removes all tokens for an agent.
	DeleteAgentTokens(agentID string) error

	// DeleteAgentGroupMemberships removes all group memberships for an agent.
	DeleteAgentGroupMemberships(agentID string) error
}

// JobQueries defines all job-related database operations.
type JobQueries interface {
	// CreateJob inserts a new job.
	CreateJob(jobID, agentID, name, sourcePath, destPath string, schedule, syncFlags *string, status, createdAt string) error

	// CreateGroupJob inserts a new job with group dispatch metadata (group_id, dispatch_id).
	CreateGroupJob(jobID, agentID, name, sourcePath, destPath string, schedule, syncFlags *string, status, createdAt string, groupID int, dispatchID string) error

	// GetJob returns a single job by ID, or sql.ErrNoRows if not found.
	GetJob(jobID string) (Job, error)

	// ListJobs returns jobs with optional search and status filters, with pagination.
	ListJobs(search, status, agentID string, limit, offset int) ([]Job, int, error)

	// UpdateJobStatus updates the status of a job.
	UpdateJobStatus(jobID, status string) error

	// DeleteJob removes a job by ID.
	DeleteJob(jobID string) error

	// JobExists checks if a job exists.
	JobExists(jobID string) (bool, error)

	// GetJobName returns the name and agent_id of a job.
	GetJobName(jobID string) (name, agentID string, err error)

	// CreateTemplateJob inserts a transient job fired by a backup template.
	// The job carries the template command and is created with status "pending".
	CreateTemplateJob(runID, agentID, name, command, createdAt string) error
}

// GroupQueries defines group-related database operations (minimal set for job dispatch).
type GroupQueries interface {
	// GetGroup returns a group by ID, or nil if not found.
	GetGroup(id int) (*AgentGroup, error)

	// GetGroupMembers returns all agent IDs in a group.
	GetGroupMembers(groupID int) ([]string, error)
}

// JobRunQueries defines job run history tracking.
type JobRunQueries interface {
	// ListJobRuns returns job runs for a specific job with pagination.
	ListJobRuns(jobID string, limit, offset int) ([]JobRun, int, error)

	// CountJobRuns returns the total number of runs for a job.
	CountJobRuns(jobID string) (int, error)

	// ListAllJobRuns returns job runs with filters and pagination (for Reports).
	ListAllJobRuns(filters map[string]string, limit, offset int) ([]JobRun, int, error)

	// GetFirstJobRun returns the ID of the first job run for a job (created by trigger),
	// or empty string if no run exists.
	GetFirstJobRun(jobID string) (runID string, err error)

	// CreateJobRun inserts a new job run with initial result data.
	CreateJobRun(id, jobID string, exitCode int, output, startedAt, finishedAt string) error

	// UpdateJobRun updates an existing job run with result data.
	UpdateJobRun(id string, exitCode int, output, startedAt, finishedAt string) error
}

// UserQueries defines all user-related database operations.
type UserQueries interface {
	// CreateUser inserts a new user with the given credentials and role.
	CreateUser(username, passwordHash, role string, mustChange bool) error

	// GetUserByUsername returns a user by username, or nil if not found.
	GetUserByUsername(username string) (*User, error)

	// GetUserByID returns a user by ID, or nil if not found.
	GetUserByID(id int) (*User, error)

	// CountUsers returns the total number of users.
	CountUsers() (int, error)

	// UpdatePassword updates a user's password hash and must_change_password flag.
	UpdatePassword(userID int, newHash string, mustChange bool) error

	// ListUsers returns all users.
	ListUsers() ([]User, error)

	// DeleteUser removes a user by ID.
	DeleteUser(userID int) error

	// UpdateUserRole updates a user's role.
	UpdateUserRole(userID int, role string) error
}

// Extended GroupQueries with full CRUD operations.
// Note: GroupQueries already has GetGroup, GetGroupMembers from earlier extension.
// Extending here with additional operations.
type ExtendedGroupQueries interface {
	// GetGroup returns a group by ID, or nil if not found.
	GetGroup(id int) (*AgentGroup, error)

	// GetGroupMembers returns all agent IDs in a group.
	GetGroupMembers(groupID int) ([]string, error)

	// CreateGroup creates a new agent group.
	CreateGroup(name, description string) (*AgentGroup, error)

	// ListGroups returns all agent groups.
	ListGroups() ([]AgentGroup, error)

	// UpdateGroup updates a group's name and description.
	UpdateGroup(id int, name, description string) error

	// DeleteGroup removes a group by ID.
	DeleteGroup(id int) error

	// AddAgentToGroup adds an agent to a group.
	AddAgentToGroup(groupID int, agentID string) error

	// RemoveAgentFromGroup removes an agent from a group.
	RemoveAgentFromGroup(groupID int, agentID string) error
}

// AllQueries is a union interface that includes all query types.
// Used by services that need to access multiple query interfaces.
type AllQueries interface {
	JobQueries
	JobRunQueries
	UserQueries
	ExtendedGroupQueries
}

// Agent represents an agent in the database.
type Agent struct {
	ID                string
	Hostname          string
	OS                string
	Arch              string
	Version           string
	Status            string
	LastSeen          *string
	RegisteredAt      string
	RollbackAvailable bool
}

// Job represents a job in the database.
type Job struct {
	ID         string
	AgentID    string
	Name       string
	SourcePath string
	DestPath   string
	Schedule   *string
	SyncFlags  *string
	Status     string
	CreatedAt  string
}

// JobRun represents a single execution of a job.
type JobRun struct {
	ID         string
	JobID      string
	StartedAt  string
	FinishedAt *string
	Status     string
	ExitCode   *int
	Output     *string
}

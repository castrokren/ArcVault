package business

import (
	"database/sql"
	"fmt"
	"time"

	"arcvault/coordinator/db"
)

// ----------------------------------------------------------------------------
// mockAgentQueries implements db.AgentQueries for AgentService tests
// ----------------------------------------------------------------------------

type mockAgentQueries struct {
	agents         map[string]db.Agent
	runningJobsMap map[string]int
	registerErr    error
	getErr         error
	listErr        error
	deleteErr      error
}

func newMockAgentQueries() *mockAgentQueries {
	return &mockAgentQueries{
		agents:         map[string]db.Agent{},
		runningJobsMap: map[string]int{},
	}
}

func (m *mockAgentQueries) RegisterAgent(agentID, hostname, os, arch, version, coordinatorID string) error {
	if m.registerErr != nil {
		return m.registerErr
	}
	now := time.Now().UTC().Format(time.RFC3339)
	m.agents[agentID] = db.Agent{
		ID:           agentID,
		Hostname:     hostname,
		OS:           os,
		Arch:         arch,
		Version:      version,
		Status:       "online",
		RegisteredAt: now,
	}
	return nil
}

func (m *mockAgentQueries) GetAgent(agentID string) (db.Agent, error) {
	if m.getErr != nil {
		return db.Agent{}, m.getErr
	}
	a, ok := m.agents[agentID]
	if !ok {
		return db.Agent{}, sql.ErrNoRows
	}
	return a, nil
}

func (m *mockAgentQueries) UpdateAgentHeartbeat(agentID, coordinatorID string, rollbackAvailable bool) error {
	return nil
}

func (m *mockAgentQueries) ListAgents(search, status string, limit, offset int) ([]db.Agent, int, error) {
	if m.listErr != nil {
		return nil, 0, m.listErr
	}
	var all []db.Agent
	for _, a := range m.agents {
		all = append(all, a)
	}
	total := len(all)
	if offset >= total {
		return []db.Agent{}, total, nil
	}
	end := offset + limit
	if end > total {
		end = total
	}
	return all[offset:end], total, nil
}

func (m *mockAgentQueries) CountRunningJobs(agentID string) (int, error) {
	return m.runningJobsMap[agentID], nil
}

func (m *mockAgentQueries) DeleteAgent(agentID string) error {
	if m.deleteErr != nil {
		return m.deleteErr
	}
	delete(m.agents, agentID)
	return nil
}

func (m *mockAgentQueries) DeleteAgentTokens(agentID string) error           { return nil }
func (m *mockAgentQueries) DeleteAgentGroupMemberships(agentID string) error { return nil }

// ----------------------------------------------------------------------------
// mockJobDB implements db.AllQueries for JobService tests
// ----------------------------------------------------------------------------

type mockJobDB struct {
	jobs    map[string]db.Job
	runs    map[string][]db.JobRun // keyed by jobID
	groups  map[int]*db.AgentGroup
	members map[int][]string

	// per-call error overrides
	createJobErr   error
	getJobErr      error
	listJobsErr    error
	getJobNameErr  error
	getFirstRunErr error
	firstRunID     string // returned by GetFirstJobRun
	createRunErr   error
	updateRunErr   error
}

func newMockJobDB() *mockJobDB {
	return &mockJobDB{
		jobs:    map[string]db.Job{},
		runs:    map[string][]db.JobRun{},
		groups:  map[int]*db.AgentGroup{},
		members: map[int][]string{},
	}
}

// -- JobQueries --

func (m *mockJobDB) CreateJob(jobID, agentID, name, sourcePath, destPath string, schedule, syncFlags *string, status, createdAt string) error {
	if m.createJobErr != nil {
		return m.createJobErr
	}
	m.jobs[jobID] = db.Job{
		ID:         jobID,
		AgentID:    agentID,
		Name:       name,
		SourcePath: sourcePath,
		DestPath:   destPath,
		Schedule:   schedule,
		SyncFlags:  syncFlags,
		Status:     status,
		CreatedAt:  createdAt,
	}
	return nil
}

func (m *mockJobDB) CreateGroupJob(jobID, agentID, name, sourcePath, destPath string, schedule, syncFlags *string, status, createdAt string, groupID int, dispatchID string) error {
	return m.CreateJob(jobID, agentID, name, sourcePath, destPath, schedule, syncFlags, status, createdAt)
}

func (m *mockJobDB) GetJob(jobID string) (db.Job, error) {
	if m.getJobErr != nil {
		return db.Job{}, m.getJobErr
	}
	j, ok := m.jobs[jobID]
	if !ok {
		return db.Job{}, sql.ErrNoRows
	}
	return j, nil
}

func (m *mockJobDB) ListJobs(search, status, agentID string, limit, offset int) ([]db.Job, int, error) {
	if m.listJobsErr != nil {
		return nil, 0, m.listJobsErr
	}
	var all []db.Job
	for _, j := range m.jobs {
		all = append(all, j)
	}
	total := len(all)
	return all, total, nil
}

func (m *mockJobDB) UpdateJobStatus(jobID, status string) error {
	j, ok := m.jobs[jobID]
	if !ok {
		return nil
	}
	j.Status = status
	m.jobs[jobID] = j
	return nil
}

func (m *mockJobDB) DeleteJob(jobID string) error {
	delete(m.jobs, jobID)
	return nil
}

func (m *mockJobDB) JobExists(jobID string) (bool, error) {
	_, ok := m.jobs[jobID]
	return ok, nil
}

func (m *mockJobDB) GetJobName(jobID string) (string, string, error) {
	if m.getJobNameErr != nil {
		return "", "", m.getJobNameErr
	}
	j, ok := m.jobs[jobID]
	if !ok {
		return "", "", fmt.Errorf("job not found")
	}
	return j.Name, j.AgentID, nil
}

func (m *mockJobDB) CreateTemplateJob(runID, agentID, name, command, createdAt string) error {
	return nil
}

// -- JobRunQueries --

func (m *mockJobDB) ListJobRuns(jobID string, limit, offset int) ([]db.JobRun, int, error) {
	runs := m.runs[jobID]
	return runs, len(runs), nil
}

func (m *mockJobDB) CountJobRuns(jobID string) (int, error) {
	return len(m.runs[jobID]), nil
}

func (m *mockJobDB) ListAllJobRuns(filters map[string]string, limit, offset int) ([]db.JobRun, int, error) {
	return nil, 0, nil
}

func (m *mockJobDB) GetFirstJobRun(jobID string) (string, error) {
	if m.getFirstRunErr != nil {
		return "", m.getFirstRunErr
	}
	return m.firstRunID, nil
}

func (m *mockJobDB) CreateJobRun(id, jobID string, exitCode int, output, startedAt, finishedAt string) error {
	if m.createRunErr != nil {
		return m.createRunErr
	}
	ec := exitCode
	out := output
	m.runs[jobID] = append(m.runs[jobID], db.JobRun{
		ID:       id,
		JobID:    jobID,
		ExitCode: &ec,
		Output:   &out,
	})
	return nil
}

func (m *mockJobDB) UpdateJobRun(id string, exitCode int, output, startedAt, finishedAt string) error {
	if m.updateRunErr != nil {
		return m.updateRunErr
	}
	return nil
}

// -- UserQueries (stub — JobService doesn't use these) --

func (m *mockJobDB) CreateUser(username, passwordHash, role string, mustChange bool) error {
	return nil
}
func (m *mockJobDB) GetUserByUsername(username string) (*db.User, error)              { return nil, nil }
func (m *mockJobDB) GetUserByID(id int) (*db.User, error)                             { return nil, nil }
func (m *mockJobDB) CountUsers() (int, error)                                         { return 0, nil }
func (m *mockJobDB) UpdatePassword(userID int, newHash string, mustChange bool) error { return nil }
func (m *mockJobDB) ListUsers() ([]db.User, error)                                    { return nil, nil }
func (m *mockJobDB) DeleteUser(userID int) error                                      { return nil }
func (m *mockJobDB) UpdateUserRole(userID int, role string) error                     { return nil }

// -- ExtendedGroupQueries --

func (m *mockJobDB) GetGroup(id int) (*db.AgentGroup, error) {
	g, ok := m.groups[id]
	if !ok {
		return nil, nil
	}
	return g, nil
}

func (m *mockJobDB) GetGroupMembers(groupID int) ([]string, error) {
	return m.members[groupID], nil
}

func (m *mockJobDB) CreateGroup(name, description string) (*db.AgentGroup, error) { return nil, nil }
func (m *mockJobDB) ListGroups() ([]db.AgentGroup, error)                         { return nil, nil }
func (m *mockJobDB) UpdateGroup(id int, name, description string) error           { return nil }
func (m *mockJobDB) DeleteGroup(id int) error                                     { return nil }
func (m *mockJobDB) AddAgentToGroup(groupID int, agentID string) error            { return nil }
func (m *mockJobDB) RemoveAgentFromGroup(groupID int, agentID string) error       { return nil }

// -- CredentialProfileQueries (stub) --

func (m *mockJobDB) CreateCredentialProfile(id, name, credType string, encryptedData []byte) error {
	return nil
}
func (m *mockJobDB) GetCredentialProfile(id string) (*db.CredentialProfile, error)     { return nil, nil }
func (m *mockJobDB) ListCredentialProfiles() ([]*db.CredentialProfile, error)          { return nil, nil }
func (m *mockJobDB) DeleteCredentialProfile(id string) error                           { return nil }
func (m *mockJobDB) HasJobsReferencingProfile(profileID string) (bool, error)          { return false, nil }
func (m *mockJobDB) GetJobCredentialProfileID(jobID string) (string, error)            { return "", nil }
func (m *mockJobDB) SnapshotJobRunCredentials(runID, credentialProfileID, credentialProfileName string) error {
	return nil
}

func (m *mockJobDB) InsertUserAuditLog(ctx db.UserAuditLogContext) error { return nil }
func (m *mockJobDB) ListUserAuditLogs(filter db.UserAuditLogFilter) ([]db.UserAuditLogEntry, int, error) {
	return nil, 0, nil
}

// ----------------------------------------------------------------------------
// mockUserQueries implements db.UserQueries for UserService tests
// ----------------------------------------------------------------------------

type mockUserQueries struct {
	users     map[string]*db.User
	nextID    int
	createErr error
}

func newMockUserQueries() *mockUserQueries {
	return &mockUserQueries{
		users:  map[string]*db.User{},
		nextID: 1,
	}
}

func (m *mockUserQueries) CreateUser(username, passwordHash, role string, mustChange bool) error {
	if m.createErr != nil {
		return m.createErr
	}
	id := m.nextID
	m.nextID++
	m.users[username] = &db.User{
		ID:                 id,
		Username:           username,
		PasswordHash:       passwordHash,
		Role:               role,
		MustChangePassword: mustChange,
		CreatedAt:          time.Now(),
	}
	return nil
}

func (m *mockUserQueries) GetUserByUsername(username string) (*db.User, error) {
	u, ok := m.users[username]
	if !ok {
		return nil, nil
	}
	return u, nil
}

func (m *mockUserQueries) GetUserByID(id int) (*db.User, error) {
	for _, u := range m.users {
		if u.ID == id {
			return u, nil
		}
	}
	return nil, nil
}

func (m *mockUserQueries) CountUsers() (int, error) { return len(m.users), nil }

func (m *mockUserQueries) UpdatePassword(userID int, newHash string, mustChange bool) error {
	for _, u := range m.users {
		if u.ID == userID {
			u.PasswordHash = newHash
			u.MustChangePassword = mustChange
			return nil
		}
	}
	return nil
}

func (m *mockUserQueries) ListUsers() ([]db.User, error) {
	var all []db.User
	for _, u := range m.users {
		all = append(all, *u)
	}
	return all, nil
}

func (m *mockUserQueries) DeleteUser(userID int) error {
	for k, u := range m.users {
		if u.ID == userID {
			delete(m.users, k)
			return nil
		}
	}
	return nil
}

func (m *mockUserQueries) UpdateUserRole(userID int, role string) error {
	for _, u := range m.users {
		if u.ID == userID {
			u.Role = role
			return nil
		}
	}
	return nil
}

// ----------------------------------------------------------------------------
// mockGroupQueries implements db.ExtendedGroupQueries for GroupService tests
// ----------------------------------------------------------------------------

type mockGroupQueries struct {
	groups  map[int]*db.AgentGroup
	members map[int][]string
	nextID  int
}

func newMockGroupQueries() *mockGroupQueries {
	return &mockGroupQueries{
		groups:  map[int]*db.AgentGroup{},
		members: map[int][]string{},
		nextID:  1,
	}
}

func (m *mockGroupQueries) GetGroup(id int) (*db.AgentGroup, error) {
	g, ok := m.groups[id]
	if !ok {
		return nil, nil
	}
	return g, nil
}

func (m *mockGroupQueries) GetGroupMembers(groupID int) ([]string, error) {
	return m.members[groupID], nil
}

func (m *mockGroupQueries) CreateGroup(name, description string) (*db.AgentGroup, error) {
	id := m.nextID
	m.nextID++
	g := &db.AgentGroup{ID: id, Name: name, Description: description, CreatedAt: time.Now()}
	m.groups[id] = g
	return g, nil
}

func (m *mockGroupQueries) ListGroups() ([]db.AgentGroup, error) {
	var all []db.AgentGroup
	for _, g := range m.groups {
		all = append(all, *g)
	}
	return all, nil
}

func (m *mockGroupQueries) UpdateGroup(id int, name, description string) error {
	g, ok := m.groups[id]
	if !ok {
		return fmt.Errorf("not found")
	}
	g.Name = name
	g.Description = description
	return nil
}

func (m *mockGroupQueries) DeleteGroup(id int) error {
	delete(m.groups, id)
	return nil
}

func (m *mockGroupQueries) AddAgentToGroup(groupID int, agentID string) error {
	m.members[groupID] = append(m.members[groupID], agentID)
	return nil
}

func (m *mockGroupQueries) RemoveAgentFromGroup(groupID int, agentID string) error {
	var filtered []string
	for _, id := range m.members[groupID] {
		if id != agentID {
			filtered = append(filtered, id)
		}
	}
	m.members[groupID] = filtered
	return nil
}

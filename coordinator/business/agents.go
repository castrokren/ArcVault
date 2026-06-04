package business

import (
	"fmt"

	"arcvault/coordinator/db"
)

// AgentService handles agent-related business logic.
type AgentService struct {
	queries db.AgentQueries
}

// NewAgentService creates a new agent service.
func NewAgentService(queries db.AgentQueries) *AgentService {
	return &AgentService{
		queries: queries,
	}
}

// AgentDTO is the data transfer object for agents (API response).
type AgentDTO struct {
	ID                string  `json:"id"`
	Hostname          string  `json:"hostname"`
	OS                string  `json:"os"`
	Arch              string  `json:"arch"`
	Version           string  `json:"version"`
	Status            string  `json:"status"`
	LastSeen          *string `json:"last_seen"`
	RegisteredAt      string  `json:"registered_at"`
	RollbackAvailable bool    `json:"rollback_available"`
}

// RegisterAgent registers or updates an agent.
// Returns the agent DTO.
func (s *AgentService) RegisterAgent(agentID, hostname, os, arch, version, coordinatorID string) (*AgentDTO, error) {
	if agentID == "" || hostname == "" || os == "" || version == "" {
		return nil, fmt.Errorf("agent_id, hostname, os, and version are required")
	}

	if err := s.queries.RegisterAgent(agentID, hostname, os, arch, version, coordinatorID); err != nil {
		return nil, fmt.Errorf("failed to register agent: %w", err)
	}

	// Fetch and return the registered agent
	agent, err := s.queries.GetAgent(agentID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch registered agent: %w", err)
	}

	return &AgentDTO{
		ID:                agent.ID,
		Hostname:          agent.Hostname,
		OS:                agent.OS,
		Arch:              agent.Arch,
		Version:           agent.Version,
		Status:            agent.Status,
		LastSeen:          agent.LastSeen,
		RegisteredAt:      agent.RegisteredAt,
		RollbackAvailable: agent.RollbackAvailable,
	}, nil
}

// UpdateAgentHeartbeat updates an agent's heartbeat.
func (s *AgentService) UpdateAgentHeartbeat(agentID, coordinatorID string, rollbackAvailable bool) error {
	if err := s.queries.UpdateAgentHeartbeat(agentID, coordinatorID, rollbackAvailable); err != nil {
		return fmt.Errorf("failed to update heartbeat: %w", err)
	}
	return nil
}

// ListAgentsDTO returns agents with pagination.
type ListAgentsDTO struct {
	Agents []AgentDTO `json:"data"`
	Total  int        `json:"total"`
	Page   int        `json:"page"`
	Pages  int        `json:"pages"`
	Limit  int        `json:"limit"`
}

// ListAgents returns agents with optional filters and pagination.
func (s *AgentService) ListAgents(search, status string, limit, offset int) (*ListAgentsDTO, error) {
	agents, total, err := s.queries.ListAgents(search, status, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to list agents: %w", err)
	}

	dtos := make([]AgentDTO, len(agents))
	for i, agent := range agents {
		dtos[i] = AgentDTO{
			ID:                agent.ID,
			Hostname:          agent.Hostname,
			OS:                agent.OS,
			Arch:              agent.Arch,
			Version:           agent.Version,
			Status:            agent.Status,
			LastSeen:          agent.LastSeen,
			RegisteredAt:      agent.RegisteredAt,
			RollbackAvailable: agent.RollbackAvailable,
		}
	}

	pages := (total + limit - 1) / limit
	if pages == 0 {
		pages = 1
	}

	return &ListAgentsDTO{
		Agents: dtos,
		Total:  total,
		Page:   (offset / limit) + 1,
		Pages:  pages,
		Limit:  limit,
	}, nil
}

// DeleteAgent deletes an agent, but only if it has no running jobs.
func (s *AgentService) DeleteAgent(agentID string) error {
	// Check if agent exists
	_, err := s.queries.GetAgent(agentID)
	if err != nil {
		return fmt.Errorf("agent not found")
	}

	// Block deletion if there are running jobs
	running, err := s.queries.CountRunningJobs(agentID)
	if err != nil {
		return fmt.Errorf("failed to check running jobs: %w", err)
	}
	if running > 0 {
		return fmt.Errorf("agent has %d running jobs — stop all jobs before deleting", running)
	}

	// Delete the agent (cascades to tokens and group memberships)
	if err := s.queries.DeleteAgent(agentID); err != nil {
		return fmt.Errorf("failed to delete agent: %w", err)
	}

	return nil
}

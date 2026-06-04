package business

import (
	"fmt"
	"time"

	"arcvault/coordinator/db"
)

// GroupService handles group-related business logic.
type GroupService struct {
	db db.ExtendedGroupQueries
}

// NewGroupService creates a new group service.
func NewGroupService(database db.ExtendedGroupQueries) *GroupService {
	return &GroupService{
		db: database,
	}
}

// GroupDTO is the data transfer object for groups (API response).
type GroupDTO struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	AgentCount  int    `json:"agent_count"`
	CreatedAt   string `json:"created_at"`
}

// CreateGroupInput validates and holds group creation data.
type CreateGroupInput struct {
	Name        string
	Description string
}

// ValidateCreateGroup validates group creation input.
func (input *CreateGroupInput) Validate() error {
	if input.Name == "" {
		return fmt.Errorf("name is required")
	}
	return nil
}

// CreateGroup creates a new agent group.
func (s *GroupService) CreateGroup(input *CreateGroupInput) (*GroupDTO, error) {
	if err := input.Validate(); err != nil {
		return nil, err
	}

	agentGroup, err := s.db.CreateGroup(input.Name, input.Description)
	if err != nil {
		return nil, fmt.Errorf("failed to create group: %w", err)
	}

	return &GroupDTO{
		ID:          agentGroup.ID,
		Name:        agentGroup.Name,
		Description: agentGroup.Description,
		AgentCount:  0,
		CreatedAt:   agentGroup.CreatedAt.Format(time.RFC3339),
	}, nil
}

// ListGroups returns all groups with agent counts.
func (s *GroupService) ListGroups() ([]GroupDTO, error) {
	groups, err := s.db.ListGroups()
	if err != nil {
		return nil, fmt.Errorf("failed to list groups: %w", err)
	}

	dtos := make([]GroupDTO, len(groups))
	for i, g := range groups {
		members, _ := s.db.GetGroupMembers(g.ID)
		dtos[i] = GroupDTO{
			ID:          g.ID,
			Name:        g.Name,
			Description: g.Description,
			AgentCount:  len(members),
			CreatedAt:   g.CreatedAt.Format(time.RFC3339),
		}
	}

	return dtos, nil
}

// GetGroup retrieves a single group with agent count.
func (s *GroupService) GetGroup(groupID int) (*GroupDTO, error) {
	group, err := s.db.GetGroup(groupID)
	if err != nil {
		return nil, fmt.Errorf("failed to get group: %w", err)
	}
	if group == nil {
		return nil, fmt.Errorf("group not found")
	}

	members, _ := s.db.GetGroupMembers(groupID)

	return &GroupDTO{
		ID:          group.ID,
		Name:        group.Name,
		Description: group.Description,
		AgentCount:  len(members),
		CreatedAt:   group.CreatedAt.Format(time.RFC3339),
	}, nil
}

// UpdateGroupInput holds group update data.
type UpdateGroupInput struct {
	Name        string
	Description string
}

// ValidateUpdateGroup validates group update input.
func (input *UpdateGroupInput) Validate() error {
	if input.Name == "" {
		return fmt.Errorf("name is required")
	}
	return nil
}

// UpdateGroup updates a group's name and description.
func (s *GroupService) UpdateGroup(groupID int, input *UpdateGroupInput) (*GroupDTO, error) {
	if err := input.Validate(); err != nil {
		return nil, err
	}

	// Verify group exists
	group, err := s.db.GetGroup(groupID)
	if err != nil {
		return nil, fmt.Errorf("failed to get group: %w", err)
	}
	if group == nil {
		return nil, fmt.Errorf("group not found")
	}

	if err := s.db.UpdateGroup(groupID, input.Name, input.Description); err != nil {
		return nil, fmt.Errorf("failed to update group: %w", err)
	}

	// Fetch and return updated group
	return s.GetGroup(groupID)
}

// DeleteGroup removes a group by ID.
func (s *GroupService) DeleteGroup(groupID int) error {
	// Verify group exists
	group, err := s.db.GetGroup(groupID)
	if err != nil {
		return fmt.Errorf("failed to get group: %w", err)
	}
	if group == nil {
		return fmt.Errorf("group not found")
	}

	if err := s.db.DeleteGroup(groupID); err != nil {
		return fmt.Errorf("failed to delete group: %w", err)
	}

	return nil
}

// AddAgentToGroup adds an agent to a group.
func (s *GroupService) AddAgentToGroup(groupID int, agentID string) error {
	// Verify group exists
	group, err := s.db.GetGroup(groupID)
	if err != nil {
		return fmt.Errorf("failed to get group: %w", err)
	}
	if group == nil {
		return fmt.Errorf("group not found")
	}

	if err := s.db.AddAgentToGroup(groupID, agentID); err != nil {
		return fmt.Errorf("failed to add agent to group: %w", err)
	}

	return nil
}

// RemoveAgentFromGroup removes an agent from a group.
func (s *GroupService) RemoveAgentFromGroup(groupID int, agentID string) error {
	// Verify group exists
	group, err := s.db.GetGroup(groupID)
	if err != nil {
		return fmt.Errorf("failed to get group: %w", err)
	}
	if group == nil {
		return fmt.Errorf("group not found")
	}

	if err := s.db.RemoveAgentFromGroup(groupID, agentID); err != nil {
		return fmt.Errorf("failed to remove agent from group: %w", err)
	}

	return nil
}

// GetGroupAgents returns all agents in a group.
func (s *GroupService) GetGroupAgents(groupID int) ([]string, error) {
	// Verify group exists
	group, err := s.db.GetGroup(groupID)
	if err != nil {
		return nil, fmt.Errorf("failed to get group: %w", err)
	}
	if group == nil {
		return nil, fmt.Errorf("group not found")
	}

	agents, err := s.db.GetGroupMembers(groupID)
	if err != nil {
		return nil, fmt.Errorf("failed to get group members: %w", err)
	}

	return agents, nil
}

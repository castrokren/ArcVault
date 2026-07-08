package server

import "fmt"

// CreateGroupRequest defines the request to create a new group
type CreateGroupRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

// Validate checks if CreateGroupRequest is valid
func (r *CreateGroupRequest) Validate() error {
	if r.Name == "" {
		return fmt.Errorf("name is required")
	}
	if len(r.Name) < 1 || len(r.Name) > 255 {
		return fmt.Errorf("name must be 1-255 characters")
	}
	if len(r.Description) > 1000 {
		return fmt.Errorf("description must be 0-1000 characters")
	}
	return nil
}

// GroupResponse defines the group response with agent count
type GroupResponse struct {
	GroupID     int    `json:"group_id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	AgentCount  int    `json:"agent_count"`
	CreatedAt   string `json:"created_at"`
}

// PaginatedGroupsResponse wraps paginated groups list
type PaginatedGroupsResponse struct {
	Data       []GroupResponse `json:"data"`
	Pagination PaginationMeta  `json:"pagination"`
}

// UpdateGroupRequest defines the request to update a group
type UpdateGroupRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

// Validate checks if UpdateGroupRequest is valid
func (r *UpdateGroupRequest) Validate() error {
	if r.Name != "" {
		if len(r.Name) < 1 || len(r.Name) > 255 {
			return fmt.Errorf("name must be 1-255 characters if provided")
		}
	}
	if len(r.Description) > 1000 {
		return fmt.Errorf("description must be 0-1000 characters")
	}
	return nil
}

// UpdateGroupResponse defines the response after updating a group
type UpdateGroupResponse struct {
	GroupID     int    `json:"group_id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	AgentCount  int    `json:"agent_count"`
	UpdatedAt   string `json:"updated_at"`
}

package server

import "fmt"

// CreateUserRequest defines the request to create a new user
type CreateUserRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Role     string `json:"role"`
}

// Validate checks if CreateUserRequest is valid
func (r *CreateUserRequest) Validate() error {
	if r.Username == "" {
		return fmt.Errorf("username is required")
	}
	if len(r.Username) < 1 || len(r.Username) > 255 {
		return fmt.Errorf("username must be 1-255 characters")
	}
	if r.Password == "" {
		return fmt.Errorf("password is required")
	}
	if len(r.Password) < 8 {
		return fmt.Errorf("password must be 8 or more characters")
	}
	if r.Role == "" {
		return fmt.Errorf("role is required")
	}
	if r.Role != "admin" && r.Role != "viewer" {
		return fmt.Errorf("role must be 'admin' or 'viewer'")
	}
	return nil
}

// UserResponse defines the user response (no password hash exposed)
type UserResponse struct {
	UserID            int    `json:"user_id"`
	Username          string `json:"username"`
	Role              string `json:"role"`
	MustChangePassword bool   `json:"must_change_password"`
	CreatedAt         string `json:"created_at"`
}

// PaginatedUsersResponse wraps paginated users list
type PaginatedUsersResponse struct {
	Data       []UserResponse `json:"data"`
	Pagination PaginationMeta `json:"pagination"`
}

// UpdateUserRoleRequest defines the request to update a user's role
type UpdateUserRoleRequest struct {
	Role string `json:"role"`
}

// Validate checks if UpdateUserRoleRequest is valid
func (r *UpdateUserRoleRequest) Validate() error {
	if r.Role == "" {
		return fmt.Errorf("role is required")
	}
	if r.Role != "admin" && r.Role != "viewer" {
		return fmt.Errorf("role must be 'admin' or 'viewer'")
	}
	return nil
}

// UpdateUserRoleResponse defines the response after updating user role
type UpdateUserRoleResponse struct {
	UserID    int    `json:"user_id"`
	Username  string `json:"username"`
	Role      string `json:"role"`
	UpdatedAt string `json:"updated_at"`
}

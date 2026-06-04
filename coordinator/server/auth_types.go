package server

import "fmt"

// LoginRequest defines the shape of a login request body
type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// Validate checks if LoginRequest is valid
func (r *LoginRequest) Validate() error {
	if r.Username == "" {
		return fmt.Errorf("username is required")
	}
	if r.Password == "" {
		return fmt.Errorf("password is required")
	}
	if len(r.Password) < 8 {
		return fmt.Errorf("password must be 8 or more characters")
	}
	return nil
}

// ChangePasswordRequest defines password change request
type ChangePasswordRequest struct {
	OldPassword string `json:"old_password"`
	NewPassword string `json:"new_password"`
}

// Validate checks if ChangePasswordRequest is valid
func (r *ChangePasswordRequest) Validate() error {
	if r.OldPassword == "" {
		return fmt.Errorf("old_password is required")
	}
	if r.NewPassword == "" {
		return fmt.Errorf("new_password is required")
	}
	if len(r.OldPassword) < 8 {
		return fmt.Errorf("old_password must be 8 or more characters")
	}
	if len(r.NewPassword) < 8 {
		return fmt.Errorf("new_password must be 8 or more characters")
	}
	if r.OldPassword == r.NewPassword {
		return fmt.Errorf("new_password must be different from old_password")
	}
	return nil
}

// ChangePasswordResponse defines the response after password change
type ChangePasswordResponse struct {
	Message string `json:"message"`
}

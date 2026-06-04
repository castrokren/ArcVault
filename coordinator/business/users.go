package business

import (
	"fmt"
	"time"

	"arcvault/coordinator/db"
	"golang.org/x/crypto/bcrypt"
)

// UserService handles user-related business logic.
type UserService struct {
	db db.UserQueries
}

// NewUserService creates a new user service.
func NewUserService(database db.UserQueries) *UserService {
	return &UserService{
		db: database,
	}
}

// UserDTO is the data transfer object for users (API response).
// Note: PasswordHash is never included in responses.
type UserDTO struct {
	ID                 int       `json:"id"`
	Username           string    `json:"username"`
	Role               string    `json:"role"`
	MustChangePassword bool      `json:"must_change_password"`
	CreatedAt          string    `json:"created_at"`
}

// CreateUserInput validates and holds user creation data.
type CreateUserInput struct {
	Username string
	Password string
	Role     string
}

// ValidateCreateUser validates user creation input.
func (input *CreateUserInput) Validate() error {
	if input.Username == "" {
		return fmt.Errorf("username is required")
	}
	if input.Password == "" {
		return fmt.Errorf("password is required")
	}
	if input.Role == "" {
		return fmt.Errorf("role is required")
	}
	if input.Role != "admin" && input.Role != "viewer" {
		return fmt.Errorf("invalid role: must be 'admin' or 'viewer'")
	}
	return nil
}

// CreateUser creates a new user with the given credentials.
// The user will be required to change their password on first login.
func (s *UserService) CreateUser(input *CreateUserInput) (*UserDTO, error) {
	if err := input.Validate(); err != nil {
		return nil, err
	}

	// Check if user already exists
	existing, err := s.db.GetUserByUsername(input.Username)
	if err != nil {
		return nil, fmt.Errorf("failed to check existing user: %w", err)
	}
	if existing != nil {
		return nil, fmt.Errorf("user already exists")
	}

	// Hash password
	hash, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	// Create user with must_change_password = true
	if err := s.db.CreateUser(input.Username, string(hash), input.Role, true); err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	// Fetch and return the created user
	user, err := s.db.GetUserByUsername(input.Username)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch created user: %w", err)
	}
	if user == nil {
		return nil, fmt.Errorf("user not found after creation")
	}

	return &UserDTO{
		ID:                 user.ID,
		Username:           user.Username,
		Role:               user.Role,
		MustChangePassword: user.MustChangePassword,
		CreatedAt:          user.CreatedAt.Format(time.RFC3339),
	}, nil
}

// GetUserByUsername retrieves a user by username.
// Note: Returns user data without password hash.
func (s *UserService) GetUserByUsername(username string) (*UserDTO, error) {
	user, err := s.db.GetUserByUsername(username)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	if user == nil {
		return nil, fmt.Errorf("user not found")
	}

	return &UserDTO{
		ID:                 user.ID,
		Username:           user.Username,
		Role:               user.Role,
		MustChangePassword: user.MustChangePassword,
		CreatedAt:          user.CreatedAt.Format(time.RFC3339),
	}, nil
}

// GetUserByID retrieves a user by ID.
func (s *UserService) GetUserByID(id int) (*UserDTO, error) {
	user, err := s.db.GetUserByID(id)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	if user == nil {
		return nil, fmt.Errorf("user not found")
	}

	return &UserDTO{
		ID:                 user.ID,
		Username:           user.Username,
		Role:               user.Role,
		MustChangePassword: user.MustChangePassword,
		CreatedAt:          user.CreatedAt.Format(time.RFC3339),
	}, nil
}

// ListUsers returns all users (excluding password hashes).
func (s *UserService) ListUsers() ([]UserDTO, error) {
	users, err := s.db.ListUsers()
	if err != nil {
		return nil, fmt.Errorf("failed to list users: %w", err)
	}

	dtos := make([]UserDTO, len(users))
	for i, user := range users {
		dtos[i] = UserDTO{
			ID:                 user.ID,
			Username:           user.Username,
			Role:               user.Role,
			MustChangePassword: user.MustChangePassword,
			CreatedAt:          user.CreatedAt.Format(time.RFC3339),
		}
	}

	return dtos, nil
}

// ValidateCredentials checks if the username and password are correct.
// This is used for login operations and should NOT expose which field is wrong.
func (s *UserService) ValidateCredentials(username, password string) (*UserDTO, error) {
	user, err := s.db.GetUserByUsername(username)
	if err != nil {
		return nil, fmt.Errorf("authentication failed")
	}
	if user == nil {
		return nil, fmt.Errorf("authentication failed")
	}

	// Compare password
	err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password))
	if err != nil {
		return nil, fmt.Errorf("authentication failed")
	}

	return &UserDTO{
		ID:                 user.ID,
		Username:           user.Username,
		Role:               user.Role,
		MustChangePassword: user.MustChangePassword,
		CreatedAt:          user.CreatedAt.Format(time.RFC3339),
	}, nil
}

// UpdatePassword updates a user's password and clears the must_change_password flag.
func (s *UserService) UpdatePassword(userID int, oldPassword, newPassword string) error {
	// Fetch user to verify old password
	user, err := s.db.GetUserByID(userID)
	if err != nil {
		return fmt.Errorf("failed to get user: %w", err)
	}
	if user == nil {
		return fmt.Errorf("user not found")
	}

	// Verify old password
	err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(oldPassword))
	if err != nil {
		return fmt.Errorf("incorrect password")
	}

	// Hash new password
	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	// Update password and clear must_change_password flag
	if err := s.db.UpdatePassword(userID, string(hash), false); err != nil {
		return fmt.Errorf("failed to update password: %w", err)
	}

	return nil
}

// DeleteUser removes a user by ID.
func (s *UserService) DeleteUser(userID int) error {
	// Verify user exists
	user, err := s.db.GetUserByID(userID)
	if err != nil {
		return fmt.Errorf("failed to get user: %w", err)
	}
	if user == nil {
		return fmt.Errorf("user not found")
	}

	if err := s.db.DeleteUser(userID); err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}

	return nil
}

// UpdateUserRole updates a user's role.
func (s *UserService) UpdateUserRole(userID int, role string) error {
	// Validate role
	if role != "admin" && role != "viewer" {
		return fmt.Errorf("invalid role: must be 'admin' or 'viewer'")
	}

	// Verify user exists
	user, err := s.db.GetUserByID(userID)
	if err != nil {
		return fmt.Errorf("failed to get user: %w", err)
	}
	if user == nil {
		return fmt.Errorf("user not found")
	}

	if err := s.db.UpdateUserRole(userID, role); err != nil {
		return fmt.Errorf("failed to update role: %w", err)
	}

	return nil
}

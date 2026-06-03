package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

// JWTClaims holds the claims in a JWT token
type JWTClaims struct {
	UserID      int    `json:"user_id"`
	Username    string `json:"username"`
	Role        string `json:"role"`
	MustChange  bool   `json:"must_change"`
	jwt.RegisteredClaims
}

// APIContract: matches dashboard/src/types/api.ts LoginResponse interface
// Last synced: 2026-06-03
type LoginResponse struct {
	Token                string `json:"token"`
	Role                 string `json:"role"`
	MustChangePassword   bool   `json:"must_change_password"`
}

// APIContract: refresh endpoint returns same shape as login
// Last synced: 2026-06-03
type RefreshTokenResponse struct {
	Token                string `json:"token"`
	Role                 string `json:"role"`
	MustChangePassword   bool   `json:"must_change_password"`
}

// UserClaimsCtxKey is the context key for storing user claims
type UserClaimsCtxKey struct{}

// GenerateJWT creates a signed JWT token with the given claims.
// Token expires in 24 hours.
func GenerateJWT(userID int, username, role string, mustChange bool, secret string) (string, error) {
	claims := &JWTClaims{
		UserID:     userID,
		Username:   username,
		Role:       role,
		MustChange: mustChange,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   fmt.Sprintf("%d", userID),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

// ValidateJWT verifies and parses a JWT token, returning the claims if valid.
func ValidateJWT(tokenString, secret string) (*JWTClaims, error) {
	claims := &JWTClaims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		// Verify signing method
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(secret), nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}

	if !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}

	return claims, nil
}

// JWTMiddleware validates JWT tokens in Authorization header.
// Falls back to admin/agent token validation for backward compatibility.
// Stores user claims in request context for downstream handlers.
func (s *Server) JWTMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")

		// Try JWT first
		if authHeader != "" && strings.HasPrefix(authHeader, "Bearer ") {
			tokenString := strings.TrimPrefix(authHeader, "Bearer ")
			claims, err := ValidateJWT(tokenString, s.cfg.JWTSecret)
			if err == nil {
				// JWT is valid, store claims in context
				ctx := context.WithValue(r.Context(), UserClaimsCtxKey{}, claims)
				next(w, r.WithContext(ctx))
				return
			}
		}

		// Fall back to admin/agent token validation (backward compatibility)
		if authHeader != "" {
			token := authHeader
			if strings.HasPrefix(token, "Bearer ") {
				token = strings.TrimPrefix(token, "Bearer ")
			}

			// admin token — always valid (for backward compatibility)
			if token == s.cfg.AdminToken {
				// Inject fake admin claims for compatibility with role-based middleware
				adminClaims := &JWTClaims{
					UserID:     0, // Special ID for admin token
					Username:   "admin",
					Role:       "admin",
					MustChange: false,
				}
				ctx := context.WithValue(r.Context(), UserClaimsCtxKey{}, adminClaims)
				next(w, r.WithContext(ctx))
				return
			}

			// check agent token in DB
			if _, err := s.db.ValidateToken(token); err == nil {
				// Agent token is valid, but don't inject claims (agent endpoints handle this differently)
				next(w, r)
				return
			}
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized"})
	}
}

// RequireRole returns middleware that checks if the user has one of the required roles.
// The user claims must be in the request context (set by JWTMiddleware).
func RequireRole(roles ...string) func(http.HandlerFunc) http.HandlerFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			claims, ok := r.Context().Value(UserClaimsCtxKey{}).(*JWTClaims)
			if !ok {
				// No user claims in context - deny access
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusForbidden)
				json.NewEncoder(w).Encode(map[string]string{"error": "forbidden"})
				return
			}

			// Check if user role is in required roles
			hasRole := false
			for _, role := range roles {
				if claims.Role == role {
					hasRole = true
					break
				}
			}

			if !hasRole {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusForbidden)
				json.NewEncoder(w).Encode(map[string]string{"error": "insufficient permissions"})
				return
			}

			next(w, r)
		}
	}
}

// RequirePasswordChange returns middleware that checks if the user must change their password.
// Returns 403 if must_change is true. Allows certain routes to proceed regardless.
func RequirePasswordChange(exemptPaths ...string) func(http.HandlerFunc) http.HandlerFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			// Check if this path is exempt
			for _, path := range exemptPaths {
				if r.URL.Path == path {
					next(w, r)
					return
				}
			}

			claims, ok := r.Context().Value(UserClaimsCtxKey{}).(*JWTClaims)
			if ok && claims.MustChange {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusForbidden)
				json.NewEncoder(w).Encode(map[string]string{"error": "must_change_password"})
				return
			}

			next(w, r)
		}
	}
}

// GetUserClaims extracts user claims from request context.
// Returns nil if claims are not present.
func GetUserClaims(r *http.Request) *JWTClaims {
	claims, ok := r.Context().Value(UserClaimsCtxKey{}).(*JWTClaims)
	if !ok {
		return nil
	}
	return claims
}

// === Auth Endpoint Handlers ===

// handleLogin handles POST /api/auth/login
// Body: {"username":"...","password":"..."}
// Returns: {"token":"...","role":"...","must_change_password":true/false}
func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid request"})
		return
	}

	// Look up user by username
	user, err := s.db.GetUserByUsername(req.Username)
	if err != nil || user == nil {
		// Return generic error to prevent user enumeration
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid username or password"})
		return
	}

	// Verify password
	err = bcryptCompare(user.PasswordHash, req.Password)
	if err != nil {
		// Return generic error to prevent user enumeration
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid username or password"})
		return
	}

	// Generate JWT token
	token, err := GenerateJWT(user.ID, user.Username, user.Role, user.MustChangePassword, s.cfg.JWTSecret)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "failed to generate token"})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(LoginResponse{
		Token:                token,
		Role:                 user.Role,
		MustChangePassword:   user.MustChangePassword,
	})
}

// handleLogout handles POST /api/auth/logout
// Returns: {"ok":true}
func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]bool{"ok": true})
}

// handleAuthMe handles GET /api/auth/me
// Returns: {"id":1,"username":"admin","role":"admin","must_change_password":false}
func (s *Server) handleAuthMe(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	claims := GetUserClaims(r)
	if claims == nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized"})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"id":                     claims.UserID,
		"username":               claims.Username,
		"role":                   claims.Role,
		"must_change_password":   claims.MustChange,
	})
}

// handleChangePassword handles PUT /api/auth/change-password
// Body: {"current_password":"...","new_password":"..."}
func (s *Server) handleChangePassword(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	claims := GetUserClaims(r)
	if claims == nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized"})
		return
	}

	var req struct {
		CurrentPassword string `json:"current_password"`
		NewPassword     string `json:"new_password"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid request"})
		return
	}

	// Validate new password length
	if len(req.NewPassword) < 8 {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "password must be at least 8 characters"})
		return
	}

	// Get current user from DB to verify current password
	user, err := s.db.GetUserByID(claims.UserID)
	if err != nil || user == nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "user not found"})
		return
	}

	// Verify current password
	if err := bcryptCompare(user.PasswordHash, req.CurrentPassword); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"error": "current password is incorrect"})
		return
	}

	// Hash new password
	newHash, err := bcryptHash(req.NewPassword)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "failed to update password"})
		return
	}

	// Update password and clear must_change flag
	if err := s.db.UpdatePassword(claims.UserID, newHash, false); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "failed to update password"})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]bool{"ok": true})
}

// handleRefreshToken handles POST /api/auth/refresh
// Validates the caller's existing JWT and issues a fresh one with a new 24h expiry.
// The current user's role is re-read from the DB so role changes take effect immediately.
func (s *Server) handleRefreshToken(w http.ResponseWriter, r *http.Request) {
	claims := GetUserClaims(r)
	if claims == nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized"})
		return
	}

	// Re-verify the user still exists (catches deleted accounts).
	user, err := s.db.GetUserByID(claims.UserID)
	if err != nil || user == nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"error": "user not found"})
		return
	}

	token, err := GenerateJWT(user.ID, user.Username, user.Role, user.MustChangePassword, s.cfg.JWTSecret)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "failed to generate token"})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(RefreshTokenResponse{
		Token:                token,
		Role:                 user.Role,
		MustChangePassword:   user.MustChangePassword,
	})
}

// bcryptHash hashes a password using bcrypt
func bcryptHash(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(hash), err
}

// bcryptCompare compares a bcrypt hash with a plaintext password
func bcryptCompare(hash, password string) error {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
}

// === User Management Endpoints (Admin Only) ===

// handleListUsers handles GET /api/users — admin only
func (s *Server) handleListUsers(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	claims := GetUserClaims(r)
	if claims == nil || claims.Role != "admin" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(map[string]string{"error": "admin only"})
		return
	}

	users, err := s.db.ListUsers()
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "failed to list users"})
		return
	}

	// Omit password_hash from response
	type UserResponse struct {
		ID                   int       `json:"id"`
		Username             string    `json:"username"`
		Role                 string    `json:"role"`
		MustChangePassword   bool      `json:"must_change_password"`
		CreatedAt            time.Time `json:"created_at"`
	}

	var response []UserResponse
	for _, u := range users {
		response = append(response, UserResponse{
			ID:                   u.ID,
			Username:             u.Username,
			Role:                 u.Role,
			MustChangePassword:   u.MustChangePassword,
			CreatedAt:            u.CreatedAt,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// handleCreateUser handles POST /api/users — admin only
// Body: {"username":"...","password":"...","role":"..."}
func (s *Server) handleCreateUser(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	claims := GetUserClaims(r)
	if claims == nil || claims.Role != "admin" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(map[string]string{"error": "admin only"})
		return
	}

	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
		Role     string `json:"role"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid request"})
		return
	}

	// Validate role
	if req.Role != "admin" && req.Role != "operator" && req.Role != "viewer" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid role"})
		return
	}

	// Hash password
	hash, err := bcryptHash(req.Password)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "failed to create user"})
		return
	}

	// Create user with must_change_password=true
	if err := s.db.CreateUser(req.Username, hash, req.Role, true); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "username already exists"})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{"ok": "user created"})
}

// handleDeleteUser handles DELETE /api/users/{id} — admin only
// Prevents deleting own account
func (s *Server) handleDeleteUser(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	claims := GetUserClaims(r)
	if claims == nil || claims.Role != "admin" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(map[string]string{"error": "admin only"})
		return
	}

	// Parse user ID from URL
	userIDStr := r.PathValue("id")
	var userID int
	if _, err := fmt.Sscanf(userIDStr, "%d", &userID); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid user id"})
		return
	}

	// Prevent deleting own account
	if userID == claims.UserID {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "cannot delete own account"})
		return
	}

	// Delete user
	if err := s.db.DeleteUser(userID); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "failed to delete user"})
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// handleUpdateUserRole handles PUT /api/users/{id}/role — admin only
// Body: {"role":"..."}
func (s *Server) handleUpdateUserRole(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	claims := GetUserClaims(r)
	if claims == nil || claims.Role != "admin" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(map[string]string{"error": "admin only"})
		return
	}

	// Parse user ID from URL
	userIDStr := r.PathValue("id")
	var userID int
	if _, err := fmt.Sscanf(userIDStr, "%d", &userID); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid user id"})
		return
	}

	var req struct {
		Role string `json:"role"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid request"})
		return
	}

	// Validate role
	if req.Role != "admin" && req.Role != "operator" && req.Role != "viewer" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid role"})
		return
	}

	// Update role
	if err := s.db.UpdateUserRole(userID, req.Role); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "failed to update role"})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"ok": "role updated"})
}

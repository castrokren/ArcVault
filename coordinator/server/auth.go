package server

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
	"golang.org/x/time/rate"

	"arcvault/coordinator/business"
)

// JWTClaims holds the claims in a JWT token
type JWTClaims struct {
	UserID     int    `json:"user_id"`
	Username   string `json:"username"`
	Role       string `json:"role"`
	MustChange bool   `json:"must_change"`
	jwt.RegisteredClaims
}

// APIContract: matches dashboard/src/types/api.ts LoginResponse interface
// Last synced: 2026-06-03
type LoginResponse struct {
	Token              string `json:"token"`
	Role               string `json:"role"`
	MustChangePassword bool   `json:"must_change_password"`
}

// APIContract: refresh endpoint returns same shape as login
// Last synced: 2026-06-03
type RefreshTokenResponse struct {
	Token              string `json:"token"`
	Role               string `json:"role"`
	MustChangePassword bool   `json:"must_change_password"`
}

// UserClaimsCtxKey is the context key for storing user claims
type UserClaimsCtxKey struct{}

// AgentIDCtxKey is the context key for storing the authenticated agent's ID.
type AgentIDCtxKey struct{}

// newJTI returns a random JWT ID. Every issued token needs one: logout
// revocation keys the revoked_tokens table on it, and a token without a jti
// can never be revoked.
func newJTI() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// GenerateJWT creates a signed JWT token with the given claims.
// Token expires in 4 hours.
func GenerateJWT(userID int, username, role string, mustChange bool, secret string) (string, error) {
	jti, err := newJTI()
	if err != nil {
		return "", fmt.Errorf("could not generate token id: %w", err)
	}

	claims := &JWTClaims{
		UserID:     userID,
		Username:   username,
		Role:       role,
		MustChange: mustChange,
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        jti,
			Subject:   fmt.Sprintf("%d", userID),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(4 * time.Hour)),
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

// tokenRevoked reports whether the token behind these claims has been revoked.
// A token with no jti predates jti issuance and cannot be revoked, so it is
// treated as revoked rather than trusted.
func (s *Server) tokenRevoked(claims *JWTClaims) (bool, error) {
	if claims.ID == "" {
		return true, nil
	}
	return s.db.IsTokenRevoked(claims.ID)
}

// StartTokenPruner deletes already-expired rows from revoked_tokens on a ticker.
// Nothing else calls PruneExpiredTokens, so without this the table grows one row
// per logout, forever. A revoked token that is also expired is rejected by
// ValidateJWT anyway, so dropping the row loses nothing.
//
// ponytail: own ticker rather than folding into StartOfflineDetector -- a
// function named "offline detector" that also GCs tokens is a 3am puzzle.
func (s *Server) StartTokenPruner(interval time.Duration) {
	prune := func() {
		if err := s.db.PruneExpiredTokens(); err != nil {
			log.Printf("TokenPruner: prune failed: %v", err)
		}
	}
	prune() // once at startup, to clear whatever the last run left behind
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for range ticker.C {
			prune()
		}
	}()
}

// JWTMiddleware validates a JWT in the Authorization header and stores the
// claims in the request context for downstream handlers.
//
// It does NOT accept the admin or agent token. Those are machine credentials,
// not user sessions: routes that legitimately accept them wrap the handler in
// authMiddleware / agentOrViewerRoute / adminTokenRoute, which check the token
// explicitly. Previously JWTMiddleware fell back to injecting fake admin claims
// for the admin token, which made it a role-bypassing master key on every
// user route (user management, roles, credentials); that fallback is gone.
func (s *Server) JWTMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")

		if authHeader != "" && strings.HasPrefix(authHeader, "Bearer ") {
			tokenString := strings.TrimPrefix(authHeader, "Bearer ")
			claims, err := ValidateJWT(tokenString, s.cfg.JWTSecret)
			if err == nil {
				// Revocation check is fail-closed: a token we cannot prove is live
				// (no jti, or the revocation store is unreachable) is not accepted.
				revoked, err := s.tokenRevoked(claims)
				if err != nil {
					log.Printf("JWTMiddleware: revocation check failed: %v", err)
				}
				if revoked || err != nil {
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusUnauthorized)
					json.NewEncoder(w).Encode(map[string]string{"error": "token has been revoked"})
					return
				}
				// JWT is valid, store claims in context
				ctx := context.WithValue(r.Context(), UserClaimsCtxKey{}, claims)
				next(w, r.WithContext(ctx))
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

// GetAgentID extracts the agent ID from request context.
// Returns empty string if not set (e.g., JWT-authenticated requests).
func GetAgentID(r *http.Request) string {
	agentID, ok := r.Context().Value(AgentIDCtxKey{}).(string)
	if !ok {
		return ""
	}
	return agentID
}

// Login throttling is applied to two independent keys per attempt:
//
//	ip:<addr>       — stops one host hammering many accounts
//	user:<username> — stops many hosts hammering one account
//
// Per-IP alone left a distributed attack against a known username (admin)
// effectively unlimited. Both keys share the same budget: 5 attempt burst,
// refilling 1 per 12s, plus a 10-minute lockout after 10 consecutive failures.
const (
	loginBurst       = 5
	loginLockAfter   = 10
	loginLockFor     = 10 * time.Minute
	limiterIdleAfter = 30 * time.Minute
	limiterMaxKeys   = 4096
)

// clientIPKey derives the throttle key for the caller's network address.
// It deliberately uses RemoteAddr, never X-Forwarded-For: a forgeable key
// would let an attacker sidestep the limit with a fresh header per request.
func clientIPKey(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		host = r.RemoteAddr
	}
	return "ip:" + host
}

// accountKey derives the throttle key for a username. It is applied whether or
// not the account exists, so a locked response reveals nothing about existence.
func accountKey(username string) string {
	return "user:" + strings.ToLower(strings.TrimSpace(username))
}

// loginKeyAllowed consumes one token for key. Returns false when the key is
// locked out or over its rate. Callers must hold no locks.
func (s *Server) loginKeyAllowed(key string) bool {
	s.loginLimiterMu.Lock()
	entry, ok := s.loginLimiters[key]
	if !ok {
		s.pruneLoginLimitersLocked()
		entry = &loginRateLimiter{limiter: rate.NewLimiter(rate.Every(time.Minute/loginBurst), loginBurst)}
		s.loginLimiters[key] = entry
	}
	entry.lastSeen = time.Now()

	if entry.lockedAt != nil {
		if time.Since(*entry.lockedAt) < loginLockFor {
			s.loginLimiterMu.Unlock()
			return false
		}
		// Lockout expired — reset
		entry.lockedAt = nil
		entry.failures = 0
	}
	s.loginLimiterMu.Unlock()

	return entry.limiter.Allow()
}

// pruneLoginLimitersLocked drops idle, unlocked entries once the map grows past
// limiterMaxKeys, so a wide scan of source IPs cannot grow it without bound.
// Caller must hold loginLimiterMu.
func (s *Server) pruneLoginLimitersLocked() {
	if len(s.loginLimiters) < limiterMaxKeys {
		return
	}
	for k, e := range s.loginLimiters {
		if e.lockedAt == nil && time.Since(e.lastSeen) > limiterIdleAfter {
			delete(s.loginLimiters, k)
		}
	}
}

// loginAllowed applies the per-IP throttle and writes 429 if it trips.
// Call before any bcrypt work. The per-account throttle is applied separately
// once the username is known — see accountAllowed.
func (s *Server) loginAllowed(w http.ResponseWriter, r *http.Request) bool {
	if s.loginKeyAllowed(clientIPKey(r)) {
		return true
	}
	writeLoginThrottled(w)
	return false
}

// accountAllowed applies the per-account throttle and writes 429 if it trips.
func (s *Server) accountAllowed(w http.ResponseWriter, username string) bool {
	if s.loginKeyAllowed(accountKey(username)) {
		return true
	}
	writeLoginThrottled(w)
	return false
}

// writeLoginThrottled emits one response shape for both throttles, so the
// caller cannot tell which limit it hit.
func writeLoginThrottled(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusTooManyRequests)
	json.NewEncoder(w).Encode(ErrorResponse{Error: "too many failed attempts, try again later"})
}

// recordLoginFailure increments the failure counter for a key and locks it at 10.
func (s *Server) recordLoginFailure(keys ...string) {
	s.loginLimiterMu.Lock()
	defer s.loginLimiterMu.Unlock()
	for _, key := range keys {
		entry, ok := s.loginLimiters[key]
		if !ok {
			continue
		}
		entry.failures++
		if entry.failures >= loginLockAfter {
			now := time.Now()
			entry.lockedAt = &now
		}
	}
}

// recordLoginSuccess resets the failure counters for the given keys.
func (s *Server) recordLoginSuccess(keys ...string) {
	s.loginLimiterMu.Lock()
	defer s.loginLimiterMu.Unlock()
	for _, key := range keys {
		if entry, ok := s.loginLimiters[key]; ok {
			entry.failures = 0
			entry.lockedAt = nil
		}
	}
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

	// Rate limit before any bcrypt work
	if !s.loginAllowed(w, r) {
		return
	}

	var req LoginRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(ErrorResponse{Error: "invalid request"})
		return
	}

	// Validate request
	if err := req.Validate(); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(ErrorResponse{Error: err.Error()})
		return
	}

	// Throttle the target account too, so a distributed attack on one username
	// is limited even though every request arrives from a different IP.
	if !s.accountAllowed(w, req.Username) {
		log.Printf("Login throttled for account %q from %s", req.Username, r.RemoteAddr)
		return
	}

	throttleKeys := []string{clientIPKey(r), accountKey(req.Username)}

	// Use service to validate credentials
	user, err := s.userService.ValidateCredentials(req.Username, req.Password)
	if err != nil {
		s.recordLoginFailure(throttleKeys...)
		log.Printf("Login failed for username %q from %s", req.Username, r.RemoteAddr)
		s.auditService.LogAction(nil, req.Username, "", "auth.login", business.ClientIP(r), false, nil, nil, strPtr("invalid credentials"))
		// Return generic error to prevent user enumeration
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(ErrorResponse{Error: "invalid username or password"})
		return
	}
	s.recordLoginSuccess(throttleKeys...)

	// Generate JWT token
	token, err := GenerateJWT(user.ID, user.Username, user.Role, user.MustChangePassword, s.cfg.JWTSecret)
	if err != nil {
		s.auditService.LogAction(nil, req.Username, "", "auth.login", business.ClientIP(r), false, nil, nil, strPtr("token generation failed"))
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(ErrorResponse{Error: "failed to generate token"})
		return
	}

	s.auditService.LogAction(&user.ID, user.Username, user.Role, "auth.login", business.ClientIP(r), true, nil, nil, nil)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"user_id":              user.ID,
		"username":             user.Username,
		"role":                 user.Role,
		"token":                token,
		"expires_in":           14400,
		"must_change_password": user.MustChangePassword,
	})
}

// handleLogout handles POST /api/auth/logout
// Returns: {"ok":true}
func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	claims := GetUserClaims(r)
	if claims != nil && claims.ID != "" {
		expiry := time.Now().Add(4 * time.Hour) // matches token lifetime
		if claims.ExpiresAt != nil {
			expiry = claims.ExpiresAt.Time
		}
		// A logout that fails to revoke leaves the token live — say so rather
		// than reporting success.
		if err := s.db.RevokeToken(claims.ID, expiry); err != nil {
			log.Printf("Logout: failed to revoke token for %q: %v", claims.Username, err)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(ErrorResponse{Error: "logout failed"})
			return
		}
	}

	if claims != nil {
		s.auditService.LogAction(&claims.UserID, claims.Username, claims.Role, "auth.logout", business.ClientIP(r), true, nil, nil, nil)
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
		"id":                   claims.UserID,
		"username":             claims.Username,
		"role":                 claims.Role,
		"must_change_password": claims.MustChange,
	})
}

// handleChangePassword handles POST /api/auth/change-password
// Body: {"old_password":"...","new_password":"..."}
func (s *Server) handleChangePassword(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	claims := GetUserClaims(r)
	if claims == nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(ErrorResponse{Error: "unauthorized"})
		return
	}

	// Throttle before bcrypt: a stolen token must not allow unlimited
	// old-password brute force. Keyed per user, reusing the login limiter.
	pwKey := fmt.Sprintf("pwchange:%d", claims.UserID)
	if !s.loginKeyAllowed(pwKey) {
		writeLoginThrottled(w)
		return
	}

	var req ChangePasswordRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(ErrorResponse{Error: "invalid request"})
		return
	}

	// Validate request
	if err := req.Validate(); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(ErrorResponse{Error: err.Error()})
		return
	}

	// Use service to update password
	if err := s.userService.UpdatePassword(claims.UserID, req.OldPassword, req.NewPassword); err != nil {
		s.recordLoginFailure(pwKey)
		s.auditService.LogAction(&claims.UserID, claims.Username, claims.Role, "auth.change_password", business.ClientIP(r), false, strPtr("user"), nil, strPtr(err.Error()))
		w.Header().Set("Content-Type", "application/json")
		if err.Error() == "incorrect password" {
			w.WriteHeader(http.StatusUnauthorized)
		} else {
			w.WriteHeader(http.StatusInternalServerError)
		}
		json.NewEncoder(w).Encode(ErrorResponse{Error: err.Error()})
		return
	}
	s.recordLoginSuccess(pwKey)

	s.auditService.LogAction(&claims.UserID, claims.Username, claims.Role, "auth.change_password", business.ClientIP(r), true, strPtr("user"), nil, nil)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(MessageResponse{Message: "Password changed successfully"})
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
	user, err := s.userService.GetUserByID(claims.UserID)
	if err != nil {
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
		Token:              token,
		Role:               user.Role,
		MustChangePassword: user.MustChangePassword,
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

	p := ParsePagination(r)

	users, err := s.userService.ListUsers()
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "failed to list users"})
		return
	}

	total := len(users)
	offset := (p.Page - 1) * p.Limit
	if offset >= total {
		offset = total
	}
	end := offset + p.Limit
	if end > total {
		end = total
	}
	paginatedUsers := users[offset:end]

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(NewPaginatedResponse(paginatedUsers, total, p.Page, p.Limit))
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
		json.NewEncoder(w).Encode(ErrorResponse{Error: "not authorized"})
		return
	}

	var req CreateUserRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(ErrorResponse{Error: "invalid request"})
		return
	}

	// Validate request
	if err := req.Validate(); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(ErrorResponse{Error: err.Error()})
		return
	}

	input := &business.CreateUserInput{
		Username: req.Username,
		Password: req.Password,
		Role:     req.Role,
	}

	user, err := s.userService.CreateUser(input)
	if err != nil {
		s.auditService.LogAction(&claims.UserID, claims.Username, claims.Role, "user.create", business.ClientIP(r), false, strPtr("user"), &req.Username, strPtr(err.Error()))
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(ErrorResponse{Error: err.Error()})
		return
	}

	s.auditService.LogAction(&claims.UserID, claims.Username, claims.Role, "user.create", business.ClientIP(r), true, strPtr("user"), &req.Username, nil)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(UserResponse{
		UserID:             user.ID,
		Username:           user.Username,
		Role:               user.Role,
		MustChangePassword: user.MustChangePassword,
		CreatedAt:          user.CreatedAt,
	})
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

	// Look up target user before deletion for audit log
	targetUser, _ := s.userService.GetUserByID(userID)
	targetUsername := ""
	if targetUser != nil {
		targetUsername = targetUser.Username
	}
	idStr := strconv.Itoa(userID)

	// Use service to delete user
	if err := s.userService.DeleteUser(userID); err != nil {
		s.auditService.LogAction(&claims.UserID, claims.Username, claims.Role, "user.delete", business.ClientIP(r), false, strPtr("user"), &idStr, strPtr(err.Error()))
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "failed to delete user"})
		return
	}

	s.auditService.LogAction(&claims.UserID, claims.Username, claims.Role, "user.delete", business.ClientIP(r), true, strPtr("user"), &idStr, strPtr("deleted user: "+targetUsername))

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
		json.NewEncoder(w).Encode(ErrorResponse{Error: "not authorized"})
		return
	}

	// Parse user ID from URL
	userIDStr := r.PathValue("id")
	var userID int
	if _, err := fmt.Sscanf(userIDStr, "%d", &userID); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(ErrorResponse{Error: "invalid user id"})
		return
	}

	var req UpdateUserRoleRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(ErrorResponse{Error: "invalid request"})
		return
	}

	// Validate request
	if err := req.Validate(); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(ErrorResponse{Error: err.Error()})
		return
	}

	// Use service to update user role
	if err := s.userService.UpdateUserRole(userID, req.Role); err != nil {
		idStr := strconv.Itoa(userID)
		s.auditService.LogAction(&claims.UserID, claims.Username, claims.Role, "user.update_role", business.ClientIP(r), false, strPtr("user"), &idStr, strPtr(err.Error()))
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(ErrorResponse{Error: err.Error()})
		return
	}

	idStr := strconv.Itoa(userID)
	s.auditService.LogAction(&claims.UserID, claims.Username, claims.Role, "user.update_role", business.ClientIP(r), true, strPtr("user"), &idStr, strPtr("new role: "+req.Role))

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(UpdateUserRoleResponse{
		UserID:    userID,
		Username:  "", // Would need to fetch from service to populate
		Role:      req.Role,
		UpdatedAt: time.Now().UTC().Format(time.RFC3339),
	})
}

func strPtr(s string) *string { return &s }

// LoginRequest defines the shape of a login request body
type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// Validate checks if LoginRequest is valid
// validatePasswordStrength checks common password complexity requirements.
func validatePasswordStrength(password string) error {
	if len(password) < 8 {
		return fmt.Errorf("password must be 8 or more characters")
	}
	if !containsUppercase(password) {
		return fmt.Errorf("password must contain at least one uppercase letter")
	}
	if !containsLowercase(password) {
		return fmt.Errorf("password must contain at least one lowercase letter")
	}
	if !containsDigit(password) {
		return fmt.Errorf("password must contain at least one digit")
	}
	if !containsSpecial(password) {
		return fmt.Errorf("password must contain at least one special character")
	}
	return nil
}

// Helper functions for password validation
func containsUppercase(s string) bool {
	for _, r := range s {
		if r >= 'A' && r <= 'Z' {
			return true
		}
	}
	return false
}

func containsLowercase(s string) bool {
	for _, r := range s {
		if r >= 'a' && r <= 'z' {
			return true
		}
	}
	return false
}

func containsDigit(s string) bool {
	for _, r := range s {
		if r >= '0' && r <= '9' {
			return true
		}
	}
	return false
}

func containsSpecial(s string) bool {
	for _, r := range s {
		if (r >= 33 && r <= 47) || (r >= 58 && r <= 64) || (r >= 91 && r <= 96) || (r >= 123 && r <= 126) {
			return true
		}
	}
	return false
}

func (r *LoginRequest) Validate() error {
	if r.Username == "" {
		return fmt.Errorf("username is required")
	}
	if r.Password == "" {
		return fmt.Errorf("password is required")
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
	if err := validatePasswordStrength(r.NewPassword); err != nil {
		return fmt.Errorf("new_password: %w", err)
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
	if err := validatePasswordStrength(r.Password); err != nil {
		return err
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
	UserID             int    `json:"user_id"`
	Username           string `json:"username"`
	Role               string `json:"role"`
	MustChangePassword bool   `json:"must_change_password"`
	CreatedAt          string `json:"created_at"`
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

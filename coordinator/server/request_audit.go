package server

import (
	"net/http"
	"time"

	"arcvault/coordinator/business"
	"arcvault/coordinator/db"
)

// requestAuditMiddleware wraps an http.Handler and logs every API request
// to the user audit log with user identity, method, path, status, and latency.
func (s *Server) requestAuditMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Skip health check and WebSocket endpoints to reduce noise
		if r.URL.Path == "/health" || r.URL.Path == "/ws" || r.URL.Path == "/ws/agent" || r.URL.Path == "/ws/federation" {
			next.ServeHTTP(w, r)
			return
		}

		start := time.Now()

		// Wrap ResponseWriter to capture status code
		sw := &statusWriter{ResponseWriter: w, statusCode: http.StatusOK}
		next.ServeHTTP(sw, r)

		latency := time.Since(start)
		latencyMs := int(latency.Milliseconds())

		// Extract user info from context (optional — may be nil)
		claims := GetUserClaims(r)

		var userID *int
		username := ""
		userRole := ""
		if claims != nil {
			userID = &claims.UserID
			username = claims.Username
			userRole = claims.Role
		}

		method := r.Method
		path := r.URL.Path
		statusCode := sw.statusCode
		ip := business.ClientIP(r)

		ctx := db.UserAuditLogContext{
			UserID:        userID,
			Username:      username,
			UserRole:      userRole,
			Action:        "request",
			ResourceType:  &method,
			ResourceID:    &path,
			IPAddress:     ip,
			// 4xx is a failed request: logging 401/403 as success hid brute force.
			Success:       statusCode >= 200 && statusCode < 400,
			RequestMethod: &method,
			RequestPath:   &path,
			StatusCode:    &statusCode,
			LatencyMs:     &latencyMs,
		}

		// Best-effort logging — never block the request
		_ = s.db.InsertUserAuditLog(ctx)
	})
}

// statusWriter wraps http.ResponseWriter to capture the status code.
type statusWriter struct {
	http.ResponseWriter
	statusCode int
}

func (sw *statusWriter) WriteHeader(code int) {
	sw.statusCode = code
	sw.ResponseWriter.WriteHeader(code)
}

// logAudit is a convenience wrapper for explicit action logging in mutation handlers.
func (s *Server) logAudit(r *http.Request, claims *JWTClaims, action string, success bool, resourceType, resourceID *string) {
	var userID *int
	username := ""
	userRole := ""
	if claims != nil {
		userID = &claims.UserID
		username = claims.Username
		userRole = claims.Role
	}
	_ = s.auditService.LogAction(userID, username, userRole, action, business.ClientIP(r), success, resourceType, resourceID, nil)
}

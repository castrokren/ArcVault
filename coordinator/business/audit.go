package business

import (
	"net"
	"net/http"
	"strings"
	"sync/atomic"
	"time"

	"arcvault/coordinator/db"
)

// AuditService handles user action audit logging.
type AuditService struct {
	queries db.AuditQueries
}

// NewAuditService creates a new audit service.
func NewAuditService(queries db.AuditQueries) *AuditService {
	return &AuditService{
		queries: queries,
	}
}

// UserAuditEntryDTO is the data transfer object for user audit log entries (API response).
type UserAuditEntryDTO struct {
	ID            int       `json:"id"`
	UserID        *int      `json:"user_id,omitempty"`
	Username      string    `json:"username"`
	UserRole      string    `json:"user_role"`
	Action        string    `json:"action"`
	ResourceType  *string   `json:"resource_type,omitempty"`
	ResourceID    *string   `json:"resource_id,omitempty"`
	Details       *string   `json:"details,omitempty"`
	IPAddress     string    `json:"ip_address"`
	Success       bool      `json:"success"`
	RequestMethod *string   `json:"request_method,omitempty"`
	RequestPath   *string   `json:"request_path,omitempty"`
	StatusCode    *int      `json:"status_code,omitempty"`
	LatencyMs     *int      `json:"latency_ms,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
}

// LogAction records a structured user action in the audit log.
// This is the primary method for explicit action logging in mutation handlers.
func (s *AuditService) LogAction(userID *int, username, userRole, action, ipAddress string, success bool, resourceType, resourceID *string, details *string) error {
	ctx := db.UserAuditLogContext{
		UserID:       userID,
		Username:     username,
		UserRole:     userRole,
		Action:       action,
		ResourceType: resourceType,
		ResourceID:   resourceID,
		Details:      details,
		IPAddress:    ipAddress,
		Success:      success,
	}
	return s.queries.InsertUserAuditLog(ctx)
}

// AuditLogFilter holds filtering parameters for listing audit logs.
type AuditLogFilter struct {
	Action       string
	UserID       int
	ResourceType string
	ResourceID   string
	Username     string
	FromDate     *time.Time
	ToDate       *time.Time
	Success      *bool
	Page         int
	Limit        int
}

// ListAuditLogs retrieves paginated audit log entries.
func (s *AuditService) ListAuditLogs(filter AuditLogFilter) ([]UserAuditEntryDTO, int, error) {
	dbFilter := db.UserAuditLogFilter{
		Action:       filter.Action,
		UserID:       filter.UserID,
		ResourceType: filter.ResourceType,
		ResourceID:   filter.ResourceID,
		Username:     filter.Username,
		FromDate:     filter.FromDate,
		ToDate:       filter.ToDate,
		Success:      filter.Success,
		Offset:       (filter.Page - 1) * filter.Limit,
		Limit:        filter.Limit,
	}
	if dbFilter.Offset < 0 {
		dbFilter.Offset = 0
	}

	entries, total, err := s.queries.ListUserAuditLogs(dbFilter)
	if err != nil {
		return nil, 0, err
	}

	dtos := make([]UserAuditEntryDTO, len(entries))
	for i, e := range entries {
		dtos[i] = UserAuditEntryDTO{
			ID:            e.ID,
			UserID:        e.UserID,
			Username:      e.Username,
			UserRole:      e.UserRole,
			Action:        e.Action,
			ResourceType:  e.ResourceType,
			ResourceID:    e.ResourceID,
			Details:       e.Details,
			IPAddress:     e.IPAddress,
			Success:       e.Success,
			RequestMethod: e.RequestMethod,
			RequestPath:   e.RequestPath,
			StatusCode:    e.StatusCode,
			LatencyMs:     e.LatencyMs,
			CreatedAt:     e.CreatedAt,
		}
	}

	return dtos, total, nil
}

// trustProxyHeaders controls whether ClientIP believes X-Forwarded-For and
// X-Real-IP. It is off by default: ArcVault normally terminates TLS itself, so
// those headers are supplied by the caller and let anyone forge the IP recorded
// in every audit row. Enable it only when a reverse proxy you control sets them.
var trustProxyHeaders atomic.Bool

// SetTrustProxyHeaders configures ClientIP. Call once at startup from config.
func SetTrustProxyHeaders(trust bool) { trustProxyHeaders.Store(trust) }

// ClientIP extracts the client IP address from an HTTP request.
// Falls back to RemoteAddr, which is the only value a client cannot forge.
func ClientIP(r *http.Request) string {
	if trustProxyHeaders.Load() {
		if fwd := r.Header.Get("X-Forwarded-For"); fwd != "" {
			// Take the rightmost entry: that is the one our own proxy appended.
			// A client that pre-seeds its own X-Forwarded-For only prepends.
			parts := strings.Split(fwd, ",")
			return strings.TrimSpace(parts[len(parts)-1])
		}
		if realIP := r.Header.Get("X-Real-IP"); realIP != "" {
			return strings.TrimSpace(realIP)
		}
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

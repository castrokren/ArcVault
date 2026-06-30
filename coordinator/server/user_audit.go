package server

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"arcvault/coordinator/business"
)

// UserAuditEntryResponse is the API response DTO for user audit log entries.
type UserAuditEntryResponse struct {
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

// handleListUserAuditLogs returns paginated user action audit log entries.
// Query params: page, limit, action, user_id, username, resource_type, resource_id, from_date, to_date, success
// Access: viewer+ (viewerRoute)
func (s *Server) handleListUserAuditLogs(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	page, _ := strconv.Atoi(q.Get("page"))
	if page < 1 {
		page = 1
	}
	limit, _ := strconv.Atoi(q.Get("limit"))
	if limit < 1 || limit > 100 {
		limit = 50
	}

	filter := business.AuditLogFilter{
		Action:       q.Get("action"),
		ResourceType: q.Get("resource_type"),
		ResourceID:   q.Get("resource_id"),
		Username:     q.Get("username"),
		Page:         page,
		Limit:        limit,
	}

	if uid := q.Get("user_id"); uid != "" {
		if id, err := strconv.Atoi(uid); err == nil {
			filter.UserID = id
		}
	}

	if fromStr := q.Get("from_date"); fromStr != "" {
		if t, err := time.Parse(time.RFC3339, fromStr); err == nil {
			filter.FromDate = &t
		}
	}
	if toStr := q.Get("to_date"); toStr != "" {
		if t, err := time.Parse(time.RFC3339, toStr); err == nil {
			filter.ToDate = &t
		}
	}

	if successStr := q.Get("success"); successStr != "" {
		success := successStr == "true" || successStr == "1"
		filter.Success = &success
	}

	entries, total, err := s.auditService.ListAuditLogs(filter)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(ErrorResponse{Error: "failed to list audit logs: " + err.Error()})
		return
	}

	// Convert to response DTOs
	response := make([]UserAuditEntryResponse, len(entries))
	for i, e := range entries {
		response[i] = UserAuditEntryResponse{
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

	pages := (total + limit - 1) / limit
	if pages < 1 {
		pages = 1
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(PaginatedResponse{
		Data:  response,
		Total: total,
		Page:  page,
		Pages: pages,
		Limit: limit,
	})
}

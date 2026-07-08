package db

import (
	"fmt"
	"time"
)

// UserAuditLogEntry represents a single user action audit log entry.
type UserAuditLogEntry struct {
	ID            int        `json:"id"`
	UserID        *int       `json:"user_id,omitempty"`
	Username      string     `json:"username"`
	UserRole      string     `json:"user_role"`
	Action        string     `json:"action"`
	ResourceType  *string    `json:"resource_type,omitempty"`
	ResourceID    *string    `json:"resource_id,omitempty"`
	Details       *string    `json:"details,omitempty"`
	IPAddress     string     `json:"ip_address"`
	Success       bool       `json:"success"`
	RequestMethod *string    `json:"request_method,omitempty"`
	RequestPath   *string    `json:"request_path,omitempty"`
	StatusCode    *int       `json:"status_code,omitempty"`
	LatencyMs     *int       `json:"latency_ms,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
}

// UserAuditLogContext holds the data to insert a new audit log entry.
type UserAuditLogContext struct {
	UserID        *int
	Username      string
	UserRole      string
	Action        string
	ResourceType  *string
	ResourceID    *string
	Details       *string
	IPAddress     string
	Success       bool
	RequestMethod *string
	RequestPath   *string
	StatusCode    *int
	LatencyMs     *int
}

// UserAuditLogFilter holds filtering criteria for ListUserAuditLogs.
type UserAuditLogFilter struct {
	Action       string
	UserID       int
	ResourceType string
	ResourceID   string
	Username     string
	FromDate     *time.Time
	ToDate       *time.Time
	Success      *bool
	Limit        int
	Offset       int
}

// InsertUserAuditLog inserts a new user action audit log entry.
func (d *DB) InsertUserAuditLog(ctx UserAuditLogContext) error {
	_, err := d.conn.Exec(`
INSERT INTO user_audit_log (
	user_id, username, user_role, action, resource_type, resource_id,
	details, ip_address, success, request_method, request_path,
	status_code, latency_ms, created_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP)
`, ctx.UserID, ctx.Username, ctx.UserRole, ctx.Action, ctx.ResourceType, ctx.ResourceID,
		ctx.Details, ctx.IPAddress, boolToInt(ctx.Success),
		ctx.RequestMethod, ctx.RequestPath, ctx.StatusCode, ctx.LatencyMs)
	return err
}

// ListUserAuditLogs retrieves user audit logs with filtering and pagination.
// Returns (entries, totalCount, error).
func (d *DB) ListUserAuditLogs(filter UserAuditLogFilter) ([]UserAuditLogEntry, int, error) {
	args := []interface{}{}
	where := " WHERE 1=1"

	if filter.Action != "" {
		where += " AND action = ?"
		args = append(args, filter.Action)
	}
	if filter.UserID != 0 {
		where += " AND user_id = ?"
		args = append(args, filter.UserID)
	}
	if filter.Username != "" {
		where += " AND username = ?"
		args = append(args, filter.Username)
	}
	if filter.ResourceType != "" {
		where += " AND resource_type = ?"
		args = append(args, filter.ResourceType)
	}
	if filter.ResourceID != "" {
		where += " AND resource_id = ?"
		args = append(args, filter.ResourceID)
	}
	if filter.FromDate != nil {
		where += " AND created_at >= ?"
		args = append(args, *filter.FromDate)
	}
	if filter.ToDate != nil {
		where += " AND created_at <= ?"
		args = append(args, *filter.ToDate)
	}
	if filter.Success != nil {
		where += " AND success = ?"
		args = append(args, boolToInt(*filter.Success))
	}

	// Get total count
	var total int
	countArgs := append([]interface{}{}, args...)
	if err := d.conn.QueryRow("SELECT COUNT(*) FROM user_audit_log"+where, countArgs...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count user_audit_log: %w", err)
	}

	// Set defaults
	if filter.Limit == 0 {
		filter.Limit = 50
	}
	if filter.Limit > 1000 {
		filter.Limit = 1000
	}

	// Get paginated results
	query := `SELECT id, user_id, username, user_role, action, resource_type, resource_id,
		details, ip_address, success, request_method, request_path,
		status_code, latency_ms, created_at
		FROM user_audit_log` + where + ` ORDER BY created_at DESC LIMIT ? OFFSET ?`
	args = append(args, filter.Limit, filter.Offset)

	rows, err := d.conn.Query(query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("query user_audit_log: %w", err)
	}
	defer rows.Close()

	var logs []UserAuditLogEntry
	for rows.Next() {
		var entry UserAuditLogEntry
		var successInt int
		if err := rows.Scan(
			&entry.ID, &entry.UserID, &entry.Username, &entry.UserRole,
			&entry.Action, &entry.ResourceType, &entry.ResourceID,
			&entry.Details, &entry.IPAddress, &successInt,
			&entry.RequestMethod, &entry.RequestPath,
			&entry.StatusCode, &entry.LatencyMs, &entry.CreatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("scan user_audit_log: %w", err)
		}
		entry.Success = successInt != 0
		logs = append(logs, entry)
	}
	if logs == nil {
		logs = []UserAuditLogEntry{}
	}

	return logs, total, rows.Err()
}

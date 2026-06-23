package db

import (
	"time"
)

// CommandAuditLog represents a logged command execution in audit mode.
type CommandAuditLog struct {
	ID            int       `json:"id"`
	TemplateID    *string   `json:"template_id,omitempty"`
	JobID         *string   `json:"job_id,omitempty"`
	CommandString string    `json:"command_string"`
	ProgramName   string    `json:"program_name"`
	IsWhitelisted bool      `json:"is_whitelisted"`
	Mode          string    `json:"mode"`
	AuditResult   string    `json:"audit_result"`
	ExecutedAt    time.Time `json:"executed_at"`
	AgentID       string    `json:"agent_id"`
}

// CommandAuditContext holds the audit context for a command execution.
type CommandAuditContext struct {
	TemplateID    *string
	JobID         *string
	CommandString string
	ProgramName   string
	IsWhitelisted bool
	Mode          string
	AuditResult   string
	AgentID       string
}

// LogCommand inserts an audit log entry for a command execution.
func (d *DB) LogCommand(ctx CommandAuditContext) error {
	_, err := d.conn.Exec(`
INSERT INTO command_audit_log (
	template_id, job_id, command_string, program_name, is_whitelisted,
	mode, audit_result, executed_at, agent_id
) VALUES (?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP, ?)
`, ctx.TemplateID, ctx.JobID, ctx.CommandString, ctx.ProgramName,
		boolToInt(ctx.IsWhitelisted), ctx.Mode, ctx.AuditResult, ctx.AgentID)
	return err
}

// AuditLogFilter holds filtering criteria for GetAuditLog.
type AuditLogFilter struct {
	ProgramName   string // filter by program name (exact match)
	IsWhitelisted *bool  // filter by whitelist status (nil = all)
	AgentID       string // filter by agent_id
	FromTime      *time.Time
	ToTime        *time.Time
	Limit         int
	Offset        int
}

// GetAuditLog retrieves audit logs with filtering and pagination.
func (d *DB) GetAuditLog(filter AuditLogFilter) ([]CommandAuditLog, int, error) {
	args := []interface{}{}
	where := " WHERE 1=1"

	if filter.ProgramName != "" {
		where += " AND program_name = ?"
		args = append(args, filter.ProgramName)
	}

	if filter.IsWhitelisted != nil {
		where += " AND is_whitelisted = ?"
		args = append(args, boolToInt(*filter.IsWhitelisted))
	}

	if filter.AgentID != "" {
		where += " AND agent_id = ?"
		args = append(args, filter.AgentID)
	}

	if filter.FromTime != nil {
		where += " AND executed_at >= ?"
		args = append(args, filter.FromTime)
	}

	if filter.ToTime != nil {
		where += " AND executed_at <= ?"
		args = append(args, filter.ToTime)
	}

	// Get total count
	var total int
	countArgs := append([]interface{}{}, args...)
	if err := d.conn.QueryRow("SELECT COUNT(*) FROM command_audit_log"+where, countArgs...).Scan(&total); err != nil {
		return nil, 0, err
	}

	// Set defaults
	if filter.Limit == 0 {
		filter.Limit = 100
	}
	if filter.Limit > 10000 {
		filter.Limit = 10000
	}

	// Get paginated results
	query := "SELECT id, template_id, job_id, command_string, program_name, is_whitelisted, mode, audit_result, executed_at, agent_id FROM command_audit_log" +
		where + " ORDER BY executed_at DESC LIMIT ? OFFSET ?"
	args = append(args, filter.Limit, filter.Offset)

	rows, err := d.conn.Query(query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var logs []CommandAuditLog
	for rows.Next() {
		var log CommandAuditLog
		var isWhitelistedInt int
		if err := rows.Scan(
			&log.ID, &log.TemplateID, &log.JobID, &log.CommandString,
			&log.ProgramName, &isWhitelistedInt, &log.Mode, &log.AuditResult,
			&log.ExecutedAt, &log.AgentID,
		); err != nil {
			return nil, 0, err
		}
		log.IsWhitelisted = isWhitelistedInt != 0
		logs = append(logs, log)
	}

	return logs, total, rows.Err()
}

// NonWhitelistedProgramStats holds statistics for a non-whitelisted program.
type NonWhitelistedProgramStats struct {
	ProgramName    string `json:"program_name"`
	ExecutionCount int    `json:"execution_count"`
	LastExecutedAt string `json:"last_executed_at,omitempty"`
	UniqueAgents   int    `json:"unique_agents"`
}

// GetNonWhitelistedPrograms returns distinct non-whitelisted programs with execution counts.
func (d *DB) GetNonWhitelistedPrograms() ([]NonWhitelistedProgramStats, error) {
	query := `
SELECT
	program_name,
	COUNT(*) as execution_count,
	MAX(executed_at) as last_executed_at,
	COUNT(DISTINCT agent_id) as unique_agents
FROM command_audit_log
WHERE is_whitelisted = 0
GROUP BY program_name
ORDER BY execution_count DESC
`
	rows, err := d.conn.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var stats []NonWhitelistedProgramStats
	for rows.Next() {
		var stat NonWhitelistedProgramStats
		if err := rows.Scan(&stat.ProgramName, &stat.ExecutionCount, &stat.LastExecutedAt, &stat.UniqueAgents); err != nil {
			return nil, err
		}
		stats = append(stats, stat)
	}

	return stats, rows.Err()
}

// GetAuditStats returns counts of whitelisted vs non-whitelisted commands within a time range.
type AuditStats struct {
	WhitelistedCount    int `json:"whitelisted_count"`
	NonWhitelistedCount int `json:"non_whitelisted_count"`
	TotalCount          int `json:"total_count"`
	UniquePrograms      int `json:"unique_programs"`
}

// GetAuditStats retrieves audit statistics for a time range.
func (d *DB) GetAuditStats(fromTime, toTime time.Time) (*AuditStats, error) {
	var stats AuditStats

	// Counts by whitelist status
	err := d.conn.QueryRow(`
SELECT
	COUNT(CASE WHEN is_whitelisted = 1 THEN 1 END) as whitelisted,
	COUNT(CASE WHEN is_whitelisted = 0 THEN 1 END) as non_whitelisted,
	COUNT(*) as total,
	COUNT(DISTINCT program_name) as unique_programs
FROM command_audit_log
WHERE executed_at >= ? AND executed_at <= ?
`, fromTime, toTime).Scan(
		&stats.WhitelistedCount,
		&stats.NonWhitelistedCount,
		&stats.TotalCount,
		&stats.UniquePrograms,
	)

	return &stats, err
}

// boolToInt converts a bool to 1 or 0 for SQLite storage.
func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

// intToBool converts 1 or 0 from SQLite to a bool.
func intToBool(i int) bool {
	return i != 0
}

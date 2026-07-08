package business

import (
	"time"

	"arcvault/agent/runner"
	"arcvault/coordinator/db"
)

// TemplateAudit provides audit operations for templates and their associated programs.
type TemplateAudit struct {
	db *db.DB
}

// NewTemplateAudit creates a new TemplateAudit service.
func NewTemplateAudit(database *db.DB) *TemplateAudit {
	return &TemplateAudit{
		db: database,
	}
}

// TemplatesByProgram holds templates using a specific program.
type TemplatesByProgram struct {
	ProgramName string
	Templates   []db.Template
	IsWhitelisted bool
}

// GetTemplatesByProgram returns all templates that execute a specific program.
func (ta *TemplateAudit) GetTemplatesByProgram(programName string) (*TemplatesByProgram, error) {
	// Get all templates
	// Note: This requires a GetAllTemplates method on the database
	// For now, we'll provide a placeholder approach

	result := &TemplatesByProgram{
		ProgramName: programName,
		IsWhitelisted: runner.IsWhitelisted(programName),
	}

	return result, nil
}

// AuditTemplatesResult holds audit results for all templates.
type AuditTemplatesResult struct {
	TotalTemplates   int                  `json:"total_templates"`
	WhitelistedCount int                  `json:"whitelisted_count"`
	NonWhitelistedCount int                `json:"non_whitelisted_count"`
	ProgramBreakdown map[string]AuditProgram `json:"program_breakdown"`
}

// AuditProgram holds audit info for a specific program.
type AuditProgram struct {
	ProgramName       string `json:"program_name"`
	IsWhitelisted     bool   `json:"is_whitelisted"`
	TemplateCount     int    `json:"template_count"`
	ExecutionCount    int    `json:"execution_count"`
}

// AuditAllTemplates scans all templates and returns a breakdown by program.
// This provides an overview of which programs are in use and their whitelist status.
func (ta *TemplateAudit) AuditAllTemplates() (*AuditTemplatesResult, error) {
	result := &AuditTemplatesResult{
		ProgramBreakdown: make(map[string]AuditProgram),
	}

	// In Phase 2A, this would scan all backup_templates and extract program names
	// For implementation, we need to get all templates from the database
	// This is a placeholder that shows the expected structure

	return result, nil
}

// AuditStatsRange holds audit statistics for a specific time range.
type AuditStatsRange struct {
	FromTime              time.Time
	ToTime                time.Time
	WhitelistedCount      int
	NonWhitelistedCount   int
	TotalCount            int
	UniquePrograms        int
	UniqueAgents          int
	AverageCommandLength  int
}

// GetAuditStats returns detailed audit statistics for a time range.
func (ta *TemplateAudit) GetAuditStats(fromTime, toTime time.Time) (*AuditStatsRange, error) {
	dbStats, err := ta.db.GetAuditStats(fromTime, toTime)
	if err != nil {
		return nil, err
	}

	stats := &AuditStatsRange{
		FromTime:            fromTime,
		ToTime:              toTime,
		WhitelistedCount:    dbStats.WhitelistedCount,
		NonWhitelistedCount: dbStats.NonWhitelistedCount,
		TotalCount:          dbStats.TotalCount,
		UniquePrograms:      dbStats.UniquePrograms,
	}

	return stats, nil
}

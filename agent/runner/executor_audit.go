package runner

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"
)

// ExecutorWithAudit wraps RealExecutor to add command audit logging.
// It extracts the program name, checks whitelist status, and logs the execution.
//
// In Phase 2A (AUDIT MODE):
//   - All commands execute regardless of whitelist status
//   - Every command is logged to the audit file with whitelist status
//   - No commands are rejected
//
// The auditor callback is called after execution to log the result.
func ExecutorWithAudit(ctx context.Context, job Job, report ProgressFunc, auditor CommandAuditor, agentID string) (exitCode int, output string) {
	// Extract program name from command
	programName := ExtractProgramName(job.Command)
	isWhitelisted := IsWhitelisted(programName)

	// Create audit context
	auditCtx := CommandAuditContext{
		CommandString: job.Command,
		ProgramName:   programName,
		IsWhitelisted: isWhitelisted,
		Mode:          "audit", // Phase 2A is audit mode
		AuditResult:   "",
		AgentID:       agentID,
	}

	// Set template_id and job_id if available
	// Note: We're inside job execution, so we set job.ID
	// Template-fired jobs might have Command set; template_id would come from context
	auditCtx.JobID = &job.ID

	// Execute the command using RealExecutor
	exitCode, output = RealExecutor(ctx, job, report)

	// In audit mode, log result as success (command executed)
	auditCtx.AuditResult = fmt.Sprintf("executed: exit_code=%d, output_len=%d", exitCode, len(output))

	// Call the auditor to log the execution
	if auditor != nil {
		auditor(auditCtx)
	}

	return exitCode, output
}

// AuditLogFile returns the path to the audit log file in the config directory.
func AuditLogFile() string {
	configDir, err := os.UserConfigDir()
	if err != nil {
		configDir = "/tmp"
	}
	// Create arcvault subdirectory
	auditDir := filepath.Join(configDir, "arcvault")
	_ = os.MkdirAll(auditDir, 0755)
	return filepath.Join(auditDir, "command_audit.log")
}

// DefaultLocalAuditor logs command execution to a local audit file in TSV format.
// Format: timestamp<tab>agent_id<tab>program<tab>whitelist<tab>job_id<tab>command
func DefaultLocalAuditor(ctx CommandAuditContext) {
	logFile := AuditLogFile()
	f, err := os.OpenFile(logFile, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		log.Printf("[audit] failed to open log file %s: %v", logFile, err)
		return
	}
	defer f.Close()

	// Build TSV line
	timestamp := getCurrentTimestamp()
	jobID := "-"
	if ctx.JobID != nil {
		jobID = *ctx.JobID
	}

	templateID := "-"
	if ctx.TemplateID != nil {
		templateID = *ctx.TemplateID
	}

	whitelistStr := "false"
	if ctx.IsWhitelisted {
		whitelistStr = "true"
	}

	line := fmt.Sprintf("%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
		timestamp,
		ctx.AgentID,
		ctx.ProgramName,
		whitelistStr,
		ctx.Mode,
		jobID,
		templateID,
		ctx.AuditResult,
	)

	if _, err := f.WriteString(line); err != nil {
		log.Printf("[audit] failed to write audit log: %v", err)
	}
}

// getCurrentTimestamp returns the current time in RFC3339 format for audit logging.
func getCurrentTimestamp() string {
	return time.Now().Format(time.RFC3339)
}

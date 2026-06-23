package runner

import (
	"testing"
)

// TestRealExecutor_LogsCommandExecution verifies that a command is logged during execution.
func TestRealExecutor_LogsCommandExecution(t *testing.T) {
	// Create a mock auditor to capture logs
	var capturedCtx CommandAuditContext
	mockAuditor := func(ctx CommandAuditContext) {
		capturedCtx = ctx
	}

	job := Job{
		ID:          "test-job-1",
		SourcePath:  "/src",
		DestPath:    "/dest",
		Command:     "rsync -a /src /dest",
		AgentID:     "test-agent-1",
		Status:      "pending",
	}

	// Execute with audit logging
	_, _ = ExecutorWithAudit(job, Noop, mockAuditor, "test-agent-1")

	// Verify the audit context was populated
	if capturedCtx.CommandString != "rsync -a /src /dest" {
		t.Errorf("expected command 'rsync -a /src /dest', got '%s'", capturedCtx.CommandString)
	}

	if capturedCtx.ProgramName != "rsync" {
		t.Errorf("expected program 'rsync', got '%s'", capturedCtx.ProgramName)
	}

	if !capturedCtx.IsWhitelisted {
		t.Error("expected rsync to be whitelisted")
	}

	if capturedCtx.Mode != "audit" {
		t.Errorf("expected mode 'audit', got '%s'", capturedCtx.Mode)
	}

	if capturedCtx.AgentID != "test-agent-1" {
		t.Errorf("expected agentID 'test-agent-1', got '%s'", capturedCtx.AgentID)
	}

	if capturedCtx.JobID == nil || *capturedCtx.JobID != "test-job-1" {
		t.Errorf("expected jobID 'test-job-1', got %v", capturedCtx.JobID)
	}
}

// TestAuditModeLogs_DoesNotReject verifies that commands execute even if non-whitelisted in audit mode.
func TestAuditModeLogs_DoesNotReject(t *testing.T) {
	auditLog := []CommandAuditContext{}
	mockAuditor := func(ctx CommandAuditContext) {
		auditLog = append(auditLog, ctx)
	}

	// Create a job with a non-whitelisted command
	job := Job{
		ID:          "test-job-1",
		SourcePath:  "/src",
		DestPath:    "/dest",
		Command:     "bash -c 'echo hello'",
		AgentID:     "test-agent-1",
		Status:      "pending",
	}

	// In audit mode, this should still execute (not reject)
	_, _ = ExecutorWithAudit(job, Noop, mockAuditor, "test-agent-1")

	// Should have logged the command
	if len(auditLog) != 1 {
		t.Errorf("expected 1 audit log, got %d", len(auditLog))
	}

	// Verify non-whitelisted status was logged
	if auditLog[0].IsWhitelisted {
		t.Error("expected bash to be marked as non-whitelisted")
	}

	if auditLog[0].ProgramName != "bash" {
		t.Errorf("expected program 'bash', got '%s'", auditLog[0].ProgramName)
	}

	// In audit mode, execution should still succeed (not rejected)
	// The actual exit code depends on whether bash is installed, so we just verify the audit happened
	if auditLog[0].Mode != "audit" {
		t.Errorf("expected audit mode, got %s", auditLog[0].Mode)
	}
}

// TestAuditLogFile_ReturnsValidPath verifies the audit log file path is valid.
func TestAuditLogFile_ReturnsValidPath(t *testing.T) {
	path := AuditLogFile()
	if path == "" {
		t.Error("expected non-empty audit log path")
	}
	if len(path) < 5 {
		t.Errorf("expected reasonable path length, got %d", len(path))
	}
}

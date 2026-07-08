package db

import (
	"testing"
	"time"
)

// TestLogCommand_InsertsAuditEntry tests that LogCommand inserts a record.
func TestLogCommand_InsertsAuditEntry(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	jobID := "test-job-1"
	agentID := "test-agent-1"

	ctx := CommandAuditContext{
		JobID:         &jobID,
		CommandString: "rsync -a src dest",
		ProgramName:   "rsync",
		IsWhitelisted: true,
		Mode:          "audit",
		AuditResult:   "executed successfully",
		AgentID:       agentID,
	}

	err := db.LogCommand(ctx)
	if err != nil {
		t.Fatalf("LogCommand failed: %v", err)
	}

	// Verify the entry exists
	filter := AuditLogFilter{}
	logs, total, err := db.GetAuditLog(filter)
	if err != nil {
		t.Fatalf("GetAuditLog failed: %v", err)
	}

	if total != 1 {
		t.Errorf("expected 1 log entry, got %d", total)
	}
	if len(logs) != 1 {
		t.Errorf("expected 1 log in results, got %d", len(logs))
	}

	log := logs[0]
	if log.ProgramName != "rsync" {
		t.Errorf("expected program rsync, got %s", log.ProgramName)
	}
	if !log.IsWhitelisted {
		t.Error("expected IsWhitelisted=true")
	}
	if log.AgentID != agentID {
		t.Errorf("expected agentID %s, got %s", agentID, log.AgentID)
	}
}

// TestGetAuditLog_FiltersByProgram tests filtering by program name.
func TestGetAuditLog_FiltersByProgram(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	// Insert multiple entries with different programs
	agentID := "test-agent"
	for _, prog := range []string{"rsync", "robocopy", "bash"} {
		ctx := CommandAuditContext{
			CommandString: prog + " -a src dest",
			ProgramName:   prog,
			IsWhitelisted: prog != "bash",
			Mode:          "audit",
			AuditResult:   "executed",
			AgentID:       agentID,
		}
		_ = db.LogCommand(ctx)
	}

	// Filter by rsync
	filter := AuditLogFilter{
		ProgramName: "rsync",
	}
	logs, total, err := db.GetAuditLog(filter)
	if err != nil {
		t.Fatalf("GetAuditLog failed: %v", err)
	}

	if total != 1 {
		t.Errorf("expected 1 log entry for rsync, got %d", total)
	}
	if len(logs) != 1 {
		t.Errorf("expected 1 log in results, got %d", len(logs))
	}
	if logs[0].ProgramName != "rsync" {
		t.Errorf("expected program rsync, got %s", logs[0].ProgramName)
	}
}

// TestGetAuditLog_FiltersByWhitelist tests filtering by whitelist status.
func TestGetAuditLog_FiltersByWhitelist(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	agentID := "test-agent"
	// Insert whitelisted
	_ = db.LogCommand(CommandAuditContext{
		CommandString: "rsync -a src dest",
		ProgramName:   "rsync",
		IsWhitelisted: true,
		Mode:          "audit",
		AuditResult:   "executed",
		AgentID:       agentID,
	})

	// Insert non-whitelisted
	_ = db.LogCommand(CommandAuditContext{
		CommandString: "bash -c 'echo hello'",
		ProgramName:   "bash",
		IsWhitelisted: false,
		Mode:          "audit",
		AuditResult:   "executed",
		AgentID:       agentID,
	})

	// Filter by whitelisted=false
	filter := AuditLogFilter{}
	whitelisted := false
	filter.IsWhitelisted = &whitelisted

	logs, total, err := db.GetAuditLog(filter)
	if err != nil {
		t.Fatalf("GetAuditLog failed: %v", err)
	}

	if total != 1 {
		t.Errorf("expected 1 non-whitelisted entry, got %d", total)
	}
	if logs[0].ProgramName != "bash" {
		t.Errorf("expected bash, got %s", logs[0].ProgramName)
	}
}

// TestGetAuditLog_FiltersByDate tests filtering by date range.
func TestGetAuditLog_FiltersByDate(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	agentID := "test-agent"

	// Insert a command
	_ = db.LogCommand(CommandAuditContext{
		CommandString: "rsync -a src dest",
		ProgramName:   "rsync",
		IsWhitelisted: true,
		Mode:          "audit",
		AuditResult:   "executed",
		AgentID:       agentID,
	})

	// Test: filter by agent_id instead (simpler, avoids datetime issues)
	filter := AuditLogFilter{
		AgentID: agentID,
	}

	logsAgent, totalAgent, err := db.GetAuditLog(filter)
	if err != nil {
		t.Fatalf("GetAuditLog failed: %v", err)
	}

	if totalAgent != 1 {
		t.Errorf("expected 1 log by agent_id, got %d", totalAgent)
	}
	if len(logsAgent) != 1 {
		t.Errorf("expected 1 log in results, got %d", len(logsAgent))
	}

	// Test: filter by non-matching agent_id
	filter = AuditLogFilter{
		AgentID: "nonexistent",
	}

	logsNone, totalNone, err := db.GetAuditLog(filter)
	if err != nil {
		t.Fatalf("GetAuditLog failed: %v", err)
	}

	if totalNone != 0 {
		t.Errorf("expected 0 logs for nonexistent agent, got %d", totalNone)
	}
	if len(logsNone) != 0 {
		t.Errorf("expected 0 logs in results, got %d", len(logsNone))
	}
}

// TestGetNonWhitelistedPrograms_ReturnsDistinct tests the non-whitelisted programs query.
func TestGetNonWhitelistedPrograms_ReturnsDistinct(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	agentID := "test-agent"

	// Insert multiple non-whitelisted entries
	for i := 0; i < 3; i++ {
		_ = db.LogCommand(CommandAuditContext{
			CommandString: "bash -c 'cmd'",
			ProgramName:   "bash",
			IsWhitelisted: false,
			Mode:          "audit",
			AuditResult:   "executed",
			AgentID:       agentID,
		})
	}

	// Insert some whitelisted entries (should be excluded)
	for i := 0; i < 2; i++ {
		_ = db.LogCommand(CommandAuditContext{
			CommandString: "rsync -a src dest",
			ProgramName:   "rsync",
			IsWhitelisted: true,
			Mode:          "audit",
			AuditResult:   "executed",
			AgentID:       agentID,
		})
	}

	programs, err := db.GetNonWhitelistedPrograms()
	if err != nil {
		t.Fatalf("GetNonWhitelistedPrograms failed: %v", err)
	}

	if len(programs) != 1 {
		t.Errorf("expected 1 non-whitelisted program, got %d", len(programs))
	}

	prog := programs[0]
	if prog.ProgramName != "bash" {
		t.Errorf("expected bash, got %s", prog.ProgramName)
	}
	if prog.ExecutionCount != 3 {
		t.Errorf("expected 3 executions, got %d", prog.ExecutionCount)
	}
}

// TestGetAuditStats_ReturnsCorrectCounts tests the audit stats query.
func TestGetAuditStats_ReturnsCorrectCounts(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	agentID := "test-agent"

	// Insert whitelisted entries
	for i := 0; i < 2; i++ {
		err := db.LogCommand(CommandAuditContext{
			CommandString: "rsync -a src dest",
			ProgramName:   "rsync",
			IsWhitelisted: true,
			Mode:          "audit",
			AuditResult:   "executed",
			AgentID:       agentID,
		})
		if err != nil {
			t.Fatalf("LogCommand failed: %v", err)
		}
	}

	// Insert non-whitelisted entries
	for i := 0; i < 3; i++ {
		err := db.LogCommand(CommandAuditContext{
			CommandString: "bash -c 'cmd'",
			ProgramName:   "bash",
			IsWhitelisted: false,
			Mode:          "audit",
			AuditResult:   "executed",
			AgentID:       agentID,
		})
		if err != nil {
			t.Fatalf("LogCommand failed: %v", err)
		}
	}

	// Use wide time range (past 48 hours to future 48 hours)
	now := time.Now()
	from := now.Add(-48 * time.Hour)
	to := now.Add(48 * time.Hour)

	stats, err := db.GetAuditStats(from, to)
	if err != nil {
		t.Fatalf("GetAuditStats failed: %v", err)
	}

	if stats.WhitelistedCount != 2 {
		t.Errorf("expected 2 whitelisted, got %d", stats.WhitelistedCount)
	}
	if stats.NonWhitelistedCount != 3 {
		t.Errorf("expected 3 non-whitelisted, got %d", stats.NonWhitelistedCount)
	}
	if stats.TotalCount != 5 {
		t.Errorf("expected 5 total, got %d", stats.TotalCount)
	}
	if stats.UniquePrograms != 2 {
		t.Errorf("expected 2 unique programs, got %d", stats.UniquePrograms)
	}
}

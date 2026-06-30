package db

import (
	"testing"
	"time"
)

// TestInsertAndListUserAuditLog inserts 3 entries and lists them without filters.
func TestInsertAndListUserAuditLog(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	actions := []string{"user.create", "user.delete", "user.update"}
	for _, action := range actions {
		err := db.InsertUserAuditLog(UserAuditLogContext{
			Action:    action,
			IPAddress: "127.0.0.1",
		})
		if err != nil {
			t.Fatalf("InsertUserAuditLog failed for %s: %v", action, err)
		}
	}

	logs, total, err := db.ListUserAuditLogs(UserAuditLogFilter{})
	if err != nil {
		t.Fatalf("ListUserAuditLogs failed: %v", err)
	}

	if total != 3 {
		t.Errorf("expected total 3, got %d", total)
	}
	if len(logs) != 3 {
		t.Errorf("expected 3 logs in results, got %d", len(logs))
	}

	// Verify DESC order: the last inserted should be first
	for i := 0; i < len(logs)-1; i++ {
		if logs[i].CreatedAt.Before(logs[i+1].CreatedAt) {
			t.Errorf("entries not in DESC order: log[%d].CreatedAt=%v < log[%d].CreatedAt=%v",
				i, logs[i].CreatedAt, i+1, logs[i+1].CreatedAt)
		}
	}
}

// TestListUserAuditLogWithActionFilter filters entries by action.
func TestListUserAuditLogWithActionFilter(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	entries := []UserAuditLogContext{
		{Action: "user.create", IPAddress: "127.0.0.1"},
		{Action: "job.create", IPAddress: "127.0.0.1"},
		{Action: "user.create", IPAddress: "127.0.0.1"},
	}
	for _, e := range entries {
		if err := db.InsertUserAuditLog(e); err != nil {
			t.Fatalf("InsertUserAuditLog failed: %v", err)
		}
	}

	logs, total, err := db.ListUserAuditLogs(UserAuditLogFilter{Action: "user.create"})
	if err != nil {
		t.Fatalf("ListUserAuditLogs failed: %v", err)
	}

	if total != 2 {
		t.Errorf("expected 2 entries for user.create, got %d", total)
	}
	if len(logs) != 2 {
		t.Errorf("expected 2 logs in results, got %d", len(logs))
	}
	for _, log := range logs {
		if log.Action != "user.create" {
			t.Errorf("expected action user.create, got %s", log.Action)
		}
	}
}

// TestListUserAuditLogWithDateFilter filters entries by date range.
func TestListUserAuditLogWithDateFilter(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	// SQLite CURRENT_TIMESTAMP stores UTC, so use UTC for filter bounds.
	nowUTC := time.Now().UTC()
	threeMinAgo := nowUTC.Add(-3 * time.Minute)
	twoMinLater := nowUTC.Add(2 * time.Minute)

	_ = db.InsertUserAuditLog(UserAuditLogContext{Action: "early.entry", IPAddress: "127.0.0.1"})
	_ = db.InsertUserAuditLog(UserAuditLogContext{Action: "mid.entry", IPAddress: "127.0.0.1"})
	_ = db.InsertUserAuditLog(UserAuditLogContext{Action: "late.entry", IPAddress: "127.0.0.1"})

	// All 3 should be in the range covering last few minutes
	logs, total, err := db.ListUserAuditLogs(UserAuditLogFilter{
		FromDate: &threeMinAgo,
		ToDate:   &twoMinLater,
	})
	if err != nil {
		t.Fatalf("ListUserAuditLogs failed: %v", err)
	}
	if total != 3 {
		t.Errorf("expected 3 entries in full range, got %d", total)
	}
	if len(logs) != 3 {
		t.Errorf("expected 3 logs, got %d", len(logs))
	}

	// Filter to an empty range (well before entries)
	twoDaysAgoUTC := nowUTC.Add(-48 * time.Hour)
	oneDayAgoUTC := nowUTC.Add(-24 * time.Hour)
	_, emptyTotal, err := db.ListUserAuditLogs(UserAuditLogFilter{
		FromDate: &twoDaysAgoUTC,
		ToDate:   &oneDayAgoUTC,
	})
	if err != nil {
		t.Fatalf("ListUserAuditLogs with empty range failed: %v", err)
	}
	if emptyTotal != 0 {
		t.Errorf("expected 0 entries in empty range, got %d", emptyTotal)
	}
}

// TestListUserAuditLogWithSuccessFilter filters entries by success status.
func TestListUserAuditLogWithSuccessFilter(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	successTrue := true
	successFalse := false

	entries := []UserAuditLogContext{
		{Action: "user.login", IPAddress: "127.0.0.1", Success: true},
		{Action: "user.login", IPAddress: "127.0.0.1", Success: false},
		{Action: "user.login", IPAddress: "127.0.0.1", Success: true},
	}
	for _, e := range entries {
		if err := db.InsertUserAuditLog(e); err != nil {
			t.Fatalf("InsertUserAuditLog failed: %v", err)
		}
	}

	// Filter success=true
	logsTrue, totalTrue, err := db.ListUserAuditLogs(UserAuditLogFilter{Success: &successTrue})
	if err != nil {
		t.Fatalf("ListUserAuditLogs (success=true) failed: %v", err)
	}
	if totalTrue != 2 {
		t.Errorf("expected 2 successful entries, got %d", totalTrue)
	}
	if len(logsTrue) != 2 {
		t.Errorf("expected 2 logs, got %d", len(logsTrue))
	}
	for _, log := range logsTrue {
		if !log.Success {
			t.Error("expected success=true in filtered results")
		}
	}

	// Filter success=false
	logsFalse, totalFalse, err := db.ListUserAuditLogs(UserAuditLogFilter{Success: &successFalse})
	if err != nil {
		t.Fatalf("ListUserAuditLogs (success=false) failed: %v", err)
	}
	if totalFalse != 1 {
		t.Errorf("expected 1 failed entry, got %d", totalFalse)
	}
	if len(logsFalse) != 1 {
		t.Errorf("expected 1 log, got %d", len(logsFalse))
	}
	if logsFalse[0].Success {
		t.Error("expected success=false in filtered results")
	}
}

// TestListUserAuditLogPagination tests pagination with Limit and Offset.
func TestListUserAuditLogPagination(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	for i := 0; i < 25; i++ {
		if err := db.InsertUserAuditLog(UserAuditLogContext{
			Action:    "action",
			IPAddress: "127.0.0.1",
		}); err != nil {
			t.Fatalf("InsertUserAuditLog failed: %v", err)
		}
	}

	// Page 1: Limit=10, Offset=0
	logs1, total1, err := db.ListUserAuditLogs(UserAuditLogFilter{Limit: 10, Offset: 0})
	if err != nil {
		t.Fatalf("ListUserAuditLogs page 1 failed: %v", err)
	}
	if total1 != 25 {
		t.Errorf("expected total 25, got %d", total1)
	}
	if len(logs1) != 10 {
		t.Errorf("expected 10 logs on page 1, got %d", len(logs1))
	}

	// Page 2: Limit=10, Offset=10
	logs2, total2, err := db.ListUserAuditLogs(UserAuditLogFilter{Limit: 10, Offset: 10})
	if err != nil {
		t.Fatalf("ListUserAuditLogs page 2 failed: %v", err)
	}
	if total2 != 25 {
		t.Errorf("expected total 25, got %d", total2)
	}
	if len(logs2) != 10 {
		t.Errorf("expected 10 logs on page 2, got %d", len(logs2))
	}

	// Page 3: Limit=10, Offset=20
	logs3, total3, err := db.ListUserAuditLogs(UserAuditLogFilter{Limit: 10, Offset: 20})
	if err != nil {
		t.Fatalf("ListUserAuditLogs page 3 failed: %v", err)
	}
	if total3 != 25 {
		t.Errorf("expected total 25, got %d", total3)
	}
	if len(logs3) != 5 {
		t.Errorf("expected 5 logs on page 3, got %d", len(logs3))
	}

	// Verify no ID overlap across pages
	seen := make(map[int]bool)
	for _, log := range append(append(logs1, logs2...), logs3...) {
		if seen[log.ID] {
			t.Errorf("duplicate ID %d across pages", log.ID)
		}
		seen[log.ID] = true
	}
}

// TestInsertUserAuditLogEmpty verifies inserting with minimal fields succeeds.
func TestInsertUserAuditLogEmpty(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	err := db.InsertUserAuditLog(UserAuditLogContext{
		Action:    "minimal.action",
		IPAddress: "10.0.0.1",
	})
	if err != nil {
		t.Fatalf("InsertUserAuditLog with minimal fields failed: %v", err)
	}

	logs, total, err := db.ListUserAuditLogs(UserAuditLogFilter{})
	if err != nil {
		t.Fatalf("ListUserAuditLogs failed: %v", err)
	}
	if total != 1 {
		t.Errorf("expected 1 entry, got %d", total)
	}
	if len(logs) != 1 {
		t.Fatalf("expected 1 log in results, got %d", len(logs))
	}
	if logs[0].ID <= 0 {
		t.Errorf("expected ID > 0, got %d", logs[0].ID)
	}
	if logs[0].Action != "minimal.action" {
		t.Errorf("expected action minimal.action, got %s", logs[0].Action)
	}
	if logs[0].IPAddress != "10.0.0.1" {
		t.Errorf("expected IP 10.0.0.1, got %s", logs[0].IPAddress)
	}
}

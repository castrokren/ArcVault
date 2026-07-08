package business

import (
	"strings"
	"testing"
	"time"

	"arcvault/coordinator/db"
)

func fakeGroup(id int, name string) *db.AgentGroup {
	return &db.AgentGroup{ID: id, Name: name, CreatedAt: time.Now()}
}

func TestCreateJob_missingFieldsReturnsError(t *testing.T) {
	svc := NewJobService(newMockJobDB())

	cases := []struct {
		name, src, dest, wantErr string
	}{
		{"", "C:\\src", "D:\\dst", "name is required"},
		{"backup", "", "D:\\dst", "source_path is required"},
		{"backup", "C:\\src", "", "dest_path is required"},
	}

	for _, tc := range cases {
		_, err := svc.CreateJob("agent-01", tc.name, tc.src, tc.dest, nil, nil)
		if err == nil || err.Error() != tc.wantErr {
			t.Errorf("CreateJob(%q,%q,%q): expected %q, got %v", tc.name, tc.src, tc.dest, tc.wantErr, err)
		}
	}
}

func TestCreateJob_success(t *testing.T) {
	mock := newMockJobDB()
	svc := NewJobService(mock)

	job, err := svc.CreateJob("agent-01", "nightly-backup", "C:\\src", "D:\\dst", nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if job.AgentID != "agent-01" {
		t.Errorf("expected AgentID 'agent-01', got %q", job.AgentID)
	}
	if job.Name != "nightly-backup" {
		t.Errorf("expected Name 'nightly-backup', got %q", job.Name)
	}
	if job.Status != "pending" {
		t.Errorf("expected Status 'pending', got %q", job.Status)
	}
	if job.ID == "" {
		t.Error("expected non-empty ID")
	}
	if _, ok := mock.jobs[job.ID]; !ok {
		t.Error("expected job to be stored in DB")
	}
}

func TestCreateJob_syncFlagsSerialised(t *testing.T) {
	mock := newMockJobDB()
	svc := NewJobService(mock)

	flags := map[string]interface{}{"delete": true, "checksum": false}
	job, err := svc.CreateJob("agent-01", "backup", "C:\\src", "D:\\dst", nil, flags)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if job.SyncFlags == nil {
		t.Fatal("expected SyncFlags in returned DTO")
	}
	stored := mock.jobs[job.ID]
	if stored.SyncFlags == nil || !strings.Contains(*stored.SyncFlags, "delete") {
		t.Error("expected sync_flags JSON stored in DB")
	}
}

func TestGetJob_notFound(t *testing.T) {
	svc := NewJobService(newMockJobDB())

	_, err := svc.GetJob("does-not-exist")
	if err == nil {
		t.Fatal("expected error for unknown job")
	}
}

func TestGetJob_found(t *testing.T) {
	mock := newMockJobDB()
	svc := NewJobService(mock)

	created, _ := svc.CreateJob("agent-01", "backup", "C:\\src", "D:\\dst", nil, nil)

	got, err := svc.GetJob(created.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.ID != created.ID {
		t.Errorf("expected ID %q, got %q", created.ID, got.ID)
	}
	if got.Name != "backup" {
		t.Errorf("expected Name 'backup', got %q", got.Name)
	}
}

func TestCreateJobForGroup_validationErrors(t *testing.T) {
	svc := NewJobService(newMockJobDB())

	cases := []struct {
		name, src, dest, wantErr string
	}{
		{"", "C:\\src", "D:\\dst", "name is required"},
		{"backup", "", "D:\\dst", "source_path is required"},
		{"backup", "C:\\src", "", "dest_path is required"},
	}

	for _, tc := range cases {
		_, err := svc.CreateJobForGroup(1, tc.name, tc.src, tc.dest, nil, nil)
		if err == nil || err.Error() != tc.wantErr {
			t.Errorf("CreateJobForGroup: expected %q, got %v", tc.wantErr, err)
		}
	}
}

func TestCreateJobForGroup_groupNotFound(t *testing.T) {
	svc := NewJobService(newMockJobDB())

	_, err := svc.CreateJobForGroup(99, "backup", "C:\\src", "D:\\dst", nil, nil)
	if err == nil || err.Error() != "group not found" {
		t.Errorf("expected 'group not found', got %v", err)
	}
}

func TestCreateJobForGroup_emptyGroupReturnsError(t *testing.T) {
	mock := newMockJobDB()
	mock.groups[1] = fakeGroup(1, "empty-team")
	// no members added
	svc := NewJobService(mock)

	_, err := svc.CreateJobForGroup(1, "backup", "C:\\src", "D:\\dst", nil, nil)
	if err == nil || err.Error() != "group has no members" {
		t.Errorf("expected 'group has no members', got %v", err)
	}
}

func TestCreateJobForGroup_createsOneJobPerMember(t *testing.T) {
	mock := newMockJobDB()
	mock.groups[1] = fakeGroup(1, "team")
	mock.members[1] = []string{"agent-a", "agent-b", "agent-c"}
	svc := NewJobService(mock)

	resp, err := svc.CreateJobForGroup(1, "backup", "C:\\src", "D:\\dst", nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.Jobs) != 3 {
		t.Errorf("expected 3 jobs, got %d", len(resp.Jobs))
	}
	if resp.DispatchID == "" {
		t.Error("expected non-empty DispatchID")
	}
	// Each job should target a different agent
	agentIDs := map[string]bool{}
	for _, j := range resp.Jobs {
		agentIDs[j.AgentID] = true
	}
	if len(agentIDs) != 3 {
		t.Errorf("expected jobs for 3 distinct agents, got %d", len(agentIDs))
	}
}

func TestPostJobResults_jobNotFound(t *testing.T) {
	svc := NewJobService(newMockJobDB())

	_, err := svc.PostJobResults("nonexistent", 0, "output", "", "")
	if err == nil || err.Error() != "job not found" {
		t.Errorf("expected 'job not found', got %v", err)
	}
}

func TestPostJobResults_updatesExistingRun(t *testing.T) {
	mock := newMockJobDB()
	svc := NewJobService(mock)

	created, _ := svc.CreateJob("agent-01", "backup", "C:\\src", "D:\\dst", nil, nil)
	mock.firstRunID = "run-existing-001" // simulate trigger-created run

	result, err := svc.PostJobResults(created.ID, 0, "all done", "2026-01-01T00:00:00Z", "2026-01-01T01:00:00Z")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.JobName != "backup" {
		t.Errorf("expected JobName 'backup', got %q", result.JobName)
	}
	if result.AgentID != "agent-01" {
		t.Errorf("expected AgentID 'agent-01', got %q", result.AgentID)
	}
}

func TestPostJobResults_createsRunWhenNoneExists(t *testing.T) {
	mock := newMockJobDB()
	svc := NewJobService(mock)

	created, _ := svc.CreateJob("agent-01", "backup", "C:\\src", "D:\\dst", nil, nil)
	mock.firstRunID = "" // no trigger-created run

	_, err := svc.PostJobResults(created.ID, 1, "failed", "", "2026-01-01T01:00:00Z")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(mock.runs[created.ID]) != 1 {
		t.Errorf("expected 1 run to be created, got %d", len(mock.runs[created.ID]))
	}
}

func TestListJobs_paginationMath(t *testing.T) {
	mock := newMockJobDB()
	svc := NewJobService(mock)

	for i := 0; i < 11; i++ {
		svc.CreateJob("agent-01", "backup", "C:\\src", "D:\\dst", nil, nil)
	}

	result, err := svc.ListJobs("", "", "", 5, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Total != 11 {
		t.Errorf("expected Total=11, got %d", result.Total)
	}
	// ceil(11/5) = 3
	if result.Pages != 3 {
		t.Errorf("expected Pages=3, got %d", result.Pages)
	}
}

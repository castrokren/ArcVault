package server

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// --- DELETE /api/agents/{id} ---

func TestDeleteAgent_deletesExistingAgent(t *testing.T) {
	s := newTestServer(t)
	registerTestAgent(t, s, "agent-del-01", "linux", "amd64")

	req := httptest.NewRequest(http.MethodDelete, "/api/agents/agent-del-01", nil)
	req.Header.Set("Authorization", authHeader())
	rr := httptest.NewRecorder()
	s.router.ServeHTTP(rr, req)

	if rr.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d: %s", rr.Code, rr.Body.String())
	}

	// Verify agent is gone from the DB.
	var count int
	s.db.Conn().QueryRow(`SELECT COUNT(*) FROM agents WHERE id = 'agent-del-01'`).Scan(&count)
	if count != 0 {
		t.Error("expected agent to be deleted from DB")
	}
}

func TestDeleteAgent_returnsNotFoundForMissingAgent(t *testing.T) {
	s := newTestServer(t)

	req := httptest.NewRequest(http.MethodDelete, "/api/agents/does-not-exist", nil)
	req.Header.Set("Authorization", authHeader())
	rr := httptest.NewRecorder()
	s.router.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rr.Code)
	}
}

func TestDeleteAgent_blocksIfAgentHasRunningJob(t *testing.T) {
	s := newTestServer(t)
	registerTestAgent(t, s, "agent-busy-01", "linux", "amd64")

	// Seed a running job directly in the DB.
	_, err := s.db.Conn().Exec(`
		INSERT INTO jobs (id, agent_id, name, source_path, dest_path, status)
		VALUES ('job-running-01', 'agent-busy-01', 'busy-backup', '/src', '/dest', 'running')
	`)
	if err != nil {
		t.Fatalf("failed to seed running job: %v", err)
	}

	req := httptest.NewRequest(http.MethodDelete, "/api/agents/agent-busy-01", nil)
	req.Header.Set("Authorization", authHeader())
	rr := httptest.NewRecorder()
	s.router.ServeHTTP(rr, req)

	if rr.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestDeleteAgent_cleansUpTokensOnDelete(t *testing.T) {
	s := newTestServer(t)
	registerTestAgent(t, s, "agent-token-01", "linux", "amd64")

	// Create a token for this agent.
	_, err := s.db.CreateAgentToken("agent-token-01")
	if err != nil {
		t.Fatalf("failed to create agent token: %v", err)
	}

	// Verify token exists before delete.
	var tokenCount int
	s.db.Conn().QueryRow(`SELECT COUNT(*) FROM tokens WHERE agent_id = 'agent-token-01'`).Scan(&tokenCount)
	if tokenCount == 0 {
		t.Fatal("expected token to exist before delete")
	}

	req := httptest.NewRequest(http.MethodDelete, "/api/agents/agent-token-01", nil)
	req.Header.Set("Authorization", authHeader())
	rr := httptest.NewRecorder()
	s.router.ServeHTTP(rr, req)

	if rr.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d: %s", rr.Code, rr.Body.String())
	}

	// Verify token is gone.
	s.db.Conn().QueryRow(`SELECT COUNT(*) FROM tokens WHERE agent_id = 'agent-token-01'`).Scan(&tokenCount)
	if tokenCount != 0 {
		t.Error("expected agent token to be deleted")
	}
}

func TestDeleteAgent_cleansUpGroupMembershipsOnDelete(t *testing.T) {
	s := newTestServer(t)
	registerTestAgent(t, s, "agent-grp-01", "linux", "amd64")

	// Create a group and add agent to it.
	group, err := s.db.CreateGroup("test-group", "")
	if err != nil {
		t.Fatalf("failed to create group: %v", err)
	}
	if err := s.db.AddAgentToGroup(group.ID, "agent-grp-01"); err != nil {
		t.Fatalf("failed to add agent to group: %v", err)
	}

	// Verify membership before delete.
	var memberCount int
	s.db.Conn().QueryRow(
		`SELECT COUNT(*) FROM agent_group_members WHERE agent_id = 'agent-grp-01'`,
	).Scan(&memberCount)
	if memberCount == 0 {
		t.Fatal("expected group membership to exist before delete")
	}

	req := httptest.NewRequest(http.MethodDelete, "/api/agents/agent-grp-01", nil)
	req.Header.Set("Authorization", authHeader())
	rr := httptest.NewRecorder()
	s.router.ServeHTTP(rr, req)

	if rr.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d: %s", rr.Code, rr.Body.String())
	}

	// Verify membership is gone.
	s.db.Conn().QueryRow(
		`SELECT COUNT(*) FROM agent_group_members WHERE agent_id = 'agent-grp-01'`,
	).Scan(&memberCount)
	if memberCount != 0 {
		t.Error("expected group membership to be deleted")
	}
}

func TestDeleteAgent_preservesHistoricalJobs(t *testing.T) {
	s := newTestServer(t)
	registerTestAgent(t, s, "agent-hist-01", "linux", "amd64")

	// Seed a completed job (not running — should NOT block deletion).
	_, err := s.db.Conn().Exec(`
		INSERT INTO jobs (id, agent_id, name, source_path, dest_path, status)
		VALUES ('job-done-01', 'agent-hist-01', 'old-backup', '/src', '/dest', 'success')
	`)
	if err != nil {
		t.Fatalf("failed to seed historical job: %v", err)
	}

	req := httptest.NewRequest(http.MethodDelete, "/api/agents/agent-hist-01", nil)
	req.Header.Set("Authorization", authHeader())
	rr := httptest.NewRecorder()
	s.router.ServeHTTP(rr, req)

	if rr.Code != http.StatusNoContent {
		t.Fatalf("expected 204 (historical jobs should not block), got %d: %s", rr.Code, rr.Body.String())
	}

	// Verify job is still in DB (preserved).
	var jobCount int
	s.db.Conn().QueryRow(`SELECT COUNT(*) FROM jobs WHERE id = 'job-done-01'`).Scan(&jobCount)
	if jobCount == 0 {
		t.Error("expected historical job to be preserved after agent delete")
	}
}

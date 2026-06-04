package business

import (
	"testing"

	"arcvault/coordinator/db"
)

func newFakeAgent(id string) db.Agent {
	return db.Agent{ID: id, Hostname: "host-" + id, OS: "linux", Status: "online", RegisteredAt: "2026-01-01T00:00:00Z"}
}

func TestRegisterAgent_missingFieldsReturnsError(t *testing.T) {
	svc := NewAgentService(newMockAgentQueries())

	cases := []struct {
		agentID, hostname, os, version string
	}{
		{"", "host", "linux", "1.0"},
		{"id", "", "linux", "1.0"},
		{"id", "host", "", "1.0"},
		{"id", "host", "linux", ""},
	}

	for _, tc := range cases {
		_, err := svc.RegisterAgent(tc.agentID, tc.hostname, tc.os, "amd64", tc.version, "c1")
		if err == nil {
			t.Errorf("expected error for agentID=%q hostname=%q os=%q version=%q", tc.agentID, tc.hostname, tc.os, tc.version)
		}
	}
}

func TestRegisterAgent_success(t *testing.T) {
	svc := NewAgentService(newMockAgentQueries())

	agent, err := svc.RegisterAgent("agent-01", "web-server", "linux", "amd64", "1.0.0", "coord-01")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if agent.ID != "agent-01" {
		t.Errorf("expected ID 'agent-01', got %q", agent.ID)
	}
	if agent.Hostname != "web-server" {
		t.Errorf("expected hostname 'web-server', got %q", agent.Hostname)
	}
	if agent.Status != "online" {
		t.Errorf("expected status 'online', got %q", agent.Status)
	}
}

func TestDeleteAgent_notFoundReturnsError(t *testing.T) {
	svc := NewAgentService(newMockAgentQueries())

	err := svc.DeleteAgent("does-not-exist")
	if err == nil || err.Error() != "agent not found" {
		t.Errorf("expected 'agent not found', got %v", err)
	}
}

func TestDeleteAgent_blockedByRunningJobs(t *testing.T) {
	mock := newMockAgentQueries()
	mock.agents["agent-01"] = newFakeAgent("agent-01")
	mock.runningJobsMap["agent-01"] = 2
	svc := NewAgentService(mock)

	err := svc.DeleteAgent("agent-01")
	if err == nil {
		t.Fatal("expected error for agent with running jobs")
	}
	expected := "agent has 2 running jobs — stop all jobs before deleting"
	if err.Error() != expected {
		t.Errorf("expected %q, got %q", expected, err.Error())
	}
}

func TestDeleteAgent_success(t *testing.T) {
	mock := newMockAgentQueries()
	mock.agents["agent-01"] = newFakeAgent("agent-01")
	svc := NewAgentService(mock)

	if err := svc.DeleteAgent("agent-01"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, exists := mock.agents["agent-01"]; exists {
		t.Error("expected agent to be removed from store")
	}
}

func TestListAgents_paginationMath(t *testing.T) {
	mock := newMockAgentQueries()
	for i := 0; i < 7; i++ {
		id := "agent-" + string(rune('a'+i))
		mock.agents[id] = newFakeAgent(id)
	}
	svc := NewAgentService(mock)

	result, err := svc.ListAgents("", "", 3, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Total != 7 {
		t.Errorf("expected Total=7, got %d", result.Total)
	}
	// pages = ceil(7/3) = 3
	if result.Pages != 3 {
		t.Errorf("expected Pages=3, got %d", result.Pages)
	}
	if result.Page != 1 {
		t.Errorf("expected Page=1, got %d", result.Page)
	}
}

func TestListAgents_emptyResultStillOnePage(t *testing.T) {
	svc := NewAgentService(newMockAgentQueries())

	result, err := svc.ListAgents("", "", 10, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// 0 items with limit 10 → ceil(0/10) = 0, but service clamps to 1
	if result.Pages != 1 {
		t.Errorf("expected Pages=1 for empty result, got %d", result.Pages)
	}
}

package db

import (
	"testing"
)

// --- GetAgentIDByToken ---

func TestGetAgentIDByToken_validTokenReturnsAgentID(t *testing.T) {
	d := newTestDB(t)

	token, err := d.CreateAgentToken("agent-abc")
	if err != nil {
		t.Fatalf("CreateAgentToken: %v", err)
	}

	agentID, err := d.GetAgentIDByToken(token)
	if err != nil {
		t.Fatalf("GetAgentIDByToken: %v", err)
	}
	if agentID != "agent-abc" {
		t.Errorf("expected agent-abc, got %q", agentID)
	}
}

func TestGetAgentIDByToken_invalidTokenReturnsError(t *testing.T) {
	d := newTestDB(t)

	_, err := d.GetAgentIDByToken("does-not-exist")
	if err == nil {
		t.Fatal("expected error for invalid token, got nil")
	}
}

func TestGetAgentIDByToken_bootstrapTokenReturnsAgentID(t *testing.T) {
	d := newTestDB(t)

	token, err := d.CreateAgentToken("bootstrap-node-1")
	if err != nil {
		t.Fatalf("CreateAgentToken: %v", err)
	}

	agentID, err := d.GetAgentIDByToken(token)
	if err != nil {
		t.Fatalf("GetAgentIDByToken: %v", err)
	}
	if agentID != "bootstrap-node-1" {
		t.Errorf("expected bootstrap-node-1, got %q", agentID)
	}
}

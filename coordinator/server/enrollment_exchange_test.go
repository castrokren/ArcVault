package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// registerWith posts a registration as agentID using the given bearer token and
// returns the decoded response body.
func registerWith(t *testing.T, s *Server, agentID, bearer string) map[string]string {
	t.Helper()

	body := `{"agent_id":"` + agentID + `","hostname":"box","os":"windows","arch":"amd64","version":"v0.6.0"}`
	req := httptest.NewRequest(http.MethodPost, "/api/agents/register", bytes.NewBufferString(body))
	req.Header.Set("Authorization", bearer)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	s.router.ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("register %s: expected 201, got %d: %s", agentID, rr.Code, rr.Body.String())
	}

	var out map[string]string
	if err := json.Unmarshal(rr.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode register response: %v (body %s)", err, rr.Body.String())
	}
	return out
}

// bootstrap.ps1 writes the enrollment token into agent-config.yaml as the agent's
// permanent auth_token, but that token expires an hour after the script was
// generated and nothing else ever replaced it — so the agent stopped
// authenticating an hour after enrollment. Registration must hand back a
// long-lived per-agent token to take its place.
func TestRegister_exchangesEnrollmentTokenForPerAgentToken(t *testing.T) {
	s := newTestServer(t)

	enrollment, err := s.db.CreateAgentToken("bootstrap:HOST-A")
	if err != nil {
		t.Fatalf("mint enrollment token: %v", err)
	}

	resp := registerWith(t, s, "HOST-A", "Bearer "+enrollment)

	issued := resp["token"]
	if issued == "" {
		t.Fatal("no token returned — the agent would keep using the expiring enrollment token")
	}
	if issued == enrollment {
		t.Fatal("returned the same token; no exchange happened")
	}

	// The issued token must work, and must be bound to this agent (not to
	// "bootstrap:HOST-A"), so job credentials resolve to the right machine.
	if _, err := s.db.ValidateToken(issued); err != nil {
		t.Errorf("issued token does not validate: %v", err)
	}
	boundTo, err := s.db.GetAgentIDByToken(issued)
	if err != nil {
		t.Fatalf("GetAgentIDByToken: %v", err)
	}
	if boundTo != "HOST-A" {
		t.Errorf("issued token is bound to %q, want HOST-A", boundTo)
	}

	// The enrollment token is left to expire on its own: deleting it here would
	// brick the machine if this response never arrived.
	if _, err := s.db.ValidateToken(enrollment); err != nil {
		t.Errorf("enrollment token was revoked on exchange; a lost response would leave the agent with no credential: %v", err)
	}
}

// An agent that already holds a per-agent token must not be handed a new one on
// every registration — that would mint a fresh credential per agent restart, the
// sprawl that superseding was added to stop.
func TestRegister_noExchangeForExistingPerAgentToken(t *testing.T) {
	s := newTestServer(t)

	agentToken, err := s.db.CreateAgentToken("HOST-B")
	if err != nil {
		t.Fatalf("mint agent token: %v", err)
	}

	resp := registerWith(t, s, "HOST-B", "Bearer "+agentToken)

	if got := resp["token"]; got != "" {
		t.Errorf("unexpected token issued to an agent that already had one: %q", got)
	}
	if _, err := s.db.ValidateToken(agentToken); err != nil {
		t.Errorf("the agent's existing token stopped working: %v", err)
	}
}

// Ops scripts register with the admin token. That is not an enrollment, so it
// must not mint a per-agent credential as a side effect.
func TestRegister_noExchangeForAdminToken(t *testing.T) {
	s := newTestServer(t)

	resp := registerWith(t, s, "HOST-C", machineAuthHeader())

	if got := resp["token"]; got != "" {
		t.Errorf("admin-token registration issued a per-agent token: %q", got)
	}
}

// Registration still reports success and keeps the documented fields.
func TestRegister_responseKeepsStatusAndAgentID(t *testing.T) {
	s := newTestServer(t)

	enrollment, err := s.db.CreateAgentToken("bootstrap")
	if err != nil {
		t.Fatalf("mint enrollment token: %v", err)
	}

	resp := registerWith(t, s, "HOST-D", "Bearer "+enrollment)

	if resp["status"] != "registered" {
		t.Errorf("status = %q, want registered", resp["status"])
	}
	if resp["agent_id"] != "HOST-D" {
		t.Errorf("agent_id = %q, want HOST-D", resp["agent_id"])
	}
}

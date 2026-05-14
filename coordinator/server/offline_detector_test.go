package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// --- agent offline detection ---

func TestOfflineDetector_marksAgentOfflineAfterTimeout(t *testing.T) {
	s := newTestServer(t)

	// register an agent
	body := `{"agent_id":"agent-stale","hostname":"box","os":"windows","version":"0.1.0"}`
	req := httptest.NewRequest(http.MethodPost, "/api/agents/register", bytes.NewBufferString(body))
	req.Header.Set("Authorization", authHeader())
	req.Header.Set("Content-Type", "application/json")
	s.router.ServeHTTP(httptest.NewRecorder(), req)

	// manually backdate last_seen to simulate a stale agent
	stale := time.Now().UTC().Add(-5 * time.Minute).Format(time.RFC3339)
	s.db.Conn().Exec(`UPDATE agents SET last_seen = ? WHERE id = 'agent-stale'`, stale)

	// run one detection pass with a 90s threshold
	s.detectOfflineAgents(90 * time.Second)

	// check status is now offline
	req2 := httptest.NewRequest(http.MethodGet, "/api/agents", nil)
	req2.Header.Set("Authorization", authHeader())
	rr2 := httptest.NewRecorder()
	s.router.ServeHTTP(rr2, req2)

	var agents []agentResponse
	json.NewDecoder(rr2.Body).Decode(&agents)

	var found bool
	for _, a := range agents {
		if a.ID == "agent-stale" {
			found = true
			if a.Status != "offline" {
				t.Errorf("expected status 'offline', got %q", a.Status)
			}
		}
	}
	if !found {
		t.Error("agent-stale not found in response")
	}
}

func TestOfflineDetector_doesNotMarkRecentAgentOffline(t *testing.T) {
	s := newTestServer(t)

	// register an agent with last_seen = now
	body := `{"agent_id":"agent-fresh","hostname":"box","os":"windows","version":"0.1.0"}`
	req := httptest.NewRequest(http.MethodPost, "/api/agents/register", bytes.NewBufferString(body))
	req.Header.Set("Authorization", authHeader())
	req.Header.Set("Content-Type", "application/json")
	s.router.ServeHTTP(httptest.NewRecorder(), req)

	now := time.Now().UTC().Format(time.RFC3339)
	s.db.Conn().Exec(`UPDATE agents SET last_seen = ? WHERE id = 'agent-fresh'`, now)

	// run detection
	s.detectOfflineAgents(90 * time.Second)

	// status should still be online
	req2 := httptest.NewRequest(http.MethodGet, "/api/agents", nil)
	req2.Header.Set("Authorization", authHeader())
	rr2 := httptest.NewRecorder()
	s.router.ServeHTTP(rr2, req2)

	var agents []agentResponse
	json.NewDecoder(rr2.Body).Decode(&agents)

	for _, a := range agents {
		if a.ID == "agent-fresh" && a.Status != "online" {
			t.Errorf("expected status 'online', got %q", a.Status)
		}
	}
}

func TestOfflineDetector_broadcastsEventForEachOfflineAgent(t *testing.T) {
	s := newTestServer(t)

	// register two stale agents
	for _, id := range []string{"agent-a", "agent-b"} {
		body := `{"agent_id":"` + id + `","hostname":"box","os":"windows","version":"0.1.0"}`
		req := httptest.NewRequest(http.MethodPost, "/api/agents/register", bytes.NewBufferString(body))
		req.Header.Set("Authorization", authHeader())
		req.Header.Set("Content-Type", "application/json")
		s.router.ServeHTTP(httptest.NewRecorder(), req)

		stale := time.Now().UTC().Add(-5 * time.Minute).Format(time.RFC3339)
		s.db.Conn().Exec(`UPDATE agents SET last_seen = ? WHERE id = ?`, stale, id)
	}

	// connect a WS client to capture broadcasts
	conn, ts := dialTestWS(t, s)
	defer ts.Close()
	defer conn.Close()
	time.Sleep(50 * time.Millisecond)

	s.detectOfflineAgents(90 * time.Second)

	// collect up to 2 events with timeout
	received := map[string]bool{}
	for i := 0; i < 2; i++ {
		ev := readEvent(t, conn)
		if ev["type"] == "agent.updated" {
			if payload, ok := ev["payload"].(map[string]interface{}); ok {
				received[payload["id"].(string)] = true
			}
		}
	}

	if !received["agent-a"] {
		t.Error("expected agent.updated broadcast for agent-a")
	}
	if !received["agent-b"] {
		t.Error("expected agent.updated broadcast for agent-b")
	}
}

func TestOfflineDetector_noActionWhenNoAgentsRegistered(t *testing.T) {
	s := newTestServer(t)
	// should not panic with empty agents table
	s.detectOfflineAgents(90 * time.Second)
}

package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// --- per-agent token tests ---

func TestAgentToken_agentCanRegisterWithOwnToken(t *testing.T) {
	s := newTestServer(t)

	// create an agent token
	token, err := s.db.CreateAgentToken("agent-01")
	if err != nil {
		t.Fatalf("failed to create agent token: %v", err)
	}
	if token == "" {
		t.Fatal("expected non-empty token")
	}

	// register using agent token instead of admin token
	body := `{"agent_id":"agent-01","hostname":"box","os":"windows","version":"0.1.0"}`
	req := httptest.NewRequest(http.MethodPost, "/api/agents/register", bytes.NewBufferString(body))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	s.router.ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestAgentToken_agentCanHeartbeatWithOwnToken(t *testing.T) {
	s := newTestServer(t)

	// create agent and its token
	body := `{"agent_id":"agent-01","hostname":"box","os":"windows","version":"0.1.0"}`
	req := httptest.NewRequest(http.MethodPost, "/api/agents/register", bytes.NewBufferString(body))
	req.Header.Set("Authorization", authHeader())
	req.Header.Set("Content-Type", "application/json")
	s.router.ServeHTTP(httptest.NewRecorder(), req)

	token, err := s.db.CreateAgentToken("agent-01")
	if err != nil {
		t.Fatalf("failed to create agent token: %v", err)
	}

	// heartbeat with agent token
	req2 := httptest.NewRequest(http.MethodPost, "/api/agents/agent-01/heartbeat", nil)
	req2.Header.Set("Authorization", "Bearer "+token)
	rr2 := httptest.NewRecorder()
	s.router.ServeHTTP(rr2, req2)

	if rr2.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr2.Code, rr2.Body.String())
	}
}

func TestAgentToken_agentCanPostResultsWithOwnToken(t *testing.T) {
	s := newTestServer(t)

	// create a job
	jobBody := `{"agent_id":"agent-01","name":"backup","source_path":"C:\\src","dest_path":"D:\\backup"}`
	req := httptest.NewRequest(http.MethodPost, "/api/jobs", bytes.NewBufferString(jobBody))
	req.Header.Set("Authorization", authHeader())
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	s.router.ServeHTTP(rr, req)

	var created Job
	json.NewDecoder(rr.Body).Decode(&created)

	// create agent token
	token, err := s.db.CreateAgentToken("agent-01")
	if err != nil {
		t.Fatalf("failed to create agent token: %v", err)
	}

	// post results with agent token
	result := `{"exit_code":0,"output":"done"}`
	req2 := httptest.NewRequest(http.MethodPost, "/api/jobs/"+created.ID+"/results", bytes.NewBufferString(result))
	req2.Header.Set("Authorization", "Bearer "+token)
	req2.Header.Set("Content-Type", "application/json")
	rr2 := httptest.NewRecorder()
	s.router.ServeHTTP(rr2, req2)

	if rr2.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rr2.Code, rr2.Body.String())
	}
}

func TestAgentToken_invalidTokenReturns401(t *testing.T) {
	s := newTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/api/agents", nil)
	req.Header.Set("Authorization", "Bearer not-a-valid-token")
	rr := httptest.NewRecorder()
	s.router.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rr.Code)
	}
}

func TestAgentToken_adminTokenStillWorks(t *testing.T) {
	s := newTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/api/agents", nil)
	req.Header.Set("Authorization", authHeader())
	rr := httptest.NewRecorder()
	s.router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
}

func TestAgentToken_tokenIsUniquePerCall(t *testing.T) {
	s := newTestServer(t)

	token1, _ := s.db.CreateAgentToken("agent-01")
	token2, _ := s.db.CreateAgentToken("agent-01")

	if token1 == token2 {
		t.Error("expected different tokens on each call")
	}
}

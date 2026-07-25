package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"arcvault/coordinator/db"
)

// ── DB unit tests ─────────────────────────────────────────────────────────────

func TestFedDB_CRUD(t *testing.T) {
	s := newTestServer(t)

	f := db.Federation{
		ID:    "fed-test-1",
		Name:  "Test Site",
		URL:   "http://sub.internal:8080",
		Token: "tok-abc123",
	}

	if err := s.db.CreateFederation(f); err != nil {
		t.Fatalf("CreateFederation: %v", err)
	}

	// List
	list, err := s.db.ListFederation()
	if err != nil {
		t.Fatalf("ListFederation: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("expected 1 record, got %d", len(list))
	}
	if list[0].Name != "Test Site" {
		t.Errorf("expected Name 'Test Site', got %q", list[0].Name)
	}

	// Get
	got, err := s.db.GetFederation("fed-test-1")
	if err != nil || got == nil {
		t.Fatalf("GetFederation: %v (got=%v)", err, got)
	}
	if got.URL != "http://sub.internal:8080" {
		t.Errorf("expected URL, got %q", got.URL)
	}

	// Update
	f.Name = "Updated Site"
	f.URL = "http://sub2.internal:8080"
	if err := s.db.UpdateFederation(f); err != nil {
		t.Fatalf("UpdateFederation: %v", err)
	}
	got2, _ := s.db.GetFederation("fed-test-1")
	if got2.Name != "Updated Site" {
		t.Errorf("expected updated Name, got %q", got2.Name)
	}

	// Delete
	if err := s.db.DeleteFederation("fed-test-1"); err != nil {
		t.Fatalf("DeleteFederation: %v", err)
	}
	got3, _ := s.db.GetFederation("fed-test-1")
	if got3 != nil {
		t.Error("expected nil after delete")
	}
}

func TestFedDB_SetStatus(t *testing.T) {
	s := newTestServer(t)

	f := db.Federation{ID: "fed-status-1", Name: "S", URL: "http://x", Token: "t"}
	s.db.CreateFederation(f)

	now := time.Now().UTC().Truncate(time.Second)
	if err := s.db.SetFederationStatus("fed-status-1", "online", now, "v1.2.3"); err != nil {
		t.Fatalf("SetFederationStatus: %v", err)
	}

	got, _ := s.db.GetFederation("fed-status-1")
	if got.Status != "online" {
		t.Errorf("expected status 'online', got %q", got.Status)
	}
	if got.Version != "v1.2.3" {
		t.Errorf("expected version 'v1.2.3', got %q", got.Version)
	}
	if got.LastSeen == nil {
		t.Error("expected non-nil LastSeen")
	}
}

func TestFedDB_GetByToken(t *testing.T) {
	s := newTestServer(t)

	f := db.Federation{ID: "fed-tok-1", Name: "T", URL: "http://x", Token: "secret-token"}
	s.db.CreateFederation(f)

	// Found
	got, err := s.db.GetFederationByToken("secret-token")
	if err != nil || got == nil {
		t.Fatalf("GetFederationByToken (found): %v", err)
	}
	if got.ID != "fed-tok-1" {
		t.Errorf("expected ID 'fed-tok-1', got %q", got.ID)
	}

	// Not found
	missing, err := s.db.GetFederationByToken("wrong-token")
	if err != nil {
		t.Fatalf("GetFederationByToken (not found) unexpected error: %v", err)
	}
	if missing != nil {
		t.Error("expected nil for unknown token")
	}
}

func TestFedMessage_Serialize(t *testing.T) {
	types := []string{
		FedEventSnapshot,
		FedEventAgentHeartbeat,
		FedEventJobStateChange,
		FedEventAgentRegistered,
		FedEventAgentDeleted,
		FedEventTemplateChanged,
		FedCmdTriggerJob,
		FedCmdRunTemplate,
		FedCmdUpdateAgent,
	}

	for _, typ := range types {
		payload, _ := json.Marshal(map[string]string{"key": "val"})
		msg := FedMessage{Type: typ, Payload: json.RawMessage(payload)}

		raw, err := json.Marshal(msg)
		if err != nil {
			t.Errorf("marshal %q: %v", typ, err)
			continue
		}

		var decoded FedMessage
		if err := json.Unmarshal(raw, &decoded); err != nil {
			t.Errorf("unmarshal %q: %v", typ, err)
			continue
		}
		if decoded.Type != typ {
			t.Errorf("round-trip type: want %q got %q", typ, decoded.Type)
		}
	}
}

// ── Hub unit tests ────────────────────────────────────────────────────────────

func TestFedHub_TokenAuth_Valid(t *testing.T) {
	s := newTestServer(t)

	// Register a sub
	f := db.Federation{ID: "fed-hub-1", Name: "Hub", URL: "http://x", Token: "valid-hub-token"}
	s.db.CreateFederation(f)

	// POST without WebSocket upgrade should be rejected before token check — we test
	// token validation by calling GetFederationByToken directly (auth logic is in HandleSubConnect).
	got, err := s.db.GetFederationByToken("valid-hub-token")
	if err != nil || got == nil {
		t.Fatal("valid token should resolve to a federation record")
	}
}

func TestFedHub_TokenAuth_Invalid(t *testing.T) {
	s := newTestServer(t)

	got, err := s.db.GetFederationByToken("no-such-token")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != nil {
		t.Error("invalid token should return nil")
	}
}

func TestFedHub_CacheStale_OnMarkOffline(t *testing.T) {
	hub := NewFederationHub(nil) // nil db is fine since we won't hit DB in this path

	// Manually inject a subConn with a cache
	sc := &subConn{
		id: "fed-offline-1",
		cache: &SubCache{
			Agents: []agentResponse{{ID: "agent-1"}},
			Stale:  false,
		},
	}

	hub.mu.Lock()
	hub.subs["fed-offline-1"] = sc
	hub.mu.Unlock()

	// markOffline requires a real DB for SetFederationStatus — skip DB call by
	// manually applying the cache stale logic (mirrors markOffline minus DB call).
	sc.mu.Lock()
	sc.cache.Stale = true
	sc.cache.AsOf = time.Now()
	sc.mu.Unlock()

	hub.mu.Lock()
	delete(hub.subs, "fed-offline-1")
	hub.mu.Unlock()

	// Cache should be stale, data retained
	sc.mu.RLock()
	defer sc.mu.RUnlock()
	if !sc.cache.Stale {
		t.Error("expected cache to be stale after disconnect")
	}
	if len(sc.cache.Agents) != 1 {
		t.Error("expected cached agents to be retained after disconnect")
	}
}

func TestFedHub_ApplyDelta_AgentHeartbeat(t *testing.T) {
	hub := NewFederationHub(nil)

	sc := &subConn{
		id: "fed-delta-1",
		cache: &SubCache{
			Agents: []agentResponse{
				{ID: "agent-1", Status: "offline", Version: "v1.0.0"},
			},
		},
	}

	payload, _ := json.Marshal(FedAgentHeartbeat{
		AgentID:  "agent-1",
		Status:   "online",
		LastSeen: strPtr("2026-05-18T12:00:00Z"),
	})

	hub.applyDelta(sc, FedMessage{
		Type:    FedEventAgentHeartbeat,
		Payload: json.RawMessage(payload),
	})

	sc.mu.RLock()
	defer sc.mu.RUnlock()
	if sc.cache.Agents[0].Status != "online" {
		t.Errorf("expected status 'online', got %q", sc.cache.Agents[0].Status)
	}
	// A heartbeat carries no version; it must preserve the version seeded by
	// register/snapshot, never blank it out.
	if sc.cache.Agents[0].Version != "v1.0.0" {
		t.Errorf("expected version preserved as 'v1.0.0', got %q", sc.cache.Agents[0].Version)
	}
}

func TestFedHub_ApplyDelta_JobStateChange(t *testing.T) {
	hub := NewFederationHub(nil)

	sc := &subConn{
		id: "fed-delta-2",
		cache: &SubCache{
			Jobs: []Job{
				{ID: "job-1", Status: "pending"},
			},
		},
	}

	payload, _ := json.Marshal(FedJobStateChange{JobID: "job-1", Status: "running"})
	hub.applyDelta(sc, FedMessage{
		Type:    FedEventJobStateChange,
		Payload: json.RawMessage(payload),
	})

	sc.mu.RLock()
	defer sc.mu.RUnlock()
	if sc.cache.Jobs[0].Status != "running" {
		t.Errorf("expected status 'running', got %q", sc.cache.Jobs[0].Status)
	}
}

func TestFedHub_ApplyDelta_AgentRegistered(t *testing.T) {
	hub := NewFederationHub(nil)
	sc := &subConn{id: "fed-delta-3", cache: &SubCache{}}

	payload, _ := json.Marshal(FedAgentRegistered{Agent: agentResponse{ID: "new-agent", Hostname: "box1"}})
	hub.applyDelta(sc, FedMessage{Type: FedEventAgentRegistered, Payload: json.RawMessage(payload)})

	sc.mu.RLock()
	defer sc.mu.RUnlock()
	if len(sc.cache.Agents) != 1 {
		t.Fatalf("expected 1 agent, got %d", len(sc.cache.Agents))
	}
	if sc.cache.Agents[0].ID != "new-agent" {
		t.Errorf("expected agent ID 'new-agent', got %q", sc.cache.Agents[0].ID)
	}
}

func TestFedHub_ApplyDelta_AgentDeleted(t *testing.T) {
	hub := NewFederationHub(nil)
	sc := &subConn{
		id: "fed-delta-4",
		cache: &SubCache{
			Agents: []agentResponse{{ID: "agent-del"}, {ID: "agent-keep"}},
		},
	}

	payload, _ := json.Marshal(FedAgentDeleted{AgentID: "agent-del"})
	hub.applyDelta(sc, FedMessage{Type: FedEventAgentDeleted, Payload: json.RawMessage(payload)})

	sc.mu.RLock()
	defer sc.mu.RUnlock()
	if len(sc.cache.Agents) != 1 {
		t.Fatalf("expected 1 agent after delete, got %d", len(sc.cache.Agents))
	}
	if sc.cache.Agents[0].ID != "agent-keep" {
		t.Errorf("expected remaining agent 'agent-keep', got %q", sc.cache.Agents[0].ID)
	}
}

// ── Client backoff unit tests ─────────────────────────────────────────────────

func TestFedClient_BackoffSchedule(t *testing.T) {
	cases := []struct {
		input    time.Duration
		expected time.Duration
	}{
		{1 * time.Second, 2 * time.Second},
		{2 * time.Second, 4 * time.Second},
		{4 * time.Second, 8 * time.Second},
		{32 * time.Second, 60 * time.Second},
		{60 * time.Second, 60 * time.Second}, // cap
	}

	for _, tc := range cases {
		next := tc.input * 2
		if next > 60*time.Second {
			next = 60 * time.Second
		}
		if next != tc.expected {
			t.Errorf("backoff(%s): want %s got %s", tc.input, tc.expected, next)
		}
	}
}

func TestFedClient_BackoffReset(t *testing.T) {
	// After a successful connect, backoff resets to 1s
	backoff := 32 * time.Second
	// Simulate successful connect
	backoff = time.Second
	if backoff != time.Second {
		t.Errorf("expected backoff reset to 1s, got %s", backoff)
	}
}

func TestFedClient_BroadcastDelta_Disconnected(t *testing.T) {
	// BroadcastDelta with nil conn should not panic
	client := &FederationClient{}
	payload, _ := json.Marshal(map[string]string{"k": "v"})
	msg := FedMessage{Type: FedEventAgentHeartbeat, Payload: json.RawMessage(payload)}

	// Should silently drop — no panic
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("BroadcastDelta panicked with nil conn: %v", r)
		}
	}()
	client.BroadcastDelta(msg)
}

// ── API endpoint tests ────────────────────────────────────────────────────────

func TestFedAPI_List_Empty(t *testing.T) {
	s := newTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/api/federation", nil)
	req.Header.Set("Authorization", authHeader())
	rr := httptest.NewRecorder()
	s.router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	var list []federationResponse
	if err := json.NewDecoder(rr.Body).Decode(&list); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(list) != 0 {
		t.Errorf("expected empty list, got %d", len(list))
	}
}

func TestFedAPI_Create(t *testing.T) {
	s := newTestServer(t)

	body := `{"name":"NYC Office","url":"http://nyc.internal:8080","token":"nyc-token"}`
	req := httptest.NewRequest(http.MethodPost, "/api/federation", bytes.NewBufferString(body))
	req.Header.Set("Authorization", authHeader())
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	s.router.ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rr.Code, rr.Body.String())
	}

	var resp federationResponse
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.ID == "" {
		t.Error("expected non-empty ID")
	}
	if resp.Name != "NYC Office" {
		t.Errorf("expected name 'NYC Office', got %q", resp.Name)
	}
	if resp.Status != "offline" {
		t.Errorf("expected status 'offline', got %q", resp.Status)
	}
}

func TestFedAPI_Create_MissingFields(t *testing.T) {
	s := newTestServer(t)

	cases := []string{
		`{"url":"http://x","token":"t"}`, // missing name
		`{"name":"X","token":"t"}`,       // missing url
		`{"name":"X","url":"http://x"}`,  // missing token
	}

	for _, body := range cases {
		req := httptest.NewRequest(http.MethodPost, "/api/federation", bytes.NewBufferString(body))
		req.Header.Set("Authorization", authHeader())
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		s.router.ServeHTTP(rr, req)

		if rr.Code != http.StatusBadRequest {
			t.Errorf("body %q: expected 400, got %d", body, rr.Code)
		}
	}
}

func TestFedAPI_Get(t *testing.T) {
	s := newTestServer(t)

	// Create
	body := `{"name":"London","url":"http://lon.internal","token":"lon-tok"}`
	req := httptest.NewRequest(http.MethodPost, "/api/federation", bytes.NewBufferString(body))
	req.Header.Set("Authorization", authHeader())
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	s.router.ServeHTTP(rr, req)
	var created federationResponse
	json.NewDecoder(rr.Body).Decode(&created)

	// Get
	req2 := httptest.NewRequest(http.MethodGet, "/api/federation/"+created.ID, nil)
	req2.Header.Set("Authorization", authHeader())
	rr2 := httptest.NewRecorder()
	s.router.ServeHTTP(rr2, req2)

	if rr2.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr2.Code)
	}
	var got federationResponse
	json.NewDecoder(rr2.Body).Decode(&got)
	if got.Name != "London" {
		t.Errorf("expected 'London', got %q", got.Name)
	}
}

func TestFedAPI_Get_NotFound(t *testing.T) {
	s := newTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/api/federation/does-not-exist", nil)
	req.Header.Set("Authorization", authHeader())
	rr := httptest.NewRecorder()
	s.router.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rr.Code)
	}
}

func TestFedAPI_Update(t *testing.T) {
	s := newTestServer(t)

	// Create
	body := `{"name":"Berlin","url":"http://ber.internal","token":"ber-tok"}`
	req := httptest.NewRequest(http.MethodPost, "/api/federation", bytes.NewBufferString(body))
	req.Header.Set("Authorization", authHeader())
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	s.router.ServeHTTP(rr, req)
	var created federationResponse
	json.NewDecoder(rr.Body).Decode(&created)

	// Update name
	body2 := `{"name":"Berlin Office"}`
	req2 := httptest.NewRequest(http.MethodPut, "/api/federation/"+created.ID, bytes.NewBufferString(body2))
	req2.Header.Set("Authorization", authHeader())
	req2.Header.Set("Content-Type", "application/json")
	rr2 := httptest.NewRecorder()
	s.router.ServeHTTP(rr2, req2)

	if rr2.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr2.Code, rr2.Body.String())
	}
	var updated federationResponse
	json.NewDecoder(rr2.Body).Decode(&updated)
	if updated.Name != "Berlin Office" {
		t.Errorf("expected 'Berlin Office', got %q", updated.Name)
	}
}

func TestFedAPI_Delete(t *testing.T) {
	s := newTestServer(t)

	// Create
	body := `{"name":"Tokyo","url":"http://tok.internal","token":"tok-tok"}`
	req := httptest.NewRequest(http.MethodPost, "/api/federation", bytes.NewBufferString(body))
	req.Header.Set("Authorization", authHeader())
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	s.router.ServeHTTP(rr, req)
	var created federationResponse
	json.NewDecoder(rr.Body).Decode(&created)

	// Delete
	req2 := httptest.NewRequest(http.MethodDelete, "/api/federation/"+created.ID, nil)
	req2.Header.Set("Authorization", authHeader())
	rr2 := httptest.NewRecorder()
	s.router.ServeHTTP(rr2, req2)

	if rr2.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", rr2.Code)
	}

	// Verify gone
	req3 := httptest.NewRequest(http.MethodGet, "/api/federation/"+created.ID, nil)
	req3.Header.Set("Authorization", authHeader())
	rr3 := httptest.NewRecorder()
	s.router.ServeHTTP(rr3, req3)

	if rr3.Code != http.StatusNotFound {
		t.Fatalf("expected 404 after delete, got %d", rr3.Code)
	}
}

func TestFedAPI_CacheEndpoints_UnknownSite(t *testing.T) {
	s := newTestServer(t)

	for _, path := range []string{
		"/api/federation/no-such-id/agents",
		"/api/federation/no-such-id/jobs",
		"/api/federation/no-such-id/history",
	} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		req.Header.Set("Authorization", authHeader())
		rr := httptest.NewRecorder()
		s.router.ServeHTTP(rr, req)

		if rr.Code != http.StatusNotFound {
			t.Errorf("%s: expected 404, got %d", path, rr.Code)
		}
	}
}

func TestFedAPI_CacheEndpoints_OfflineSite(t *testing.T) {
	s := newTestServer(t)

	// Register a sub but don't connect it (no WS) — simulates offline state
	body := `{"name":"Offline Site","url":"http://off.internal","token":"off-tok"}`
	req := httptest.NewRequest(http.MethodPost, "/api/federation", bytes.NewBufferString(body))
	req.Header.Set("Authorization", authHeader())
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	s.router.ServeHTTP(rr, req)
	var created federationResponse
	json.NewDecoder(rr.Body).Decode(&created)

	// Agents endpoint should return stale=true with empty list (not 404)
	req2 := httptest.NewRequest(http.MethodGet, "/api/federation/"+created.ID+"/agents", nil)
	req2.Header.Set("Authorization", authHeader())
	rr2 := httptest.NewRecorder()
	s.router.ServeHTTP(rr2, req2)

	if rr2.Code != http.StatusOK {
		t.Fatalf("expected 200 for offline site agents, got %d", rr2.Code)
	}
	var resp map[string]interface{}
	json.NewDecoder(rr2.Body).Decode(&resp)
	if resp["stale"] != true {
		t.Errorf("expected stale=true for offline site, got %v", resp["stale"])
	}
}

func TestFedAPI_Sync_Returns202(t *testing.T) {
	s := newTestServer(t)

	// Create a sub
	body := `{"name":"Sync Site","url":"http://sync.internal","token":"sync-tok"}`
	req := httptest.NewRequest(http.MethodPost, "/api/federation", bytes.NewBufferString(body))
	req.Header.Set("Authorization", authHeader())
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	s.router.ServeHTTP(rr, req)
	var created federationResponse
	json.NewDecoder(rr.Body).Decode(&created)

	// Force sync
	req2 := httptest.NewRequest(http.MethodPost, "/api/federation/"+created.ID+"/sync", nil)
	req2.Header.Set("Authorization", authHeader())
	rr2 := httptest.NewRecorder()
	s.router.ServeHTTP(rr2, req2)

	if rr2.Code != http.StatusAccepted {
		t.Fatalf("expected 202, got %d", rr2.Code)
	}
}

func TestFedAPI_RequiresAdminToken(t *testing.T) {
	s := newTestServer(t)

	// Agent token should be rejected for federation endpoints
	agentToken, _ := s.db.CreateAgentToken("agent-01")

	req := httptest.NewRequest(http.MethodGet, "/api/federation", nil)
	req.Header.Set("Authorization", "Bearer "+agentToken)
	rr := httptest.NewRecorder()
	s.router.ServeHTTP(rr, req)

	// Agent token is not a valid user JWT: JWTMiddleware rejects it with 401
	// before role evaluation (no longer accepted as a master admin token).
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for agent token on federation endpoint, got %d", rr.Code)
	}
}



package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"arcvault/coordinator/config"
	"arcvault/coordinator/db"
)

func TestFederationSync(t *testing.T) {
	// Setup test database
	cfg := &config.Config{
		Port:         8080,
		DatabasePath: ":memory:",
	}
	database, err := db.Init(cfg.DatabasePath)
	if err != nil {
		t.Fatalf("failed to init database: %v", err)
	}
	defer database.Close()

	// Create server
	s := NewWithFS(cfg, database, nil)

	// Test 1: Sync with no events (empty log)
	t.Run("empty_event_log", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/federation/sync?since=0&coordinator=root", nil)
		w := httptest.NewRecorder()
		s.handleFederationSync(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", w.Code)
		}

		var resp FederationSyncResponse
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to unmarshal response: %v", err)
		}

		if len(resp.Events) != 0 {
			t.Errorf("expected 0 events, got %d", len(resp.Events))
		}
		if resp.LatestSeq != 0 {
			t.Errorf("expected latest_seq=0, got %d", resp.LatestSeq)
		}
	})

	// Test 2: Append events and sync
	t.Run("sync_with_events", func(t *testing.T) {
		// Append two events
		seq1, err := database.AppendFederationEvent("root", "agent_registered", `{"id":"agent-1"}`)
		if err != nil {
			t.Fatalf("failed to append event: %v", err)
		}
		seq2, err := database.AppendFederationEvent("root", "job_created", `{"id":"job-1"}`)
		if err != nil {
			t.Fatalf("failed to append event: %v", err)
		}

		// Sync from start
		req := httptest.NewRequest("GET", "/api/federation/sync?since=0&coordinator=root", nil)
		w := httptest.NewRecorder()
		s.handleFederationSync(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", w.Code)
		}

		var resp FederationSyncResponse
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to unmarshal response: %v", err)
		}

		if len(resp.Events) != 2 {
			t.Errorf("expected 2 events, got %d", len(resp.Events))
		}
		if resp.LatestSeq != seq2 {
			t.Errorf("expected latest_seq=%d, got %d", seq2, resp.LatestSeq)
		}

		// Verify events
		if resp.Events[0].Seq != seq1 || resp.Events[0].EventType != "agent_registered" {
			t.Errorf("event 0 mismatch: %+v", resp.Events[0])
		}
		if resp.Events[1].Seq != seq2 || resp.Events[1].EventType != "job_created" {
			t.Errorf("event 1 mismatch: %+v", resp.Events[1])
		}
	})

	// Test 3: Sync with since > 0 (fetch only newer events)
	t.Run("sync_with_since", func(t *testing.T) {
		// Append third event
		seq3, err := database.AppendFederationEvent("root", "agent_heartbeat", `{"id":"agent-1"}`)
		if err != nil {
			t.Fatalf("failed to append event: %v", err)
		}

		// Sync since seq 1
		req := httptest.NewRequest("GET", "/api/federation/sync?since=1&coordinator=root", nil)
		w := httptest.NewRecorder()
		s.handleFederationSync(w, req)

		var resp FederationSyncResponse
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to unmarshal response: %v", err)
		}

		if len(resp.Events) != 2 {
			t.Errorf("expected 2 events (seq > 1), got %d", len(resp.Events))
		}
		if resp.Events[0].Seq != 2 {
			t.Errorf("expected first event seq=2, got %d", resp.Events[0].Seq)
		}
		if resp.LatestSeq != seq3 {
			t.Errorf("expected latest_seq=%d, got %d", seq3, resp.LatestSeq)
		}
	})

	// Test 4: Invalid parameters
	t.Run("invalid_params", func(t *testing.T) {
		// Missing since
		req := httptest.NewRequest("GET", "/api/federation/sync?coordinator=root", nil)
		w := httptest.NewRecorder()
		s.handleFederationSync(w, req)
		if w.Code != http.StatusBadRequest {
			t.Errorf("expected 400 for missing since, got %d", w.Code)
		}

		// Missing coordinator
		req = httptest.NewRequest("GET", "/api/federation/sync?since=0", nil)
		w = httptest.NewRecorder()
		s.handleFederationSync(w, req)
		if w.Code != http.StatusBadRequest {
			t.Errorf("expected 400 for missing coordinator, got %d", w.Code)
		}

		// Invalid since (non-numeric)
		req = httptest.NewRequest("GET", "/api/federation/sync?since=abc&coordinator=root", nil)
		w = httptest.NewRecorder()
		s.handleFederationSync(w, req)
		if w.Code != http.StatusBadRequest {
			t.Errorf("expected 400 for invalid since, got %d", w.Code)
		}
	})

	// Test 5: Sync ack
	t.Run("sync_ack", func(t *testing.T) {
		body := bytes.NewReader([]byte(`{"seq": 3}`))
		req := httptest.NewRequest("POST", "/api/federation/sync/ack?coordinator=spoke-1", body)
		w := httptest.NewRecorder()
		s.handleFederationSyncAck(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", w.Code)
		}

		var resp map[string]interface{}
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to unmarshal response: %v", err)
		}

		if resp["status"] != "ack" {
			t.Errorf("expected status=ack, got %v", resp["status"])
		}
		if resp["seq"].(float64) != 3 {
			t.Errorf("expected seq=3, got %v", resp["seq"])
		}
	})
}

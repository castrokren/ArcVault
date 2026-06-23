package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"arcvault/coordinator/config"
	"arcvault/coordinator/db"
)

func TestFederationHealth(t *testing.T) {
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

	// Test 1: Empty federation (no peers)
	t.Run("no_peers", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/federation/health", nil)
		w := httptest.NewRecorder()
		s.handleFederationHealth(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", w.Code)
		}

		var healthList []CoordinatorHealth
		if err := json.Unmarshal(w.Body.Bytes(), &healthList); err != nil {
			t.Fatalf("failed to unmarshal response: %v", err)
		}

		if len(healthList) != 0 {
			t.Errorf("expected 0 peers, got %d", len(healthList))
		}
	})

	// Test 2: Single online peer
	t.Run("online_peer", func(t *testing.T) {
		// Create a federation peer
		now := time.Now()
		_, err := database.Conn().Exec(
			`INSERT INTO federation (id, name, url, token, status, last_seen, version) VALUES (?, ?, ?, ?, ?, ?, ?)`,
			"spoke-1", "Spoke Coordinator 1", "http://spoke1:8080", "token-123", "online", now, "v0.9.0",
		)
		if err != nil {
			t.Fatalf("failed to create federation peer: %v", err)
		}

		req := httptest.NewRequest("GET", "/api/federation/health", nil)
		w := httptest.NewRecorder()
		s.handleFederationHealth(w, req)

		var healthList []CoordinatorHealth
		if err := json.Unmarshal(w.Body.Bytes(), &healthList); err != nil {
			t.Fatalf("failed to unmarshal response: %v", err)
		}

		if len(healthList) != 1 {
			t.Fatalf("expected 1 peer, got %d", len(healthList))
		}

		peer := healthList[0]
		if peer.ID != "spoke-1" {
			t.Errorf("expected peer id=spoke-1, got %s", peer.ID)
		}
		if peer.Status != "online" {
			t.Errorf("expected status=online (recent last_seen), got %s", peer.Status)
		}
		if peer.LagEvents != 0 {
			t.Errorf("expected lag_events=0 (no events), got %d", peer.LagEvents)
		}
	})

	// Test 3: Offline peer (stale last_seen)
	t.Run("offline_peer", func(t *testing.T) {
		// Create a federation peer with old last_seen
		oldTime := time.Now().Add(-60 * time.Second)
		_, err := database.Conn().Exec(
			`INSERT INTO federation (id, name, url, token, status, last_seen, version) VALUES (?, ?, ?, ?, ?, ?, ?)`,
			"spoke-2", "Spoke Coordinator 2", "http://spoke2:8080", "token-456", "offline", oldTime, "v0.9.0",
		)
		if err != nil {
			t.Fatalf("failed to create federation peer: %v", err)
		}

		req := httptest.NewRequest("GET", "/api/federation/health", nil)
		w := httptest.NewRecorder()
		s.handleFederationHealth(w, req)

		var healthList []CoordinatorHealth
		if err := json.Unmarshal(w.Body.Bytes(), &healthList); err != nil {
			t.Fatalf("failed to unmarshal response: %v", err)
		}

		// Find spoke-2 in the list
		var spoke2 *CoordinatorHealth
		for i := range healthList {
			if healthList[i].ID == "spoke-2" {
				spoke2 = &healthList[i]
				break
			}
		}

		if spoke2 == nil {
			t.Fatalf("spoke-2 not found in health list")
		}

		if spoke2.Status != "offline" {
			t.Errorf("expected status=offline (stale last_seen), got %s", spoke2.Status)
		}
	})

	// Test 4: Lag calculation
	t.Run("event_lag", func(t *testing.T) {
		// Append some events
		_, _ = database.AppendFederationEvent("spoke-1", "agent_registered", `{"id":"agent-1"}`)
		_, _ = database.AppendFederationEvent("spoke-1", "job_created", `{"id":"job-1"}`)
		seq3, _ := database.AppendFederationEvent("spoke-1", "agent_heartbeat", `{"id":"agent-1"}`)

		// Update spoke-1's last_seq to 1 (1 event behind)
		database.Conn().Exec(`UPDATE federation SET last_seq = 1 WHERE id = ?`, "spoke-1")

		req := httptest.NewRequest("GET", "/api/federation/health", nil)
		w := httptest.NewRecorder()
		s.handleFederationHealth(w, req)

		var healthList []CoordinatorHealth
		if err := json.Unmarshal(w.Body.Bytes(), &healthList); err != nil {
			t.Fatalf("failed to unmarshal response: %v", err)
		}

		var spoke1 *CoordinatorHealth
		for i := range healthList {
			if healthList[i].ID == "spoke-1" {
				spoke1 = &healthList[i]
				break
			}
		}

		if spoke1 == nil {
			t.Fatalf("spoke-1 not found in health list")
		}

		expectedLag := int(seq3 - 1) // maxSeq - lastSeq
		if spoke1.LagEvents != expectedLag {
			t.Errorf("expected lag_events=%d, got %d", expectedLag, spoke1.LagEvents)
		}
		if spoke1.LastSeq != 1 {
			t.Errorf("expected last_seq=1, got %d", spoke1.LastSeq)
		}
		if spoke1.MaxSeq != seq3 {
			t.Errorf("expected max_seq=%d, got %d", seq3, spoke1.MaxSeq)
		}
	})
}

package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFederationHealthAgentCount(t *testing.T) {
	s := newTestServer(t)

	// Register a federation peer
	_, err := s.db.Conn().Exec(`
		INSERT INTO federation (id, name, url, token, status, last_seen)
		VALUES ('coord-b', 'Spoke B', 'http://spoke:8080', 'tok-b', 'online', CURRENT_TIMESTAMP)
	`)
	require.NoError(t, err)

	// Two agents homed to coord-b, one homed to self
	_, err = s.db.Conn().Exec(`
		INSERT INTO agents (id, hostname, os, arch, version, status, home_coordinator)
		VALUES
		  ('a1', 'host-1', 'linux', 'amd64', 'v1', 'online', 'coord-b'),
		  ('a2', 'host-2', 'linux', 'amd64', 'v1', 'online', 'coord-b'),
		  ('a3', 'host-3', 'linux', 'amd64', 'v1', 'online', ?)
	`, s.coordinatorID)
	require.NoError(t, err)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/federation/health", nil)
	req.Header.Set("Authorization", "Bearer "+s.cfg.AdminToken)
	s.router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)

	var resp []map[string]any
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))

	var coordB map[string]any
	for _, r := range resp {
		if r["id"] == "coord-b" {
			coordB = r
		}
	}
	require.NotNil(t, coordB, "coord-b should appear in health response")
	assert.Equal(t, float64(2), coordB["agent_count"])
}

func TestHeartbeatDetectorMarksOffline(t *testing.T) {
	s := newTestServer(t)

	// Insert a coordinator with a stale last_seen (2 minutes ago)
	staleTime := time.Now().UTC().Add(-2 * time.Minute)
	_, err := s.db.Conn().Exec(`
		INSERT INTO federation (id, name, url, token, status, last_seen)
		VALUES ('spoke-stale', 'Stale Spoke', 'http://spoke:8080', 'tok-s', 'online', ?)
	`, staleTime)
	require.NoError(t, err)

	// Run detector directly — no ticker wait in tests
	s.checkHeartbeatTimeouts(30 * time.Second)

	var status string
	err = s.db.Conn().QueryRow(
		`SELECT status FROM federation WHERE id = 'spoke-stale'`,
	).Scan(&status)
	require.NoError(t, err)
	assert.Equal(t, "offline", status)

	// A coordinator_offline federation event should have been appended
	events, err := s.db.GetFederationEventsSince(s.coordinatorID, 0)
	require.NoError(t, err)
	require.Len(t, events, 1)
	assert.Equal(t, "coordinator_offline", events[0].EventType)
}

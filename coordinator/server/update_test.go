package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"arcvault/coordinator/config"
	"arcvault/coordinator/db"
	"arcvault/coordinator/updater"
)

// testHelper creates a test server with in-memory database
func newUpdateTestServer(t *testing.T) *Server {
	t.Helper()
	database, err := db.Init(":memory:")
	if err != nil {
		t.Fatalf("failed to init test db: %v", err)
	}
	t.Cleanup(func() { database.Close() })

	cfg := &config.Config{
		Port:       8080,
		AdminToken: "test-token",
	}
	return NewWithFS(cfg, database, nil)
}

// TestCheckUpdateEndpoint tests the /api/update/check endpoint.
func TestCheckUpdateEndpoint(t *testing.T) {
	srv := newUpdateTestServer(t)

	// Cache some update info
	cachedInfo := &updater.UpdateInfo{
		Current:         "v0.2.0",
		Latest:          "v0.3.0",
		UpdateAvailable: true,
		ReleaseURL:      "https://github.com/castrokren/ArcVault/releases/tag/v0.3.0",
		AssetURL:        "https://example.com/binary",
	}
	SetUpdateCache(cachedInfo)

	// Make request
	req := httptest.NewRequest("GET", "/api/update/check", nil)
	req.Header.Set("Authorization", "Bearer test-token")
	w := httptest.NewRecorder()

	srv.adminMiddleware(srv.handleCheckUpdate)(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var result updater.UpdateInfo
	if err := json.NewDecoder(w.Body).Decode(&result); err != nil {
		t.Errorf("Failed to decode response: %v", err)
	}

	if result.Latest != "v0.3.0" {
		t.Errorf("Expected latest v0.3.0, got %s", result.Latest)
	}

	if !result.UpdateAvailable {
		t.Errorf("Expected UpdateAvailable to be true")
	}
}

// TestCheckUpdateCached tests that the cache is used for subsequent requests.
func TestCheckUpdateCached(t *testing.T) {
	srv := newUpdateTestServer(t)

	// Cache initial info
	cachedInfo := &updater.UpdateInfo{
		Current:         "v0.2.0",
		Latest:          "v0.3.0",
		UpdateAvailable: true,
	}
	SetUpdateCache(cachedInfo)

	// Make first request
	req1 := httptest.NewRequest("GET", "/api/update/check", nil)
	req1.Header.Set("Authorization", "Bearer test-token")
	w1 := httptest.NewRecorder()

	srv.adminMiddleware(srv.handleCheckUpdate)(w1, req1)

	var result1 updater.UpdateInfo
	json.NewDecoder(w1.Body).Decode(&result1)

	// Make second request
	req2 := httptest.NewRequest("GET", "/api/update/check", nil)
	req2.Header.Set("Authorization", "Bearer test-token")
	w2 := httptest.NewRecorder()

	srv.adminMiddleware(srv.handleCheckUpdate)(w2, req2)

	var result2 updater.UpdateInfo
	json.NewDecoder(w2.Body).Decode(&result2)

	// Both should return same cached data
	if result1.Latest != result2.Latest {
		t.Errorf("Cached results differ")
	}
}

// TestApplyUpdateRejectsNonAdmin tests that non-admin tokens cannot trigger updates.
func TestApplyUpdateRejectsNonAdmin(t *testing.T) {
	srv := newUpdateTestServer(t)

	// Create an agent token
	agentToken, _ := srv.db.CreateAgentToken("test-agent")

	// Try to update with agent token
	req := httptest.NewRequest("POST", "/api/update/apply", nil)
	req.Header.Set("Authorization", "Bearer "+agentToken)
	w := httptest.NewRecorder()

	// Use the raw handler since we need to test adminMiddleware rejection
	adminHandler := srv.adminMiddleware(srv.handleApplyUpdate)
	adminHandler(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("Expected status 403, got %d (body: %s)", w.Code, w.Body.String())
	}
}

// TestApplyUpdateAlreadyRunning tests that a second update request returns 409.
func TestApplyUpdateAlreadyRunning(t *testing.T) {
	srv := newUpdateTestServer(t)

	// Manually set updateRunning to true
	updateRunning.Store(true)
	defer updateRunning.Store(false)

	// Try to start update
	req := httptest.NewRequest("POST", "/api/update/apply", nil)
	req.Header.Set("Authorization", "Bearer test-token")
	w := httptest.NewRecorder()

	// Skip adminMiddleware for this test, call handler directly
	srv.handleApplyUpdate(w, req)

	if w.Code != http.StatusConflict {
		t.Errorf("Expected status 409, got %d", w.Code)
	}

	var result map[string]string
	json.NewDecoder(w.Body).Decode(&result)
	if result["error"] != "update already in progress" {
		t.Errorf("Expected error message about update already in progress")
	}
}

// TestUpdateCaching tests the cache get/set functions.
func TestUpdateCaching(t *testing.T) {
	// Clear cache first
	SetUpdateCache(nil)

	// Get should return nil when cache is empty
	if GetUpdateCache() != nil {
		t.Errorf("Expected nil cache")
	}

	// Set cache
	info := &updater.UpdateInfo{
		Current: "v0.2.0",
		Latest:  "v0.3.0",
	}
	SetUpdateCache(info)

	// Get should return the set info
	cached := GetUpdateCache()
	if cached == nil {
		t.Errorf("Expected cached info")
	}
	if cached.Latest != "v0.3.0" {
		t.Errorf("Expected latest v0.3.0, got %s", cached.Latest)
	}
}

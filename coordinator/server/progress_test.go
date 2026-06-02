package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"arcvault/coordinator/db"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestProgressEndpoint_StoresPercentage verifies POST endpoint stores percentage in DB.
func TestProgressEndpoint_StoresPercentage(t *testing.T) {
	srv := setupTestServer(t)
	defer srv.db.Close()

	// Create test job
	jobID := "test-job-001"
	_, err := srv.db.Conn().Exec(
		`INSERT INTO jobs (id, agent_id, name, source_path, dest_path) VALUES (?, ?, ?, ?, ?)`,
		jobID, "agent-1", "Test Job", "/source", "/dest",
	)
	require.NoError(t, err)

	// POST progress
	payload := map[string]interface{}{
		"percentage": 50,
		"logs":       []string{"line 1", "line 2"},
		"status":     "running",
	}
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest("POST", fmt.Sprintf("/api/jobs/%s/progress", jobID), bytes.NewReader(body))
	w := httptest.NewRecorder()

	srv.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code, "should return 200")

	// Verify percentage stored
	var percentage int
	err = srv.db.Conn().QueryRow(
		`SELECT progress FROM job_runs WHERE job_id = ? ORDER BY started_at DESC LIMIT 1`,
		jobID,
	).Scan(&percentage)
	require.NoError(t, err)
	assert.Equal(t, 50, percentage, "percentage should be stored")
}

// TestProgressEndpoint_AppendsToJobLogs verifies POST appends log lines.
func TestProgressEndpoint_AppendsToJobLogs(t *testing.T) {
	srv := setupTestServer(t)
	defer srv.db.Close()

	// Create test job and run
	jobID := "test-job-002"
	_, err := srv.db.Conn().Exec(
		`INSERT INTO jobs (id, agent_id, name, source_path, dest_path) VALUES (?, ?, ?, ?, ?)`,
		jobID, "agent-1", "Test Job", "/source", "/dest",
	)
	require.NoError(t, err)

	// POST progress with logs
	logs := []string{"starting backup", "copying files", "50% complete"}
	payload := map[string]interface{}{
		"percentage": 50,
		"logs":       logs,
		"status":     "running",
	}
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest("POST", fmt.Sprintf("/api/jobs/%s/progress", jobID), bytes.NewReader(body))
	w := httptest.NewRecorder()

	srv.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	// Verify logs stored
	rows, err := srv.db.Conn().Query(
		`SELECT line FROM job_logs WHERE job_id = ? ORDER BY created_at ASC`,
		jobID,
	)
	require.NoError(t, err)
	defer rows.Close()

	var storedLogs []string
	for rows.Next() {
		var line string
		err := rows.Scan(&line)
		require.NoError(t, err)
		storedLogs = append(storedLogs, line)
	}

	assert.Equal(t, logs, storedLogs, "logs should be stored in order")
}

// TestProgressEndpoint_InvalidPercentage_Returns400 verifies invalid percentage returns 400.
func TestProgressEndpoint_InvalidPercentage_Returns400(t *testing.T) {
	srv := setupTestServer(t)
	defer srv.db.Close()

	jobID := "test-job-003"
	_, err := srv.db.Conn().Exec(
		`INSERT INTO jobs (id, agent_id, name, source_path, dest_path) VALUES (?, ?, ?, ?, ?)`,
		jobID, "agent-1", "Test Job", "/source", "/dest",
	)
	require.NoError(t, err)

	testCases := []int{-1, 101, 150, -50}
	for _, percentage := range testCases {
		payload := map[string]interface{}{
			"percentage": percentage,
			"logs":       []string{},
			"status":     "running",
		}
		body, _ := json.Marshal(payload)
		req := httptest.NewRequest("POST", fmt.Sprintf("/api/jobs/%s/progress", jobID), bytes.NewReader(body))
		w := httptest.NewRecorder()

		srv.router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code, "percentage %d should return 400", percentage)
	}
}

// TestProgressEndpoint_MissingJob_Returns404 verifies missing job returns 404.
func TestProgressEndpoint_MissingJob_Returns404(t *testing.T) {
	srv := setupTestServer(t)
	defer srv.db.Close()

	payload := map[string]interface{}{
		"percentage": 50,
		"logs":       []string{},
		"status":     "running",
	}
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest("POST", "/api/jobs/nonexistent-job/progress", bytes.NewReader(body))
	w := httptest.NewRecorder()

	srv.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code, "nonexistent job should return 404")
}

// TestProgressEndpoint_StatusValues_Valid verifies valid status values are accepted.
func TestProgressEndpoint_StatusValues_Valid(t *testing.T) {
	srv := setupTestServer(t)
	defer srv.db.Close()

	jobID := "test-job-004"
	_, err := srv.db.Conn().Exec(
		`INSERT INTO jobs (id, agent_id, name, source_path, dest_path) VALUES (?, ?, ?, ?, ?)`,
		jobID, "agent-1", "Test Job", "/source", "/dest",
	)
	require.NoError(t, err)

	validStatuses := []string{"running", "completed", "cancelled", "error"}
	for _, status := range validStatuses {
		payload := map[string]interface{}{
			"percentage": 50,
			"logs":       []string{},
			"status":     status,
		}
		body, _ := json.Marshal(payload)
		req := httptest.NewRequest("POST", fmt.Sprintf("/api/jobs/%s/progress", jobID), bytes.NewReader(body))
		w := httptest.NewRecorder()

		srv.router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code, "status %s should be accepted", status)
	}
}

// TestProgressEndpoint_StatusValues_Invalid_Returns400 verifies invalid status returns 400.
func TestProgressEndpoint_StatusValues_Invalid_Returns400(t *testing.T) {
	srv := setupTestServer(t)
	defer srv.db.Close()

	jobID := "test-job-005"
	_, err := srv.db.Conn().Exec(
		`INSERT INTO jobs (id, agent_id, name, source_path, dest_path) VALUES (?, ?, ?, ?, ?)`,
		jobID, "agent-1", "Test Job", "/source", "/dest",
	)
	require.NoError(t, err)

	invalidStatuses := []string{"invalid", "running!", "pending", ""}
	for _, status := range invalidStatuses {
		payload := map[string]interface{}{
			"percentage": 50,
			"logs":       []string{},
			"status":     status,
		}
		body, _ := json.Marshal(payload)
		req := httptest.NewRequest("POST", fmt.Sprintf("/api/jobs/%s/progress", jobID), bytes.NewReader(body))
		w := httptest.NewRecorder()

		srv.router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code, "status %s should return 400", status)
	}
}

// TestGetProgress_ReturnsLatestPercentage verifies GET endpoint returns latest percentage.
func TestGetProgress_ReturnsLatestPercentage(t *testing.T) {
	srv := setupTestServer(t)
	defer srv.db.Close()

	jobID := "test-job-get-001"
	_, err := srv.db.Conn().Exec(
		`INSERT INTO jobs (id, agent_id, name, source_path, dest_path) VALUES (?, ?, ?, ?, ?)`,
		jobID, "agent-1", "Test Job", "/source", "/dest",
	)
	require.NoError(t, err)
	// Trigger auto-creates job_run on job insert

	// POST progress
	payload := map[string]interface{}{
		"percentage": 75,
		"logs":       []string{},
		"status":     "running",
	}
	body, _ := json.Marshal(payload)
	postReq := httptest.NewRequest("POST", fmt.Sprintf("/api/jobs/%s/progress", jobID), bytes.NewReader(body))
	postW := httptest.NewRecorder()
	srv.router.ServeHTTP(postW, postReq)
	require.Equal(t, http.StatusOK, postW.Code, "POST should succeed")

	// GET progress
	getReq := httptest.NewRequest("GET", fmt.Sprintf("/api/jobs/%s/progress", jobID), nil)
	w := httptest.NewRecorder()
	srv.router.ServeHTTP(w, getReq)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, float64(75), resp["percentage"], "percentage should match")
}

// TestGetProgress_Returns404ForMissingJob verifies missing job returns 404.
func TestGetProgress_Returns404ForMissingJob(t *testing.T) {
	srv := setupTestServer(t)
	defer srv.db.Close()

	req := httptest.NewRequest("GET", "/api/jobs/nonexistent-job/progress", nil)
	w := httptest.NewRecorder()

	srv.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

// TestGetProgress_ReturnsRecentLogs verifies GET returns recent log lines.
func TestGetProgress_ReturnsRecentLogs(t *testing.T) {
	srv := setupTestServer(t)
	defer srv.db.Close()

	jobID := "test-job-get-002"
	_, err := srv.db.Conn().Exec(
		`INSERT INTO jobs (id, agent_id, name, source_path, dest_path) VALUES (?, ?, ?, ?, ?)`,
		jobID, "agent-1", "Test Job", "/source", "/dest",
	)
	require.NoError(t, err)
	// Trigger auto-creates job_run on job insert

	// POST progress with logs
	logs := []string{"line 1", "line 2", "line 3"}
	payload := map[string]interface{}{
		"percentage": 50,
		"logs":       logs,
		"status":     "running",
	}
	body, _ := json.Marshal(payload)
	postReq := httptest.NewRequest("POST", fmt.Sprintf("/api/jobs/%s/progress", jobID), bytes.NewReader(body))
	postW := httptest.NewRecorder()
	srv.router.ServeHTTP(postW, postReq)
	require.Equal(t, http.StatusOK, postW.Code, "POST should succeed")

	// GET progress
	getReq := httptest.NewRequest("GET", fmt.Sprintf("/api/jobs/%s/progress", jobID), nil)
	w := httptest.NewRecorder()
	srv.router.ServeHTTP(w, getReq)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)

	respLogs, ok := resp["logs"].([]interface{})
	require.True(t, ok, "logs should be array")
	assert.Equal(t, 3, len(respLogs), "should have 3 logs")
}

// TestGetProgress_ReturnsStatusField verifies GET returns status.
func TestGetProgress_ReturnsStatusField(t *testing.T) {
	srv := setupTestServer(t)
	defer srv.db.Close()

	jobID := "test-job-get-003"
	_, err := srv.db.Conn().Exec(
		`INSERT INTO jobs (id, agent_id, name, source_path, dest_path) VALUES (?, ?, ?, ?, ?)`,
		jobID, "agent-1", "Test Job", "/source", "/dest",
	)
	require.NoError(t, err)
	// Trigger auto-creates job_run on job insert

	// POST progress with different statuses
	statuses := []string{"running", "completed"}
	for _, status := range statuses {
		payload := map[string]interface{}{
			"percentage": 100,
			"logs":       []string{},
			"status":     status,
		}
		body, _ := json.Marshal(payload)
		postReq := httptest.NewRequest("POST", fmt.Sprintf("/api/jobs/%s/progress", jobID), bytes.NewReader(body))
		postW := httptest.NewRecorder()
		srv.router.ServeHTTP(postW, postReq)
		require.Equal(t, http.StatusOK, postW.Code, "POST should succeed")

		// GET progress
		getReq := httptest.NewRequest("GET", fmt.Sprintf("/api/jobs/%s/progress", jobID), nil)
		w := httptest.NewRecorder()
		srv.router.ServeHTTP(w, getReq)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp map[string]interface{}
		err = json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, status, resp["status"], "status should match")
	}
}

// TestProgressBroadcast_SendsToAllClients verifies progress updates are broadcast to clients.
func TestProgressBroadcast_SendsToAllClients(t *testing.T) {
	srv := setupTestServer(t)
	defer srv.db.Close()

	jobID := "test-job-broadcast-001"
	_, err := srv.db.Conn().Exec(
		`INSERT INTO jobs (id, agent_id, name, source_path, dest_path) VALUES (?, ?, ?, ?, ?)`,
		jobID, "agent-1", "Test Job", "/source", "/dest",
	)
	require.NoError(t, err)

	// POST progress
	payload := map[string]interface{}{
		"percentage": 60,
		"logs":       []string{"progress"},
		"status":     "running",
	}
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest("POST", fmt.Sprintf("/api/jobs/%s/progress", jobID), bytes.NewReader(body))
	w := httptest.NewRecorder()

	srv.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code, "progress post should succeed")
}

// TestProgressBroadcastPayload_ContainsAllFields verifies broadcast event has required fields.
func TestProgressBroadcastPayload_ContainsAllFields(t *testing.T) {
	srv := setupTestServer(t)
	defer srv.db.Close()

	jobID := "test-job-broadcast-002"
	_, err := srv.db.Conn().Exec(
		`INSERT INTO jobs (id, agent_id, name, source_path, dest_path) VALUES (?, ?, ?, ?, ?)`,
		jobID, "agent-1", "Test Job", "/source", "/dest",
	)
	require.NoError(t, err)

	// POST progress
	payload := map[string]interface{}{
		"percentage": 80,
		"logs":       []string{},
		"status":     "running",
	}
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest("POST", fmt.Sprintf("/api/jobs/%s/progress", jobID), bytes.NewReader(body))
	w := httptest.NewRecorder()

	srv.router.ServeHTTP(w, req)

	// Verify response
	assert.Equal(t, http.StatusOK, w.Code)
	var resp ProgressResponse
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.True(t, resp.Success)
}

// TestProgressUpdate_MultipleClients verifies event format is correct.
func TestProgressUpdate_MultipleClients(t *testing.T) {
	srv := setupTestServer(t)
	defer srv.db.Close()

	jobID := "test-job-broadcast-003"
	_, err := srv.db.Conn().Exec(
		`INSERT INTO jobs (id, agent_id, name, source_path, dest_path) VALUES (?, ?, ?, ?, ?)`,
		jobID, "agent-1", "Test Job", "/source", "/dest",
	)
	require.NoError(t, err)

	// POST progress with specific values
	payload := map[string]interface{}{
		"percentage": 45,
		"logs":       []string{"step 1", "step 2"},
		"status":     "running",
	}
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest("POST", fmt.Sprintf("/api/jobs/%s/progress", jobID), bytes.NewReader(body))
	w := httptest.NewRecorder()

	srv.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	// Verify database was updated
	var dbPercentage int
	var dbStatus string
	err = srv.db.Conn().QueryRow(
		`SELECT progress, status FROM job_runs WHERE job_id = ?`,
		jobID,
	).Scan(&dbPercentage, &dbStatus)
	require.NoError(t, err)
	assert.Equal(t, 45, dbPercentage)
	assert.Equal(t, "running", dbStatus)
}

// --- GET /api/jobs/{id}/logs (Phase 21a-4 full logs with pagination) ---

// TestGetJobLogs_ReturnsAllLogs verifies pagination retrieves full log history.
func TestGetJobLogs_ReturnsAllLogs(t *testing.T) {
	srv := setupTestServer(t)
	defer srv.db.Close()

	jobID := "test-job-logs-001"
	_, err := srv.db.Conn().Exec(
		`INSERT INTO jobs (id, agent_id, name, source_path, dest_path) VALUES (?, ?, ?, ?, ?)`,
		jobID, "agent-1", "Test Job", "/source", "/dest",
	)
	require.NoError(t, err)

	// Insert 100 log lines
	logs := make([]string, 100)
	for i := 0; i < 100; i++ {
		logs[i] = fmt.Sprintf("log line %d", i+1)
		_, err := srv.db.Conn().Exec(
			`INSERT INTO job_logs (job_id, line) VALUES (?, ?)`,
			jobID, logs[i],
		)
		require.NoError(t, err)
	}

	// Request first page (25 items, default limit)
	req := httptest.NewRequest("GET", fmt.Sprintf("/api/jobs/%s/logs", jobID), nil)
	w := httptest.NewRecorder()

	srv.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp PaginatedResponse
	err = json.NewDecoder(w.Body).Decode(&resp)
	require.NoError(t, err)

	assert.Equal(t, 100, resp.Total, "total should be 100")
	assert.Equal(t, 1, resp.Page)
	assert.Equal(t, 25, resp.Limit)
	assert.Equal(t, 4, resp.Pages, "ceil(100/25) = 4")
	assert.Equal(t, 25, len(resp.Data.([]interface{})), "first page should have 25 logs")

	// Verify first log is "log line 1" (chronological order)
	data := resp.Data.([]interface{})
	assert.Equal(t, "log line 1", data[0])
}

// TestGetJobLogs_PaginationSecondPage verifies second page retrieval.
func TestGetJobLogs_PaginationSecondPage(t *testing.T) {
	srv := setupTestServer(t)
	defer srv.db.Close()

	jobID := "test-job-logs-002"
	_, err := srv.db.Conn().Exec(
		`INSERT INTO jobs (id, agent_id, name, source_path, dest_path) VALUES (?, ?, ?, ?, ?)`,
		jobID, "agent-1", "Test Job", "/source", "/dest",
	)
	require.NoError(t, err)

	// Insert 60 log lines
	for i := 0; i < 60; i++ {
		_, err := srv.db.Conn().Exec(
			`INSERT INTO job_logs (job_id, line) VALUES (?, ?)`,
			jobID, fmt.Sprintf("log line %d", i+1),
		)
		require.NoError(t, err)
	}

	// Request second page
	req := httptest.NewRequest("GET", fmt.Sprintf("/api/jobs/%s/logs?page=2&limit=25", jobID), nil)
	w := httptest.NewRecorder()

	srv.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp PaginatedResponse
	err = json.NewDecoder(w.Body).Decode(&resp)
	require.NoError(t, err)

	assert.Equal(t, 60, resp.Total)
	assert.Equal(t, 2, resp.Page)
	assert.Equal(t, 25, resp.Limit)
	assert.Equal(t, 3, resp.Pages)
	assert.Equal(t, 25, len(resp.Data.([]interface{})))

	// Verify first item on page 2 is "log line 26"
	data := resp.Data.([]interface{})
	assert.Equal(t, "log line 26", data[0])
}

// TestGetJobLogs_CustomLimit verifies custom limit parameter.
func TestGetJobLogs_CustomLimit(t *testing.T) {
	srv := setupTestServer(t)
	defer srv.db.Close()

	jobID := "test-job-logs-003"
	_, err := srv.db.Conn().Exec(
		`INSERT INTO jobs (id, agent_id, name, source_path, dest_path) VALUES (?, ?, ?, ?, ?)`,
		jobID, "agent-1", "Test Job", "/source", "/dest",
	)
	require.NoError(t, err)

	// Insert 50 log lines
	for i := 0; i < 50; i++ {
		_, err := srv.db.Conn().Exec(
			`INSERT INTO job_logs (job_id, line) VALUES (?, ?)`,
			jobID, fmt.Sprintf("log line %d", i+1),
		)
		require.NoError(t, err)
	}

	// Request with limit=10
	req := httptest.NewRequest("GET", fmt.Sprintf("/api/jobs/%s/logs?page=1&limit=10", jobID), nil)
	w := httptest.NewRecorder()

	srv.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp PaginatedResponse
	err = json.NewDecoder(w.Body).Decode(&resp)
	require.NoError(t, err)

	assert.Equal(t, 50, resp.Total)
	assert.Equal(t, 1, resp.Page)
	assert.Equal(t, 10, resp.Limit)
	assert.Equal(t, 5, resp.Pages)
	assert.Equal(t, 10, len(resp.Data.([]interface{})))
}

// TestGetJobLogs_Returns404ForMissingJob verifies nonexistent job returns 404.
func TestGetJobLogs_Returns404ForMissingJob(t *testing.T) {
	srv := setupTestServer(t)
	defer srv.db.Close()

	req := httptest.NewRequest("GET", "/api/jobs/nonexistent-job/logs", nil)
	w := httptest.NewRecorder()

	srv.router.ServeHTTP(w, req)

	// Should not error for nonexistent job, just return empty list
	assert.Equal(t, http.StatusOK, w.Code)

	var resp PaginatedResponse
	err := json.NewDecoder(w.Body).Decode(&resp)
	require.NoError(t, err)

	assert.Equal(t, 0, resp.Total)
	data := resp.Data.([]interface{})
	assert.Equal(t, 0, len(data))
}

// TestGetJobLogs_MaxLimitEnforced verifies limit > 100 is capped to 100.
func TestGetJobLogs_MaxLimitEnforced(t *testing.T) {
	srv := setupTestServer(t)
	defer srv.db.Close()

	jobID := "test-job-logs-004"
	_, err := srv.db.Conn().Exec(
		`INSERT INTO jobs (id, agent_id, name, source_path, dest_path) VALUES (?, ?, ?, ?, ?)`,
		jobID, "agent-1", "Test Job", "/source", "/dest",
	)
	require.NoError(t, err)

	// Insert 150 log lines
	for i := 0; i < 150; i++ {
		_, err := srv.db.Conn().Exec(
			`INSERT INTO job_logs (job_id, line) VALUES (?, ?)`,
			jobID, fmt.Sprintf("log line %d", i+1),
		)
		require.NoError(t, err)
	}

	// Request with limit=200 (should be capped to 100)
	req := httptest.NewRequest("GET", fmt.Sprintf("/api/jobs/%s/logs?page=1&limit=200", jobID), nil)
	w := httptest.NewRecorder()

	srv.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp PaginatedResponse
	err = json.NewDecoder(w.Body).Decode(&resp)
	require.NoError(t, err)

	assert.Equal(t, 150, resp.Total)
	assert.Equal(t, 100, resp.Limit, "limit should be capped to 100")
	assert.Equal(t, 100, len(resp.Data.([]interface{})))
}

// TestGetJobLogs_ChronologicalOrder verifies logs are returned in chronological order (oldest first).
func TestGetJobLogs_ChronologicalOrder(t *testing.T) {
	srv := setupTestServer(t)
	defer srv.db.Close()

	jobID := "test-job-logs-005"
	_, err := srv.db.Conn().Exec(
		`INSERT INTO jobs (id, agent_id, name, source_path, dest_path) VALUES (?, ?, ?, ?, ?)`,
		jobID, "agent-1", "Test Job", "/source", "/dest",
	)
	require.NoError(t, err)

	// Insert 10 log lines
	for i := 0; i < 10; i++ {
		_, err := srv.db.Conn().Exec(
			`INSERT INTO job_logs (job_id, line) VALUES (?, ?)`,
			jobID, fmt.Sprintf("log line %d", i+1),
		)
		require.NoError(t, err)
	}

	// Request all logs
	req := httptest.NewRequest("GET", fmt.Sprintf("/api/jobs/%s/logs", jobID), nil)
	w := httptest.NewRecorder()

	srv.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp PaginatedResponse
	err = json.NewDecoder(w.Body).Decode(&resp)
	require.NoError(t, err)

	data := resp.Data.([]interface{})
	assert.Equal(t, 10, len(data))

	// Verify chronological order: first should be "log line 1"
	assert.Equal(t, "log line 1", data[0])
	assert.Equal(t, "log line 10", data[9])
}

// setupTestServer creates a test server with test database.
func setupTestServer(t *testing.T) *Server {
	testDB, err := db.Init(":memory:")
	require.NoError(t, err)

	// Create test agent
	_, err = testDB.Conn().Exec(
		`INSERT INTO agents (id, hostname, os, arch, version) VALUES (?, ?, ?, ?, ?)`,
		"agent-1", "test-host", "linux", "x86_64", "v1.0",
	)
	require.NoError(t, err)

	// Create a default job_run template for tests
	// Tests will create jobs and this ensures job_runs exist
	_, err = testDB.Conn().Exec(
		`CREATE TRIGGER auto_job_run AFTER INSERT ON jobs
		 BEGIN
		   INSERT INTO job_runs (id, job_id, started_at) VALUES (
		     'run-' || NEW.id,
		     NEW.id,
		     CURRENT_TIMESTAMP
		   );
		 END`,
	)
	// Ignore error if trigger already exists
	_ = err

	srv := &Server{
		db:     testDB,
		router: http.NewServeMux(),
		hub:    newHub(),
	}

	// Register progress routes
	srv.router.HandleFunc("POST /api/jobs/{id}/progress", srv.handleProgress)
	srv.router.HandleFunc("GET /api/jobs/{id}/progress", srv.handleGetProgress)
	srv.router.HandleFunc("GET /api/jobs/{id}/logs", srv.handleGetJobLogs)

	return srv
}

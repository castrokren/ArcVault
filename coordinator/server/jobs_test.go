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

// --- test helpers ---

func newTestServer(t *testing.T) *Server {
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
	// pass empty staticDir so tests don't try to serve files from disk
	return NewWithStatic(cfg, database, "")
}

func authHeader() string {
	return "Bearer test-token"
}

// --- POST /api/jobs ---

func TestCreateJob_returnsCreatedWithJobJSON(t *testing.T) {
	s := newTestServer(t)

	body := `{"agent_id":"agent-01","name":"nightly-backup","source_path":"C:\\src","dest_path":"D:\\backup","schedule":"0 2 * * *"}`
	req := httptest.NewRequest(http.MethodPost, "/api/jobs", bytes.NewBufferString(body))
	req.Header.Set("Authorization", authHeader())
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	s.router.ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rr.Code, rr.Body.String())
	}

	var job Job
	if err := json.NewDecoder(rr.Body).Decode(&job); err != nil {
		t.Fatalf("response is not valid JSON: %v", err)
	}
	if job.ID == "" {
		t.Error("expected non-empty job ID")
	}
	if job.Name != "nightly-backup" {
		t.Errorf("expected name 'nightly-backup', got %q", job.Name)
	}
	if job.AgentID != "agent-01" {
		t.Errorf("expected agent_id 'agent-01', got %q", job.AgentID)
	}
	if job.SourcePath != "C:\\src" {
		t.Errorf("expected source_path 'C:\\src', got %q", job.SourcePath)
	}
	if job.DestPath != "D:\\backup" {
		t.Errorf("expected dest_path 'D:\\backup', got %q", job.DestPath)
	}
	if job.Status != "pending" {
		t.Errorf("expected status 'pending', got %q", job.Status)
	}
}

func TestCreateJob_missingNameReturnsBadRequest(t *testing.T) {
	s := newTestServer(t)

	body := `{"agent_id":"agent-01","source_path":"C:\\src","dest_path":"D:\\backup"}`
	req := httptest.NewRequest(http.MethodPost, "/api/jobs", bytes.NewBufferString(body))
	req.Header.Set("Authorization", authHeader())
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	s.router.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
}

func TestCreateJob_missingAgentIDReturnsBadRequest(t *testing.T) {
	s := newTestServer(t)

	body := `{"name":"nightly-backup","source_path":"C:\\src","dest_path":"D:\\backup"}`
	req := httptest.NewRequest(http.MethodPost, "/api/jobs", bytes.NewBufferString(body))
	req.Header.Set("Authorization", authHeader())
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	s.router.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
}

func TestCreateJob_missingSourcePathReturnsBadRequest(t *testing.T) {
	s := newTestServer(t)

	body := `{"agent_id":"agent-01","name":"nightly-backup","dest_path":"D:\\backup"}`
	req := httptest.NewRequest(http.MethodPost, "/api/jobs", bytes.NewBufferString(body))
	req.Header.Set("Authorization", authHeader())
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	s.router.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
}

func TestCreateJob_missingDestPathReturnsBadRequest(t *testing.T) {
	s := newTestServer(t)

	body := `{"agent_id":"agent-01","name":"nightly-backup","source_path":"C:\\src"}`
	req := httptest.NewRequest(http.MethodPost, "/api/jobs", bytes.NewBufferString(body))
	req.Header.Set("Authorization", authHeader())
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	s.router.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
}

func TestCreateJob_unauthenticatedReturns401(t *testing.T) {
	s := newTestServer(t)

	body := `{"agent_id":"agent-01","name":"nightly-backup","source_path":"C:\\src","dest_path":"D:\\backup"}`
	req := httptest.NewRequest(http.MethodPost, "/api/jobs", bytes.NewBufferString(body))
	rr := httptest.NewRecorder()

	s.router.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rr.Code)
	}
}

// --- GET /api/jobs ---

func TestListJobs_returnsEmptyArrayWhenNoJobs(t *testing.T) {
	s := newTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/api/jobs", nil)
	req.Header.Set("Authorization", authHeader())
	rr := httptest.NewRecorder()

	s.router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}

	var jobs []Job
	if err := json.NewDecoder(rr.Body).Decode(&jobs); err != nil {
		t.Fatalf("response is not valid JSON: %v", err)
	}
	if len(jobs) != 0 {
		t.Errorf("expected empty list, got %d jobs", len(jobs))
	}
}

func TestListJobs_returnsCreatedJobs(t *testing.T) {
	s := newTestServer(t)

	for _, name := range []string{"job-alpha", "job-beta"} {
		body := `{"agent_id":"agent-01","name":"` + name + `","source_path":"C:\\src","dest_path":"D:\\backup"}`
		req := httptest.NewRequest(http.MethodPost, "/api/jobs", bytes.NewBufferString(body))
		req.Header.Set("Authorization", authHeader())
		req.Header.Set("Content-Type", "application/json")
		s.router.ServeHTTP(httptest.NewRecorder(), req)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/jobs", nil)
	req.Header.Set("Authorization", authHeader())
	rr := httptest.NewRecorder()

	s.router.ServeHTTP(rr, req)

	var jobs []Job
	if err := json.NewDecoder(rr.Body).Decode(&jobs); err != nil {
		t.Fatalf("response is not valid JSON: %v", err)
	}
	if len(jobs) != 2 {
		t.Errorf("expected 2 jobs, got %d", len(jobs))
	}
}

func TestListJobs_filtersByAgentID(t *testing.T) {
	s := newTestServer(t)

	for _, agentID := range []string{"agent-01", "agent-01", "agent-02"} {
		body := `{"agent_id":"` + agentID + `","name":"backup","source_path":"C:\\src","dest_path":"D:\\backup"}`
		req := httptest.NewRequest(http.MethodPost, "/api/jobs", bytes.NewBufferString(body))
		req.Header.Set("Authorization", authHeader())
		req.Header.Set("Content-Type", "application/json")
		s.router.ServeHTTP(httptest.NewRecorder(), req)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/jobs?agent_id=agent-01", nil)
	req.Header.Set("Authorization", authHeader())
	rr := httptest.NewRecorder()
	s.router.ServeHTTP(rr, req)

	var jobs []Job
	if err := json.NewDecoder(rr.Body).Decode(&jobs); err != nil {
		t.Fatalf("response is not valid JSON: %v", err)
	}
	if len(jobs) != 2 {
		t.Errorf("expected 2 jobs for agent-01, got %d", len(jobs))
	}
}

// --- GET /api/jobs/{id} ---

func TestGetJob_returnsJobByID(t *testing.T) {
	s := newTestServer(t)

	body := `{"agent_id":"agent-01","name":"find-me","source_path":"C:\\src","dest_path":"D:\\backup"}`
	req := httptest.NewRequest(http.MethodPost, "/api/jobs", bytes.NewBufferString(body))
	req.Header.Set("Authorization", authHeader())
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	s.router.ServeHTTP(rr, req)

	var created Job
	json.NewDecoder(rr.Body).Decode(&created)

	req2 := httptest.NewRequest(http.MethodGet, "/api/jobs/"+created.ID, nil)
	req2.Header.Set("Authorization", authHeader())
	rr2 := httptest.NewRecorder()
	s.router.ServeHTTP(rr2, req2)

	if rr2.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr2.Code, rr2.Body.String())
	}

	var fetched Job
	if err := json.NewDecoder(rr2.Body).Decode(&fetched); err != nil {
		t.Fatalf("response is not valid JSON: %v", err)
	}
	if fetched.ID != created.ID {
		t.Errorf("expected ID %q, got %q", created.ID, fetched.ID)
	}
	if fetched.SourcePath != "C:\\src" {
		t.Errorf("expected source_path 'C:\\src', got %q", fetched.SourcePath)
	}
	if fetched.DestPath != "D:\\backup" {
		t.Errorf("expected dest_path 'D:\\backup', got %q", fetched.DestPath)
	}
}

func TestGetJob_unknownIDReturns404(t *testing.T) {
	s := newTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/api/jobs/does-not-exist", nil)
	req.Header.Set("Authorization", authHeader())
	rr := httptest.NewRecorder()

	s.router.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rr.Code)
	}
}

// --- DELETE /api/jobs/{id} ---

func TestDeleteJob_returns204AndJobIsGone(t *testing.T) {
	s := newTestServer(t)

	body := `{"agent_id":"agent-01","name":"delete-me","source_path":"C:\\src","dest_path":"D:\\backup"}`
	req := httptest.NewRequest(http.MethodPost, "/api/jobs", bytes.NewBufferString(body))
	req.Header.Set("Authorization", authHeader())
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	s.router.ServeHTTP(rr, req)

	var created Job
	json.NewDecoder(rr.Body).Decode(&created)

	req2 := httptest.NewRequest(http.MethodDelete, "/api/jobs/"+created.ID, nil)
	req2.Header.Set("Authorization", authHeader())
	rr2 := httptest.NewRecorder()
	s.router.ServeHTTP(rr2, req2)

	if rr2.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", rr2.Code)
	}

	req3 := httptest.NewRequest(http.MethodGet, "/api/jobs/"+created.ID, nil)
	req3.Header.Set("Authorization", authHeader())
	rr3 := httptest.NewRecorder()
	s.router.ServeHTTP(rr3, req3)

	if rr3.Code != http.StatusNotFound {
		t.Fatalf("expected 404 after delete, got %d", rr3.Code)
	}
}

func TestDeleteJob_unknownIDReturns404(t *testing.T) {
	s := newTestServer(t)

	req := httptest.NewRequest(http.MethodDelete, "/api/jobs/does-not-exist", nil)
	req.Header.Set("Authorization", authHeader())
	rr := httptest.NewRecorder()

	s.router.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rr.Code)
	}
}

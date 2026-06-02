package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// --- ParsePagination unit tests ---

func TestParsePaginationDefaults(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/agents", nil)
	p := ParsePagination(req)
	if p.Page != 1 {
		t.Errorf("expected page=1, got %d", p.Page)
	}
	if p.Limit != 25 {
		t.Errorf("expected limit=25, got %d", p.Limit)
	}
}

func TestParsePaginationCustom(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/agents?page=3&limit=50", nil)
	p := ParsePagination(req)
	if p.Page != 3 {
		t.Errorf("expected page=3, got %d", p.Page)
	}
	if p.Limit != 50 {
		t.Errorf("expected limit=50, got %d", p.Limit)
	}
}

func TestParsePaginationClamped(t *testing.T) {
	// page=0 → 1, limit=500 → 100
	req := httptest.NewRequest(http.MethodGet, "/api/agents?page=0&limit=500", nil)
	p := ParsePagination(req)
	if p.Page != 1 {
		t.Errorf("expected page=1, got %d", p.Page)
	}
	if p.Limit != 100 {
		t.Errorf("expected limit=100, got %d", p.Limit)
	}

	// limit=-1 → 25
	req2 := httptest.NewRequest(http.MethodGet, "/api/agents?limit=-1", nil)
	p2 := ParsePagination(req2)
	if p2.Limit != 25 {
		t.Errorf("expected limit=25 for limit=-1, got %d", p2.Limit)
	}
}

// --- Paginated endpoint tests ---
// These tests assume the handlers have been updated to return PaginatedResponse.
// They will fail until the handlers are updated (that's expected).

// registerAgent is a local helper to POST /api/agents/register.
func registerAgent(t *testing.T, s *Server, agentID, hostname string) {
	t.Helper()
	body := `{"agent_id":"` + agentID + `","hostname":"` + hostname + `","os":"linux","arch":"amd64","version":"0.1.0"}`
	req := httptest.NewRequest(http.MethodPost, "/api/agents/register", bytes.NewBufferString(body))
	req.Header.Set("Authorization", authHeader())
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	s.router.ServeHTTP(rr, req)
	if rr.Code != http.StatusCreated {
		t.Fatalf("registerAgent: expected 201, got %d: %s", rr.Code, rr.Body.String())
	}
}

// createJob is a local helper to POST /api/jobs and return the created Job.
func createJob(t *testing.T, s *Server, agentID, name string) Job {
	t.Helper()
	body := `{"agent_id":"` + agentID + `","name":"` + name + `","source_path":"C:\\src","dest_path":"D:\\backup"}`
	req := httptest.NewRequest(http.MethodPost, "/api/jobs", bytes.NewBufferString(body))
	req.Header.Set("Authorization", authHeader())
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	s.router.ServeHTTP(rr, req)
	if rr.Code != http.StatusCreated {
		t.Fatalf("createJob: expected 201, got %d: %s", rr.Code, rr.Body.String())
	}
	var j Job
	if err := json.NewDecoder(rr.Body).Decode(&j); err != nil {
		t.Fatalf("createJob: failed to decode response: %v", err)
	}
	return j
}

// postJobResult posts a result for a job to generate a run record.
func postJobResult(t *testing.T, s *Server, jobID string) {
	t.Helper()
	body := `{"exit_code":0,"output":"done"}`
	req := httptest.NewRequest(http.MethodPost, "/api/jobs/"+jobID+"/results", bytes.NewBufferString(body))
	req.Header.Set("Authorization", authHeader())
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	s.router.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK && rr.Code != http.StatusCreated && rr.Code != http.StatusNoContent {
		t.Fatalf("postJobResult: unexpected status %d: %s", rr.Code, rr.Body.String())
	}
}

func TestPaginatedAgents(t *testing.T) {
	s := newTestServer(t)

	registerAgent(t, s, "agent-01", "box1")
	registerAgent(t, s, "agent-02", "box2")
	registerAgent(t, s, "agent-03", "box3")

	req := httptest.NewRequest(http.MethodGet, "/api/agents?page=1&limit=2", nil)
	req.Header.Set("Authorization", authHeader())
	rr := httptest.NewRecorder()
	s.router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var resp struct {
		Data  json.RawMessage `json:"data"`
		Total int             `json:"total"`
		Page  int             `json:"page"`
		Pages int             `json:"pages"`
		Limit int             `json:"limit"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.Total != 3 {
		t.Errorf("expected total=3, got %d", resp.Total)
	}
	if resp.Page != 1 {
		t.Errorf("expected page=1, got %d", resp.Page)
	}
	if resp.Pages != 2 {
		t.Errorf("expected pages=2, got %d", resp.Pages)
	}
	if resp.Limit != 2 {
		t.Errorf("expected limit=2, got %d", resp.Limit)
	}

	var items []json.RawMessage
	if err := json.Unmarshal(resp.Data, &items); err != nil {
		t.Fatalf("failed to unmarshal data array: %v", err)
	}
	if len(items) != 2 {
		t.Errorf("expected 2 items in data, got %d", len(items))
	}
}

func TestPaginatedAgentsFiltered(t *testing.T) {
	s := newTestServer(t)

	// Newly registered agents default to status=online.
	registerAgent(t, s, "agent-online-1", "box1")
	registerAgent(t, s, "agent-online-2", "box2")

	req := httptest.NewRequest(http.MethodGet, "/api/agents?status=online&page=1&limit=25", nil)
	req.Header.Set("Authorization", authHeader())
	rr := httptest.NewRecorder()
	s.router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var resp struct {
		Data  json.RawMessage `json:"data"`
		Total int             `json:"total"`
		Page  int             `json:"page"`
		Pages int             `json:"pages"`
		Limit int             `json:"limit"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.Total != 2 {
		t.Errorf("expected total=2 online agents, got %d", resp.Total)
	}
}

func TestPaginatedJobs(t *testing.T) {
	s := newTestServer(t)

	createJob(t, s, "agent-01", "job-1")
	createJob(t, s, "agent-01", "job-2")
	createJob(t, s, "agent-01", "job-3")

	req := httptest.NewRequest(http.MethodGet, "/api/jobs?page=1&limit=2", nil)
	req.Header.Set("Authorization", authHeader())
	rr := httptest.NewRecorder()
	s.router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var resp struct {
		Data  json.RawMessage `json:"data"`
		Total int             `json:"total"`
		Page  int             `json:"page"`
		Pages int             `json:"pages"`
		Limit int             `json:"limit"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.Total != 3 {
		t.Errorf("expected total=3, got %d", resp.Total)
	}
	if resp.Pages != 2 {
		t.Errorf("expected pages=2, got %d", resp.Pages)
	}
	if resp.Limit != 2 {
		t.Errorf("expected limit=2, got %d", resp.Limit)
	}
}

func TestPaginatedJobsFiltered(t *testing.T) {
	s := newTestServer(t)

	// Newly created jobs default to status=pending.
	createJob(t, s, "agent-01", "job-1")
	createJob(t, s, "agent-01", "job-2")

	// Filter by status=pending — expect both.
	req := httptest.NewRequest(http.MethodGet, "/api/jobs?status=pending&page=1&limit=25", nil)
	req.Header.Set("Authorization", authHeader())
	rr := httptest.NewRecorder()
	s.router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var resp struct {
		Data  json.RawMessage `json:"data"`
		Total int             `json:"total"`
		Page  int             `json:"page"`
		Pages int             `json:"pages"`
		Limit int             `json:"limit"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode pending response: %v", err)
	}
	if resp.Total != 2 {
		t.Errorf("expected total=2 pending jobs, got %d", resp.Total)
	}

	// Filter by status=failed — expect none.
	req2 := httptest.NewRequest(http.MethodGet, "/api/jobs?status=failed&page=1&limit=25", nil)
	req2.Header.Set("Authorization", authHeader())
	rr2 := httptest.NewRecorder()
	s.router.ServeHTTP(rr2, req2)

	if rr2.Code != http.StatusOK {
		t.Fatalf("expected 200 for failed filter, got %d: %s", rr2.Code, rr2.Body.String())
	}

	var resp2 struct {
		Data  json.RawMessage `json:"data"`
		Total int             `json:"total"`
		Page  int             `json:"page"`
		Pages int             `json:"pages"`
		Limit int             `json:"limit"`
	}
	if err := json.NewDecoder(rr2.Body).Decode(&resp2); err != nil {
		t.Fatalf("failed to decode failed response: %v", err)
	}
	if resp2.Total != 0 {
		t.Errorf("expected total=0 failed jobs, got %d", resp2.Total)
	}
}

func TestPaginatedJobRuns(t *testing.T) {
	s := newTestServer(t)

	j := createJob(t, s, "agent-01", "run-test-job")
	postJobResult(t, s, j.ID)
	postJobResult(t, s, j.ID)
	postJobResult(t, s, j.ID)

	req := httptest.NewRequest(http.MethodGet, "/api/jobs/"+j.ID+"/runs?page=1&limit=2", nil)
	req.Header.Set("Authorization", authHeader())
	rr := httptest.NewRecorder()
	s.router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var resp struct {
		Data  json.RawMessage `json:"data"`
		Total int             `json:"total"`
		Page  int             `json:"page"`
		Pages int             `json:"pages"`
		Limit int             `json:"limit"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	// Trigger creates 1 run; posting results 3 times updates that same run
	if resp.Total != 1 {
		t.Errorf("expected total=1 (trigger-created run updated 3 times), got %d", resp.Total)
	}
	if resp.Pages != 1 {
		t.Errorf("expected pages=1, got %d", resp.Pages)
	}

	var items []json.RawMessage
	if err := json.Unmarshal(resp.Data, &items); err != nil {
		t.Fatalf("failed to unmarshal data array: %v", err)
	}
	if len(items) != 1 {
		t.Errorf("expected 1 item on page 1, got %d", len(items))
	}
}

func TestPaginationOffsetCalc(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/agents?page=3&limit=25", nil)
	p := ParsePagination(req)

	offset := (p.Page - 1) * p.Limit
	if offset != 50 {
		t.Errorf("expected offset=50, got %d", offset)
	}
}

func TestPaginationEmptyPage(t *testing.T) {
	s := newTestServer(t)

	registerAgent(t, s, "agent-01", "box1")
	registerAgent(t, s, "agent-02", "box2")

	req := httptest.NewRequest(http.MethodGet, "/api/agents?page=99&limit=25", nil)
	req.Header.Set("Authorization", authHeader())
	rr := httptest.NewRecorder()
	s.router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var resp struct {
		Data  json.RawMessage `json:"data"`
		Total int             `json:"total"`
		Page  int             `json:"page"`
		Pages int             `json:"pages"`
		Limit int             `json:"limit"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.Total != 2 {
		t.Errorf("expected total=2, got %d", resp.Total)
	}
	if resp.Pages != 1 {
		t.Errorf("expected pages=1, got %d", resp.Pages)
	}

	var items []json.RawMessage
	if err := json.Unmarshal(resp.Data, &items); err != nil {
		t.Fatalf("failed to unmarshal data array: %v", err)
	}
	if len(items) != 0 {
		t.Errorf("expected 0 items on page 99, got %d", len(items))
	}
}

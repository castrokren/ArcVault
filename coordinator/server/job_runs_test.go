package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// --- GET /api/jobs/{id}/runs ---

func TestGetJobRuns_returnsEmptyArrayWhenNoRuns(t *testing.T) {
	s := newTestServer(t)

	body := `{"agent_id":"agent-01","name":"backup","source_path":"C:\\src","dest_path":"D:\\backup"}`
	req := httptest.NewRequest(http.MethodPost, "/api/jobs", bytes.NewBufferString(body))
	req.Header.Set("Authorization", authHeader())
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	s.router.ServeHTTP(rr, req)

	var created Job
	json.NewDecoder(rr.Body).Decode(&created)

	req2 := httptest.NewRequest(http.MethodGet, "/api/jobs/"+created.ID+"/runs", nil)
	req2.Header.Set("Authorization", authHeader())
	rr2 := httptest.NewRecorder()
	s.router.ServeHTTP(rr2, req2)

	if rr2.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr2.Code, rr2.Body.String())
	}

	var resp PaginatedResponse
	if err := json.NewDecoder(rr2.Body).Decode(&resp); err != nil {
		t.Fatalf("response is not valid JSON: %v", err)
	}
	if resp.Total != 0 {
		t.Errorf("expected total=0, got %d", resp.Total)
	}
}

func TestGetJobRuns_returnsRunsForJob(t *testing.T) {
	s := newTestServer(t)

	body := `{"agent_id":"agent-01","name":"backup","source_path":"C:\\src","dest_path":"D:\\backup"}`
	req := httptest.NewRequest(http.MethodPost, "/api/jobs", bytes.NewBufferString(body))
	req.Header.Set("Authorization", authHeader())
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	s.router.ServeHTTP(rr, req)

	var created Job
	json.NewDecoder(rr.Body).Decode(&created)

	for i := 0; i < 2; i++ {
		result := `{"exit_code":0,"output":"done"}`
		req2 := httptest.NewRequest(http.MethodPost, "/api/jobs/"+created.ID+"/results", bytes.NewBufferString(result))
		req2.Header.Set("Authorization", authHeader())
		req2.Header.Set("Content-Type", "application/json")
		s.router.ServeHTTP(httptest.NewRecorder(), req2)
	}

	req3 := httptest.NewRequest(http.MethodGet, "/api/jobs/"+created.ID+"/runs", nil)
	req3.Header.Set("Authorization", authHeader())
	rr3 := httptest.NewRecorder()
	s.router.ServeHTTP(rr3, req3)

	if rr3.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr3.Code, rr3.Body.String())
	}

	var resp PaginatedResponse
	if err := json.NewDecoder(rr3.Body).Decode(&resp); err != nil {
		t.Fatalf("response is not valid JSON: %v", err)
	}
	if resp.Total != 2 {
		t.Errorf("expected total=2, got %d", resp.Total)
	}
}

func TestGetJobRuns_runsContainCorrectFields(t *testing.T) {
	s := newTestServer(t)

	body := `{"agent_id":"agent-01","name":"backup","source_path":"C:\\src","dest_path":"D:\\backup"}`
	req := httptest.NewRequest(http.MethodPost, "/api/jobs", bytes.NewBufferString(body))
	req.Header.Set("Authorization", authHeader())
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	s.router.ServeHTTP(rr, req)

	var created Job
	json.NewDecoder(rr.Body).Decode(&created)

	result := `{"exit_code":1,"output":"something went wrong"}`
	req2 := httptest.NewRequest(http.MethodPost, "/api/jobs/"+created.ID+"/results", bytes.NewBufferString(result))
	req2.Header.Set("Authorization", authHeader())
	req2.Header.Set("Content-Type", "application/json")
	s.router.ServeHTTP(httptest.NewRecorder(), req2)

	req3 := httptest.NewRequest(http.MethodGet, "/api/jobs/"+created.ID+"/runs", nil)
	req3.Header.Set("Authorization", authHeader())
	rr3 := httptest.NewRecorder()
	s.router.ServeHTTP(rr3, req3)

	var resp struct {
		Data  []JobRun `json:"data"`
		Total int      `json:"total"`
	}
	json.NewDecoder(rr3.Body).Decode(&resp)

	if resp.Total != 1 {
		t.Fatalf("expected 1 run, got %d", resp.Total)
	}
	run := resp.Data[0]
	if run.ID == "" {
		t.Error("expected non-empty run ID")
	}
	if run.JobID != created.ID {
		t.Errorf("expected job_id %q, got %q", created.ID, run.JobID)
	}
	if run.ExitCode != 1 {
		t.Errorf("expected exit_code 1, got %d", run.ExitCode)
	}
	if run.Output != "something went wrong" {
		t.Errorf("expected output 'something went wrong', got %q", run.Output)
	}
	if run.FinishedAt == "" {
		t.Error("expected non-empty finished_at")
	}
}

func TestGetJobRuns_unknownJobIDReturns404(t *testing.T) {
	s := newTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/api/jobs/does-not-exist/runs", nil)
	req.Header.Set("Authorization", authHeader())
	rr := httptest.NewRecorder()
	s.router.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rr.Code)
	}
}

func TestGetJobRuns_onlyReturnsRunsForRequestedJob(t *testing.T) {
	s := newTestServer(t)

	var jobIDs []string
	for _, name := range []string{"job-a", "job-b"} {
		body := `{"agent_id":"agent-01","name":"` + name + `","source_path":"C:\\src","dest_path":"D:\\backup"}`
		req := httptest.NewRequest(http.MethodPost, "/api/jobs", bytes.NewBufferString(body))
		req.Header.Set("Authorization", authHeader())
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		s.router.ServeHTTP(rr, req)
		var j Job
		json.NewDecoder(rr.Body).Decode(&j)
		jobIDs = append(jobIDs, j.ID)
	}

	result := `{"exit_code":0,"output":"done"}`
	req2 := httptest.NewRequest(http.MethodPost, "/api/jobs/"+jobIDs[0]+"/results", bytes.NewBufferString(result))
	req2.Header.Set("Authorization", authHeader())
	req2.Header.Set("Content-Type", "application/json")
	s.router.ServeHTTP(httptest.NewRecorder(), req2)

	req3 := httptest.NewRequest(http.MethodGet, "/api/jobs/"+jobIDs[1]+"/runs", nil)
	req3.Header.Set("Authorization", authHeader())
	rr3 := httptest.NewRecorder()
	s.router.ServeHTTP(rr3, req3)

	var resp PaginatedResponse
	json.NewDecoder(rr3.Body).Decode(&resp)
	if resp.Total != 0 {
		t.Errorf("expected 0 runs for job-b, got %d", resp.Total)
	}
}

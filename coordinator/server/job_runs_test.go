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

	// create a job
	body := `{"agent_id":"agent-01","name":"backup","source_path":"C:\\src","dest_path":"D:\\backup"}`
	req := httptest.NewRequest(http.MethodPost, "/api/jobs", bytes.NewBufferString(body))
	req.Header.Set("Authorization", authHeader())
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	s.router.ServeHTTP(rr, req)

	var created Job
	json.NewDecoder(rr.Body).Decode(&created)

	// get runs -- should be empty
	req2 := httptest.NewRequest(http.MethodGet, "/api/jobs/"+created.ID+"/runs", nil)
	req2.Header.Set("Authorization", authHeader())
	rr2 := httptest.NewRecorder()
	s.router.ServeHTTP(rr2, req2)

	if rr2.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr2.Code, rr2.Body.String())
	}

	var runs []JobRun
	if err := json.NewDecoder(rr2.Body).Decode(&runs); err != nil {
		t.Fatalf("response is not valid JSON: %v", err)
	}
	if len(runs) != 0 {
		t.Errorf("expected empty list, got %d runs", len(runs))
	}
}

func TestGetJobRuns_returnsRunsForJob(t *testing.T) {
	s := newTestServer(t)

	// create a job
	body := `{"agent_id":"agent-01","name":"backup","source_path":"C:\\src","dest_path":"D:\\backup"}`
	req := httptest.NewRequest(http.MethodPost, "/api/jobs", bytes.NewBufferString(body))
	req.Header.Set("Authorization", authHeader())
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	s.router.ServeHTTP(rr, req)

	var created Job
	json.NewDecoder(rr.Body).Decode(&created)

	// post two results
	for i := 0; i < 2; i++ {
		result := `{"exit_code":0,"output":"done"}`
		req2 := httptest.NewRequest(http.MethodPost, "/api/jobs/"+created.ID+"/results", bytes.NewBufferString(result))
		req2.Header.Set("Authorization", authHeader())
		req2.Header.Set("Content-Type", "application/json")
		s.router.ServeHTTP(httptest.NewRecorder(), req2)
	}

	// get runs
	req3 := httptest.NewRequest(http.MethodGet, "/api/jobs/"+created.ID+"/runs", nil)
	req3.Header.Set("Authorization", authHeader())
	rr3 := httptest.NewRecorder()
	s.router.ServeHTTP(rr3, req3)

	if rr3.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr3.Code, rr3.Body.String())
	}

	var runs []JobRun
	if err := json.NewDecoder(rr3.Body).Decode(&runs); err != nil {
		t.Fatalf("response is not valid JSON: %v", err)
	}
	if len(runs) != 2 {
		t.Errorf("expected 2 runs, got %d", len(runs))
	}
}

func TestGetJobRuns_runsContainCorrectFields(t *testing.T) {
	s := newTestServer(t)

	// create a job
	body := `{"agent_id":"agent-01","name":"backup","source_path":"C:\\src","dest_path":"D:\\backup"}`
	req := httptest.NewRequest(http.MethodPost, "/api/jobs", bytes.NewBufferString(body))
	req.Header.Set("Authorization", authHeader())
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	s.router.ServeHTTP(rr, req)

	var created Job
	json.NewDecoder(rr.Body).Decode(&created)

	// post a result
	result := `{"exit_code":1,"output":"something went wrong"}`
	req2 := httptest.NewRequest(http.MethodPost, "/api/jobs/"+created.ID+"/results", bytes.NewBufferString(result))
	req2.Header.Set("Authorization", authHeader())
	req2.Header.Set("Content-Type", "application/json")
	s.router.ServeHTTP(httptest.NewRecorder(), req2)

	// get runs
	req3 := httptest.NewRequest(http.MethodGet, "/api/jobs/"+created.ID+"/runs", nil)
	req3.Header.Set("Authorization", authHeader())
	rr3 := httptest.NewRecorder()
	s.router.ServeHTTP(rr3, req3)

	var runs []JobRun
	json.NewDecoder(rr3.Body).Decode(&runs)

	run := runs[0]
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

	// create two jobs
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

	// post results only for job-a
	result := `{"exit_code":0,"output":"done"}`
	req2 := httptest.NewRequest(http.MethodPost, "/api/jobs/"+jobIDs[0]+"/results", bytes.NewBufferString(result))
	req2.Header.Set("Authorization", authHeader())
	req2.Header.Set("Content-Type", "application/json")
	s.router.ServeHTTP(httptest.NewRecorder(), req2)

	// get runs for job-b -- should be empty
	req3 := httptest.NewRequest(http.MethodGet, "/api/jobs/"+jobIDs[1]+"/runs", nil)
	req3.Header.Set("Authorization", authHeader())
	rr3 := httptest.NewRecorder()
	s.router.ServeHTTP(rr3, req3)

	var runs []JobRun
	json.NewDecoder(rr3.Body).Decode(&runs)
	if len(runs) != 0 {
		t.Errorf("expected 0 runs for job-b, got %d", len(runs))
	}
}

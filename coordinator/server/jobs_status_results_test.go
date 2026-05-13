package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// --- PATCH /api/jobs/{id}/status ---

func TestUpdateJobStatus_setsStatusToRunning(t *testing.T) {
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

	// patch status to running
	patch := `{"status":"running"}`
	req2 := httptest.NewRequest(http.MethodPatch, "/api/jobs/"+created.ID+"/status", bytes.NewBufferString(patch))
	req2.Header.Set("Authorization", authHeader())
	req2.Header.Set("Content-Type", "application/json")
	rr2 := httptest.NewRecorder()
	s.router.ServeHTTP(rr2, req2)

	if rr2.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr2.Code, rr2.Body.String())
	}

	var updated Job
	if err := json.NewDecoder(rr2.Body).Decode(&updated); err != nil {
		t.Fatalf("response is not valid JSON: %v", err)
	}
	if updated.Status != "running" {
		t.Errorf("expected status 'running', got %q", updated.Status)
	}
}

func TestUpdateJobStatus_setsStatusToCompleted(t *testing.T) {
	s := newTestServer(t)

	body := `{"agent_id":"agent-01","name":"backup","source_path":"C:\\src","dest_path":"D:\\backup"}`
	req := httptest.NewRequest(http.MethodPost, "/api/jobs", bytes.NewBufferString(body))
	req.Header.Set("Authorization", authHeader())
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	s.router.ServeHTTP(rr, req)

	var created Job
	json.NewDecoder(rr.Body).Decode(&created)

	patch := `{"status":"completed"}`
	req2 := httptest.NewRequest(http.MethodPatch, "/api/jobs/"+created.ID+"/status", bytes.NewBufferString(patch))
	req2.Header.Set("Authorization", authHeader())
	req2.Header.Set("Content-Type", "application/json")
	rr2 := httptest.NewRecorder()
	s.router.ServeHTTP(rr2, req2)

	if rr2.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr2.Code, rr2.Body.String())
	}

	var updated Job
	json.NewDecoder(rr2.Body).Decode(&updated)
	if updated.Status != "completed" {
		t.Errorf("expected status 'completed', got %q", updated.Status)
	}
}

func TestUpdateJobStatus_invalidStatusReturnsBadRequest(t *testing.T) {
	s := newTestServer(t)

	body := `{"agent_id":"agent-01","name":"backup","source_path":"C:\\src","dest_path":"D:\\backup"}`
	req := httptest.NewRequest(http.MethodPost, "/api/jobs", bytes.NewBufferString(body))
	req.Header.Set("Authorization", authHeader())
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	s.router.ServeHTTP(rr, req)

	var created Job
	json.NewDecoder(rr.Body).Decode(&created)

	patch := `{"status":"exploded"}`
	req2 := httptest.NewRequest(http.MethodPatch, "/api/jobs/"+created.ID+"/status", bytes.NewBufferString(patch))
	req2.Header.Set("Authorization", authHeader())
	req2.Header.Set("Content-Type", "application/json")
	rr2 := httptest.NewRecorder()
	s.router.ServeHTTP(rr2, req2)

	if rr2.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr2.Code)
	}
}

func TestUpdateJobStatus_unknownIDReturns404(t *testing.T) {
	s := newTestServer(t)

	patch := `{"status":"running"}`
	req := httptest.NewRequest(http.MethodPatch, "/api/jobs/does-not-exist/status", bytes.NewBufferString(patch))
	req.Header.Set("Authorization", authHeader())
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	s.router.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rr.Code)
	}
}

// --- POST /api/jobs/{id}/results ---

func TestPostJobResults_storesResultAndReturns201(t *testing.T) {
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

	// post result
	result := `{"exit_code":0,"output":"Copied 42 files"}`
	req2 := httptest.NewRequest(http.MethodPost, "/api/jobs/"+created.ID+"/results", bytes.NewBufferString(result))
	req2.Header.Set("Authorization", authHeader())
	req2.Header.Set("Content-Type", "application/json")
	rr2 := httptest.NewRecorder()
	s.router.ServeHTTP(rr2, req2)

	if rr2.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rr2.Code, rr2.Body.String())
	}

	var run JobRun
	if err := json.NewDecoder(rr2.Body).Decode(&run); err != nil {
		t.Fatalf("response is not valid JSON: %v", err)
	}
	if run.ID == "" {
		t.Error("expected non-empty run ID")
	}
	if run.JobID != created.ID {
		t.Errorf("expected job_id %q, got %q", created.ID, run.JobID)
	}
	if run.ExitCode != 0 {
		t.Errorf("expected exit_code 0, got %d", run.ExitCode)
	}
	if run.Output != "Copied 42 files" {
		t.Errorf("expected output 'Copied 42 files', got %q", run.Output)
	}
}

func TestPostJobResults_nonZeroExitCode(t *testing.T) {
	s := newTestServer(t)

	body := `{"agent_id":"agent-01","name":"backup","source_path":"C:\\src","dest_path":"D:\\backup"}`
	req := httptest.NewRequest(http.MethodPost, "/api/jobs", bytes.NewBufferString(body))
	req.Header.Set("Authorization", authHeader())
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	s.router.ServeHTTP(rr, req)

	var created Job
	json.NewDecoder(rr.Body).Decode(&created)

	result := `{"exit_code":1,"output":"ERROR: source path not found"}`
	req2 := httptest.NewRequest(http.MethodPost, "/api/jobs/"+created.ID+"/results", bytes.NewBufferString(result))
	req2.Header.Set("Authorization", authHeader())
	req2.Header.Set("Content-Type", "application/json")
	rr2 := httptest.NewRecorder()
	s.router.ServeHTTP(rr2, req2)

	if rr2.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rr2.Code, rr2.Body.String())
	}

	var run JobRun
	json.NewDecoder(rr2.Body).Decode(&run)
	if run.ExitCode != 1 {
		t.Errorf("expected exit_code 1, got %d", run.ExitCode)
	}
}

func TestPostJobResults_unknownJobIDReturns404(t *testing.T) {
	s := newTestServer(t)

	result := `{"exit_code":0,"output":"done"}`
	req := httptest.NewRequest(http.MethodPost, "/api/jobs/does-not-exist/results", bytes.NewBufferString(result))
	req.Header.Set("Authorization", authHeader())
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	s.router.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rr.Code)
	}
}

func TestPostJobResults_unauthenticatedReturns401(t *testing.T) {
	s := newTestServer(t)

	result := `{"exit_code":0,"output":"done"}`
	req := httptest.NewRequest(http.MethodPost, "/api/jobs/any-id/results", bytes.NewBufferString(result))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	s.router.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rr.Code)
	}
}

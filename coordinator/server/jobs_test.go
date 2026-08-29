package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

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

// --- Group dispatch (fan-out) ---

func TestCreateJob_groupDispatchWithMembers(t *testing.T) {
	s := newTestServer(t)

	// Create a group with 3 members
	group, err := s.db.CreateGroup("test-group", "Test group")
	if err != nil {
		t.Fatalf("failed to create group: %v", err)
	}

	for i := 1; i <= 3; i++ {
		agentID := "agent-0" + string(rune('0'+i))
		if err := s.db.AddAgentToGroup(group.ID, agentID); err != nil {
			t.Fatalf("failed to add agent to group: %v", err)
		}
	}

	// Create job with group_id
	body := fmt.Sprintf(`{"group_id":%d,"name":"batch-backup","source_path":"C:\\src","dest_path":"D:\\backup"}`, group.ID)
	req := httptest.NewRequest(http.MethodPost, "/api/jobs", bytes.NewBufferString(body))
	req.Header.Set("Authorization", authHeader())
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	s.router.ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rr.Code, rr.Body.String())
	}

	var resp map[string]interface{}
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("response is not valid JSON: %v", err)
	}

	// Verify response structure
	if resp["dispatch_id"] == nil {
		t.Error("expected dispatch_id in response")
	}
	if resp["group_id"] != float64(group.ID) {
		t.Errorf("expected group_id %d, got %v", group.ID, resp["group_id"])
	}

	jobsList := resp["jobs"].([]interface{})
	if len(jobsList) != 3 {
		t.Errorf("expected 3 jobs, got %d", len(jobsList))
	}

	// Verify all jobs have correct properties
	for i, jobItem := range jobsList {
		job := jobItem.(map[string]interface{})
		if job["name"] != "batch-backup" {
			t.Errorf("job %d: expected name 'batch-backup', got %q", i, job["name"])
		}
		if job["status"] != "pending" {
			t.Errorf("job %d: expected status 'pending', got %q", i, job["status"])
		}
	}
}

func TestCreateJob_groupDispatchEmptyGroupReturnsError(t *testing.T) {
	s := newTestServer(t)

	// Create an empty group (no members)
	group, err := s.db.CreateGroup("empty-group", "Empty group")
	if err != nil {
		t.Fatalf("failed to create group: %v", err)
	}

	// Try to create job with empty group
	body := fmt.Sprintf(`{"group_id":%d,"name":"batch-backup","source_path":"C:\\src","dest_path":"D:\\backup"}`, group.ID)
	req := httptest.NewRequest(http.MethodPost, "/api/jobs", bytes.NewBufferString(body))
	req.Header.Set("Authorization", authHeader())
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	s.router.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
	if !bytes.Contains(rr.Body.Bytes(), []byte("no members")) {
		t.Errorf("expected 'no members' error message, got: %s", rr.Body.String())
	}
}

func TestCreateJob_groupDispatchInvalidGroupReturns404(t *testing.T) {
	s := newTestServer(t)

	// Try to create job with non-existent group
	body := `{"group_id":9999,"name":"batch-backup","source_path":"C:\\src","dest_path":"D:\\backup"}`
	req := httptest.NewRequest(http.MethodPost, "/api/jobs", bytes.NewBufferString(body))
	req.Header.Set("Authorization", authHeader())
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	s.router.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rr.Code)
	}
	if !bytes.Contains(rr.Body.Bytes(), []byte("not found")) {
		t.Errorf("expected 'not found' error message, got: %s", rr.Body.String())
	}
}

func TestCreateJob_cannotProvideBothAgentAndGroupID(t *testing.T) {
	s := newTestServer(t)

	body := `{"agent_id":"agent-01","group_id":1,"name":"backup","source_path":"C:\\src","dest_path":"D:\\backup"}`
	req := httptest.NewRequest(http.MethodPost, "/api/jobs", bytes.NewBufferString(body))
	req.Header.Set("Authorization", authHeader())
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	s.router.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
	if !bytes.Contains(rr.Body.Bytes(), []byte("either agent_id or group_id")) {
		t.Errorf("expected validation error, got: %s", rr.Body.String())
	}
}

func TestCreateJob_mustProvideAgentOrGroupID(t *testing.T) {
	s := newTestServer(t)

	body := `{"name":"backup","source_path":"C:\\src","dest_path":"D:\\backup"}`
	req := httptest.NewRequest(http.MethodPost, "/api/jobs", bytes.NewBufferString(body))
	req.Header.Set("Authorization", authHeader())
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	s.router.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
	if !bytes.Contains(rr.Body.Bytes(), []byte("either agent_id or group_id")) {
		t.Errorf("expected validation error, got: %s", rr.Body.String())
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

	var resp PaginatedResponse
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("response is not valid JSON: %v", err)
	}
	if resp.Total != 0 {
		t.Errorf("expected total=0, got %d", resp.Total)
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

	var resp PaginatedResponse
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("response is not valid JSON: %v", err)
	}
	if resp.Total != 2 {
		t.Errorf("expected total=2, got %d", resp.Total)
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

	var resp PaginatedResponse
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("response is not valid JSON: %v", err)
	}
	if resp.Total != 2 {
		t.Errorf("expected total=2 for agent-01, got %d", resp.Total)
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

// --- POST /api/jobs/{id}/cancel (Phase 20) ---

func TestCancelJob_cancelRunningJobReturnsOK(t *testing.T) {
	s := newTestServer(t)

	// Create a job
	body := `{"agent_id":"agent-01","name":"nightly-backup","source_path":"C:\\src","dest_path":"D:\\backup"}`
	req := httptest.NewRequest(http.MethodPost, "/api/jobs", bytes.NewBufferString(body))
	req.Header.Set("Authorization", authHeader())
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	s.router.ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("failed to create job: %d", rr.Code)
	}

	var job Job
	json.NewDecoder(rr.Body).Decode(&job)
	jobID := job.ID

	// Manually set status to "running" in database (simulates a running job)
	s.db.Conn().Exec(`UPDATE jobs SET status = ? WHERE id = ?`, "running", jobID)

	// Cancel the job
	cancelReq := httptest.NewRequest(http.MethodPost, "/api/jobs/"+jobID+"/cancel", nil)
	cancelReq.Header.Set("Authorization", authHeader())
	cancelRR := httptest.NewRecorder()
	s.router.ServeHTTP(cancelRR, cancelReq)

	if cancelRR.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", cancelRR.Code, cancelRR.Body.String())
	}

	// Verify response contains the job with status="canceling"
	var cancelledJob Job
	if err := json.NewDecoder(cancelRR.Body).Decode(&cancelledJob); err != nil {
		t.Fatalf("response is not valid JSON: %v", err)
	}
	if cancelledJob.Status != "canceling" {
		t.Errorf("expected status 'canceling', got %q", cancelledJob.Status)
	}

	// Verify status was actually persisted to database (fetch the job again)
	getReq := httptest.NewRequest(http.MethodGet, "/api/jobs/"+jobID, nil)
	getReq.Header.Set("Authorization", authHeader())
	getRR := httptest.NewRecorder()
	s.router.ServeHTTP(getRR, getReq)

	if getRR.Code != http.StatusOK {
		t.Fatalf("failed to fetch job after cancel: %d", getRR.Code)
	}

	var persistedJob Job
	if err := json.NewDecoder(getRR.Body).Decode(&persistedJob); err != nil {
		t.Fatalf("response is not valid JSON: %v", err)
	}
	if persistedJob.Status != "canceling" {
		t.Errorf("expected persisted status 'canceling', got %q", persistedJob.Status)
	}
}

func TestCancelJob_cancelPendingJobSucceeds(t *testing.T) {
	s := newTestServer(t)

	// Create a job (status will be "pending" by default)
	body := `{"agent_id":"agent-01","name":"nightly-backup","source_path":"C:\\src","dest_path":"D:\\backup"}`
	req := httptest.NewRequest(http.MethodPost, "/api/jobs", bytes.NewBufferString(body))
	req.Header.Set("Authorization", authHeader())
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	s.router.ServeHTTP(rr, req)

	var job Job
	if err := json.NewDecoder(rr.Body).Decode(&job); err != nil {
		t.Fatalf("failed to decode created job: %v", err)
	}
	jobID := job.ID

	// Cancel the pending job — should succeed
	cancelReq := httptest.NewRequest(http.MethodPost, "/api/jobs/"+jobID+"/cancel", nil)
	cancelReq.Header.Set("Authorization", authHeader())
	cancelRR := httptest.NewRecorder()
	s.router.ServeHTTP(cancelRR, cancelReq)

	if cancelRR.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", cancelRR.Code)
	}

	var cancelled Job
	if err := json.NewDecoder(cancelRR.Body).Decode(&cancelled); err != nil {
		t.Fatalf("failed to decode cancelled job: %v", err)
	}
	if cancelled.Status != "cancelled" {
		t.Errorf("expected status 'cancelled', got %q", cancelled.Status)
	}
}

func TestCancelJob_cancelNonexistentJobReturns404(t *testing.T) {
	s := newTestServer(t)

	// Try to cancel a non-existent job
	cancelReq := httptest.NewRequest(http.MethodPost, "/api/jobs/job-nonexistent/cancel", nil)
	cancelReq.Header.Set("Authorization", authHeader())
	cancelRR := httptest.NewRecorder()
	s.router.ServeHTTP(cancelRR, cancelReq)

	if cancelRR.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", cancelRR.Code)
	}
}

func TestCancelJob_unauthenticatedReturns401(t *testing.T) {
	s := newTestServer(t)

	// Create a job
	body := `{"agent_id":"agent-01","name":"nightly-backup","source_path":"C:\\src","dest_path":"D:\\backup"}`
	req := httptest.NewRequest(http.MethodPost, "/api/jobs", bytes.NewBufferString(body))
	req.Header.Set("Authorization", authHeader())
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	s.router.ServeHTTP(rr, req)

	var job Job
	json.NewDecoder(rr.Body).Decode(&job)
	jobID := job.ID

	// Try to cancel without authentication
	cancelReq := httptest.NewRequest(http.MethodPost, "/api/jobs/"+jobID+"/cancel", nil)
	cancelRR := httptest.NewRecorder()
	s.router.ServeHTTP(cancelRR, cancelReq)

	if cancelRR.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", cancelRR.Code)
	}
}

// --- POST /api/jobs/{id}/progress (Phase 20 progress tracking) ---

func TestPostJobProgress_updatesProgressForRunningJob(t *testing.T) {
	s := newTestServer(t)

	// Create and start a job
	body := `{"agent_id":"agent-01","name":"nightly-backup","source_path":"C:\\src","dest_path":"D:\\backup"}`
	req := httptest.NewRequest(http.MethodPost, "/api/jobs", bytes.NewBufferString(body))
	req.Header.Set("Authorization", authHeader())
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	s.router.ServeHTTP(rr, req)

	var job Job
	json.NewDecoder(rr.Body).Decode(&job)
	jobID := job.ID

	// Set job to running
	s.db.Conn().Exec(`UPDATE jobs SET status = ? WHERE id = ?`, "running", jobID)

	// Post progress update (Phase 21a-3 format)
	progressBody := `{
		"percentage": 75,
		"logs": ["Starting backup", "50% complete"],
		"status": "running"
	}`
	progressReq := httptest.NewRequest(http.MethodPost, "/api/jobs/"+jobID+"/progress", bytes.NewBufferString(progressBody))
	progressReq.Header.Set("Authorization", authHeader())
	progressReq.Header.Set("Content-Type", "application/json")
	progressRR := httptest.NewRecorder()
	s.router.ServeHTTP(progressRR, progressReq)

	if progressRR.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", progressRR.Code, progressRR.Body.String())
	}

	// Verify progress was stored in job_runs
	var progress int
	var status string
	err := s.db.Conn().QueryRow(
		`SELECT progress, status FROM job_runs WHERE job_id = ? ORDER BY started_at DESC LIMIT 1`,
		jobID,
	).Scan(&progress, &status)
	if err != nil {
		t.Fatalf("failed to query progress: %v", err)
	}
	if progress != 75 {
		t.Errorf("expected 75%% progress, got %d", progress)
	}
	if status != "running" {
		t.Errorf("expected running status, got %s", status)
	}

	// Verify logs were stored
	var logCount int
	s.db.Conn().QueryRow(
		`SELECT COUNT(*) FROM job_logs WHERE job_id = ?`,
		jobID,
	).Scan(&logCount)
	if logCount != 2 {
		t.Errorf("expected 2 logs, got %d", logCount)
	}
}

func TestPostJobProgress_nonexistentJobReturns404(t *testing.T) {
	s := newTestServer(t)

	progressBody := `{"percentage": 50, "logs": [], "status": "running"}`
	progressReq := httptest.NewRequest(http.MethodPost, "/api/jobs/job-nonexistent/progress", bytes.NewBufferString(progressBody))
	progressReq.Header.Set("Authorization", authHeader())
	progressReq.Header.Set("Content-Type", "application/json")
	progressRR := httptest.NewRecorder()
	s.router.ServeHTTP(progressRR, progressReq)

	if progressRR.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", progressRR.Code)
	}
}

func TestPostJobProgress_unauthenticatedReturns401(t *testing.T) {
	s := newTestServer(t)

	// Create a job
	body := `{"agent_id":"agent-01","name":"nightly-backup","source_path":"C:\\src","dest_path":"D:\\backup"}`
	req := httptest.NewRequest(http.MethodPost, "/api/jobs", bytes.NewBufferString(body))
	req.Header.Set("Authorization", authHeader())
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	s.router.ServeHTTP(rr, req)

	var job Job
	json.NewDecoder(rr.Body).Decode(&job)
	jobID := job.ID

	// Try to post progress without authentication
	progressBody := `{"files_processed": 10, "bytes_transferred": 1000, "total_files": 100, "total_bytes": 10000}`
	progressReq := httptest.NewRequest(http.MethodPost, "/api/jobs/"+jobID+"/progress", bytes.NewBufferString(progressBody))
	progressRR := httptest.NewRecorder()
	s.router.ServeHTTP(progressRR, progressReq)

	if progressRR.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", progressRR.Code)
	}
}

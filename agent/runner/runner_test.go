package runner

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// fakeCoordinator stands in for the real coordinator during tests.
// It tracks calls so tests can assert what the runner did.
type fakeCoordinator struct {
	jobs          []jobResponse
	statusUpdates []statusUpdate
	results       []jobResult
}

type jobResponse struct {
	ID         string `json:"id"`
	AgentID    string `json:"agent_id"`
	Name       string `json:"name"`
	SourcePath string `json:"source_path"`
	DestPath   string `json:"dest_path"`
	Status     string `json:"status"`
}

type statusUpdate struct {
	JobID  string
	Status string
}

type jobResult struct {
	JobID    string
	ExitCode int
	Output   string
}

func (f *fakeCoordinator) server(t *testing.T) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()

	// GET /api/jobs?agent_id=...&status=pending
	mux.HandleFunc("GET /api/jobs", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(f.jobs)
	})

	// PATCH /api/jobs/{id}/status
	mux.HandleFunc("PATCH /api/jobs/{id}/status", func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		var body struct {
			Status string `json:"status"`
		}
		json.NewDecoder(r.Body).Decode(&body)
		f.statusUpdates = append(f.statusUpdates, statusUpdate{JobID: id, Status: body.Status})

		// return updated job
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(jobResponse{ID: id, Status: body.Status})
	})

	// POST /api/jobs/{id}/results
	mux.HandleFunc("POST /api/jobs/{id}/results", func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		var body struct {
			ExitCode int    `json:"exit_code"`
			Output   string `json:"output"`
		}
		json.NewDecoder(r.Body).Decode(&body)
		f.results = append(f.results, jobResult{JobID: id, ExitCode: body.ExitCode, Output: body.Output})
		w.WriteHeader(http.StatusCreated)
	})

	return httptest.NewServer(mux)
}

// --- tests ---

func TestRunner_claimsAPendingJob(t *testing.T) {
	fake := &fakeCoordinator{
		jobs: []jobResponse{
			{ID: "job-001", AgentID: "agent-01", Name: "backup", SourcePath: "C:\\src", DestPath: "D:\\backup", Status: "pending"},
		},
	}
	srv := fake.server(t)
	defer srv.Close()

	cfg := Config{
		AgentID:        "agent-01",
		CoordinatorURL: srv.URL,
		AuthToken:      "test-token",
		PollInterval:   50 * time.Millisecond,
	}

	r := New(cfg, echoExecutor)
	go r.Start()
	defer r.Stop()

	// wait for one poll cycle
	time.Sleep(200 * time.Millisecond)

	if len(fake.statusUpdates) == 0 {
		t.Fatal("expected runner to claim the job, but no status updates were made")
	}
	first := fake.statusUpdates[0]
	if first.JobID != "job-001" {
		t.Errorf("expected job-001 to be claimed, got %q", first.JobID)
	}
	if first.Status != "running" {
		t.Errorf("expected status 'running', got %q", first.Status)
	}
}

func TestRunner_postsResultAfterExecution(t *testing.T) {
	fake := &fakeCoordinator{
		jobs: []jobResponse{
			{ID: "job-002", AgentID: "agent-01", Name: "backup", SourcePath: "C:\\src", DestPath: "D:\\backup", Status: "pending"},
		},
	}
	srv := fake.server(t)
	defer srv.Close()

	cfg := Config{
		AgentID:        "agent-01",
		CoordinatorURL: srv.URL,
		AuthToken:      "test-token",
		PollInterval:   50 * time.Millisecond,
	}

	r := New(cfg, echoExecutor)
	go r.Start()
	defer r.Stop()

	time.Sleep(200 * time.Millisecond)

	if len(fake.results) == 0 {
		t.Fatal("expected runner to post results, but none were posted")
	}
	result := fake.results[0]
	if result.JobID != "job-002" {
		t.Errorf("expected results for job-002, got %q", result.JobID)
	}
}

func TestRunner_marksJobCompletedOnSuccess(t *testing.T) {
	fake := &fakeCoordinator{
		jobs: []jobResponse{
			{ID: "job-003", AgentID: "agent-01", Name: "backup", SourcePath: "C:\\src", DestPath: "D:\\backup", Status: "pending"},
		},
	}
	srv := fake.server(t)
	defer srv.Close()

	cfg := Config{
		AgentID:        "agent-01",
		CoordinatorURL: srv.URL,
		AuthToken:      "test-token",
		PollInterval:   50 * time.Millisecond,
	}

	r := New(cfg, echoExecutor)
	go r.Start()
	defer r.Stop()

	time.Sleep(200 * time.Millisecond)

	// find the final status update (should be "completed")
	var finalStatus string
	for _, u := range fake.statusUpdates {
		if u.JobID == "job-003" {
			finalStatus = u.Status
		}
	}
	if finalStatus != "completed" {
		t.Errorf("expected final status 'completed', got %q", finalStatus)
	}
}

func TestRunner_marksJobFailedOnNonZeroExitCode(t *testing.T) {
	fake := &fakeCoordinator{
		jobs: []jobResponse{
			{ID: "job-004", AgentID: "agent-01", Name: "backup", SourcePath: "C:\\src", DestPath: "D:\\backup", Status: "pending"},
		},
	}
	srv := fake.server(t)
	defer srv.Close()

	cfg := Config{
		AgentID:        "agent-01",
		CoordinatorURL: srv.URL,
		AuthToken:      "test-token",
		PollInterval:   50 * time.Millisecond,
	}

	r := New(cfg, failExecutor) // always returns exit code 1
	go r.Start()
	defer r.Stop()

	time.Sleep(200 * time.Millisecond)

	var finalStatus string
	for _, u := range fake.statusUpdates {
		if u.JobID == "job-004" {
			finalStatus = u.Status
		}
	}
	if finalStatus != "failed" {
		t.Errorf("expected final status 'failed', got %q", finalStatus)
	}
}

func TestRunner_doesNothingWhenNoJobsPending(t *testing.T) {
	fake := &fakeCoordinator{
		jobs: []jobResponse{}, // empty
	}
	srv := fake.server(t)
	defer srv.Close()

	cfg := Config{
		AgentID:        "agent-01",
		CoordinatorURL: srv.URL,
		AuthToken:      "test-token",
		PollInterval:   50 * time.Millisecond,
	}

	r := New(cfg, echoExecutor)
	go r.Start()
	defer r.Stop()

	time.Sleep(200 * time.Millisecond)

	if len(fake.statusUpdates) != 0 {
		t.Errorf("expected no status updates, got %d", len(fake.statusUpdates))
	}
	if len(fake.results) != 0 {
		t.Errorf("expected no results, got %d", len(fake.results))
	}
}

// --- test executors ---

// echoExecutor simulates a successful backup (exit code 0)
func echoExecutor(job Job) (exitCode int, output string) {
	return 0, "ok"
}

// failExecutor simulates a failed backup (exit code 1)
func failExecutor(job Job) (exitCode int, output string) {
	return 1, "error: source not found"
}

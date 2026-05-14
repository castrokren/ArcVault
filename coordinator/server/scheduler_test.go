package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// --- job scheduler ---

func TestScheduler_resetsScheduledJobToPending(t *testing.T) {
	s := newTestServer(t)

	// create a scheduled job and mark it completed
	body := `{"agent_id":"agent-01","name":"scheduled-backup","source_path":"C:\\src","dest_path":"D:\\backup","schedule":"* * * * *"}`
	req := httptest.NewRequest(http.MethodPost, "/api/jobs", bytes.NewBufferString(body))
	req.Header.Set("Authorization", authHeader())
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	s.router.ServeHTTP(rr, req)

	var created Job
	json.NewDecoder(rr.Body).Decode(&created)

	// mark it completed (simulating a previous run)
	patch := `{"status":"completed"}`
	req2 := httptest.NewRequest(http.MethodPatch, "/api/jobs/"+created.ID+"/status", bytes.NewBufferString(patch))
	req2.Header.Set("Authorization", authHeader())
	req2.Header.Set("Content-Type", "application/json")
	s.router.ServeHTTP(httptest.NewRecorder(), req2)

	// trigger scheduler manually
	s.triggerScheduledJobs()

	// job should be back to pending
	req3 := httptest.NewRequest(http.MethodGet, "/api/jobs/"+created.ID, nil)
	req3.Header.Set("Authorization", authHeader())
	rr3 := httptest.NewRecorder()
	s.router.ServeHTTP(rr3, req3)

	var job Job
	json.NewDecoder(rr3.Body).Decode(&job)

	if job.Status != "pending" {
		t.Errorf("expected status 'pending' after scheduler tick, got %q", job.Status)
	}
}

func TestScheduler_doesNotResetJobWithNoSchedule(t *testing.T) {
	s := newTestServer(t)

	// create a job WITHOUT a schedule and mark it completed
	body := `{"agent_id":"agent-01","name":"one-shot","source_path":"C:\\src","dest_path":"D:\\backup"}`
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
	s.router.ServeHTTP(httptest.NewRecorder(), req2)

	s.triggerScheduledJobs()

	// job should remain completed
	req3 := httptest.NewRequest(http.MethodGet, "/api/jobs/"+created.ID, nil)
	req3.Header.Set("Authorization", authHeader())
	rr3 := httptest.NewRecorder()
	s.router.ServeHTTP(rr3, req3)

	var job Job
	json.NewDecoder(rr3.Body).Decode(&job)

	if job.Status != "completed" {
		t.Errorf("expected status 'completed' (no schedule), got %q", job.Status)
	}
}

func TestScheduler_doesNotResetRunningJob(t *testing.T) {
	s := newTestServer(t)

	// create a scheduled job and mark it running
	body := `{"agent_id":"agent-01","name":"running-job","source_path":"C:\\src","dest_path":"D:\\backup","schedule":"* * * * *"}`
	req := httptest.NewRequest(http.MethodPost, "/api/jobs", bytes.NewBufferString(body))
	req.Header.Set("Authorization", authHeader())
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	s.router.ServeHTTP(rr, req)

	var created Job
	json.NewDecoder(rr.Body).Decode(&created)

	patch := `{"status":"running"}`
	req2 := httptest.NewRequest(http.MethodPatch, "/api/jobs/"+created.ID+"/status", bytes.NewBufferString(patch))
	req2.Header.Set("Authorization", authHeader())
	req2.Header.Set("Content-Type", "application/json")
	s.router.ServeHTTP(httptest.NewRecorder(), req2)

	s.triggerScheduledJobs()

	// job should still be running — don't interrupt it
	req3 := httptest.NewRequest(http.MethodGet, "/api/jobs/"+created.ID, nil)
	req3.Header.Set("Authorization", authHeader())
	rr3 := httptest.NewRecorder()
	s.router.ServeHTTP(rr3, req3)

	var job Job
	json.NewDecoder(rr3.Body).Decode(&job)

	if job.Status != "running" {
		t.Errorf("expected status 'running' (job in progress), got %q", job.Status)
	}
}

func TestScheduler_broadcastsEventWhenJobRescheduled(t *testing.T) {
	s := newTestServer(t)

	// create a scheduled job and mark it completed
	body := `{"agent_id":"agent-01","name":"broadcast-test","source_path":"C:\\src","dest_path":"D:\\backup","schedule":"* * * * *"}`
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
	s.router.ServeHTTP(httptest.NewRecorder(), req2)

	// connect WS client
	conn, ts := dialTestWS(t, s)
	defer ts.Close()
	defer conn.Close()
	time.Sleep(50 * time.Millisecond)

	s.triggerScheduledJobs()

	ev := readEvent(t, conn)
	if ev["type"] != "job.updated" {
		t.Errorf("expected type 'job.updated', got %q", ev["type"])
	}
	payload, ok := ev["payload"].(map[string]interface{})
	if !ok {
		t.Fatal("expected payload to be an object")
	}
	if payload["status"] != "pending" {
		t.Errorf("expected payload.status 'pending', got %q", payload["status"])
	}
}

func TestScheduler_noActionWhenNoScheduledJobs(t *testing.T) {
	s := newTestServer(t)
	// should not panic with no scheduled jobs
	s.triggerScheduledJobs()
}

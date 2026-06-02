package notifications

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"arcvault/coordinator/config"
)

func TestSlackNotifier_Send(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("expected Content-Type application/json, got %s", r.Header.Get("Content-Type"))
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	notifier := NewSlackNotifier(&config.SlackConfig{
		WebhookURL: server.URL,
	})

	event := JobFailureEvent{
		JobID:      "job1",
		JobName:    "Test Job",
		AgentID:    "agent1",
		RunID:      "run1",
		StartedAt:  time.Now().Add(-time.Minute),
		FinishedAt: time.Now(),
		ErrorMsg:   "connection timeout",
	}

	if err := notifier.Send(event); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSlackNotifier_HTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	notifier := NewSlackNotifier(&config.SlackConfig{
		WebhookURL: server.URL,
	})

	event := JobFailureEvent{
		JobID:      "job1",
		JobName:    "Test Job",
		AgentID:    "agent1",
		RunID:      "run1",
		StartedAt:  time.Now().Add(-time.Minute),
		FinishedAt: time.Now(),
	}

	if err := notifier.Send(event); err == nil {
		t.Fatal("expected error from HTTP 500, got nil")
	}
}

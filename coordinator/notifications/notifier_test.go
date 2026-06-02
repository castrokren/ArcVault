package notifications

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"arcvault/coordinator/config"
)

// ── helpers ──────────────────────────────────────────────────────────────────

func sampleEvent() JobFailureEvent {
	started := time.Date(2026, 5, 17, 10, 0, 0, 0, time.UTC)
	return JobFailureEvent{
		JobID:      "job-42",
		JobName:    "daily-backup",
		AgentID:    "agent-01",
		RunID:      "run-abc123",
		StartedAt:  started,
		FinishedAt: started.Add(90 * time.Second),
		ErrorMsg:   "exit status 1",
	}
}

// ── Dispatcher ────────────────────────────────────────────────────────────────

func TestNewDispatcher_NilConfig(t *testing.T) {
	d := NewDispatcher(nil)
	if len(d.notifiers) != 0 {
		t.Fatalf("expected 0 notifiers for nil config, got %d", len(d.notifiers))
	}
}

func TestNewDispatcher_OnFailureFalse(t *testing.T) {
	cfg := &config.NotificationConfig{
		OnFailure: false,
		Webhook:   &config.WebhookConfig{URL: "https://example.com", Secret: "s"},
	}
	d := NewDispatcher(cfg)
	if len(d.notifiers) != 0 {
		t.Fatalf("expected 0 notifiers when OnFailure=false, got %d", len(d.notifiers))
	}
}

func TestNewDispatcher_WebhookOnly(t *testing.T) {
	cfg := &config.NotificationConfig{
		OnFailure: true,
		Webhook:   &config.WebhookConfig{URL: "https://example.com", Secret: "s"},
	}
	d := NewDispatcher(cfg)
	if len(d.notifiers) != 1 {
		t.Fatalf("expected 1 notifier, got %d", len(d.notifiers))
	}
}

func TestNewDispatcher_EmailOnly(t *testing.T) {
	cfg := &config.NotificationConfig{
		OnFailure: true,
		Email: &config.EmailConfig{
			SMTPHost: "smtp.example.com",
			SMTPPort: 587,
			From:     "arc@example.com",
			To:       []string{"ops@example.com"},
		},
	}
	d := NewDispatcher(cfg)
	if len(d.notifiers) != 1 {
		t.Fatalf("expected 1 notifier, got %d", len(d.notifiers))
	}
}

func TestNewDispatcher_BothNotifiers(t *testing.T) {
	cfg := &config.NotificationConfig{
		OnFailure: true,
		Webhook:   &config.WebhookConfig{URL: "https://example.com"},
		Email: &config.EmailConfig{
			SMTPHost: "smtp.example.com",
			SMTPPort: 587,
			From:     "arc@example.com",
			To:       []string{"ops@example.com"},
		},
	}
	d := NewDispatcher(cfg)
	if len(d.notifiers) != 2 {
		t.Fatalf("expected 2 notifiers, got %d", len(d.notifiers))
	}
}

func TestNewDispatcher_EmptyWebhookURL(t *testing.T) {
	cfg := &config.NotificationConfig{
		OnFailure: true,
		Webhook:   &config.WebhookConfig{URL: ""},
	}
	d := NewDispatcher(cfg)
	if len(d.notifiers) != 0 {
		t.Fatalf("expected 0 notifiers for empty webhook URL, got %d", len(d.notifiers))
	}
}

func TestDispatch_NoOp(t *testing.T) {
	d := NewDispatcher(nil)
	d.Dispatch(sampleEvent()) // no panic = pass
}

// ── Webhook ───────────────────────────────────────────────────────────────────

func TestWebhookNotifier_SendSuccess(t *testing.T) {
	var received webhookPayload
	var gotSig string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotSig = r.Header.Get("X-ArcVault-Signature")
		if err := json.NewDecoder(r.Body).Decode(&received); err != nil {
			t.Errorf("decode body: %v", err)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	cfg := &config.WebhookConfig{URL: srv.URL, Secret: "test-secret"}
	n := NewWebhookNotifier(cfg)
	event := sampleEvent()

	if err := n.Send(event); err != nil {
		t.Fatalf("Send() error: %v", err)
	}

	if received.Event != "job.failed" {
		t.Errorf("event = %q, want %q", received.Event, "job.failed")
	}
	if received.JobID != event.JobID {
		t.Errorf("job_id = %q, want %q", received.JobID, event.JobID)
	}
	if received.AgentID != event.AgentID {
		t.Errorf("agent_id = %q, want %q", received.AgentID, event.AgentID)
	}
	if received.ErrorMsg != event.ErrorMsg {
		t.Errorf("error = %q, want %q", received.ErrorMsg, event.ErrorMsg)
	}
	if received.DurationMs != 90000 {
		t.Errorf("duration_ms = %d, want 90000", received.DurationMs)
	}
	if !strings.HasPrefix(gotSig, "sha256=") {
		t.Errorf("signature header = %q, want sha256=...", gotSig)
	}
}

func TestWebhookNotifier_NoSecretOmitsSig(t *testing.T) {
	var gotSig string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotSig = r.Header.Get("X-ArcVault-Signature")
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	cfg := &config.WebhookConfig{URL: srv.URL, Secret: ""}
	n := NewWebhookNotifier(cfg)
	if err := n.Send(sampleEvent()); err != nil {
		t.Fatalf("Send() error: %v", err)
	}
	if gotSig != "" {
		t.Errorf("expected no signature header when secret is empty, got %q", gotSig)
	}
}

func TestWebhookNotifier_NonSuccessStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	cfg := &config.WebhookConfig{URL: srv.URL}
	n := NewWebhookNotifier(cfg)
	if err := n.Send(sampleEvent()); err == nil {
		t.Fatal("expected error for 500 response, got nil")
	}
}

func TestWebhookNotifier_BadURL(t *testing.T) {
	cfg := &config.WebhookConfig{URL: "http://127.0.0.1:0"}
	n := NewWebhookNotifier(cfg)
	if err := n.Send(sampleEvent()); err == nil {
		t.Fatal("expected error for unreachable URL, got nil")
	}
}

// ── sign() ────────────────────────────────────────────────────────────────────

func TestSign_Deterministic(t *testing.T) {
	body := []byte(`{"event":"job.failed"}`)
	s1 := sign(body, "secret")
	s2 := sign(body, "secret")
	if s1 != s2 {
		t.Errorf("sign() not deterministic: %q vs %q", s1, s2)
	}
	if !strings.HasPrefix(s1, "sha256=") {
		t.Errorf("sign() should start with sha256=, got %q", s1)
	}
}

func TestSign_DifferentSecrets(t *testing.T) {
	body := []byte(`{"event":"job.failed"}`)
	s1 := sign(body, "secret-a")
	s2 := sign(body, "secret-b")
	if s1 == s2 {
		t.Error("different secrets should produce different signatures")
	}
}

// ── Email helpers ─────────────────────────────────────────────────────────────

func TestBuildEmailBody_ContainsFields(t *testing.T) {
	event := sampleEvent()
	body := buildEmailBody(event)

	checks := []string{
		"daily-backup",
		"agent-01",
		"exit status 1",
		"1m30s",
	}
	for _, want := range checks {
		if !strings.Contains(body, want) {
			t.Errorf("email body missing %q\nbody:\n%s", want, body)
		}
	}
}

func TestBuildMIMEMessage_Headers(t *testing.T) {
	msg := buildMIMEMessage(
		"from@example.com",
		[]string{"to@example.com"},
		"Test Subject",
		"body text",
	)
	for _, want := range []string{
		"From: from@example.com",
		"To: to@example.com",
		"Subject: Test Subject",
		"Content-Type: text/plain",
		"body text",
	} {
		if !strings.Contains(msg, want) {
			t.Errorf("MIME message missing %q\nmsg:\n%s", want, msg)
		}
	}
}

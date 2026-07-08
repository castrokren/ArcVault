package notifications

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"arcvault/coordinator/config"
)

const webhookTimeout = 10 * time.Second

// webhookPayload is the JSON body POSTed to the webhook URL.
type webhookPayload struct {
	Event      string `json:"event"`
	JobID      string `json:"job_id"`
	JobName    string `json:"job_name"`
	AgentID    string `json:"agent_id"`
	RunID      string `json:"run_id"`
	StartedAt  string `json:"started_at"`
	FinishedAt string `json:"finished_at"`
	DurationMs int64  `json:"duration_ms"`
	ErrorMsg   string `json:"error,omitempty"`
}

// WebhookNotifier delivers failure events via signed HTTP POST.
type WebhookNotifier struct {
	cfg    *config.WebhookConfig
	client *http.Client
}

func NewWebhookNotifier(cfg *config.WebhookConfig) *WebhookNotifier {
	return &WebhookNotifier{
		cfg:    cfg,
		client: &http.Client{Timeout: webhookTimeout},
	}
}

func (w *WebhookNotifier) Send(event JobFailureEvent) error {
	payload := webhookPayload{
		Event:      "job.failed",
		JobID:      event.JobID,
		JobName:    event.JobName,
		AgentID:    event.AgentID,
		RunID:      event.RunID,
		StartedAt:  event.StartedAt.UTC().Format(time.RFC3339),
		FinishedAt: event.FinishedAt.UTC().Format(time.RFC3339),
		DurationMs: event.FinishedAt.Sub(event.StartedAt).Milliseconds(),
		ErrorMsg:   event.ErrorMsg,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("webhook: marshal payload: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, w.cfg.URL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("webhook: build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "ArcVault-Notifier/1.0")

	if w.cfg.Secret != "" {
		req.Header.Set("X-ArcVault-Signature", sign(body, w.cfg.Secret))
	}

	resp, err := w.client.Do(req)
	if err != nil {
		return fmt.Errorf("webhook: POST %s: %w", w.cfg.URL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("webhook: POST %s returned %d", w.cfg.URL, resp.StatusCode)
	}
	return nil
}

// sign returns "sha256=<hex>" matching the GitHub webhook convention.
func sign(body []byte, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	return "sha256=" + hex.EncodeToString(mac.Sum(nil))
}

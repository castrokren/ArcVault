package notifications

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"arcvault/coordinator/config"
)

const teamsTimeout = 10 * time.Second

// teamsAdaptiveCard is the Teams Adaptive Card structure.
type teamsAdaptiveCard struct {
	Type    string        `json:"type"`
	Version string        `json:"version"`
	Body    []interface{} `json:"body"`
}

// teamsAttachment wraps the Adaptive Card.
type teamsAttachment struct {
	ContentType string            `json:"contentType"`
	Content     teamsAdaptiveCard `json:"content"`
}

// teamsPayload is the Teams webhook payload.
type teamsPayload struct {
	Type        string           `json:"type"`
	Attachments []teamsAttachment `json:"attachments"`
}

// TeamsNotifier delivers failure events via Teams incoming webhook.
type TeamsNotifier struct {
	cfg    *config.TeamsConfig
	client *http.Client
}

func NewTeamsNotifier(cfg *config.TeamsConfig) *TeamsNotifier {
	return &TeamsNotifier{
		cfg:    cfg,
		client: &http.Client{Timeout: teamsTimeout},
	}
}

func (t *TeamsNotifier) Send(event JobFailureEvent) error {
	duration := event.FinishedAt.Sub(event.StartedAt).Round(time.Second)

	body := []interface{}{
		map[string]interface{}{
			"type": "TextBlock",
			"text": "⚠️ ArcVault Job Alert",
			"weight": "bolder",
			"size": "large",
		},
		map[string]interface{}{
			"type": "TextBlock",
			"text": fmt.Sprintf("Job: %s", event.JobName),
		},
		map[string]interface{}{
			"type": "TextBlock",
			"text": fmt.Sprintf("Agent: %s", event.AgentID),
		},
		map[string]interface{}{
			"type": "TextBlock",
			"text": fmt.Sprintf("Run ID: %s", event.RunID),
		},
		map[string]interface{}{
			"type": "TextBlock",
			"text": fmt.Sprintf("Duration: %s", duration),
		},
		map[string]interface{}{
			"type": "TextBlock",
			"text": "Status: Failed",
			"color": "attention",
		},
	}

	if event.ErrorMsg != "" {
		body = append(body, map[string]interface{}{
			"type": "TextBlock",
			"text": fmt.Sprintf("Error: %s", event.ErrorMsg),
			"wrap": true,
		})
	}

	card := teamsAdaptiveCard{
		Type:    "AdaptiveCard",
		Version: "1.4",
		Body:    body,
	}

	attachment := teamsAttachment{
		ContentType: "application/vnd.microsoft.card.adaptive",
		Content:     card,
	}

	payload := teamsPayload{
		Type:        "message",
		Attachments: []teamsAttachment{attachment},
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("teams: marshal payload: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, t.cfg.WebhookURL, bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("teams: build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := t.client.Do(req)
	if err != nil {
		return fmt.Errorf("teams: POST %s: %w", t.cfg.WebhookURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("teams: POST returned %d", resp.StatusCode)
	}
	return nil
}

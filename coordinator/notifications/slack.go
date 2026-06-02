package notifications

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"arcvault/coordinator/config"
)

const slackTimeout = 10 * time.Second

// slackPayload is the Slack Blocks payload.
type slackPayload struct {
	Blocks []interface{} `json:"blocks"`
}

// SlackNotifier delivers failure events via Slack incoming webhook.
type SlackNotifier struct {
	cfg    *config.SlackConfig
	client *http.Client
}

func NewSlackNotifier(cfg *config.SlackConfig) *SlackNotifier {
	return &SlackNotifier{
		cfg:    cfg,
		client: &http.Client{Timeout: slackTimeout},
	}
}

func (s *SlackNotifier) Send(event JobFailureEvent) error {
	duration := event.FinishedAt.Sub(event.StartedAt).Round(time.Second)

	blocks := []interface{}{
		map[string]interface{}{
			"type": "header",
			"text": map[string]interface{}{
				"type": "plain_text",
				"text": "⚠️ ArcVault Job Alert",
			},
		},
		map[string]interface{}{
			"type": "section",
			"text": map[string]interface{}{
				"type": "mrkdwn",
				"text": fmt.Sprintf("*Job:* %s\n*Agent:* %s\n*Run ID:* %s\n*Duration:* %s\n*Status:* Failed",
					event.JobName, event.AgentID, event.RunID, duration),
			},
		},
	}

	if event.ErrorMsg != "" {
		blocks = append(blocks, map[string]interface{}{
			"type": "section",
			"text": map[string]interface{}{
				"type": "mrkdwn",
				"text": fmt.Sprintf("*Error:*\n```\n%s\n```", event.ErrorMsg),
			},
		})
	}

	payload := slackPayload{Blocks: blocks}
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("slack: marshal payload: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, s.cfg.WebhookURL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("slack: build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("slack: POST %s: %w", s.cfg.WebhookURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("slack: POST returned %d", resp.StatusCode)
	}
	return nil
}

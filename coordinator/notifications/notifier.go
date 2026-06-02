package notifications

import (
	"fmt"
	"log"
	"time"

	"arcvault/coordinator/config"
)

// JobFailureEvent carries all the context a notifier needs.
type JobFailureEvent struct {
	JobID      string
	JobName    string
	AgentID    string
	RunID      string
	StartedAt  time.Time
	FinishedAt time.Time
	ErrorMsg   string
}

// Notifier is the interface every delivery method implements.
type Notifier interface {
	Send(event JobFailureEvent) error
}

// Dispatcher holds the active notifiers built from config.
type Dispatcher struct {
	notifiers []Notifier
}

// NewDispatcher builds a Dispatcher from the notification config.
// Returns a no-op dispatcher (zero notifiers) when cfg is nil or OnFailure is false.
func NewDispatcher(cfg *config.NotificationConfig) *Dispatcher {
	d := &Dispatcher{}
	if cfg == nil || !cfg.OnFailure {
		return d
	}
	if cfg.Webhook != nil && cfg.Webhook.URL != "" {
		d.notifiers = append(d.notifiers, NewWebhookNotifier(cfg.Webhook))
	}
	if cfg.Email != nil && cfg.Email.SMTPHost != "" && len(cfg.Email.To) > 0 {
		d.notifiers = append(d.notifiers, NewEmailNotifier(cfg.Email))
	}
	if cfg.Slack != nil && cfg.Slack.WebhookURL != "" {
		d.notifiers = append(d.notifiers, NewSlackNotifier(cfg.Slack))
	}
	if cfg.Teams != nil && cfg.Teams.WebhookURL != "" {
		d.notifiers = append(d.notifiers, NewTeamsNotifier(cfg.Teams))
	}
	return d
}

// Dispatch fans out to every configured notifier.
// Errors are logged but never block the caller.
func (d *Dispatcher) Dispatch(event JobFailureEvent) {
	if len(d.notifiers) == 0 {
		return
	}
	for _, n := range d.notifiers {
		if err := n.Send(event); err != nil {
			log.Printf("[notifications] delivery failed for job %s run %s: %v",
				event.JobID, event.RunID, err)
		}
	}
}

// summary returns a plain-text summary of the failure for use in message bodies.
func summary(e JobFailureEvent) string {
	duration := e.FinishedAt.Sub(e.StartedAt).Round(time.Second)
	msg := fmt.Sprintf(
		"Job:      %s (ID %s)\nAgent:    %s\nRun ID:   %s\nStarted:  %s\nEnded:    %s\nDuration: %s",
		e.JobName, e.JobID,
		e.AgentID,
		e.RunID,
		e.StartedAt.UTC().Format(time.RFC3339),
		e.FinishedAt.UTC().Format(time.RFC3339),
		duration,
	)
	if e.ErrorMsg != "" {
		msg += "\nError:    " + e.ErrorMsg
	}
	return msg
}

package notifications

import (
	"fmt"
	"net/smtp"
	"strings"
	"time"

	"arcvault/coordinator/config"
)

// EmailNotifier delivers failure events via SMTP.
type EmailNotifier struct {
	cfg *config.EmailConfig
}

func NewEmailNotifier(cfg *config.EmailConfig) *EmailNotifier {
	return &EmailNotifier{cfg: cfg}
}

func (e *EmailNotifier) Send(event JobFailureEvent) error {
	addr := fmt.Sprintf("%s:%d", e.cfg.SMTPHost, e.cfg.SMTPPort)

	var auth smtp.Auth
	if e.cfg.Username != "" {
		auth = smtp.PlainAuth("", e.cfg.Username, e.cfg.Password, e.cfg.SMTPHost)
	}

	subject := fmt.Sprintf("[ArcVault] Job failed: %s (agent %s)", event.JobName, event.AgentID)
	body := buildEmailBody(event)
	msg := buildMIMEMessage(e.cfg.From, e.cfg.To, subject, body)

	if err := smtp.SendMail(addr, auth, e.cfg.From, e.cfg.To, []byte(msg)); err != nil {
		return fmt.Errorf("email: send via %s: %w", addr, err)
	}
	return nil
}

func buildEmailBody(e JobFailureEvent) string {
	duration := e.FinishedAt.Sub(e.StartedAt).Round(time.Second)
	var sb strings.Builder
	sb.WriteString("ArcVault has detected a job failure.\r\n\r\n")
	sb.WriteString(fmt.Sprintf("Job:      %s (ID %s)\r\n", e.JobName, e.JobID))
	sb.WriteString(fmt.Sprintf("Agent:    %s\r\n", e.AgentID))
	sb.WriteString(fmt.Sprintf("Run ID:   %s\r\n", e.RunID))
	sb.WriteString(fmt.Sprintf("Started:  %s\r\n", e.StartedAt.UTC().Format(time.RFC3339)))
	sb.WriteString(fmt.Sprintf("Ended:    %s\r\n", e.FinishedAt.UTC().Format(time.RFC3339)))
	sb.WriteString(fmt.Sprintf("Duration: %s\r\n", duration))
	if e.ErrorMsg != "" {
		sb.WriteString(fmt.Sprintf("Error:    %s\r\n", e.ErrorMsg))
	}
	sb.WriteString("\r\n--\r\nArcVault Coordinator\r\n")
	return sb.String()
}

func buildMIMEMessage(from string, to []string, subject, body string) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("From: %s\r\n", from))
	sb.WriteString(fmt.Sprintf("To: %s\r\n", strings.Join(to, ", ")))
	sb.WriteString(fmt.Sprintf("Subject: %s\r\n", subject))
	sb.WriteString("MIME-Version: 1.0\r\n")
	sb.WriteString("Content-Type: text/plain; charset=utf-8\r\n")
	sb.WriteString("\r\n")
	sb.WriteString(body)
	return sb.String()
}

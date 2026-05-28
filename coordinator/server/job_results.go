package server

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"arcvault/coordinator/notifications"
)

// JobRun represents a single execution result for a job.
type JobRun struct {
	ID         string `json:"id"`
	JobID      string `json:"job_id"`
	ExitCode   int    `json:"exit_code"`
	Output     string `json:"output"`
	StartedAt  string `json:"started_at"`
	FinishedAt string `json:"finished_at"`
}

func newRunID() string {
	b := make([]byte, 8)
	rand.Read(b)
	return "run-" + hex.EncodeToString(b)
}

// handlePostJobResults handles POST /api/jobs/{id}/results
func (s *Server) handlePostJobResults(w http.ResponseWriter, r *http.Request) {
	jobID := r.PathValue("id")

	// fetch job name and agent_id for notification context
	var jobName, agentID string
	err := s.db.Conn().QueryRow(`SELECT name, agent_id FROM jobs WHERE id = ?`, jobID).
		Scan(&jobName, &agentID)
	if err == sql.ErrNoRows {
		http.Error(w, "job not found", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, "failed to query job", http.StatusInternalServerError)
		return
	}

	var input struct {
		ExitCode  int    `json:"exit_code"`
		Output    string `json:"output"`
		StartedAt string `json:"started_at,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}

	finishedAt := time.Now().UTC()

	// Parse started_at from input, fallback to finished_at for backward compatibility
	startedAt := finishedAt
	if input.StartedAt != "" {
		if parsed, err := time.Parse(time.RFC3339, input.StartedAt); err == nil {
			startedAt = parsed
		}
	}

	run := JobRun{
		ID:         newRunID(),
		JobID:      jobID,
		ExitCode:   input.ExitCode,
		Output:     input.Output,
		StartedAt:  startedAt.Format(time.RFC3339),
		FinishedAt: finishedAt.Format(time.RFC3339),
	}

	_, err = s.db.Conn().Exec(
		`INSERT INTO job_runs (id, job_id, exit_code, output, started_at, finished_at)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		run.ID, run.JobID, run.ExitCode, run.Output, run.StartedAt, run.FinishedAt,
	)
	if err != nil {
		http.Error(w, "failed to store result", http.StatusInternalServerError)
		return
	}

	// dispatch failure notification
	if input.ExitCode != 0 {
		s.Notifier.Dispatch(notifications.JobFailureEvent{
			JobID:      jobID,
			JobName:    jobName,
			AgentID:    agentID,
			RunID:      run.ID,
			StartedAt:  startedAt,
			FinishedAt: finishedAt,
			ErrorMsg:   run.Output,
		})
	}

	// check duration_exceeded rules and fire alerts if threshold is exceeded
	rules, err := s.db.GetAlertRulesForJob(jobID)
	if err == nil {
		duration := finishedAt.Sub(startedAt).Seconds()
		for _, rule := range rules {
			if rule.RuleType == "duration_exceeded" && rule.Enabled && duration > float64(rule.Threshold) {
				s.Notifier.Dispatch(notifications.JobFailureEvent{
					JobID:      jobID,
					JobName:    jobName,
					AgentID:    agentID,
					RunID:      run.ID,
					StartedAt:  startedAt,
					FinishedAt: finishedAt,
					ErrorMsg:   fmt.Sprintf("Job duration exceeded threshold: %ds > %ds", int64(duration), int64(rule.Threshold)),
				})
			}
		}
	}

	// broadcast to WebSocket clients
	s.hub.Broadcast(Event{
		Type: "job.result",
		Payload: map[string]interface{}{
			"job_id":    run.JobID,
			"exit_code": run.ExitCode,
		},
	})

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(run)
}

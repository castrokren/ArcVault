package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"arcvault/coordinator/notifications"
)

// JobRun represents a single execution result for a job.
type JobRun struct {
	ID            string `json:"id"`
	JobID         string `json:"job_id"`
	JobName       string `json:"job_name"`
	SourcePath    string `json:"source_path"`
	DestPath      string `json:"dest_path"`
	AgentID       string `json:"agent_id"`
	AgentHostname string `json:"agent_hostname"`
	ExitCode      int    `json:"exit_code"`
	Output        string `json:"output"`
	Status        string `json:"status"`
	StartedAt     string `json:"started_at"`
	FinishedAt    string `json:"finished_at"`
}

// handlePostJobResults handles POST /api/jobs/{id}/results
func (s *Server) handlePostJobResults(w http.ResponseWriter, r *http.Request) {
	jobID := r.PathValue("id")

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

	// Use service layer to store job results
	jobInfo, err := s.jobService.PostJobResults(jobID, input.ExitCode, input.Output, startedAt.Format(time.RFC3339), finishedAt.Format(time.RFC3339))
	if err != nil {
		if err.Error() == "job not found" {
			http.Error(w, "job not found", http.StatusNotFound)
		} else {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}

	// Build response
	run := JobRun{
		ID:         jobID, // Note: actual run ID would need to be returned from service for proper tracking
		JobID:      jobID,
		ExitCode:   input.ExitCode,
		Output:     input.Output,
		StartedAt:  startedAt.Format(time.RFC3339),
		FinishedAt: finishedAt.Format(time.RFC3339),
	}

	// dispatch failure notification
	if input.ExitCode != 0 {
		s.Notifier.Dispatch(notifications.JobFailureEvent{
			JobID:      jobID,
			JobName:    jobInfo.JobName,
			AgentID:    jobInfo.AgentID,
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
					JobName:    jobInfo.JobName,
					AgentID:    jobInfo.AgentID,
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

package server

import (
	"database/sql"
	"fmt"
	"log"
	"sync"
	"time"

	"arcvault/coordinator/db"
	"arcvault/coordinator/notifications"

	"github.com/robfig/cron/v3"
)

// templateEntryMu guards templateEntryMap and templateCron, which are
// accessed from both the scheduler goroutine and API handler goroutines.
var (
	templateEntryMu  sync.Mutex
	templateEntryMap = map[string]cron.EntryID{}
	templateCron     *cron.Cron
)

// triggerScheduledJobs resets completed/failed scheduled jobs back to pending
// so the agent picks them up again on the next poll. Jobs that are currently
// running or pending are left untouched. Safe to call directly in tests.
func (s *Server) triggerScheduledJobs() {
	rows, err := s.db.Conn().Query(
		`SELECT id FROM jobs
		 WHERE schedule IS NOT NULL
		 AND schedule != ''
		 AND status IN ('completed', 'failed')`,
	)
	if err != nil {
		log.Printf("Scheduler: query failed: %v", err)
		return
	}
	defer rows.Close()

	var jobIDs []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err == nil {
			jobIDs = append(jobIDs, id)
		}
	}

	for _, id := range jobIDs {
		_, err := s.db.Conn().Exec(
			`UPDATE jobs SET status = 'pending' WHERE id = ?`, id,
		)
		if err != nil {
			log.Printf("Scheduler: failed to reset job %s: %v", id, err)
			continue
		}
		log.Printf("Scheduler: reset job %s to pending", id)
		s.hub.Broadcast(Event{
			Type:    "job.updated",
			Payload: map[string]string{"id": id, "status": "pending"},
		})
	}
}

// checkMissedSchedules checks for scheduled jobs that haven't run within their threshold.
// Fires missed_schedule alerts for jobs with matching rules.
func (s *Server) checkMissedSchedules() {
	// Get all jobs with a schedule
	rows, err := s.db.Conn().Query(`
		SELECT id, name, agent_id, schedule FROM jobs
		WHERE schedule IS NOT NULL AND schedule != ''
	`)
	if err != nil {
		log.Printf("Scheduler: checkMissedSchedules query failed: %v", err)
		return
	}
	defer rows.Close()

	now := time.Now().UTC()

	for rows.Next() {
		var jobID, jobName, agentID, schedule string
		if err := rows.Scan(&jobID, &jobName, &agentID, &schedule); err != nil {
			continue
		}

		// Get alert rules for this job with rule_type = missed_schedule
		rules, err := s.db.GetAlertRulesForJob(jobID)
		if err != nil {
			continue
		}

		for _, rule := range rules {
			if rule.RuleType != "missed_schedule" || !rule.Enabled {
				continue
			}

			// Get the most recent job run for this job
			var lastRun *time.Time
			err := s.db.Conn().QueryRow(`
				SELECT MAX(finished_at) FROM job_runs WHERE job_id = ?
			`, jobID).Scan(&lastRun)
			if err != nil && err != sql.ErrNoRows {
				continue
			}

			// If no runs found or last run is older than threshold, fire alert
			if lastRun == nil || now.Sub(*lastRun).Seconds() > float64(rule.Threshold) {
				// To avoid duplicate alerts, check if we've already fired one recently
				var recentAlertCount int64
				s.db.Conn().QueryRow(`
					SELECT COUNT(*) FROM alert_history
					WHERE job_id = ? AND rule_type = 'missed_schedule'
					AND fired_at > datetime('now', '-' || ? || ' seconds')
				`, jobID, rule.Threshold).Scan(&recentAlertCount)

				// Only fire if we haven't already fired one within the threshold window
				if recentAlertCount == 0 {
					s.Notifier.Dispatch(notifications.JobFailureEvent{
						JobID:      jobID,
						JobName:    jobName,
						AgentID:    agentID,
						RunID:      "",
						StartedAt:  now,
						FinishedAt: now,
						ErrorMsg:   fmt.Sprintf("Scheduled job missed: last run was %.0f seconds ago (threshold: %d seconds)", now.Sub(*lastRun).Seconds(), rule.Threshold),
					})
				}
			}
		}
	}
}

// StartScheduler starts a cron-based scheduler that triggers scheduled jobs
// at their defined intervals. Each job's schedule field is a standard
// 5-field cron expression (e.g. "0 2 * * *" for 2am daily).
// Also runs triggerScheduledJobs on a simple fallback ticker for jobs
// whose cron expression has already elapsed.
func (s *Server) StartScheduler() {
	c := cron.New()
	templateCron = c

	// load all scheduled jobs and register them with robfig/cron
	rows, err := s.db.Conn().Query(
		`SELECT id, schedule FROM jobs WHERE schedule IS NOT NULL AND schedule != ''`,
	)
	if err != nil {
		log.Printf("Scheduler: failed to load jobs: %v", err)
		return
	}
	defer rows.Close()

	jobCount := 0
	for rows.Next() {
		var id, schedule string
		if err := rows.Scan(&id, &schedule); err != nil {
			continue
		}
		jobID := id // capture for closure
		_, err := c.AddFunc(schedule, func() {
			log.Printf("Scheduler: cron tick for job %s", jobID)
			s.db.Conn().Exec(
				`UPDATE jobs SET status = 'pending' WHERE id = ? AND status NOT IN ('pending', 'running')`,
				jobID,
			)
			s.hub.Broadcast(Event{
				Type:    "job.updated",
				Payload: map[string]string{"id": jobID, "status": "pending"},
			})
		})
		if err != nil {
			log.Printf("Scheduler: invalid cron expression %q for job %s: %v", schedule, jobID, err)
			continue
		}
		jobCount++
	}

	// load all enabled templates
	tCount := s.loadTemplateSchedules(c)

	// Add daily task to prune federation_events (Phase 16)
	// Runs at 2 AM UTC daily
	_, err = c.AddFunc("0 2 * * *", func() {
		rowsDeleted, errPrune := s.db.PruneFederationEvents(7)
		if errPrune != nil {
			log.Printf("Scheduler: federation_events prune failed: %v", errPrune)
		} else {
			log.Printf("Scheduler: pruned %d federation_events older than 7 days", rowsDeleted)
		}
	})
	if err != nil {
		log.Printf("Scheduler: failed to add federation_events prune task: %v", err)
	}

	// Add daily task to prune alert_history (Phase 17)
	// Runs at 3 AM UTC daily
	retentionDays := s.cfg.AlertHistoryRetentionDays
	if retentionDays == 0 {
		retentionDays = 30 // default to 30 days
	}
	_, err = c.AddFunc("0 3 * * *", func() {
		errPrune := s.db.PruneAlertHistory(retentionDays)
		if errPrune != nil {
			log.Printf("Scheduler: alert_history prune failed: %v", errPrune)
		} else {
			log.Printf("Scheduler: pruned alert_history older than %d days", retentionDays)
		}
	})
	if err != nil {
		log.Printf("Scheduler: failed to add alert_history prune task: %v", err)
	}

	c.Start()
	log.Printf("Scheduler: started with %d scheduled job(s) and %d template(s)", jobCount, tCount)

	// fallback ticker — re-evaluates every minute for any jobs
	// that were created after startup and checks for missed schedules
	go func() {
		ticker := time.NewTicker(60 * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			s.triggerScheduledJobs()
			s.checkMissedSchedules()
		}
	}()
}

// loadTemplateSchedules registers cron entries for all enabled templates.
// Returns the number of templates successfully registered.
func (s *Server) loadTemplateSchedules(c *cron.Cron) int {
	templates, err := s.db.ListTemplates()
	if err != nil {
		log.Printf("Scheduler: failed to load templates: %v", err)
		return 0
	}
	count := 0
	for _, t := range templates {
		if !t.Enabled {
			continue
		}
		if err := s.addTemplateEntry(c, t); err != nil {
			log.Printf("Scheduler: failed to register template %s: %v", t.ID, err)
			continue
		}
		count++
	}
	return count
}

// addTemplateEntry registers a single template with the given cron instance.
func (s *Server) addTemplateEntry(c *cron.Cron, t db.Template) error {
	entryID, err := c.AddFunc(t.Schedule, func() {
		s.fireTemplate(t)
	})
	if err != nil {
		return fmt.Errorf("invalid cron expression %q: %w", t.Schedule, err)
	}
	templateEntryMu.Lock()
	templateEntryMap[t.ID] = entryID
	templateEntryMu.Unlock()
	log.Printf("Scheduler: registered template %s (%s)", t.ID, t.Schedule)
	return nil
}

// AddTemplateSchedule adds or replaces the cron entry for a template.
// Called by the API after create or update.
func (s *Server) AddTemplateSchedule(t db.Template) error {
	if templateCron == nil {
		return fmt.Errorf("scheduler not started")
	}
	s.RemoveTemplateSchedule(t.ID)
	if !t.Enabled {
		return nil
	}
	return s.addTemplateEntry(templateCron, t)
}

// RemoveTemplateSchedule removes the cron entry for a template by ID.
// Called by the API after delete or disable.
func (s *Server) RemoveTemplateSchedule(id string) {
	if templateCron == nil {
		return
	}
	templateEntryMu.Lock()
	defer templateEntryMu.Unlock()
	if entryID, ok := templateEntryMap[id]; ok {
		templateCron.Remove(entryID)
		delete(templateEntryMap, id)
		log.Printf("Scheduler: removed template %s", id)
	}
}

// NextTemplateRun returns the next scheduled run time for a template,
// or nil if the template is disabled or the scheduler is not running.
func (s *Server) NextTemplateRun(id string) *time.Time {
	if templateCron == nil {
		return nil
	}
	templateEntryMu.Lock()
	entryID, ok := templateEntryMap[id]
	templateEntryMu.Unlock()
	if !ok {
		return nil
	}
	entry := templateCron.Entry(entryID)
	if entry.ID == 0 {
		return nil
	}
	t := entry.Next
	return &t
}

// fireTemplate inserts a transient job row into the jobs table so the agent's
// existing HTTP poll loop picks it up. The job carries the template's command
// field; the agent executor runs it directly when command is non-empty.
func (s *Server) fireTemplate(t db.Template) {
	runID := fmt.Sprintf("tpl-%s-%d", t.ID, time.Now().UnixNano())
	now := time.Now().UTC().Format(time.RFC3339)

	log.Printf("Scheduler: firing template %s (agent %s)", t.ID, t.AgentID)

	_, err := s.db.Conn().Exec(
		`INSERT INTO jobs (id, agent_id, name, source_path, dest_path, command, status, created_at)
		 VALUES (?, ?, ?, '', '', ?, 'pending', ?)`,
		runID, t.AgentID, t.Name, t.Command, now,
	)
	if err != nil {
		log.Printf("Scheduler: failed to insert job for template %s: %v", t.ID, err)
		return
	}

	s.hub.Broadcast(Event{
		Type:    "job.created",
		Payload: map[string]string{"id": runID, "agent_id": t.AgentID, "status": "pending"},
	})
}

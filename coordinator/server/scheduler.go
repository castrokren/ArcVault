package server

import (
	"log"
	"time"

	"github.com/robfig/cron/v3"
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

// StartScheduler starts a cron-based scheduler that triggers scheduled jobs
// at their defined intervals. Each job's schedule field is a standard
// 5-field cron expression (e.g. "0 2 * * *" for 2am daily).
// Also runs triggerScheduledJobs on a simple fallback ticker for jobs
// whose cron expression has already elapsed.
func (s *Server) StartScheduler() {
	// load all scheduled jobs and register them with robfig/cron
	c := cron.New()

	rows, err := s.db.Conn().Query(
		`SELECT id, schedule FROM jobs WHERE schedule IS NOT NULL AND schedule != ''`,
	)
	if err != nil {
		log.Printf("Scheduler: failed to load jobs: %v", err)
		return
	}
	defer rows.Close()

	count := 0
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
		count++
	}

	c.Start()
	log.Printf("Scheduler: started with %d scheduled job(s)", count)

	// fallback ticker — re-evaluates every minute for any jobs
	// that were created after startup
	go func() {
		ticker := time.NewTicker(60 * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			s.triggerScheduledJobs()
		}
	}()
}

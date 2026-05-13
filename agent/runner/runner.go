package runner

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"
)

// Job represents a pending job returned by the coordinator.
type Job struct {
	ID         string `json:"id"`
	AgentID    string `json:"agent_id"`
	Name       string `json:"name"`
	SourcePath string `json:"source_path"`
	DestPath   string `json:"dest_path"`
	Status     string `json:"status"`
}

// Executor is a function that runs a job and returns exit code + output.
// Swappable for tests or for real robocopy/rsync in production.
type Executor func(job Job) (exitCode int, output string)

// Config holds everything the runner needs to talk to the coordinator.
type Config struct {
	AgentID        string
	CoordinatorURL string
	AuthToken      string
	PollInterval   time.Duration
}

// Runner polls the coordinator for pending jobs and executes them.
type Runner struct {
	cfg      Config
	executor Executor
	stop     chan struct{}
}

// New creates a Runner with the given config and executor.
func New(cfg Config, executor Executor) *Runner {
	if cfg.PollInterval == 0 {
		cfg.PollInterval = 30 * time.Second
	}
	return &Runner{
		cfg:      cfg,
		executor: executor,
		stop:     make(chan struct{}),
	}
}

// Start begins the polling loop. Blocking — run in a goroutine.
func (r *Runner) Start() {
	log.Printf("Job runner started (poll every %s)", r.cfg.PollInterval)
	ticker := time.NewTicker(r.cfg.PollInterval)
	defer ticker.Stop()

	// poll immediately on start, then on each tick
	r.poll()
	for {
		select {
		case <-ticker.C:
			r.poll()
		case <-r.stop:
			log.Println("Job runner stopped")
			return
		}
	}
}

// Stop signals the runner to exit its polling loop.
func (r *Runner) Stop() {
	close(r.stop)
}

// poll fetches pending jobs and processes each one.
func (r *Runner) poll() {
	jobs, err := r.fetchPendingJobs()
	if err != nil {
		log.Printf("Runner: failed to fetch jobs: %v", err)
		return
	}
	for _, job := range jobs {
		r.process(job)
	}
}

// fetchPendingJobs calls GET /api/jobs?agent_id=...&status=pending
func (r *Runner) fetchPendingJobs() ([]Job, error) {
	url := fmt.Sprintf("%s/api/jobs?agent_id=%s&status=pending", r.cfg.CoordinatorURL, r.cfg.AgentID)
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+r.cfg.AuthToken)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("poll request failed: %w", err)
	}
	defer resp.Body.Close()

	var jobs []Job
	if err := json.NewDecoder(resp.Body).Decode(&jobs); err != nil {
		return nil, fmt.Errorf("failed to decode jobs: %w", err)
	}
	return jobs, nil
}

// process claims a job, executes it, and posts the result.
func (r *Runner) process(job Job) {
	// 1. claim the job
	if err := r.updateStatus(job.ID, "running"); err != nil {
		log.Printf("Runner: failed to claim job %s: %v", job.ID, err)
		return
	}

	// 2. execute
	exitCode, output := r.executor(job)

	// 3. post result
	if err := r.postResult(job.ID, exitCode, output); err != nil {
		log.Printf("Runner: failed to post result for job %s: %v", job.ID, err)
	}

	// 4. mark final status
	finalStatus := "completed"
	if exitCode != 0 {
		finalStatus = "failed"
	}
	if err := r.updateStatus(job.ID, finalStatus); err != nil {
		log.Printf("Runner: failed to set final status for job %s: %v", job.ID, err)
	}
}

// updateStatus calls PATCH /api/jobs/{id}/status
func (r *Runner) updateStatus(jobID, status string) error {
	body, _ := json.Marshal(map[string]string{"status": status})
	url := fmt.Sprintf("%s/api/jobs/%s/status", r.cfg.CoordinatorURL, jobID)
	req, err := http.NewRequest(http.MethodPatch, url, bytes.NewBuffer(body))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+r.cfg.AuthToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("status update request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status %d", resp.StatusCode)
	}
	return nil
}

// postResult calls POST /api/jobs/{id}/results
func (r *Runner) postResult(jobID string, exitCode int, output string) error {
	body, _ := json.Marshal(map[string]interface{}{
		"exit_code": exitCode,
		"output":    output,
	})
	url := fmt.Sprintf("%s/api/jobs/%s/results", r.cfg.CoordinatorURL, jobID)
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(body))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+r.cfg.AuthToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("post result request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("unexpected status %d", resp.StatusCode)
	}
	return nil
}

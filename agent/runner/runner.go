package runner

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

// Job represents a pending job returned by the coordinator.
type Job struct {
	ID         string      `json:"id"`
	AgentID    string      `json:"agent_id"`
	Name       string      `json:"name"`
	SourcePath string      `json:"source_path"`
	DestPath   string      `json:"dest_path"`
	Command    string      `json:"command"`
	Status     string      `json:"status"`
	SyncFlags  *SyncFlags  `json:"sync_flags,omitempty"`
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
	Client         *http.Client
}

// Runner polls the coordinator for pending jobs and executes them.
type Runner struct {
	cfg       Config
	executor  Executor
	stop      chan struct{}
	stopOnce  sync.Once
}

// New creates a Runner with the given config and executor.
func New(cfg Config, executor Executor) (*Runner, error) {
	if !strings.HasPrefix(cfg.CoordinatorURL, "https://") && !strings.HasPrefix(cfg.CoordinatorURL, "http://localhost") && !strings.HasPrefix(cfg.CoordinatorURL, "http://127.0.0.1") {
		return nil, fmt.Errorf("CoordinatorURL must use HTTPS, got: %s", cfg.CoordinatorURL)
	}
	if cfg.PollInterval == 0 {
		cfg.PollInterval = 30 * time.Second
	}
	if cfg.Client == nil {
		transport := &http.Transport{
			MaxIdleConns:    10,
			IdleConnTimeout: 90 * time.Second,
			DisableKeepAlives: false,
		}
		cfg.Client = &http.Client{
			Transport: transport,
			Timeout:   30 * time.Second,
		}
	}
	return &Runner{
		cfg:      cfg,
		executor: executor,
		stop:     make(chan struct{}),
	}, nil
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
	r.stopOnce.Do(func() { close(r.stop) })
}

// poll fetches pending jobs and processes each one.
func (r *Runner) poll() {
	jobs, err := r.fetchPendingJobs()
	if err != nil {
		log.Printf("Runner: failed to fetch jobs: %v", err)
		return
	}
	for _, job := range jobs {
		if job.ID == "" || job.SourcePath == "" || job.DestPath == "" {
			log.Printf("Runner: skipping invalid job data: %+v", job)
			continue
		}
		r.process(job)
	}
}

// fetchPendingJobs calls GET /api/jobs?agent_id=...&status=pending
func (r *Runner) fetchPendingJobs() ([]Job, error) {
	u, err := url.Parse(r.cfg.CoordinatorURL)
	if err != nil {
		return nil, fmt.Errorf("parse coordinator URL: %w", err)
	}
	u.Path = "/api/jobs"
	u.RawQuery = url.Values{
		"agent_id": {r.cfg.AgentID},
		"status":   {"pending"},
	}.Encode()

	req, err := http.NewRequest(http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+r.cfg.AuthToken)

	resp, err := r.cfg.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("poll request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		if resp.StatusCode >= 500 {
			log.Printf("Runner: coordinator server error %d: %s", resp.StatusCode, string(body))
		} else if resp.StatusCode >= 400 {
			log.Printf("Runner: coordinator client error %d: %s", resp.StatusCode, string(body))
		}
		return nil, fmt.Errorf("unexpected status %d: %s", resp.StatusCode, string(body))
	}

	// Coordinator returns a paginated envelope: {"data": [...], "total": N, ...}
	var envelope struct {
		Data []Job `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&envelope); err != nil {
		return nil, fmt.Errorf("failed to decode jobs: %w", err)
	}
	return envelope.Data, nil
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
	body, err := json.Marshal(map[string]string{"status": status})
	if err != nil {
		return fmt.Errorf("marshal status: %w", err)
	}

	u, err := url.Parse(r.cfg.CoordinatorURL)
	if err != nil {
		return fmt.Errorf("parse coordinator URL: %w", err)
	}
	u.Path = "/api/jobs/" + url.PathEscape(jobID) + "/status"

	req, err := http.NewRequest(http.MethodPatch, u.String(), bytes.NewBuffer(body))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+r.cfg.AuthToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := r.cfg.Client.Do(req)
	if err != nil {
		return fmt.Errorf("status update request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		if resp.StatusCode >= 500 {
			log.Printf("Runner: coordinator server error %d: %s", resp.StatusCode, string(respBody))
		} else if resp.StatusCode >= 400 {
			log.Printf("Runner: coordinator client error %d: %s", resp.StatusCode, string(respBody))
		}
		return fmt.Errorf("unexpected status %d: %s", resp.StatusCode, string(respBody))
	}
	return nil
}

// postResult calls POST /api/jobs/{id}/results
func (r *Runner) postResult(jobID string, exitCode int, output string) error {
	body, err := json.Marshal(map[string]interface{}{
		"exit_code": exitCode,
		"output":    output,
	})
	if err != nil {
		return fmt.Errorf("marshal result: %w", err)
	}

	u, err := url.Parse(r.cfg.CoordinatorURL)
	if err != nil {
		return fmt.Errorf("parse coordinator URL: %w", err)
	}
	u.Path = "/api/jobs/" + url.PathEscape(jobID) + "/results"

	req, err := http.NewRequest(http.MethodPost, u.String(), bytes.NewBuffer(body))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+r.cfg.AuthToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := r.cfg.Client.Do(req)
	if err != nil {
		return fmt.Errorf("post result request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		respBody, _ := io.ReadAll(resp.Body)
		if resp.StatusCode >= 500 {
			log.Printf("Runner: coordinator server error %d: %s", resp.StatusCode, string(respBody))
		} else if resp.StatusCode >= 400 {
			log.Printf("Runner: coordinator client error %d: %s", resp.StatusCode, string(respBody))
		}
		return fmt.Errorf("unexpected status %d: %s", resp.StatusCode, string(respBody))
	}
	return nil
}

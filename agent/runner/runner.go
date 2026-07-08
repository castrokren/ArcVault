package runner

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"arcvault/agent/config"
	"arcvault/agent/honcho"
)

type JobCredentials struct {
	Type     string                 `json:"type"`
	Data     map[string]interface{} `json:"data"`
}

type Job struct {
	ID          string           `json:"id"`
	AgentID     string           `json:"agent_id"`
	Name        string           `json:"name"`
	SourcePath  string           `json:"source_path"`
	DestPath    string           `json:"dest_path"`
	Command     string           `json:"command"`
	Status      string           `json:"status"`
	SyncFlags   *SyncFlags       `json:"sync_flags,omitempty"`
	Credentials *JobCredentials  `json:"credentials,omitempty"`
}

type Executor func(ctx context.Context, job Job, report ProgressFunc) (exitCode int, output string)

type Config struct {
	AgentID        string
	CoordinatorURL string
	AuthToken      string
	CACertFile     string
	PollInterval   time.Duration
	Client         *http.Client
}

type Runner struct {
	cfg       Config
	executor  Executor
	stop       chan struct{}
	stopOnce   sync.Once
	cancelFuncs sync.Map
	pollNow     chan struct{}
	hc          *honcho.MetricsCollector
}

func New(cfg Config, executor Executor) (*Runner, error) {
	if !strings.HasPrefix(cfg.CoordinatorURL, "https://") && !strings.HasPrefix(cfg.CoordinatorURL, "http://localhost") && !strings.HasPrefix(cfg.CoordinatorURL, "http://127.0.0.1") {
		return nil, fmt.Errorf("CoordinatorURL must use HTTPS, got: %s", cfg.CoordinatorURL)
	}
	if cfg.PollInterval == 0 {
		cfg.PollInterval = 30 * time.Second
	}
	if cfg.Client == nil {
		tlsConfig, err := config.BuildTLSConfig(cfg.CACertFile)
		if err != nil {
			return nil, fmt.Errorf("failed to build TLS config: %w", err)
		}
		transport := &http.Transport{
			MaxIdleConns:       10,
			IdleConnTimeout:    90 * time.Second,
			DisableKeepAlives:  false,
			TLSClientConfig:    tlsConfig,
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
		pollNow:  make(chan struct{}, 1),
	}, nil
}

func (r *Runner) SetHonchoCollector(hc *honcho.MetricsCollector) {
	r.hc = hc
}

func (r *Runner) Start() {
	log.Printf("Job runner started (poll every %s)", r.cfg.PollInterval)
	ticker := time.NewTicker(r.cfg.PollInterval)
	defer ticker.Stop()

	r.poll()
	for {
		select {
		case <-ticker.C:
			r.poll()
		case <-r.pollNow:
			r.poll()
		case <-r.stop:
			log.Println("Job runner stopped")
			return
		}
	}
}

func (r *Runner) Stop() {
	r.stopOnce.Do(func() { close(r.stop) })
}

func (r *Runner) PollNow() {
	select {
	case r.pollNow <- struct{}{}:
	default:
	}
}

func (r *Runner) CancelJob(jobID string) bool {
	cancel, ok := r.cancelFuncs.LoadAndDelete(jobID)
	if !ok {
		return false
	}
	cancel.(context.CancelFunc)()
	return true
}

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

	var envelope struct {
		Data []Job `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&envelope); err != nil {
		return nil, fmt.Errorf("failed to decode jobs: %w", err)
	}
	return envelope.Data, nil
}

func (r *Runner) process(job Job) {
	startTime := time.Now()

	if err := r.updateStatus(job.ID, "running"); err != nil {
		log.Printf("Runner: failed to claim job %s: %v", job.ID, err)
		return
	}

	cleanup, err := applyCredentials(job)
	defer cleanup()
	if err != nil {
		log.Printf("Runner: failed to apply credentials for job %s: %v", job.ID, err)
		if err := r.postResult(job.ID, 1, fmt.Sprintf("credential error: %v", err)); err != nil {
			log.Printf("Runner: failed to post error result for job %s: %v", job.ID, err)
		}
		if err := r.updateStatus(job.ID, "failed"); err != nil {
			log.Printf("Runner: failed to set failed status for job %s: %v", job.ID, err)
		}
		r.recordExecution(job, "failed", 1, int(time.Since(startTime).Seconds()), 0, err.Error())
		return
	}

	log.Printf("Runner: executing job %s (src=%q dst=%q)", job.ID, job.SourcePath, job.DestPath)
	ctx, cancel := context.WithCancel(context.Background())
	r.cancelFuncs.Store(job.ID, cancel)
	exitCode := 1
	output := ""
	func() {
		defer func() {
			r.cancelFuncs.Delete(job.ID)
			cancel()
			if rec := recover(); rec != nil {
				log.Printf("Runner: panic executing job %s: %v", job.ID, rec)
				output = fmt.Sprintf("executor panic: %v", rec)
			}
		}()
		exitCode, output = r.executor(ctx, job, Noop)
	}()
	log.Printf("Runner: job %s finished — exit code %d, output length %d bytes", job.ID, exitCode, len(output))
	if len(output) > 0 {
		log.Printf("Runner: job %s output (first 512 bytes):\n%s", job.ID, truncate(output, 512))
	}

	if err := r.postResult(job.ID, exitCode, output); err != nil {
		log.Printf("Runner: failed to post result for job %s: %v", job.ID, err)
	}

	finalStatus := "completed"
	if exitCode != 0 {
		finalStatus = "failed"
	}
	if err := r.updateStatus(job.ID, finalStatus); err != nil {
		log.Printf("Runner: failed to set final status for job %s: %v", job.ID, err)
	}

	duration := int(time.Since(startTime).Seconds())
	errMsg := ""
	if exitCode != 0 {
		errMsg = truncate(output, 256)
	}
	r.recordExecution(job, finalStatus, exitCode, duration, 0, errMsg)
}

func (r *Runner) recordExecution(job Job, status string, exitCode, durationSec, retryCount int, errMsg string) {
	if r.hc == nil {
		return
	}
	exec := honcho.JobExecution{
		JobID:       job.ID,
		JobName:     job.Name,
		Source:      job.SourcePath,
		Destination: job.DestPath,
		Status:      status,
		ExitCode:    exitCode,
		Duration:    durationSec,
		RetryCount:  retryCount,
		Error:       errMsg,
		Timestamp:   time.Now(),
	}
	r.hc.RecordExecution(exec)
}

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

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}

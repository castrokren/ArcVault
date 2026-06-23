package honcho

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Client manages communication with Honcho memory server.
type Client struct {
	baseURL     string
	workspaceID string
	httpClient  *http.Client
}

// JobExecution represents a single backup job execution stored in Honcho.
type JobExecution struct {
	JobID       string            `json:"job_id"`
	JobName     string            `json:"job_name"`
	Source      string            `json:"source"`
	Destination string            `json:"destination"`
	Status      string            `json:"status"` // "success", "failed", "partial"
	ExitCode    int               `json:"exit_code"`
	Duration    int               `json:"duration_seconds"`
	RetryCount  int               `json:"retry_count"`
	BytesTotal  int64             `json:"bytes_total"`
	BytesCopied int64             `json:"bytes_copied"`
	Throughput  float64           `json:"throughput_mbps"`
	Error       string            `json:"error,omitempty"`
	Timestamp   time.Time         `json:"timestamp"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}

// ExecutionPattern represents aggregated patterns from historical executions.
type ExecutionPattern struct {
	SourcePath         string  `json:"source_path"`
	FailureRate        float64 `json:"failure_rate"`
	TimeoutCount       int     `json:"timeout_count"`
	AvgDuration        int     `json:"avg_duration_seconds"`
	MostCommonError    string  `json:"most_common_error"`
	SuccessfulRetries  int     `json:"successful_retries"`
	LastFailureTime    string  `json:"last_failure_time,omitempty"`
	RecommendedRetries int     `json:"recommended_retries"`
}

// NewClient creates a Honcho client.
// baseURL: "http://localhost:8000"
// workspaceID: obtained from Honcho (typically the agent's ID or name)
func NewClient(baseURL, workspaceID string) *Client {
	return &Client{
		baseURL:     baseURL,
		workspaceID: workspaceID,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// HealthCheck verifies Honcho is reachable.
func (c *Client) HealthCheck() error {
	resp, err := c.httpClient.Get(c.baseURL + "/health")
	if err != nil {
		return fmt.Errorf("honcho health check failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("honcho health check returned %d", resp.StatusCode)
	}

	var result map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("honcho health check response invalid: %w", err)
	}

	if result["status"] != "ok" {
		return fmt.Errorf("honcho status not ok: %v", result)
	}

	return nil
}

// StoreExecution saves a single job execution to Honcho.
func (c *Client) StoreExecution(exec JobExecution) error {
	if exec.Timestamp.IsZero() {
		exec.Timestamp = time.Now()
	}

	payload, err := json.Marshal(map[string]interface{}{
		"type":    "job_execution",
		"content": fmt.Sprintf("Job: %s | Status: %s | Duration: %ds | Retries: %d | Error: %s", exec.JobName, exec.Status, exec.Duration, exec.RetryCount, exec.Error),
		"metadata": map[string]interface{}{
			"job_id":       exec.JobID,
			"job_name":     exec.JobName,
			"status":       exec.Status,
			"exit_code":    exec.ExitCode,
			"duration":     exec.Duration,
			"retry_count":  exec.RetryCount,
			"bytes_total":  exec.BytesTotal,
			"bytes_copied": exec.BytesCopied,
			"throughput":   exec.Throughput,
			"error":        exec.Error,
		},
	})
	if err != nil {
		return fmt.Errorf("failed to marshal execution: %w", err)
	}

	url := fmt.Sprintf("%s/v3/workspaces/%s/messages", c.baseURL, c.workspaceID)
	resp, err := c.httpClient.Post(url, "application/json", bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("failed to store execution in honcho: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("honcho store failed (%d): %s", resp.StatusCode, string(body))
	}

	return nil
}

// BatchStoreExecutions stores multiple job executions at once.
func (c *Client) BatchStoreExecutions(executions []JobExecution) error {
	for _, exec := range executions {
		if err := c.StoreExecution(exec); err != nil {
			// Log but continue — don't fail entire batch on one error
			fmt.Printf("warning: failed to store execution for %s: %v\n", exec.JobID, err)
		}
	}
	return nil
}

// QueryPatterns retrieves historical patterns for a source path.
// Returns aggregated failure rates, error types, and recommendations.
func (c *Client) QueryPatterns(sourcePath string) (*ExecutionPattern, error) {
	// This would query Honcho's message history for a source path.
	// For now, we stub this — implementation depends on Honcho's search API.
	url := fmt.Sprintf("%s/v3/workspaces/%s/messages?source=%s", c.baseURL, c.workspaceID, sourcePath)
	resp, err := c.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to query patterns from honcho: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("honcho query failed (%d)", resp.StatusCode)
	}

	// Parse response and aggregate patterns.
	// This is a simplified stub — real implementation would aggregate from message history.
	return &ExecutionPattern{
		SourcePath:         sourcePath,
		FailureRate:        0.1,
		RecommendedRetries: 3,
	}, nil
}

// GetRecommendedRetryStrategy queries Honcho and returns suggested retry count for a source.
func (c *Client) GetRecommendedRetryStrategy(sourcePath string) int {
	pattern, err := c.QueryPatterns(sourcePath)
	if err != nil {
		return 3 // default
	}
	if pattern.FailureRate > 0.5 {
		return 5
	}
	if pattern.FailureRate > 0.2 {
		return 4
	}
	return pattern.RecommendedRetries
}

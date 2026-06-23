package honcho

import (
	"fmt"
	"log"
	"time"
)

// ErrorContext contains information about a job failure.
type ErrorContext struct {
	SourcePath  string
	DestPath    string
	Error       string
	RetryCount  int
	MaxRetries  int
	ExitCode    int
	Duration    int
}

// SuggestRecoveryStrategy uses Honcho to recommend how to handle a job failure.
// Returns: (shouldRetry, suggestedDelay, reasoning)
func (c *Client) SuggestRecoveryStrategy(ctx ErrorContext) (shouldRetry bool, suggestedDelaySeconds int, reasoning string) {
	// Query Honcho for historical patterns on this source
	pattern, err := c.QueryPatterns(ctx.SourcePath)
	if err != nil {
		log.Printf("honcho query failed, using defaults: %v\n", err)
		// Fallback to simple logic if Honcho unavailable
		return ctx.RetryCount < ctx.MaxRetries, 5, "Honcho unavailable, using default strategy"
	}

	// Based on pattern, decide retry strategy
	if pattern.TimeoutCount > 5 {
		// This source frequently times out — increase wait time
		suggestedDelaySeconds = 30
		reasoning = fmt.Sprintf("Source has %d timeout failures; increasing wait time", pattern.TimeoutCount)
	} else if pattern.FailureRate > 0.7 {
		// High failure rate — give up after 2 retries
		if ctx.RetryCount >= 2 {
			return false, 0, fmt.Sprintf("Source failure rate %.0f%% exceeds threshold, giving up", pattern.FailureRate*100)
		}
		suggestedDelaySeconds = 10
		reasoning = fmt.Sprintf("High failure rate (%.0f%%), but still within retry limit", pattern.FailureRate*100)
	} else if pattern.FailureRate > 0.3 {
		// Moderate failure rate — use normal strategy
		if ctx.RetryCount >= pattern.RecommendedRetries {
			return false, 0, fmt.Sprintf("Failure rate %.0f%% suggests %d retries (limit reached)", pattern.FailureRate*100, pattern.RecommendedRetries)
		}
		suggestedDelaySeconds = 5
		reasoning = fmt.Sprintf("Moderate failure rate (%.0f%%), retry within limit (%d/%d)", pattern.FailureRate*100, ctx.RetryCount, pattern.RecommendedRetries)
	} else {
		// Low failure rate — treat as anomaly, retry once
		if ctx.RetryCount >= 1 {
			return false, 0, fmt.Sprintf("Low failure rate (%.0f%%), anomalous failure, giving up", pattern.FailureRate*100)
		}
		suggestedDelaySeconds = 2
		reasoning = fmt.Sprintf("Low historical failure rate (%.0f%%), trying once more", pattern.FailureRate*100)
	}

	shouldRetry = ctx.RetryCount < ctx.MaxRetries
	return shouldRetry, suggestedDelaySeconds, reasoning
}

// LogError stores an error event in Honcho for future analysis.
func (c *Client) LogError(jobID, jobName, sourcePath, errorMsg string, retryCount, exitCode int) error {
	exec := JobExecution{
		JobID:      jobID,
		JobName:    jobName,
		Source:     sourcePath,
		Status:     "failed",
		ExitCode:   exitCode,
		RetryCount: retryCount,
		Error:      errorMsg,
		Timestamp:  time.Now(),
	}
	return c.StoreExecution(exec)
}

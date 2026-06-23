# Agent ↔ Honcho Integration

This module enables the ArcVault Agent to:
1. **On job error:** Query Honcho for historical patterns and adjust retry strategy
2. **End of day:** Batch store all job execution metrics (pass/fail, duration, errors, performance)
3. **For Claude code review:** Store rich context that Claude can analyze to suggest Agent optimizations

## Architecture

```
Agent Job Execution
  ├─ Success → MetricsCollector.RecordExecution()
  └─ Error → HonchoClient.SuggestRecoveryStrategy()
               └─ Query patterns for source
               └─ Adjust retry logic based on history
               └─ Log error to Honcho

Daily Batch (e.g., 10pm)
  └─ MetricsCollector.ProcessBatch()
     └─ Store all execution metrics to Honcho

Claude Code Review
  └─ Query Honcho for patterns
  └─ "This source has 70% failure rate; recommend exponential backoff"
  └─ Suggest retry/timeout optimizations
```

## Setup

### 1. Initialize Honcho Client in Agent Startup

In `agent/main.go` or your agent initialization code:

```go
import (
	"arcvault/agent/honcho"
)

func runAgent() {
	// ... existing agent setup ...

	// Initialize Honcho (verify connectivity)
	honchoClient := honcho.NewClient("http://localhost:8000", "agent-001")
	if err := honchoClient.HealthCheck(); err != nil {
		log.Printf("warning: honcho not available: %v (metrics disabled)\n", err)
	}

	// Initialize metrics collector (batch at 22:00/10pm)
	collector := honcho.NewMetricsCollector(honchoClient, 22)

	// Start batch processor (runs every minute)
	go func() {
		ticker := time.NewTicker(1 * time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			if err := collector.ProcessBatch(); err != nil {
				log.Printf("batch process failed: %v\n", err)
			}
		}
	}()

	// ... rest of agent startup, pass collector/client to job runner ...
}
```

### 2. Record Successful Job Completion

In your job executor (e.g., `agent/runner/runner.go`):

```go
func (r *Runner) process(ctx context.Context, job Job, honchoCollector *honcho.MetricsCollector) error {
	// ... execute job ...

	exitCode, output := r.executor(job, progressReporter)

	// Record metrics
	exec := honcho.JobExecution{
		JobID:       job.ID,
		JobName:     job.Name,
		Source:      job.SourcePath,
		Destination: job.DestPath,
		Status:      "success",
		ExitCode:    exitCode,
		Duration:    int(time.Since(startTime).Seconds()),
		RetryCount:  retries,
		BytesCopied: progressReporter.BytesCopied,
		Throughput:  calculateThroughput(progressReporter.BytesCopied, startTime),
	}

	if honchoCollector != nil {
		honchoCollector.RecordExecution(exec)
	}

	return nil
}
```

### 3. Query Honcho on Job Error for Retry Strategy

In your error handling:

```go
if err := r.executor(job, progressReporter); err != nil {
	// Ask Honcho: should we retry this?
	errCtx := honcho.ErrorContext{
		SourcePath:  job.SourcePath,
		DestPath:    job.DestPath,
		Error:       err.Error(),
		RetryCount:  retries,
		MaxRetries:  5,
		ExitCode:    exitCode,
		Duration:    int(time.Since(startTime).Seconds()),
	}

	shouldRetry, delaySeconds, reasoning := honchoClient.SuggestRecoveryStrategy(errCtx)
	log.Printf("retry suggestion: %v (delay: %ds) — reason: %s\n", shouldRetry, delaySeconds, reasoning)

	if shouldRetry {
		log.Printf("retrying in %d seconds...\n", delaySeconds)
		time.Sleep(time.Duration(delaySeconds) * time.Second)
		retries++
		goto retry // or loop back
	} else {
		log.Printf("giving up on job after %d retries\n", retries)
	}

	// Log failure to Honcho
	if honchoClient != nil {
		honchoClient.LogError(job.ID, job.Name, job.SourcePath, err.Error(), retries, exitCode)
	}
}
```

### 4. Shutdown: Flush Pending Metrics

When Agent stops (graceful shutdown), flush any buffered metrics:

```go
func shutdown(collector *honcho.MetricsCollector) {
	log.Println("flushing pending metrics to honcho...")
	if err := collector.Flush(); err != nil {
		log.Printf("flush failed: %v\n", err)
	}
}
```

## Querying from Claude

Once metrics are stored in Honcho, Claude can analyze patterns for code review:

```python
# Example Python code for Claude-assisted analysis

import requests
import json

honcho_base = "http://localhost:8000"
workspace_id = "agent-001"

# Query execution history
response = requests.get(f"{honcho_base}/v3/workspaces/{workspace_id}/messages")
messages = response.json()

# Extract patterns
failures = [m for m in messages if 'failed' in m.get('content', '').lower()]
timeout_errors = [m for m in messages if 'timeout' in m.get('content', '').lower()]

# Prepare context for Claude
context = f"""
Agent execution history from Honcho:
- Total executions: {len(messages)}
- Failures: {len(failures)} ({100*len(failures)/len(messages):.0f}%)
- Timeout errors: {len(timeout_errors)}
- Most common error: {most_common_error}

Metadata patterns:
- Sources with high failure: {high_failure_sources}
- Average retry count: {avg_retries}
"""

# Send to Claude API with context
response = client.messages.create(
    model="claude-3-5-sonnet-20241022",
    max_tokens=1024,
    messages=[
        {
            "role": "user",
            "content": f"{context}\n\nBased on this execution history, suggest improvements to the Agent's retry logic and timeout handling."
        }
    ]
)

print(response.content[0].text)
```

## Data Stored in Honcho

Each execution stores:
```json
{
  "job_id": "backup-001",
  "job_name": "Weekly Full Backup",
  "source": "/data/documents",
  "destination": "\\\\nas\\backups",
  "status": "success",
  "exit_code": 0,
  "duration_seconds": 3600,
  "retry_count": 0,
  "bytes_total": 1000000000,
  "bytes_copied": 950000000,
  "throughput_mbps": 250.5,
  "error": null,
  "timestamp": "2026-06-12T19:30:00Z",
  "metadata": { ... }
}
```

## Testing Honcho Connection

```bash
# Verify Honcho health
curl http://localhost:8000/health

# List workspace messages (after storing some executions)
curl http://localhost:8000/v3/workspaces/agent-001/messages

# Clear buffer on demand
# (honchoCollector.Flush() in code)
```

## Error Handling

- If Honcho is unavailable on startup, Agent continues with warnings (metrics disabled)
- If batch store fails, metrics are held in buffer for retry next batch
- Call `Flush()` on graceful shutdown to ensure no loss
- On persistent Honcho failure, Agent degrades gracefully (no pattern-based retry optimization)

## Next Steps

1. Wire this into the Agent's main runner
2. Test with a few job executions
3. Run daily batch at midnight or 10pm
4. Query Honcho from Claude for code review insights

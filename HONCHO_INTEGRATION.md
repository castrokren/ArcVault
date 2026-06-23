# Honcho ↔ ArcVault ↔ Claude Integration

**Goal:** Use Honcho to store Agent execution history, enabling Claude to analyze patterns and suggest smarter Agent behavior.

---

## What's Deployed

✅ **Honcho Memory Server**
- Running at `http://localhost:8000`
- Configured with OpenRouter (gpt-4-turbo) for embeddings & reasoning
- PostgreSQL 15 + pgvector for vector search
- Redis for caching

✅ **Agent ↔ Honcho Integration (Go modules)**
- `agent/honcho/honcho.go` — Honcho API client
- `agent/honcho/error_recovery.go` — Smart retry strategy based on history
- `agent/honcho/batch_storage.go` — Daily batching of execution metrics

---

## How It Works

### 1. Agent Records Execution Metrics

**On job success:**
```
Agent runs backup job
  → Collects: duration, bytes, throughput, retry count
  → Buffers in MetricsCollector
```

**On job error:**
```
Agent encounters failure
  → Queries Honcho: "What happened last time with this source?"
  → Honcho returns: failure rate, timeout pattern, recommended retries
  → Agent adjusts strategy (retry delay, max retries)
  → Logs error to Honcho
```

**Daily (10pm):**
```
MetricsCollector.ProcessBatch()
  → Stores all buffered executions to Honcho
  → Clears buffer
```

### 2. Claude Analyzes Patterns

**During code review:**
```
Claude queries Honcho:
  "What are the failure patterns for sources?"

Honcho returns:
  - Source /data/documents: 5% failure rate, avg 3600s duration
  - Source /data/media: 35% failure rate, 12 timeouts, avg 2h 30m duration
  - Source /data/archives: 0% failure rate

Claude suggests:
  "Source /data/media has high failure — try exponential backoff,
   increase timeout to 3 hours, and reduce concurrent transfers."
```

### 3. Agent Evolves

**Smarter decisions:**
- High-timeout sources → longer wait between retries
- High-failure sources → give up after 2 retries (don't waste time)
- Low-failure sources → treat anomalies as temporary glitches, retry once

**Learned retry strategies:**
- Default: 3 retries, 5s delay
- High-failure pattern: 2 retries, 10s delay
- Timeout pattern: 5 retries, 30s delay (exponential)

---

## Architecture

```
┌─────────────────────────────────────────────────────┐
│                   ArcVault Agent                     │
│  ┌───────────────────────────────────────────────┐  │
│  │ Job Execution                                 │  │
│  │  - Runs backup job (robocopy/rsync)           │  │
│  │  - On error: Query Honcho for retry strategy  │  │
│  │  - On success: Record metrics                 │  │
│  └──────────┬──────────────────────────────────┬─┘  │
│             │                                  │     │
│  ┌──────────▼──────────┐        ┌──────────────▼──┐ │
│  │ MetricsCollector   │        │ Honcho Client   │ │
│  │ (buffers metrics)  │        │ (API calls)     │ │
│  └──────────┬──────────┘        └──────────┬──────┘ │
│             │ (batch at 22:00)             │        │
│             └──────────────────┬───────────┘        │
└────────────────────────────────┼───────────────────┘
                                 │
                    ┌────────────▼───────────┐
                    │  Honcho Memory Server  │
                    │  http://localhost:8000 │
                    │                        │
                    │ Stores:                │
                    │ - Execution history    │
                    │ - Failure patterns     │
                    │ - Retry strategies     │
                    │ - Performance metrics  │
                    └────────────┬───────────┘
                                 │
                    ┌────────────▼──────────────┐
                    │   Claude Code Review      │
                    │                           │
                    │ Queries Honcho:           │
                    │ "What failed? Why?"       │
                    │                           │
                    │ Suggests improvements:    │
                    │ - Timeout adjustments     │
                    │ - Retry strategies        │
                    │ - Concurrency tuning      │
                    └───────────────────────────┘
```

---

## Next Steps to Integrate

### Step 1: Wire Honcho into Agent Startup
Edit `agent/main.go`:
```go
// Initialize Honcho client
honchoClient := honcho.NewClient("http://localhost:8000", "agent-001")
honchoClient.HealthCheck()

// Initialize metrics collector (batch at 22:00)
collector := honcho.NewMetricsCollector(honchoClient, 22)

// Start batch processor
go func() {
	ticker := time.NewTicker(1 * time.Minute)
	for range ticker.C {
		collector.ProcessBatch()
	}
}()
```

### Step 2: Record Metrics on Job Completion
Edit `agent/runner/runner.go`:
```go
// After job runs, record execution
exec := honcho.JobExecution{
	JobID: job.ID,
	Status: "success",
	Duration: elapsed,
	// ... other fields
}
collector.RecordExecution(exec)
```

### Step 3: Use Honcho on Job Error
Edit error handler:
```go
if err != nil {
	shouldRetry, delay, reason := honchoClient.SuggestRecoveryStrategy(errCtx)
	log.Printf("Honcho suggests: retry=%v, delay=%ds (%s)\n", shouldRetry, delay, reason)
	time.Sleep(time.Duration(delay) * time.Second)
}
```

### Step 4: Query from Claude
Create a Python script to feed Honcho data to Claude:
```python
import requests
honcho_api = "http://localhost:8000"

# Get execution history
msgs = requests.get(f"{honcho_api}/v3/workspaces/agent-001/messages").json()

# Send to Claude with context
context = f"Execution history from Honcho: {len(msgs)} jobs, {failures} failures"
claude_response = client.messages.create(
    model="claude-3-5-sonnet-20241022",
    messages=[{"role": "user", "content": f"{context}\nSuggest Agent improvements."}]
)
print(claude_response.content[0].text)
```

---

## Data Flow Example

**Day 1 - Morning:**
```
Job: Backup /data/documents → Success (1h, 500MB)
Job: Backup /data/media → Timeout → Query Honcho → Retry after 30s → Success (2h30m)
```
→ Metrics buffered in Agent

**Day 1 - 10pm:**
```
MetricsCollector.ProcessBatch()
→ Sends 47 execution records to Honcho
→ Honcho stores with metadata
```

**Day 2 - Code Review:**
```
Claude queries Honcho → Sees pattern:
- /data/media: 12 timeouts this month, avg duration 2h30m
- Suggests: increase timeout to 3h, use exponential backoff

Agent updated → Smarter retry logic
```

---

## Files Created

```
agent/
├── honcho/
│   ├── honcho.go           # Core API client
│   ├── error_recovery.go   # Retry strategy + error logging
│   ├── batch_storage.go    # Metrics collection & batching
│   └── INTEGRATION.md      # Setup & usage guide
```

Plus: `HONCHO_INTEGRATION.md` (this file) — high-level overview

---

## Testing

```bash
# Verify Honcho is running
curl http://localhost:8000/health
# Expected: {"status":"ok"}

# After agent runs some jobs, check stored data
curl http://localhost:8000/v3/workspaces/agent-001/messages | jq '.'

# Query patterns for a source
curl "http://localhost:8000/v3/workspaces/agent-001/messages?source=%2Fdata%2Fdocuments"
```

---

## Summary

- ✅ Honcho deployed and running
- ✅ Agent modules written (honcho.go, error_recovery.go, batch_storage.go)
- ✅ Integration guide complete (INTEGRATION.md)
- ⏳ Next: Wire into Agent code, test, deploy
- ⏳ Then: Create Claude query script for code review insights

This gives your Agent **memory** — it learns from past failures and Claude can see those patterns to suggest smarter strategies.

# Phase 22: Integration Testing & Stress Tests — Design Spec

**Date:** May 29, 2026  
**Status:** Design approved, ready for implementation  
**Objective:** Prove ArcVault is production-ready through load, failure, and edge-case testing

---

## Overview

Phase 22 is a three-tier integrated testing phase that validates ArcVault under real-world conditions:

1. **Load Testing** — System behavior under 100+ concurrent jobs
2. **Failure Scenarios** — Recovery from agent disconnects, coordinator crashes, timeouts
3. **Edge Cases & Workflows** — Large paths, permissions, federation sync, multi-agent dispatch

Success = Production-ready system + documented limits + regression test suite.

---

## Architecture

### Three-Layer Testing Approach

#### **Layer 1: Load Test Harness** (`arcvault-test` CLI)

Standalone command-line tool for stress testing the coordinator with mock agents.

**Purpose:**
- Spawn N mock agents simultaneously
- Create M jobs with configurable duration and size
- Measure throughput, latency, resource usage
- Inject failures at runtime
- Generate JSON reports for trending

**File Structure:**
```
coordinator/cmd/arcvault-test/
├── main.go                 # CLI entry point, command routing
├── harness.go              # Harness orchestrator (spawn agents, create jobs)
├── mock_agent.go           # Simulates real agent (register, heartbeat, execute)
├── load_scenario.go        # Load profiles (10/50/100+ agents, job configs)
├── failure_injector.go     # Inject failures (disconnect, timeout, crash)
└── metrics.go              # Collect & report results (throughput, latency, etc.)
```

**CLI Interface:**
```bash
arcvault-test load \
  --agents=50 \
  --jobs-per-agent=20 \
  --job-duration=5m \
  --source-path=/tmp/test-data \
  --failure-rate=0.1 \
  --failure-type=disconnect \
  --output=report.json
```

**Outputs:**
- `report.json` — Throughput, latency (p50/p95/p99), success rate, error breakdown
- `timeline.log` — Per-agent/job execution timeline for debugging
- Metrics: peak memory, connection count, database performance

**Mock Agent Behavior:**
- Registers with coordinator (creates agent record in DB)
- Sends heartbeat every 5 seconds (simulates real agent)
- Creates jobs on demand from harness
- Executes jobs for specified duration
- Handles failures injected by harness (disconnect, timeout)
- Reports job completion/failure back to coordinator

---

#### **Layer 2: Failure Injection Framework**

Integrated into the harness to simulate real-world failure modes.

**Failure Modes:**

| Failure | Mechanism | Detection | Expected Recovery |
|---------|-----------|-----------|---|
| **Agent Disconnect** | Close WebSocket mid-job | Coordinator misses heartbeat | Job pauses, resumes when agent reconnects |
| **Coordinator Crash** | Kill coordinator process, restart | Harness restarts, agents reconnect | In-flight jobs recovered from `job_runs` table |
| **Network Timeout** | Inject 30s+ delay on coordinator requests | Request hangs then times out | Retry logic succeeds on next attempt |
| **Database Lock** | Concurrent writes cause lock contention | Slower performance, potential deadlock | Queries queue/retry, no data loss |
| **Resource Exhaustion** | Create 10,000+ connections or fill memory | System slowdown or OOM | Graceful degradation, no corruption |

**Implementation Details:**
- Failures are **randomized** based on `--failure-rate` (e.g., 0.1 = 10% of jobs experience failure)
- Failure **timing is configurable** — inject after N% of job completion (e.g., agent disconnects at 50%)
- Harness **tracks which jobs** experienced failures and validates recovery
- Metrics include: time-to-detect, recovery time, jobs lost vs. resumed

**Key Scenario: Agent Disconnect During Long Job** (your concern)
- Start 20 agents, each running a 30-minute job
- Inject agent disconnects randomly, every 5 minutes
- Measure: How quickly does job resume? Is progress preserved?
- Expected: Job pauses ≤1 second, resumes within 5 seconds of reconnect, no data loss

---

#### **Layer 3: Integration & Edge Case Tests**

Unit and integration tests covering end-to-end workflows and boundary conditions.

**A) End-to-End Workflows** (new test files)
- `coordinator/server/integration_test.go` — Multi-agent job dispatch, group failover, federation sync
- `coordinator/server/workflow_test.go` — Complete backup scenarios, scheduling, notifications

Tests:
- Single job → all agents execute independently
- Group dispatch → job creates one execution per agent, all tracked
- Federation sync → primary + 2 secondaries stay consistent under load
- Job cancellation → running agents halt gracefully
- Scheduled jobs → cron triggers create jobs at correct times

**B) Edge Cases** (expand existing unit tests)

*Large paths & file counts:*
- Source path with 100+ nested directories
- 100,000+ files in single job
- File sizes from 0 bytes to multi-GB
- Symlinks and junctions (cross-drive, circular refs)

*Permission & I/O errors:*
- Read-only source directory → job fails with clear error
- No-write destination → job fails gracefully, no partial state
- Concurrent writes to same destination → no data corruption
- Mid-job permission change → detect and handle

*Long-running jobs:*
- Job running 30+ minutes with agent disconnect at 50%
- Progress checkpoint preserved in `job_runs.progress`
- Job resumes from checkpoint, not from start

**C) Recovery Scenarios** (stress + integration)
- Long job + agent disconnect → job pauses, resumes on reconnect
- Job in-flight when coordinator restarts → recovered from DB
- Multiple agents disconnect → each reconnects independently
- Coordinator crash during federation sync → sync resumes correctly

**Test Format:**
- Existing `*_test.go` files (standard Go test style)
- Use test database (`test.db`)
- Mock WebSocket for disconnect scenarios
- Run via `go test ./... -v`

---

## Test Execution

### Local Development
```bash
# Unit + edge case tests
go test ./... -v

# Load test with mock agents
go run ./coordinator/cmd/arcvault-test load --agents=10 --jobs=50 --output=report.json

# Specific failure scenario
go run ./coordinator/cmd/arcvault-test load --agents=20 --job-duration=5m \
  --failure-type=disconnect --failure-rate=0.2 --output=disconnect_test.json
```

### CI/CD Pipeline (GitHub Actions)
- **On commit:** Run unit tests (`go test ./...`)
- **Nightly:** Run full load suite (10/50/100 agents)
- **Weekly:** Run all failure scenarios
- Compare metrics against baseline; alert if throughput drops 20%+
- Archive reports in `reports/` directory for trending

### Reports & Metrics

Each test run produces:
- **`results.json`** — Throughput, latency (p50/p95/p99), success rate, error breakdown
- **`timeline.log`** — Per-agent/job execution timeline
- **`summary.txt`** — Human-readable pass/fail summary, bottlenecks

Example report:
```json
{
  "test": "load_100_agents_20_jobs",
  "duration_seconds": 3600,
  "agents_spawned": 100,
  "jobs_created": 2000,
  "jobs_completed": 1998,
  "jobs_failed": 2,
  "throughput_jobs_per_second": 0.555,
  "latency_p50_ms": 120,
  "latency_p95_ms": 450,
  "latency_p99_ms": 890,
  "peak_memory_mb": 512,
  "agents_max_concurrent_connections": 105,
  "recovery_time_disconnect_seconds": 4.2,
  "failures_injected": 200,
  "failures_recovered": 198
}
```

---

## Success Criteria

✅ **All integration tests pass** (end-to-end workflows execute correctly)  
✅ **Load test completes 100+ concurrent jobs without crashes or data loss**  
✅ **Agent disconnect recovery:** Job resumes within 5 seconds of agent reconnect  
✅ **Coordinator crash recovery:** All in-flight jobs preserved in DB and recovered  
✅ **Edge case tests:** Large paths, permissions, long jobs all handle gracefully  
✅ **Regression suite:** Unit + integration tests can run in CI/CD nightly  
✅ **Documented limits:** Know exact throughput and resource requirements (e.g., "100 agents = 2GB RAM")

---

## Implementation Timeline

| Week | Focus |
|------|-------|
| 1-2 | Build load harness + mock agent framework |
| 2-3 | Failure injector + basic load scenarios (10/50 agents) |
| 3 | Integration tests + end-to-end workflows |
| 3-4 | Edge case tests + long-job scenarios |
| 4 | Full test suite execution, performance tuning, documentation |

---

## Data Flow

```
┌─────────────────────────────────────────────────────────┐
│ arcvault-test harness (CLI)                             │
│ ┌──────────────────────────────────────────────────┐   │
│ │ Load Scenario                                    │   │
│ │ (agents=50, jobs=20, duration=5m)                │   │
│ └───────────────┬──────────────────────────────────┘   │
│                 │                                       │
│ ┌───────────────▼──────────────────────────────────┐   │
│ │ Spawn Mock Agents → Register in coordinator     │   │
│ │ Each agent: heartbeat loop + job executor       │   │
│ └───────────────┬──────────────────────────────────┘   │
│                 │                                       │
│ ┌───────────────▼──────────────────────────────────┐   │
│ │ Create Jobs (batches)                           │   │
│ │ ├─ job_id, agent_id, duration, sync_flags       │   │
│ │ └─ stored in coordinator.jobs table             │   │
│ └───────────────┬──────────────────────────────────┘   │
│                 │                                       │
│ ┌───────────────▼──────────────────────────────────┐   │
│ │ Failure Injector                                 │   │
│ │ ├─ Randomly disconnect agents                   │   │
│ │ ├─ Simulate timeouts                            │   │
│ │ └─ Crash coordinator (restart)                  │   │
│ └───────────────┬──────────────────────────────────┘   │
│                 │                                       │
│ ┌───────────────▼──────────────────────────────────┐   │
│ │ Metrics Collector                               │   │
│ │ ├─ Throughput (jobs/sec)                        │   │
│ │ ├─ Latency (p50/p95/p99)                        │   │
│ │ ├─ Recovery time                                │   │
│ │ └─ Resource usage                               │   │
│ └───────────────┬──────────────────────────────────┘   │
│                 │                                       │
│ ┌───────────────▼──────────────────────────────────┐   │
│ │ Report Generation (JSON + summary)              │   │
│ └──────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────┘
         │
         ▼
   Coordinator (running)
   ├─ Accept agent registrations
   ├─ Store jobs & progress
   ├─ Handle WebSocket (heartbeat, disconnect)
   ├─ Recover from crashes
   └─ Persist state in DB
```

---

## Known Constraints & Assumptions

- **Testing environment:** Single machine or local network (not geographically distributed)
- **Agent simulation:** Mock agents are in-process; real network behavior may differ slightly
- **Database:** SQLite for testing (production may use MySQL/Postgres)
- **Failure injection:** External from coordinator (not internal instrumentation)
- **Long jobs:** Simulated with sleep loops, not actual backup I/O

---

## Open Questions & Future Work

- Should we support distributed failure injection (failures injected on remote agents)?
- Should harness produce performance graphs (throughput over time)?
- Should we benchmark against target specs (e.g., "100 agents must complete in X minutes")?
- Integration with monitoring tools (Grafana, Prometheus)?

---

## References

- `coordinator/server/jobs.go` — Job creation & execution
- `coordinator/server/progress.go` — Progress tracking
- `coordinator/server/federation.go` — Federation sync
- Existing test files: `jobs_test.go`, `federation_test.go`, etc.

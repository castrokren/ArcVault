# Phase 17 Implementation Plan — Enhanced Monitoring & Alerting
**Project:** ArcVault2.0
**Version target:** v1.0.0
**Branch:** `feature/phase-17-alerting`
**Precondition:** Phase 16 gaps closed, all tests passing ✅
**Last updated:** 2026-05-22

---

## Context

Phase 12 shipped a basic notification system:
- `Dispatcher` — fan-out to webhook + email, single attempt, log on failure
- `JobFailureEvent` — fires on `exit_code != 0` only
- `job_results.go` — trigger point; `StartedAt == FinishedAt` (no start time recorded)
- No alert rules, no alert history, no retry

Phase 17 extends this without rewriting it.

---

## What Phase 17 Adds

### 1. Job Start Time Fix
`StartedAt` in `JobFailureEvent` is currently set to `FinishedAt` (no start time). Fix by recording `started_at` when the result is posted.

### 2. Alert Rules
Configurable per-job conditions stored in DB:
- `on_failure` — existing behavior, now a rule
- `duration_exceeded` — fires if job runtime > threshold (seconds)
- `missed_schedule` — fires if scheduled job hasn't run in N seconds

### 3. Webhook Retry
Currently single-attempt. Add async retry: 3 attempts, exponential backoff (5s → 15s → 45s). Never blocks the job result handler.

### 4. Slack + Teams Notifiers
Slack blocks API and Teams Adaptive Card, both via incoming webhook URLs. No OAuth, no app installation.

### 5. Alert History
New `alert_history` table. Every fired alert persisted with delivery status. Dashboard view with re-send button (admin only).

---

## Design Decisions

| ID | Decision |
|----|----------|
| D-019 | Alert rules stored in DB — allows per-job rules without coordinator restart |
| D-020 | Retry is async (goroutine) — never blocks the job result handler |
| D-021 | Alert history retained 30 days (configurable), pruned by scheduler |
| D-022 | Slack/Teams use incoming webhook URLs — no OAuth, no app install |
| D-023 | Duration threshold alert fires at job completion, not mid-run |
| D-024 | Missed schedule detection runs on existing scheduler ticker (every 60s) |

---

## New DB Schema

```sql
CREATE TABLE IF NOT EXISTS alert_rules (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    job_id      TEXT,                    -- NULL = applies to all jobs
    rule_type   TEXT NOT NULL,           -- on_failure | duration_exceeded | missed_schedule
    threshold   INTEGER,                 -- seconds (duration_exceeded or missed_schedule)
    enabled     INTEGER NOT NULL DEFAULT 1,
    created_at  DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS alert_history (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    rule_id     INTEGER REFERENCES alert_rules(id),
    job_id      TEXT,
    run_id      TEXT,
    rule_type   TEXT NOT NULL,
    fired_at    DATETIME DEFAULT CURRENT_TIMESTAMP,
    channel     TEXT NOT NULL,           -- webhook | email | slack | teams
    status      TEXT NOT NULL,           -- delivered | failed | retrying
    attempts    INTEGER NOT NULL DEFAULT 1,
    last_error  TEXT
);
```

Migration also adds: `ALTER TABLE job_runs ADD COLUMN started_at DATETIME`

---

## New API Endpoints

```
GET    /api/alert-rules              — list all rules (viewer+)
POST   /api/alert-rules              — create rule (admin)
PUT    /api/alert-rules/{id}         — update rule (admin)
DELETE /api/alert-rules/{id}         — delete rule (admin)
GET    /api/alert-history            — recent alerts with status (viewer+)
POST   /api/alert-history/{id}/retry — manual re-send (admin)
```

---

## Architecture

```
New Go files:
  coordinator/notifications/slack.go
  coordinator/notifications/slack_test.go
  coordinator/notifications/teams.go
  coordinator/notifications/teams_test.go
  coordinator/notifications/retry.go
  coordinator/notifications/retry_test.go
  coordinator/server/alert_rules.go
  coordinator/server/alert_rules_test.go
  coordinator/server/alert_history.go
  coordinator/server/alert_history_test.go
  coordinator/db/alert_rules.go
  dashboard/src/views/Alerts.vue

Modified:
  coordinator/db/db.go               — new tables + started_at migration
  coordinator/config/config.go       — SlackConfig, TeamsConfig
  coordinator/notifications/notifier.go — Dispatcher writes history + calls retry
  coordinator/server/job_results.go  — record started_at; evaluate duration rules
  coordinator/server/scheduler.go    — missed schedule detector + alert history prune
  coordinator/server/server.go       — register new routes
  dashboard/src/api.js               — alert rule + history endpoints
  dashboard/src/router/index.js      — /alerts route
```

---

## Task List

Tasks are ordered by dependency. Complete each fully before starting the next.
Run `go test ./...` after every backend task.

---

### PRE-FLIGHT
- [ ] `git checkout main && git pull`
- [ ] `git checkout -b feature/phase-17-alerting`
- [ ] `go test ./...` — confirm all tests pass before writing a line

---

### TASK 1 — Fix job start time

**Files:** `coordinator/db/db.go`, `coordinator/server/job_results.go`

**Steps:**
1. `db.go` — add to `migrate()`:
   ```go
   // Idempotent: add started_at to job_runs for Phase 17 accurate duration tracking.
   d.conn.Exec(`ALTER TABLE job_runs ADD COLUMN started_at DATETIME`)
   ```
2. `job_results.go` — extend the input struct to accept `started_at`:
   ```go
   var input struct {
       ExitCode  int    `json:"exit_code"`
       Output    string `json:"output"`
       StartedAt string `json:"started_at,omitempty"`
   }
   ```
3. Record `started_at` in the INSERT:
   ```go
   startedAt := input.StartedAt
   if startedAt == "" {
       startedAt = finishedAt.Format(time.RFC3339) // backward compat
   }
   // INSERT INTO job_runs (id, job_id, exit_code, output, started_at, finished_at)
   ```
4. Parse `startedAt` into `time.Time` and use real value in `JobFailureEvent.StartedAt`
5. Update `JobRun` struct: add `StartedAt string \`json:"started_at"\``

**Verify:**
```powershell
go build ./coordinator/...
go test ./coordinator/server/ -run TestJobResults
```

---

### TASK 2 — DB: alert_rules + alert_history tables

**Files:** `coordinator/db/db.go`, `coordinator/db/alert_rules.go` (new)

**Steps:**
1. `db.go` — add both tables to `migrate()` (idempotent `CREATE TABLE IF NOT EXISTS`)
2. Create `coordinator/db/alert_rules.go`:

```go
package db

import "time"

type AlertRule struct {
    ID        int64     `json:"id"`
    JobID     string    `json:"job_id"`     // empty = applies to all jobs
    RuleType  string    `json:"rule_type"`  // on_failure | duration_exceeded | missed_schedule
    Threshold int       `json:"threshold"`  // seconds; 0 for on_failure
    Enabled   bool      `json:"enabled"`
    CreatedAt time.Time `json:"created_at"`
}

type AlertHistory struct {
    ID        int64     `json:"id"`
    RuleID    int64     `json:"rule_id"`
    JobID     string    `json:"job_id"`
    RunID     string    `json:"run_id"`
    RuleType  string    `json:"rule_type"`
    FiredAt   time.Time `json:"fired_at"`
    Channel   string    `json:"channel"`
    Status    string    `json:"status"`
    Attempts  int       `json:"attempts"`
    LastError string    `json:"last_error"`
}

func (d *DB) ListAlertRules() ([]AlertRule, error)
func (d *DB) GetAlertRulesForJob(jobID string) ([]AlertRule, error) // returns rules for jobID + global rules
func (d *DB) CreateAlertRule(r AlertRule) (int64, error)
func (d *DB) UpdateAlertRule(r AlertRule) error
func (d *DB) DeleteAlertRule(id int64) error
func (d *DB) AppendAlertHistory(h AlertHistory) (int64, error)
func (d *DB) UpdateAlertHistoryStatus(id int64, status, lastError string, attempts int) error
func (d *DB) ListAlertHistory(limit int) ([]AlertHistory, error)
func (d *DB) PruneAlertHistory(olderThanDays int) error
```

**Verify:**
```powershell
go build ./coordinator/...
go test ./coordinator/db/
```

---

### TASK 3 — Webhook retry

**Files:** `coordinator/notifications/retry.go` (new), `coordinator/notifications/retry_test.go` (new)

**Steps:**
1. Define `AlertHistoryWriter` interface (keeps notifications package decoupled from db package):
   ```go
   type AlertHistoryWriter interface {
       AppendAlertHistory(h AlertHistory) (int64, error)
       UpdateAlertHistoryStatus(id int64, status, lastError string, attempts int) error
   }
   ```
2. `RetryDispatch(ctx context.Context, n Notifier, event JobFailureEvent, channel string, histWriter AlertHistoryWriter)`:
   - Write initial history row with `status="retrying"`
   - Attempt 1 immediately
   - On failure: sleep 5s, attempt 2; sleep 15s, attempt 3; sleep 45s, give up
   - On success: update history row `status="delivered"`
   - On final failure: update history row `status="failed"`, set `last_error`
   - Runs in a goroutine — caller does `go RetryDispatch(...)`
3. Tests: mock notifier that fails N times, verify retry count and final status

**Verify:**
```powershell
go test ./coordinator/notifications/ -run TestRetry -v
```

---

### TASK 4 — Slack + Teams notifiers

**Files:** `coordinator/notifications/slack.go`, `coordinator/notifications/teams.go`, test files, `coordinator/config/config.go`

**Steps:**
1. `config.go` — add to `NotificationConfig`:
   ```go
   Slack *SlackConfig `json:"slack,omitempty"`
   Teams *TeamsConfig `json:"teams,omitempty"`
   ```
   ```go
   type SlackConfig struct {
       WebhookURL string `json:"webhook_url"`
   }
   type TeamsConfig struct {
       WebhookURL string `json:"webhook_url"`
   }
   ```

2. `slack.go` — post Slack blocks payload:
   ```json
   {
     "blocks": [
       {"type": "header", "text": {"type": "plain_text", "text": "⚠ ArcVault Job Alert"}},
       {"type": "section", "text": {"type": "mrkdwn", "text": "*Job:* <job_name>\n*Agent:* <agent_id>\n*Error:* <error_msg>"}}
     ]
   }
   ```

3. `teams.go` — post Teams Adaptive Card:
   ```json
   {
     "type": "message",
     "attachments": [{
       "contentType": "application/vnd.microsoft.card.adaptive",
       "content": { "type": "AdaptiveCard", "version": "1.4", "body": [...] }
     }]
   }
   ```

4. Wire both into `NewDispatcher()` — same pattern as existing webhook/email

5. Tests: mock HTTP server per notifier, verify payload shape + Content-Type header

**Verify:**
```powershell
go test ./coordinator/notifications/ -v
```

---

### TASK 5 — Alert rules CRUD

**Files:** `coordinator/server/alert_rules.go` (new), `coordinator/server/alert_rules_test.go` (new), `coordinator/server/server.go`

**Steps:**
1. Implement handlers:
   - `handleListAlertRules` — viewer+
   - `handleCreateAlertRule` — admin; validate `rule_type` ∈ {on_failure, duration_exceeded, missed_schedule}
   - `handleUpdateAlertRule` — admin
   - `handleDeleteAlertRule` — admin
2. Register in `server.go`:
   ```go
   s.router.HandleFunc("GET /api/alert-rules",        s.viewerRoute(s.handleListAlertRules))
   s.router.HandleFunc("POST /api/alert-rules",       s.adminRoute(s.handleCreateAlertRule))
   s.router.HandleFunc("PUT /api/alert-rules/{id}",   s.adminRoute(s.handleUpdateAlertRule))
   s.router.HandleFunc("DELETE /api/alert-rules/{id}",s.adminRoute(s.handleDeleteAlertRule))
   ```
3. Tests: list empty, create valid, create invalid rule_type (400), update, delete

**Verify:**
```powershell
go test ./coordinator/server/ -run TestAlertRules -v
```

---

### TASK 6 — Duration exceeded alert

**Files:** `coordinator/server/job_results.go`

**Steps:**
1. After storing job run, query `s.db.GetAlertRulesForJob(jobID)` for `rule_type = 'duration_exceeded'`
2. Parse `startedAt` and `finishedAt` → compute duration in seconds
3. If duration > rule.Threshold: fire alert via `go RetryDispatch(...)` per notifier
4. Use a new `AlertEvent` type (or extend `JobFailureEvent` with an `AlertType string` field):
   ```go
   type AlertType string
   const (
       AlertJobFailure         AlertType = "job.failed"
       AlertDurationExceeded   AlertType = "job.duration_exceeded"
       AlertMissedSchedule     AlertType = "job.missed_schedule"
   )
   ```
5. Update `JobFailureEvent` to include `AlertType AlertType`

**Verify:**
```powershell
go test ./coordinator/server/ -run TestDurationAlert -v
```

---

### TASK 7 — Missed schedule detector

**Files:** `coordinator/server/scheduler.go`

**Steps:**
1. Add to scheduler (separate from existing job ticker or new 60s ticker):
   ```go
   func (s *Server) checkMissedSchedules() {
       // Get all jobs with schedule set
       // For each: get missed_schedule alert rules
       // Check: SELECT MAX(finished_at) FROM job_runs WHERE job_id = ?
       // If now - last_run > rule.Threshold: fire alert
   }
   ```
2. Call `go RetryDispatch(...)` per notifier when threshold exceeded
3. Avoid repeat-firing: check alert_history — don't fire if already fired in the last threshold window

**Verify:**
```powershell
go test ./coordinator/server/ -run TestMissedSchedule -v
```

---

### TASK 8 — Alert history endpoint + manual retry

**Files:** `coordinator/server/alert_history.go` (new), `coordinator/server/alert_history_test.go` (new), `coordinator/server/server.go`

**Steps:**
1. `handleListAlertHistory` — viewer+; query last 100 rows, ordered by `fired_at DESC`
2. `handleRetryAlert` — admin; look up history row, re-fire the alert via notifier, update status
3. Register routes:
   ```go
   s.router.HandleFunc("GET /api/alert-history",             s.viewerRoute(s.handleListAlertHistory))
   s.router.HandleFunc("POST /api/alert-history/{id}/retry", s.adminRoute(s.handleRetryAlert))
   ```
4. Tests: list empty, list with rows, retry failed row (re-fires), retry delivered row (no-op or 409)

**Verify:**
```powershell
go test ./coordinator/server/ -run TestAlertHistory -v
```

---

### TASK 9 — Scheduler: prune alert history

**Files:** `coordinator/server/scheduler.go`, `coordinator/config/config.go`

**Steps:**
1. Add `AlertHistoryRetentionDays int` to `Config` (default 30)
2. Add daily scheduled task alongside existing `PruneFederationEvents`:
   ```go
   s.db.PruneAlertHistory(s.cfg.AlertHistoryRetentionDays)
   ```

**Verify:**
```powershell
go build ./coordinator/...
```

---

### TASK 10 — Frontend: Alerts.vue

**Files:** `dashboard/src/views/Alerts.vue` (new), `dashboard/src/api.js`, `dashboard/src/router/index.js`

**Design:**
- Two sections stacked vertically: **Alert Rules** above, **Alert History** below
- Product register, restrained color strategy (tinted neutrals + one accent)
- Rules table columns: Job | Rule Type | Threshold | Enabled | Delete
- History table columns: Time | Job | Rule Type | Channel | Status | Retry
- Status pills: OKLCH — green `delivered`, amber `retrying`, red `failed`
- Empty history state: "No alerts have fired." — subdued, not alarming
- Auto-refresh history every 30s (same pattern as FederationHealth.vue)
- Role gates: viewer sees both tables read-only; admin sees create rule form + delete + retry button

**Steps:**
1. Add to `api.js`:
   ```js
   export const getAlertRules = () => api('/api/alert-rules')
   export const createAlertRule = (rule) => api('/api/alert-rules', { method: 'POST', body: rule })
   export const deleteAlertRule = (id) => api(`/api/alert-rules/${id}`, { method: 'DELETE' })
   export const getAlertHistory = () => api('/api/alert-history')
   export const retryAlert = (id) => api(`/api/alert-history/${id}/retry`, { method: 'POST' })
   ```
2. Create `Alerts.vue`
3. Add route to `router/index.js`: `{ path: '/alerts', component: Alerts }`
4. Add "Alerts" nav link (alongside Federation, Jobs, etc.)

---

### TASK 11 — Tests + smoke test + cleanup

**Steps:**
1. `go test ./...` — all tests green
2. Manual smoke test:
   - Create a `duration_exceeded` alert rule for a job with threshold=1 (1 second)
   - Trigger the job, wait for completion
   - Verify alert appears in `/api/alert-history` with `status=delivered`
   - Kill the webhook endpoint, trigger again
   - Verify `status=failed` after retry exhaustion
3. Update `CONTEXT.md`: version → v1.0.0, Phase 17 complete
4. Update `MEMORY.md`: Phase 17 entry

---

### TASK 12 — Commit + tag

```powershell
git add -A
git commit -m "Phase 17 complete: alert rules, webhook retry, Slack/Teams, alert history (v1.0.0)"
git tag -a v1.0.0 -m "v1.0.0 — Phase 17: Enhanced monitoring and alerting"
git push origin feature/phase-17-alerting
git push origin v1.0.0
```

---

## Summary Table

| Task | Area | New Files | Modified Files | Effort |
|------|------|-----------|----------------|--------|
| 1 | Job start time fix | — | `db/db.go`, `server/job_results.go` | 30m |
| 2 | DB: alert tables | `db/alert_rules.go` | `db/db.go` | 1h |
| 3 | Webhook retry | `notifications/retry.go`, `_test.go` | `notifications/notifier.go` | 1.5h |
| 4 | Slack + Teams | `notifications/slack.go`, `teams.go`, tests | `config/config.go` | 2h |
| 5 | Alert rules CRUD | `server/alert_rules.go`, `_test.go` | `server/server.go` | 1.5h |
| 6 | Duration alert | — | `server/job_results.go` | 1h |
| 7 | Missed schedule | — | `server/scheduler.go` | 1.5h |
| 8 | Alert history + retry | `server/alert_history.go`, `_test.go` | `server/server.go` | 1h |
| 9 | Prune scheduler | — | `server/scheduler.go`, `config/config.go` | 30m |
| 10 | Alerts.vue | `views/Alerts.vue` | `api.js`, `router/index.js` | 2.5h |
| 11 | Tests + smoke + cleanup | — | `CONTEXT.md`, `MEMORY.md` | 1h |
| 12 | Commit + tag | — | — | 15m |

**Total estimate:** 2–3 weeks (solo, part-time)

---

## Rules for This Plan

- ❌ Never rewrite without explicit approval
- ✅ Pre-flight: branch from main, confirm all tests pass before starting
- ✅ Test after every backend task (`go test ./coordinator/...`)
- ✅ Proof before "done" — no claiming complete without test output
- ✅ Bugs traced to root cause before fixing
- ✅ D-019–D-024 decisions are locked — check before deviating
- ✅ PowerShell line continuation: backtick (`), not backslash
- ✅ Backward compat: existing `on_failure` webhook + email config unchanged
- ✅ `RetryDispatch` always runs in a goroutine — never blocks job result handler

# Phase 16 Implementation Plan — Federation Failover, Load Balancing & State Consistency
**Project:** ArcVault2.0
**Version target:** v0.9.0
**Branch:** `feature/phase-16-federation-ha`
**Precondition:** Phase 15 complete, v0.8.0 tagged ✅
**Last updated:** 2026-05-22

---

## What Phase 14 Already Has

Phase 14 shipped hub-and-spoke federation:
- `FederationHub` — root coordinator accepts spoke WebSocket connections
- `FederationClient` — spoke coordinator dials root on startup
- `federation` table — stores peer coordinator records + tokens
- 9 API endpoints — register, list, heartbeat, agent/job broadcast
- `agents.go` broadcasts `agent_registered` + heartbeat deltas to peers
- `Federation.vue` + `SiteSelector.vue` — basic UI

Phase 16 does NOT rewrite any of this. It extends it.

---

## What Phase 16 Adds

### 1. Failover Detection
- Root coordinator goes down → spokes detect disconnect, enter **standalone mode** automatically
- Spoke goes down → root marks it `status=offline`, stops broadcasting to it
- Recovery → spoke reconnects, triggers **state resync** to catch up on missed events
- Dashboard shows stale banner already (Phase 14) — Phase 16 drives it correctly

### 2. Agent Load Balancing
- Agents currently connect to exactly one coordinator (hard-coded in config)
- Phase 16: agents carry a **coordinator list** in config (`coordinators: [url1, url2]`)
- On connect failure → agent tries next coordinator in list (round-robin fallback)
- Coordinator tracks which coordinator the agent is "homed" to — no job duplication

### 3. State Consistency (Event Log)
- New DB table: `federation_events` — append-only log of agent/job state changes
- Each event has a `seq` (sequence number, per-coordinator monotonic counter)
- On reconnect: spoke sends its `last_seq` → root replays missed events since that seq
- Resolves the "stale banner" problem — UI knows exactly how far behind a spoke is

---

## Design Decisions

| ID | Decision |
|----|----------|
| D-013 | Standalone mode is safe: spoke keeps running jobs normally when disconnected from root |
| D-014 | State sync is root→spoke only (root is source of truth); no spoke→spoke sync |
| D-015 | Agent failover is client-side (agent config, not coordinator-managed) |
| D-016 | `federation_events` log is pruned after 7 days (configurable) to prevent unbounded growth |
| D-017 | Sequence numbers are per-coordinator (not global); root assigns seq to its own events only |
| D-018 | No automatic job migration on failover — jobs stay on their assigned coordinator |

---

## Architecture Overview

```
New DB tables:    federation_events
New Go files:     server/federation_sync.go, server/federation_sync_test.go
                  server/federation_health.go, server/federation_health_test.go
New Vue files:    views/FederationHealth.vue
Modified:         agent/config/config.go         — coordinators list
                  agent/ws/ws.go                 — failover dial loop
                  coordinator/server/federation.go — sync endpoint, health endpoint
                  coordinator/server/federation_hub.go — heartbeat timeout → offline
                  coordinator/db/db.go            — federation_events table
                  dashboard/src/views/Federation.vue — health panel
                  dashboard/src/api.js            — federation health endpoint
```

**State sync flow:**
```
Spoke reconnects
  → sends GET /api/federation/sync?since=<last_seq>
  → root replays federation_events since that seq
  → spoke applies events to local DB
  → stale banner clears
```

**Agent failover flow:**
```
Agent config: coordinators: [root:8080, spoke1:8080, spoke2:8080]
  → dial root (primary)
  → root unreachable → try spoke1
  → spoke1 unreachable → try spoke2
  → all unreachable → retry loop with backoff (30s, 60s, 120s cap)
```

---

## New DB Schema

```sql
CREATE TABLE IF NOT EXISTS federation_events (
    id           INTEGER PRIMARY KEY AUTOINCREMENT,
    seq          INTEGER NOT NULL,
    coordinator  TEXT NOT NULL,          -- source coordinator ID
    event_type   TEXT NOT NULL,          -- agent_registered, agent_heartbeat, job_created, job_updated
    payload      TEXT NOT NULL,          -- JSON
    created_at   DATETIME DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_federation_events_seq ON federation_events(coordinator, seq);
```

---

## New API Endpoints

```
GET  /api/federation/sync?since=<seq>&coordinator=<id>   — replay events since seq (root only)
GET  /api/federation/health                               — all coordinators + status + lag
POST /api/federation/sync/ack                            — spoke acknowledges sync complete
```

---

## Task List

Tasks are ordered by dependency. Complete each fully before starting the next.
Run `go test ./coordinator/server/` after every backend task.

---

### PRE-FLIGHT
- [ ] Create branch `feature/phase-16-federation-ha` from `main`
- [ ] Confirm 183 tests pass: `go test ./...` from `C:\Projects\ArcVault2.0`

---

### TASK 1 — DB: federation_events table

**Files:** `coordinator/db/db.go`, `coordinator/db/federation_events.go` (new)

**Steps:**
1. Add `federation_events` table + index to `createTables()` in `db.go`
2. Create `coordinator/db/federation_events.go` with:
   - `AppendFederationEvent(coordinatorID, eventType, payload string) (seq int64, err error)`
   - `GetFederationEventsSince(coordinatorID string, sinceSeq int64) ([]FederationEvent, error)`
   - `PruneFederationEvents(olderThanDays int) error`
   - `FederationEvent` struct: `{ID, Seq, Coordinator, EventType, Payload, CreatedAt}`

**Verify:**
```powershell
go build ./coordinator/...
```

---

### TASK 2 — Event append: wire into existing broadcast points

**Files:** `coordinator/server/agents.go`, `coordinator/server/jobs.go`

**Steps:**
1. In `agents.go` — after existing `agent_registered` + heartbeat delta broadcasts, call `s.db.AppendFederationEvent(s.coordinatorID, "agent_registered", payload)`
2. In `jobs.go` — after job created/updated, call `s.db.AppendFederationEvent(s.coordinatorID, "job_created"/"job_updated", payload)`
3. `s.coordinatorID` needs to be set on Server struct — add to `server.go` init, read from config (default: hostname)

**Verify:**
```powershell
go test ./coordinator/server/ -run TestAgents
go test ./coordinator/server/ -run TestJobs
```

---

### TASK 3 — Sync endpoint: GET /api/federation/sync

**Files:** `coordinator/server/federation_sync.go` (new), `coordinator/server/federation_sync_test.go` (new)

**Steps:**
1. `handleFederationSync` — admin only, root only
   - Parse `since` (int64) and `coordinator` (string) query params
   - Call `s.db.GetFederationEventsSince(coordinator, since)`
   - Return `{events: [...], latest_seq: N}`
2. `handleFederationSyncAck` — POST, spoke confirms sync applied
   - Update `federation` table: set `last_seq = N` for that coordinator
3. Register both routes in `server.go`
4. Write tests: empty log, events present, invalid params

**Verify:**
```powershell
go test ./coordinator/server/ -run TestFederationSync
```

---

### TASK 4 — Spoke reconnect: auto-resync on reconnect

**Files:** `coordinator/server/federation_hub.go` (or federation_client.go — wherever spoke dial logic lives)

**Steps:**
1. On successful reconnect, spoke sends `GET /api/federation/sync?since=<last_known_seq>&coordinator=<self_id>`
2. Applies returned events to local DB (agent status updates, job status updates)
3. Sends POST `/api/federation/sync/ack` with `seq=<latest_seq>`
4. Clears internal `staleSince` flag → stale banner dismisses

**Verify:**
```powershell
go test ./coordinator/server/ -run TestFederationReconnect
```

---

### TASK 5 — Health endpoint: GET /api/federation/health

**Files:** `coordinator/server/federation_health.go` (new), `coordinator/server/federation_health_test.go` (new)

**Steps:**
1. `handleFederationHealth` — viewer+
   - For each coordinator in `federation` table:
     - `status`: online/offline (last heartbeat < 30s = online)
     - `last_seen`: timestamp of last heartbeat
     - `lag_events`: count of events since coordinator's `last_seq`
     - `agent_count`: agents homed to this coordinator
   - Return array of coordinator health objects
2. Register route in `server.go`
3. Write tests: all online, some offline, lag calculation

**Verify:**
```powershell
go test ./coordinator/server/ -run TestFederationHealth
```

---

### TASK 6 — Heartbeat timeout → offline marking

**Files:** `coordinator/server/federation_hub.go`

**Steps:**
1. Root coordinator runs a background goroutine (ticker, 15s interval)
2. For each registered coordinator: if last heartbeat > 30s ago, set `status=offline` in `federation` table
3. If status transitions online→offline, append `federation_events` entry of type `coordinator_offline`
4. On heartbeat received from a previously-offline coordinator: set `status=online`, trigger resync

**Verify:**
```powershell
go test ./coordinator/server/ -run TestFederationHeartbeat
```

---

### TASK 7 — Agent failover: coordinator list in config

**Files:** `agent/config/config.go`, `agent/ws/ws.go`

**Steps:**
1. `config.go` — add `Coordinators []string` field (list of coordinator URLs)
   - If `Coordinators` is empty, fall back to existing `CoordinatorURL` single field (backward compat)
2. `ws.go` — update dial loop:
   - On connection failure, try next coordinator in list
   - Full list exhausted → exponential backoff (30s → 60s → 120s cap), then retry from start
   - On successful connect, log which coordinator was used
3. No changes to server side — agent registration works the same regardless of which coordinator answers

**Verify:**
```powershell
go test ./agent/...
```

---

### TASK 8 — Prune job: scheduled cleanup of federation_events

**Files:** `coordinator/server/scheduler.go` (or new `federation_maintenance.go`)

**Steps:**
1. Add daily scheduled task: `s.db.PruneFederationEvents(7)` (7 days, configurable via config)
2. Add `FederationEventRetentionDays int` to config (default 7)
3. Log prune result (N events deleted)

**Verify:**
```powershell
go build ./coordinator/...
```

---

### TASK 9 — Frontend: FederationHealth.vue

**Files:** `dashboard/src/views/FederationHealth.vue` (new), `dashboard/src/api.js`, `dashboard/src/router/index.js`

**Design (impeccable — product register, restrained color strategy):**
- Table layout (not card grid — coordinators are data, not marketing objects)
- Columns: Coordinator ID | Status | Last Seen | Event Lag | Agent Count
- Status pill: green `online` / amber `reconnecting` / red `offline` — OKLCH colors, no gradients
- Lag column: `0 events` = dimmed; `>0` = amber number; `>50` = red (stale threshold)
- Auto-refresh every 15s (same pattern as Agents.vue)
- Empty state: "No federation peers registered" with link to federation setup docs
- Integrates into existing Federation.vue as a new tab OR replaces the basic peer list
- Role gate: viewer+ can see health; admin-only actions (force sync button) hidden for operator/viewer

**Steps:**
1. Add `getFederationHealth()` to `api.js`
2. Create `FederationHealth.vue` with table, status pills, auto-refresh
3. Add route `/federation/health` to `router/index.js`
4. Update `Federation.vue` to link to health view or embed as tab

---

### TASK 10 — Tests + smoke test + cleanup

**Steps:**
1. Run full suite: `go test ./...` — confirm all tests pass
2. Manual smoke test:
   - Start root coordinator
   - Start spoke coordinator (points to root)
   - Verify spoke appears in `/api/federation/health` as online
   - Kill root → verify spoke enters standalone mode (continues running)
   - Restart root → verify spoke reconnects and stale banner clears
   - Start agent with `coordinators` list → verify it connects to first available
3. Update `CONTEXT.md`: version → v0.9.0, test count, phase status
4. Update `MEMORY.md`: Phase 16 entry, new files, design decisions D-013–D-018

---

### TASK 11 — Commit + tag

```powershell
git add -A
git commit -m "Phase 16 complete: federation failover, load balancing, state consistency (v0.9.0)"
git tag -a v0.9.0 -m "v0.9.0 — Phase 16: HA federation with failover, agent load balancing, event log sync"
git push origin feature/phase-16-federation-ha
git push origin v0.9.0
```

---

## Summary Table

| Task | Area | New Files | Modified Files | Effort |
|------|------|-----------|----------------|--------|
| 1 | DB: events table | `db/federation_events.go` | `db/db.go` | 1–2h |
| 2 | Event append wiring | — | `server/agents.go`, `server/jobs.go`, `server/server.go` | 1h |
| 3 | Sync endpoint | `server/federation_sync.go`, `_test.go` | `server/server.go` | 2–3h |
| 4 | Spoke auto-resync | — | `server/federation_hub.go` or client | 2h |
| 5 | Health endpoint | `server/federation_health.go`, `_test.go` | `server/server.go` | 1–2h |
| 6 | Heartbeat → offline | — | `server/federation_hub.go` | 1–2h |
| 7 | Agent failover | — | `agent/config/config.go`, `agent/ws/ws.go` | 2h |
| 8 | Event pruning | — | `server/scheduler.go` or new file | 30m |
| 9 | FederationHealth.vue | `views/FederationHealth.vue` | `api.js`, `router/index.js`, `Federation.vue` | 2–3h |
| 10 | Tests + smoke + cleanup | — | `CONTEXT.md`, `MEMORY.md` | 1–2h |
| 11 | Commit + tag | — | — | 15m |

**Total estimate:** 2–3 weeks (solo, part-time)

---

## Rules for This Plan

- ❌ Never rewrite without explicit approval
- ✅ Pre-flight: branch from main, confirm 183 tests pass before starting
- ✅ Test after every backend task (`go test ./coordinator/server/`)
- ✅ Proof before "done" — no claiming complete without test output
- ✅ Bugs traced to root cause before fixing
- ✅ D-013–D-018 decisions are locked — check before deviating
- ✅ PowerShell line continuation: backtick (`), not backslash
- ✅ Backward compat required: single `coordinator_url` in agent config still works (D-015)

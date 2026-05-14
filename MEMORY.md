# ArcVault Project Memory
**Project Name:** ArcVault
**Type:** OS-agnostic Backup Orchestrator
**Status:** Phase 4 — Tasks 1-3 Complete
**Last Updated:** May 14, 2026

---

## Project Vision
ArcVault solves key limitations in RoboBackup:
- RoboBackup: Windows-only, limited monitoring, no remote visibility
- ArcVault: Cross-platform (Windows/Mac/Linux), real-time monitoring dashboard, self-hosted, agents coordinate through central coordinator

**Architecture:** Lightweight agents on each machine -> central Go coordinator -> Vue.js web dashboard

---

## Current Status

### Phase 1: COMPLETE
### Phase 2: COMPLETE
### Phase 3: COMPLETE

### Phase 4: IN PROGRESS (Tasks 1-3 done)

**Task 1 -- GET /api/jobs/{id}/runs: COMPLETE**
- Returns all job_runs for a given job, ordered by finished_at DESC
- 404 if job not found, empty array if no runs
- History.vue updated to use real endpoint

**Task 2 -- Agent offline detection: COMPLETE**
- coordinator/server/offline_detector.go
- detectOfflineAgents(threshold) -- marks stale agents offline, broadcasts agent.updated
- StartOfflineDetector(interval, threshold) -- runs on ticker, called from Start()
- Default: check every 60s, mark offline after 90s
- Agents.vue refreshes on agent.updated WebSocket events

**Task 3 -- Job scheduling: COMPLETE**
- coordinator/server/scheduler.go
- triggerScheduledJobs() -- resets completed/failed scheduled jobs to pending
- StartScheduler() -- loads jobs, registers with robfig/cron, 60s fallback ticker
- Only resets jobs with non-empty schedule field
- Does not interrupt running jobs
- Broadcasts job.updated on reschedule
- Called from coordinator Start()

**Task 4 -- Production build: NOT STARTED**
- npm run build in dashboard/
- Coordinator serves dashboard/dist as static files
- Single deployment: coordinator binary + dist/

---

## Technical Details

### Stack
- **Language:** Go (coordinator + agents)
- **Frontend:** Vue 3 + Vite 8, vue-router@4
- **Database:** SQLite via modernc.org/sqlite (pure Go, no CGO)
- **Authentication:** Single admin token (Bearer header or ?token= for WS)
- **Sync Tools:** Robocopy (Windows), Rsync (Unix/Mac)
- **WebSocket:** github.com/gorilla/websocket v1.5.3
- **Scheduler:** github.com/robfig/cron/v3
- **Module:** single monorepo, module name: `arcvault`

### Dependencies (Go)
- modernc.org/sqlite
- gopkg.in/yaml.v3
- github.com/gorilla/websocket v1.5.3
- github.com/robfig/cron/v3
- golang.org/x/crypto

### Dependencies (JS)
- vite@8, vue@3, vue-router@4

### Project Layout
```
coordinator/
  main.go
  cmd/commands.go
  config/config.go
  db/db.go
  server/
    server.go              -- CORS, routes, Start() wires all bg services
    agents.go
    hub.go                 -- WebSocket hub, ?token= auth
    jobs.go
    job_status.go
    job_results.go
    job_runs.go            -- GET /api/jobs/{id}/runs
    offline_detector.go    -- agent offline detection
    scheduler.go           -- cron job scheduling
    hub_test.go
    jobs_test.go
    jobs_status_results_test.go
    job_runs_test.go
    offline_detector_test.go
    scheduler_test.go
agent/
  main.go
  config/config.go
  heartbeat/heartbeat.go
  runner/
    runner.go
    runner_test.go
    executor.go
dashboard/
  src/
    main.js
    App.vue
    api.js
    router/index.js
    composables/useWebSocket.js
    views/
      Agents.vue
      Jobs.vue
      History.vue
go.mod
```

### Test Count
- 45 tests total, all passing
- coordinator/server: 40 tests
- agent/runner: 5 tests

### API Endpoints
| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET    | /health | No | Health check |
| GET    | /ws | ?token= | WebSocket |
| POST   | /api/agents/register | Bearer | Register agent |
| POST   | /api/agents/{id}/heartbeat | Bearer | Agent heartbeat |
| GET    | /api/agents | Bearer | List agents |
| POST   | /api/jobs | Bearer | Create job |
| GET    | /api/jobs | Bearer | List jobs (?agent_id=) |
| GET    | /api/jobs/{id} | Bearer | Get job |
| DELETE | /api/jobs/{id} | Bearer | Delete job |
| PATCH  | /api/jobs/{id}/status | Bearer | Update status |
| POST   | /api/jobs/{id}/results | Bearer | Store run result |
| GET    | /api/jobs/{id}/runs | Bearer | List job runs |

### WebSocket Events
| Type | Trigger | Payload |
|------|---------|---------|
| job.updated | PATCH status / scheduler reschedule | {id, status} |
| job.result | POST results | {job_id, exit_code} |
| agent.updated | offline detector | {id, status} |

### Schemas
**jobs:** id, agent_id, name, source_path, dest_path, schedule, status, created_at
**job_runs:** id, job_id, started_at, finished_at, exit_code, output
**agents:** id, hostname, os, version, status, last_seen, registered_at

---

## Windows Development Notes
- Run tests from repo root: `cd C:\Projects\ArcVault2.0 && go test ./... -v`
- Watch for duplicate files -- causes redeclaration errors, use Get-ChildItem to check
- Dashboard dev: `cd dashboard && npm run dev` → http://localhost:5173
- Coordinator: `go run ./coordinator start`
- Always use PowerShell WriteAllText with UTF8 no-BOM when writing Go files manually

---

## Phase 4 Roadmap
1. GET /api/jobs/{id}/runs ✅
2. Agent offline detection ✅
3. Job scheduling (cron) ✅
4. Production build ⬜ -- NEXT
   - npm run build → dashboard/dist
   - Coordinator serves dist/ as static files on GET /
   - Single deployment unit

---

## Git Status
**Latest commit:** Phase 4 Tasks 1-3 complete
**Remote:** https://github.com/castrokren/ArcVault
**Branch:** main

---
**End of Memory Document**

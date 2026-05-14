# ArcVault Project Memory
**Project Name:** ArcVault
**Type:** OS-agnostic Backup Orchestrator
**Status:** Phase 4 COMPLETE
**Last Updated:** May 14, 2026

---

## Project Vision
ArcVault solves key limitations in RoboBackup:
- RoboBackup: Windows-only, limited monitoring, no remote visibility
- ArcVault: Cross-platform (Windows/Mac/Linux), real-time monitoring dashboard, self-hosted, agents coordinate through central coordinator

**Architecture:** Lightweight agents on each machine -> central Go coordinator -> Vue.js web dashboard

---

## Current Status

### Phase 1: COMPLETE — binaries, init/start commands, YAML config
### Phase 2: COMPLETE — SQLite, HTTP server, agent register + heartbeat
### Phase 3: COMPLETE — job CRUD, agent runner, WebSocket, Vue dashboard
### Phase 4: COMPLETE — job runs history, offline detection, cron scheduling, production build

---

## Phase 4 Details

**Task 1 -- GET /api/jobs/{id}/runs: COMPLETE**
- coordinator/server/job_runs.go
- Returns job_runs ordered by finished_at DESC
- 404 if job not found, empty array [] if no runs
- History.vue uses real endpoint

**Task 2 -- Agent offline detection: COMPLETE**
- coordinator/server/offline_detector.go
- detectOfflineAgents(threshold) -- marks stale agents offline, broadcasts agent.updated
- StartOfflineDetector(60s interval, 90s threshold) -- called from Start()
- Agents.vue refreshes on agent.updated events

**Task 3 -- Job scheduling: COMPLETE**
- coordinator/server/scheduler.go
- triggerScheduledJobs() -- resets completed/failed scheduled jobs to pending
- StartScheduler() -- robfig/cron per job + 60s fallback ticker
- Skips jobs without schedule, skips running jobs
- Broadcasts job.updated on reschedule

**Task 4 -- Production build: COMPLETE**
- dashboard/dist built with `npm run build`
- coordinator/server/server.go -- NewWithStatic(cfg, db, staticDir)
- Serves dashboard/dist as static files on GET /
- Gracefully skips if dist not found (tests pass empty string)
- Single deployment: `go run ./coordinator start` from repo root

---

## Technical Details

### Stack
- **Language:** Go (coordinator + agents)
- **Frontend:** Vue 3 + Vite 8, vue-router@4 (hash history)
- **Database:** SQLite via modernc.org/sqlite (pure Go, no CGO)
- **Authentication:** Single admin token (Bearer or ?token= for WS)
- **Sync Tools:** Robocopy (Windows, exit 1-7 = success), Rsync (Unix/Mac)
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
    server.go                  -- CORS, routes, static, Start() wires all services
    agents.go
    hub.go                     -- WebSocket hub, ?token= auth
    jobs.go
    job_status.go
    job_results.go
    job_runs.go                -- GET /api/jobs/{id}/runs
    offline_detector.go
    scheduler.go
    hub_test.go
    jobs_test.go               -- newTestServer uses NewWithStatic("") 
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
  dist/                        -- production build output
go.mod
```

### Test Count
- 45 tests total, all passing
- coordinator/server: 40 tests
- agent/runner: 5 tests

### Key Design Decisions
- Hash routing (`createWebHashHistory`) -- no server-side routing needed, static serving just works
- WS auth via `?token=` query param -- browsers can't set headers on WS connections
- `NewWithStatic(cfg, db, "")` in tests -- skips static serving gracefully
- robfig/cron per-job + 60s fallback ticker -- handles jobs created after startup
- Offline threshold 90s -- agent heartbeats every 30s, so 3 missed = offline

### Production Run
```powershell
cd C:\Projects\ArcVault2.0
go run ./coordinator start   # port 443, serves API + dashboard
go run ./agent               # on each machine
```

### Dashboard access
- Dev: http://localhost:5173 (npm run dev in dashboard/)
- Production: http://localhost:443 (coordinator serves dist/)

---

## Windows Development Notes
- Run tests from repo root: `go test ./... -v`
- Watch for duplicate files -- use Get-ChildItem, delete with Remove-Item
- Use PowerShell WriteAllText with UTF8 no-BOM for Go files
- npm run build from dashboard/ before deploying

---

## Git Status
**Latest commit:** Phase 4 complete
**Remote:** https://github.com/castrokren/ArcVault
**Branch:** main

---
**End of Memory Document**

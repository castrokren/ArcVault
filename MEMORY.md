# ArcVault Project Memory
**Project Name:** ArcVault
**Type:** OS-agnostic Backup Orchestrator
**Status:** Phase 3 COMPLETE
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
- Go binaries compile on Windows
- coordinator.exe init / start
- agent.exe reads YAML config
- GitHub: https://github.com/castrokren/ArcVault

### Phase 2: COMPLETE
- coordinator.exe start -- SQLite, HTTP server
- agent.exe -- registers, heartbeats every 30s

### Phase 3: COMPLETE (31 tests passing)

**Task 1 -- Job CRUD: COMPLETE**
- POST/GET/DELETE /api/jobs, GET /api/jobs/{id}
- Required fields: agent_id, name, source_path, dest_path
- Optional: schedule
- GET /api/jobs supports ?agent_id= filter

**Task 2 -- Job Runner: COMPLETE**
- agent/runner/runner.go -- polls every 30s
- GET /api/jobs?agent_id=...&status=pending
- Claims (PATCH running), executes, posts results, marks completed/failed
- agent/runner/executor.go -- RealExecutor: robocopy (Windows, exit 1-7 = success), rsync (Unix)
- agent/main.go -- heartbeat + runner as goroutines, SIGINT/SIGTERM shutdown

**Task 3 -- Job Results: COMPLETE**
- POST /api/jobs/{id}/results -- stores in job_runs table
- PATCH /api/jobs/{id}/status -- pending/running/completed/failed

**Task 4 -- WebSocket: COMPLETE**
- coordinator/server/hub.go -- gorilla/websocket hub
- GET /ws -- token via ?token= query param (browsers can't set WS headers)
- Events: job.updated, job.result
- Event shape: {"type": "...", "payload": {...}}

**Task 5 -- Vue Dashboard: COMPLETE**
- Vue 3 + Vite 8, Node 24, vue-router@4
- Token gate on first load, stored in localStorage
- Agents view: table with status badges, last seen
- Jobs view: table + create form + delete, status badges
- History view: job runs (falls back gracefully if no /runs endpoint)
- Real-time: WebSocket lastEvent prop passed to all views, auto-refresh on events
- CORS: coordinator wraps router with corsMiddleware

---

## Technical Details

### Stack
- **Language:** Go (coordinator + agents)
- **Frontend:** Vue 3 + Vite 8, vue-router@4
- **Database:** SQLite via modernc.org/sqlite (pure Go, no CGO)
- **Authentication:** Single admin token (Bearer header or ?token= for WS)
- **Sync Tools:** Robocopy (Windows), Rsync (Unix/Mac)
- **WebSocket:** github.com/gorilla/websocket v1.5.3
- **Module:** single monorepo, module name: `arcvault`

### Dependencies (Go)
- modernc.org/sqlite
- gopkg.in/yaml.v3
- github.com/gorilla/websocket v1.5.3
- github.com/robfig/cron/v3 (available, not yet used)
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
    server.go         -- CORS middleware, routes
    agents.go
    hub.go            -- WebSocket hub, ?token= auth
    jobs.go
    job_status.go
    job_results.go
    hub_test.go
    jobs_test.go
    jobs_status_results_test.go
agent/
  main.go             -- goroutines + signal shutdown
  config/config.go
  heartbeat/heartbeat.go
  runner/
    runner.go
    runner_test.go
    executor.go       -- robocopy/rsync
dashboard/
  src/
    main.js
    App.vue           -- nav, token gate, WS indicator
    api.js            -- fetch wrapper
    router/index.js
    composables/useWebSocket.js
    views/
      Agents.vue
      Jobs.vue
      History.vue
go.mod
```

### Test Count
- 31 tests total, all passing
- coordinator/server: 26 tests
- agent/runner: 5 tests

### API Endpoints
| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET    | /health | No | Health check |
| GET    | /ws | ?token= | WebSocket |
| POST   | /api/agents/register | Bearer | Register agent |
| POST   | /api/agents/{id}/heartbeat | Bearer | Agent heartbeat |
| GET    | /api/agents | Bearer | List all agents |
| POST   | /api/jobs | Bearer | Create job |
| GET    | /api/jobs | Bearer | List jobs (?agent_id=) |
| GET    | /api/jobs/{id} | Bearer | Get job |
| DELETE | /api/jobs/{id} | Bearer | Delete job |
| PATCH  | /api/jobs/{id}/status | Bearer | Update status |
| POST   | /api/jobs/{id}/results | Bearer | Store run result |

### WebSocket Events
| Type | Trigger | Payload |
|------|---------|---------|
| job.updated | PATCH status | {id, status} |
| job.result | POST results | {job_id, exit_code} |

### Schemas
**jobs:** id, agent_id, name, source_path, dest_path, schedule, status, created_at
**job_runs:** id, job_id, started_at, finished_at, exit_code, output

---

## Windows Development Notes
- PowerShell: use `[System.IO.File]::WriteAllText` with UTF8 no-BOM encoding
- Run tests from repo root: `cd C:\Projects\ArcVault2.0 && go test ./... -v`
- Watch for duplicate files (server_2.go etc) -- causes redeclaration errors
- Dashboard dev server: `cd dashboard && npm run dev` → http://localhost:5173

---

## Phase 4 Candidates
1. **Job scheduling** -- cron expressions via robfig/cron (already in go.mod)
2. **GET /api/jobs/{id}/runs** -- full history view in dashboard
3. **Production build** -- `npm run build` + serve static from coordinator, or goreleaser
4. **Agent offline detection** -- mark agents offline if no heartbeat for N minutes
5. **Multi-token auth** -- per-agent tokens instead of single admin token

---

## Git Status
**Latest commit:** Phase 3 complete
**Remote:** https://github.com/castrokren/ArcVault
**Branch:** main

---
**End of Memory Document**

# ArcVault Project Memory
**Project Name:** ArcVault
**Type:** OS-agnostic Backup Orchestrator
**Status:** Phase 3 — Tasks 1-4 Complete
**Last Updated:** May 13, 2026

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
- coordinator.exe init -- prompts for port/db path, generates token
- agent.exe -- reads YAML config
- Project structure established
- Code pushed to GitHub (https://github.com/castrokren/ArcVault)

### Phase 2: COMPLETE
- coordinator.exe start -- initializes SQLite, starts HTTP server
- agent.exe -- registers, sends heartbeat every 30s
- SQLite schema: agents, tokens, jobs, job_runs tables

### Phase 3: IN PROGRESS (Tasks 1-4 done)

**Task 1 -- Job CRUD: COMPLETE**
- POST/GET/DELETE/GET /api/jobs with source_path, dest_path fields
- GET /api/jobs supports ?agent_id= filter
- jobs table: id, agent_id, name, source_path, dest_path, schedule, status, created_at

**Task 2 -- Job Runner: COMPLETE**
- agent/runner/runner.go -- polls coordinator every 30s
- Fetches pending jobs for this agent via GET /api/jobs?agent_id=...&status=pending
- Claims job (PATCH status=running), executes robocopy/rsync, posts results
- agent/runner/executor.go -- RealExecutor: robocopy (Windows, exit codes 1-7 normalized to 0), rsync (Unix)
- agent/main.go -- heartbeat + runner both run as goroutines, blocks on SIGINT/SIGTERM

**Task 3 -- Job Results endpoint: COMPLETE**
- POST /api/jobs/{id}/results -- stores exit_code + output in job_runs table
- PATCH /api/jobs/{id}/status -- updates job status (pending/running/completed/failed)

**Task 4 -- WebSocket: COMPLETE**
- coordinator/server/hub.go -- Hub struct, gorilla/websocket, broadcasts to all clients
- GET /ws -- WebSocket upgrade endpoint (Bearer auth required)
- Events: job.updated (on PATCH status), job.result (on POST results)
- Event shape: {"type": "job.updated", "payload": {"id": "...", "status": "..."}}

**Task 5 -- Vue Dashboard: NOT STARTED**
- Will live in dashboard/
- Vue 3 + Vite
- Agent status panel, job list, job history
- Real-time updates via WebSocket

---

## Technical Details

### Stack
- **Language:** Go (coordinator + agents)
- **Frontend:** Vue 3 + Vite (not yet started)
- **Database:** SQLite via modernc.org/sqlite (pure Go, no CGO)
- **Authentication:** Token-based (crypto/rand generated, stored in config.json)
- **Sync Tools:** Robocopy (Windows), Rsync (Unix/Mac)
- **WebSocket:** github.com/gorilla/websocket v1.5.3
- **Module:** single monorepo at repo root, module name: `arcvault`

### Dependencies
- modernc.org/sqlite -- SQLite driver (pure Go)
- gopkg.in/yaml.v3 -- agent YAML config parsing
- github.com/gorilla/websocket v1.5.3 -- WebSocket
- github.com/robfig/cron/v3 -- available for job scheduling (future)
- golang.org/x/crypto

### Project Layout
```
coordinator/
  main.go
  cmd/commands.go
  config/config.go
  db/db.go
  server/
    server.go
    agents.go
    hub.go
    jobs.go
    job_status.go
    job_results.go
    hub_test.go
    jobs_test.go
    jobs_status_results_test.go
agent/
  main.go
  config/config.go
  heartbeat/heartbeat.go
  runner/
    runner.go
    runner_test.go
    executor.go
dashboard/         <-- Task 5, not started
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
| GET    | /ws | Bearer | WebSocket |
| POST   | /api/agents/register | Bearer | Register agent |
| POST   | /api/agents/{id}/heartbeat | Bearer | Agent heartbeat |
| GET    | /api/agents | Bearer | List all agents |
| POST   | /api/jobs | Bearer | Create job |
| GET    | /api/jobs | Bearer | List jobs (?agent_id= filter) |
| GET    | /api/jobs/{id} | Bearer | Get job |
| DELETE | /api/jobs/{id} | Bearer | Delete job |
| PATCH  | /api/jobs/{id}/status | Bearer | Update job status |
| POST   | /api/jobs/{id}/results | Bearer | Store job run result |

### WebSocket Events
| Type | Trigger | Payload |
|------|---------|---------|
| job.updated | PATCH /api/jobs/{id}/status | {id, status} |
| job.result | POST /api/jobs/{id}/results | {job_id, exit_code} |

### Job Schema
```
id          TEXT PRIMARY KEY
agent_id    TEXT NOT NULL
name        TEXT NOT NULL
source_path TEXT NOT NULL
dest_path   TEXT NOT NULL
schedule    TEXT (nullable)
status      TEXT DEFAULT 'pending'  -- pending/running/completed/failed
created_at  DATETIME
```

### Job Run Schema
```
id          TEXT PRIMARY KEY
job_id      TEXT NOT NULL
started_at  DATETIME (nullable)
finished_at DATETIME
exit_code   INTEGER
output      TEXT
```

### agent-config.yaml
```yaml
agent_id: agent-01
hostname: my-machine
os: windows
coordinator_url: http://localhost:443
auth_token: <token-from-init>
version: 0.1.0
```

---

## Windows Development Notes

### PowerShell File Creation (No BOM)
```powershell
$utf8 = New-Object System.Text.UTF8Encoding $false
[System.IO.File]::WriteAllText("C:\full\path\file.go", $content, $utf8)
```

### Run tests from repo root
```powershell
cd C:\Projects\ArcVault2.0
go test ./coordinator/server/... -v
go test ./... -v
```

### Watch out for duplicate files
Stray numbered files (e.g. server_2.go, job_results_3.go) cause redeclaration errors.
Always check with Get-ChildItem and delete duplicates before running tests.

---

## Phase 3 Roadmap
1. Job CRUD ✅
2. Job Runner ✅
3. Job Results endpoint ✅
4. WebSocket ✅
5. Vue Dashboard ⬜ -- NEXT

---

## Git Status
**Latest commit:** Phase 3 Task 4 complete -- WebSocket hub
**Remote:** https://github.com/castrokren/ArcVault
**Branch:** main

---
**End of Memory Document**

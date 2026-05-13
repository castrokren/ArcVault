# ArcVault Project Memory
**Project Name:** ArcVault
**Type:** OS-agnostic Backup Orchestrator
**Status:** Phase 3 In Progress
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
- Project structure established (coordinator/, agent/, dashboard/)
- Code pushed to GitHub (https://github.com/castrokren/ArcVault)

### Phase 2: COMPLETE
**What Works:**
- coordinator.exe start -- initializes SQLite, starts HTTP server
- GET  /health -- unauthenticated health check
- POST /api/agents/register -- agent registration with token auth
- POST /api/agents/{id}/heartbeat -- agent heartbeat with token auth
- GET  /api/agents -- list all agents with token auth
- agent.exe -- registers with coordinator on startup, sends heartbeat every 30s
- SQLite schema: agents, tokens, jobs, job_runs tables

### Phase 3: IN PROGRESS
**Coordinator: COMPLETE (21 tests passing)**
- POST   /api/jobs -- create job (requires agent_id, name, source_path, dest_path)
- GET    /api/jobs -- list jobs, supports ?agent_id= filter
- GET    /api/jobs/{id} -- get single job
- DELETE /api/jobs/{id} -- delete job
- PATCH  /api/jobs/{id}/status -- update status (pending/running/completed/failed)
- POST   /api/jobs/{id}/results -- store job run result (exit_code, output)

**Agent Runner: NOT STARTED**
- Will live at agent/runner/runner.go
- Polls coordinator for pending jobs every 30s
- Claims job (sets status=running), executes, posts results

---

## Technical Details

### Stack
- **Language:** Go (coordinator + agents)
- **Frontend:** Vue 3 + Vite (not yet started)
- **Database:** SQLite via modernc.org/sqlite (pure Go, no CGO)
- **Authentication:** Token-based (crypto/rand generated, stored in config.json)
- **Sync Tools:** Robocopy (Windows), Rsync (Unix/Mac)
- **Module:** single monorepo, module name: `arcvault`

### Dependencies
- modernc.org/sqlite -- SQLite driver (pure Go)
- gopkg.in/yaml.v3 -- agent YAML config parsing
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
    jobs.go
    job_status.go
    job_results.go
    jobs_test.go
    jobs_status_results_test.go
agent/
  main.go
  config/config.go
  heartbeat/heartbeat.go
  runner/          <-- NEXT
dashboard/         <-- future
```

### API Endpoints
| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET    | /health | No | Health check |
| POST   | /api/agents/register | Bearer | Register agent |
| POST   | /api/agents/{id}/heartbeat | Bearer | Agent heartbeat |
| GET    | /api/agents | Bearer | List all agents |
| POST   | /api/jobs | Bearer | Create job |
| GET    | /api/jobs | Bearer | List jobs (?agent_id= filter) |
| GET    | /api/jobs/{id} | Bearer | Get job |
| DELETE | /api/jobs/{id} | Bearer | Delete job |
| PATCH  | /api/jobs/{id}/status | Bearer | Update job status |
| POST   | /api/jobs/{id}/results | Bearer | Store job run result |

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
# RIGHT (no BOM)
$utf8 = New-Object System.Text.UTF8Encoding $false
[System.IO.File]::WriteAllText("C:\full\path\file.go", $content, $utf8)
```

### Always Use Full Paths
```powershell
[System.IO.File]::WriteAllText("C:\Projects\ArcVault2.0\file.go", $content, $utf8)
```

### Run tests from repo root
```powershell
cd C:\Projects\ArcVault2.0
go test ./coordinator/server/... -v
go test ./... -v
```

---

## Phase 3 Roadmap
**Goal:** Job scheduling and Vue dashboard

### Tasks (in order):
1. **Job CRUD** ✅ -- POST/GET/DELETE /api/jobs
2. **Job Runner** -- Execute robocopy/rsync jobs on agent (NEXT)
3. **Job Results** ✅ -- POST /api/jobs/{id}/results endpoint
4. **WebSocket** -- Real-time dashboard updates
5. **Vue Dashboard** -- Agent status, job list, job history

---

## Git Status
**Latest commit:** Phase 3 Task 1c complete
**Remote:** https://github.com/castrokren/ArcVault
**Branch:** main

---
**End of Memory Document**

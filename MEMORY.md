# ArcVault Project Memory
**Project Name:** ArcVault
**Type:** OS-agnostic Backup Orchestrator
**Status:** Phase 2 Complete
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
- Go 1.26.3 binaries compile on Windows
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

**Key Decisions Made:**
- Used modernc.org/sqlite (pure Go, no CGO/GCC required on Windows)
- Used stdlib net/http router (Go 1.22+ method+path matching, no Gorilla Mux)
- Token auth via Authorization: Bearer <token> header
- Config stored at ~/.arcvault/config.json

---

## Technical Details

### Stack
- **Language:** Go 1.26.3 (coordinator + agents)
- **Frontend:** Vue 3 + Vite (not yet started)
- **Database:** SQLite via modernc.org/sqlite (pure Go, no CGO)
- **Authentication:** Token-based (crypto/rand generated, stored in config.json)
- **Sync Tools:** Robocopy (Windows), Rsync (Unix/Mac)

### Dependencies
- modernc.org/sqlite -- SQLite driver (pure Go)
- gopkg.in/yaml.v3 -- agent YAML config parsing

### Project Layout
coordinator/
main.go          -- entry point, routes init/start/help commands
cmd/commands.go  -- InitCommand, StartCommand
config/config.go -- Config struct, Save/Load/GetConfigPath
db/db.go         -- DB struct, Init, migrate (schema)
server/
server.go      -- Server struct, New, Start, registerRoutes, authMiddleware
agents.go      -- handleRegister, handleHeartbeat, handleListAgents
agent/
main.go               -- entry point, loads config, registers, starts heartbeat
config/config.go      -- YAML config loader
heartbeat/heartbeat.go -- Register, Start, send

### API Endpoints
| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET  | /health | No | Health check |
| POST | /api/agents/register | Bearer token | Register agent |
| POST | /api/agents/{id}/heartbeat | Bearer token | Agent heartbeat |
| GET  | /api/agents | Bearer token | List all agents |

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
# WRONG (adds BOM)
@"content"@ | Out-File -Encoding UTF8 file.go

# RIGHT (no BOM)
$utf8 = New-Object System.Text.UTF8Encoding $false
[System.IO.File]::WriteAllText("C:\full\path\file.go", $content, $utf8)
```

### Always Use Full Paths
```powershell
# WRONG - resolves to C:\WINDOWS\system32
[System.IO.File]::WriteAllText("file.go", $content, $utf8)

# RIGHT
[System.IO.File]::WriteAllText("C:\Projects\ArcVault2.0\file.go", $content, $utf8)
```

---

## Phase 3 Roadmap
**Goal:** Job scheduling and Vue dashboard

### Tasks (in order):
1. **Job CRUD** -- POST/GET/DELETE /api/jobs endpoints
2. **Job Runner** -- Execute robocopy/rsync jobs on agent
3. **Job Results** -- POST /api/jobs/{id}/results endpoint
4. **WebSocket** -- Real-time dashboard updates
5. **Vue Dashboard** -- Agent status, job list, job history

---

## Git Status
**Latest commit:** Phase 2 complete
**Remote:** https://github.com/castrokren/ArcVault
**Branch:** main

---
**End of Memory Document**
# ArcVault Project Memory

**Project Name:** ArcVault  
**Type:** OS-agnostic Backup Orchestrator  
**Status:** Phase 1 Complete  
**Last Updated:** May 13, 2026

---

## Project Vision

ArcVault solves key limitations in RoboBackup:
- ❌ RoboBackup: Windows-only, limited monitoring, no remote visibility
- ✅ ArcVault: Cross-platform (Windows/Mac/Linux), real-time monitoring dashboard, self-hosted, agents coordinate through central coordinator

**Architecture:** Lightweight agents on each machine → central Go coordinator → Vue.js web dashboard

---

## Current Status

### Phase 1: ✅ COMPLETE

**What Works:**
- Go 1.26.3 binaries compile on Windows
- coordinator.exe init — prompts for port/db path, generates token
- gent.exe — reads YAML config, ready for registration
- Project structure established (coordinator/, agent/, dashboard/)
- Code pushed to GitHub (https://github.com/castrokren/ArcVault)

**What's Not Yet Implemented:**
- HTTP server (coordinator receives requests)
- Database operations (SQLite)
- Agent registration & heartbeat loop
- Job scheduling
- Dashboard connectivity

---

## Technical Details

### Stack
- **Language:** Go 1.26.3 (coordinator + agents)
- **Frontend:** Vue 3 + Vite
- **Database:** SQLite (self-hosted)
- **Authentication:** Token-based (crypto/rand generated)
- **Sync Tools:** Robocopy (Windows), Rsync (Unix/Mac)

### Project Layout
### Key Files

**coordinator/main.go**
- Handles init, start, help commands
- init → prompts for port, db path → generates token
- Token example: 0552fd7095c30b2829b73a1f97b7bb91e0435e2ccebc5126cc8f33637b59f78

**agent-config.yaml**
`yaml
agent_id: agent-01
hostname: my-machine
os: windows
coordinator_url: http://localhost:443
auth_token: <token-from-init>
version: 0.1.0
`

---

## Phase 2 Roadmap

**Goal:** Make coordinator and agents actually communicate

### Tasks (in order):
1. **HTTP Server** — Gorilla mux router in coordinator/server/
2. **Database** — SQLite schema + migrations (agents, jobs, job_runs, tokens)
3. **Agent Registration** — POST /api/agents/register endpoint
4. **Heartbeat Loop** — Agent sends status every 30 seconds
5. **Job Scheduling** — Cron-based triggers
6. **WebSocket** — Real-time dashboard updates
7. **API Endpoints** — CRUD for jobs, job execution, result reporting

---

## Windows Development Notes

### PowerShell File Creation Issues Solved

**Problem 1: BOM in UTF-8 files**
`powershell
# ❌ WRONG (adds BOM)
@"content"@ | Out-File -Encoding UTF8 file.go

# ✅ RIGHT (no BOM)
System.Text.UTF8Encoding = New-Object System.Text.UTF8Encoding False
[System.IO.File]::WriteAllText("C:\full\path\file.go", , System.Text.UTF8Encoding)
`

**Problem 2: Relative paths resolve to C:\WINDOWS\system32**
`powershell
# ❌ WRONG
[System.IO.File]::WriteAllText("go.mod", , System.Text.UTF8Encoding)

# ✅ RIGHT (use full path)
[System.IO.File]::WriteAllText("C:\Projects\ArcVault2.0\go.mod", , System.Text.UTF8Encoding)
`

---

## How to Continue Phase 2

1. cd C:\Projects\ArcVault2.0
2. Create coordinator/server/server.go with HTTP server
3. Create coordinator/db/db.go with SQLite initialization
4. Implement /api/agents/register and /api/agents/{id}/heartbeat endpoints
5. Implement agent heartbeat loop
6. Test with manual requests
7. Connect dashboard

---

## Git Status

**Latest commit:** Phase 1 complete with working binaries
**Remote:** https://github.com/castrokren/ArcVault
**Branch:** main

---

**End of Memory Document**
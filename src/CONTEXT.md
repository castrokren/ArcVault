# Building Workspace

**Last updated:** May 22, 2026

## What happens here

Writing Go code (coordinator + agent), building the Vue dashboard, testing, and debugging. Code lives in `coordinator/`, `agent/`, and `dashboard/` — not in this folder.

## Build commands

```powershell
go build -o dist/coordinator.exe ./coordinator/
go build -o dist/agent.exe ./agent/
go test ./...
go vet ./...
go test ./coordinator/...
```

## Architecture (v0.9.0)

```
Agent (agent-config.yaml with coordinators list)
  ├─ WebSocket to coordinator with failover
  └─ HTTP POST /api/agents/register
       └─ Coordinator (Gorilla mux, JWT-protected)
            ├─ SQLite (agents, jobs, job_runs, tokens, federation_events, federation)
            ├─ Cron scheduler (robfig/cron)
            ├─ Federation events log (per-coordinator monotonic sequence)
            ├─ State sync (root→spoke via federation_events)
            └─ WebSocket → Vue 3 dashboard (RBAC protected)
```

## Key files (Latest)

- `coordinator/main.go` — CLI entry: `init | start | help`
- `coordinator/cmd/commands.go` — InitCommand(), StartCommand()
- `coordinator/config/config.go` — Full config, including CoordinatorID
- `coordinator/db/db.go` — SQLite schema with federation_events table
- `coordinator/db/federation_events.go` — Event log operations (NEW Phase 16)
- `coordinator/server/federation_sync.go` — State sync endpoints (NEW Phase 16)
- `coordinator/server/federation_health.go` — Health monitoring (NEW Phase 16)
- `agent/config/config.go` — Coordinator list for failover
- `agent/ws/ws.go` — WebSocket client with failover logic

## Dependencies

| Package | Purpose |
|---|---|
| github.com/gorilla/mux | HTTP router |
| github.com/gorilla/websocket | WebSocket |
| github.com/golang-jwt/jwt/v5 | JWT tokens |
| github.com/robfig/cron/v3 | Job scheduling |
| github.com/mattn/go-sqlite3 | SQLite (requires CGO) |
| github.com/joho/godotenv | .env loading |
| golang.org/x/crypto | Password hashing |

## Standards

- Verify code compiles and runs before moving to the next task
- Use full absolute paths in PowerShell — relative paths resolve to `C:\Windows\system32`
- Write .go files with no-BOM UTF-8 encoding:

```powershell
$utf8NoBom = New-Object System.Text.UTF8Encoding $false
[System.IO.File]::WriteAllText("C:\Projects\ArcVault2.0\path\to\file.go", $content, $utf8NoBom)
```

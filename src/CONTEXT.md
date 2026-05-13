# Building Workspace

**Last updated:** May 13, 2026

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

## Architecture

```
Agent (agent-config.yaml)
  └─ HTTP POST /api/agents/register
       └─ Coordinator (Gorilla mux)
            ├─ SQLite (agents, jobs, job_runs, tokens)
            ├─ Cron scheduler (robfig/cron)
            └─ WebSocket → Vue 3 dashboard
```

## Key files

- `coordinator/main.go` — CLI entry: `init | start | help`
- `coordinator/cmd/commands.go` — InitCommand(), StartCommand()
- `coordinator/config/` — placeholder (unimplemented)
- `coordinator/db/` — placeholder (unimplemented)
- `coordinator/server/` — placeholder (unimplemented)
- `agent/main.go` — stub only

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

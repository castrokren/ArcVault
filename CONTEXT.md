# ArcVault2.0 -- Project Status
**Last updated:** May 14, 2026

## Current Phase
Phase 4 COMPLETE.

## What works
- `coordinator.exe start` -- single command starts everything:
  - HTTP API (all endpoints)
  - WebSocket hub (real-time events)
  - CORS middleware
  - Offline detector (60s interval, 90s threshold)
  - Cron job scheduler (robfig/cron per job + 60s fallback ticker)
  - Serves Vue dashboard from dashboard/dist at GET /
- `agent.exe` -- registers, heartbeats every 30s, polls + executes jobs, posts results
- 45 tests passing (coordinator/server: 40, agent/runner: 5)
- Production deployment: run coordinator from repo root, dashboard served on same port

## Full API
| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET    | /health | No | Health check |
| GET    | /ws | ?token= | WebSocket |
| GET    | / | No | Dashboard (static) |
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

## Production deployment
```
cd C:\Projects\ArcVault2.0
go run ./coordinator start   # serves API + dashboard on port 443
go run ./agent               # run on each machine to back up
```

## Possible Phase 5 ideas
- goreleaser / cross-platform binary distribution
- Per-agent tokens instead of single admin token
- Email/webhook notifications on job failure
- Dashboard: dark/light theme toggle, pagination, search
- Agent auto-update mechanism

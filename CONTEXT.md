# ArcVault2.0 -- Project Status
**Last updated:** May 14, 2026

## Current Phase
Phase 3 COMPLETE. Planning Phase 4.

## What works
- `coordinator.exe init` -- prompts for port/db path, generates admin token, saves to ~/.arcvault/config.json
- `coordinator.exe start` -- loads config, initializes SQLite, starts HTTP server with CORS
- `agent.exe` -- registers, heartbeats every 30s, polls for jobs, executes robocopy/rsync, posts results
- All coordinator API endpoints tested and working (31 tests passing)
- WebSocket hub -- broadcasts job.updated and job.result events, token via query param
- Vue 3 dashboard -- agents panel, jobs panel (create/delete), history, live WebSocket indicator

## Full API
| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET    | /health | No | Health check |
| GET    | /ws | token=? | WebSocket connection |
| POST   | /api/agents/register | Bearer | Register agent |
| POST   | /api/agents/{id}/heartbeat | Bearer | Agent heartbeat |
| GET    | /api/agents | Bearer | List all agents |
| POST   | /api/jobs | Bearer | Create job |
| GET    | /api/jobs | Bearer | List jobs (?agent_id= filter) |
| GET    | /api/jobs/{id} | Bearer | Get job |
| DELETE | /api/jobs/{id} | Bearer | Delete job |
| PATCH  | /api/jobs/{id}/status | Bearer | Update job status |
| POST   | /api/jobs/{id}/results | Bearer | Store job run result |

## Known gaps / Phase 4 candidates
- No GET /api/jobs/{id}/runs endpoint (History view falls back gracefully)
- No job scheduling (cron) -- robfig/cron already in go.mod
- No multi-agent support testing
- No authentication beyond single admin token
- No build/release pipeline (binaries for distribution)
- Dashboard not built for production (no npm run build + serve)

## Next step
Decide Phase 4 scope -- options:
1. Job scheduling (cron expressions, robfig/cron)
2. GET /api/jobs/{id}/runs + full history view
3. Production build + distribution (goreleaser or manual)
4. Multi-agent testing + agent offline detection

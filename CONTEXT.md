# ArcVault2.0 -- Project Status
**Last updated:** May 14, 2026

## Current Phase
Phase 4 in progress. Tasks 1-3 complete. Production build not yet started.

## What works
- `coordinator.exe init` -- prompts for port/db path, generates admin token
- `coordinator.exe start` -- HTTP server, SQLite, CORS, offline detector, cron scheduler
- `agent.exe` -- registers, heartbeats, polls jobs, executes robocopy/rsync, posts results
- 45 tests passing across coordinator/server and agent/runner
- Vue 3 dashboard -- agents, jobs, history (real /runs data), live WebSocket

## Full API
| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET    | /health | No | Health check |
| GET    | /ws | ?token= | WebSocket |
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

## Background services (start on coordinator start)
- Offline detector: checks every 60s, marks offline after 90s no heartbeat
- Cron scheduler: robfig/cron per job schedule + 60s fallback ticker

## Open items
- Production build not yet done (Task 4)

## Next step
Task 4 -- Production build
- `npm run build` in dashboard/
- Coordinator serves dashboard/dist as static files
- Single binary + dist folder deployment

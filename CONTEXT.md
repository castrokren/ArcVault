# ArcVault2.0 -- Project Status
**Last updated:** May 13, 2026

## Current Phase
Phase 3 in progress. Tasks 1-4 complete. Vue dashboard not yet started.

## What works
- `coordinator.exe init` -- prompts for port/db path, generates admin token, saves to ~/.arcvault/config.json
- `coordinator.exe start` -- loads config, initializes SQLite, starts HTTP server
- `agent.exe` -- loads agent-config.yaml, registers with coordinator, sends heartbeat every 30s, polls for jobs, executes robocopy/rsync, posts results
- All coordinator API endpoints tested and working (31 tests passing)
- WebSocket hub -- broadcasts job.updated and job.result events to connected clients

## Full API
| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET    | /health | No | Health check |
| GET    | /ws | Bearer | WebSocket connection |
| POST   | /api/agents/register | Bearer | Register agent |
| POST   | /api/agents/{id}/heartbeat | Bearer | Agent heartbeat |
| GET    | /api/agents | Bearer | List all agents |
| POST   | /api/jobs | Bearer | Create job |
| GET    | /api/jobs | Bearer | List jobs (?agent_id= filter) |
| GET    | /api/jobs/{id} | Bearer | Get job |
| DELETE | /api/jobs/{id} | Bearer | Delete job |
| PATCH  | /api/jobs/{id}/status | Bearer | Update job status |
| POST   | /api/jobs/{id}/results | Bearer | Store job run result |

## Open items
- No Vue dashboard yet (Task 5)

## Next step
Task 5 -- Vue 3 + Vite dashboard (dashboard/)
- Agent status panel (online/offline, last seen)
- Job list (status, agent, source/dest paths)
- Job history (run results, exit codes, output)
- Real-time updates via WebSocket

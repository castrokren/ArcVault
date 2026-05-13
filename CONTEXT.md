# ArcVault2.0 -- Project Status
**Last updated:** May 13, 2026

## Current Phase
Phase 3 in progress. Coordinator complete. Agent runner not yet started.

## What works
- `coordinator.exe init` -- prompts for port/db path, generates admin token, saves to ~/.arcvault/config.json
- `coordinator.exe start` -- loads config, initializes SQLite, starts HTTP server
- `agent.exe` -- loads agent-config.yaml, registers with coordinator, sends heartbeat every 30s
- All Phase 2 API endpoints tested and working (health, register, heartbeat, list agents)
- All Phase 3 coordinator endpoints tested and working (21 tests passing):
  - POST   /api/jobs
  - GET    /api/jobs (supports ?agent_id= filter)
  - GET    /api/jobs/{id}
  - DELETE /api/jobs/{id}
  - PATCH  /api/jobs/{id}/status
  - POST   /api/jobs/{id}/results
- jobs table has: id, agent_id, name, source_path, dest_path, schedule, status, created_at
- job_runs table has: id, job_id, started_at, finished_at, exit_code, output

## Open items
- No agent job runner yet (agent/runner/runner.go)
- No Vue dashboard
- No WebSocket

## Next step
Task 2 -- Agent job runner (agent/runner/runner.go)
- Poll GET /api/jobs?agent_id={id}&status=pending
- Claim job via PATCH /api/jobs/{id}/status -> running
- Execute robocopy (Windows) or rsync (Unix)
- Post result via POST /api/jobs/{id}/results

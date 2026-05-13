# ArcVault2.0 -- Project Status
**Last updated:** May 13, 2026

## Current Phase
Phase 2 complete. Phase 3 not yet started.

## What works
- `coordinator.exe init` -- prompts for port/db path, generates admin token, saves to ~/.arcvault/config.json
- `coordinator.exe start` -- loads config, initializes SQLite, starts HTTP server
- `agent.exe` -- loads agent-config.yaml, registers with coordinator, sends heartbeat every 30s
- All API endpoints tested and working (health, register, heartbeat, list agents)

## Open items
- No job scheduling yet
- No job runner on agent
- No Vue dashboard
- No WebSocket

## Next step
Phase 3 -- start with job CRUD endpoints (POST/GET/DELETE /api/jobs)
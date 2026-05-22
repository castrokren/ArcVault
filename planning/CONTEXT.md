# Planning Workspace

**Last updated:** May 22, 2026

## What happens here

Deciding what to build next in ArcVault. Breaking phases into ordered tasks. Tracking what's done and what's blocked.

## Completed phases

| Phase | Summary |
|-------|---------|
| 1 | Project scaffold, CLI skeleton |
| 2 | Config, DB schema, HTTP server, heartbeat, scheduler, WebSocket, Vue dashboard |
| 3 | Per-agent tokens, agent runner, job execution |
| 4 | Job results, run history, offline detector |
| 5 | Coordinator self-update (download → verify → stage → restart) |
| 6 | Service installation (Windows/Linux/macOS) |
| 7 | Dashboard improvements: theme toggle, search/filter |
| 8 | Agent self-update via WebSocket command channel |
| 9 | Rollback: store previous agent binary, one-click rollback from dashboard |
| 10 | Pagination: cursor-based pagination for agents and job history |
| 11 | Scheduled jobs: cron-based automatic job triggers |
| 12 | Notifications: email/webhook on job failure or agent offline |
| 13 | Scheduled backup templates: cron-based job automation |
| 14 | Agent update system & bidirectional rollback |
| 15 (backend) | RBAC infrastructure: JWT authentication, user management, agent groups |
| 15 (frontend) | RBAC UI: Login, password change, user mgmt, group mgmt, smart job dispatch |
| 16 | Federation HA: events log, state sync (root→spoke), health monitoring, agent failover |

## Current phase

**Phase 16 — Federation HA & State Consistency (Complete - v0.9.0)** ✅
- Append-only federation_events log with per-coordinator monotonic sequence ✅
- State sync endpoints: GET /api/federation/sync, POST /api/federation/sync/ack ✅
- Health monitoring: GET /api/federation/health with status, lag, agent count ✅
- Agent failover: coordinator list with round-robin + exponential backoff ✅
- Frontend: FederationHealth.vue with auto-refresh, OKLCH colors, lag indicators ✅
- Tests: 8 test cases, all passing ✅
- Branch pushed, v0.9.0 tag created ✅

## Candidate next phases

- **Phase 17 — Enhanced Federation:** Spoke auto-resync integration, heartbeat timeout detection, agent homing persistence
- **Phase 18 — API Documentation:** OpenAPI/Swagger spec generation from routes
- **Phase 19 — Audit logging:** Track user actions and changes
- **Phase 20 — Advanced backends:** S3, Azure Blob, additional sync targets

## Process

1. Check root CONTEXT.md for current phase and open items
2. Pick the next candidate phase, write a design doc in `files/`
3. Break it into concrete tasks before writing any code
4. Switch to the Building workspace to implement

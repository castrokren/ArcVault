# Planning Workspace

**Last updated:** May 16, 2026

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

## Candidate next phases

- **Phase 9 — Rollback:** store previous agent binary, allow one-click rollback from dashboard
- **Phase 10 — Pagination:** cursor-based pagination for agents and job history lists
- **Phase 11 — Scheduled jobs:** cron-based automatic job triggers (already have robfig/cron)
- **Phase 12 — Notifications:** email/webhook on job failure or agent offline

## Process

1. Check root CONTEXT.md for current phase and open items
2. Pick the next candidate phase, write a design doc in `files/`
3. Break it into concrete tasks before writing any code
4. Switch to the Building workspace to implement

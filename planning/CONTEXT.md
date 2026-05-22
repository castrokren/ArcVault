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

## Current phase

**Phase 15 (frontend) — RBAC UI Components:** Login, password change, user management, group management
- Backend infrastructure complete (JWT, user management, groups, role-based routes) ✅
- Task 8 complete: Group fan-out job dispatch on POST /api/jobs ✅
- Pending: Vue components (Login.vue, ChangePassword.vue, Users.vue, Groups.vue, AuthGuard.vue)
- Pending: Integration with existing Agents.vue and Jobs.vue

## Candidate next phases

- **Phase 16 — API Documentation:** OpenAPI/Swagger spec generation from routes
- **Phase 17 — Audit logging:** Track user actions and changes
- **Phase 18 — Multi-coordinator federation:** Support secondary coordinators for high availability

## Process

1. Check root CONTEXT.md for current phase and open items
2. Pick the next candidate phase, write a design doc in `files/`
3. Break it into concrete tasks before writing any code
4. Switch to the Building workspace to implement

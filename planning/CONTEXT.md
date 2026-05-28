# Planning Workspace

**Last updated:** May 28, 2026

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
| 16 | Federation HA: events log, state sync, health monitoring, agent failover |
| 17 | Enhanced monitoring & alerting: alert rules engine, Slack/Teams, webhook retry, history tracking |
| 18 (frontend) | Full dashboard design system overhaul — all 21 Vue files, new token system, animated login |

## Current state

**v1.0.2 — Production** (as of May 28, 2026)

- v1.0.0 shipped with Phase 17 complete (111 tests passing)
- Dashboard design overhaul shipped to production (2026-05-27)
- v1.0.1 bugfixes applied (2026-05-28): agent dropdown nil slice, update check plain-text error
- v1.0.2 (2026-05-28): Delete agents — `DELETE /api/agents/{id}`, confirmation modal, 6 tests

## Candidate next phases

| Candidate | Value | Effort |
|-----------|-------|--------|
| CLI tooling (headless ops, scripting) | High — enables automation without dashboard | Medium |
| OpenAPI / Swagger spec | Medium — useful for API consumers | Low |
| Audit logging (user action tracking) | Medium — compliance use cases | Medium |
| Additional sync backends (S3, Azure Blob) | High — core use case expansion | High |
| Advanced reporting / compliance export | Medium | Medium |

## Small improvements queued

- `started_at` column on `job_runs` for accurate notification durations
- Password reset via email
- User search/filter in admin panel
- Email TLS client certificate auth support

## User-requested features — ordered easiest → hardest (2026-05-28)

1. ~~**Delete agents**~~ ✅ Done (2026-05-28)
2. ~~**Schedule builder UI**~~ ✅ Done (2026-05-28) — ScheduleBuilder.vue with Off/Interval/Daily/Weekly/Monthly/Custom modes, live preview, wired into Jobs + Templates
3. **Robocopy/rsync flags** *(Medium)* — add flags column to jobs DB schema, pass flags through API and agent execution, multi-select UI in job form
4. **Cancel scheduled/running backups** *(Medium–Hard)* — canceling a pending job is a status update; canceling a running job requires a kill signal from coordinator → agent over WebSocket
5. **Backup progress indicator** *(Hard)* — agent must emit byte/file-count events mid-execution, coordinator broadcasts via WebSocket, frontend renders live progress

## Process

1. Finish v1.0.1 GitHub release
2. Pick a candidate phase above, write a design doc in `planning/`
3. Break into concrete tasks before writing any code
4. Switch to Building workspace to implement

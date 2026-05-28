---
name: Phase 18 Planning
status: active
created: 2026-05-26
last_updated: 2026-05-28
---

# Phase 18 Planning

## Objective

Decide what to build next for ArcVault after v1.0.1. The project is production-ready. All phases through 18 are complete.

## Constraints

- Maintain single binary deployment model
- No CGO dependencies
- All new features must not regress existing 111 tests
- Favor features that serve the core use case: cross-platform backup orchestration with visibility

## Current Blockers

v1.0.1 exe rebuild + GitHub release upload still pending.

## Recent Decisions

- v1.0.0 released (Phase 17 complete, 2026-05-22)
- Dashboard design system overhauled and shipped to production (2026-05-27)
- v1.0.1 bugfixes applied (2026-05-28): Jobs agent dropdown, update check JSON error response

## Candidate Features

| Feature | Value | Effort | Notes |
|---------|-------|--------|-------|
| CLI tooling | High | Medium | Headless ops, scripting, no dashboard needed |
| OpenAPI / Swagger spec | Medium | Low | API docs from existing routes |
| Audit logging | Medium | Medium | User action tracking, compliance |
| S3 / Azure Blob backends | High | High | Core use case expansion |
| Advanced reporting | Medium | Medium | Compliance export, analytics |

## Small Improvements Queued (can ship in any patch)

- `started_at` on `job_runs` for accurate notification durations
- Password reset via email
- User search/filter in admin panel
- Email TLS client cert auth

## User-Requested Features — ordered easiest → hardest (2026-05-28)

1. ~~**Delete agents**~~ ✅ Done — v1.0.2 (2026-05-28)
2. ~~**Schedule builder UI**~~ ✅ Done — v1.0.3 (2026-05-28) — `ScheduleBuilder.vue` Off/Interval/Daily/Weekly/Monthly/Custom
3. **Robocopy/rsync flags** *(Medium)* — DB schema change (flags column), API passthrough, agent uses flags at run time, multi-select UI — **NEXT UP**
4. **Cancel scheduled/running backups** *(Medium–Hard)* — pending cancel is a status update; running cancel needs coordinator→agent kill signal over WebSocket
5. **Backup progress indicator** *(Hard)* — agent emits progress events mid-run, coordinator broadcasts, frontend renders live

## Success Criteria

- v1.0.1 GitHub release complete
- One Phase 19 candidate chosen
- Design doc written to `planning/`

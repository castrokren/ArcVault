# STATE — User Action Audit Logging

## Goal
Implement full user action audit trail: track every meaningful user-initiated action (who, what, when, IP, success/fail) in a dedicated DB table, with a middleware that auto-logs all API requests and explicit structured logging in every mutation handler.

## Invariants / decisions
- Backend only — no frontend audit page (deferred to follow-up)
- Request audit middleware auto-logs every API call (method, path, user, status, latency) — best-effort, never blocks
- Structured action logging in mutation handlers adds semantic action names (e.g. "user.create", "job.cancel") with resource type/ID
- Both systems write to the same `user_audit_log` table
- Middleware skips `/health`, `/ws`, `/ws/agent`, `/ws/federation` to reduce noise
- All audit writes are non-blocking (best-effort, errors silently dropped)

## Done
- ✅ DB table `user_audit_log` with migration + 4 indexes
- ✅ `db/audit.go` — InsertUserAuditLog + ListUserAuditLogs (filterable, paginated)
- ✅ `AuditQueries` interface in queries.go, included in AllQueries
- ✅ `business/audit.go` — AuditService (LogAction, ListAuditLogs, ClientIP)
- ✅ `server/request_audit.go` — requestAuditMiddleware with statusWriter
- ✅ `server/user_audit.go` — handleListUserAuditLogs handler
- ✅ `server/server.go` — wired auditService, middleware, route
- ✅ 35+ mutation handlers across 9 files wired with explicit audit calls
- ✅ 6 DB tests (insert, list, filter, pagination, empty)
- ✅ 10 business tests (LogAction, filter, pagination, ClientIP)

## Verification results
- ✅ `go build ./coordinator/...` — clean
- ✅ `go vet ./coordinator/...` — clean
- ✅ `go test ./coordinator/db/...` — all pass
- ✅ `go test ./coordinator/business/...` — all pass (51 tests)
- ✅ `go test ./coordinator/server/...` — all pass
- ✅ `go test ./agent/...` — all pass
- ✅ `npx vitest run` — 89/89 pass
- ✅ `npm run build` — 0 errors, 0 warnings

## In-progress
- (none — complete)

## Next
- Frontend AuditLog.vue page with searchable table, filters, pagination
- Add audit log retention/pruning (optional — configurable TTL)
- Export audit logs to CSV/JSON

## Open questions
- Should audit log retention be automatic (cron-based pruning) or manual?
- Should the middleware be configurable (include/exclude paths)?

## File map
- STATE.md — This file (epic tracker)
- `coordinator/db/db.go` — migration
- `coordinator/db/audit.go` — DB queries
- `coordinator/db/queries.go` — AuditQueries interface
- `coordinator/business/audit.go` — AuditService
- `coordinator/server/request_audit.go` — request audit middleware
- `coordinator/server/user_audit.go` — audit list handler
- `coordinator/server/server.go` — wired audit service + middleware + route
- 9 handler files with audit calls (auth, jobs, agents, credentials, templates, groups, federation, update, rollback, alerts)
- `coordinator/db/audit_test.go` — 6 tests
- `coordinator/business/audit_test.go` — 10 tests

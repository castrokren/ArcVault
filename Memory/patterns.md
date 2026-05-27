---
name: ArcVault Patterns
category: memory
priority: medium
last_updated: 2026-05-26
last_accessed: 2026-05-26
stale_after_days: 90
auto_summarize: true
archive_policy: keep
---

# ArcVault Reusable Patterns

Operational logic that has appeared in 2+ phases and is worth treating as a standard approach.

---

## Pattern: New API Endpoint (Go)

Appears in: Phase 12, 15, 16, 17

Standard steps:
1. Define handler function in `coordinator/server/<feature>.go`
2. Register route in `coordinator/server/server.go` (group under `/api/`)
3. Apply auth middleware (admin-only or role-based as needed)
4. Add DB functions in `coordinator/db/<feature>.go` if needed
5. Wire DB migration in `coordinator/db/db.go` migrate()
6. Add API call in `dashboard/src/api.js`
7. Write test in `coordinator/server/<feature>_test.go`
8. Run `go test ./...` — must pass

## Pattern: New Vue View

Appears in: Phase 15, 16, 17

Standard steps:
1. Create `dashboard/src/views/<Feature>.vue`
2. Add route in `dashboard/src/router/index.js`
3. Add nav link in `dashboard/src/App.vue`
4. Wire API call from `dashboard/src/api.js`
5. Apply role guard if admin-only (visible-but-disabled pattern)
6. Support light/dark theme via CSS custom properties

## Pattern: New Notification Channel

Appears in: Phase 12 (webhook, email), Phase 17 (Slack, Teams)

Standard steps:
1. Implement `Notifier` interface in `coordinator/notifications/<channel>.go`
2. Add config struct in `coordinator/config/config.go`
3. Wire into `NewDispatcher()` in `coordinator/notifications/notifier.go`
4. Write tests in `coordinator/notifications/<channel>_test.go`
5. Update `CONTEXT.md` notification config example if user-facing

## Pattern: New DB Table

Appears in: Phase 13, 15, 16, 17

Standard steps:
1. Add table definition to `coordinator/db/db.go` → `migrate()`
2. Add struct + CRUD functions in `coordinator/db/<table>.go`
3. Use `IF NOT EXISTS` — migrations must be safe to re-run
4. Add index on foreign keys and common filter columns
5. Never drop columns — add only

## Pattern: Background Goroutine Service

Appears in: Phase 16 (heartbeat detector), Phase 17 (scheduler tasks)

Standard steps:
1. Implement as a method on `*Server` (or `*Coordinator`)
2. Accept `context.Context` for graceful shutdown
3. Use `time.Ticker` for periodic work
4. Log errors; never panic; never block caller
5. Start with `go s.StartXxx()` in `server.Start()`

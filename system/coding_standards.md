---
name: ArcVault Coding Standards
category: system
priority: critical
last_updated: 2026-05-26
stale_after_days: 180
---

# ArcVault Coding Standards

## Go (Coordinator & Agent)

- **Package names:** lowercase, single word — `config`, `server`, `db`, `runner`
- **File names:** snake_case — `agent_config.go`, `job_results.go`
- **Test files:** `<file>_test.go` adjacent to the file under test
- **Module name:** `arcvault` (single monorepo, one `go.mod`)
- **No CGO:** Use `modernc.org/sqlite` (pure Go SQLite) — never use CGO dependencies
- **Error handling:** Errors logged but never block critical handlers (e.g. notifications never block job result handler)
- **Async work:** Use goroutines; never block HTTP handlers for background operations
- **Version injection:** `-X main.Version={{.Version}}` via ldflags at build time

## Vue 3 (Dashboard)

- **Component names:** PascalCase — `AgentUpdateModal.vue`, `FederationHealth.vue`
- **Composables:** camelCase, prefixed with `use` — `useAuth.js`, `useFederationLag.js`
- **Router:** hash history (`createWebHashHistory`)
- **State management:** composables + `ref`/`reactive` — no Vuex/Pinia
- **Auth:** JWT stored in localStorage (remember-me) or memory (session-only)
- **Theme:** CSS custom properties for full light/dark mode support
- **Embedding:** Dashboard built to `coordinator/static/dist/`, embedded via `//go:embed`

## API Design

- **Routes:** kebab-case, prefixed with `/api/` — `/api/job-runs`, `/api/federation/health`
- **Auth:** Admin token OR agent token accepted by middleware
- **Pagination:** All list endpoints — `?page=` (1-indexed), `?limit=` (default 25, max 100), response `{data, total, page, pages, limit}`
- **WebSocket:** gorilla/websocket v1.5.3

## Database (SQLite)

- **Migrations:** Additive only — never drop columns in production migrations
- **Timestamps:** Unix epoch integers or ISO strings — be consistent per table
- **Indexes:** Add for all foreign keys and common filter/sort columns

## Testing

- Run `go test ./...` to verify all tests pass before marking any task complete
- 110 tests total baseline (108 pass + 2 skip on Windows) — do not regress
- Test files live adjacent to the code they test
- Integration tests preferred over mocks where practical

## PowerShell (Windows)

- Line continuation: backtick `` ` `` not backslash
- Use `go test ./...` not `go test ./... \` across lines

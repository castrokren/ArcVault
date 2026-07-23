# CONTEXT — coordinator

Last updated: 2026-07-22

## What happens here
The hub of ArcVault's hub-and-spoke architecture. A Go HTTPS server (default port 443)
that serves the REST API (`/api/*`), the embedded Vue dashboard (`static/dist`), and a
WebSocket hub for connected agents. Storage is SQLite in WAL mode.

## Process — how work flows
Three layers, strictly one direction:

```
server/   (HTTP handlers, auth, routing)
   ↓
business/ (services: AgentService, JobService, UserService, GroupService, AuditService)
   ↓
db/       (query interfaces, SQLite)
```

Handlers never touch `db/` directly — they call `business/`, which calls `db/`.

## What files live here
- `server/` — HTTP handlers, JWT auth, federation, alert rules, credentials, agent tokens
- `business/` — service layer (DTOs), one file per domain (agents, jobs, users, groups, audit)
- `db/` — SQLite query layer, schema, migrations
- `api/` — API type definitions
- `notifications/` — webhook/email/Slack/Teams notifiers
- `updater/` — self-update from GitHub releases
- `service/` — Windows/macOS/Linux service wrapper (install/start/stop)
- `config/` — coordinator config loading
- `cmd/` — CLI entry subcommands
- `static/` — embedded dashboard build output (`dist`)
- `main.go` — entry point
- `tests/` — integration tests

## Standards / rules specific to this workspace
- **Password complexity:** min 8 chars, uppercase, lowercase, digit, special character —
  enforced in `validatePasswordStrength()` (`server/auth.go`) and mirrored in
  `business/users.go`.
- **Admin UI routes are guarded server-side too** — `adminRoute` middleware requires admin
  role + completed password change for GET/POST/DELETE `/api/users`, PUT `/api/users/{id}/role`.
- **Pagination:** `MaxPage` = 10000, `MaxLimit` = 100, `DefaultLimit` = 25 — enforced on
  every paginated endpoint.
- **Contract-tested docs:** `docs/backend.md` (every route) and `docs/service.md` (service
  names + subcommands) are checked against this code by `internal/docs` (Go tests). Change
  a route or subcommand → update the matching CONTRACT block in the same commit, or the
  pre-commit hook blocks it. See `docs/itworks.md`.
- Go packages here: lowercase, single word. Files: snake_case.

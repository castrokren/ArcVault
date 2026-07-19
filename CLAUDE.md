# CLAUDE.md — ArcVault 2.0

Guidance for Claude Code in this repository. Keep every instruction MD file under 100 lines.

## Architecture

ArcVault is a hub-and-spoke backup orchestrator:

- **Coordinator** (`coordinator/`, Go) — the hub. HTTPS server on 443 serving the REST API (`/api/*`), the embedded Vue dashboard (`static/dist`), and a WebSocket hub for agents. SQLite (WAL) storage. Three layers: `server/` handlers → `business/` services → `db/` query interfaces. Provides JWT RBAC (viewer/operator/admin), cron scheduler with templates and missed-schedule detection, alert rules engine with webhook/email/Slack/Teams notifiers, federation failover between coordinators, user-action audit log, and self-update from GitHub releases.
- **Agent** (`agent/`, Go) — the spoke, a Windows service on each backup host. Registers with a per-agent token, heartbeats, executes robocopy/rsync jobs (`runner/`), captures output, honors cancel commands over WebSocket, self-updates, and fails over across a coordinator list with exponential backoff.
- **Dashboard** (`dashboard/`, Vue 3 + Vite) — built then embedded into the coordinator binary. Token-based design system ("Obsidian Pro"), Zod API contract layer in `src/schemas/`.
- **Installer** (`installer/windows/`, Python/Tkinter + PyInstaller) — one .exe that installs coordinator and/or agent as Windows services.
- **Build** (`scripts/`) — `rebuild-and-restart.ps1` is the only sanctioned build+deploy pipeline. Version flows from the `VERSION` file via ldflags; never hardcode it.

## Naming Conventions

- Go packages: lowercase, single word (`config`, `server`, `db`)
- Go files: snake_case (`agent_config.go`)
- Vue components: PascalCase
- API routes: kebab-case, prefixed with `/api/`

## Hard Rules

- Deploy only via `.\scripts\rebuild-and-restart.ps1` — never hand-build without ldflags.
- Never commit secrets: `config.json`, `*.pem`, `*.key`, `.env` are gitignored and must stay that way.
- Docs live in `docs/` (see [docs/RUNBOOK.md](docs/RUNBOOK.md)).
- **Contract-tested docs (workflow, not optional):** `docs/backend.md`, `docs/frontend.md`,
  `docs/service.md` carry CONTRACT blocks checked against the code by `internal/docs` (Go) and
  `dashboard/src/docs` (vitest). Change a route/view/subcommand → update the matching block in the
  same commit (a failing test prints the corrected block to paste). Run `.\scripts\install-hooks.ps1`
  once per clone; the pre-commit hook then blocks drifting commits. Full procedure: **docs/itworks.md**.

## Security Constraints

- **Password complexity:** min 8 chars, uppercase, lowercase, digit, special character enforced server-side in `validatePasswordStrength()` (`coordinator/server/auth.go`) and business layer (`coordinator/business/users.go`).
- **Admin UI routes** are guarded by router-level role checks (`beforeEach` guard) using `meta: { requiresRole: 'admin' }`. Applied to federation, users, groups, alerts, credentials routes.
- **Pagination:** max page 10000 (`MaxPage`), max limit 100 (`MaxLimit`), default limit 25 (`DefaultLimit`). Enforced on all paginated endpoints.
- **User management** endpoints (GET/POST/DELETE /api/users, PUT /api/users/{id}/role) require `adminRoute` middleware (admin role + password change completed).

# CONTEXT — ArcVault 2.0

Last updated: 2026-07-22

## Current version
**v0.6.0** (single source of truth: `VERSION` file, flows through ldflags via
`scripts/rebuild-and-restart.ps1` — never hardcoded elsewhere).

## What this project produces, and for whom
A self-hosted, cross-platform backup orchestrator with a real-time web dashboard, for teams
that want full visibility into backup jobs across many machines without depending on a SaaS
or cloud vendor. One coordinator binary (with the dashboard embedded) manages any number of
lightweight agents installed on the machines being backed up.

## Architecture
ArcVault is a hub-and-spoke backup orchestrator:

- **Coordinator** (`coordinator/`, Go) — the hub. HTTPS server on 443 serving the REST API
  (`/api/*`), the embedded Vue dashboard (`static/dist`), and a WebSocket hub for agents.
  SQLite (WAL) storage. Three layers: `server/` handlers → `business/` services → `db/`
  query interfaces. Provides JWT RBAC (viewer/operator/admin), cron scheduler with templates
  and missed-schedule detection, alert rules engine with webhook/email/Slack/Teams
  notifiers, federation failover between coordinators, user-action audit log, and
  self-update from GitHub releases.
- **Agent** (`agent/`, Go) — the spoke, a Windows service on each backup host. Registers
  with a per-agent token, heartbeats, executes robocopy/rsync jobs (`runner/`), captures
  output, honors cancel commands over WebSocket, self-updates, and fails over across a
  coordinator list with exponential backoff.
- **Dashboard** (`dashboard/`, Vue 3 + Vite) — built then embedded into the coordinator
  binary. Token-based design system ("Obsidian Pro"), Zod API contract layer in
  `src/schemas/`.
- **Installer** (`installer/windows/`, Python/Tkinter + PyInstaller) — one .exe that
  installs coordinator and/or agent as Windows services.
- **Build** (`scripts/`) — `rebuild-and-restart.ps1` is the only sanctioned build+deploy
  pipeline. Version flows from the `VERSION` file via ldflags; never hardcode it.

Per-workspace detail (files, process, workspace-specific rules) lives in each workspace's
own `CONTEXT.md` — see `coordinator/CONTEXT.md`, `agent/CONTEXT.md`, `dashboard/CONTEXT.md`,
`installer/CONTEXT.md`.

## What "done" looks like for a session
- The change builds and deploys clean via `.\scripts\rebuild-and-restart.ps1` (the only
  sanctioned pipeline — see `docs/RUNBOOK.md`).
- Any touched route/view/subcommand has its CONTRACT block updated in `docs/backend.md`,
  `docs/frontend.md`, or `docs/service.md` in the same commit (see `docs/itworks.md`).
- Relevant tests pass: `go test ./...` and `cd dashboard && npx vitest run`.
- No secrets (`config.json`, `*.pem`, `*.key`, `.env`) got committed.
- `tasks/<phase>/STATE.md` is updated with Done / In-progress / Next / Open questions before
  the session ends.

## Open items / next actions
- Release hygiene tracked in `tasks/release-hygiene/STATE.md` — dashboard version-display
  fixes and agent version-staleness detection landed on `security/hardening-v0.6.0`; awaiting
  browser verification of the stale/update badges and a decision on committing the staged
  changes.
- Security hardening tracked in `tasks/security-hardening/` (see `PLAN-review-fixes.md`).
- This CONTEXT.md restructure itself (see `REFERENCES.md` for naming conventions, root
  CLAUDE.md for the routing table).

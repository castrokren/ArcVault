# CONTEXT — ArcVault 2.0

Last updated: 2026-07-24

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

## Agent enrollment — the one flow to know
New machines are enrolled from the dashboard's **Enroll Agent** button (Agents view), which
downloads `GET /api/admin/bootstrap.ps1`. That mints a `bootstrap:<hostname>` token with a
**1-hour expiry** and embeds it, plus the coordinator cert, in an install script.

The admin token is **not** an agent credential. It is fleet-wide, never expires, and cannot
be revoked for one machine. `acceptMachineToken` (`coordinator/server/server.go`) still
accepts it so pre-2026-07-24 agents keep working, and logs
`[auth] DEPRECATED: <ip> authenticated with the admin token` once per host. **That log is the
migration checklist** — when it goes quiet, the `isAdminToken` branch can be deleted. See
`tasks/security-hardening/STATE.md` Phase 3/4.

## Open items / next actions
- **Not deployed.** Eight commits sit on `security/hardening-v0.6.0` (SQLite WAL fix, version
  staleness, bootstrap enrollment, per-agent installer tokens, TLS-on-first-start, dead
  thumbprint removal, dashboard test/dep cleanup). Run `.\scripts\rebuild-and-restart.ps1`.
- **Needs a human in a browser:** the Enroll Agent form and the stale/update version badges
  have never been clicked against a live coordinator (no credentials on file).
- **Needs a scratch-box install:** the installer's GUI path changed (mint order, cert wait);
  verify a real coordinator+agent install and an agent-only install before cutting a release.
- Release hygiene tracked in `tasks/release-hygiene/STATE.md` — steps 2-5 (merge to main,
  re-point the v0.6.0 tag, delete poison tags incl. `v5.01`, add `publish-release.ps1`) are
  still untouched.
- Security hardening tracked in `tasks/security-hardening/` (see `PLAN-review-fixes.md`).
- Dashboard cleanup awaiting a decision: two unused components (`Sparkline.vue`,
  `orbit/OrbitField.vue`) and a 4× duplicated page header — see `dashboard/CONTEXT.md`.

# CONTEXT — ArcVault 2.0

Last updated: 2026-07-25

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
**1-hour expiry** and embeds it, plus the coordinator cert, in an install script. On
registration the agent **exchanges** that short-lived token for a permanent per-agent one and
rewrites its own `agent-config.yaml`. Verified end to end on the real fleet 2026-07-25.

Three things that will bite you here, all learned the hard way:

1. **`host` must be set in `C:\ArcVault\config.json`.** It is optional and the installer never
   writes it. Without it the script URL falls back to the browser's Host header, and a loopback
   result is now refused with 409 rather than producing a script the target machine cannot use.
2. **The cert must cover the address the script uses.** `cert.pem` is generated once and never
   regenerated (deliberately — agents pin it). If DHCP moves this machine, or you change
   `host`, the SANs go stale and agents fail with `x509: certificate signed by unknown
   authority`. Prefer a hostname over an IP. Regenerating means `coordinator.exe rekey-cert`
   AND hand-copying `cert.pem` to every agent's `coordinator.crt` — `rebuild-and-restart.ps1`
   does **not** refresh those.
3. **The script needs an elevated PowerShell** (`#Requires -RunAsAdministrator`) and an
   `Unblock-File` after download. Double-clicking a `.ps1` opens an editor and does nothing.

The admin token is **not** an agent credential. It is fleet-wide, never expires, and cannot be
revoked for one machine. `acceptMachineToken` (`coordinator/server/server.go`) still accepts it
so older agents keep working, and logs `[auth] DEPRECATED: <ip> authenticated with the admin
token` once per host. **That log is the migration checklist** — when it goes quiet, the
`isAdminToken` branch can be deleted. See `tasks/security-hardening/STATE.md` Phase 3/4.

## Open items / next actions
Branch `security/hardening-v0.6.0` is **deployed** (coordinator.exe 2026-07-25 09:07) and the
fleet is healthy: `DESKTOP-EE77F38` v0.6.0 online, `SRB3FLPC010` v0.5.0 online, and
`SMILOW3FLSP001` offline since 06-11 and not yet re-enrolled.

Ordered next actions live in `tasks/release-hygiene/STATE.md` → **Next**. The two that matter
most:
- **Re-enroll `SMILOW3FLSP001`.** It has no per-agent token row, so it is the last machine that
  might still be on the admin token — and therefore the last thing gating Phase 4.
- **Run `scripts/repair-service-env.ps1`** (elevated). The live service has no `Environment`
  registry value, so `ARCVAULT_JWT_SECRET` / `ARCVAULT_CREDENTIAL_KEY` are never injected. The
  installer wrote them in a shape SCM does not read; details in
  `tasks/security-hardening/STATE.md`.

Still unverified: the dashboard UI by eye (no credentials on file — Enroll Agent and the
version badges have only been driven over the API), and the installer's GUI path since the
token/cert/registry changes. Dashboard cleanup awaiting a decision: two unused components and a
4× duplicated page header — see `dashboard/CONTEXT.md`.

# ArcVault 2.0 Features Inventory

**Purpose**: Single source of truth for user-visible features. Prevents accidental deletion during refactors/redesigns.

## Dashboard Features

### Agent Management (Agents.vue)
- **List agents** — table view with hostname, OS, version, last-seen timestamp
- **Search/filter** — search by hostname or agent ID; filter by status (online/offline/all)
- **Status rail** — left sidebar showing online/offline/offline-long counts
- **Fleet readout band** — fleet status card, online-rate sparkline, version-drift indicator
- **Update agent** — per-agent update button (online agents only), calls POST /api/agents/{id}/update
  - Prerequisite: agent must be online and have available update
  - Opens `AgentUpdateModal` (modal showing 4-step progress: downloading → verifying → staging → restarting)
  - Relays `update_progress` WS events to UI until complete or 60s reconnect timeout
- **Get token** — per-agent token generator button, calls POST /api/agents/{id}/token
  - Opens `AgentTokenModal` (modal showing generated token + copy-to-clipboard button)
  - Required for installing agent on new machine (token goes into installer wizard)
  - Pre-5.0 feature, restored in this session

### Coordinator Management (Update routes)
- **Check for coordinator update** — GET /api/update/check → shows banner if update available
- **Apply coordinator update** — POST /api/update/apply → self-updates from GitHub release
  - Downloads binary + SHA256SUMS, verifies checksum, stops service, swaps binary, restarts
  - Progress relayed via WS to UpdateModal

### Jobs (Jobs.vue)
- **List jobs** — paginated table, search/filter by job name
- **Create/edit job** — wizard form with credential selection, robocopy/rsync options
- **Run job** — manual trigger, shows live job progress in Jobs view
- **Cancel job** — mid-run job cancellation via POST /api/jobs/{id}/cancel

### Admin Features
- **Users management** (Users.vue) — admin-only, CRUD users, assign roles (admin/operator/viewer)
- **Groups management** (Groups.vue) — admin-only, CRUD agent groups for role-based access
- **Credentials** (Credentials.vue) — admin-only, manage encrypted backup credentials (SMB/S3/etc)
- **Templates** (Templates.vue) — admin-only, create reusable job templates
- **Federation** (Federation.vue) — admin-only, configure failover to peer coordinators

## Agent Features

### Registration & Heartbeat
- **Register with coordinator** — agent sends os/arch/version on startup, gets agent token
- **Heartbeat** — agent POSTs to /api/agents/{id}/heartbeat every 30s (default)

### Job Execution (runner/)
- **Run job** — pulls job from coordinator, executes robocopy/rsync, captures output/exit code
- **Cancel job** — agent receives `cancel_command` over WS, stops running job cleanly

### Self-Update (updater/)
- **Download new binary** — pulls agent.exe from GitHub release (URL supplied by coordinator)
- **Verify checksum** — verifies SHA256 against SHA256SUMS
- **Swap binary** — backs up current agent.exe → agent.previous, renames agent.new → agent.exe
- **Restart service** — Windows service stop/start (or launchctl/systemctl on macOS/Linux)
- **Stream progress** — emits `update_progress` events back to coordinator WS every step

### Rollback
- **Check rollback available** — checks if agent.previous backup exists
- **Rollback to previous version** — restores agent.previous → agent.exe, restarts service
- **Stream rollback progress** — emits `rollback_progress` events (parallel to update flow)

## Missing/TODO Features

- **Agent auto-update** — agents currently never auto-update; operator must trigger per-agent
- **Token revocation** — no UI to revoke generated tokens
- **Token expiry** — tokens are permanent (no TTL)
- **Staged rollout** — no staged update deployment (all-or-nothing per agent)
- **Per-tenant settings** — no per-tenant version pinning or feature toggles

## Known Issues / Regressions

- `GET /api/admin/token` removed (security hardening phase 2, Jul 8) — admin token never exposed in browser
- Agent WS auth mismatch (security hardening phase 0 commit 05b5736) — fixed in this session (2c462c2)
- Windows agent self-update race (agent/updater/updater_windows.go) — fixed in this session (2c462c2)

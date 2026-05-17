# ArcVault2.0 -- Project Status
**Last updated:** May 17, 2026

## Current Phase
**Phase 10 complete (backend).** Bidirectional rollback for coordinator and agents with live progress streaming.

## What works
- `coordinator init` / `start` / `create-agent-token <id>` / `check-update` / `install-service` / `uninstall-service`
- `agent` (no args = run) / `install-service` / `uninstall-service`
- Single binary deployment, dashboard embedded
- Per-agent tokens: each agent gets its own token via `coordinator create-agent-token <agent-id>`
- Admin token still works for dashboard and management
- **Self-update system:**
  - `coordinator check-update` — check for newer releases (CLI)
  - `/api/update/check` — cached version info (dashboard)
  - `/api/update/apply` — initiate update with progress streaming (WebSocket)
  - Background poller: checks GitHub releases every 24h
  - Dashboard banner + modal with multi-state UI (confirm → progress → success/error)
  - Atomic update flow: download → verify → stage → restart (Windows/Linux/macOS)
- **Pagination for large lists:**
  - Shared `ParsePagination()` helper and `PaginatedResponse` envelope
  - All list endpoints (`/api/agents`, `/api/jobs`, `/api/jobs/{id}/runs`) accept `?page=`, `?limit=`
  - Search/filter moved to server-side: `?search=`, `?status=` now SQL WHERE clauses
  - New global `/api/job-runs` endpoint for full job run history
  - Reusable `Pagination.vue` component with ellipsis logic ("Showing 1–25 of 284")
  - Dashboard: filter + pagination controls on all three list views
  - WebSocket updates refresh current page without resetting state
- **Rollback system:**
  - `coordinator check-rollback` — check if backup exists (CLI)
  - `/api/rollback-available` — check if rollback is available
  - `/api/rollback` — initiate coordinator rollback with progress streaming
  - `/api/agents/{id}/rollback` — send rollback_command to agent via WebSocket
  - Agent heartbeat reports live `rollback_available` status (disk check, not cached)
  - Backup created before every update (coordinator and agent)
  - Rollback restores previous binary, verifies, and restarts
  - Golden rule preserved: running binary never touched before verification completes
- **94 tests passing** (87 from Phase 9 + 7 new rollback tests)
  - coordinator/server: 65 (includes pagination + updated list endpoint tests)
  - coordinator/updater: 9
  - agent/runner: 5
  - agent/updater: 8 (2 skip on Windows, run on Linux/macOS)
+ coordinator/updater/ — platform-agnostic download/verify/stage
+ coordinator/updater/{windows,linux,darwin}.go — service control
+ coordinator/server/update.go — REST endpoints + progress streaming
+ dashboard/src/components/{UpdateBanner,UpdateModal}.vue

## Phase 8 additions
+ coordinator/db — arch column in agents table (idempotent ALTER TABLE migration)
+ coordinator/server/hub.go — agentConns map, SendToAgent(), handleAgentWS() (/ws/agent)
+ coordinator/server/agent_update.go — POST /api/agents/{id}/update (admin only)
+ coordinator/updater — FetchLatestRelease(), ResolveAgentAssetURL(), ReleaseAsset type
+ agent/updater/ — HandleUpdateCommand, download/verify/stage + platform ApplyUpdate
+ agent/ws/ws.go — persistent WS client (auto-reconnect), dispatches update_command
+ agent/heartbeat — arch field added to Register()
+ agent/main.go — passes runtime.GOARCH, starts WS client goroutine
+ dashboard/src/components/AgentUpdateModal.vue — confirm→progress→reconnecting→success/error
+ dashboard/src/views/Agents.vue — "Update" badge + per-agent update button

## Phase 9 additions
+ coordinator/server/pagination.go — PaginationParams, PaginatedResponse, ParsePagination, NewPaginatedResponse
+ coordinator/server/pagination_test.go — 10 tests (defaults, clamping, offset, endpoints)
+ coordinator/server/agents.go — server-side search/filter + pagination on handleListAgents
+ coordinator/server/jobs.go — server-side search/filter + pagination on handleListJobs
+ coordinator/server/job_runs.go — pagination on handleGetJobRuns + new handleListAllJobRuns
+ coordinator/server/server.go — register GET /api/job-runs route
+ dashboard/src/components/Pagination.vue — reusable page control with ellipsis ("Showing X–Y of Z")
+ dashboard/src/api.js — buildQuery(), updated getAgents/getJobs + new getJobRuns
+ dashboard/src/views/Agents.vue — server-side filtering + pagination controls
+ dashboard/src/views/Jobs.vue — server-side filtering + pagination controls
+ dashboard/src/views/History.vue — single /api/job-runs call instead of N+1 per-job fetch

## Phase 10 additions (backend complete)
+ coordinator/updater/updater.go — BackupCurrent(), Rollback(), IsRollbackAvailable() + getBackupDir()
+ coordinator/updater/updater_test.go — 4 tests (backup, availability, rollback errors)
+ coordinator/db/db.go — rollback_available column (idempotent ALTER TABLE)
+ coordinator/server/agents.go — heartbeatRequest struct, rollback_available field in agentResponse
+ coordinator/server/rollback.go — handleRollbackAvailable(), handleRollback(), handleAgentRollback()
+ coordinator/server/server.go — register 3 rollback endpoints (GET /api/rollback-available, POST /api/rollback, POST /api/agents/{id}/rollback)
+ agent/heartbeat/heartbeat.go — send rollback_available in heartbeat payload, isRollbackAvailable() disk check
+ agent/updater/updater.go — BackupCurrent(), Rollback(), IsRollbackAvailable() + getBackupDir()
+ agent/updater/updater_test.go — 3 tests (backup, availability, rollback errors)
+ agent/ws/ws.go — handleRollbackCommand() dispatcher, getAgentBinaryPath()

## Per-agent token workflow
```
# Generate token for an agent
coordinator create-agent-token agent-01
# → prints token, add to agent-config.yaml as auth_token

# Agent authenticates with its own token
# Admin token stays private to dashboard/operator
```

## Service installation
| Platform | Install command | Start command |
|----------|----------------|---------------|
| Windows (admin) | coordinator install-service | sc start arcvault-coordinator |
| Linux (root) | sudo coordinator install-service | sudo systemctl start arcvault-coordinator |
| macOS (root) | sudo coordinator install-service | sudo launchctl start com.arcvault.coordinator |
| Same for agent | agent install-service | (platform equivalent) |

## Phase 7 (complete)
**Dashboard improvements: COMPLETE**
- Theme toggle (dark/light mode with localStorage persistence)
- Search and filter for Agents view (search by ID/hostname, status filter)
- Search and filter for Jobs view (search by name/agent_id, status filter)
- Light mode styling with proper color scheme
- All 65 tests passing

## Phase 8 (complete)
**Agent self-update: COMPLETE**
- Operator clicks Update on an agent card → coordinator sends update_command via WebSocket
- Agent downloads, verifies, stages, and restarts with new binary
- Live progress streamed back through hub to dashboard modal
- Update flow: confirm → progress → reconnecting → success/error
- Platform service control: Windows (net stop/start), Linux (systemctl), macOS (launchctl)
- Golden rule enforced: binary never modified before staging completes
- arch field added to agent registration and stored in DB
- All 77 tests passing

## Phase 10 (backend complete, frontend pending)
**Rollback system: BACKEND COMPLETE**
- One-version-back rollback for coordinator and agents
- Backup created on every update, restored on rollback
- Live disk check for rollback availability (no caching)
- **Pending:** RollbackModal.vue, Rollback button in Agents view, api.js helpers

## Future work (not started)
Possible next phases:
- Job execution history visualization (timeline, per-agent run charts)
- Scheduled backups (cron-based job triggers with templates)
- Multi-coordinator federation
- Agent groups and role-based permissions
- Failure notification webhooks/email

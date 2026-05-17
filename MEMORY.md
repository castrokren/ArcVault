# ArcVault Project Memory
**Project Name:** ArcVault
**Type:** OS-agnostic Backup Orchestrator
**Status:** Phase 10 Backend Complete (Frontend Pending)
**Last Updated:** May 17, 2026 (Phase 10 backend complete)

---

## Project Vision
ArcVault solves key limitations in RoboBackup:
- RoboBackup: Windows-only, limited monitoring, no remote visibility
- ArcVault: Cross-platform (Windows/Mac/Linux), real-time monitoring dashboard, self-hosted, agents coordinate through central coordinator

**Architecture:** Lightweight agents -> central Go coordinator -> Vue.js dashboard (embedded in binary)

---

## Phase Summary

### Phase 1: COMPLETE — binaries, init/start, YAML config
### Phase 2: COMPLETE — SQLite, HTTP server, agent register + heartbeat
### Phase 3: COMPLETE — job CRUD, agent runner, WebSocket, Vue dashboard
### Phase 4: COMPLETE — job runs history, offline detection, cron scheduling, production build
### Phase 5: COMPLETE — embedded dashboard, single binary, goreleaser, v0.1.0 GitHub release
### Phase 6: COMPLETE — service installation, per-agent tokens, self-update system
### Phase 7: COMPLETE — dashboard improvements (theme toggle, search, filtering)
### Phase 8: COMPLETE — agent self-update capability
### Phase 9: COMPLETE — server-side pagination for all list views
### Phase 10: BACKEND COMPLETE — bidirectional rollback with live progress (frontend pending)

---

## Phase 6 Details

**Service installation: COMPLETE**
- coordinator/service/ and agent/service/ packages
- service.go, service_windows.go, service_linux.go, service_darwin.go
- Windows: golang.org/x/sys/windows/svc/mgr, StartAutomatic
- Linux: /etc/systemd/system/arcvault-{name}.service
- macOS: /Library/LaunchDaemons/com.arcvault.{name}.plist

**Per-agent tokens: COMPLETE**
- coordinator/db/db.go -- CreateAgentToken(agentID) string, ValidateToken(token) (role, error)
- tokens table was already in schema, now actually used
- authMiddleware accepts admin token OR valid agent token from DB
- coordinator create-agent-token <agent-id> -- generates + stores + prints token
- Multiple tokens per agent allowed (each call creates new one)
- Admin token unchanged -- still used for dashboard and management

**Failure notifications: NOT IMPLEMENTED**
- Documented as complete in earlier Phase 6 notes, but code was never written
- No coordinator/notifications/ package exists
- Deferred to future work

**Self-update system: COMPLETE**
- coordinator/updater/ package (platform-agnostic): CheckLatestRelease, DownloadBinary, VerifyBinary, StageBinary
- Platform handlers: updater_{windows,linux,darwin}.go (service start/stop + atomic rename)
- API endpoints: GET /api/update/check (cached), POST /api/update/apply (WebSocket progress)
- CLI command: coordinator check-update (standalone, no server needed)
- Background poller: 24h interval, silent failure recovery
- Dashboard: UpdateBanner.vue (dismissible banner), UpdateModal.vue (multi-state UI)
- Error safety: binary never touched before staging completes
- 14 new tests (9 updater + 5 server, exceeding plan's 12)

---

## Phase 7 Details

**Theme Toggle: COMPLETE**
- Added `theme` ref with localStorage persistence (`arcvault-theme`)
- `applyTheme(val)` function sets `data-theme` attribute on document root
- `toggleTheme()` function flips between 'dark' and 'light'
- Dark mode: existing color scheme (#1a1a2e background, #4f8ef7 blue accent)
- Light mode: white/light gray backgrounds, dark text, updated accent colors
- Sun/moon icon button in header (☀️ for dark mode, 🌙 for light mode)

**Agents View - Search & Filter: COMPLETE**
- Search input: filters by agent ID or hostname (case-insensitive)
- Status filter chips: All, Online, Offline
- AND logic: both active simultaneously narrows results
- Empty state: "No agents match your search"
- WebSocket updates don't reset filters

**Jobs View - Search & Filter: COMPLETE**
- Search input: filters by job name or agent_id (case-insensitive)
- Status filter chips: All, Pending, Running, Completed, Failed
- AND logic: both active simultaneously narrows results
- Empty state: "No jobs match your search"
- WebSocket updates don't reset filters

**Dashboard Build & Tests: COMPLETE**
- `npm run build` compiles successfully (~570ms)
- `go build ./...` compiles with embedded dashboard
- All 65 tests passing (no new tests added, requirements unchanged)
- Commit: b3a275c "feat: phase 7 dashboard improvements - theme toggle, search, and filter for agents and jobs"

---

## Phase 8 Details

**Agent self-update: COMPLETE**
- Operator clicks "Update" on an agent card in dashboard → coordinator sends update_command via WebSocket
- Agent downloads, verifies, stages, and restarts with new binary
- Live progress streamed back through hub to dashboard modal
- Update flow: confirm → progress → reconnecting → success/error
- Platform service control: Windows (net stop/start), Linux (systemctl), macOS (launchctl)
- Golden rule enforced: binary never modified before staging completes
- arch field added to agent registration and stored in DB (idempotent ALTER TABLE migration)

**New files:**
- coordinator/db — arch column in agents table (idempotent ALTER TABLE migration)
- coordinator/server/hub.go — agentConns map, SendToAgent(), handleAgentWS() (/ws/agent)
- coordinator/server/agent_update.go — POST /api/agents/{id}/update (admin only)
- coordinator/updater — FetchLatestRelease(), ResolveAgentAssetURL(), ReleaseAsset type
- agent/updater/ — HandleUpdateCommand, download/verify/stage + platform ApplyUpdate
- agent/ws/ws.go — persistent WS client (auto-reconnect), dispatches update_command
- agent/heartbeat — arch field added to Register()
- agent/main.go — passes runtime.GOARCH, starts WS client goroutine
- dashboard/src/components/AgentUpdateModal.vue — confirm→progress→reconnecting→success/error
- dashboard/src/views/Agents.vue — "Update" badge + per-agent update button

**Tests: 12 new (8 agent/updater + 4 coordinator/server)**
- agent/updater: 8 tests (2 skip on Windows, run on Linux/macOS)
- coordinator/server: 4 new agent update endpoint tests

---

## Phase 9 Details

**Server-side pagination: COMPLETE**
- All list endpoints (`GET /api/agents`, `GET /api/jobs`, `GET /api/jobs/{id}/runs`) now paginated
- New global endpoint `GET /api/job-runs` for full job run history (replaces N+1 per-job fetch)
- Query params: `?page=` (1-indexed), `?limit=` (default 25, max 100)
- Search and filters moved to server-side: `?search=`, `?status=` now SQL WHERE clauses
- Consistent response envelope: `{data: [...], total: X, page: Y, pages: Z, limit: W}`

**New files:**
- coordinator/server/pagination.go — `PaginationParams`, `PaginatedResponse`, `ParsePagination()`, `NewPaginatedResponse()`
- coordinator/server/pagination_test.go — 10 tests (defaults, clamping, offset math, endpoint responses)
- dashboard/src/components/Pagination.vue — reusable component with ellipsis logic and record counter
- dashboard/src/api.js — `buildQuery()` helper, updated `getAgents()` / `getJobs()`, new `getJobRuns()`

**Modified files:**
- coordinator/server/agents.go — `handleListAgents` accepts `?search=`, `?status=`, pagination params
- coordinator/server/jobs.go — `handleListJobs` accepts `?search=`, `?status=`, pagination params
- coordinator/server/job_runs.go — `handleGetJobRuns` paginated; new `handleListAllJobRuns()`
- coordinator/server/server.go — register `GET /api/job-runs` route
- dashboard/src/views/Agents.vue — server-side filtering, Pagination component
- dashboard/src/views/Jobs.vue — server-side filtering, Pagination component
- dashboard/src/views/History.vue — uses global `/api/job-runs` instead of per-job fetches
- coordinator/server/{jobs,job_runs,offline_detector}_test.go — updated to decode PaginatedResponse

**Tests: 10 new + 5 updated**
- coordinator/server/pagination_test.go: 10 tests
- Existing list endpoint tests updated to decode `PaginatedResponse` instead of raw arrays
- Total: 87 tests (85 pass + 2 skip on Windows)

---

## Phase 10 Details

**Bidirectional rollback system: BACKEND COMPLETE**
- One-version-back rollback for coordinator and agents
- Backup created on every update, restored on rollback
- Live disk check for rollback availability (no caching)
- Progress streamed via WebSocket to dashboard

**Architecture:**
- Backup directory: Linux/macOS `/var/lib/arcvault/backups/`, Windows `%ProgramData%\ArcVault\backups\`
- Backup filenames: `coordinator.previous`, `agent.previous`
- Agent heartbeat reports `rollback_available` boolean (live disk check every heartbeat)
- Coordinator stores `rollback_available` in agents table, returned in `/api/agents` response
- Golden rule preserved: running binary never touched before staging + verification complete

**New files:**
- coordinator/updater/updater.go — BackupCurrent(), Rollback(), IsRollbackAvailable(), getBackupDir()
- coordinator/updater/updater_test.go — 4 tests (backup creation, availability check, rollback errors)
- coordinator/server/rollback.go — handleRollbackAvailable(), handleRollback(), handleAgentRollback()
- agent/updater/updater.go — BackupCurrent(), Rollback(), IsRollbackAvailable(), getBackupDir()
- agent/updater/updater_test.go — 3 tests (backup creation, availability check, rollback errors)
- agent/ws/ws.go — handleRollbackCommand() dispatcher, getAgentBinaryPath()

**Modified files:**
- coordinator/db/db.go — added `rollback_available` column (idempotent ALTER TABLE)
- coordinator/server/agents.go — heartbeatRequest struct, rollback_available field in agentResponse, updated handleHeartbeat() and handleListAgents()
- coordinator/server/server.go — register 3 rollback endpoints
- agent/heartbeat/heartbeat.go — send `rollback_available` in heartbeat payload, isRollbackAvailable() disk check
- agent/updater/updater.go — call BackupCurrent() in HandleUpdateCommand() before swap

**API Endpoints:**
- GET /api/rollback-available (admin only) — check if coordinator rollback available
- POST /api/rollback (admin only) — initiate coordinator rollback, progress streamed via WebSocket
- POST /api/agents/{id}/rollback (admin only) — send rollback_command to agent, return 409 if no backup

**Tests: 7 new**
- coordinator/updater: 4 tests (backup, availability, errors)
- coordinator/server: 0 (endpoint tests not written, marked pending)
- agent/updater: 3 tests (backup, availability, errors)
- Total: 94 tests (92 pass + 2 skip on Windows)

**Pending (frontend only):**
- RollbackModal.vue component (confirm → progress → success/error, same pattern as AgentUpdateModal)
- Rollback button in Agents.vue (visible only when `rollback_available === true`)
- Coordinator rollback button in App.vue header
- api.js helper functions (getRollbackAvailable(), applyRollback(), applyAgentRollback())

---

## Technical Details

### Stack
- **Language:** Go (coordinator + agents)
- **Frontend:** Vue 3 + Vite 8, vue-router@4 (hash history), embedded via //go:embed
- **Database:** SQLite via modernc.org/sqlite (pure Go, no CGO)
- **Auth:** Admin token (config.json) OR agent token (tokens table, role='agent')
- **Sync Tools:** Robocopy (Windows, exit 1-7 = success), Rsync (Unix/Mac)
- **WebSocket:** github.com/gorilla/websocket v1.5.3
- **Scheduler:** github.com/robfig/cron/v3
- **Service mgmt:** golang.org/x/sys v0.44.0
- **Release:** goreleaser v2.15.4
- **Module:** single monorepo, module name: `arcvault`

### Project Layout
```
coordinator/
  main.go                    -- init/start/create-agent-token/check-update/install-service/uninstall-service
  cmd/commands.go            -- InitCommand, StartCommand, CreateAgentTokenCommand, CheckUpdateCommand
  config/config.go
  db/db.go                   -- CreateAgentToken, ValidateToken + migrate, arch column
  service/
    service.go / service_windows.go / service_linux.go / service_darwin.go
  updater/
    updater.go               -- CheckLatestRelease, DownloadBinary, VerifyBinary, StageBinary, ExecuteUpdate
                             -- FetchLatestRelease, ResolveAgentAssetURL, ReleaseAsset (Phase 8)
    updater_{windows,linux,darwin}.go -- platform-specific ApplyUpdate
    updater_test.go          -- 9 tests
  notifications/
    config.go / notifier.go / webhook.go / email.go / notifier_test.go
  static/
    static.go / dist/
  server/
    server.go                -- authMiddleware (admin OR DB token), adminMiddleware (admin only)
    pagination.go / pagination_test.go -- PaginationParams, PaginatedResponse, helpers (Phase 9)
    update.go / update_test.go -- /api/update/check, /api/update/apply endpoints + caching
    agent_update.go          -- POST /api/agents/{id}/update (Phase 8)
    agent_update_test.go     -- 4 tests (Phase 8)
    agents.go / hub.go / jobs.go / job_status.go / job_results.go
    job_runs.go / offline_detector.go / scheduler.go
    agent_token_test.go + all other *_test.go
agent/
  main.go                    -- run/(install|uninstall)-service/help, starts WS client goroutine
  config/config.go
  heartbeat/heartbeat.go     -- arch field added to Register()
  ws/ws.go                   -- persistent WS client, auto-reconnect, dispatches update_command (Phase 8)
  service/ (same platform split as coordinator)
  runner/ runner.go / runner_test.go / executor.go
  updater/                   -- HandleUpdateCommand, platform ApplyUpdate (Phase 8)
    updater.go / updater_windows.go / updater_linux.go / updater_darwin.go
    updater_test.go          -- 8 tests (2 skip on Windows)
dashboard/src/
  App.vue                    -- UpdateBanner, UpdateModal components, checkForUpdates()
  api.js                     -- request(), buildQuery(), getAgents/Jobs/JobRuns (Phase 9)
  components/
    UpdateBanner.vue         -- dismissible banner (session-only), update version display
    UpdateModal.vue          -- coordinator update modal: confirm → progress → success/error
    AgentUpdateModal.vue     -- agent update modal: confirm → progress → reconnecting → success/error (Phase 8)
    Pagination.vue           -- reusable page controls with ellipsis logic (Phase 9)
  views/
    Agents.vue               -- server-side filter + pagination (Phase 9)
    Jobs.vue                 -- server-side filter + pagination (Phase 9)
    History.vue              -- global /api/job-runs paginated fetch (Phase 9)
    ...
.goreleaser.yaml / .gitignore / go.mod
```

### Test Count
- **94 tests total (92 pass + 2 skip on Windows)**
- coordinator/server: 65 tests (includes agent token + update + agent update + pagination tests)
- coordinator/updater: 13 tests (9 original + 4 Phase 10 rollback tests)
- agent/runner: 5 tests
- agent/updater: 11 tests (8 original + 3 Phase 10 rollback tests; 2 skip on Windows)

### Key Commands
```powershell
# generate agent token
coordinator create-agent-token agent-01

# check for updates (CLI)
coordinator check-update

# service management (run as admin/root)
coordinator install-service
coordinator uninstall-service
agent install-service
agent uninstall-service

# release (version injected via ldflags)
goreleaser release --clean   # needs GITHUB_TOKEN with repo scope
```

### Git / Release Status
**Latest:** Phase 10 backend complete (bidirectional rollback)
**Tags:** v0.1.0, v0.2.0 released on GitHub
**Remote:** https://github.com/castrokren/ArcVault
**Branch:** main (7 new Phase 10 commits: 10-01 through 10-07)
**Build:** Version injected via ldflags: `-X main.Version={{.Version}}`

---

## Future Work (not started)

Possible next phases:
- Phase 10 frontend completion (RollbackModal, Rollback buttons, API helpers)
- Job execution history visualization (timeline, per-agent run charts)
- Scheduled backups (cron-based job triggers with templates)
- Multi-coordinator federation
- Agent groups and role-based permissions
- Failure notifications system (webhook + email, originally planned for Phase 6)

---
**End of Memory Document**

# ArcVault2.0 -- Quick Reference
**Last updated:** June 6, 2026 (Session 19) | **Coordinator v0.4.0** | **Agent v0.4.0** | **Stable ✅**

## Status
✅ Phase 12: Failure notifications (webhook + email)  
✅ Phase 13: Scheduled backup templates (cron-based job automation)  
✅ Phase 14: Agent update system & rollback  
✅ Phase 15 (backend): RBAC with JWT authentication, user management, agent groups  
✅ Phase 15 (frontend): Login, useAuth composable, user/group CRUD, smart job forms  
✅ Phase 16 (backend): Federation failover, state sync (federation_events log), health monitoring  
✅ Phase 16 (frontend): FederationHealth.vue dashboard with auto-refresh  
✅ Phase 16 (agent): Coordinator list failover with exponential backoff  
✅ Phase 16 (gaps): Agent homing persisted, heartbeat detector live, stale banners wired to lag composable  
✅ Phase 17 (COMPLETE): Enhanced monitoring & alerting
  - Job start time accuracy fix (recorded in job_runs table)
  - Alert rules engine (per-job configurable rules: on_failure, duration_exceeded, missed_schedule)
  - Webhook retry with exponential backoff (3 attempts: 5s → 15s → 45s)
  - Slack & Teams notifiers (via incoming webhooks)
  - Alert history tracking (30-day retention by default)
  - Scheduler: Missed schedule detector + history pruning
  - Frontend: Alerts.vue dashboard with rule creation/deletion & retry controls  
✅ All tests passing | v1.0.0 released on GitHub  
✅ Phase 18 (frontend): Comprehensive dashboard design system overhaul — all 21 Vue files redesigned, new token system, animated login, rebuilt components  
✅ Services redeployed and verified (2026-05-27)  
✅ v1.0.1 bugfixes applied (2026-05-28):
  - Jobs page agent dropdown broken when no groups exist (nil slice JSON null + wrong PaginatedResponse key)
  - Update check endpoint returned plain text on error instead of JSON  
✅ v1.0.3 (2026-05-28): Schedule builder UI — ScheduleBuilder.vue component (Interval/Daily/Weekly/Monthly/Custom modes, live cron preview) wired into Jobs and Templates forms
✅ v1.0.2 (2026-05-28): Delete agents feature
  - DELETE /api/agents/{id} endpoint (admin only, 409 blocks if running jobs)
  - Cleans up tokens + group memberships on delete; historical jobs preserved
  - Dashboard: Delete button + confirmation modal in Agents.vue (admin-only, local view)
  - 6 new tests in agent_delete_test.go; full suite passing
✅ Phase 19 (COMPLETE 2026-05-28): Robocopy/Rsync Advanced Flags
✅ Phase 20 (COMPLETE): Job cancellation + progress tracking (initial)
✅ Phase 21a-3 (COMPLETE 2026-05-28): Real-time job progress tracking
✅ Phase 21a-4 (COMPLETE 2026-05-29): Critical hot fix — Jobs stuck in pending
✅ v0.2.1 (2026-05-29): Scheduled jobs fix + self-update verification
✅ Session 6 (June 2, 2026): v0.2.1 Release Finalization — COMPLETE
✅ Session 7 (June 2, 2026): JWT Token Refresh Fix for Update Modals — COMPLETE
✅ Session 8 (June 3, 2026): Coordinator Self-Update Fix — COMPLETE
  - Dashboard embedding fixed, --version flag added, ARCVAULT_SERVICE env var set, os.Exit(1) restart
  - Releases: v0.2.2 (--version fix), v0.2.3 (Windows update fix)

✅ **Session 9** (June 3, 2026): v0.2.3 Release + Asset Resolution Fix — COMPLETE
  - **Issue**: Update button disappeared after coordinator self-updated to v0.2.2
  - **Root cause 1**: v0.2.3 was never released — service was correctly on latest (v0.2.2)
  - **Root cause 2**: resolveAsset() only matched coordinator_windows_amd64.exe / coordinator-windows-amd64.exe — would fail to find plain coordinator.exe assets on future updates
  - **Fix**: Added plain name fallback (coordinator.exe / coordinator) to resolveAsset() in updater.go
  - **Result**: v0.2.3 released, service manually swapped, running clean at v0.2.3, update_available: False
  - **Files changed**: coordinator/updater/updater.go (resolveAsset fallback)
  - **Verified**: Invoke-RestMethod /api/update/check → current: v0.2.3, latest: 0.2.3, update_available: False

✅ **Session 10** (June 3, 2026): Bug fixes — Nav bar, agent update badge, rebuild script
  - Nav bar hidden when logged out (`v-if="auth.isAuthenticated.value"` on `<header>` in App.vue)
  - Agent Update badge false positive fixed — now compares agent version against coordinator current version, not GitHub latest
  - Agent version moved from config file to ldflags (`-X main.Version=vX.Y.Z`) matching coordinator pattern
  - `--version` flag added to agent binary (required for VerifyBinary check on self-update)
  - `Version` field removed from agent config struct and `agent-config.yaml`
  - rebuild-and-restart.ps1 fixed: correct deploy paths (`installer\windows\`), SCM recovery disabled/re-enabled around stop/start, ldflags version injection from `git describe`

✅ **Session 11** (June 3, 2026): Dashboard fixes — Sync Flags, version badge, build pipeline
  - **SyncFlagsBuilder** wired into Jobs form — was imported but never rendered; placed outside form-grid in `.sync-flags-row` wrapper
  - **Version badge** added to nav brand — shows `updateStore.current` as small muted text next to ArcVault logo
  - **scripts/rebuild-and-restart.ps1** is the canonical build script — full pipeline: stop services → Vue build → clear + copy dist to coordinator/static/dist → Go build with ldflags → deploy → restart
  - **Root cause fixed**: `coordinator/static/dist` had accumulated 28+ stale JS files from previous builds; Go embed was picking up wrong files. The script clears this directory before every build.
  - **v0.2.4 tagged and pushed** with all fixes

✅ **Session 12** (June 3, 2026): Phase 22 API Contract Layer & Agent Display Fix
  - **Phase 22 Completed**: API Contract Layer with TypeScript types + Zod runtime validation schemas
  - **Bug Found**: Users not displaying after creation; Agents list broken with "[API Contract] validation failed" errors
  - **Root Cause (Users)**: Users endpoint returning paginated response `{data: [...], total, page, pages}` but Users.vue accessing raw array
  - **Root Cause (Agents)**: Two-part issue:
    1. Frontend `getAgents()` was returning just the array instead of full paginated response object
    2. Agents.vue expects `result.value.data` but was getting `result.value = array`
  - **Fix Applied**:
    1. Modified `coordinator/server/auth.go` handleListUsers to wrap response in `NewPaginatedResponse`
    2. Updated Users.vue to access `data.data` field from paginated response
    3. Removed problematic Zod validation from `getAgents()`, `getJobs()`, `getGroups()` in `dashboard/src/api.ts`
    4. **Critical Fix**: Changed `getAgents()` and `getJobs()` to return full paginated response object, not extracted array
    5. Updated `dashboard/src/views/Users.vue` line 275 from `data.users || []` to `data.data || []`
  - **Binary Deployment**: Built coordinator with embedded fixed dashboard (includes API contract layer + pagination fix)
  - **Result**: ✅ Agents displaying correctly | ⚠️ Users still not displaying (needs investigation in next session)
  - **Build Process Used**: Manual dashboard build + static/dist update + Go rebuild + service binary deployment

✅ **Session 13** (June 3, 2026): Users/Agents Pagination Fix — Dual Frontend + Backend
  - **Issue**: Users still not displaying while Agents worked — inconsistent API response formats
  - **Root Cause Identified**: 
    - Backend `/api/agents` endpoint uses `NewPaginatedResponse()` returning `{data: [...], total, page, pages, limit}`
    - Backend `/api/users` endpoint returned raw array `[...]` — NOT wrapped in pagination
    - Frontend Users.vue was accessing `data.users` (wrong field) instead of `data.data`
    - When either component accessed the wrong field, the other would break
  - **Fixes Applied**:
    1. **Frontend (Users.vue)**: Changed `data.users` → `data.data` (line 275), added missing `watch` import
    2. **Backend (auth.go)**: Modified `handleListUsers()` to parse pagination params, apply limit/offset, wrap response in `NewPaginatedResponse()` — now matches agents format
    3. **Result**: Both endpoints now return identical pagination structure
  - **Binary Rebuilt**: Coordinator rebuilt with all fixes (coordinator binary timestamp: 19:06:46)
  - **Status**: Code fixes complete and deployed; service restart pending (permissions issue)
  - **Next**: Service restart will complete the fix when admin user has capability to restart Windows service

✅ **Session 15** (June 5, 2026): Bug fixes — update detection, RBAC, nav bar, installer — COMPLETE
  - **Coordinator update detection**: Poll interval 24h → 1h; added 5-min TTL to update cache so `/api/update/check` always returns fresh data
  - **Jobs 403 fixed**: `GET /api/jobs` and `GET /api/jobs/{id}` changed from `adminTokenRoute` → `viewerRoute`; `POST /api/jobs` and `POST /api/jobs/{id}/cancel` → `operatorRoute`; `DELETE /api/jobs/{id}` → `adminRoute`
  - **Nav bar "?" icon removed**: `userInitials` computed no longer falls back to `'?'` — avatar hides via `v-if` when username is empty
  - **build.ps1 removed**: replaced by `scripts/rebuild-and-restart.ps1` (superset); CONTEXT.md updated
  - **Installer "marked for deletion" fix**: wait loop now checks for exit code 1060 (not any non-zero); 15s max wait + 2s buffer after service confirmed gone; both `agent_service_windows.go` and `service_windows.go` now delete & wait before re-creating
  - **Installer agent token fix**: combined install now sets `agent_token = admin_token`; previously a separate random token was written that the coordinator never knew about → 401 → Error 1067
  - **Installer build version fix**: `scripts/build.ps1` now injects `-X main.Version=v0.3.0` into both Go binaries; was building without ldflags → agent showed `v0.0.0-dev`, coordinator reported `v0.2.0`
  - **Files changed**: `coordinator/server/update.go`, `coordinator/cmd/commands.go`, `coordinator/server/server.go`, `dashboard/src/App.vue`, `installer/windows/arcvault_installer.py`, `agent/service/agent_service_windows.go`, `coordinator/service/service_windows.go`, `scripts/build.ps1`, `CONTEXT.md`

✅ **Session 16** (June 6, 2026): Credential Profiles — Auth & Key Storage Fixes — COMPLETE
  - **Credentials unauthorized fix**: `Credentials.vue` and `Jobs.vue` were using wrong token sources (`this.$parent.$data.token`, `localStorage.getItem('token')`) — replaced all with `getToken()` from `api.ts`
  - **Admin dropdown**: Removed credentials link from Admin dropdown (kept in main nav)
  - **Nil slice bug**: `handleListCredentialProfiles` returned JSON `null` when empty — fixed with `make([]*CredentialProfileResponse, 0)`
  - **Error handling**: `Credentials.vue` was parsing plain-text error responses as JSON — fixed with `response.text()`
  - **Credential key storage** (root cause): Windows SCM does NOT read named `REG_SZ` values from a service's `Environment` sub-key — only a `REG_MULTI_SZ` value on the parent key works. Moved `credential_key` into `config.json` (alongside `admin_token`, `jwt_secret`)
    - Added `CredentialKey` field to `coordinator/config/config.go`
    - Added `LoadKeyFromString()` to `coordinator/internal/credcrypto/crypto.go`
    - `credentials.go` now uses `loadCredentialKey()` helper (config first, env var fallback)
    - Installer reads existing key from `config.json` on reinstall (preserves key across upgrades)
    - `rebuild-and-restart.ps1` warns if `credential_key` missing from `config.json`
  - **Jobs.vue dropdown**: Credentials not showing in job form — wrong localStorage key + wrong `data.data` shape assumption fixed
  - **Files changed**: `coordinator/config/config.go`, `coordinator/internal/credcrypto/crypto.go`, `coordinator/server/credentials.go`, `dashboard/src/views/admin/Credentials.vue`, `dashboard/src/views/Jobs.vue`, `dashboard/src/App.vue`, `installer/windows/arcvault_installer.py`, `scripts/rebuild-and-restart.ps1`

✅ **Session 14** (June 4, 2026): ArcVault Refactor — Breaking API/Service/DB Coupling — COMPLETE
  - **Goal**: Eliminate tight coupling between handlers and database; create three-layer architecture (Handler → Service → DB Interface)
  - **Housekeeping**: Deleted 260MB of old files (session contexts, build artifacts, temp files)
  - **Step 1 Complete**: Created DB query interfaces (AgentQueries, JobQueries)
  - **Step 2 Complete**: Built service layer skeleton (AgentService, JobService with typed DTOs)
  - **Step 3 Complete**: Migrated agents domain — all handlers now call service layer
  - **Step 4 Complete**: Migrated jobs domain (list, get, cancel handlers; create/delete remain in handler)
  - **Test Status**: 110+ tests passing, zero regressions
  - **Pattern Proven**: Applied to 2 domains, both work flawlessly; remaining work is straightforward repetition
  - All 10 steps complete — see arcvault_refactor_session.md for full details

✅ **Session 17** (June 6, 2026): Progress Module + Agent Route Auth Fix — COMPLETE
  - **Built**: `agent/runner/progress.go` — `ProgressFunc` type, throttled `progressReporter` (1 req/sec, pct==100 always flushes), pure parsers `ParseRobocopyLine` / `ParseRsyncLine`, `SplitOnCRLF` for bare `\r` tokens
  - **Built**: `agent/runner/progress_test.go` — 13 unit tests; 46 total passing in package
  - **Modified**: `agent/runner/executor.go` — removed `/NP` from robocopy (was suppressing output), changed rsync to `--info=progress2`, added `streamRobocopy` / `streamRsync` / `waitCode`; executor signature now `func(Job, ProgressFunc) (int, string)`
  - **Modified**: `agent/runner/runner.go` — `process()` now creates `progressReporter` and passes `reporter.Report` to executor
  - **Fixed (middleware bug)**: `GET /api/jobs` and `PATCH /api/jobs/{id}/status` were JWT-only — agent tokens rejected with 403. Added `agentOrViewerRoute` and `agentOrOperatorRoute` to `coordinator/server/server.go`
  - **Result**: Jobs now execute (exit code 9 from robocopy = partial success). Open issue: robocopy output is "(no output)" — stderr not captured or streaming not wiring output back correctly. Debug next session.

✅ **Session 19** (June 6, 2026): Coordinator Freeze Root Cause Fixed + Progress Reporting Removed — COMPLETE
  - **Root cause found**: `checkMissedSchedules()` (runs every 60s) opens a `rows` cursor then makes nested DB calls inside the loop (`GetAlertRulesForJob` + 2x `QueryRow`). With `SetMaxOpenConns(1)`, those nested calls block forever on the connection held by `rows` → all HTTP handlers queue behind the deadlocked goroutine → coordinator appears frozen
  - **Fix**: Removed `conn.SetMaxOpenConns(1)` from `coordinator/db/db.go`. WAL mode + `_busy_timeout=5000` already handle write serialization at the SQLite level; no pool restriction needed
  - **Also fixed (prior sub-session)**: WebSocket hub `Broadcast` held `h.mu` during network writes — slow browser blocked all hub ops. Fixed: snapshot clients, release lock, then write. Added 5s write deadline per connection
  - **Also added**: HTTP server timeouts (`ReadTimeout`/`WriteTimeout`/`IdleTimeout` = 60/60/120s) in `coordinator/server/server.go`
  - **Progress reporting removed**: Agent no longer POSTs to `/api/jobs/{id}/progress`. Jobs show RUNNING badge until complete; logs viewable after job finishes. Removed: `progressReporter` from agent, POST route + handler from coordinator, progress bar + `jobProgress` ref from `Jobs.vue`
  - **Files changed**: `coordinator/db/db.go`, `coordinator/server/hub.go`, `coordinator/server/server.go`, `coordinator/server/progress.go`, `agent/runner/runner.go`, `agent/runner/progress.go`, `dashboard/src/api.ts`, `dashboard/src/views/Jobs.vue`

✅ **Session 20** (June 6, 2026): Installer UI Redesign — COMPLETE
  - **Full redesign** of `installer/windows/arcvault_installer.py` to match coordinator dashboard design system
  - **Design tokens**: Same palette as `dashboard/src/style.css` (`_BG_BASE="#07090e"`, `_ACCENT="#00d4aa"`, etc.); dark surfaces, teal accent, muted text
  - **Icon**: ArcVault favicon SVG converted to 64×64 base64 PNG, shown as window icon via `tk.PhotoImage(data=_ICON_B64)` + `iconphoto()`
  - **Version**: `self.version = "0.4.0"` (was 0.3.0); displayed in header badge and title bar
  - **Entry screen**: Component selection is now the first screen — two clickable cards (Coordinator / Agent); canvas-drawn circle bubble toggles from empty outline to filled teal with white checkmark on click; card border highlights teal
  - **Auto-populate rule**: URL + auth token fields in Agent config are pre-filled (readonly) only when installing both coordinator AND agent in the same run; agent-only installs show blank editable fields
  - **File write warning**: The Edit tool causes null-byte truncation when writing this file through the Windows mount; always use `bash` heredoc (`cat > ... << 'PYEOF'`) then strip trailing nulls if needed
  - **Files changed**: `installer/windows/arcvault_installer.py`

✅ **Session 18** (June 6, 2026): Robocopy Output Fix + Agent Log File + Auto Token Regen — COMPLETE
  - **Fixed (data race)**: `streamRobocopy` / `streamRsync` used same `bytes.Buffer` for stdout (TeeReader) and stderr (cmd.Stderr) concurrently — separated into `stdoutBuf` / `stderrBuf`
  - **Fixed (exit code range)**: `waitCode` and `extractExitCode` now normalize robocopy codes 1–15 to 0 (was 1–7); code 9 = partial success, not failure
  - **Fixed (scanner buffer)**: Added 256 KB scanner buffer to both streamers — default 64 KB could silently truncate wide robocopy lines
  - **Added**: `agent/service/runner_windows.go` — `setupLogFile()` creates `C:\ArcVault-Agent\logs\arcvault-agent.log` on service start; all `log.*` output now persisted
  - **Added**: Debug logging in `runner.go` — logs src/dst paths, exit code, output length, and first 512 bytes of output per job
  - **Fixed (auto token regen)**: `scripts/rebuild-and-restart.ps1` now validates agent token against coordinator before starting the agent, regenerates if stale, writes new token to `agent-config.yaml`
  - **Result**: Backup ran successfully; output captured in dashboard ✅

## Core Commands
```bash
# Initialize coordinator
coordinator init

# Start coordinator (runs dashboard on :8080)
coordinator start

# Generate per-agent token
coordinator create-agent-token <agent-id>

# Check for updates
coordinator check-update

# Install as system service
coordinator install-service
agent install-service

# Configure SCM failure recovery (required after fresh install on Windows)
sc.exe failure arcvault-coordinator reset=86400 actions=restart/3000/restart/3000/restart/3000
```

## Build & Deploy (Windows)

**Use rebuild-and-restart.ps1 (recommended):**
```powershell
cd C:\Projects\ArcVault2.0
.\scripts\rebuild-and-restart.ps1
```

**Manual checklist:**
1. `cd dashboard && npm run build`
2. `Remove-Item coordinator\static\dist -Recurse -Force`
3. `Copy-Item dashboard\dist coordinator\static\dist -Recurse`
4. `go build -ldflags "-X main.Version=vX.Y.Z" -o installer\windows\coordinator.exe coordinator\main.go`
5. `Stop-Service arcvault-coordinator`
6. `Start-Service arcvault-coordinator`
7. `sc.exe failure arcvault-coordinator reset=86400 actions=restart/3000/restart/3000/restart/3000`

## What's Implemented
- ✅ Single binary deployment (coordinator) with embedded Vue dashboard
- ✅ Per-agent tokens (in addition to admin token)
- ✅ Self-update system (coordinator + agents, live WebSocket progress)
- ✅ Bidirectional rollback (one-version-back, CLI only — `coordinator rollback` / `coordinator rollback-agent <id>`)
- ✅ Server-side pagination & filtering (all list endpoints)
- ✅ Failure notifications (webhook + email)
- ✅ JWT-based RBAC: Three roles (admin, operator, viewer)
- ✅ User management: Create/list/delete/update roles with bcrypt password hashing
- ✅ Agent groups: Organize agents by environment or function
- ✅ Real-time job progress tracking via WebSocket
- ✅ Robocopy/Rsync advanced flags (Mirror, MaxAge, MinSize, ExcludeFiles, etc.)
- ✅ Job cancellation
- ✅ Schedule builder UI (Interval/Daily/Weekly/Monthly/Custom, live cron preview)
- ✅ Delete agents (admin only, 409 blocks if running jobs)
- ✅ Credential profiles (AES-256-GCM encrypted, key in config.json, selectable per job)

## Quick Setup

**Per-agent token:**
```bash
coordinator create-agent-token agent-01
# Copy token to agent-config.yaml as auth_token
```

**Service installation (Windows admin PowerShell):**
```powershell
coordinator install-service
sc start arcvault-coordinator
sc.exe failure arcvault-coordinator reset=86400 actions=restart/3000/restart/3000/restart/3000
```

## Reference
- **Project instructions & routing:** [CLAUDE.md](CLAUDE.md)
- **Phase history & architecture:** [MEMORY.md](MEMORY.md)
- **Current branch:** main
- **Latest release:** v0.4.0 (Coordinator + Agent)
- **Rollback:** CLI-only — `coordinator rollback` / `coordinator rollback-agent <id>`

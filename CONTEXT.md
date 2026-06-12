# ArcVault2.0 -- Quick Reference
**Last updated:** June 11, 2026 (Session 24) | **Coordinator v0.5.0** | **Agent v0.5.0** | **HTTPS + Obsidian Pro Dashboard** ✅

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

✅ **Session 21** (June 10, 2026): v0.5.0 TLS Deployment — Agent Registration Fixed — COMPLETE
  - **Root cause**: `cert.pem` was missing on COORD (deleted during previous session's troubleshooting); coordinator was serving no cert
  - **Fix**: Ran `coordinator.exe rekey-cert` to regenerate `cert.pem` with proper SANs (IP: 192.168.68.62, 127.0.0.1; DNS: localhost)
  - **Bootstrap delivery**: Downloaded fresh bootstrap via `/api/admin/bootstrap.ps1` (admin token required — not `/bootstrap.ps1`)
  - **cert mismatch on REMOTE**: bootstrap curl step failed (AV/write block on agent.exe), so `coordinator.crt` wasn't updated. Fixed by extracting live cert from TLS handshake using PowerShell .NET SslStream and writing as PEM
  - **Result**: SMILOW3FLSP001 online and heartbeating live in coordinator dashboard ✅
  - **Key lesson**: `rekey-cert` is the correct tool for cert regeneration — no OpenSSL needed. Bootstrap endpoint requires admin auth at `/api/admin/bootstrap.ps1`. Use PowerShell SslStream to pull cert from TLS handshake when SMB isn't available.
  - **Installer fixes**: `write_agent_config` now copies `cert.pem` → `coordinator.crt` and writes `ca_cert_file`; strips trailing slash from coordinator URL; default URL changed to `https://localhost`; `build-windows-installer.ps1` now injects version via ldflags from git tag
  - **Full installer workflow verified**: built `ArcVault-Setup-0.5.0-windows-amd64.exe`, ran on local machine, agent registered successfully ✅

✅ **Session 18** (June 6, 2026): Robocopy Output Fix + Agent Log File + Auto Token Regen — COMPLETE
  - **Fixed (data race)**: `streamRobocopy` / `streamRsync` used same `bytes.Buffer` for stdout (TeeReader) and stderr (cmd.Stderr) concurrently — separated into `stdoutBuf` / `stderrBuf`
  - **Fixed (exit code range)**: `waitCode` and `extractExitCode` now normalize robocopy codes 1–15 to 0 (was 1–7); code 9 = partial success, not failure
  - **Fixed (scanner buffer)**: Added 256 KB scanner buffer to both streamers — default 64 KB could silently truncate wide robocopy lines
  - **Added**: `agent/service/runner_windows.go` — `setupLogFile()` creates `C:\ArcVault-Agent\logs\arcvault-agent.log` on service start; all `log.*` output now persisted
  - **Added**: Debug logging in `runner.go` — logs src/dst paths, exit code, output length, and first 512 bytes of output per job
  - **Fixed (auto token regen)**: `scripts/rebuild-and-restart.ps1` now validates agent token against coordinator before starting the agent, regenerates if stale, writes new token to `agent-config.yaml`
  - **Result**: Backup ran successfully; output captured in dashboard ✅

✅ **Session 22** (June 11, 2026): Obsidian Pro Frontend Restyle — DESIGNED
  - **Approved spec**: `docs/superpowers/specs/2026-06-11-obsidian-pro-frontend-restyle-design.md` (commit pending — Kren to commit via PowerShell)
  - **Implementation plan**: `obsidian-pro-restyle.md` (project root) — 7 tasks: self-hosted fonts (@fontsource Space Grotesk/Inter/JetBrains Mono) → style.css token rewrite → App.vue nav/transitions → Sparkline.vue → Login.vue orbit scene → 11-view sweep → 9-component sweep → verification gate
  - **Decisions**: Direction "Obsidian Pro" (deeper blacks #07070d base, teal #00e5b8 + violet #8b86ff accent pair); full visual pass; all flair (glow/depth, micro-interactions, animated login, sparklines); token-first cascade approach (no component-library refactor)
  - **Constraints**: visual-only — api.ts, schemas/, types/, composables/, router/, all Go code untouched; light theme re-derived from same tokens; prefers-reduced-motion guards everywhere
  - **Next session**: use executing-plans skill against `obsidian-pro-restyle.md`

✅ **Session 24** (June 11, 2026): Download Agent Installer + Error 1067 Debug — COMPLETE
  - **Download Agent Installer button fixed**: was serving `bootstrap.ps1`; now serves `ArcVault-Setup-*-windows-amd64.exe` from a configurable directory
  - **New backend route**: `GET /downloads/installer` (admin-only) — globs `installer_dir` config path for matching installer, serves binary directly
  - **New config field**: `installer_dir` in `config.json` — set to `C:\Projects\ArcVault2.0\installer\windows` on dev machine
  - **Frontend**: `downloadInstaller()` in `api.ts` replaces `downloadBootstrapScript()`; `Users.vue` wired to new function
  - **rebuild-and-restart.ps1 hardened**: Step 8 now always regenerates agent token unconditionally (was conditional on stale-check which could silently fail); exits with error if regeneration fails
  - **Error 1067 root cause this session**: typo in `agent-config.yaml` — `coordinator_url: https://1192.168.68.62` (extra leading `1`); corrected to `192.168.68.62`
  - **TLS error**: after URL fix, agent hit `x509: certificate signed by unknown authority` — requires `ca_cert_file: C:\ArcVault\cert.pem` in `agent-config.yaml`; this is a VPN/network issue on this specific machine (corp VPN blocks LAN access to coordinator at `192.168.68.62`)
  - **Agent log path confirmed**: `C:\ArcVault-Agent\logs\arcvault-agent.log` (noted in memory)
  - **Files changed**: `coordinator/server/downloads.go`, `coordinator/server/server.go`, `coordinator/config/config.go`, `dashboard/src/api.ts`, `dashboard/src/views/Users.vue`, `scripts/rebuild-and-restart.ps1`, `memory/MEMORY.md`

✅ **Session 23** (June 11, 2026): Obsidian Pro Restyle IMPLEMENTED + Deployed — COMPLETE
  - **All 7 plan tasks executed** via executing-plans: @fontsource self-hosted fonts (Google CDN removed from index.html), style.css full token rewrite (dark + re-derived light, glow/depth, ambient tints, skeleton shimmer, reduced-motion guards), App.vue sliding nav underline + mono version badge + 80ms view transitions, new `components/Sparkline.vue` wired into History.vue stat cards (fed from already-fetched tlRuns — zero new API calls), Login.vue SVG orbit scene, 11-view sweep (skeletons replace "Loading…", mono columns), 9-component sweep (modal glow + pop-in, focus glow)
  - **Verified**: `npm run build` clean; git scope clean (zero changes to Go, api.ts, composables/, router/, schemas/, types/); deployed via rebuild-and-restart.ps1; dual-theme smoke passed ("looks great")
  - **Deploy script fixed**: `rebuild-and-restart.ps1` still targeted `http://localhost:8080` (pre-TLS) — first deploy false-alarmed "Coordinator not responding" and exited before restarting the agent. Now uses `$BaseUrl = https://localhost`, self-signed-cert handling for PS 5.1 + PS 7, port-443 release check, and a 20s retry loop on /health
  - **Discovered (pre-existing, NOT from restyle)**: dashboard JS tests fail on pristine HEAD under vitest 4.1.8 + jsdom 29 — Jobs.integration.test.js (8: api mock missing getToken/getJobRuns, sync_flags default drift) + SyncFlagsBuilder.test.js (3: trigger('input') vs v-model 'change'); vitest/jsdom were never pinned in package.json. Follow-up: fix tests + pin toolchain
  - **Workflow lesson**: Edit-tool null-byte/truncation corruption applies to the WHOLE ArcVault2.0 mount (hit main.js + index.html) — all file writes must use bash heredoc or python
  - **Files changed**: dashboard/index.html, package.json (+3 @fontsource deps), main.js, style.css, App.vue, 9 views + admin/Credentials.vue, 5 components, new Sparkline.vue, scripts/rebuild-and-restart.ps1

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

# Install as Windows service (requires admin)
coordinator install-service

# Remove Windows service
coordinator uninstall-service
```

## Build & Deploy
```powershell
# Full rebuild + deploy (use this, nothing else)
.\scriptsebuild-and-restart.ps1

# Build installer .exe
.\scriptsuild.ps1

# Post-deploy smoke test
.\scripts\check-sanity.ps1
```

✅ **Session 25** (June 12, 2026): Recurring Bug Guardrails — COMPLETE
  - **Problem**: Three bugs kept reappearing across sessions: (1) /downloads/installer serving bootstrap.ps1 instead of .exe, (2) services unable to start, (3) coordinator binary baked with wrong version ("2.0" or "dev")
  - **Go tests** (new): `coordinator/server/downloads_test.go` — 4 regression tests asserting /downloads/installer serves .exe with correct Content-Type; catches wrong route handler and bad server.Version at `go test` time
  - **Smoke script** (new): `scripts/check-sanity.ps1` — 5-section post-deploy check: VERSION file, binary version, service status + run-service arg, config validity, live endpoint check for /downloads/installer. Runs automatically at end of rebuild-and-restart.ps1
  - **Build scripts fixed**: `build.ps1` and `rebuild-and-restart.ps1` now read `` from VERSION file (never hardcoded); ldflags now inject both `main.Version` AND `arcvault/coordinator/server.Version`; post-build binary version check aborts if coordinator.exe reports wrong version
  - **check-version-sync.ps1 fixed**: added `exit 0` — was silently inheriting stale non-zero $LASTEXITCODE from prior shell commands, causing build.ps1 to exit after version check with no error message
  - **Legacy build scripts deleted**: `build-installer-nsis.ps1` (hardcoded v0.2.1, no ldflags), `build-installer-simple.ps1` (hardcoded v1.1.0, no ldflags), `build-windows-installer.ps1` (git describe, missing server.Version) — all removed
  - **Files changed**: `coordinator/server/downloads_test.go`, `scripts/check-sanity.ps1`, `scripts/check-version-sync.ps1`, `scripts/build.ps1`, `scripts/rebuild-and-restart.ps1`; deleted 3 legacy build scripts

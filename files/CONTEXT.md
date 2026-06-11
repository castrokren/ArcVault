# ArcVault2.0 -- Quick Reference
**Last updated:** June 3, 2026 | **Coordinator:** v0.2.4 | **Agent:** v0.2.4 | **Release Ready ✅**

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
  - **Versions:** coordinator v0.2.4, agent v0.2.4, dashboard embedded in coordinator binary
  - **SyncFlagsBuilder** wired into Jobs form — was imported but never rendered; placed outside form-grid in `.sync-flags-row` wrapper
  - **Version badge** added to nav brand — shows `updateStore.current` as small muted text next to ArcVault logo
  - **build.ps1** created at project root — single command full pipeline: Vue build → clear + copy dist to coordinator/static/dist → Go build with ldflags → Stop-Service → copy binary → Start-Service
  - **Root cause fixed**: `coordinator/static/dist` had accumulated 28+ stale JS files from previous builds; Go embed was picking up wrong files. build.ps1 clears this directory before every build.
  - **v0.2.4 tagged and pushed** with all fixes

🎯 **Next:** Begin Phase 22 (scope TBD)

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

**Use build.ps1 (recommended):**
```powershell
cd C:\Projects\ArcVault2.0
.\build.ps1
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
- ✅ Bidirectional rollback (one-version-back)
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
- **Latest release:** v0.2.4

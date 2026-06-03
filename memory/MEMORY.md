# Memory Index

**Updated:** June 3, 2026 | **Current Version:** v0.2.3

## Current Status

✅ **v0.2.3 Windows Self-Update Fix** — Coordinator self-update fully working on Windows
  - Dashboard embedding fixed (copy dashboard/dist → coordinator/static/dist before build)
  - --version flag added to release binary (required by VerifyBinary)
  - ARCVAULT_SERVICE=1 now set in RunService() so IsServiceMode() returns true
  - Service auto-restarts via SCM failure recovery + os.Exit(1)
  - SCM failure actions must be configured after fresh install:
    `sc.exe failure arcvault-coordinator reset=86400 actions=restart/3000/restart/3000/restart/3000`

🎯 **Next:** Commit all changes → build v0.2.3 release binary → gh release create v0.2.3

## Memory Files

- [Phase 22 Complete](phase22_complete.md) — Full integration testing suite, stress tests, agent disconnect recovery validation
- [History Tab Fix](history_tab_fix.md) — Bug fix for Agent Run Breakdown chart (root causes + solutions)
- [Phase 21a-4 Implementation](phase21a4_implementation.md) — Jobs stuck in pending hot fix (sync_flags + robocopy flags)
- [Phase 21a-4 Lessons Learned](phase21a4_lessons_learned.md) — Debugging insights from hot fix
- [JWT Token Refresh Fix (Session 7)](#jwt-token-refresh-fix) — Update endpoints returning 401 due to missing token persistence
- **[Windows Self-Update Fix (Session 8)](#windows-self-update-fix)** — Full update flow fix for Windows service mode

## Quick Reference

### Latest Issues Fixed

| Date | Issue | Root Cause | Fix |
|------|-------|-----------|-----|
| May 29 | History tab blank | Missing API parameters + no backend filtering + missing agent_id | Added after/search/status params to API, fixed JOIN, updated status field |
| May 29 | Job status stuck as running | Status field not updated on completion | Updated job_results.go to set status based on exit_code + migration to fix historical data |
| May 29 | Agent Run Breakdown not grouping | job_runs table lacks agent_id column | Always JOIN with jobs table to get agent_id |
| June 2 | Update endpoints return 401 Unauthorized | UpdateModal using api.js refreshToken() which doesn't save token | Changed to useAuth().refreshToken() which persists token to localStorage |
| June 3 | Coordinator self-update broken (401 + verify fail + access denied + no restart) | Stale dashboard build + missing --version in release binary + ARCVAULT_SERVICE not set + service can't self-restart | Copy dist before build, set env var in RunService, use os.Exit(1) + SCM recovery actions |

### Phase 22 Key Results

- ✅ Agent disconnect recovery: 100% success rate
- ✅ Linear scaling to 100 agents: ~1000 jobs/sec throughput
- ✅ Memory efficient: 0.3MB for 100 agents
- ✅ Edge cases covered: large paths, high file counts, permissions, disconnects at 50%

### Windows Service Installation Notes (Session 6, June 2)

**Setup Wizard Behavior:**
- Interactive Go-based CLI, not CLI-parameterizable
- Accepts input for: installation type (1=Coordinator, 2=Agent, etc.), port, HTTPS flag, confirmation
- Creates two registry-based Windows services:
  - `arcvault-coordinator` → runs `C:\Projects\ArcVault2.0\installer\windows\coordinator.exe run-service`
  - `arcvault-agent` → runs `C:\Projects\ArcVault2.0\installer\windows\agent.exe run-service`
- Generates config files in same directory: `config.json` (coordinator), `agent-config.yaml` (agent)
- Uses dev binaries, not production install path (no C:\Program Files\ArcVault)

**Agent Service Startup Issue — RESOLVED (Session 6, 13:37 UTC)**
- **Problem:** Agent service exit code 1067, agent couldn't register with coordinator (401 Unauthorized)
- **Root Cause:** Token mismatch between agent config and coordinator database
  - Service loads config from `installer/windows/config.json` (not project root)
  - Agent token was invalid or not in coordinator's tokens table
  - Agent registration failed on startup, causing service crash
- **Solution:** 
  1. Regenerate coordinator and agent tokens using coordinator from installer directory
  2. Ensure both configs use the same database path and valid tokens
  3. Restart services to reload config
- **Test Results:** ✅ Agent now registers as DESKTOP-EE77F38, online, heartbeat working
- **Lesson:** Config file location depends on executable location — paths are relative to exe directory via `filepath.Join(filepath.Dir(exe), "config.json")`

### Windows Self-Update Architecture (Session 8, June 3)

**How it works:**
1. Frontend calls POST /api/update/apply with JWT token
2. Backend downloads release binary from GitHub to coordinator.download.tmp
3. VerifyBinary() runs the binary with --version — must return non-empty string
4. StageBinary() renames tmp → coordinator.new
5. ExecuteUpdate() checks IsServiceMode() — requires ARCVAULT_SERVICE=1 env var
6. ApplyUpdate() (Windows): renames coordinator.exe → coordinator.exe.old, renames coordinator.new → coordinator.exe
7. os.Exit(1) triggers SCM failure recovery → service restarts with new binary

**Critical requirements:**
- Release binary MUST support `--version` flag (exits 0, prints version string)
- `ARCVAULT_SERVICE=1` must be set in RunService() before svc.Run()
- SCM failure recovery must be configured: `sc.exe failure arcvault-coordinator reset=86400 actions=restart/3000/restart/3000/restart/3000`
- Dashboard dist must be copied to coordinator/static/dist before building coordinator binary

**Files involved:**
- `coordinator/service/runner_windows.go` — sets ARCVAULT_SERVICE=1
- `coordinator/updater/updater_windows.go` — ApplyUpdate with rename+exit
- `coordinator/updater/updater.go` — VerifyBinary, ExecuteUpdate, IsServiceMode
- `coordinator/server/update.go` — handleApplyUpdate, performUpdate

### Production Readiness

ArcVault v0.2.3 is **RELEASE READY**:
- ✅ Coordinator self-update working end-to-end on Windows
- ✅ Dashboard 401 fix deployed
- ✅ Service naming standardized (arcvault-coordinator, arcvault-agent)
- ✅ Agent service startup verified (DESKTOP-EE77F38 online)
- ⚠️ SCM failure recovery must be manually configured after fresh install (not automated yet)

---

## JWT Token Refresh Fix (Session 7, June 2, 2026)

### Problem
Update endpoints were returning 401 Unauthorized:
- `/api/update/apply` (coordinator update)
- `/api/agents/:id/update` (agent update)

### Root Cause
Both UpdateModal.vue and AgentUpdateModal.vue were importing `refreshToken` from `api.js`:
```javascript
import { getToken, refreshToken, applyCoordinatorUpdate } from '../api.js'
```

The `api.js` refreshToken() function returns the response but **does NOT save the new token to localStorage**. After the call, the frontend still had the old (expired) JWT token.

### Solution
Changed both components to use the `useAuth()` composable's refreshToken() function, which saves the new token to localStorage before returning.

### Files Changed
- `dashboard/src/components/UpdateModal.vue`
- `dashboard/src/components/AgentUpdateModal.vue`
- Commit: c3ab937

### Lesson
When using api.js's fetch wrappers, always verify that the caller is responsible for persisting tokens. The useAuth() composable handles persistence; raw api.js calls do not.

---

## Windows Self-Update Fix (Session 8, June 3, 2026)

### Problem
Coordinator self-update was failing with multiple cascading issues:
1. `Bearer null` sent in Authorization header → 401
2. `binary verification failed: exit status 1`
3. `failed to replace binary: Access is denied`
4. Service not restarting after update

### Root Causes & Fixes

**Issue 1: Stale dashboard embedded in binary**
- npm run build outputs to `dashboard/dist`, not `coordinator/static/dist`
- Must copy: `Remove-Item coordinator\static\dist -Recurse -Force` then `Copy-Item dashboard\dist coordinator\static\dist -Recurse`
- Then rebuild coordinator binary to embed fresh dashboard

**Issue 2: Release binary missing --version flag**
- v0.2.1 GitHub release was built from older main.go that didn't handle `--version`
- VerifyBinary() runs `binary --version` and checks exit code + output
- Fix: released v0.2.2 with --version case in main.go switch statement

**Issue 3: ARCVAULT_SERVICE never set**
- `IsServiceMode()` checks `os.Getenv("ARCVAULT_SERVICE") == "1"`
- `RunService()` in runner_windows.go never set this env var
- Without it, ExecuteUpdate() takes terminal mode path (direct rename over running exe → Access Denied)
- Fix: added `os.Setenv("ARCVAULT_SERVICE", "1")` to RunService()

**Issue 4: Service cannot restart itself**
- After stopping itself via `net stop`, the service process is dead — nothing runs after
- `net start` from within a service process is blocked by SCM
- Fix: use `os.Exit(1)` after binary replacement — SCM failure recovery restarts the service
- Requires: `sc.exe failure arcvault-coordinator reset=86400 actions=restart/3000/restart/3000/restart/3000`

### Files Changed
- `coordinator/service/runner_windows.go` — added os.Setenv("ARCVAULT_SERVICE", "1")
- `coordinator/updater/updater_windows.go` — rename-out-of-way + os.Exit(1)
- `coordinator/static/dist` — updated dashboard build

### Lesson
Windows service self-update requires: (1) correct env var to detect service mode, (2) rename-out-of-way instead of rename-over, (3) SCM failure recovery instead of self-restart. Never try to restart a Windows service from within its own process.

---

**See also:** C:\Projects\ArcVault2.0\CONTEXT.md for full version history

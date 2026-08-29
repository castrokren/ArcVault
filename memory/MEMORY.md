# Memory Index

**Updated:** June 29, 2026 | **Current Version:** v0.5.0

## Current Status

✅ **v0.5.0 — Installed & Running**
  - Coordinator: v0.5.0 | Agent: v0.5.0
  - TLS/HTTPS ✅ | Credential profiles ✅ | RBAC ✅ | Federation HA ✅
  - Obsidian Pro design system ✅ | Cancel backups ✅ | Login animations ✅
  - **NEW: User Action Audit Logging** — request audit middleware + structured action logging in all mutation handlers
  - **NEW: Orbital Login (Plan A-D implemented)** — orbital canvas, purple theme, motion-v entrance, but warp animation needs optimization

🎯 **Next Actions:**
  - **PRIORITY: Fix login warp animation freeze** — Currently disabled; need to optimize OrbitField.warp() or find lightweight alternative that doesn't block UI
  - Frontend AuditLog.vue page with searchable table, filters, date range picker
  - Audit log retention/pruning (cron-based TTL)
  - Fix pre-existing test failures in `internal/bootstrap` and `internal/tlscert`
  - Update planning/CONTEXT.md (outdated — last modified May 29)

## Memory Files

- [Session 30 (June 30): Orbital Login Performance Fix](#orbital-login-performance) — backdrop-filter blur reduced (24px→8px), warp animation simplified (1.05s→300ms), login now responsive but warp still needs full optimization
- [Session 20: Installer UI Redesign](../CONTEXT.md) — full dark-theme redesign matching dashboard; bubble checkbox cards; icon embedded; URL/token auto-populate only on combined installs; use bash heredoc to write the file
- [Sessions 18–19: Robocopy + Freeze Fix](../CONTEXT.md) — robocopy output/exit code fixed; coordinator freeze fixed (removed SetMaxOpenConns(1)); progress module removed; auto token regen in rebuild script
- [Phase 22 Complete](phase22_complete.md) — Full integration testing suite, stress tests, agent disconnect recovery validation
- [History Tab Fix](history_tab_fix.md) — Bug fix for Agent Run Breakdown chart
- [Phase 21a-4 Implementation](phase21a4_implementation.md) — Jobs stuck in pending hot fix
- [Phase 21a-4 Lessons Learned](phase21a4_lessons_learned.md) — Debugging insights from hot fix
- [JWT Token Refresh Fix (Session 7)](#jwt-token-refresh-fix) — Update endpoints returning 401
- [Windows Self-Update Fix (Session 8)](#windows-self-update-fix) — Full update flow fix for Windows service mode
- [Asset Resolution Fix (Session 9)](#asset-resolution-fix) — resolveAsset fallback for plain coordinator.exe
- [Session 28: User Action Audit Logging](#user-action-audit-logging) — Full user action audit trail: auto-logging middleware + structured handler logging across 35+ mutation handlers; DB table, AuditService, API endpoint, 16 tests

## Quick Reference

### Latest Issues Fixed

| Date | Issue | Root Cause | Fix |
|------|-------|-----------|-----|
| May 29 | History tab blank | Missing API parameters + no backend filtering + missing agent_id | Added after/search/status params to API, fixed JOIN, updated status field |
| May 29 | Job status stuck as running | Status field not updated on completion | Updated job_results.go to set status based on exit_code |
| June 2 | Update endpoints return 401 | UpdateModal using api.js refreshToken() which doesn't save token | Changed to useAuth().refreshToken() which persists token |
| June 3 | Coordinator self-update broken | Stale dashboard + missing --version + ARCVAULT_SERVICE not set + can't self-restart | Copy dist before build, set env var in RunService, os.Exit(1) + SCM recovery |
| June 3 | Nav bar visible when logged out | `<header>` had no auth guard | Added `v-if="auth.isAuthenticated.value"` to header in App.vue |
| June 3 | Agent Update badge false positive | `updateAvailable()` reused coordinator updateStore.available + latest | Compare agent.version against updateStore.current instead |
| June 3 | Agent version hardcoded to 0.1.0 | Version read from config file with hardcoded fallback | Moved to ldflags `-X main.Version`, added `--version` flag, removed from config |
| June 3 | rebuild-and-restart.ps1 deploying to wrong path | Deploy target was C:\ArcVault\ not installer\windows\ | Fixed paths, added SCM disable/re-enable, added ldflags version injection |
| June 3 | Update button missing after update | v0.2.3 not yet released; resolveAsset couldn't find coordinator.exe asset | Released v0.2.3, added plain name fallback to resolveAsset() |

### Windows Self-Update Architecture

**How it works:**
1. Frontend calls POST /api/update/apply with JWT token
2. Backend downloads release binary from GitHub to coordinator.download.tmp
3. VerifyBinary() runs the binary with --version — must return non-empty string
4. StageBinary() renames tmp → coordinator.new
5. ExecuteUpdate() checks IsServiceMode() — requires ARCVAULT_SERVICE=1 env var
6. ApplyUpdate() (Windows): renames coordinator.exe → coordinator.exe.old, renames coordinator.new → coordinator.exe
7. os.Exit(1) triggers SCM failure recovery → service restarts with new binary

**Critical requirements:**
- Release binary MUST support `--version` flag
- `ARCVAULT_SERVICE=1` must be set in RunService() before svc.Run()
- SCM failure recovery must be configured after every fresh install:
  `sc.exe failure arcvault-coordinator reset=86400 actions=restart/3000/restart/3000/restart/3000`
- Dashboard dist must be copied to coordinator/static/dist before building coordinator binary

**Asset naming:**
- Preferred: `coordinator_windows_amd64.exe` (platform-specific)
- Fallback: `coordinator.exe` (plain name — added Session 9)
- resolveAsset() in coordinator/updater/updater.go handles both

**Files involved:**
- `coordinator/service/runner_windows.go` — sets ARCVAULT_SERVICE=1
- `coordinator/updater/updater_windows.go` — ApplyUpdate with rename+exit
- `coordinator/updater/updater.go` — VerifyBinary, ExecuteUpdate, IsServiceMode, resolveAsset
- `coordinator/server/update.go` — handleApplyUpdate, performUpdate

### Windows Service Installation Notes

**Services created by setup wizard:**
- `arcvault-coordinator` → `C:\Projects\ArcVault2.0\installer\windows\coordinator.exe run-service`
- `arcvault-agent` → `C:\Projects\ArcVault2.0\installer\windows\agent.exe run-service`
- Config files in same directory: `config.json` (coordinator), `agent-config.yaml` (agent)
- Config paths are relative to exe directory via `filepath.Join(filepath.Dir(exe), "config.json")`

**Log file paths:**
- Agent log: `C:\ArcVault-Agent\logs\arcvault-agent.log`
- Coordinator log: check Windows Event Viewer or `C:\ArcVault\` for coordinator logs

**Token mismatch lesson (Session 6):**
- Agent token must be valid in coordinator's database
- Regenerate using coordinator binary from installer directory, not project root
- Mismatched tokens → agent 401 → service exit code 1067
- rebuild-and-restart.ps1 now always regenerates token unconditionally (Step 8)

**Error 1067 checklist:**
1. Check `C:\ArcVault-Agent\logs\arcvault-agent.log` — the real error is always there
2. Common causes: bad coordinator URL in agent-config.yaml (typo), token mismatch, coordinator not running

### Production Readiness

ArcVault v0.4.0 is **RELEASE READY**:
- ✅ Coordinator self-update working end-to-end on Windows
- ✅ Asset resolution works for both coordinator_windows_amd64.exe and coordinator.exe naming
- ✅ Dashboard 401 fix deployed
- ✅ Service naming standardized (arcvault-coordinator, arcvault-agent)
- ✅ Agent service startup verified (DESKTOP-EE77F38 online)
- ⚠️ SCM failure recovery must be manually configured after fresh install (not automated yet)

---

## Asset Resolution Fix (Session 9, June 3, 2026)

**Issue:** Update button disappeared after coordinator self-updated; running v0.2.2, GitHub showed v0.2.2 as latest because v0.2.3 hadn't been released yet.

**Secondary bug found:** resolveAsset() only matched platform-specific names (coordinator_windows_amd64.exe). GitHub release had asset named coordinator.exe — would cause silent failure on future updates.

**Fix:** Added plain name fallback loop to resolveAsset() in coordinator/updater/updater.go. Checks coordinator_windows_amd64.exe first, falls back to coordinator.exe.

**Result:** v0.2.3 released with fix, service running clean, update flow verified end-to-end.

---

## JWT Token Refresh Fix (Session 7, June 2, 2026)

**Issue:** Update endpoints returning 401 — UpdateModal/AgentUpdateModal using api.js refreshToken() which doesn't persist token to localStorage. Fixed by switching to useAuth().refreshToken(). Commit: c3ab937.

---

## Windows Self-Update Fix (Session 8, June 3, 2026)

Four cascading issues fixed: stale dashboard embedding, missing --version flag in release binary, ARCVAULT_SERVICE env var never set, service can't self-restart via net start. Solutions: copy dist before build, --version in main.go switch, os.Setenv in RunService(), os.Exit(1) + SCM recovery actions.

**Lesson:** Never try to restart a Windows service from within its own process. Use os.Exit(1) + SCM failure recovery.

---

---

## Phase 22 API Contract Layer & Pagination Fix (Session 12, June 3, 2026)

### The Problem

After implementing Phase 22 API Contract Layer (TypeScript types + Zod validation), two critical bugs emerged:

1. **Users not displaying** after user creation — created users were invisible on Users tab
2. **Agents not displaying** — "[API Contract] /api/agents validation failed" error in browser console

### Root Causes

#### Users Bug
- **Backend**: `/api/users` endpoint returned raw array `[...users]` instead of paginated response
- **Frontend**: `Users.vue` was trying to access `data.users` field which didn't exist
- **Result**: `users.value = undefined || []` → empty list

**Fix:**
```go
// In coordinator/server/auth.go handleListUsers
return c.JSON(http.StatusOK, NewPaginatedResponse(users, total, page, limit))
```

```javascript
// In dashboard/src/views/Users.vue line 275
users.value = data.data || []  // Changed from data.users || []
```

#### Agents Bug (Two-Part)
**Part 1: API Response Mismatch**
- `/api/agents` returns paginated response: `{data: [...], total: 1, page: 1, pages: 1, limit: 25}`
- But `api.ts getAgents()` was extracting and returning just the array
- `Agents.vue` expected full response object with `.data` property

**Part 2: Frontend Misunderstanding**
- `Agents.vue` line 138: `const agents = computed(() => ... (result.value.data || []))`
- `Agents.vue` line 186: `result.value = await getAgents(...)`
- Code expected `getAgents()` to return `{data: [...], total, page, pages, limit}`
- But it was getting just `[...]` which has no `.data` property

**Fix:**
```typescript
// In dashboard/src/api.ts
export const getAgents = async ({ page = 1, limit = 25, search = '', status = '' } = {}) => {
  const res = await request('GET', `/api/agents${buildQuery({ page, limit, search, status })}`)
  return res  // Return full paginated response object, not extracted array
}

// Same fix applied to getJobs()
export const getJobs = async ({ page = 1, limit = 25, search = '', status = '', agentID = '' } = {}) => {
  const res = await request('GET', `/api/jobs${buildQuery({ page, limit, search, status, agent_id: agentID })}`)
  return res  // Return full paginated response object
}
```

### Deployment Process

1. **Dashboard Build**: `npm run build` in dashboard directory
2. **Static Files**: Copy `dashboard/dist/*` to `coordinator/static/dist/`
3. **Go Rebuild**: `go build -o arcvault-coordinator.exe .` in coordinator directory
4. **Binary Deploy**: Copy `coordinator/arcvault-coordinator.exe` to `installer/windows/coordinator.exe`
5. **Service Restart**: Restart ArcVault Coordinator service via Services.msc
6. **Verification**: Check dashboard — agents should display, no API validation errors

### Result

✅ **Agents**: Now displaying correctly (1 agent online)
⚠️ **Users**: Still not displaying (needs investigation in next session)

### Key Lesson

Paginated list endpoints need special handling:
- Backend must return consistent response shape: `{data: [...], total, page, pages, limit}`
- Frontend API functions should return the full response object, not extract data
- Components should access the `.data` property from the response

**Pattern:**
```typescript
// ❌ WRONG
export const getAgents = async (opts) => {
  const res = await request(...)
  return (res.data || res) as Types.Agent[]  // Extracted array loses pagination metadata
}

// ✅ CORRECT
export const getAgents = async (opts) => {
  return await request(...)  // Return full object with data, total, page, pages, limit
}

// Component:
result.value = await getAgents(...)
agents = computed(() => result.value.data || [])
```

---

## Why Users & Agents Keep Breaking Each Other (Session 13, June 3, 2026)

### The Problem
After Phase 22 API contract layer was implemented, fixing one endpoint would break the other:
- Fix Agents → Users broken
- Fix Users → Agents broken

### Root Cause: Dual-Layer Inconsistency

**Backend Inconsistency** (Primary Issue)
- `/api/agents` endpoint: Returns `{data: [...], total, page, pages, limit}`
- `/api/users` endpoint: Returns `[...]` (raw array, NO pagination wrapper)

**Frontend Inconsistency** (Secondary Issue)
- Agents.vue: Accessed `result.value.data` expecting paginated format
- Users.vue: Accessed `data.users` expecting... something different entirely

### How They Broke Each Other

1. **Session 12 Attempt**: Frontend fix changed Users.vue to match "paginated" format
   - But backend still returned raw array — data.users field didn't exist
   - So Users remained broken
   - Agents still worked because backend returned correct format

2. **When someone tried to "fix" Users by changing the backend**:
   - If they only fixed the frontend to access `.data` instead of `.users`
   - But backend still returned raw array...
   - Then both would break until backend also wrapped response

3. **The Cycle**:
   - Frontend and backend had to both agree on format
   - Partial fixes at only one layer caused cascading failures
   - Each "fix" attempt at the wrong layer broke the other component

### The Solution: Consistency at Both Layers

✅ **Backend**: Both endpoints now use `NewPaginatedResponse()`
```go
// agents.go line 187
json.NewEncoder(w).Encode(NewPaginatedResponse(agents, total, p.Page, p.Limit))

// auth.go line 479 (AFTER FIX)
json.NewEncoder(w).Encode(NewPaginatedResponse(response, total, p.Page, p.Limit))
```

✅ **Frontend**: Both components access `.data` property
```typescript
// Agents.vue line 138
const agents = computed(() => ... (result.value.data || []))

// Users.vue line 275 (AFTER FIX)  
users.value = data.data || []
```

### Key Lesson
**API contracts must be consistent across all endpoints of the same type.** When paginated list endpoints exist, they should all:
1. Accept same pagination parameters (page, limit, search, etc.)
2. Return identical response structure: `{data: [...], total, page, pages, limit}`
3. Apply pagination at the backend (not the frontend)

Partial consistency at just one layer (frontend OR backend) creates a fragile system where "fixes" for one endpoint break another.

---

## User Action Audit Logging (Session 28, June 29, 2026)

### What was built

A complete user action audit trail system with two layers:

**1. Request Audit Middleware** (`server/request_audit.go`)
- Wraps every API request (skips `/health`, `/ws/*`)
- Captures: method, path, user identity, status code, latency
- Best-effort insert — never blocks the request
- Action type: `"request"`

**2. Structured Action Logging** (35+ mutation handlers)
- Each handler logs semantic actions: `"user.create"`, `"job.cancel"`, `"auth.login"`
- Captures: resource type, resource ID, success/failure, error details
- IP extracted via `ClientIP()` helper (X-Forwarded-For → X-Real-IP → RemoteAddr)

### Architecture

| Layer | Package | Key File |
|-------|---------|----------|
| Migration | `db/db.go` | `CREATE TABLE user_audit_log (...)` |
| DB Queries | `db/audit.go` | `InsertUserAuditLog()`, `ListUserAuditLogs()` |
| Interface | `db/queries.go` | `AuditQueries` |
| Business | `business/audit.go` | `AuditService.LogAction()`, `ListAuditLogs()`, `ClientIP()` |
| Middleware | `server/request_audit.go` | `requestAuditMiddleware` |
| API | `server/user_audit.go` | `GET /api/audit/user-actions` |

### Actions Tracked

- **Auth**: login, logout, change_password
- **Users**: create, delete, update_role
- **Jobs**: create, delete, cancel
- **Agents**: register, delete, update, rollback
- **Credentials**: create, delete
- **Templates**: create, update, delete, run
- **Groups**: create, update, delete, add_agent, remove_agent
- **Federation**: create, update, delete, sync
- **Coordinator**: update, rollback
- **Alert Rules**: create, update, delete, retry
- **Auto (middleware)**: every API request

### Key Design Decisions

- **Middleware + explicit calls**: Middleware provides comprehensive raw trail; explicit calls add semantic richness
- **Single table**: Both middleware and handler calls write to the same `user_audit_log` table
- **Best-effort**: Audit writes never block or fail the request (errors silently dropped)
- **No frontend yet**: Backend-only to start; AuditLog.vue deferred to follow-up

### Files Changed/Created (22 files)

New: `db/audit.go`, `db/audit_test.go`, `business/audit.go`, `business/audit_test.go`, `server/request_audit.go`, `server/user_audit.go`
Modified: `db/db.go`, `db/queries.go`, `server/server.go`, `business/mocks_test.go`, and 9 handler files

---

**See also:** C:\Projects\ArcVault2.0\CONTEXT.md for full version history

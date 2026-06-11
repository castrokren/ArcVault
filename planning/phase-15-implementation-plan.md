# Phase 15 Implementation Plan — Agent Groups + RBAC
**Project:** ArcVault2.0  
**Version target:** v0.8.0  
**Branch:** `feature/phase-15-rbac`  
**Precondition:** Phase 14 complete, v0.7.0 tagged ✅  
**Last updated:** 2026-05-20

---

## Decisions Driving This Plan

| ID | Decision |
|---|---|
| D-002 | Existing per-agent bearer tokens kept as service-account tokens (backward compat) |
| D-003 | Stateless coordinator — JWT chosen over DB sessions |
| D-010 | Signed JWT (HS256), secret in config, 24h expiry, logout = client-side drop |
| D-011 | Group job dispatch = fan-out to all agents in group; each gets own jobs row |
| D-012 | First-run auto-seeds admin/changeme; forced password change on first login |

---

## Architecture Overview

```
New DB tables:    users, agent_groups, agent_group_members
New Go files:     auth.go, auth_test.go, groups.go, groups_test.go
                  db/users.go, db/groups.go
New Vue files:    Login.vue, ChangePassword.vue, Groups.vue, Users.vue
                  components/AuthGuard.vue
Modified:         db.go, server.go, agents.go, jobs.go,
                  router/index.js, api.js, App.vue (+ move to correct location)
```

**Middleware stack (all API routes):**
```
Request → JWTMiddleware (or legacy AgentTokenMiddleware) → RoleCheck → Handler
```

---

## Pre-flight

- [ ] Create branch `feature/phase-15-rbac` from `main`
- [ ] Confirm existing 111 tests pass: `go test ./...` from project root

---

## Task List

Tasks are ordered by dependency. Complete each fully before starting the next.
Run `go test ./coordinator/server/` after every backend task.

---

### TASK 1 — DB Schema: users

**Files:** `coordinator/db/db.go`, `coordinator/db/users.go` (new)

**Steps:**
1. Add to `db.go` `createTables()`:
```sql
CREATE TABLE IF NOT EXISTS users (
    id                   INTEGER PRIMARY KEY AUTOINCREMENT,
    username             TEXT NOT NULL UNIQUE,
    password_hash        TEXT NOT NULL,
    role                 TEXT NOT NULL CHECK(role IN ('admin','operator','viewer')),
    must_change_password INTEGER NOT NULL DEFAULT 0,
    created_at           DATETIME DEFAULT CURRENT_TIMESTAMP
);
```
2. Create `coordinator/db/users.go` with the following functions:
   - `CreateUser(username, passwordHash, role string, mustChange bool) error`
   - `GetUserByUsername(username string) (*User, error)`
   - `CountUsers() (int, error)`
   - `UpdatePassword(userID int, newHash string, mustChange bool) error`
   - `ListUsers() ([]User, error)`
   - `DeleteUser(userID int) error`
   - `User` struct: `ID, Username, PasswordHash, Role, MustChangePassword, CreatedAt`

3. Add first-run seeding to coordinator startup (in `server.go` or `main.go`):
```go
count, _ := db.CountUsers()
if count == 0 {
    hash, _ := bcrypt.GenerateFromPassword([]byte("changeme"), bcrypt.DefaultCost)
    db.CreateUser("admin", string(hash), "admin", true)
    log.Println("[startup] Default admin user created (admin/changeme) — change password on first login")
}
```

**Verify:**
- [ ] `go test ./coordinator/db/` — new user CRUD tests pass
- [ ] Manual: start coordinator fresh, confirm log line printed, confirm `users` table has 1 row

---

### TASK 2 — DB Schema: agent_groups

**Files:** `coordinator/db/db.go`, `coordinator/db/groups.go` (new)

**Steps:**
1. Add to `db.go` `createTables()`:
```sql
CREATE TABLE IF NOT EXISTS agent_groups (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    name        TEXT NOT NULL UNIQUE,
    description TEXT,
    created_at  DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS agent_group_members (
    group_id  INTEGER NOT NULL REFERENCES agent_groups(id) ON DELETE CASCADE,
    agent_id  TEXT NOT NULL,
    PRIMARY KEY (group_id, agent_id)
);
```
2. Create `coordinator/db/groups.go` with the following functions:
   - `CreateGroup(name, description string) (*AgentGroup, error)`
   - `GetGroup(id int) (*AgentGroup, error)`
   - `ListGroups() ([]AgentGroup, error)`
   - `UpdateGroup(id int, name, description string) error`
   - `DeleteGroup(id int) error`
   - `AddAgentToGroup(groupID int, agentID string) error`
   - `RemoveAgentFromGroup(groupID int, agentID string) error`
   - `GetGroupMembers(groupID int) ([]string, error)`
   - `GetAgentGroup(agentID string) (*AgentGroup, error)` — returns group agent belongs to, or nil
   - `AgentGroup` struct: `ID, Name, Description, CreatedAt`

**Verify:**
- [ ] `go test ./coordinator/db/` — all group CRUD tests pass

---

### TASK 3 — JWT Auth Middleware

**Files:** `coordinator/server/auth.go` (new), `coordinator/config/config.go`

**Steps:**
1. Add `JWTSecret string` to config `Config` struct — auto-generated 32-byte random on first run if absent, written back to config.json

2. Create `coordinator/server/auth.go` with:
   - `GenerateJWT(userID int, username, role string, secret string) (string, error)` — HS256, 24h expiry, claims: `sub`, `username`, `role`, `must_change`
   - `ValidateJWT(tokenString, secret string) (*JWTClaims, error)`
   - `JWTMiddleware(secret string) gin.HandlerFunc` — extracts Bearer token, sets `userClaims` in context; falls through to `AgentTokenMiddleware` if not a JWT (backward compat for agent tokens)
   - `RequireRole(roles ...string) gin.HandlerFunc` — reads `userClaims` from context, 403 if role not in list
   - `RequirePasswordChange() gin.HandlerFunc` — 403 with `{"error":"must_change_password"}` if `must_change=true` (exempts `/api/auth/change-password`)
   - `JWTClaims` struct: `UserID int`, `Username string`, `Role string`, `MustChange bool`

3. Existing agent token auth remains as `AgentTokenMiddleware` — unchanged, checked after JWT fails

**Verify:**
- [ ] `go test ./coordinator/server/ -run TestAuth` — JWT generate/validate/expire/role tests pass

---

### TASK 4 — Auth API Endpoints

**Files:** `coordinator/server/auth.go` (extend), `coordinator/server/server.go`

**Endpoints:**
```
POST  /api/auth/login            — no auth required
POST  /api/auth/logout           — auth required (client drops token; server returns 200)
GET   /api/auth/me               — auth required
PUT   /api/auth/change-password  — auth required (works even if must_change=true)
```

**Steps:**
1. `POST /api/auth/login`:
   - Body: `{"username":"...","password":"..."}`
   - Lookup user, bcrypt compare
   - Return: `{"token":"...","role":"...","must_change_password":true/false}`
   - 401 on bad credentials — same message for user-not-found and wrong-password (no enumeration)

2. `POST /api/auth/logout`: returns `{"ok":true}` (token dropped client-side)

3. `GET /api/auth/me`: returns `{"id":1,"username":"admin","role":"admin","must_change_password":false}`

4. `PUT /api/auth/change-password`:
   - Body: `{"current_password":"...","new_password":"..."}`
   - Verify current, bcrypt new, set `must_change_password=false`
   - Min password length: 8 chars

5. Register all `/api/auth/` routes in `server.go` — these bypass `RequirePasswordChange` middleware

**Verify:**
- [ ] `go test ./coordinator/server/ -run TestAuthEndpoints`
- [ ] Manual: login with admin/changeme → get token → `GET /api/auth/me` → returns admin user

---

### TASK 5 — User Management Endpoints (admin only)

**Files:** `coordinator/server/auth.go` (extend)

**Endpoints:**
```
GET    /api/users              — admin only
POST   /api/users              — admin only
DELETE /api/users/{id}         — admin only (cannot delete self)
PUT    /api/users/{id}/role    — admin only
```

**Steps:**
1. `GET /api/users` — list all users (omit `password_hash` from response)
2. `POST /api/users` — body: `{"username":"...","password":"...","role":"..."}` — creates user with `must_change=true`
3. `DELETE /api/users/{id}` — prevent deleting own account (return 400)
4. `PUT /api/users/{id}/role` — body: `{"role":"..."}` — updates role only

**Verify:**
- [ ] `go test ./coordinator/server/ -run TestUserManagement`

---

### TASK 6 — Wire Auth Middleware into All Existing Routes

**Files:** `coordinator/server/server.go`

**Steps:**
1. Apply `JWTMiddleware` globally to all `/api/` routes
2. Apply `RequirePasswordChange` globally (exempts `/api/auth/` routes)
3. Role-gate existing routes:
   - `admin` only: `/api/agents` DELETE, `/api/federation` POST/PUT/DELETE, `/api/users` all
   - `operator+`: `/api/jobs` POST (trigger), `/api/templates` POST (run)
   - `viewer+`: all GET endpoints
4. Agent endpoints (`/ws/agent`, `/api/agent/*`) keep existing `AgentTokenMiddleware` — not JWT-gated

**Verify:**
- [ ] `go test ./coordinator/server/` — all 111 existing tests still pass
- [ ] New auth middleware tests pass
- [ ] Manual: call `/api/agents` without token → 401; with viewer token → 200

---

### TASK 7 — Groups API Endpoints

**Files:** `coordinator/server/groups.go` (new), `coordinator/server/groups_test.go` (new)

**Endpoints:**
```
GET    /api/groups                        — viewer+
POST   /api/groups                        — admin only
GET    /api/groups/{id}                   — viewer+
PUT    /api/groups/{id}                   — admin only
DELETE /api/groups/{id}                   — admin only
POST   /api/groups/{id}/agents            — admin only  body: {"agent_id":"..."}
DELETE /api/groups/{id}/agents/{agentID}  — admin only
GET    /api/groups/{id}/agents            — viewer+
```

**Steps:**
1. Implement all 8 handlers using `db/groups.go` helpers
2. `DELETE /api/groups/{id}` — cascade removes memberships (DB ON DELETE CASCADE handles it)
3. `POST /api/groups/{id}/agents` — validate agent exists before adding
4. Response shape for group: `{"id":1,"name":"prod","description":"...","agent_count":3,"created_at":"..."}`

**Verify:**
- [ ] `go test ./coordinator/server/ -run TestGroups` — full CRUD + membership tests pass

---

### TASK 8 — Group Fan-Out Job Dispatch

**Files:** `coordinator/server/jobs.go`, `coordinator/db/db.go`

**Steps:**
1. Add columns to `jobs` table:
```sql
ALTER TABLE jobs ADD COLUMN group_id    INTEGER REFERENCES agent_groups(id);
ALTER TABLE jobs ADD COLUMN dispatch_id TEXT;
```
   `dispatch_id` is a shared UUID across all jobs in a fan-out batch (for dashboard grouping)

2. Modify `POST /api/jobs` to accept optional `group_id` in body:
   - If `group_id` present: fetch group members, create one job row per agent, all share same `dispatch_id`
   - If group is empty: return 400 `{"error":"group has no agents"}`
   - If both `agent_id` and `group_id` provided: return 400

3. `GET /api/jobs` — existing pagination/filtering unchanged; add optional `?group_id=` and `?dispatch_id=` filter params

**Verify:**
- [ ] `go test ./coordinator/server/ -run TestGroupDispatch`
- [ ] Manual: create group with 2 agents, POST job to group → 2 job rows with same `dispatch_id`

---

### TASK 9 — App.vue Relocation + Vue Auth Layer

**Files:** `dashboard/src/App.vue` (move from `router/`), `dashboard/src/auth.js` (new), `dashboard/src/router/index.js`, `dashboard/src/api.js`

**Steps:**
1. Move `dashboard/src/router/App.vue` → `dashboard/src/App.vue` (fixes Phase 14 known issue)
2. Update `dashboard/src/main.js` import path if needed

3. Create `dashboard/src/auth.js`:
```js
import { reactive } from 'vue'
export const auth = reactive({ user: null, token: null })
```

4. Add JWT token management to `api.js`:
   - `setAuthToken(token)` / `getAuthToken()` / `clearAuthToken()` — localStorage
   - Axios request interceptor: attach `Authorization: Bearer <token>` header
   - Axios response interceptor: on 401 → clear token → redirect to `/login`

5. Add navigation guard to `router/index.js`:
```js
router.beforeEach((to, from, next) => {
  if (to.meta.requiresAuth && !auth.token) next('/login')
  else next()
})
```
6. Mark all existing routes with `meta: { requiresAuth: true }`

**Verify:**
- [ ] Dashboard loads unauthenticated → redirected to `/login`
- [ ] No console errors on redirect

---

### TASK 10 — Login.vue + ChangePassword.vue

**Files:** `dashboard/src/views/Login.vue` (new), `dashboard/src/views/ChangePassword.vue` (new), `dashboard/src/router/index.js`

**Steps:**
1. Create `Login.vue`:
   - Fields: username, password
   - On submit: `POST /api/auth/login`
   - On success: store token + user in `auth`, redirect to `/`
   - If `must_change_password=true`: redirect to `/change-password` instead
   - Error state: "Invalid username or password" (no field-specific errors)

2. Create `ChangePassword.vue`:
   - Fields: current password, new password, confirm new password
   - On success: set `auth.user.must_change_password = false`, redirect to `/`

3. Add routes:
   - `/login` — no auth required
   - `/change-password` — auth required, accessible even when `must_change=true`

4. Style: match existing ArcVault dashboard aesthetic (dark theme, same CSS vars)

**Verify:**
- [ ] Login with admin/changeme → redirected to `/change-password`
- [ ] Change password → redirected to dashboard
- [ ] Invalid login → error message shown, no redirect

---

### TASK 11 — Groups.vue

**Files:** `dashboard/src/views/Groups.vue` (new), `dashboard/src/router/index.js`, `dashboard/src/api.js`

**Steps:**
1. Add group API calls to `api.js`:
   - `getGroups()`, `createGroup(data)`, `updateGroup(id, data)`, `deleteGroup(id)`
   - `getGroupAgents(groupId)`, `addAgentToGroup(groupId, agentId)`, `removeAgentFromGroup(groupId, agentId)`

2. Create `Groups.vue`:
   - Table: Name | Description | Agent Count | Actions (edit/delete) — admin only
   - "New Group" button (admin only) → inline form or modal
   - Click group row → expand to show member agents with remove buttons
   - Add agent to group: dropdown of ungrouped agents

3. Register route `/groups` in `router/index.js`, add to nav

**Verify:**
- [ ] Create group, add 2 agents — count updates correctly
- [ ] Remove agent from group — count decrements
- [ ] Delete group — agents still exist (ungrouped)

---

### TASK 12 — Agents.vue + Jobs.vue Group Integration

**Files:** `dashboard/src/views/Agents.vue`, `dashboard/src/views/Jobs.vue`

**Steps:**
1. `Agents.vue`:
   - Add "Group" column to agent table (shows group name or "—")
   - Add group filter dropdown alongside existing site filter

2. `Jobs.vue`:
   - Add "Run on Group" option in trigger modal (dropdown of groups)
   - Fan-out jobs grouped visually by `dispatch_id` (collapsible rows)
   - Add `?group_id=` filter to job list

**Verify:**
- [ ] Agent table shows group column correctly
- [ ] Trigger job on group → fan-out rows appear grouped by `dispatch_id` in Jobs view

---

### TASK 13 — Role-Gated UI + Users.vue

**Files:** `dashboard/src/components/AuthGuard.vue` (new), `dashboard/src/views/Users.vue` (new), all views

**Steps:**
1. Create `AuthGuard.vue` — simple wrapper that checks `auth.user.role`:
```vue
<AuthGuard :roles="['admin']">
  <button>Delete Agent</button>
</AuthGuard>
```

2. Wrap admin-only UI elements across all views:
   - Delete buttons on Agents, Jobs, Federation
   - "New Group" button in Groups.vue
   - User management nav item

3. Create `Users.vue` (admin-only view):
   - Table: Username | Role | Must Change Password | Actions
   - Create user button, delete user button (cannot delete self)
   - Change role dropdown inline

4. Register route `/users` in `router/index.js`, add to nav (hidden for non-admin via `AuthGuard`)

**Verify:**
- [ ] Login as viewer → delete buttons hidden, admin nav items hidden
- [ ] Login as operator → can trigger jobs, cannot delete agents
- [ ] Login as admin → full access

---

### TASK 14 — Full Test Pass + Smoke Test + Cleanup

**Steps:**
1. `go test ./...` — all tests pass (target: 111 existing + ~40 new = ~150 total)
2. `cd dashboard && npm run build` — no errors
3. `go build -o coordinator/arcvault-coordinator.exe coordinator/main.go` — clean build

Smoke test checklist:
- [ ] Fresh coordinator start → seeded admin user log line printed
- [ ] Login with admin/changeme → forced to change password
- [ ] Change password → dashboard loads
- [ ] Create group → add agents → trigger job on group → fan-out rows visible in Jobs
- [ ] Create operator user → login → verify role restrictions enforced
- [ ] Login as viewer → verify read-only UI
- [ ] Existing agent WebSocket connection still works (backward compat)
- [ ] No console errors in browser

Memory file updates:
- [ ] Fix `roadmap.md` — update phases 13–17 to reflect what actually shipped
- [ ] Fix `design-planning/CONTEXT.md` — current focus → Phase 15 complete
- [ ] Fix `CLAUDE.md` — version ref v0.5.0 → v0.8.0
- [ ] Update root `CONTEXT.md` — Phase 15 complete, v0.8.0 tagged

---

### TASK 15 — Commit

```
git add coordinator/db/users.go coordinator/db/groups.go coordinator/db/db.go `
      coordinator/server/auth.go coordinator/server/auth_test.go `
      coordinator/server/groups.go coordinator/server/groups_test.go `
      coordinator/server/jobs.go coordinator/server/server.go `
      coordinator/config/config.go `
      dashboard/src/App.vue dashboard/src/auth.js dashboard/src/api.js `
      dashboard/src/router/index.js `
      dashboard/src/views/Login.vue dashboard/src/views/ChangePassword.vue `
      dashboard/src/views/Groups.vue dashboard/src/views/Users.vue `
      dashboard/src/components/AuthGuard.vue `
      dashboard/src/views/Agents.vue dashboard/src/views/Jobs.vue
git commit -m "feat: agent groups + RBAC (user login roles, JWT auth, group fan-out dispatch)"
```

---

## Done

All tasks complete → use `finishing-a-development-branch` skill to merge and tag v0.8.0.

---

## Summary Table

| Task | Area | New Files | Modified Files | Estimated effort |
|---|---|---|---|---|
| 1 | DB: users | `db/users.go` | `db/db.go` | 1–2h |
| 2 | DB: groups | `db/groups.go` | `db/db.go` | 1–2h |
| 3 | JWT middleware | `server/auth.go` | `config/config.go` | 2–3h |
| 4 | Auth endpoints | — | `server/auth.go`, `server.go` | 2h |
| 5 | User mgmt endpoints | — | `server/auth.go` | 1–2h |
| 6 | Wire middleware | — | `server/server.go` | 1–2h |
| 7 | Groups endpoints | `server/groups.go`, `server/groups_test.go` | `server/server.go` | 2–3h |
| 8 | Group fan-out dispatch | — | `server/jobs.go`, `db/db.go` | 2–3h |
| 9 | Vue auth layer + App.vue fix | `auth.js` | `api.js`, `router/index.js`, `App.vue` | 2h |
| 10 | Login.vue + ChangePassword.vue | `Login.vue`, `ChangePassword.vue` | `router/index.js` | 2–3h |
| 11 | Groups.vue | `Groups.vue` | `api.js`, `router/index.js` | 2–3h |
| 12 | Agents + Jobs group integration | — | `Agents.vue`, `Jobs.vue` | 2h |
| 13 | Role-gated UI + Users.vue | `AuthGuard.vue`, `Users.vue` | All views | 2–3h |
| 14 | Tests + smoke test + cleanup | — | Memory files, `roadmap.md` | 2h |
| 15 | Commit | — | — | 15m |

**Total estimate:** 4–5 weeks (solo, part-time)

---

## Rules for This Plan

- ❌ Never rewrite without Kren's approval
- ✅ Pre-flight: branch from main, confirm 111 tests pass before starting
- ✅ Test after every backend task (`go test ./coordinator/server/`)
- ✅ Proof before "done" — no claiming complete without test output
- ✅ Bugs traced to root cause before fixing
- ✅ Follow existing patterns: Go packages lowercase, Vue PascalCase, routes kebab-case
- ✅ PowerShell line continuation: backtick (`), not backslash

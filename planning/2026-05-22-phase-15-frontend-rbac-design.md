# Phase 15 Frontend RBAC Design

**Date:** 2026-05-22  
**Phase:** 15 (Frontend)  
**Status:** Design Approved  
**Author:** Claude + Kren Castro

---

## Executive Summary

Implement JWT-based RBAC frontend for ArcVault, replacing the current admin-token-only system with per-user authentication and role-based access control. Three roles (admin, operator, viewer) control feature visibility and API access.

**Key Design Choices:**
- Hybrid auth: Admin bearer token for first setup, username/password login for users
- Visible-but-disabled UI: Role-based features shown but disabled, not hidden
- Smart form: Single job creation form with intelligent agent/group switching
- Full CRUD: Complete user and group management interfaces
- Auto-refresh: JWT tokens auto-refresh on expiry with optional "remember me" persistence

---

## 1. Authentication & Session Management

### 1.1 Auth Flow

**First-Time Setup (Admin):**
1. User opens dashboard, sees token-gate (current implementation)
2. Admin enters bearer token → saved to localStorage
3. Dashboard becomes available → admin can access Users/Groups panels
4. Admin creates first user account (username, password, role)
5. Admin logs out or browser refresh

**User Login (Subsequent):**
1. User opens dashboard, no token in localStorage/memory
2. Redirected to Login.vue form
3. Enter username/password → POST /api/auth/login
4. Backend returns JWT token + user info (id, username, role)
5. User checks "Remember me" checkbox:
   - If checked: Token stored in localStorage, persists across refreshes
   - If unchecked: Token stored in memory only, lost on refresh
6. Redirect to /agents dashboard

**Token Lifecycle:**
- JWT expiry: 24 hours (backend-set)
- Auto-refresh: On every API request, if token within 1 hour of expiry, silently refresh
- Manual logout: DELETE /api/auth/logout, clear token from storage
- Expired token on API call: Auto-refresh; if refresh fails, redirect to login

### 1.2 Auth State Management

**useAuth.js composable:**
```javascript
// State
currentUser: { id, username, role } // from JWT payload
isAuthenticated: boolean
userRole: 'admin' | 'operator' | 'viewer'
rememberMe: boolean

// Methods
login(username, password, rememberMe)
logout()
refreshToken() // called automatically before API requests
hasRole(requiredRole) // admin > operator > viewer hierarchy
canAccess(feature) // admin-only, operator+, all
```

**Token Storage:**
- If rememberMe: localStorage key `arcvault_jwt`
- If not rememberMe: variable `currentToken` (memory only)
- Flag `arcvault_remember_me` stored in localStorage to check on mount

**Auto-Refresh Trigger:**
- Wrap all API calls in a interceptor
- Before each request, check if token expires < 1 hour
- If yes, refresh silently
- If refresh fails, redirect to /login

---

## 2. Role-Based Access Control

### 2.1 Three Roles

| Role | Agents | Jobs | History | Templates | Users | Groups | Admin |
|------|--------|------|---------|-----------|-------|--------|-------|
| **Admin** | ✅ | ✅ | ✅ | ✅ CRUD | ✅ CRUD | ✅ CRUD | ✅ |
| **Operator** | ✅ | ✅ | ✅ | ✅ RO | ❌ | ❌ | ❌ |
| **Viewer** | ✅ RO | ✅ RO | ✅ RO | ✅ RO | ❌ | ❌ | ❌ |

**Abbreviations:**
- ✅ = Full access
- ✅ RO = Read-only
- ❌ = No access

### 2.2 UI Enforcement

**Navigation:**
- All nav items visible for all roles
- Non-accessible items appear disabled (grayed, not clickable)
- Hover tooltip: "Requires admin role" or "Admin feature"

**Routes:**
- Protected routes use AuthGuard wrapper
- Non-authenticated: Redirect to /login
- Insufficient role: Show error view "403 Forbidden"

**Button/Form Disabling:**
- Create buttons disabled if role < required
- Edit/Delete buttons disabled if role < required
- Entire view disabled if viewer trying to access operator-only feature

**Example: Users.vue for operator:**
- Table visible, shows all users
- "Create User" button disabled with tooltip
- Delete/Edit buttons disabled

---

## 3. New Components

### 3.1 Login.vue

**Purpose:** Authenticate user with username/password, obtain JWT token

**Template:**
```vue
<form @submit.prevent="handleLogin">
  <input v-model="username" type="text" placeholder="Username" />
  <input v-model="password" type="password" placeholder="Password" />
  <label>
    <input v-model="rememberMe" type="checkbox" />
    Remember me
  </label>
  <button type="submit" :disabled="loading">Login</button>
  <div v-if="error" class="error">{{ error }}</div>
</form>
```

**Logic:**
- Form fields: username, password, rememberMe checkbox
- POST /api/auth/login with credentials
- Store JWT + user info in auth composable
- Redirect to /agents on success
- Show error message on failure (invalid credentials, server error)
- Check if already authenticated on mount → redirect to /agents

**Styling:** Match existing ArcVault dark/light theme

---

### 3.2 ChangePassword.vue

**Purpose:** Allow users to change their password

**Can be:**
- Modal (triggered from nav menu)
- Or dedicated view route /change-password

**Recommended: Modal** (less disruptive)

**Template:**
```vue
<div class="modal">
  <h2>Change Password</h2>
  <form @submit.prevent="handleChangePassword">
    <input v-model="currentPassword" type="password" placeholder="Current password" />
    <input v-model="newPassword" type="password" placeholder="New password" />
    <input v-model="confirmPassword" type="password" placeholder="Confirm password" />
    <div v-if="strength" class="strength-indicator" :class="strength">
      {{ strength }}
    </div>
    <button type="submit" :disabled="loading">Change Password</button>
    <button @click="closeModal">Cancel</button>
  </form>
</div>
```

**Logic:**
- Current password validation (required)
- New password strength check (min 8 chars, mix of uppercase/numbers recommended)
- Confirm password match
- POST /api/auth/change-password
- Show success message on completion
- Close modal and refresh

---

### 3.3 Users.vue

**Purpose:** Admin panel for user management (CRUD)

**Features:**
- List all users in table (paginated)
- Columns: ID, Username, Role, Created At, Actions
- Create button → modal with username, password, role dropdown
- Edit button → modal to change role or reset password
- Delete button → confirmation modal

**Table:**
```vue
<table>
  <tr>
    <th>ID</th><th>Username</th><th>Role</th><th>Created</th><th>Actions</th>
  </tr>
  <tr v-for="user in users">
    <td>{{ user.id }}</td>
    <td>{{ user.username }}</td>
    <td>{{ user.role }}</td>
    <td>{{ user.created_at }}</td>
    <td>
      <button @click="editUser(user)" :disabled="!isAdmin">Edit</button>
      <button @click="deleteUser(user)" :disabled="!isAdmin">Delete</button>
    </td>
  </tr>
</table>
```

**Modals:**
- Create: username (text), password (password), role (dropdown: admin/operator/viewer)
- Edit: role dropdown, "Reset Password" button (sends reset link or generates temp password)
- Delete: "Are you sure?" confirmation

**API Calls:**
- GET /api/users → list
- POST /api/users → create
- PUT /api/users/{id} → update role
- DELETE /api/users/{id} → delete

---

### 3.4 Groups.vue

**Purpose:** Admin panel for agent group management (CRUD + membership)

**Features:**
- List groups in table
- Columns: ID, Name, Description, Member Count, Actions
- Create button → modal
- Edit button → modal
- Members button → sub-view (add/remove agents)
- Delete button → confirmation

**Main Table:**
```vue
<table>
  <tr>
    <th>ID</th><th>Name</th><th>Description</th><th>Members</th><th>Actions</th>
  </tr>
  <tr v-for="group in groups">
    <td>{{ group.id }}</td>
    <td>{{ group.name }}</td>
    <td>{{ group.description }}</td>
    <td>{{ group.member_count }}</td>
    <td>
      <button @click="manageMembers(group)">Members</button>
      <button @click="editGroup(group)">Edit</button>
      <button @click="deleteGroup(group)">Delete</button>
    </td>
  </tr>
</table>
```

**Modals:**
- Create: name (text), description (text)
- Edit: name, description
- Members sub-view:
  - List agents in group (with remove button)
  - Dropdown to select agents not in group
  - "Add Agent" button

**API Calls:**
- GET /api/groups → list
- POST /api/groups → create
- PUT /api/groups/{id} → update
- DELETE /api/groups/{id} → delete
- GET /api/groups/{id}/agents → get members
- POST /api/groups/{id}/agents → add agent
- DELETE /api/groups/{id}/agents/{agentId} → remove agent

---

### 3.5 AuthGuard.vue

**Purpose:** Route protection component for authenticated/role-restricted routes

**Usage:**
```vue
<AuthGuard :requiredRole="admin">
  <Users />
</AuthGuard>
```

**Logic:**
- Check if user is authenticated (has valid JWT)
- If not: Redirect to /login
- If yes but insufficient role: Show error view
- If yes and sufficient role: Render wrapped component

**Hierarchy:** admin > operator > viewer (admin can access anything, viewer has most restrictions)

---

## 4. Integration with Existing Components

### 4.1 App.vue Changes

**Remove current token-gate:**
```vue
<!-- REMOVE -->
<div v-if="!tokenSet" class="token-gate">
  ...
</div>

<!-- REPLACE WITH -->
<Login v-if="!isAuthenticated" />
```

**Update nav for RBAC:**
```vue
<nav>
  <router-link to="/agents">Agents</router-link>
  <router-link to="/jobs">Jobs</router-link>
  <router-link to="/history">History</router-link>
  <router-link to="/templates">Templates</router-link>
  <router-link 
    v-if="isAdmin"
    to="/users"
    :class="{ disabled: !isAdmin }"
  >Users</router-link>
  <router-link 
    v-if="isAdmin"
    to="/groups"
    :class="{ disabled: !isAdmin }"
  >Groups</router-link>
</nav>
```

**Add user menu:**
```vue
<div class="nav-right">
  <span>{{ currentUser.username }} ({{ currentUser.role }})</span>
  <button @click="showChangePasswordModal = true">Change Password</button>
  <button @click="logout">Logout</button>
</div>
```

**Remove old admin token logic:**
- Replace tokenSet logic with isAuthenticated from useAuth
- Remove saveToken, hasToken functions (now in login form)

### 4.2 Router Changes

**Add new routes:**
```javascript
const routes = [
  { path: '/', redirect: '/agents' },
  { path: '/login', component: Login },
  
  // Protected routes
  { path: '/agents', component: Agents },
  { path: '/jobs', component: Jobs },
  { path: '/history', component: History },
  { path: '/templates', component: Templates },
  { path: '/federation', component: Federation },
  
  // Admin routes
  { path: '/users', component: Users, meta: { requiresRole: 'admin' } },
  { path: '/groups', component: Groups, meta: { requiresRole: 'admin' } },
  { path: '/change-password', component: ChangePassword },
]
```

**Add router guards:**
```javascript
router.beforeEach((to, from, next) => {
  const { isAuthenticated, userRole } = useAuth()
  
  if (to.path === '/login') {
    next() // login page always accessible
    return
  }
  
  if (!isAuthenticated) {
    next('/login')
    return
  }
  
  if (to.meta.requiresRole && !canAccess(to.meta.requiresRole)) {
    next('/forbidden')
    return
  }
  
  next()
})
```

### 4.3 Jobs.vue Changes

**Smart job creation form:**

Current: Single agent dispatch
```vue
<form @submit="createJob">
  <input v-model="job.agent_id" placeholder="Agent ID" />
  <input v-model="job.name" placeholder="Job name" />
  ...
  <button @click="dispatchJob">Create Job</button>
</form>
```

**New: Smart form with group support**
```vue
<form @submit="createJob">
  <div class="dispatch-mode">
    <label>
      <input v-model="dispatchMode" type="radio" value="agent" />
      Single Agent
    </label>
    <label>
      <input v-model="dispatchMode" type="radio" value="group" />
      Group Dispatch
    </label>
  </div>
  
  <template v-if="dispatchMode === 'agent'">
    <input v-model="job.agent_id" placeholder="Select agent" list="agents" />
  </template>
  
  <template v-if="dispatchMode === 'group'">
    <select v-model="job.group_id">
      <option value="">Select group</option>
      <option v-for="g in groups" :value="g.id">{{ g.name }}</option>
    </select>
    <div v-if="selectedGroup" class="group-preview">
      <p>Will create {{ selectedGroup.member_count }} jobs for:</p>
      <ul>
        <li v-for="agent in groupMembers">{{ agent }}</li>
      </ul>
    </div>
  </template>
  
  <input v-model="job.name" placeholder="Job name" />
  <input v-model="job.source_path" placeholder="Source path" />
  <input v-model="job.dest_path" placeholder="Destination path" />
  
  <button @click="dispatchJob" :disabled="!canCreateJob">
    {{ dispatchMode === 'group' ? 'Dispatch to Group' : 'Create Job' }}
  </button>
</form>
```

**Logic:**
- dispatchMode ref: 'agent' or 'group'
- When mode changes, clear the other field
- When group selected, fetch/show group members preview
- Button text changes: "Create Job" vs "Dispatch to Group"
- API call uses agent_id OR group_id based on mode

**API integration:**
```javascript
async function dispatchJob() {
  if (dispatchMode.value === 'agent') {
    await createJob({ agent_id: job.agent_id, ... })
  } else {
    await createJob({ group_id: job.group_id, ... })
  }
  // Response is either single job or batch { dispatch_id, group_id, jobs[] }
}
```

### 4.4 Agents.vue & History.vue

**Optional enhancements (can defer if needed):**
- Add group filter dropdown (if operator+ has permission)
- Display agent's group membership in table
- Filter history by group

**Not critical for Phase 15 MVP** — basic functionality works without these.

---

## 5. API Requirements

### 5.1 Authentication Endpoints

**POST /api/auth/login**
- Request: `{ username, password }`
- Response: `{ token, user: { id, username, role } }`
- Status: 200 or 401

**POST /api/auth/logout**
- Request: (empty)
- Response: 204 No Content
- Status: 204 or 401

**POST /api/auth/change-password**
- Request: `{ current_password, new_password }`
- Response: 204 No Content
- Status: 204 or 400/401

### 5.2 User Management Endpoints

**GET /api/users**
- Response: `{ items: [{ id, username, role, created_at }], total, page, limit }`

**POST /api/users**
- Request: `{ username, password, role }`
- Response: `{ id, username, role, created_at }`

**PUT /api/users/{id}**
- Request: `{ role }` or `{ password }`
- Response: 204 No Content

**DELETE /api/users/{id}**
- Response: 204 No Content

### 5.3 Group Management Endpoints

**GET /api/groups**
- Response: `{ items: [{ id, name, description, member_count }] }`

**POST /api/groups**
- Request: `{ name, description }`
- Response: `{ id, name, description }`

**PUT /api/groups/{id}**
- Request: `{ name, description }`
- Response: 204 No Content

**DELETE /api/groups/{id}**
- Response: 204 No Content

**GET /api/groups/{id}/agents**
- Response: `{ agents: ["agent-01", "agent-02", ...] }`

**POST /api/groups/{id}/agents**
- Request: `{ agent_id }`
- Response: 204 No Content

**DELETE /api/groups/{id}/agents/{agentId}**
- Response: 204 No Content

### 5.4 Updated Job Endpoints

**POST /api/jobs** (already exists, now supports group_id)
- Request: `{ agent_id: string, name, source_path, dest_path, schedule? }` OR `{ group_id: int, name, source_path, dest_path, schedule? }`
- Response (single): `{ id, agent_id, name, ... }`
- Response (group): `{ dispatch_id, group_id, jobs: [...] }`

---

## 6. Error Handling

### 6.1 Authentication Errors

| Scenario | Status | Display |
|----------|--------|---------|
| Invalid credentials | 401 | "Invalid username or password" |
| User not found | 401 | "Invalid username or password" |
| Token expired | 401 | Auto-refresh; if fails, redirect to login |
| Malformed token | 401 | "Session invalid, please login again" |

### 6.2 Authorization Errors

| Scenario | Status | Display |
|----------|--------|---------|
| Insufficient role | 403 | "This feature requires [role] access" |
| Accessing deleted resource | 404 | "Resource not found" |
| Duplicate username | 400 | "Username already exists" |

### 6.3 Network Errors

- Offline: Show "Connection lost" banner, queue actions until online
- Server error (500): Show error toast, retry button

---

## 7. Testing Strategy

### 7.1 Manual Smoke Tests

**Authentication:**
- [ ] First-time setup with admin token works
- [ ] Login with valid credentials succeeds
- [ ] Login with invalid credentials shows error
- [ ] "Remember me" checked: stays logged in after page refresh
- [ ] "Remember me" unchecked: logged out after page refresh
- [ ] Token auto-refresh works silently
- [ ] Logout clears token and redirects to login

**RBAC:**
- [ ] Admin sees all nav items, can click all
- [ ] Operator sees Users/Groups items but disabled
- [ ] Viewer sees Users/Groups items but disabled, can't access agents/jobs
- [ ] Direct URL to /users as non-admin shows 403
- [ ] Admin can create/edit/delete users
- [ ] Admin can create/edit/delete groups

**Jobs:**
- [ ] Create single-agent job works
- [ ] Create group-dispatch job works
- [ ] Group member preview displays correctly
- [ ] Batch dispatch creates N jobs with shared dispatch_id

**ChangePassword:**
- [ ] Modal opens from nav menu
- [ ] Current password validation works
- [ ] New password strength indicator shows
- [ ] Password change succeeds and logs user out

### 7.2 Automated Testing (Future)

- Unit tests for useAuth composable
- Component tests for Login.vue (valid/invalid submit)
- Integration tests for auth flow (login → access protected route → logout)

---

## 8. File Structure

```
dashboard/src/
├── api.js (updated with auth/user/group endpoints)
├── composables/
│   ├── useAuth.js (NEW - auth state + methods)
│   └── useWebSocket.js (existing)
├── router/
│   └── index.js (updated with new routes + guards)
├── components/
│   ├── ChangePasswordModal.vue (NEW)
│   └── (existing: UpdateBanner, UpdateModal, etc.)
├── views/
│   ├── Login.vue (NEW)
│   ├── Users.vue (NEW)
│   ├── Groups.vue (NEW)
│   ├── Agents.vue (updated with group filters - optional)
│   ├── Jobs.vue (updated with smart form)
│   ├── History.vue (updated with group filters - optional)
│   └── (existing)
└── App.vue (updated - remove token-gate, add nav links)
```

---

## 9. Rollout Plan

**Task 9:** Create useAuth composable + Login.vue  
**Task 10:** ChangePassword.vue (modal)  
**Task 11:** Users.vue (full CRUD)  
**Task 12:** Groups.vue (full CRUD)  
**Task 13:** AuthGuard.vue + Router guards + Update App.vue/Jobs.vue  

Each task includes tests and verification.

---

## 10. Open Questions / Decisions Made

**Decision: JWT in localStorage with remember-me**
- Why: Balances convenience (stay logged in) and security (opt-in)
- Alternative considered: Memory-only (more secure but requires re-login on refresh)

**Decision: Visible but disabled UI**
- Why: Users understand system, UX is clear, simpler code
- Alternative considered: Hidden UI (cleaner for restricted users, more complex)

**Decision: Smart form for job dispatch**
- Why: Single, elegant form, auto-switches modes based on selection
- Alternative considered: Separate buttons (simpler but more UI clutter)

**Decision: Full CRUD for Users/Groups**
- Why: Complete admin experience, no backend-only operations
- Alternative considered: Create/read only (simpler but less flexible)

---

## Success Criteria

- ✅ Admin can login with username/password
- ✅ Users persist sessions with "remember me"
- ✅ All three roles can login and see role-appropriate UI
- ✅ Admin can manage users and groups
- ✅ Operator can create jobs (single + group dispatch)
- ✅ Viewer can view dashboards but not modify
- ✅ All new components tested and working
- ✅ Token auto-refresh works seamlessly


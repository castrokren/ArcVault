# Phase 15 Frontend RBAC — Implementation Plan

**Date:** 2026-05-22  
**Scope:** Tasks 9-13 (Frontend components + integration)  
**Estimate:** 3-4 days (dependent on testing)  
**Owner:** Claude + Kren Castro

---

## Task Breakdown

### Task 9: useAuth Composable + Login.vue

**Objective:** Create authentication state management and login interface

**Files to Create:**
- `dashboard/src/composables/useAuth.js` — Auth state + methods
- `dashboard/src/views/Login.vue` — Login form view

**Steps:**

1. **Create useAuth.js composable**
   - Export ref: `currentUser` (null initially)
   - Export ref: `isAuthenticated` (boolean)
   - Export computed: `userRole` (from currentUser)
   - Export method: `login(username, password, rememberMe)`
     - POST /api/auth/login with credentials
     - On success: save token (localStorage if rememberMe, memory if not)
     - Save rememberMe flag to localStorage
     - Set currentUser from response
     - Return token
   - Export method: `logout()`
     - Clear token from storage
     - Clear currentUser
     - POST /api/auth/logout
   - Export method: `refreshToken()` (silent, called before API requests)
     - Check if token exists and not expired
     - If < 1hr to expiry: POST /api/auth/refresh
     - Update token in storage
   - Export method: `hasRole(requiredRole)` — hierarchy check (admin > operator > viewer)
   - Export method: `canAccess(feature)` — map features to required roles
   - Add composable initialization:
     - On app mount: check localStorage for rememberMe flag
     - If true and token exists: load user from localStorage
     - Verify token validity, refresh if needed

2. **Create Login.vue component**
   - Template:
     - Form with username, password inputs
     - "Remember me" checkbox
     - Submit button (disabled while loading)
     - Error message div
   - Script:
     - Import useAuth, useRouter
     - Refs: username, password, rememberMe, loading, error
     - Method handleLogin:
       - Validate inputs (non-empty)
       - Call auth.login()
       - On success: redirect to /agents
       - On error: display error message
     - Mounted hook: if already authenticated, redirect to /agents
   - Style: Match existing ArcVault dark/light theme

3. **Update api.js**
   - Add POST /api/auth/login endpoint wrapper
   - Add POST /api/auth/logout endpoint wrapper
   - Add POST /api/auth/refresh endpoint wrapper
   - Add POST /api/auth/change-password endpoint wrapper
   - Update request() function to:
     - Call useAuth().refreshToken() before each request
     - Auto-retry on 401 after refresh
     - Handle 401 → redirect to login if refresh fails

4. **Test Login.vue**
   - [ ] Form renders with username, password, checkbox
   - [ ] Submit with valid credentials redirects to /agents
   - [ ] Submit with invalid credentials shows error message
   - [ ] Clicking Remember Me is optional
   - [ ] If already authenticated on mount, redirect to /agents
   - [ ] Error message is user-friendly

---

### Task 10: ChangePassword.vue (Modal)

**Objective:** Create password change interface

**Files to Create:**
- `dashboard/src/components/ChangePasswordModal.vue` — Password change modal

**Steps:**

1. **Create ChangePasswordModal.vue**
   - Template:
     - Modal container (backdrop + dialog)
     - Title: "Change Password"
     - Form with three inputs: currentPassword, newPassword, confirmPassword
     - Password strength indicator (show below newPassword input)
     - Buttons: "Change Password" (submit), "Cancel" (close)
     - Error/success messages
   - Script:
     - Props: `isOpen` (boolean), emit events: `@close`, `@success`
     - Refs: currentPassword, newPassword, confirmPassword, loading, error, success
     - Computed: `passwordStrength` — check newPassword:
       - Red: < 8 chars
       - Yellow: 8+ chars, no uppercase/numbers
       - Green: 8+ chars, has uppercase/numbers
     - Method handleChangePassword:
       - Validate all three fields non-empty
       - Validate newPassword === confirmPassword
       - POST /api/auth/change-password
       - On success: show success message, close modal after 1s
       - On error: show error message
       - Reset form after close
   - Style: Modal styles, strength indicator colors

2. **Update App.vue**
   - Import ChangePasswordModal
   - Add ref: `showChangePasswordModal`
   - Add user menu in nav:
     - Show current username + role
     - Button: "Change Password" → toggles modal
     - Button: "Logout" → calls auth.logout()
   - Add modal component:
     - `<ChangePasswordModal :isOpen="showChangePasswordModal" @close="showChangePasswordModal = false" />`

3. **Test ChangePasswordModal**
   - [ ] Modal opens/closes correctly
   - [ ] Password strength indicator updates as user types
   - [ ] Form validates (empty fields, password mismatch)
   - [ ] Valid submission calls API and shows success
   - [ ] Invalid submission shows error message
   - [ ] Can cancel and close without change

---

### Task 11: Users.vue (Full CRUD)

**Objective:** Create user management admin panel

**Files to Create:**
- `dashboard/src/views/Users.vue` — User management view

**Steps:**

1. **Create Users.vue**
   - Template:
     - Page title: "Users"
     - Button: "+ Create User" (opens Create modal)
     - Table: ID, Username, Role, Created, Actions (Edit, Delete)
     - Pagination if > 25 users
   - Script:
     - Import useAuth, api functions
     - Ref: users, loading, error, showCreateModal, editingUser
     - Computed: isAdmin (from useAuth)
     - Methods:
       - loadUsers(): GET /api/users, set users ref
       - createUser(username, password, role): POST /api/users, reload list
       - updateUser(id, role): PUT /api/users/{id}, reload list
       - deleteUser(id): DELETE /api/users/{id}, reload list, confirmation modal
     - Mounted: loadUsers()

2. **Create modals (template includes)**
   - Create User Modal:
     - Inputs: username (text), password (password), role (dropdown: admin/operator/viewer)
     - Buttons: Create, Cancel
     - Validation: all fields required
     - Error display
   - Edit User Modal:
     - Show username (read-only)
     - Dropdown: change role
     - Button: "Reset Password" (trigger password reset - backend determines logic)
     - Buttons: Update, Cancel
   - Delete Confirmation:
     - "Are you sure you want to delete [username]?"
     - Buttons: Confirm, Cancel

3. **Update api.js**
   - Add GET /api/users
   - Add POST /api/users
   - Add PUT /api/users/{id}
   - Add DELETE /api/users/{id}

4. **Test Users.vue**
   - [ ] Load users on mount
   - [ ] Create user: form validates, API called, list updates
   - [ ] Edit user: role changes, list updates
   - [ ] Delete user: confirmation, API called, removed from list
   - [ ] Pagination works if many users
   - [ ] Error handling for failed API calls

---

### Task 12: Groups.vue (Full CRUD + Membership)

**Objective:** Create agent group management admin panel

**Files to Create:**
- `dashboard/src/views/Groups.vue` — Group management view

**Steps:**

1. **Create Groups.vue**
   - Template:
     - Page title: "Groups"
     - Button: "+ Create Group" (opens Create modal)
     - Table: ID, Name, Description, Members, Actions (Members, Edit, Delete)
   - Script:
     - Ref: groups, agents, loading, error, showCreateModal, selectedGroup, groupMembers
     - Methods:
       - loadGroups(): GET /api/groups
       - loadAgents(): GET /api/agents
       - createGroup(name, description): POST /api/groups, reload
       - updateGroup(id, name, description): PUT /api/groups/{id}, reload
       - deleteGroup(id): DELETE /api/groups/{id}, confirmation
       - manageMembers(group): load group members, show members modal
       - addAgentToGroup(groupId, agentId): POST /api/groups/{id}/agents
       - removeAgentFromGroup(groupId, agentId): DELETE /api/groups/{id}/agents/{agentId}
     - Mounted: loadGroups(), loadAgents()

2. **Create modals**
   - Create Group Modal:
     - Inputs: name (text), description (text)
     - Validation: name required
     - Buttons: Create, Cancel
   - Edit Group Modal:
     - Inputs: name, description
     - Buttons: Update, Cancel
   - Members Modal:
     - List of current members with Remove buttons
     - Dropdown: "Add agent..." (agents not in group)
     - Button: "Add Agent"
     - Error/success messages
   - Delete Confirmation:
     - "Delete group [name]? This cannot be undone."

3. **Update api.js**
   - Add GET /api/groups
   - Add POST /api/groups
   - Add PUT /api/groups/{id}
   - Add DELETE /api/groups/{id}
   - Add GET /api/groups/{id}/agents
   - Add POST /api/groups/{id}/agents
   - Add DELETE /api/groups/{id}/agents/{agentId}

4. **Test Groups.vue**
   - [ ] Load groups and agents on mount
   - [ ] Create group: validates, API called, appears in list
   - [ ] Edit group: updates and saves
   - [ ] Delete group: confirmation, removes from list
   - [ ] Manage members: add agent to group, remove agent from group
   - [ ] Member preview shows correct count

---

### Task 13: AuthGuard + Router Guards + Integration

**Objective:** Implement route protection and update existing components

**Files to Modify:**
- `dashboard/src/components/AuthGuard.vue` (NEW)
- `dashboard/src/router/index.js` (UPDATE)
- `dashboard/src/App.vue` (UPDATE)
- `dashboard/src/views/Jobs.vue` (UPDATE)
- `dashboard/src/views/Agents.vue` (OPTIONAL - add group filters)
- `dashboard/src/views/History.vue` (OPTIONAL - add group filters)

**Steps:**

1. **Create AuthGuard.vue wrapper component**
   - Props: `requiredRole` (optional: 'admin', 'operator', 'viewer')
   - Template:
     - If not authenticated: render nothing (router guard redirects)
     - If authenticated but insufficient role: render error component "403 Forbidden"
     - If authenticated and sufficient role: render `<slot />`
   - Script:
     - Import useAuth
     - Computed: userRole, canAccess (check hierarchy)

2. **Update router/index.js**
   - Add new routes:
     - GET /login → Login.vue
     - GET /users → Users.vue (meta: requiresRole: 'admin')
     - GET /groups → Groups.vue (meta: requiresRole: 'admin')
   - Add router guards:
     - beforeEach: check authentication, check role, redirect if needed
     - If /login and already authenticated: redirect to /agents
     - If protected route and not authenticated: redirect to /login
     - If admin route and not admin: redirect to /forbidden

3. **Update App.vue**
   - Remove old token-gate div
   - Import Login.vue, useAuth, useRouter
   - Add conditional rendering:
     - If not authenticated: render `<Login />`
     - If authenticated: render nav + main content
   - Update nav:
     - Add Users link (visible but disabled if not admin)
     - Add Groups link (visible but disabled if not admin)
     - Add user menu (username, role, Change Password, Logout)
   - Add ChangePasswordModal (import from components)
   - Remove old token input logic

4. **Update Jobs.vue — Smart form for job dispatch**
   - Replace single agent dispatch with smart form:
     - Radio buttons: "Single Agent" vs "Group Dispatch"
     - If Single Agent:
       - Input/dropdown: agent_id (autocomplete from agents list)
     - If Group Dispatch:
       - Dropdown: group_id (select from groups list)
       - Preview: show group members (fetch via api)
       - Button text: "Dispatch to Group"
     - Always show: name, source_path, dest_path, schedule
   - Update createJob method:
     - If dispatchMode === 'agent': POST with agent_id
     - If dispatchMode === 'group': POST with group_id
     - Response handling:
       - Single agent: single job object
       - Group: { dispatch_id, group_id, jobs: [...] }
     - Show success message with job count

5. **Test all integrations**
   - [ ] Unauthenticated users redirected to /login
   - [ ] Login successful → redirected to /agents
   - [ ] Admin can access /users and /groups
   - [ ] Operator cannot access /users, /groups (disabled + error if direct URL)
   - [ ] Viewer cannot access /users, /groups, /jobs modify (disabled + error)
   - [ ] Single-agent job creation works
   - [ ] Group dispatch creates multiple jobs with shared dispatch_id
   - [ ] Logout clears auth and redirects to /login

---

## Verification Checklist

After all tasks complete:

- [ ] All 5 new components created and tested
- [ ] useAuth composable working (login, logout, refresh, role checks)
- [ ] Router guards protecting routes
- [ ] Users CRUD functional (create, list, edit, delete)
- [ ] Groups CRUD functional (create, list, edit, delete, members)
- [ ] Smart job form working (single agent + group dispatch)
- [ ] Role-based UI disabled/enabled correctly
- [ ] Auth token persists correctly with "Remember me"
- [ ] Token auto-refresh works seamlessly
- [ ] All error cases handled gracefully
- [ ] No console errors or warnings

---

## Definition of Done

- ✅ All components built per design spec
- ✅ All tests passing (manual smoke tests at minimum)
- ✅ Code reviewed for:
  - Consistent style with existing components
  - Proper error handling
  - No security issues
  - Clear, maintainable code
- ✅ Documentation updated (CONTEXT.md, phase notes)
- ✅ Ready for v0.8.0 release testing

---

## Timeline

- **Task 9 (useAuth + Login):** 2-3 hours
- **Task 10 (ChangePassword):** 1 hour
- **Task 11 (Users CRUD):** 2-3 hours
- **Task 12 (Groups CRUD):** 2-3 hours
- **Task 13 (Guards + Integration):** 2-3 hours

**Total:** 9-13 hours (1-2 days of focused work)

---

## Notes

- All components use Vue 3 composition API with `<script setup>`
- Follow existing code patterns from current views (Agents.vue, Jobs.vue, etc.)
- Use existing styling/theme system (dark/light toggle)
- Test on both desktop and mobile if possible
- Commit after each major task completion
- Update phase documentation as work progresses


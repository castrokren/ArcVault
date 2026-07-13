# Implementation Plan: Password Policy + Admin Hardening + Pagination

**Session**: WFS-002
**Goal**: Harden password policy (server + client), fix admin route/UI protection gaps, add pagination safeguards
**Priority**: Medium

---

## Overview

Four workstreams addressing gaps found during the security audit. All are independent.

---

## Workstream A: Password Policy Hardening

### A1: Add password complexity validation to server

**Files**: `coordinator/server/auth.go`, `coordinator/business/users.go`

**Current state**: All 3 Validate() methods only check `len >= 8`.

**Change**: Add a shared `validatePasswordStrength(password string) error` function that checks:

```
- Minimum length: 8 (already done)
- At least one uppercase letter (A-Z)
- At least one lowercase letter (a-z)
- At least one digit (0-9)
- At least one special character (!@#$%^&*()_+-=[]{}|;':\",./<>?~`)
```

Add this function in `auth.go` (as a package-level helper, not a method). Then update:

- `LoginRequest.Validate()` — call it
- `ChangePasswordRequest.Validate()` — call it on both old and new password
- `CreateUserRequest.Validate()` — call it

Also update `business/users.go` `CreateUserInput.Validate()` to match (currently only checks empty string).

**Edge cases**:
- Unicode characters should count as "special" or at least not break the regex
- Leading/trailing whitespace should be rejected or trimmed
- The same validation MUST exist at the business layer (`business/users.go`) so raw service calls are also protected

### A2: Fix client-side strength indicator to be honest

**File**: `dashboard/src/components/ChangePasswordModal.vue`

**Current**: The strength meter is purely cosmetic — does not block weak passwords.

**Change**: Block form submission when strength is not at least "medium" by adding `passwordStrength.value === 'weak'` check in `handleChangePassword`. Show inline error: "Password is too weak. Include uppercase, lowercase, digit, and special character."

Also improve `updateStrength()` to check all character classes:
```js
function updateStrength() {
  const pwd = newPassword.value
  const hasUpper = /[A-Z]/.test(pwd)
  const hasLower = /[a-z]/.test(pwd)
  const hasDigit = /[0-9]/.test(pwd)
  const hasSpecial = /[^A-Za-z0-9]/.test(pwd)
  const classes = [hasUpper, hasLower, hasDigit, hasSpecial].filter(Boolean).length

  if (pwd.length < 8) {
    passwordStrength.value = 'weak'
    strengthLabel.value = 'Weak — too short'
  } else if (classes < 3) {
    passwordStrength.value = 'weak'
    strengthLabel.value = 'Weak — add uppercase, digit, or special characters'
  } else if (classes >= 3 && pwd.length >= 10) {
    passwordStrength.value = 'strong'
    strengthLabel.value = 'Strong'
  } else {
    passwordStrength.value = 'medium'
    strengthLabel.value = 'Medium'
  }
}
```

### A3: Add client-side validation to user creation form

**File**: `dashboard/src/views/Users.vue`

**Current**: Only checks `len >= 8`.

**Change**: Add the same character-class validation before submitting the create-user request. Show inline error message matching the server's.

---

## Workstream B: Admin Route Protection Fix

### B1: Swap JWTMiddleware → adminRoute for user management

**File**: `coordinator/server/server.go` lines 345-348

**Current**:
```go
s.router.HandleFunc("GET /api/users", s.JWTMiddleware(s.handleListUsers))
s.router.HandleFunc("POST /api/users", s.JWTMiddleware(s.handleCreateUser))
s.router.HandleFunc("DELETE /api/users/{id}", s.JWTMiddleware(s.handleDeleteUser))
s.router.HandleFunc("PUT /api/users/{id}/role", s.JWTMiddleware(s.handleUpdateUserRole))
```

**Change**: Replace `s.JWTMiddleware(...)` with `s.adminRoute(...)`.

**Impact**: This adds `RequirePasswordChange` and `RequireRole("admin")` middleware to these routes. The inline `if claims.Role != "admin"` checks inside handlers become redundant — they can be kept as defense-in-depth or removed with a comment.

### B2: Fix PUT vs PATCH mismatch

**File**: `coordinator/server/auth.go` line 811, `coordinator/server/server.go` line 348

**Current**: Route registers as `PUT /api/users/{id}/role` but handler checks `r.Method != http.MethodPatch`.

**Change**: Align them. Either:
- Change route to `PATCH /api/users/{id}/role` and handler check to `http.MethodPatch`, OR
- Change handler to `http.MethodPut` and keep route as `PUT`.

(The route registration in `server.go` should be the source of truth — update the handler's method check to match.)

---

## Workstream C: Admin UI Protection

### C1: Add route-level role guards

**File**: `dashboard/src/router/index.js`

**Current**: The `beforeEach` guard only checks `isAuthenticated`.

**Change**: Add `meta: { requiresRole: 'admin' }` to sensitive routes:
```js
{ path: '/users', component: Users, meta: { requiresRole: 'admin' } },
{ path: '/groups', component: Groups, meta: { requiresRole: 'admin' } },
{ path: '/admin/credentials', component: Credentials, meta: { requiresRole: 'admin' } },
{ path: '/federation', component: Federation, meta: { requiresRole: 'admin' } },
{ path: '/federation/health', component: FederationHealth, meta: { requiresRole: 'admin' } },
{ path: '/alerts', component: Alerts, meta: { requiresRole: 'admin' } },
```

Then update the `beforeEach` guard to check role:
```js
router.beforeEach((to, from, next) => {
  const auth = useAuth()

  if (to.path === '/login') {
    if (auth.isAuthenticated.value) {
      next('/agents')
    } else {
      next()
    }
    return
  }

  if (!auth.isAuthenticated.value) {
    next('/login')
    return
  }

  // Role-based route guard
  if (to.meta.requiresRole && !auth.hasRole(to.meta.requiresRole)) {
    next('/agents')
    return
  }

  next()
})
```

### C2: Add admin check to Credentials.vue

**File**: `dashboard/src/views/admin/Credentials.vue`

**Current**: No role check at all — any authenticated user sees the page.

**Change**: Add an `isAdmin` check to the `mounted()` hook, same pattern as Users.vue:
```js
mounted() {
  if (!this.hasRole('admin')) {
    window.location.hash = '/agents'
    return
  }
  this.fetchCredentials();
},
````

This requires adding a `hasRole` method (or importing the composable). Since this component uses Options API, the simplest approach is to check the role from the auth store/composable.

### C3: Gate Credentials nav link on isAdmin

**File**: `dashboard/src/App.vue` line 25

**Current**: Credentials link is always visible.

**Change**: Add `v-if="isAdmin"` to the Credentials `<router-link>`:
```html
<router-link v-if="isAdmin" to="/admin/credentials">
  <svg ...>...</svg>
  Credentials
</router-link>
```

---

## Workstream D: Pagination Hardening

### D1: Add max page/offset cap

**File**: `coordinator/server/pagination.go`

**Current**: `ParsePagination` caps limit at 100 but page can be arbitrarily large.

**Change**: Add a max offset calculation to prevent expensive queries:
```go
if page < 1 {
    page = 1
}
// Cap page to prevent extremely large offsets
const maxPage = 10000
if page > maxPage {
    page = maxPage
}
```

Also add a constant `MaxLimit = 100` and use it instead of the magic number.

**Rationale**: With max limit 100 and max page 10000, the max offset is 999,900 — still large but bounded. SQLite can handle this with proper indexing. The existing `LIMIT ? OFFSET ?` queries will be bounded.

### D2: Add guard against negative limit values

**File**: `coordinator/server/pagination.go` line 49-51

**Current**: If limit < 1, resets to 25. However, what about integer overflow? `strconv.Atoi` returns 0 on parse error, but extremely large values could overflow int on 32-bit. Add explicit check: if `v < 1`, reset to 25.

(Already handled by current code, but add a comment noting the overflow guard.)

### D3: Consider adding total-count cap

**File**: `coordinator/db/` query functions that return total count

**Observation**: Some `COUNT(*)` queries for total could be expensive on large tables. Consider adding a hint that if count exceeds a threshold, cap the returned total (showing "1000+" instead of exact). This is a performance optimization, not a security fix.

**Deferred**: Mark as N+1 improvement, not required for this pass.

---

## Implementation Order

All workstreams are independent and can be done in any order. Suggested order:

| Step | Task | Files | Effort |
|------|------|-------|--------|
| A1 | Server password complexity validation | `auth.go`, `business/users.go` | Low |
| A2 | Fix client strength indicator | `ChangePasswordModal.vue` | Low |
| A3 | Client user-creation validation | `Users.vue` | Low |
| B1 | Swap JWTMiddleware → adminRoute | `server.go` | Low |
| B2 | Fix PUT vs PATCH mismatch | `auth.go`, `server.go` | Low |
| C1 | Route-level role guards | `router/index.js` | Low |
| C2 | Admin check in Credentials.vue | `Credentials.vue` | Low |
| C3 | Gate Credentials nav link | `App.vue` | Low |
| D1 | Max page cap in pagination | `pagination.go` | Low |

Total effort: **Low** (all tasks are small, well-scoped changes)

---

## Verification

After implementation:
1. `go build ./coordinator/...` — must compile cleanly
2. `go test ./coordinator/...` — existing tests must pass
3. Manual: Login with `Password1` should be rejected with complexity error
4. Manual: Navigate to `/users` as a viewer → should redirect to `/agents`
5. Manual: `GET /api/users` with viewer JWT → should return 403
6. Manual: `GET /api/jobs?page=9999999` → should cap to page 10000

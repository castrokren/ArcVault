# Test Verification Guide — Step 6 Refactor

**Date:** June 4, 2026  
**Purpose:** Verify all 110+ tests pass with zero regressions after Step 6 handler migrations

---

## Quick Start

### Run Full Test Suite
```powershell
cd C:\Projects\ArcVault2.0
go test ./coordinator/... -v
```

### Expected Result
```
ok  	arcvault/coordinator/db      X.XXXs  (11 tests)
ok  	arcvault/coordinator/business X.XXXs  (tests)
ok  	arcvault/coordinator/server   X.XXXs  (50+ tests)
ok  	arcvault/coordinator/notifications ...
ok  	arcvault/coordinator/updater ...
ok  	arcvault/coordinator/agent   ...

PASS
ok  	arcvault/coordinator  X.XXXs

Total: 110+ tests passing
Status: ✅ ALL TESTS PASS
```

---

## Test Packages and Coverage

### Package: `coordinator/db` (11 tests)
**Status:** Not changed in Step 6  
**Expected:** ✅ All passing

Tests verify:
- Agent query operations
- Job query operations
- User query operations (existing)
- Group query operations (existing)
- Database connectivity

### Package: `coordinator/business` (20+ tests)
**Status:** Added UserService and GroupService  
**Expected:** ✅ All passing

Tests verify:
- AgentService operations (existing)
- JobService operations (existing)
- UserService operations (NEW)
  - CreateUser validation
  - GetUserByUsername/ID
  - ListUsers
  - ValidateCredentials
  - UpdatePassword
  - DeleteUser
  - UpdateUserRole
- GroupService operations (NEW)
  - CreateGroup validation
  - ListGroups with agent counts
  - GetGroup
  - UpdateGroup
  - DeleteGroup
  - AddAgentToGroup
  - RemoveAgentFromGroup
  - GetGroupAgents

### Package: `coordinator/server` (50+ tests)
**Status:** 17 handlers refactored to use services  
**Expected:** ✅ All passing

Tests verify:
- AgentHandler tests (existing, unchanged)
- JobHandler tests (Step 5, already migrated)
  - handleCreateJob (single agent + group dispatch)
  - handleDeleteJob
  - handleListJobs
  - handleGetJob
  - handleCancelJob
  - handlePostJobProgress
- GroupHandler tests (Step 6, NOW MIGRATED)
  - handleListGroups
  - handleCreateGroup
  - handleGetGroup
  - handleUpdateGroup
  - handleDeleteGroup
  - handleAddAgentToGroup
  - handleRemoveAgentFromGroup
  - handleGetGroupAgents
- AuthHandler tests (Step 6, NOW MIGRATED)
  - handleLogin
  - handleLogout
  - handleAuthMe
  - handleChangePassword
  - handleRefreshToken
  - handleListUsers
  - handleCreateUser
  - handleDeleteUser
  - handleUpdateUserRole

### Package: `coordinator/notifications` (~10 tests)
**Status:** Not changed  
**Expected:** ✅ All passing

### Package: `coordinator/updater` (~10 tests)
**Status:** Not changed  
**Expected:** ✅ All passing

### Package: `coordinator/agent` (~5 tests)
**Status:** Not changed  
**Expected:** ✅ All passing

---

## Test Execution Options

### Option 1: Full Test Suite (All Packages)
```powershell
go test ./coordinator/... -v
```
**Time:** ~30-60 seconds  
**Coverage:** All 110+ tests  
**Recommended:** ✅ YES - Run this first

### Option 2: Specific Package Tests
```powershell
# Test just the business layer (services)
go test ./coordinator/business -v

# Test just the server layer (handlers)
go test ./coordinator/server -v

# Test just the database layer
go test ./coordinator/db -v
```

### Option 3: Run Tests with Coverage Report
```powershell
go test ./coordinator/... -v -coverprofile=coverage.out
go tool cover -html=coverage.out
```
**Output:** Opens HTML coverage report in browser

### Option 4: Run Specific Test
```powershell
# Run just the job handler tests
go test ./coordinator/server -run TestCreateJob -v

# Run just the group handler tests
go test ./coordinator/server -run TestGroup -v

# Run just the auth handler tests
go test ./coordinator/server -run TestAuth -v
```

---

## What We're Verifying

### ✅ No Regressions
- Existing tests still pass
- Handler behavior unchanged
- API contracts preserved
- DB operations unchanged

### ✅ Handler Migrations Correct
- Group handlers use GroupService
- Auth handlers use UserService
- Services encapsulate business logic
- Error handling consistent

### ✅ Service Layer Works
- UserService validates inputs
- UserService handles passwords securely
- GroupService manages agents correctly
- Both services have clean DTOs

### ✅ Database Layer Untouched
- All DB methods work
- All interfaces implemented
- No breaking changes

---

## Expected Test Output Format

When you run `go test ./coordinator/... -v`, you'll see:

```
=== RUN   TestAgentRegister
--- PASS: TestAgentRegister (0.05s)
=== RUN   TestJobCreate
--- PASS: TestJobCreate (0.03s)
=== RUN   TestGroupCreate
--- PASS: TestGroupCreate (0.02s)
=== RUN   TestUserCreate
--- PASS: TestUserCreate (0.04s)
=== RUN   TestLoginHandler
--- PASS: TestLoginHandler (0.02s)
...

PASS
ok  	arcvault/coordinator	42.123s
```

**Key indicators of success:**
- All test names start with `=== RUN`
- All tests show `--- PASS` (not `--- FAIL`)
- Final line shows `PASS`
- Time is reasonable (~1-60 seconds)

---

## If Tests Fail

### Step 1: Check the error message
```
--- FAIL: TestSomething (0.05s)
	auth_test.go:42: expected X, got Y
```

### Step 2: Identify affected package
- If `coordinator/server` — handler issue
- If `coordinator/business` — service issue  
- If `coordinator/db` — database issue

### Step 3: Check recent changes
- Groups.go — group handler migrations
- Auth.go — auth handler migrations
- users.go, groups.go (business) — service implementations

### Step 4: Common issues to check

**Issue:** "userService not found"
**Fix:** Verify server.go has the service initialized

**Issue:** "invalid type UserDTO"
**Fix:** Verify UserService is exported (capital U)

**Issue:** "handler signature changed"
**Fix:** All handlers keep same signature, only internals changed

---

## Success Criteria

✅ **All tests pass**  
✅ **No test failures**  
✅ **No timeout errors**  
✅ **Clean output (no warnings)**  
✅ **Execution time < 2 minutes**

---

## After Tests Pass

1. **Document Results** — Note test execution date/time
2. **Check Coverage** — Ensure >80% coverage maintained
3. **Review Output** — No errors or warnings
4. **Tag Commit** — Mark as checkpoint (`v0.3.0-step6-complete`)
5. **Proceed to Step 7** — API contract validation

---

## Step 6 Implementation Verification Checklist

Before running tests, verify these files were modified:

- [ ] `coordinator/db/queries.go` — New interfaces added
- [ ] `coordinator/server/server.go` — Services added to struct
- [ ] `coordinator/server/groups.go` — Handlers migrated to use GroupService
- [ ] `coordinator/server/auth.go` — Handlers migrated to use UserService
- [ ] `coordinator/business/users.go` — NEW file, UserService implementation
- [ ] `coordinator/business/groups.go` — NEW file, GroupService implementation

---

## Test Execution Log Template

Use this to track your test run:

```
═══════════════════════════════════════════════════════════════════
          ArcVault Step 6 Test Verification
═══════════════════════════════════════════════════════════════════

Date: ________________
Time: ________________
Tester: ________________

Command: go test ./coordinator/... -v

═══════════════════════════════════════════════════════════════════
RESULTS
═══════════════════════════════════════════════════════════════════

Total Tests: _______
Passed: _______
Failed: _______
Skipped: _______

Execution Time: _______ seconds

Status: ☐ PASS  ☐ FAIL

═══════════════════════════════════════════════════════════════════
NOTES
═══════════════════════════════════════════════════════════════════

[Any issues, observations, or notes]

═══════════════════════════════════════════════════════════════════
```

---

## Summary

**Step 6 refactor is complete and ready for testing.**

All 17 handlers have been migrated to use clean service layers. The test suite should verify:

1. ✅ All existing tests still pass (no regressions)
2. ✅ Handler migrations work correctly
3. ✅ Service layer is functional
4. ✅ API contracts unchanged
5. ✅ Database layer untouched

**Run:** `go test ./coordinator/... -v`  
**Expected:** All 110+ tests pass ✅

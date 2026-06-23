# Implementation Plan: Code Review Fixes

**Session ID**: WFS-code-review-fixes  
**Date**: 2026-06-12  
**Review Report**: [.workflow/review-report.md](.workflow/review-report.md)  
**Planning Notes**: [planning-notes.md](planning-notes.md)

---

## Executive Summary

This implementation plan addresses **15 findings** from the code review report across **2 source files** (dashboard/src/api.ts, dashboard/src/views/Users.vue):

- **3 High-severity issues** (CORR-001, SEC-001, READ-001)
- **8 Medium-severity issues** (CORR-002, CORR-003, SEC-002, SEC-003, PERF-001, READ-002, READ-003, READ-004)
- **4 Low-severity issues** (SEC-004, READ-005, READ-006, PERF-002)

**Total Estimated Effort**: ~3.5 hours (210 minutes)  
**Task Groups**: 3 (security/correctness first, then quality, then performance/UX)  
**Risk Level**: LOW (localized changes, minimal cross-component dependencies)

---

## Implementation Groups & Sequencing

### Group 1: Security & Critical Correctness (Must complete first)
**Effort**: 45 minutes | **Priority**: CRITICAL

These fixes address security vulnerabilities and race conditions that can cause data corruption. Must be merged before other fixes.

| Task | Title | File | Est. Time | Dependencies |
|------|-------|------|-----------|--------------|
| IMPL-001 | SEC-001: Sanitize Content-Disposition header filename | dashboard/src/api.ts | 10 min | — |
| IMPL-002 | SEC-002: Consolidate token storage to single key | dashboard/src/api.ts + useAuth.js | 20 min | — |
| IMPL-003 | CORR-001: Fix race condition in pagination | dashboard/src/views/Users.vue | 15 min | — |

**Why First**: 
- SEC-001 prevents path traversal attacks on installer downloads
- SEC-002 prevents stale token state and auth bypass
- CORR-001 prevents data inconsistency from concurrent requests

---

### Group 2: Code Quality & Consistency (After Group 1)
**Effort**: 70 minutes | **Priority**: HIGH

These fixes improve consistency, maintainability, and prevent future bugs. No blocking dependencies on other groups.

| Task | Title | File | Est. Time | Dependencies |
|------|-------|------|-----------|--------------|
| IMPL-004 | READ-001: Standardize function definition patterns | dashboard/src/api.ts | 15 min | IMPL-002 |
| IMPL-005 | READ-002: Extract magic strings to constants | dashboard/src/api.ts | 10 min | IMPL-002 |
| IMPL-006 | READ-003: Add JSDoc to public API functions | dashboard/src/api.ts | 20 min | IMPL-004, IMPL-005 |
| IMPL-007 | CORR-002: Keep modal open on delete error | dashboard/src/views/Users.vue | 15 min | IMPL-003 |
| IMPL-008 | CORR-003: Document endpoint path consistency | dashboard/src/api.ts | 10 min | IMPL-004 |

**Why This Group**: 
- READ-001: Makes the codebase consistent and easier to maintain
- READ-002 + READ-005: Reduces magic strings (enables IMPL-006)
- READ-003: Improves API usability for other developers
- CORR-002: Better error visibility without breaking changes
- CORR-003: Clarifies endpoint strategy

---

### Group 3: Performance & UX Enhancements (After Group 2)
**Effort**: 95 minutes | **Priority**: MEDIUM

These fixes improve performance, user experience, and code structure. Can be merged independently but benefits from context established in earlier groups.

| Task | Title | File | Est. Time | Dependencies |
|------|-------|------|-----------|--------------|
| IMPL-009 | PERF-001: Add response validation to paginated endpoints | dashboard/src/api.ts | 15 min | IMPL-006 |
| IMPL-010 | READ-004: Extract Users component modal logic | dashboard/src/views/Users.vue | 20 min | IMPL-007 |
| IMPL-011 | SEC-003: Add CSRF protection headers (conditional) | dashboard/src/api.ts | 10 min | IMPL-004 |
| IMPL-012 | SEC-004: Replace alert() with inline error display | dashboard/src/views/Users.vue | 10 min | IMPL-007 |
| IMPL-013 | READ-005: Improve variable naming in buildQuery | dashboard/src/api.ts | 5 min | IMPL-005 |

**Why This Group**: 
- PERF-001: Ensures API contract consistency
- READ-004: Reduces component complexity
- SEC-003: Depends on backend verification (conditional)
- SEC-004: Improves UX for error cases
- READ-005: Quick naming improvement

---

## Detailed Task Dependency Graph

```
IMPL-001 (SEC-001: Filename sanitization)
├── No dependencies
│
IMPL-002 (SEC-002: Token consolidation)
├── No dependencies
├── Blocks: IMPL-004, IMPL-005
│
IMPL-003 (CORR-001: Race condition)
├── No dependencies
├── Blocks: IMPL-007
│
IMPL-004 (READ-001: Function patterns) ⬅ Depends on IMPL-002
├── Blocks: IMPL-006, IMPL-008
│
IMPL-005 (READ-002: Magic strings) ⬅ Depends on IMPL-002
├── Blocks: IMPL-006, IMPL-013
│
IMPL-006 (READ-003: JSDoc) ⬅ Depends on IMPL-004, IMPL-005
├── Blocks: IMPL-009
│
IMPL-007 (CORR-002: Error visibility) ⬅ Depends on IMPL-003
├── Blocks: IMPL-010, IMPL-012
│
IMPL-008 (CORR-003: Endpoint docs) ⬅ Depends on IMPL-004
├── No further blocks
│
IMPL-009 (PERF-001: Response validation) ⬅ Depends on IMPL-006
├── No further blocks
│
IMPL-010 (READ-004: Component decomposition) ⬅ Depends on IMPL-007
├── No further blocks
│
IMPL-011 (SEC-003: CSRF headers) ⬅ Depends on IMPL-004
├── No further blocks
├── Note: Conditional on backend verification
│
IMPL-012 (SEC-004: Error display) ⬅ Depends on IMPL-007
├── No further blocks
│
IMPL-013 (READ-005: Variable naming) ⬅ Depends on IMPL-005
├── No further blocks
```

---

## Task Status & Progress Tracking

| Group | Group Name | Status | Progress | Target | Notes |
|-------|-----------|--------|----------|--------|-------|
| 1 | Security & Correctness | Not Started | 0/3 | 45 min | High priority |
| 2 | Code Quality | Not Started | 0/5 | 70 min | After Group 1 |
| 3 | Performance & UX | Not Started | 0/5 | 95 min | After Group 2 |
| — | **TOTAL** | — | **0/13** | **210 min** | **~3.5 hours** |

---

## Files Affected

| File | Issues Count | Severity | Impact |
|------|-------------|----------|--------|
| dashboard/src/api.ts | 9 | 3 High, 5 Medium, 1 Low | Core API client module |
| dashboard/src/views/Users.vue | 4 | 1 High, 2 Medium, 1 Low | User management view |
| dashboard/src/composables/useAuth.js | 1 (linked) | 1 Medium | Token consolidation |

---

## Pre-Implementation Checklist

- [ ] Review IMPL_PLAN.md (this document)
- [ ] Review review-report.md for full context on each finding
- [ ] Verify current test coverage for api.ts and Users.vue
- [ ] Plan verification strategy (unit tests, manual testing, visual inspection)
- [ ] Ensure git branch is clean and up to date with main
- [ ] Set up local environment for testing (npm install, build system ready)

---

## Execution Flow

### Phase 1: Security & Correctness
```
1. Start with IMPL-001 (10 min)
   └─> Review filename sanitization regex
   └─> Run unit tests on downloadInstaller()
   
2. Execute IMPL-002 (20 min)
   └─> Update api.ts and useAuth.js together
   └─> Verify getToken() and saveToken() work correctly
   └─> Check all dependent components still use tokens
   
3. Execute IMPL-003 (15 min)
   └─> Add loading check to pagination watcher
   └─> Test rapid page clicks
   └─> Verify no duplicate requests
```

### Phase 2: Code Quality
```
4. Execute IMPL-004 (15 min)
   └─> Make login() use request() helper like logout()
   └─> Verify all API functions follow same pattern
   
5. Execute IMPL-005 (10 min)
   └─> Create STORAGE_KEYS constant
   └─> Replace all magic strings in api.ts
   
6. Execute IMPL-006 (20 min)
   └─> Add JSDoc to getUsers, createUser, updateUserRole, deleteUser, getAgents, getJobs
   └─> Format according to TSDoc standard
   
7. Execute IMPL-007 (15 min)
   └─> Move error display logic in delete handler
   └─> Keep modal open on error
   └─> Add error display to modal template
   
8. Execute IMPL-008 (10 min)
   └─> Add comments to downloadInstaller and downloadBootstrapScript
   └─> Document why different endpoints
```

### Phase 3: Performance & UX
```
9. Execute IMPL-009 (15 min)
   └─> Add validateResponse to getAgents and getJobs
   └─> Verify schemas exist and match responses
   
10. Execute IMPL-010 (20 min)
    └─> Extract modal logic to useUserModal.ts composable
    └─> Or: inline extract logic with comments if full composable is too heavy
    
11. Execute IMPL-011 (10 min)
    └─> Check backend for CSRF requirements first
    └─> Add X-CSRF-Token header if needed
    
12. Execute IMPL-012 (10 min)
    └─> Remove alert() from Users.vue
    └─> Add installError ref and state management
    
13. Execute IMPL-013 (5 min)
    └─> Rename variables in buildQuery: q->queryString, k->key, v->value
```

---

## Testing Strategy

### Unit Testing
- api.ts: Test request() helper, response validation, error handling, token management
- Users.vue: Test modal state transitions, error visibility, pagination

### Manual Testing
- Test installer download with various filename scenarios
- Test rapid pagination clicks to verify no duplicate requests
- Test login/logout and token persistence
- Test error display in modals (don't close immediately)
- Test CSRF header inclusion (if implemented)

### Verification
- Run `npm run build` to ensure dist/ is correctly generated
- Run existing test suite (if available) to verify no regressions
- Check browser console for any errors or warnings

---

## Rollback Plan

If issues arise:

1. **Revert individual task**: `git revert <commit-hash>`
2. **Revert entire group**: `git revert <first-commit>..<last-commit>`
3. **Hard reset** (if not yet pushed): `git reset --hard origin/main`

All commits should include issue codes (IMPL-001, IMPL-002, etc.) for easy tracking.

---

## Notes & Constraints

### Critical Decisions
1. **Token Consolidation**: Uses single `arcvault_token` key with fallback to `arcvault_jwt` for backwards compatibility
2. **Filename Sanitization**: Whitelist approach (allow only alphanumeric, dash, underscore, dot)
3. **CSRF Protection**: Conditional on backend verification (flagged as MEDIUM because not confirmed required)
4. **Component Decomposition**: Optional full composable extraction; minimal inline refactoring acceptable

### Build System
- Dashboard uses Vite + Vue 3
- Changes to api.ts and Users.vue require `npm run build` in dashboard/
- Built output goes to coordinator/static/dist/index.html
- This file is auto-generated; do not edit directly

### Backward Compatibility
- All changes are backwards compatible
- Token consolidation handles old `arcvault_jwt` key gracefully
- API endpoint paths unchanged; only documentation improved
- No breaking changes to public APIs

---

## Success Criteria

- [ ] All 15 findings addressed in corresponding tasks
- [ ] High-severity issues (3) resolved first and merged before others
- [ ] Each task has acceptance criteria met per its IMPL-XXX JSON
- [ ] No new test failures
- [ ] Manual testing passes for affected features
- [ ] Code review quality maintained or improved
- [ ] No security vulnerabilities introduced

---

## Generated Documentation

This plan includes:
1. **IMPL_PLAN.md** — This document (overview and sequencing)
2. **IMPL-001.json** through **IMPL-013.json** — Individual task specifications
3. **TODO_LIST.md** — Checklist format for tracking progress
4. **plan.json** — Structured metadata

For each task, see the corresponding IMPL-XXX.json file in the `.task/` directory.

---

**Next Step**: Review IMPL-001.json and begin execution of Group 1 (Security & Correctness).

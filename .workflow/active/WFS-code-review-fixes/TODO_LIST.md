# Implementation Task Checklist

**Session**: WFS-code-review-fixes  
**Date Started**: 2026-06-12  
**Total Tasks**: 13  
**Estimated Total Time**: 210 minutes (3.5 hours)

---

## Group 1: Security & Correctness (45 min) — MUST COMPLETE FIRST

These are critical security and correctness fixes that must be merged before other changes.

### Priority: CRITICAL — Start immediately

- [ ] **IMPL-001** — SEC-001: Sanitize Content-Disposition header filename (10 min)
  - File: dashboard/src/api.ts (lines 300-303)
  - Status: Not Started
  - Acceptance: Filename validation with whitelist, path traversal prevention
  - Blocks: IMPL-002, IMPL-003

- [ ] **IMPL-002** — SEC-002: Consolidate token storage to single key (20 min)
  - Files: dashboard/src/api.ts, dashboard/src/composables/useAuth.js
  - Status: Not Started
  - Acceptance: Single arcvault_token key, backwards compatibility with arcvault_jwt
  - Blocks: IMPL-004, IMPL-005

- [ ] **IMPL-003** — CORR-001: Fix race condition in pagination (15 min)
  - File: dashboard/src/views/Users.vue (line 362)
  - Status: Not Started
  - Acceptance: Loading guard on watcher, no concurrent requests
  - Blocks: IMPL-007

### Group 1 Progress

- [ ] All 3 critical tasks completed
- [ ] All tests passing for Group 1
- [ ] Ready to proceed to Group 2

---

## Group 2: Code Quality & Consistency (70 min) — After Group 1

These fixes improve consistency, maintainability, and code structure.

### Priority: HIGH — Start after Group 1 is complete

- [ ] **IMPL-004** — READ-001: Standardize function definition patterns (15 min)
  - File: dashboard/src/api.ts (lines 88-109)
  - Status: Not Started
  - Depends On: IMPL-002
  - Acceptance: login() uses request() helper consistently
  - Blocks: IMPL-006, IMPL-008

- [ ] **IMPL-005** — READ-002: Extract magic strings to constants (10 min)
  - File: dashboard/src/api.ts (lines 25-41, 238-245, 359-363)
  - Status: Not Started
  - Depends On: IMPL-002
  - Acceptance: STORAGE_KEYS constant defined, all magic strings replaced
  - Blocks: IMPL-006, IMPL-013

- [ ] **IMPL-006** — READ-003: Add JSDoc to public API functions (20 min)
  - File: dashboard/src/api.ts (lines 88-204)
  - Status: Not Started
  - Depends On: IMPL-004, IMPL-005
  - Acceptance: All 8 functions documented with @param, @returns
  - Blocks: IMPL-009

- [ ] **IMPL-007** — CORR-002: Keep modal open on delete error (15 min)
  - File: dashboard/src/views/Users.vue (lines 348-358)
  - Status: Not Started
  - Depends On: IMPL-003
  - Acceptance: Modal stays open, error visible inline, user can retry
  - Blocks: IMPL-010, IMPL-012

- [ ] **IMPL-008** — CORR-003: Document endpoint path consistency (10 min)
  - File: dashboard/src/api.ts (lines 283, 323)
  - Status: Not Started
  - Depends On: IMPL-004
  - Acceptance: Comments explain /downloads/ vs /api/admin/ endpoints
  - Blocks: None

### Group 2 Progress

- [ ] All 5 code quality tasks completed
- [ ] All tests passing for Group 2
- [ ] Consistency improved across codebase
- [ ] Ready to proceed to Group 3

---

## Group 3: Performance & UX Enhancements (95 min) — After Group 2

These fixes improve performance, user experience, and code structure.

### Priority: MEDIUM — Start after Group 2 is complete

- [ ] **IMPL-009** — PERF-001: Add response validation to paginated endpoints (15 min)
  - File: dashboard/src/api.ts (lines 149-158)
  - Status: Not Started
  - Depends On: IMPL-006
  - Acceptance: getAgents and getJobs use validateResponse(), schemas imported
  - Blocks: None

- [ ] **IMPL-010** — READ-004: Extract Users component modal logic (20 min)
  - File: dashboard/src/views/Users.vue (lines 243-399)
  - Status: Not Started
  - Depends On: IMPL-007
  - Acceptance: Modal logic extracted or inline, component complexity reduced
  - Blocks: None

- [ ] **IMPL-011** — SEC-003: Add CSRF protection headers (10 min)
  - File: dashboard/src/api.ts (lines 44-66)
  - Status: Not Started
  - Depends On: IMPL-004
  - Acceptance: X-CSRF-Token header added (conditional on backend verification)
  - Blocks: None
  - Note: CONDITIONAL — verify with backend first

- [ ] **IMPL-012** — SEC-004: Replace alert() with inline error display (10 min)
  - File: dashboard/src/views/Users.vue (line 394)
  - Status: Not Started
  - Depends On: IMPL-007
  - Acceptance: alert() removed, inline error display added, better UX
  - Blocks: None

- [ ] **IMPL-013** — READ-005: Improve variable naming in buildQuery (5 min)
  - File: dashboard/src/api.ts (lines 69-75)
  - Status: Not Started
  - Depends On: IMPL-005
  - Acceptance: Variables renamed: q→queryString, k→key, v→value
  - Blocks: None

### Group 3 Progress

- [ ] All 5 performance/UX tasks completed
- [ ] All tests passing for Group 3
- [ ] Code quality and user experience improved
- [ ] Ready for final review and merge

---

## Overall Progress Tracking

### By Group

| Group | Tasks | Completed | Remaining | Est. Time | Status |
|-------|-------|-----------|-----------|-----------|--------|
| 1 | 3 | 0 | 3 | 45 min | Not Started |
| 2 | 5 | 0 | 5 | 70 min | Blocked (waiting for Group 1) |
| 3 | 5 | 0 | 5 | 95 min | Blocked (waiting for Group 2) |
| **Total** | **13** | **0** | **13** | **210 min** | **In Planning** |

### By Severity

| Severity | Count | Completed | Status |
|----------|-------|-----------|--------|
| HIGH | 3 | 0 | CRITICAL — Not Started |
| MEDIUM | 8 | 0 | High Priority — Blocked |
| LOW | 4 | 0 | Nice to Have — Blocked |
| **Total** | **15** | **0** | **Not Started** |

### By Category

| Category | Count | Tasks | Status |
|----------|-------|-------|--------|
| Security | 4 | IMPL-001, 002, 011, 012 | 1 Critical, 3 In Queue |
| Correctness | 3 | IMPL-003, 002, 007 | 2 Critical, 1 In Queue |
| Readability | 6 | IMPL-001, 004, 005, 006, 012, 013 | In Queue |
| Performance | 2 | IMPL-001, 009 | In Queue |

---

## Daily Progress Log

### Day 1 (2026-06-12)

**Time Available**: TBD  
**Target**: Complete Group 1 (Security & Correctness) = 45 min

- [ ] 10:30 AM — Start IMPL-001 (SEC-001: Filename sanitization)
- [ ] 10:40 AM — Complete IMPL-001, begin IMPL-002
- [ ] 11:00 AM — Complete IMPL-002, begin IMPL-003
- [ ] 11:15 AM — Complete IMPL-003
- [ ] 11:15 AM — Group 1 complete! Proceed to Group 2

### Day 2+ (TBD)

**Target**: Complete Groups 2 & 3 = 165 min

- [ ] Group 2 execution (70 min)
- [ ] Group 3 execution (95 min)
- [ ] Final verification and merge

---

## Pre-Implementation Checklist

- [ ] Review IMPL_PLAN.md (complete plan overview)
- [ ] Review review-report.md (original findings and context)
- [ ] Review planning-notes.md (discovery phase findings)
- [ ] Verify local git branch is clean and up to date
- [ ] Set up development environment (npm install, build system)
- [ ] Verify test suite runs successfully before changes
- [ ] Plan verification strategy (tests, manual testing, code review)

---

## Success Criteria — All Tasks

- [ ] All 13 tasks completed with acceptance criteria met
- [ ] All 15 review findings addressed
- [ ] High-severity issues (3) resolved and merged first
- [ ] No new test failures introduced
- [ ] Manual testing passes for affected features
- [ ] Code review quality maintained or improved
- [ ] No regressions in existing functionality
- [ ] Ready for merge to main branch

---

## Verification Steps After Each Group

### After Group 1 — Security & Correctness
- [ ] Run unit tests for api.ts and Users.vue
- [ ] Manual test: pagination with rapid clicks (verify no duplicate requests)
- [ ] Manual test: login/logout (verify tokens stored correctly)
- [ ] Manual test: installer download (verify filename sanitization works)

### After Group 2 — Code Quality
- [ ] Verify all functions follow consistent patterns
- [ ] Check JSDoc renders in IDE (hover over function calls)
- [ ] Verify no magic strings remain in code
- [ ] Manual test: all API functions still work correctly

### After Group 3 — Performance & UX
- [ ] Run full test suite
- [ ] Manual test: error display in modals
- [ ] Manual test: delete error shows inline
- [ ] Verify response validation catches bad responses
- [ ] Check component complexity reduction

---

## Notes

- **Time Estimates**: Each task has estimated time; actual may vary
- **Dependencies**: Follow dependency graph to avoid blocking other tasks
- **Git Strategy**: Create separate commit for each task with issue code (IMPL-001, etc.)
- **Testing**: Run tests after each task, full suite at end of each group
- **Rollback**: Each task can be independently reverted if needed

---

**Next Step**: Begin Group 1 execution. Start with IMPL-001 (SEC-001).

For detailed task specifications, see individual IMPL-XXX.json files in `.task/` directory.

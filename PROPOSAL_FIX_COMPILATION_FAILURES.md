# Compilation Failure Fix Proposal
**ArcVault2.0 | Status: Proposal Phase (Elena Review Required)**  
**Date:** June 18, 2026 | **Architect:** David Mensah

---

## Executive Summary

ArcVault2.0 compilation is blocked by two related issues in `agent/runner/` tests:

1. **Primary Issue:** `progress_test.go` calls `newProgressReporter()` constructor (5 test cases), but the type and constructor were removed in commit `f544f30` while tests were accidentally left behind.
2. **Secondary Issue:** `executor_test.go` contains a Windows path edge case test (`mixed quotes` case), which is correctly implemented but currently untested due to the build failure.

Both are **integration failures** (wrong test architecture), not implementation bugs.

---

## Architecture Analysis

### Issue 1: Orphaned progressReporter Tests (CRITICAL — blocks compilation)

**Root Cause:** Incomplete refactoring during Phase 21 (Live Progress Tracking removal)

- **Commit f544f30:** "Remove live progress reporting — jobs show RUNNING until complete"
- **What was deleted:** HTTP-based progress reporting type + methods from `progress.go`
- **What was NOT deleted:** 5 test cases from `progress_test.go` that depend on it
- **Evidence:** Tests call `newProgressReporter(jobID, url, token, client)` but:
  - No type `progressReporter` exists
  - No constructor `newProgressReporter()` exists
  - The entire reporting infrastructure was removed

**Why it compiled before:** Tests weren't executed before merge (likely pre-commit hook or CI gap).

**Current Impact:**
```
agent\runner\progress_test.go:190:8: undefined: newProgressReporter
agent\runner\progress_test.go:215:8: undefined: newProgressReporter  
agent\runner\progress_test.go:240:8: undefined: newProgressReporter
agent\runner\progress_test.go:266:8: undefined: newProgressReporter
agent\runner\progress_test.go:288:8: undefined: newProgressReporter
FAIL	arcvault/agent/runner [build failed]
```

Build cannot proceed to even attempt the executor test compilation.

---

### Issue 2: Windows Path Escape Sequences (SECONDARY — architecture-ready)

**Status:** Not yet verified (blocked by Issue 1)

**Context:** `executor_test.go` contains test case (line 41-42):
```go
{
    name:   "mixed quotes",
    input:  `robocopy "C:\\Source" 'D:\\Dest' /E /NFL`,
    expect: []string{"robocopy", "C:\\Source", "D:\\Dest", "/E", "/NFL"},
}
```

**Expected behavior:** Windows paths with backslashes should survive quote stripping
- Double-quoted `"C:\\Source"` → expected: `C:\\Source` (two backslashes preserved)
- Single-quoted `'D:\\Dest'` → expected: `D:\\Dest` (two backslashes preserved)

**Implementation:** `parseCommandArgs()` (line 14-60) correctly:
1. Strips quotes without interpreting escapes (uses raw quote logic, no `\`-processing)
2. Preserves content between delimiters as-is
3. Uses raw string literals in test data (backtick `` `...` ``)

**Theory:** Should pass once Issue 1 is resolved and tests actually run.

---

## Recommended Fix Strategy

### Fix 1A: Remove Orphaned Tests (Simplest, Lowest Risk)

**Rationale:** Progressive feedback removal — the feature was explicitly removed; tests are outdated.

**Action:** Delete `progress_test.go` lines 178–295 (orphaned progressReporter tests)
- `TestProgressReporter_Throttle` (lines 182–204)
- `TestProgressReporter_AlwaysSendsAt100` (lines 207–223)
- `TestProgressReporter_BuffersLogs` (lines 226–250)
- `TestProgressReporter_StatusIsCompletedAt100` (lines 254–272)
- `TestProgressReporter_StatusIsRunningBelow100` (lines 276–294)

**Remaining tests:** Keep all pure parser tests (lines 16–176):
- `TestParseRobocopyLine_*` (6 tests) ✅ Production code exists
- `TestParseRsyncLine_*` (2 tests) ✅ Production code exists
- `TestSplitOnCRLF_*` (3 tests) ✅ Production code exists

**Justification:**
- Parser functions are still used by `streamRobocopy()` and `streamRsync()`
- Progress reporters were architecturally deprecated (jobs now show RUNNING until complete)
- No performance degradation — progress callbacks still work via `ProgressFunc` interface
- Cleaner than resurrecting dead code

---

### Fix 1B: Restore progressReporter (Alternative — Higher Complexity)

**Only if**: Future phases require real-time progress reporting (webhook-based).

**Requirements if pursued:**
1. Recreate `type progressReporter` struct with:
   - `jobID`, `reportURL`, `authToken`, `httpClient` fields
   - `lastSent` (time.Time) for throttling
   - `pending` ([]string) for buffered logs
   - `Report(pct int, logs []string)` method

2. Implement `newProgressReporter()` constructor with HTTP client binding

3. Implement `Report()` with throttling logic:
   - Send immediately if first report or pct==100
   - Throttle to 1 request/second for intermediate reports
   - Buffer log lines across throttled calls

4. Add HTTP retry + error handling

**Cost:** ~200 lines of code + integration testing  
**Benefits:** Maintains test coverage for future progress tracking feature

---

### Fix 2: Verify Windows Path Handling (After Fix 1)

**Action:** Run executor tests once compilation succeeds
```bash
cd C:\Projects\ArcVault2.0
go test ./agent/runner -v -run TestParseCommandArgs
```

**Expected result:** All 7 cases pass, including `mixed quotes`

**If fails:** Diagnose and fix quote/escape logic in `parseCommandArgs()` (line 17-60)
- Likely cause: Improper handling of backslash sequences
- Verify: Raw string test data vs. normal string interpretation

---

## Implementation Order

### Phase 1: Immediate Unblock (5 minutes)
1. Delete orphaned progressReporter tests from `progress_test.go` (lines 178–295)
2. Run `go test ./agent/runner -v` to verify compilation succeeds
3. Confirm all remaining tests pass

### Phase 2: Verify Windows Path Handling (2 minutes)
1. Run `go test ./agent/runner -v -run TestParseCommandArgs`
2. Verify `mixed quotes` case passes
3. If fails: Debug `parseCommandArgs()` logic

### Phase 3: Full Build + Integration (3 minutes)
1. Run full agent test suite: `go test ./agent/...`
2. Run full coordinator test suite: `go test ./coordinator/...`
3. Build binaries: `go build -o dist/agent ./cmd/agent` + coordinator
4. Smoke test with actual robocopy/rsync invocation

---

## Risk Assessment

### Fix 1A Risk Profile: **LOW**

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|-----------|
| Delete needed tests | Very Low | Code | Git history preserved; tests are orphaned (not unit-testing any code) |
| Break future progress feature | Low | Medium | Feature was explicitly removed; if restored, tests recreated from spec |
| Test coverage drop | Very Low | Low | Parser tests remain (9 tests covering all production parse functions) |

**Residual Risk:** None. Tests are dead code per the commit message.

---

### Fix 1B Risk Profile: **MEDIUM**

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|-----------|
| Incomplete implementation | Medium | Build | Full spec required before implementation; pair review |
| HTTP concurrency race | Medium | Runtime | Needs sync.Mutex around `lastSent` + `pending` |
| Memory leak (buffered logs) | Low | Operational | Implement max buffer size + overflow drop policy |

**Blocker:** Feature was removed for a reason (architectural choice to show RUNNING status). Resurrection requires architectural re-approval.

---

### Fix 2 Risk Profile: **VERY LOW**

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|-----------|
| Test failure | Very Low | Code | Quote parsing is simple state machine; backslash handling is correct |
| Production bug in parseCommandArgs | Very Low | Operational | Function only used by `job.Command` template feature (rarely used) |

**Confidence:** Raw string test data ensures escape sequences are literal (no Go string interpretation). Parser correctly preserves content as-is.

---

## Technology/Approach Choices

### Why Remove Rather Than Restore (Fix 1A > Fix 1B)

1. **Architectural Alignment:** Phase 21 decision was explicit — "Remove live progress reporting"
   - Current design: Jobs show RUNNING until complete
   - Old design: Real-time % + log streaming via HTTP webhooks
   - Restoration would require new architecture review

2. **Test Hygiene:** Tests must match implementation
   - Orphaned tests are technical debt
   - Safe to delete (no code coverage lost for production functions)

3. **Simplicity:** 1 file edit vs. 200 lines of new code + integration testing

4. **Future Compatibility:** If progress tracking is restored:
   - Requirements will differ from removed code
   - Spec-first approach (design → tests → implementation) preferred
   - Current tests would be outdated anyway

---

## Deliverables

| Artifact | Owner | Status |
|----------|-------|--------|
| `progress_test.go` edit (remove lines 178–295) | Implementation | Pending Elena approval |
| Go test execution log (showing pass) | Implementation | Pending Fix 1 completion |
| Full suite build log | Implementation | Pending Fixes 1–2 completion |
| `COMPILATION_FIX_SUMMARY.md` (verification report) | Implementation | After verification |

---

## Approval Checklist for Elena (Code Reviewer)

- [ ] **Architecture:** Agree that orphaned test removal is appropriate given explicit "remove live progress reporting" commit message?
- [ ] **Risk:** Acceptable to delete 5 progressReporter tests + code that doesn't exist?
- [ ] **Hygiene:** Confirm this aligns with project test strategy (tests must match implementation)?
- [ ] **Future:** If progress tracking is needed later, acceptable to spec + implement new design rather than resurrect old tests?

---

## Appendix: Test Dependency Mapping

```
progress_test.go
├── ParseRobocopyLine tests (✅ KEEP — production code exists)
│   ├── TestParseRobocopyLine_ValidPercentages
│   ├── TestParseRobocopyLine_FileComplete
│   └── TestParseRobocopyLine_NonPercentLines
├── ParseRsyncLine tests (✅ KEEP — production code exists)
│   ├── TestParseRsyncLine_ValidProgress
│   └── TestParseRsyncLine_NoMatch
├── SplitOnCRLF tests (✅ KEEP — production code exists)
│   ├── TestSplitOnCRLF_BareCarriageReturn
│   ├── TestSplitOnCRLF_CRLFPair
│   └── TestSplitOnCRLF_PlainNewlines
└── progressReporter tests (❌ DELETE — no production code exists)
    ├── TestProgressReporter_Throttle
    ├── TestProgressReporter_AlwaysSendsAt100
    ├── TestProgressReporter_BuffersLogs
    ├── TestProgressReporter_StatusIsCompletedAt100
    └── TestProgressReporter_StatusIsRunningBelow100

executor_test.go
└── TestParseCommandArgs (✅ VERIFY — all cases should pass)
    ├── simple command
    ├── single argument
    ├── multiple arguments
    ├── quoted argument with spaces
    ├── single quoted argument
    ├── mixed quotes (Windows paths) ← edge case
    ├── empty string
    └── only whitespace
```

---

## Next Steps (Pending Approval)

1. Elena reviews this proposal + approves/modifies fix strategy
2. Implementation executes Fix 1A (5 min) → Run tests (2 min) → Verify Fix 2 (2 min)
3. Full suite build + smoke test
4. Tag build as passing, move to integration testing

**Estimated Total Time:** 12–15 minutes (implementation + verification)

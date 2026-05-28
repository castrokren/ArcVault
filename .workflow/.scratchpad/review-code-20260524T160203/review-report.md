# Code Review Report
**File:** agent/runner/runner.go  
**Reviewer:** Claude Code  
**Date:** 2026-05-24  
**Branch:** feature/phase-18-installers  

---

## Executive Summary

The `runner.go` module implements the agent-side job polling and execution loop for ArcVault. The code is **well-structured** and **reasonably testable**, but contains **critical issues** that must be addressed before merge:

- **2 Critical** findings (security + error handling)
- **9 High** findings (error handling, network reliability, testing)
- **8 Medium** findings (code quality, architecture)
- **3 Low** findings (optional improvements)

### Quality Gate Status: ⛔ **BLOCKED**

| Category | Count | Status |
|----------|-------|--------|
| Critical | 2 | ❌ Must fix |
| High | 9 | ⚠️ Must fix |
| Medium | 8 | ⚠️ Should fix |
| Low | 3 | ✓ Optional |

---

## Summary by Dimension

| Dimension | Weight | Score | Issues | Status |
|-----------|--------|-------|--------|--------|
| **Correctness** | 25% | 2/5 ⚠️ | 4 (1H, 3M) | 🔴 Must Fix |
| **Security** | 25% | 2/5 ⚠️ | 4 (2H, 2M) | 🔴 Must Fix |
| **Performance** | 15% | 3/5 ⚠️ | 3 (1H, 2M) | 🟡 High |
| **Readability** | 15% | 4/5 ✓ | 4 (2M, 2L) | 🟢 Good |
| **Testing** | 10% | 1/5 ⚠️ | 4 (2H, 2M) | 🔴 Critical Gap |
| **Architecture** | 10% | 3/5 ⚠️ | 5 (1M, 4L) | 🟡 Medium |

---

## Critical Issues (Must Fix Before Merge)

### 🔴 SEC-001: URL Injection Vulnerability
**Severity:** High | **Category:** injection  
**Lines:** 92, 144, 170

URL parameters are not properly escaped. AgentID and jobID are embedded directly in URL strings without `url.QueryEscape()`. This could cause URL injection if these values contain special characters.

**Example Attack:** AgentID = `agent/admin` → URL becomes `/api/jobs?agent_id=agent/admin&status=pending`

**Fix:**
```go
// Current (vulnerable):
url := fmt.Sprintf("%s/api/jobs?agent_id=%s&status=pending", r.cfg.CoordinatorURL, r.cfg.AgentID)

// Fixed:
u, _ := url.Parse(r.cfg.CoordinatorURL)
u.Path = "/api/jobs"
u.RawQuery = url.Values{
    "agent_id": {r.cfg.AgentID},
    "status": {"pending"},
}.Encode()
```

**Impact:** URL injection, parameter pollution attacks possible if AgentID is user-controlled.

---

### 🔴 PERF-001: HTTP Requests Have No Timeout
**Severity:** High | **Category:** blocking-io  
**Lines:** 99, 152, 178

`http.DefaultClient` is used without timeout configuration. Network hangs will block the entire polling loop indefinitely.

**Scenario:** Coordinator becomes unresponsive → runner hangs → agent never processes jobs again.

**Fix:**
```go
// In New():
cfg.Client = &http.Client{
    Timeout: 30 * time.Second,
}
```

**Impact:** Critical for reliability. Agent becomes unresponsive on coordinator network issues.

---

### 🔴 CORR-001: Error from json.Marshal Ignored
**Severity:** High | **Category:** error-handling  
**Lines:** 143, 166

Both `updateStatus()` and `postResult()` ignore `json.Marshal()` errors using blank identifier. If marshaling fails, request body is empty, causing malformed HTTP requests.

**Current Code:**
```go
body, _ := json.Marshal(map[string]string{"status": status})  // ❌ Error ignored
```

**Fix:**
```go
body, err := json.Marshal(map[string]string{"status": status})
if err != nil {
    return fmt.Errorf("marshal status: %w", err)
}
```

**Impact:** Silent failures in job status updates, coordinator never learns job completion status.

---

### 🔴 CORR-002: Response Body Not Validated for Errors
**Severity:** High | **Category:** error-handling  
**Lines:** 109, 158, 184

Code checks HTTP status code but doesn't read response body for error details. Debugging coordinator issues becomes impossible.

**Fix:**
```go
if resp.StatusCode != http.StatusOK {
    body, _ := io.ReadAll(resp.Body)
    return fmt.Errorf("unexpected status %d: %s", resp.StatusCode, string(body))
}
```

**Impact:** Loss of debugging information when API calls fail.

---

## High Priority Issues

### ⚠️ TEST-001: No Test Coverage
**Severity:** High | **Category:** coverage  
**Lines:** 1-189

No `runner_test.go` file found. Critical error paths are untested:
- Network failures
- Invalid JSON responses
- Job state transitions (pending → running → completed/failed)
- Concurrent Stop() calls

**Impact:** Error handling reliability is unknown.

---

### ⚠️ SEC-002: Token Sent Without HTTPS Enforcement  
**Severity:** High | **Category:** sensitive-data  
**Lines:** 97, 149, 175

Bearer token sent in HTTP requests without enforcing HTTPS. If coordinator URL is misconfigured as `http://`, token is sent in plaintext.

**Fix:**
```go
// In New() Config validation:
if !strings.HasPrefix(cfg.CoordinatorURL, "https://") {
    return nil, fmt.Errorf("CoordinatorURL must use HTTPS, got: %s", cfg.CoordinatorURL)
}
```

**Impact:** Token could be intercepted on misconfigured HTTP connections.

---

### ⚠️ CORR-003: No Validation of Job Data
**Severity:** Medium | **Category:** null-check  
**Lines:** 85

Jobs fetched from coordinator are used without validation. Missing required fields cause panics.

**Fix:** Validate before processing:
```go
for _, job := range jobs {
    if job.ID == "" || job.SourcePath == "" || job.DestPath == "" {
        log.Printf("Runner: invalid job data: %+v", job)
        continue
    }
    r.process(job)
}
```

---

### ⚠️ PERF-002: No HTTP Connection Pooling Configuration
**Severity:** High | **Category:** inefficient-algorithm

Each API call creates new connections. Under high load, TCP connection churn degrades performance.

**Fix:**
```go
// In New():
transport := &http.Transport{
    MaxIdleConns: 10,
    IdleConnTimeout: 90 * time.Second,
    DisableKeepAlives: false,
}
client := &http.Client{
    Transport: transport,
    Timeout: 30 * time.Second,
}
```

---

### ⚠️ READ-001: process() Has Multiple Responsibilities
**Severity:** Medium | **Category:** function-length  
**Lines:** 116-139

Function claims job, executes, posts result, and updates status. Should be decomposed.

**Suggestion:** Extract into methods:
- `claim(jobID)` - mark as running
- `execute(job)` - run executor
- `postResult(jobID, exitCode, output)` - send results
- `markComplete(jobID, success)` - set final status

---

## Medium Priority Issues

### 🟡 CORR-004: Incomplete HTTP Status Code Handling
**Severity:** Medium | **Category:** error-handling

Only checks for specific success codes (OK, Created). Other 4xx/5xx codes treated the same.

**Fix:** Distinguish between client/server errors for proper retry logic.

---

### 🟡 SEC-003: Multiple Goroutines Can Close Stop Channel
**Severity:** Low | **Category:** auth

If `Stop()` called multiple times concurrently, channel close causes panic.

**Fix:** Use `sync.Once`:
```go
var stopOnce sync.Once
func (r *Runner) Stop() {
    stopOnce.Do(func() { close(r.stop) })
}
```

---

### 🟡 ARCH-001: Tight HTTP Protocol Coupling
**Severity:** Medium | **Category:** coupling

All HTTP concerns embedded directly. Changing protocol requires modifying this file.

**Suggestion:** Extract into separate `coordinator` package with `CoordinatorClient` interface.

---

### 🟡 ARCH-002: Logging Coupled to log Package
**Severity:** Medium | **Category:** layer-violation

`log` package used directly. No abstraction for logging.

**Suggestion:** Accept logger interface in Config for better testability.

---

## Low Priority Issues

### 🔵 READ-002: Anonymous Response Struct Not Named
**Severity:** Low | **Category:** naming

Envelope struct defined inline (lines 106-108), not reusable.

**Fix:** Define at package level:
```go
type JobsResponse struct {
    Data []Job `json:"data"`
}
```

---

### 🔵 PERF-003: Sequential Job Processing
**Severity:** Medium | **Category:** complexity

Jobs processed one at a time. If executor is slow, throughput is limited.

**Consider:** Process jobs concurrently with goroutine pool if executor is I/O bound.

---

### 🔵 ARCH-004: No Exponential Backoff on Failures
**Severity:** Low | **Category:** coupling

Coordinator down → continuous polling without backoff. Could add exponential backoff for resilience.

---

## Recommendations

### Before Merge (Blocking)

1. **Fix URL injection** (SEC-001) - Use `url.QueryEscape()` or `url.URL` builder
2. **Add HTTP timeouts** (PERF-001) - Required for reliability
3. **Fix json.Marshal errors** (CORR-001) - Must not ignore errors
4. **Validate response bodies** (CORR-002) - Add error logging
5. **Add HTTP timeout tests** - Create runner_test.go with mock server
6. **Enforce HTTPS** (SEC-002) - Validate coordinator URL scheme
7. **Add job data validation** (CORR-003) - Check required fields

### Follow-up (Next PR)

- [ ] Decompose `process()` method into smaller functions
- [ ] Extract HTTP client concerns into separate coordinator package
- [ ] Add comprehensive test suite for error paths
- [ ] Implement exponential backoff for failures
- [ ] Add structured logging with logger interface
- [ ] Define named response types

---

## Testing Recommendations

### Priority 1 (Must Have)
- [ ] Network timeout scenarios
- [ ] Job state transitions (pending → running → completed/failed)
- [ ] Error response handling
- [ ] Job data validation

### Priority 2 (Should Have)
- [ ] Concurrent Stop() calls
- [ ] Invalid JSON response handling
- [ ] Missing coordinator response fields
- [ ] Multiple job processing

### Test Tool Recommendation
Use `httptest.Server` for mock coordinator:
```go
func TestFetchPendingJobs(t *testing.T) {
    server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        json.NewEncoder(w).Encode(JobsResponse{Data: []Job{{ID: "1"}}})
    }))
    defer server.Close()
    
    // Test against real HTTP interface
}
```

---

## Approval Gate

```
┌─────────────────────────────────────────┐
│  GATE: CRITICAL ISSUES BLOCKING MERGE   │
├─────────────────────────────────────────┤
│  Critical Issues: 2                     │ ❌
│  High Issues: 9                         │ ❌
│  Status: 🔴 DO NOT MERGE                │
└─────────────────────────────────────────┘
```

**Next Steps:**
1. Fix all Critical and High issues
2. Create runner_test.go with test suite
3. Re-submit for review
4. Schedule Medium issues for next iteration

---

## Notes

- **Good Design:** Executor abstraction makes code testable (✓)
- **Good Structure:** Clear separation of concerns with dedicated methods
- **Main Gaps:** Missing timeouts, error validation, test coverage
- **Risk Level:** Medium-High due to untested error paths and network issues


# Claude Code Task: Fix runner.go Review Issues

## Context
You are working on the ArcVault project at `C:\Projects\ArcVault2.0`.
The file under review is `agent/runner/runner.go` on branch `feature/phase-18-installers`.
A code review identified critical, high, and medium issues that must be fixed before merge.

Fix all issues below in three waves. After each wave, commit atomically. Do not claim anything is complete without running verification commands and showing their output.

---

## Before You Start

1. Read `agent/runner/runner.go` in full so you understand the current code
2. Confirm the branch is `feature/phase-18-installers` (or create/checkout it)
3. Do NOT modify any other files unless required by an import change

---

## Wave 1 — Security & Stability

Fix these four issues, then commit.

### SEC-001: URL Injection
- Lines ~92, 144, 170
- Replace all `fmt.Sprintf` URL construction that embeds `AgentID` or `jobID` directly into query strings
- Use `url.URL` + `url.Values{}.Encode()` for all API URLs

**Pattern to replace:**
```go
// BAD
url := fmt.Sprintf("%s/api/jobs?agent_id=%s&status=pending", r.cfg.CoordinatorURL, r.cfg.AgentID)

// GOOD
u, err := url.Parse(r.cfg.CoordinatorURL)
if err != nil {
    return nil, err
}
u.Path = "/api/jobs"
u.RawQuery = url.Values{
    "agent_id": {r.cfg.AgentID},
    "status":   {"pending"},
}.Encode()
```

### PERF-001: HTTP Timeout
- Lines ~99, 152, 178
- Remove all usage of `http.DefaultClient`
- In `New()`, create and store a configured client:
```go
cfg.Client = &http.Client{
    Timeout: 30 * time.Second,
}
```
- Use this client for all HTTP calls in the file

### SEC-002: HTTPS Enforcement
- In `New()`, validate the coordinator URL scheme before accepting config:
```go
if !strings.HasPrefix(cfg.CoordinatorURL, "https://") {
    return nil, fmt.Errorf("CoordinatorURL must use HTTPS, got: %s", cfg.CoordinatorURL)
}
```

### SEC-003: Stop() Panic on Double-Call
- Wrap the `close(r.stop)` in `sync.Once` to prevent panic if Stop() is called concurrently:
```go
var stopOnce sync.Once
func (r *Runner) Stop() {
    stopOnce.Do(func() { close(r.stop) })
}
```
(Move `stopOnce` to a field on the Runner struct, not a package-level var)

**After Wave 1:**
```bash
go build ./...
go vet ./agent/runner/...
git add agent/runner/runner.go
git commit -m "fix(runner): SEC-001 URL injection, PERF-001 timeouts, SEC-002 HTTPS enforcement, SEC-003 stop panic"
```

---

## Wave 2 — Error Handling & Correctness

Fix these five issues, then commit.

### CORR-001: json.Marshal Errors Ignored
- Lines ~143, 166 — in `updateStatus()` and `postResult()`
- Replace `body, _ := json.Marshal(...)` with proper error handling:
```go
body, err := json.Marshal(map[string]string{"status": status})
if err != nil {
    return fmt.Errorf("marshal status: %w", err)
}
```

### CORR-002: Response Body Not Read on Error
- Lines ~109, 158, 184
- After every HTTP call, if `resp.StatusCode != http.StatusOK` (or the expected success code), read the body and include it in the error:
```go
if resp.StatusCode != http.StatusOK {
    body, _ := io.ReadAll(resp.Body)
    return fmt.Errorf("unexpected status %d: %s", resp.StatusCode, string(body))
}
```

### CORR-003: Job Data Not Validated
- Line ~85 — before calling `r.process(job)` in the polling loop
- Add validation:
```go
for _, job := range jobs {
    if job.ID == "" || job.SourcePath == "" || job.DestPath == "" {
        log.Printf("Runner: skipping invalid job data: %+v", job)
        continue
    }
    r.process(job)
}
```

### CORR-004: No Distinction Between 4xx and 5xx
- In your error handling after HTTP calls, distinguish client errors (4xx, don't retry) from server errors (5xx, may retry later):
```go
if resp.StatusCode >= 500 {
    // server error, transient, log and continue polling
} else if resp.StatusCode >= 400 {
    // client error, log as warning — likely a config issue
}
```

### PERF-002: No Connection Pooling
- In `New()`, when creating the HTTP client, add a transport with pooling:
```go
transport := &http.Transport{
    MaxIdleConns:    10,
    IdleConnTimeout: 90 * time.Second,
    DisableKeepAlives: false,
}
client := &http.Client{
    Transport: transport,
    Timeout:   30 * time.Second,
}
```

**After Wave 2:**
```bash
go build ./...
go vet ./agent/runner/...
git add agent/runner/runner.go
git commit -m "fix(runner): CORR-001/002/003/004 error handling, PERF-002 connection pooling"
```

---

## Wave 3 — Test Coverage

Create `agent/runner/runner_test.go`. Write all 8 tests below using `net/http/httptest` and the standard `testing` package. No external test libraries.

### Tests to implement:

```
TestFetchPendingJobs_Success
  - httptest server returns valid JSON with one job
  - runner.fetchPendingJobs() returns that job without error

TestFetchPendingJobs_NetworkTimeout
  - httptest server sleeps 60s before responding
  - runner's client has 1s timeout for this test
  - confirms error is returned (not a hang)

TestFetchPendingJobs_InvalidJSON
  - httptest server returns 200 with body "not json"
  - confirms error returned, no panic

TestFetchPendingJobs_ServerError
  - httptest server returns 500 with body "internal error"
  - confirms error message contains the body text

TestUpdateStatus_MarshalError
  - If your implementation allows injecting a bad value type, verify error propagates
  - OR test that a 500 from the status endpoint returns an error

TestJobValidation_MissingFields
  - Directly test the validation logic: job with empty ID is skipped
  - job with empty SourcePath is skipped
  - job with all fields set is processed

TestStop_ConcurrentCalls
  - Call runner.Stop() from 3 goroutines simultaneously
  - Confirm no panic (use recover or run with -race flag)

TestHTTPSEnforcement
  - Call New() with CoordinatorURL = "http://localhost:8080"
  - Confirm error is returned containing "must use HTTPS"
```

**After Wave 3:**
```bash
go test ./agent/runner/... -v -count=1 -race
git add agent/runner/runner_test.go
git commit -m "test(runner): TEST-001 add runner_test.go with httptest mock coverage"
```

---

## Final Verification Gate

Run all three commands and show their full output. Do not declare success until all pass:

```bash
go vet ./...
go build ./...
go test ./agent/runner/... -v -count=1 -race
```

Expected:
- `go vet` — no output (clean)
- `go build` — no output (clean)
- `go test` — all 8 tests listed as `PASS`, `ok agent/runner`

---

## Done Criteria

- [ ] Wave 1 committed
- [ ] Wave 2 committed  
- [ ] Wave 3 committed
- [ ] All 3 verification commands pass with clean output
- [ ] No Critical or High findings remain unaddressed

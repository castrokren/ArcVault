# Proposal: Fix `newTestServer` Compilation Errors

**Date:** 2026-06-18  
**Status:** PROPOSAL (awaiting review)  
**Prepared for:** Elena Vasquez (Code Reviewer)

---

## Executive Summary

Two conflicting `newTestServer` function definitions exist in the `coordinator/server` package, causing compilation failures across 12+ test files. The issue stems from a recent refactoring that introduced a handler-only test variant (`downloads_test.go`) while leaving the legacy integration test helpers unchanged.

**Key Facts:**
- `downloads_test.go`: Defines `newTestServer(t *testing.T, cfg *config.Config)` (2 args)
- `jobs_test.go`: Defines `newTestServer(t *testing.T)` (1 arg)
- 12 test files call with 1 arg; 1 test file calls with 2 args
- Compiler error: "redeclared in this block" + "not enough arguments"

---

## Root Cause Analysis

### Problem 1: Duplicate Declarations

Both `downloads_test.go` and `jobs_test.go` define `newTestServer` at package scope. Go prohibits multiple function declarations in the same package, causing:

```
server\jobs_test.go:17:6: newTestServer redeclared in this block
	server\downloads_test.go:18:6: other declaration of newTestServer
```

### Problem 2: Signature Mismatch

The two definitions serve different purposes:

**`downloads_test.go` (line 18):**
```go
func newTestServer(t *testing.T, cfg *config.Config) *Server {
    // Minimal Server for HTTP handler testing.
    // No real DB, no service initialization.
    // Config must be provided by caller.
    return &Server{
        cfg:           cfg,
        db:            &db.DB{},           // stub
        router:        http.NewServeMux(),
        tokenCache:    make(map[string]tokenCacheEntry),
        loginLimiters: make(map[string]*loginRateLimiter),
    }
}
```

**`jobs_test.go` (line 17):**
```go
func newTestServer(t *testing.T) *Server {
    // Full integration test server.
    // Initializes real in-memory SQLite DB.
    // Sets up all business services.
    database, err := db.Init(":memory:")
    if err != nil {
        t.Fatalf("failed to init test db: %v", err)
    }
    t.Cleanup(func() { database.Close() })
    
    cfg := &config.Config{
        Port:       8080,
        AdminToken: "test-token",
    }
    return NewWithStatic(cfg, database, "")
}
```

### Problem 3: Call Site Divergence

- **12 test files** call `newTestServer(t)` expecting the full integration server
- **1 test file** (`downloads_test.go`) calls `newTestServer(t, &config.Config{...})` expecting minimal handler-only setup

---

## Architecture Analysis

### Test Patterns in Codebase

**Integration Tests (12 files: 108 call sites):**
- `agent_delete_test.go`, `agent_token_test.go`, `agent_update_test.go`
- `federation_health_gap_test.go`, `federation_test.go`, `hub_test.go`
- `job_runs_test.go`, `jobs_status_results_test.go`, `jobs_test.go`
- `offline_detector_test.go`, `pagination_test.go`, `scheduler_test.go`

These tests:
- Access `s.db` to query/insert data
- Call `registerAgent()`, `registerTestAgent()`, `createJob()` helpers
- Test full request lifecycle through `s.router.ServeHTTP()`
- Require real database state

**Handler-Only Tests (1 file: 3 call sites):**
- `downloads_test.go`

These tests:
- Pass custom `*config.Config` (Port, InstallerDir, AdminToken)
- Do NOT access `s.db` (handlers must not touch it)
- Do NOT initialize business services
- Test isolated HTTP handler responses

### Design Intent

The two variants represent different test layers:

| Aspect | Integration (`jobs_test.go` pattern) | Handler-only (`downloads_test.go` pattern) |
|--------|--------------------------------------|-------------------------------------------|
| **DB** | Real in-memory SQLite | Stub (empty `*db.DB{}`) |
| **Services** | All initialized (JobService, etc.) | None |
| **State** | Agents, jobs, runs persisted | Config only |
| **Test Scope** | Full request + business logic | HTTP handler response only |
| **Call Args** | `newTestServer(t)` | `newTestServer(t, cfg)` |

---

## Fix Strategy Options

### Option A: Consolidate into Single Helper (RECOMMENDED)

**Approach:** Create a unified `newTestServer` that accepts optional config, with sensible defaults for integration tests.

**Implementation:**
```go
// In a dedicated shared test helper file (e.g., test_helpers.go or server_test.go)

func newTestServer(t *testing.T, opts ...TestServerOption) *Server {
    t.Helper()
    
    // Default: full integration setup
    cfg := &config.Config{
        Port:       8080,
        AdminToken: "test-token",
    }
    var skipDB bool
    
    // Apply options (handler-only override, custom config, etc.)
    for _, opt := range opts {
        opt(cfg, &skipDB)
    }
    
    if skipDB {
        // Handler-only variant (downloads_test.go use case)
        return &Server{
            cfg:           cfg,
            db:            &db.DB{},
            router:        http.NewServeMux(),
            tokenCache:    make(map[string]tokenCacheEntry),
            loginLimiters: make(map[string]*loginRateLimiter),
        }
    }
    
    // Integration variant (jobs_test.go pattern)
    database, err := db.Init(":memory:")
    if err != nil {
        t.Fatalf("failed to init test db: %v", err)
    }
    t.Cleanup(func() { database.Close() })
    
    return NewWithStatic(cfg, database, "")
}

type TestServerOption func(*config.Config, *bool)

func WithHandlerOnly() TestServerOption {
    return func(cfg *config.Config, skipDB *bool) {
        *skipDB = true
    }
}

func WithConfig(cfg *config.Config) TestServerOption {
    return func(c *config.Config, _ *bool) {
        *c = *cfg
    }
}
```

**Call Sites:**
- Integration tests: `s := newTestServer(t)` ✓ (no change)
- Handler tests: `s := newTestServer(t, WithHandlerOnly(), WithConfig(&config.Config{...}))` 

**Pros:**
- Single definition eliminates redeclaration error
- Unified interface for all test variants
- Type-safe option pattern
- Backward compatible with integration tests
- Explicit about test intent (WithHandlerOnly() is self-documenting)

**Cons:**
- Requires introducing Option pattern
- Handler tests get slightly more verbose call syntax
- Adds one new file or extends an existing test file

---

### Option B: Move Handler-Only to downloads_test.go Scope

**Approach:** Define handler-only setup directly in `downloads_test.go` as an unexported helper; move integration helper to shared file.

**Implementation:**
```go
// In coordinator/server/test_helpers.go (new file)

// For integration tests
func newTestServer(t *testing.T) *Server {
    t.Helper()
    database, err := db.Init(":memory:")
    if err != nil {
        t.Fatalf("failed to init test db: %v", err)
    }
    t.Cleanup(func() { database.Close() })
    cfg := &config.Config{
        Port:       8080,
        AdminToken: "test-token",
    }
    return NewWithStatic(cfg, database, "")
}

// In coordinator/server/downloads_test.go (modified)

// Handler-only variant scoped to this file
func handlerTestServer(t *testing.T, cfg *config.Config) *Server {
    t.Helper()
    return &Server{
        cfg:           cfg,
        db:            &db.DB{},
        router:        http.NewServeMux(),
        tokenCache:    make(map[string]tokenCacheEntry),
        loginLimiters: make(map[string]*loginRateLimiter),
    }
}

// Update calls: newTestServer(t, cfg) → handlerTestServer(t, cfg)
```

**Pros:**
- Minimal changes to call sites (integration tests unchanged)
- Clear ownership: downloads_test.go owns its handler-only logic
- No option pattern needed
- Unambiguous function names

**Cons:**
- Two separate functions (not unified)
- downloads_test.go must remember to use `handlerTestServer` not `newTestServer`
- If other handler-only tests emerge later, duplication risk

---

### Option C: Split Into Two Files (LEAST PREFERRED)

**Approach:** Move each helper to its own file (e.g., `integration_test.go`, `handlers_test.go`).

**Pros:**
- Completely separate concerns
- No naming collision

**Cons:**
- Fragmentation (two test setup files)
- Harder to maintain parallel variants
- Integration tests lose cohesion

---

## Recommended Implementation Path

### **PRIMARY CHOICE: Option A (Unified Handler)**

**Rationale:**
1. Single definition resolves redeclaration error
2. Flexible option pattern scales if more variants needed later
3. Integration tests require no changes
4. Handler tests become explicitly intentional (`WithHandlerOnly()`)
5. Type safety and extensibility

**Phase 1: Create Unified Helper**
- Create `coordinator/server/test_helpers.go`
- Define `newTestServer(t *testing.T, opts ...TestServerOption)`
- Define `TestServerOption`, `WithHandlerOnly()`, `WithConfig()`

**Phase 2: Update downloads_test.go**
- Change 3 call sites from `newTestServer(t, cfg)` to `newTestServer(t, WithHandlerOnly(), WithConfig(cfg))`

**Phase 3: Retire Old Definitions**
- Remove `newTestServer` from `jobs_test.go` (lines 17-31)
- Remove `newTestServer` from `downloads_test.go` (lines 18-27)
- All 12 integration test files automatically use shared helper

**Phase 4: Verify**
- `go test ./server` compiles and passes
- All 108 integration call sites work without modification
- 3 handler call sites updated to new syntax

---

### **FALLBACK CHOICE: Option B (Scoped Handler)**

If stakeholders prefer minimal syntactic change:

**Phase 1:** Create `coordinator/server/test_helpers.go` with integration `newTestServer`
**Phase 2:** Rename in `downloads_test.go` to `handlerTestServer`
**Phase 3:** Update 3 call sites in `downloads_test.go`
**Phase 4:** Remove duplicate definitions from `jobs_test.go` and old `downloads_test.go`

---

## Risk Assessment

### Mitigation

**Risk:** Integration tests break due to missing service initialization
- **Mitigation:** Option A preserves default integration path; no behavior change
- **Confidence:** Very high (existing code path unchanged)

**Risk:** Handler tests regress if new dependencies added to `Server`
- **Mitigation:** Option A makes dependency injection explicit via options
- **Confidence:** High (options pattern forces intent)

**Risk:** Test framework assumes specific signature
- **Mitigation:** Variadic options are backward compatible
- **Confidence:** Very high (Go stdlib pattern)

---

## Effort Estimate

| Activity | Option A | Option B |
|----------|----------|----------|
| Create shared helper | 30 min | 30 min |
| Update downloads_test.go | 10 min | 10 min |
| Clean up old definitions | 5 min | 5 min |
| Test & verify | 15 min | 15 min |
| **Total** | **1 hour** | **1 hour** |

---

## Affected Files

### Redeclaration Sites (Both Define)
- `coordinator/server/downloads_test.go` (line 18)
- `coordinator/server/jobs_test.go` (line 17)

### Call Sites (Use newTestServer)
- `agent_delete_test.go` (6 calls)
- `agent_token_test.go` (6 calls)
- `agent_update_test.go` (4 calls)
- `downloads_test.go` (3 calls) — needs arg adjustment
- `federation_health_gap_test.go` (2 calls)
- `federation_test.go` (16 calls)
- `hub_test.go` (5 calls)
- `job_runs_test.go` (5 calls)
- `jobs_status_results_test.go` (8 calls)
- `jobs_test.go` (25 calls)
- `offline_detector_test.go` (4 calls)
- `pagination_test.go` (6 calls)
- `scheduler_test.go` (5 calls)

**Total: 112 locations across 13 files**

---

## Conclusion

The duplication and signature mismatch stem from a recent refactoring that introduced handler-only testing without consolidating the setup patterns. A unified option-based helper resolves both compilation errors while maintaining backward compatibility and enabling future test variants.

**Recommendation:** Proceed with **Option A** (Unified Handler with TestServerOption pattern).

---

## Appendix: Sample Implementation

```go
// coordinator/server/test_helpers.go (NEW FILE)

package server

import (
    "testing"
    "arcvault/coordinator/config"
    "arcvault/coordinator/db"
)

// TestServerOption is a functional option for configuring test servers.
type TestServerOption func(*config.Config, *bool) error

// newTestServer creates a test server with optional configuration.
// By default, it initializes a full integration server with in-memory DB.
// Use WithHandlerOnly() for HTTP handler-only testing.
func newTestServer(t *testing.T, opts ...TestServerOption) *Server {
    t.Helper()
    
    cfg := &config.Config{
        Port:       8080,
        AdminToken: "test-token",
    }
    var handlerOnly bool
    
    for _, opt := range opts {
        if err := opt(cfg, &handlerOnly); err != nil {
            t.Fatalf("failed to apply test option: %v", err)
        }
    }
    
    if handlerOnly {
        // Minimal server for handler testing (no DB)
        return &Server{
            cfg:           cfg,
            db:            &db.DB{},
            router:        http.NewServeMux(),
            tokenCache:    make(map[string]tokenCacheEntry),
            loginLimiters: make(map[string]*loginRateLimiter),
        }
    }
    
    // Full integration server with real DB
    database, err := db.Init(":memory:")
    if err != nil {
        t.Fatalf("failed to init test db: %v", err)
    }
    t.Cleanup(func() { database.Close() })
    
    return NewWithStatic(cfg, database, "")
}

// WithHandlerOnly marks this server for HTTP handler testing only.
func WithHandlerOnly() TestServerOption {
    return func(_ *config.Config, handlerOnly *bool) error {
        *handlerOnly = true
        return nil
    }
}

// WithConfig overrides the test server configuration.
func WithConfig(cfg *config.Config) TestServerOption {
    return func(c *config.Config, _ *bool) error {
        *c = *cfg
        return nil
    }
}
```

---

**Document prepared by:** David Mensah (Software Architect)  
**Review required from:** Elena Vasquez (Senior Code Reviewer)  
**Implementation owner:** (To be assigned)

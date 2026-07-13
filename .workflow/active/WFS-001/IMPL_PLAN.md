# Implementation Plan: Credential Scoping for Agent Tokens

**Session**: WFS-001
**Goal**: Scope credential decryption so agents can only decrypt credentials for their own assigned jobs
**Priority**: High (Medium severity per audit, Medium effort)

---

## Overview

Currently, any agent token can decrypt ALL credential profiles by calling `GET /api/jobs`. The fix adds agent identity tracking to the request context and enforces ownership checks before decrypting credentials.

---

## Task 1: Add agent identity extraction to ValidateToken

**File**: `coordinator/db/db.go`

**Current** (lines 131-144):
```go
func (db *DB) ValidateToken(token string) (string, error) {
    var role string
    err := db.DB.QueryRow(
        `SELECT role FROM tokens WHERE token = ? AND (expires_at IS NULL OR expires_at > datetime('now'))`,
        token,
    ).Scan(&role)
    ...
    return role, nil
}
```

**Change**: Add a new exported function `GetAgentIDByToken(token string) (string, error)`:

```go
// GetAgentIDByToken validates a token and returns the associated agent_id.
// Returns ("", nil) if the token exists but has no agent_id (e.g., bootstrap tokens).
func (db *DB) GetAgentIDByToken(token string) (string, error) {
    var agentID sql.NullString
    err := db.DB.QueryRow(
        `SELECT agent_id FROM tokens WHERE token = ? AND (expires_at IS NULL OR expires_at > datetime('now'))`,
        token,
    ).Scan(&agentID)
    if err == sql.ErrNoRows {
        return "", fmt.Errorf("invalid or expired token")
    }
    if err != nil {
        return "", fmt.Errorf("token lookup failed: %w", err)
    }
    return agentID.String, nil // returns "" if NULL
}
```

Leave `ValidateToken` unchanged (it's used elsewhere for role checks).

---

## Task 2: Define agent identity context key

**File**: `coordinator/server/server.go` (or `coordinator/server/auth.go`)

Add a new context key:

```go
// AgentIDCtxKey is the context key for storing the authenticated agent's ID.
type contextKey string
const AgentIDCtxKey contextKey = "agent_id"
```

If there's already a `contextKey` type defined in auth.go, use that file instead. The key should be of a private type to prevent collisions.

---

## Task 3: Store agent identity in middleware

**File**: `coordinator/server/server.go`

### 3a: Modify `authMiddleware` (around line 516-531)

After the token is validated, resolve the agent_id and store it in context:

```go
// After existing token validation logic, add:
if agentID, err := s.db.GetAgentIDByToken(tok); err == nil && agentID != "" {
    r = r.WithContext(context.WithValue(r.Context(), AgentIDCtxKey, agentID))
}
```

This should go after the `isAgentToken` check succeeds and before `next(w, r)`.

### 3b: Modify `agentOrViewerRoute` (around line 535-544)

Same pattern — after the agent token is validated, resolve agent_id and store in context:

```go
// In the agent token branch, add before next(w, r):
if agentID, err := s.db.GetAgentIDByToken(tok); err == nil && agentID != "" {
    r = r.WithContext(context.WithValue(r.Context(), AgentIDCtxKey, agentID))
}
```

### 3c: Add a helper function

Add alongside the existing `isAgentToken` and `isAdminToken` helpers:

```go
// getAgentIDFromContext extracts the agent ID from the request context.
// Returns empty string if not set (e.g., JWT-authenticated requests).
func getAgentIDFromContext(r *http.Request) string {
    if agentID, ok := r.Context().Value(AgentIDCtxKey).(string); ok {
        return agentID
    }
    return ""
}
```

---

## Task 4: Enforce ownership in handleListJobs

**File**: `coordinator/server/jobs.go`

### 4a: Modify the credential decryption loop (around lines 204-235)

Before decrypting credentials for each job, add an ownership check:

```go
// Get the authenticated agent's ID from context
agentID := getAgentIDFromContext(r)

// Inside the job loop, before decryptCredentials:
if agentID != "" && jobDTO.AgentID != agentID {
    continue // skip credentials for jobs not owned by this agent
}
```

This ensures:
- If the request is from an agent (agentID is set), only decrypt credentials for jobs where `job.AgentID == agentID`
- If the request is from a JWT-authenticated user (agentID is empty), keep existing behavior (decrypt all)
- The ownership check is per-job, so other jobs' metadata is still listed but without decrypted credentials

### 4b: Also update `handleGetJob` (around line 265-278) if needed

This handler already has dead agent code (agent can never reach it due to viewerRoute middleware). But add the same pattern for defense-in-depth:

```go
// If this is an agent request, verify ownership
if agentID := getAgentIDFromContext(r); agentID != "" && jobDTO.AgentID != agentID {
    return sendNotFound(w, "job not found")
}
```

---

## Task 5: Add import for "context" if needed

Check if `coordinator/server/server.go` already imports `"context"`. If not, add it to the import block.

Check if `coordinator/db/db.go` already imports `"database/sql"`. If not, add it.

---

## Task 6: Update test coverage

**File**: Create or update tests to verify the scoping.

At minimum, update existing tests in `coordinator/server/jobs_test.go` or `coordinator/db/db_test.go`:

1. Test that `GetAgentIDByToken` returns the correct agent_id for a valid token
2. Test that `GetAgentIDByToken` returns an error for an invalid/expired token
3. Test that `GetAgentIDByToken` handles NULL agent_id gracefully (bootstrap tokens)
4. Integration-level test: agent token for Agent A cannot see decrypted credentials for Agent B's jobs

---

## Implementation Order

| Step | Task | File(s) | Effort |
|------|------|---------|--------|
| 1 | Add GetAgentIDByToken DB function | `coordinator/db/db.go` | Low |
| 2 | Define AgentIDCtxKey and getAgentIDFromContext | `coordinator/server/server.go` | Low |
| 3a | Store agent_id in authMiddleware | `coordinator/server/server.go` | Low |
| 3b | Store agent_id in agentOrViewerRoute | `coordinator/server/server.go` | Low |
| 4 | Enforce ownership check in handleListJobs | `coordinator/server/jobs.go` | Low |
| 5 | Defense-in-depth in handleGetJob | `coordinator/server/jobs.go` | Low |
| 6 | Add tests | `coordinator/*_test.go` | Medium |

Total effort: **Low-Medium** (well-scoped, touches 3 files at most)

---

## Verification

After implementation:
1. `go build ./coordinator/...` — must compile cleanly
2. `go test ./coordinator/...` — existing tests must pass
3. Manual scenario: Agent A token, job listing should only show decrypted credentials for Agent A's jobs
4. Manual scenario: Admin/operator token, job listing should show all decrypted credentials (no regression)

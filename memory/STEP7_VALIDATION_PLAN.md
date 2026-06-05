# Step 7: API Contracts & Validation — Implementation Plan

**Date:** June 4, 2026  
**Status:** Phase 1 Complete (API Spec), Ready for Phase 2  
**Estimated Time:** ~1 hour (Phase 2-3)

---

## What We Just Created

✅ **API_SPECIFICATION.md** — Complete contract documentation including:
- All endpoints (agents, jobs, groups, auth/users)
- Request/response schemas
- Validation rules for each field
- Status codes and error conditions
- Breaking changes analysis (none)

---

## Phase 2: Request/Response Type Definitions

### Goal
Create typed request/response structs for each handler to match the API spec.

### Approach

**Create `types.go` in each domain:**

```
coordinator/server/
  ├── auth_types.go (LoginRequest, LoginResponse, ChangePasswordRequest, etc)
  ├── users_types.go (CreateUserRequest, CreateUserResponse, etc)
  ├── groups_types.go (CreateGroupRequest, UpdateGroupRequest, etc)
  ├── agents_types.go (RegisterRequest, HeartbeatRequest, etc)
  └── jobs_types.go (CreateJobRequest, JobProgressRequest, etc)
```

### Type Definition Pattern

**Example:** `auth_types.go`

```go
package server

// LoginRequest defines the shape of a login request body
type LoginRequest struct {
    Username string `json:"username"`
    Password string `json:"password"`
}

// LoginResponse defines the response after successful login
type LoginResponse struct {
    UserID            int    `json:"user_id"`
    Username          string `json:"username"`
    Token             string `json:"token"`
    ExpiresIn         int    `json:"expires_in"`
    MustChangePassword bool  `json:"must_change_password"`
}

// ChangePasswordRequest defines password change request
type ChangePasswordRequest struct {
    OldPassword string `json:"old_password"`
    NewPassword string `json:"new_password"`
}
```

### Files to Create

| File | Request Types | Response Types |
|------|---------------|----------------|
| `auth_types.go` | LoginRequest, ChangePasswordRequest | LoginResponse, ChangePasswordResponse |
| `users_types.go` | CreateUserRequest, UpdateRoleRequest | UserResponse, PaginatedUsersResponse |
| `groups_types.go` | CreateGroupRequest, UpdateGroupRequest | GroupResponse, PaginatedGroupsResponse |
| `agents_types.go` | RegisterRequest, HeartbeatRequest | AgentResponse, PaginatedAgentsResponse |
| `jobs_types.go` | CreateJobRequest, PostProgressRequest, PostResultsRequest | JobResponse, PaginatedJobsResponse, JobProgressResponse |

### Shared Types

Create in a common location (e.g., `common_types.go`):

```go
// PaginationMeta provides pagination info
type PaginationMeta struct {
    Page  int `json:"page"`
    Limit int `json:"limit"`
    Total int `json:"total"`
    Pages int `json:"pages"`
}

// PaginatedResponse wraps paginated data
type PaginatedResponse struct {
    Data       interface{}    `json:"data"`
    Pagination PaginationMeta `json:"pagination"`
}

// ErrorResponse defines error response format
type ErrorResponse struct {
    Error  string `json:"error"`
    Status int    `json:"status,omitempty"`
}

// SuccessResponse wraps successful response
type SuccessResponse struct {
    Data  interface{} `json:"data"`
    Error interface{} `json:"error"`
}
```

---

## Phase 3: Input Validation in Handlers

### Goal
Add validation methods to request types and call them in handlers.

### Approach

**Add `Validate()` method to each request type:**

```go
func (r *LoginRequest) Validate() error {
    if r.Username == "" {
        return fmt.Errorf("username is required")
    }
    if r.Password == "" {
        return fmt.Errorf("password is required")
    }
    if len(r.Password) < 8 {
        return fmt.Errorf("password must be 8+ characters")
    }
    return nil
}
```

### Handler Pattern

**Before (no validation):**
```go
func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
    var req LoginRequest
    json.NewDecoder(r.Body).Decode(&req)
    
    // Call service directly
    user, err := s.userService.ValidateCredentials(req.Username, req.Password)
    // ...
}
```

**After (with validation):**
```go
func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
    var req LoginRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, `{"error":"invalid request"}`, http.StatusBadRequest)
        return
    }
    
    // Validate request
    if err := req.Validate(); err != nil {
        http.Error(w, fmt.Sprintf(`{"error":"%s"}`, err.Error()), http.StatusBadRequest)
        return
    }
    
    // Call service
    user, err := s.userService.ValidateCredentials(req.Username, req.Password)
    // ...
}
```

### Validation Rules by Domain

**Authentication:**
- LoginRequest: username and password required, password 8+ chars
- ChangePasswordRequest: both passwords required, 8+ chars each, must differ

**Users:**
- CreateUserRequest: username required (1-255, unique), password 8+ chars, role in (admin|viewer)
- UpdateRoleRequest: role required, must be in (admin|viewer)

**Groups:**
- CreateGroupRequest: name required (1-255 chars)
- UpdateGroupRequest: name optional (1-255 if provided), description optional (0-1000)

**Agents:**
- RegisterRequest: agent_id (UUID), hostname (1-255), os (linux|darwin|windows), arch (valid), version (semver)
- HeartbeatRequest: rollback_available (boolean)

**Jobs:**
- CreateJobRequest: name (1-255), source/dest paths (1-4096), agent_id OR group_id (not both, not neither)
- PostProgressRequest: percentage (0-100), status (optional, pending|in_progress|completed|failed)
- PostResultsRequest: run_id (UUID), exit_code (0-255), error (optional)

---

## Phase 4: Error Message Standardization

### Goal
Ensure all error responses follow the API spec format.

### Pattern

```go
// Return 400 Bad Request for validation errors
w.Header().Set("Content-Type", "application/json")
w.WriteHeader(http.StatusBadRequest)
json.NewEncoder(w).Encode(map[string]string{"error": "field name is required"})

// Return 401 Unauthorized for auth failures
w.Header().Set("Content-Type", "application/json")
w.WriteHeader(http.StatusUnauthorized)
json.NewEncoder(w).Encode(map[string]string{"error": "invalid credentials"})

// Return 403 Forbidden for permission errors
w.Header().Set("Content-Type", "application/json")
w.WriteHeader(http.StatusForbidden)
json.NewEncoder(w).Encode(map[string]string{"error": "not authorized"})

// Return 404 Not Found for missing resources
w.Header().Set("Content-Type", "application/json")
w.WriteHeader(http.StatusNotFound)
json.NewEncoder(w).Encode(map[string]string{"error": "resource not found"})

// Return 500 for server errors (no details)
w.Header().Set("Content-Type", "application/json")
w.WriteHeader(http.StatusInternalServerError)
json.NewEncoder(w).Encode(map[string]string{"error": "internal server error"})
```

---

## Implementation Order

### Priority 1 (High Risk, Security)
1. **Auth handlers** (handleLogin, handleChangePassword)
   - Sensitive fields (username, password)
   - Security implications of validation errors
   
2. **User management** (handleCreateUser, handleUpdateUserRole)
   - Admin operations
   - Permission checks

### Priority 2 (Medium Risk, Complex Logic)
3. **Job handlers** (handleCreateJob, handlePostProgress)
   - Complex validation (agent_id OR group_id, not both)
   - Progress percentage validation (0-100)

4. **Agent handlers** (handleRegister, handleHeartbeat)
   - ID format validation (UUID)
   - OS/arch validation

### Priority 3 (Lower Risk)
5. **Group handlers** (handleCreateGroup, handleUpdateGroup)
   - Simpler validation rules
   - No security implications

---

## Testing Strategy

### For Each Handler

1. **Valid Request** — Verify successful execution
2. **Missing Required Field** — Verify 400 with proper error message
3. **Invalid Format** — Verify 400 (e.g., non-UUID, non-number)
4. **Out of Range** — Verify 400 (e.g., percentage > 100)
5. **Conflicting Fields** — Verify 400 (e.g., both agent_id and group_id)

### Example Test

```go
func TestLoginValidation_MissingPassword(t *testing.T) {
    body := `{"username": "admin"}`
    req := httptest.NewRequest("POST", "/api/auth/login", strings.NewReader(body))
    w := httptest.NewRecorder()
    
    server.handleLogin(w, req)
    
    if w.Code != http.StatusBadRequest {
        t.Errorf("Expected 400, got %d", w.Code)
    }
    if !strings.Contains(w.Body.String(), "password") {
        t.Errorf("Error should mention password field")
    }
}
```

---

## Success Criteria

- [x] API specification documented (Phase 1)
- [ ] All request/response types defined (Phase 2)
- [ ] Validation methods added to all request types (Phase 3)
- [ ] Handlers call Validate() before service layer (Phase 3)
- [ ] Error responses standardized (Phase 4)
- [ ] All existing tests still pass (Phase 4)
- [ ] New validation tests added for critical paths (Phase 4)

---

## Notes

1. **Type Safety:** Go's struct tags enable automatic JSON marshaling/unmarshaling
2. **Validation at Boundary:** Validate at HTTP boundary (handler), not in service layer
3. **Error Messages:** Should be user-friendly but not expose internals
4. **Status Codes:** Always use appropriate HTTP status codes
5. **No Breaking Changes:** All existing tests should continue to pass

---

## Next Steps (After Phase 3)

1. Commit with message: `Step 7 Phase 2-3: Add API contracts and validation`
2. Run full test suite: `go test ./coordinator/... -v`
3. Verify zero regressions
4. Proceed to Step 8: Cleanup & Documentation

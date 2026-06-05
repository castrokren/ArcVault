# Path Authentication Implementation Status

**Date:** 2026-06-05  
**Branch:** feat/credcrypto-rekey

## Summary

**Plan A:** ✅ **COMPLETE** (5/5 tasks)  
**Plan B:** 🟡 **IN PROGRESS** (7/9 tasks)  
**Plan C:** ⏳ **Not Started**

---

## Plan A: credcrypto Module + Rekey CLI

### Status: ✅ COMPLETE

All tasks implemented and tested:

1. ✅ `coordinator/internal/credcrypto/crypto.go` — AES-256-GCM encryption/decryption
   - `LoadKey()` from ARCVAULT_CREDENTIAL_KEY env var (hex encoded, 32 bytes)
   - `Encrypt(key, plaintext)` → nonce + ciphertext + tag format
   - `Decrypt(key, ciphertext)` → plaintext
   - Error types: `ErrKeyNotSet`, `ErrKeyInvalid`

2. ✅ `coordinator/internal/credcrypto/rekey.go` — Key rotation
   - `Rekey(db, oldKey, newKey)` — reads credentials table, decrypts with old key, encrypts with new
   - Stops on decrypt error (logs which credential failed)
   - Not transactional (single-pass, fails fast)

3. ✅ `crypto_test.go` — 7 unit tests
   - Key loading validation (set, not set, invalid hex, wrong length)
   - Encrypt/decrypt round-trip
   - Wrong key decryption fails
   - Empty plaintext handling

4. ✅ `rekey_test.go` — 3 integration tests
   - Happy path: rekey multiple credentials
   - Error handling: stops on decrypt failure
   - Empty table: handles no-op case

5. ✅ `coordinator/main.go` integration
   - `rekey --old-key <hex> --new-key <hex>` subcommand
   - Exits before HTTP server starts
   - Helper functions: `DecodeKeyHex()`, `OpenDatabase()`

### Tests

```bash
go test ./coordinator/internal/credcrypto/... -v
# Result: 10/10 PASS (0.29s)

go build ./coordinator/...
# Result: SUCCESS
```

---

## Plan B: DB Migrations, Credential Profiles CRUD, Job Integration

### Status: 🟡 IN PROGRESS (7/9 complete)

### Completed Tasks

1. ✅ **Task 1:** `credential_profiles` table migration
   - Schema: id (PK), name (UNIQUE), type, encrypted_data (BLOB), created_at
   - Idempotent creation in `db/db.go` schema block

2. ✅ **Task 2:** `jobs` table migration
   - Added `credential_profile_id TEXT` column (optional FK)
   - Idempotent ALTER in migration function

3. ✅ **Task 3:** `job_runs` table migration
   - Added `credential_profile_id TEXT` and `credential_profile_name TEXT` columns
   - Idempotent ALTERs for snapshot capture

4. ✅ **Task 4:** `POST /api/credential-profiles` handler
   - Location: `coordinator/server/credentials.go`
   - Validates: name, type, data required
   - Checks: ARCVAULT_CREDENTIAL_KEY set (503 if not)
   - Encrypts data with `credcrypto.Encrypt()`
   - Returns: id, name, type, created_at (NO encrypted_data)
   - Status: 201 Created

5. ✅ **Task 5:** `GET /api/credential-profiles` handler
   - Returns list without encrypted data
   - Response: [{id, name, type, created_at}, ...]
   - Route: `s.adminRoute()` (admin only via middleware)

6. ✅ **Task 6:** `DELETE /api/credential-profiles/{id}` handler
   - Returns 404 if profile not found
   - Returns 409 Conflict if any job references it
   - Returns 204 No Content on success
   - Validation: `db.HasJobsReferencingProfile(profileID)`

7. ✅ **Task 7:** Credential validation in job creation
   - Modified: `POST /api/jobs` handler
   - New input field: `credential_profile_id` (optional)
   - Validation flow:
     1. Profile exists (404 if not)
     2. Profile type matches agent OS (422 if not)
     3. Assign to job via `db.UpdateJobCredentialProfile()`
   - OS compatibility map in `validateCredentialTypeForAgent()`:
     - SMB → windows
     - SSH → linux, darwin, unix
     - AWS → cross-platform
     - Database → cross-platform

### Database Helpers (credentials.go)

```go
CreateCredentialProfile(id, name, type, encryptedData) error
GetCredentialProfile(id) (*CredentialProfile, error)
ListCredentialProfiles() ([]*CredentialProfile, error)
DeleteCredentialProfile(id) error
HasJobsReferencingProfile(profileID) (bool, error)
UpdateJobCredentialProfile(jobID, profileID) error           // NEW
SnapshotJobRunCredentials(runID, profileID, profileName)    // NEW
```

### Remaining Tasks

#### Task 8: Agent-Facing Credentials Injection

**Goal:** When an agent requests `GET /api/jobs`, decrypt and include credentials if:
- Request is authenticated with agent token (not JWT)
- Job has `credential_profile_id` set
- ARCVAULT_CREDENTIAL_KEY is available

**Implementation Notes:**
- Current `GET /api/jobs` uses `viewerRoute` (JWT + role-based)
- Need to support agent token auth alongside JWT
- Distinguish in handler: check for `UserClaims` in context
  - If claims present → JWT user (no credentials)
  - If no claims → agent token (include credentials if available)
- Decrypt flow:
  ```go
  profile := db.GetCredentialProfile(job.CredentialProfileID)
  plaintext := credcrypto.Decrypt(key, profile.EncryptedData)
  job.Credentials = json.Unmarshal(plaintext)
  ```

**Modified Response Type:**
```go
type Job struct {
  // ... existing fields ...
  Credentials map[string]interface{} `json:"credentials,omitempty"`
}
```

#### Task 9: Job Run Credential Snapshots

**Goal:** When job execution starts, capture credential profile state in job_runs record

**Implementation Notes:**
- Hook point: When job_runs record is created (likely in job executor/scheduler)
- Call: `db.SnapshotJobRunCredentials(runID, profileID, profileName)`
- Snapshot both ID and name (name helpful for audit trail if profile deleted later)
- Verify: completed job_runs has `credential_profile_id` and `credential_profile_name` populated

**Location:** Likely in `coordinator/service/job_execution.go` or scheduler

---

## Plan C: Agent Credential Injection + Dashboard

**Status:** ⏳ Not started

Tasks (from planning file):
1. Agent `ApplyCredentials()` method
2. Setup wizard key generation
3. Dashboard Credentials page
4. Job form credential selection

**Dependency:** Plan B Tasks 8-9 must complete first

---

## Key Implementation Notes

### Credential Encryption Model

- **Storage:** Encrypted BLOB in credential_profiles.encrypted_data
- **Key:** ARCVAULT_CREDENTIAL_KEY env var (hex-encoded 32-byte AES-256 key)
- **Format:** Plaintext is arbitrary JSON (application decides structure)
- **Encryption:** AES-256-GCM (nonce + ciphertext + tag format)

### Job ← Credential Profile Linkage

```
credential_profiles (main storage, encrypted)
     ↓
jobs.credential_profile_id (optional reference)
     ↓
job_runs.credential_profile_id + credential_profile_name (snapshot at execution)
     ↓
Agent receives decrypted credentials in job response (agent token only)
```

### Backward Compatibility

- All credential fields are optional/new
- Existing jobs without credentials work unchanged
- All migrations are idempotent (safe to re-run)

### Testing Strategy

**Done:**
- Unit tests for crypto primitives
- Integration tests with SQLite
- Build verification

**TODO for Tasks 8-9:**
- Agent token request handling
- Credential decryption in job responses
- Job run snapshot verification

---

## Next Steps

### To Complete Plan B (8-9 only)

1. **Task 8:** Modify job list handler to support agent tokens + credential injection
   - Check auth context for JWT claims
   - Decrypt & inject credentials for agent requests
   - Add `Credentials` field to Job response type

2. **Task 9:** Hook job execution to snapshot credentials
   - Find where job_runs are created
   - Call `db.SnapshotJobRunCredentials()` when execution starts

### Then Start Plan C

Depends on Plan B completion.

---

## Commit History

- `feat: Add credcrypto package...` (Plan A complete)
- `feat: Add credential profile management...` (Plan B partial, Tasks 1-6)
- `feat: Add credential profile validation...` (Plan B Task 7)

---

## Environment Setup Required

```bash
# For development/testing:
export ARCVAULT_CREDENTIAL_KEY=<hex-encoded-32-byte-key>

# Example (NOT for production):
export ARCVAULT_CREDENTIAL_KEY=0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef
```

Generate with:
```bash
openssl rand -hex 32
```

---

## Files Modified/Created

### New Files
- `coordinator/internal/credcrypto/crypto.go`
- `coordinator/internal/credcrypto/crypto_test.go`
- `coordinator/internal/credcrypto/rekey.go`
- `coordinator/internal/credcrypto/rekey_test.go`
- `coordinator/db/credentials.go`
- `coordinator/server/credentials.go`

### Modified Files
- `coordinator/main.go` (added rekey subcommand)
- `coordinator/cmd/commands.go` (added helpers)
- `coordinator/db/db.go` (added table + migrations)
- `coordinator/server/server.go` (added routes)
- `coordinator/server/jobs.go` (credential validation in creation)

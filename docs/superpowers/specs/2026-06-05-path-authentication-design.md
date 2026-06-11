# Path Authentication Design

**Date:** 2026-06-05  
**Status:** Approved  
**Feature:** Credential profiles for authenticated source/destination paths in backup jobs

---

## Overview

Users need a way to provide credentials when a job's source or destination path requires authentication (e.g. a Windows network share or an SSH remote). Credentials are stored encrypted in the coordinator and passed to the agent at job dispatch time. The credential type is automatically determined by the target agent's OS — no manual type selection required.

---

## Data Model

### New table: `credential_profiles`

| Column | Type | Notes |
|---|---|---|
| `id` | TEXT PRIMARY KEY | UUID |
| `name` | TEXT NOT NULL UNIQUE | User-facing label, e.g. "office-nas" |
| `type` | TEXT NOT NULL | `smb` \| `ssh_key` \| `ssh_password` |
| `encrypted_data` | TEXT NOT NULL | AES-256-GCM encrypted JSON blob |
| `created_at` | DATETIME NOT NULL | |

The encrypted blob structure by type:
- `smb`: `{"username": "...", "password": "...", "domain": "..."}`
- `ssh_key`: `{"username": "...", "private_key": "..."}`
- `ssh_password`: `{"username": "...", "password": "..."}`

### Updated table: `jobs`

Add one column:
```sql
ALTER TABLE jobs ADD COLUMN credential_profile_id TEXT REFERENCES credential_profiles(id);
```

Deletion of a profile that is referenced by any job is blocked at the API layer (409). No cascade or null-set — the DB constraint is enforced in application code.

### Encryption

- **Algorithm:** AES-256-GCM
- **Key source:** `ARCVAULT_CREDENTIAL_KEY` environment variable (32-byte hex string)
- **Key storage:** never stored in the DB — coordinator reads it from the environment at startup
- **Behavior when unset:** credential endpoints return `503 Service Unavailable` with a clear error message; existing jobs without credentials are unaffected

---

## API

All credential endpoints require admin role.

### `POST /api/credential-profiles`

Create a credential profile.

**Request body:**
```json
{
  "name": "office-nas",
  "type": "smb",
  "username": "domain\\user",
  "password": "secret",
  "domain": "CORP"
}
```
SSH key variant uses `private_key` instead of `password`/`domain`.

**Response (201):** `{ "id": "...", "name": "office-nas", "type": "smb", "created_at": "..." }` — secrets never returned.

**Validation:**
- `name` required, 1–255 characters, unique
- `type` must be one of `smb`, `ssh_key`, `ssh_password`
- `username` required for all types
- `password` required for `smb` and `ssh_password`
- `private_key` required for `ssh_key`
- `ARCVAULT_CREDENTIAL_KEY` must be set or returns 503

### `GET /api/credential-profiles`

List all profiles. Returns id, name, type, created_at — no secret fields.

**Response (200):**
```json
{
  "data": [
    { "id": "...", "name": "office-nas", "type": "smb", "created_at": "..." }
  ]
}
```

### `DELETE /api/credential-profiles/{id}`

Delete a profile.

- Returns **409 Conflict** if any job currently references this profile
- Returns **404** if profile not found

### Updated: `POST /api/jobs`

`CreateJobRequest` gains an optional field:
```json
{ "credential_profile_id": "uuid-here" }
```

**Validation:** if `credential_profile_id` is provided, the coordinator checks that the profile's type is compatible with the target agent's OS:
- Agent `os = "windows"` → only `smb` profiles accepted
- Agent `os = "linux"` or `"darwin"` → only `ssh_key` or `ssh_password` profiles accepted

Incompatible type returns **422 Unprocessable Entity**.

### Updated: `GET /api/jobs` (agent-facing)

When a job has a `credential_profile_id`, the coordinator decrypts the profile and adds a `credentials` object to that job entry in the response:

```json
{
  "id": "...",
  "source_path": "\\\\server\\share",
  "dest_path": "D:\\backup",
  "credentials": {
    "type": "smb",
    "username": "domain\\user",
    "password": "secret",
    "domain": "CORP"
  }
}
```

The `credentials` field is **never** present in dashboard-facing job responses. The coordinator distinguishes the two callers by auth token type — agents authenticate with agent bearer tokens, dashboard users authenticate with JWTs. The credentials field is only populated when the request carries an agent token.

---

## Agent Changes

### `Job` struct

```go
type JobCredentials struct {
    Type     string `json:"type"`
    Username string `json:"username,omitempty"`
    Password string `json:"password,omitempty"`
    Domain   string `json:"domain,omitempty"`  // smb only
    SSHKey   string `json:"ssh_key,omitempty"` // ssh_key only
}

// Added to Job struct:
Credentials *JobCredentials `json:"credentials,omitempty"`
```

### New function: `applyCredentials`

```go
func applyCredentials(job Job) (cleanup func(), err error)
```

Called in `process()` before the executor runs. Returns a cleanup function that is deferred. If `job.Credentials` is nil, returns a no-op cleanup immediately — fully backward compatible.

**SMB (Windows):**
1. Run `net use <path> /user:<domain\username> <password>`
2. Cleanup: `net use <path> /delete`
3. Applied to whichever of `source_path`/`dest_path` is a UNC path (`\\`)

**SSH key (Linux/Mac):**
1. Write `SSHKey` to `os.CreateTemp("", "arcvault-key-*")`
2. `os.Chmod(tmpFile, 0600)`
3. Set rsync `-e` flag to `ssh -i <tmpfile> -o StrictHostKeyChecking=no`
4. Cleanup: `os.Remove(tmpFile)`

**SSH password (Linux/Mac):**
1. Prepend `sshpass -p <password>` to the rsync command

If `applyCredentials` returns an error, the job is marked failed immediately without running the executor.

---

## Dashboard UI

### New page: Admin → Credentials

- Route: `/admin/credentials`
- Lists all profiles: name, type, created date, delete button
- "New Credential" button opens a form:
  - Name field (text)
  - Type selector: SMB | SSH Key | SSH Password
  - Type-specific fields appear conditionally:
    - SMB: Username, Domain, Password
    - SSH Key: Username, Private Key (textarea)
    - SSH Password: Username, Password
- No edit flow — delete and recreate to update secrets (avoids partial updates)
- Delete shows a confirmation prompt; blocked profiles (referenced by jobs) show an error toast

### Updated: Job creation form

- After an agent is selected, a new optional "Path Authentication" dropdown appears
- Fetches `GET /api/credential-profiles` and filters by agent OS:
  - Windows agent → shows only `smb` profiles
  - Linux/Mac agent → shows only `ssh_key` and `ssh_password` profiles
- Default selection is "None" — job created without credentials
- Group dispatch: credential profile applies to all agents in the group; validation checks all agents share a compatible OS

---

## Error Handling

| Scenario | Behavior |
|---|---|
| `ARCVAULT_CREDENTIAL_KEY` not set | Credential endpoints return 503; unaffected jobs run normally |
| Decryption failure at dispatch | Job marked failed before agent receives it; error logged |
| `net use` fails on agent | `applyCredentials` returns error; job marked failed |
| SSH key write fails on agent | `applyCredentials` returns error; job marked failed |
| Profile deleted while jobs reference it | Blocked — API returns 409; profile cannot be deleted until all referencing jobs are deleted first |

---

## Testing

- **Coordinator unit tests:** encryption/decryption round-trip, API validation (type compatibility, missing key env var, 409 on referenced profile delete)
- **Agent unit tests:** `applyCredentials` with mock executor for each credential type (SMB, SSH key, SSH password), nil credentials path, error path
- **Integration:** create profile → create job with profile → fetch jobs as agent → verify credentials present and decrypted correctly

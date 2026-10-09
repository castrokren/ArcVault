# ArcVault Backup Agent — API Reference

The backup agent exposes a REST API served by FastAPI/uvicorn, typically on port **8443**. All endpoints that mutate state or return sensitive data require authentication. The `/health` and `/info` endpoints are public.

---

## Table of Contents

- [Base URL](#base-url)
- [Authentication](#authentication)
  - [Static Token](#static-token)
  - [RS256 JWT](#rs256-jwt)
  - [Error Responses](#authentication-error-responses)
- [Endpoints](#endpoints)
  - [GET /health](#get-health)
  - [GET /info](#get-info)
  - [POST /jobs](#post-jobs)
  - [GET /jobs](#get-jobs)
  - [GET /jobs/{job_id}](#get-jobsjob_id)
  - [GET /jobs/{job_id}/logs](#get-jobsjob_idlogs)
  - [POST /shutdown](#post-shutdown)
  - [POST /maintenance/cleanup](#post-maintenancecleanup)
- [Data Models](#data-models)
  - [Job Object](#job-object)
  - [BackupConfig](#backupconfig)
  - [BackupOptions](#backupoptions)
  - [Credentials](#credentials)
  - [Job States](#job-states)
- [robocopy Exit Code Mapping](#robocopy-exit-code-mapping)
- [Error Reference](#error-reference)

---

## Base URL

```
https://<agent-host>:<port>
```

Default port is `8443`. TLS is optional in development but required in production (configure via `AGENT_CERTFILE` / `AGENT_KEYFILE`).

---

## Authentication

Every protected endpoint reads the token from one of two headers (checked in this order):

1. `X-API-Key: <token>`
2. `Authorization: Bearer <token>`

### Static Token

For development or legacy deployments. Set the same value on both the agent and the controller:

```
AGENT_TOKEN=my-secret-token
```

```http
X-API-Key: my-secret-token
```

### RS256 JWT

Recommended for production. Set `AGENT_REQUIRE_JWT=true` on the agent and provide the controller's RSA public key:

```
AGENT_REQUIRE_JWT=true
AGENT_JWT_PUBLIC_KEY_PATH=/opt/arcvault/keys/controller_public.pem
AGENT_JWT_ISSUER=backup-controller
AGENT_JWT_AUDIENCE=backup-agent
AGENT_JWT_LEEWAY=60
```

The controller mints short-lived JWTs (15-minute TTL, RS256) and sends them as the `X-API-Key` value. Required claims: `iss`, `aud`, `iat`, `exp`.

When both JWT public key and `AGENT_REQUIRE_JWT=false` are configured, the agent auto-detects JWTs (three-part dot-separated string) and validates them; otherwise it falls back to static token comparison.

### Authentication Error Responses

| Status | Condition |
|--------|-----------|
| `401 Unauthorized` | Missing or invalid token / JWT validation failed |

```json
{"detail": "Unauthorized"}
{"detail": "JWT required"}
{"detail": "Invalid JWT: Signature verification failed"}
```

---

## Endpoints

---

### GET /health

Liveness check. No authentication required. Use this for load-balancer health probes.

**Request**

```http
GET /health HTTP/1.1
```

**Response `200 OK`**

```json
{
  "status": "ok",
  "version": "0.1.0"
}
```

---

### GET /info

Self-report endpoint. No authentication required. Returns the agent's public URL (used by the controller for auto-discovery), current job counts, and concurrency limits.

**Request**

```http
GET /info HTTP/1.1
```

**Response `200 OK`**

```json
{
  "public_url": "https://SRB3FLPC010.domain.com:8443",
  "hostname": "SRB3FLPC010",
  "version": "0.1.0",
  "job_counts": {
    "total": 42,
    "queued": 0,
    "running": 1,
    "succeeded": 38,
    "failed": 3
  },
  "max_concurrent_jobs": 3,
  "available_slots": 2
}
```

**Environment overrides**

| Variable | Effect |
|----------|--------|
| `AGENT_PUBLIC_HOST` | Override the hostname in `public_url` |
| `AGENT_PUBLIC_URL` | Override the full `public_url` value |
| `AGENT_PUBLIC_SCHEME` | Override the scheme (`http`/`https`, default `https`) |

---

### POST /jobs

Create and enqueue a backup job. The job runs in a background thread and the response returns immediately with the assigned `job_id`.

**Authentication:** Required

**Request**

```http
POST /jobs HTTP/1.1
Content-Type: application/json
X-API-Key: <token>
```

```json
{
  "sources": [
    "C:\\Users\\Shared\\Documents",
    "D:\\ProjectData"
  ],
  "destination": "\\\\fileserver\\backups\\machine01",
  "options": {
    "include_empty_dirs": true,
    "verbose": true,
    "report_extras": true,
    "retries": 3,
    "wait_seconds": 10,
    "threads": 16,
    "copy_symlinks": false,
    "remove_system_hidden": true
  },
  "source_credentials": {
    "username": "backupuser",
    "password": "s3cret",
    "domain": "CORP"
  },
  "destination_credentials": {
    "username": "backupwriter",
    "password": "wr1ter",
    "domain": "CORP"
  }
}
```

`options` and `*_credentials` are optional. See [BackupConfig](#backupconfig) for field defaults.

**Response `200 OK`**

```json
{
  "job_id": "a3f8c21d4e6b7890abcdef1234567890"
}
```

**Response `409 Conflict` — source path overlap**

Returned when an existing queued or running job already covers one of the requested source paths (prefix match, case-insensitive).

```json
{
  "detail": {
    "error": "Source path conflicts detected",
    "conflicts": [
      "Job a3f8c21d already using overlapping path: C:\\Users\\Shared"
    ]
  }
}
```

**Response `422 Unprocessable Entity`**

Pydantic validation failure — malformed request body (e.g. empty `sources`, missing `destination`).

---

### GET /jobs

List all jobs known to this agent (from database + in-memory cache), ordered newest first.

**Authentication:** Required

**Request**

```http
GET /jobs HTTP/1.1
X-API-Key: <token>
```

**Response `200 OK`**

```json
{
  "jobs": [
    {
      "job_id": "a3f8c21d4e6b7890abcdef1234567890",
      "state": "succeeded",
      "progress": 0,
      "error": null,
      "sources": ["C:\\Users\\Shared\\Documents"],
      "created_at": 1728470400.123,
      "started_at": 1728470401.456,
      "finished_at": 1728470520.789,
      "updated_at": 1728470520.789
    },
    {
      "job_id": "b1c2d3e4f5a6b7c8d9e0f1a2b3c4d5e6",
      "state": "running",
      "progress": 0,
      "error": null,
      "sources": ["D:\\ProjectData"],
      "created_at": 1728470600.000,
      "started_at": 1728470601.000,
      "finished_at": null,
      "updated_at": 1728470650.000
    }
  ]
}
```

---

### GET /jobs/{job_id}

Get the current state of a specific job.

**Authentication:** Required

**Request**

```http
GET /jobs/a3f8c21d4e6b7890abcdef1234567890 HTTP/1.1
X-API-Key: <token>
```

**Response `200 OK`**

```json
{
  "job_id": "a3f8c21d4e6b7890abcdef1234567890",
  "state": "succeeded",
  "progress": 0,
  "error": null,
  "sources": ["C:\\Users\\Shared\\Documents"],
  "created_at": 1728470400.123,
  "started_at": 1728470401.456,
  "finished_at": 1728470520.789,
  "updated_at": 1728470520.789
}
```

**Response `404 Not Found`**

```json
{"detail": "Not found"}
```

---

### GET /jobs/{job_id}/logs

Stream live log output for a job using **Server-Sent Events (SSE)**. The connection stays open while the job is running and closes with a final event when the job completes.

**Authentication:** Required

**Request**

```http
GET /jobs/a3f8c21d4e6b7890abcdef1234567890/logs HTTP/1.1
X-API-Key: <token>
Accept: text/event-stream
```

**Response `200 OK`** — `Content-Type: text/event-stream`

Each log line is delivered as an SSE `data:` frame:

```
data: Backup started - Log file: \\fileserver\backups\machine01\#backup_logs\backup_20261009_143200.log

data: Destination disk space: 128 GB free

data: Source 1: C:\Users\Shared\Documents - accessible

data: NET USE connect \\fileserver\backups (attempt 1/3)

data: ✓ Connected to \\fileserver\backups

data: Executing robocopy 1/1: robocopy C:\Users\Shared\Documents \\fileserver\backups\machine01 /E /V /X /R:3 /W:10 /MT:16 /A-:SH

data: ...robocopy output lines...

data: robocopy finished with code 1 -> succeeded

data: Backup completed successfully - all sources processed

data: Job completed successfully
```

The stream ends (connection closes) after the sentinel is sent when the job finishes.

**Response `404 Not Found`**

Returned if the `job_id` has no active log buffer (job may be from a previous agent session — check disk logs instead).

```json
{"detail": "Not found"}
```

**Notes**

- The in-memory buffer holds up to 1 000 lines. Older lines are evicted when near capacity. For complete historical logs, read the file at `{destination}/#backup_logs/backup_<timestamp>.log`.
- To consume from a browser or `curl`: `curl -N -H "X-API-Key: <token>" https://<host>:8443/jobs/<id>/logs`

---

### POST /shutdown

Gracefully shut down the agent process. The response is sent before the process exits.

**Authentication:** Required

**Request**

```http
POST /shutdown HTTP/1.1
X-API-Key: <token>
```

**Response `200 OK`**

```json
{"status": "shutting_down"}
```

The process exits ~250ms after the response is sent.

---

### POST /maintenance/cleanup

Delete completed job records (state `succeeded` or `failed`) and their associated log files that are older than the specified number of days. Running and queued jobs are never deleted.

**Authentication:** Required

**Query Parameters**

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `days` | integer | `30` | Delete jobs older than this many days |

**Request**

```http
POST /maintenance/cleanup?days=14 HTTP/1.1
X-API-Key: <token>
```

**Response `200 OK`**

```json
{
  "deleted_jobs": 12,
  "message": "Cleanup complete"
}
```

---

## Data Models

### Job Object

| Field | Type | Description |
|-------|------|-------------|
| `job_id` | string | Hex UUID, 32 characters |
| `state` | string | See [Job States](#job-states) |
| `progress` | integer | Reserved for future use (always `0` currently) |
| `error` | string \| null | Error message when `state` is `failed` |
| `sources` | array[string] | Source paths as submitted |
| `created_at` | float | Unix timestamp — job creation |
| `started_at` | float \| null | Unix timestamp — execution start |
| `finished_at` | float \| null | Unix timestamp — execution end |
| `updated_at` | float | Unix timestamp — last state change |

---

### BackupConfig

Sent as the request body to `POST /jobs`.

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `sources` | array[string] | ✅ | — | One or more source paths. Local paths or UNC (`\\server\share\path`). At least one required. |
| `destination` | string | ✅ | — | Destination path. Must exist and be writable unless `BACKUP_SKIP_DESTINATION_CHECK=true`. |
| `options` | BackupOptions | — | See below | robocopy flags |
| `source_credentials` | Credentials | — | null | Credentials for mounting source UNC share |
| `destination_credentials` | Credentials | — | null | Credentials for mounting destination UNC share |

---

### BackupOptions

All fields are optional. Unset fields use the defaults shown.

| Field | Type | Default | robocopy flag | Description |
|-------|------|---------|---------------|-------------|
| `include_empty_dirs` | boolean | `true` | `/E` | Copy subdirectories including empty ones |
| `verbose` | boolean | `true` | `/V` | Verbose output — log all files and sizes |
| `report_extras` | boolean | `true` | `/X` | Report files in destination not in source |
| `retries` | integer (0–1000) | `1` | `/R:n` | Retry count on failed file copies |
| `wait_seconds` | integer (0–60) | `5` | `/W:n` | Wait time in seconds between retries |
| `threads` | integer (1–128) | `12` | `/MT:n` | Multithreaded copy threads |
| `copy_symlinks` | boolean | `false` | `/SL` | Copy symbolic links instead of targets |
| `remove_system_hidden` | boolean | `true` | `/A-:SH` | Remove System and Hidden attributes from copied files |

---

### Credentials

Used for UNC network share authentication via `net use`. Passwords are encrypted with Fernet before being stored in memory or on disk.

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `username` | string | ✅ | Account username |
| `password` | string | ✅ | Account password (plain text in request; encrypted at rest) |
| `domain` | string | — | Windows domain. If provided, passed as `DOMAIN\username` to `net use`. |

---

### Job States

| State | Description |
|-------|-------------|
| `queued` | Accepted but waiting for a concurrency slot (`AGENT_MAX_CONCURRENT_JOBS` limit reached) |
| `running` | robocopy subprocess is active |
| `succeeded` | All sources copied successfully (robocopy exit 0–7) |
| `failed` | Backup failed — check `error` field for reason |

---

## robocopy Exit Code Mapping

ArcVault maps robocopy's numeric exit codes to job states:

| Exit Code | State | Meaning |
|-----------|-------|---------|
| 0 | `succeeded` | No files copied — source and destination already in sync |
| 1 | `succeeded` | Files copied successfully |
| 2–7 | `succeeded` | Copy succeeded with informational differences (extras, mismatches, etc.) |
| 8–16 | `failed` | Copy failed — at least one file could not be copied |

---

## Error Reference

| Status | Meaning |
|--------|---------|
| `200 OK` | Success |
| `401 Unauthorized` | Missing, invalid, or expired authentication token |
| `404 Not Found` | Job ID does not exist |
| `409 Conflict` | Source path overlaps with a running or queued job |
| `422 Unprocessable Entity` | Request body failed Pydantic validation |
| `500 Internal Server Error` | Unexpected server error |

All error responses follow FastAPI's default format:

```json
{"detail": "<message>"}
```

or for structured errors (409):

```json
{
  "detail": {
    "error": "<summary>",
    "conflicts": ["<description>", ...]
  }
}
```

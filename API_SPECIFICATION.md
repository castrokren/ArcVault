# ArcVault API Specification v0.3.0

**Version:** 0.3.0 (Post-Refactor)  
**Last Updated:** June 4, 2026  
**Status:** Pre-Validation Phase (Step 7)

---

## Overview

This document defines the complete API contract for ArcVault Coordinator. All endpoints follow REST conventions with JSON request/response bodies.

**Base URL:** `http://localhost:8080/api`  
**Authentication:** JWT Bearer token in `Authorization` header

---

## General Response Format

### Success Response (2xx)
```json
{
  "data": { /* response body */ },
  "error": null
}
```

### Error Response (4xx, 5xx)
```json
{
  "error": "error message",
  "status": 400
}
```

### Pagination
```json
{
  "data": [ /* items */ ],
  "pagination": {
    "page": 1,
    "limit": 20,
    "total": 150,
    "pages": 8
  }
}
```

---

## Authentication API

### POST /auth/login
**Purpose:** Authenticate user and receive JWT token

**Request:**
```json
{
  "username": "string (required, 1-255 chars)",
  "password": "string (required, 8+ chars)"
}
```

**Response:** 200 OK
```json
{
  "user_id": "integer",
  "username": "string",
  "token": "string (JWT)",
  "expires_in": 3600,
  "must_change_password": "boolean"
}
```

**Errors:**
- `400` — Missing username or password
- `401` — Invalid credentials (generic: never reveal which field failed)
- `500` — Internal server error

**Validation Rules:**
- Username and password both required
- Password must be 8+ characters
- Username must be 1-255 characters

---

### POST /auth/logout
**Purpose:** Invalidate current session

**Request:** No body

**Response:** 204 No Content

**Errors:**
- `401` — Unauthenticated
- `500` — Internal server error

---

### GET /auth/me
**Purpose:** Get current authenticated user claims from JWT

**Request:** No body

**Response:** 200 OK
```json
{
  "user_id": "integer",
  "username": "string",
  "role": "string (admin|viewer)",
  "must_change_password": "boolean"
}
```

**Errors:**
- `401` — Unauthenticated
- `500` — Internal server error

---

### POST /auth/refresh
**Purpose:** Get new JWT token (extends session)

**Request:** No body (uses current token)

**Response:** 200 OK
```json
{
  "token": "string (new JWT)",
  "expires_in": 3600
}
```

**Errors:**
- `401` — Unauthenticated
- `500` — Internal server error

---

### POST /auth/change-password
**Purpose:** Change current user's password

**Request:**
```json
{
  "old_password": "string (required, 8+ chars)",
  "new_password": "string (required, 8+ chars)"
}
```

**Response:** 200 OK
```json
{
  "message": "Password changed successfully"
}
```

**Errors:**
- `400` — Passwords don't match validation rules
- `401` — Old password incorrect
- `401` — Unauthenticated
- `500` — Internal server error

**Validation Rules:**
- Both passwords required
- Each must be 8+ characters
- New password must differ from old password
- Clears `must_change_password` flag on success

---

## User Management API

### GET /users
**Purpose:** List all users (admin only)

**Query Params:**
- `page` — Page number (default: 1, min: 1)
- `limit` — Items per page (default: 20, min: 1, max: 100)

**Request:** No body

**Response:** 200 OK
```json
{
  "data": [
    {
      "user_id": "integer",
      "username": "string",
      "role": "string (admin|viewer)",
      "must_change_password": "boolean",
      "created_at": "string (ISO 8601)"
    }
  ],
  "pagination": {
    "page": 1,
    "limit": 20,
    "total": 50,
    "pages": 3
  }
}
```

**Errors:**
- `401` — Unauthenticated
- `403` — Not authorized (not admin)
- `500` — Internal server error

---

### POST /users
**Purpose:** Create new user (admin only)

**Request:**
```json
{
  "username": "string (required, 1-255 chars, unique)",
  "password": "string (required, 8+ chars)",
  "role": "string (required, admin|viewer)"
}
```

**Response:** 201 Created
```json
{
  "user_id": "integer",
  "username": "string",
  "role": "string",
  "must_change_password": true,
  "created_at": "string (ISO 8601)"
}
```

**Errors:**
- `400` — Missing required fields or validation failed
- `400` — Username already exists
- `401` — Unauthenticated
- `403` — Not authorized (not admin)
- `500` — Internal server error

**Validation Rules:**
- Username: 1-255 characters, unique, alphanumeric + underscore
- Password: 8+ characters
- Role: must be "admin" or "viewer"
- New users always have `must_change_password = true`

---

### DELETE /users/{user_id}
**Purpose:** Delete user (admin only, cannot self-delete)

**Request:** No body

**Response:** 204 No Content

**Errors:**
- `400` — Cannot delete self
- `401` — Unauthenticated
- `403` — Not authorized (not admin)
- `404` — User not found
- `500` — Internal server error

---

### PATCH /users/{user_id}/role
**Purpose:** Update user role (admin only)

**Request:**
```json
{
  "role": "string (required, admin|viewer)"
}
```

**Response:** 200 OK
```json
{
  "user_id": "integer",
  "username": "string",
  "role": "string",
  "updated_at": "string (ISO 8601)"
}
```

**Errors:**
- `400` — Invalid role value
- `401` — Unauthenticated
- `403` — Not authorized (not admin)
- `404` — User not found
- `500` — Internal server error

**Validation Rules:**
- Role must be "admin" or "viewer"

---

## Group Management API

### GET /groups
**Purpose:** List all agent groups with member counts

**Query Params:**
- `page` — Page number (default: 1, min: 1)
- `limit` — Items per page (default: 20, min: 1, max: 100)

**Request:** No body

**Response:** 200 OK
```json
{
  "data": [
    {
      "group_id": "integer",
      "name": "string",
      "description": "string",
      "agent_count": "integer",
      "created_at": "string (ISO 8601)"
    }
  ],
  "pagination": {
    "page": 1,
    "limit": 20,
    "total": 12,
    "pages": 1
  }
}
```

**Errors:**
- `401` — Unauthenticated
- `500` — Internal server error

---

### POST /groups
**Purpose:** Create new agent group

**Request:**
```json
{
  "name": "string (required, 1-255 chars)",
  "description": "string (optional, 0-1000 chars)"
}
```

**Response:** 201 Created
```json
{
  "group_id": "integer",
  "name": "string",
  "description": "string",
  "agent_count": 0,
  "created_at": "string (ISO 8601)"
}
```

**Errors:**
- `400` — Name is required
- `400` — Name must be 1-255 characters
- `401` — Unauthenticated
- `500` — Internal server error

**Validation Rules:**
- Name: required, 1-255 characters
- Description: optional, 0-1000 characters

---

### GET /groups/{group_id}
**Purpose:** Get single group with agent count

**Request:** No body

**Response:** 200 OK
```json
{
  "group_id": "integer",
  "name": "string",
  "description": "string",
  "agent_count": "integer",
  "created_at": "string (ISO 8601)"
}
```

**Errors:**
- `401` — Unauthenticated
- `404` — Group not found
- `500` — Internal server error

---

### PATCH /groups/{group_id}
**Purpose:** Update group name/description

**Request:**
```json
{
  "name": "string (optional, 1-255 chars)",
  "description": "string (optional, 0-1000 chars)"
}
```

**Response:** 200 OK
```json
{
  "group_id": "integer",
  "name": "string",
  "description": "string",
  "agent_count": "integer",
  "updated_at": "string (ISO 8601)"
}
```

**Errors:**
- `400` — Validation failed
- `401` — Unauthenticated
- `404` — Group not found
- `500` — Internal server error

---

### DELETE /groups/{group_id}
**Purpose:** Delete group

**Request:** No body

**Response:** 204 No Content

**Errors:**
- `401` — Unauthenticated
- `404` — Group not found
- `500` — Internal server error

---

### POST /groups/{group_id}/agents/{agent_id}
**Purpose:** Add agent to group

**Request:** No body

**Response:** 200 OK
```json
{
  "message": "Agent added to group"
}
```

**Errors:**
- `401` — Unauthenticated
- `404` — Group or agent not found
- `409` — Agent already in group
- `500` — Internal server error

---

### DELETE /groups/{group_id}/agents/{agent_id}
**Purpose:** Remove agent from group

**Request:** No body

**Response:** 204 No Content

**Errors:**
- `401` — Unauthenticated
- `404` — Group or agent not found
- `500` — Internal server error

---

### GET /groups/{group_id}/agents
**Purpose:** List all agents in group

**Query Params:**
- `page` — Page number (default: 1, min: 1)
- `limit` — Items per page (default: 20, min: 1, max: 100)

**Request:** No body

**Response:** 200 OK
```json
{
  "data": [
    {
      "agent_id": "string",
      "hostname": "string",
      "os": "string",
      "arch": "string",
      "version": "string",
      "status": "string (online|offline)",
      "last_seen": "string (ISO 8601, nullable)",
      "registered_at": "string (ISO 8601)"
    }
  ],
  "pagination": {
    "page": 1,
    "limit": 20,
    "total": 5,
    "pages": 1
  }
}
```

**Errors:**
- `401` — Unauthenticated
- `404` — Group not found
- `500` — Internal server error

---

## Agent Management API

### POST /agents/register
**Purpose:** Register/update agent (agent-initiated)

**Request:**
```json
{
  "agent_id": "string (required, 36 chars, UUID format)",
  "hostname": "string (required, 1-255 chars)",
  "os": "string (required, linux|darwin|windows)",
  "arch": "string (required, amd64|arm64|etc)",
  "version": "string (required, semver format)"
}
```

**Response:** 201 Created or 200 OK
```json
{
  "agent_id": "string",
  "status": "string (online)",
  "last_seen": "string (ISO 8601)",
  "registered_at": "string (ISO 8601)"
}
```

**Errors:**
- `400` — Missing required fields or validation failed
- `401` — Invalid agent token
- `500` — Internal server error

**Validation Rules:**
- agent_id: 36 characters, UUID v4 format
- hostname: 1-255 characters
- os: must be linux, darwin, or windows
- arch: must be valid (amd64, arm64, etc)
- version: must match semver (v0.1.0, etc)

---

### POST /agents/{agent_id}/heartbeat
**Purpose:** Agent heartbeat / status update

**Request:**
```json
{
  "rollback_available": "boolean"
}
```

**Response:** 200 OK
```json
{
  "status": "string (online)",
  "last_seen": "string (ISO 8601)"
}
```

**Errors:**
- `401` — Invalid agent token
- `404` — Agent not found
- `500` — Internal server error

---

### GET /agents
**Purpose:** List all agents with pagination and filters

**Query Params:**
- `page` — Page number (default: 1, min: 1)
- `limit` — Items per page (default: 20, min: 1, max: 100)
- `search` — Search hostname (optional, substring match)
- `status` — Filter by status (optional, online|offline)

**Request:** No body

**Response:** 200 OK
```json
{
  "data": [
    {
      "agent_id": "string",
      "hostname": "string",
      "os": "string",
      "arch": "string",
      "version": "string",
      "status": "string (online|offline)",
      "last_seen": "string (ISO 8601, nullable)",
      "registered_at": "string (ISO 8601)",
      "rollback_available": "boolean"
    }
  ],
  "pagination": {
    "page": 1,
    "limit": 20,
    "total": 45,
    "pages": 3
  }
}
```

**Errors:**
- `400` — Invalid pagination params
- `401` — Unauthenticated
- `500` — Internal server error

---

### GET /agents/{agent_id}
**Purpose:** Get single agent details

**Request:** No body

**Response:** 200 OK
```json
{
  "agent_id": "string",
  "hostname": "string",
  "os": "string",
  "arch": "string",
  "version": "string",
  "status": "string (online|offline)",
  "last_seen": "string (ISO 8601, nullable)",
  "registered_at": "string (ISO 8601)",
  "rollback_available": "boolean"
}
```

**Errors:**
- `401` — Unauthenticated
- `404` — Agent not found
- `500` — Internal server error

---

### DELETE /agents/{agent_id}
**Purpose:** Delete agent (admin only)

**Request:** No body

**Response:** 204 No Content

**Errors:**
- `400` — Agent has running jobs
- `401` — Unauthenticated
- `403` — Not authorized (not admin)
- `404` — Agent not found
- `500` — Internal server error

---

## Job Management API

### POST /jobs
**Purpose:** Create new job (single agent or group dispatch)

**Request (Single Agent):**
```json
{
  "name": "string (required, 1-255 chars)",
  "agent_id": "string (required, UUID format) [use this OR group_id]",
  "source_path": "string (required, 1-4096 chars)",
  "dest_path": "string (required, 1-4096 chars)",
  "schedule": "string (optional, cron format)",
  "sync_flags": "string (optional, rsync flags)"
}
```

**Request (Group Dispatch):**
```json
{
  "name": "string (required, 1-255 chars)",
  "group_id": "integer (required) [use this OR agent_id]",
  "source_path": "string (required, 1-4096 chars)",
  "dest_path": "string (required, 1-4096 chars)",
  "schedule": "string (optional, cron format)",
  "sync_flags": "string (optional, rsync flags)"
}
```

**Response:** 201 Created
```json
{
  "job_id": "string (UUID)",
  "name": "string",
  "agent_id": "string (for single) OR null (for group)",
  "group_id": "integer (for group) OR null (for single)",
  "dispatch_id": "string (for group) OR null (for single)",
  "source_path": "string",
  "dest_path": "string",
  "schedule": "string (nullable)",
  "sync_flags": "string (nullable)",
  "status": "string (pending|scheduled|running|completed|failed|cancelled)",
  "created_at": "string (ISO 8601)"
}
```

**Errors:**
- `400` — Missing required fields or validation failed
- `400` — Both agent_id and group_id provided (choose one)
- `400` — Neither agent_id nor group_id provided
- `400` — Group is empty (group dispatch)
- `401` — Unauthenticated
- `404` — Agent or group not found
- `500` — Internal server error

**Validation Rules:**
- name: required, 1-255 characters
- source_path: required, 1-4096 characters
- dest_path: required, 1-4096 characters
- Must provide agent_id OR group_id, not both
- Group must have at least 1 member (for group dispatch)
- schedule: optional, must be valid cron expression if provided
- sync_flags: optional, rsync flags string

---

### GET /jobs
**Purpose:** List all jobs with pagination and filters

**Query Params:**
- `page` — Page number (default: 1, min: 1)
- `limit` — Items per page (default: 20, min: 1, max: 100)
- `search` — Search job name (optional, substring match)
- `status` — Filter by status (optional, pending|scheduled|running|completed|failed|cancelled)
- `agent_id` — Filter by agent (optional)

**Request:** No body

**Response:** 200 OK
```json
{
  "data": [
    {
      "job_id": "string",
      "name": "string",
      "agent_id": "string",
      "source_path": "string",
      "dest_path": "string",
      "schedule": "string (nullable)",
      "sync_flags": "string (nullable)",
      "status": "string",
      "created_at": "string (ISO 8601)"
    }
  ],
  "pagination": {
    "page": 1,
    "limit": 20,
    "total": 150,
    "pages": 8
  }
}
```

**Errors:**
- `400` — Invalid pagination params
- `401` — Unauthenticated
- `500` — Internal server error

---

### GET /jobs/{job_id}
**Purpose:** Get single job details

**Request:** No body

**Response:** 200 OK
```json
{
  "job_id": "string",
  "name": "string",
  "agent_id": "string",
  "source_path": "string",
  "dest_path": "string",
  "schedule": "string (nullable)",
  "sync_flags": "string (nullable)",
  "status": "string",
  "created_at": "string (ISO 8601)"
}
```

**Errors:**
- `401` — Unauthenticated
- `404` — Job not found
- `500` — Internal server error

---

### DELETE /jobs/{job_id}
**Purpose:** Delete job

**Request:** No body

**Response:** 204 No Content

**Errors:**
- `401` — Unauthenticated
- `404` — Job not found
- `500` — Internal server error

---

### POST /jobs/{job_id}/cancel
**Purpose:** Cancel running job

**Request:** No body

**Response:** 200 OK
```json
{
  "message": "Job cancelled"
}
```

**Errors:**
- `400` — Job is not running
- `401` — Unauthenticated
- `404` — Job not found
- `500` — Internal server error

---

### GET /jobs/{job_id}/runs
**Purpose:** List job execution history

**Query Params:**
- `page` — Page number (default: 1, min: 1)
- `limit` — Items per page (default: 20, min: 1, max: 100)

**Request:** No body

**Response:** 200 OK
```json
{
  "data": [
    {
      "run_id": "string (UUID)",
      "job_id": "string",
      "run_start": "string (ISO 8601)",
      "run_end": "string (ISO 8601, nullable)",
      "status": "string (running|completed|failed)",
      "exit_code": "integer (nullable)",
      "error": "string (nullable)"
    }
  ],
  "pagination": {
    "page": 1,
    "limit": 20,
    "total": 42,
    "pages": 3
  }
}
```

**Errors:**
- `400` — Invalid pagination params
- `401` — Unauthenticated
- `404` — Job not found
- `500` — Internal server error

---

### POST /jobs/{job_id}/results
**Purpose:** Post job execution results (agent-initiated)

**Request:**
```json
{
  "run_id": "string (required, UUID)",
  "exit_code": "integer (required, 0-255)",
  "error": "string (optional)"
}
```

**Response:** 201 Created
```json
{
  "run_id": "string",
  "job_id": "string",
  "status": "string (completed|failed)",
  "exit_code": "integer",
  "error": "string (nullable)"
}
```

**Errors:**
- `400` — Missing required fields or validation failed
- `401` — Invalid agent token
- `404` — Job or run not found
- `500` — Internal server error

**Validation Rules:**
- run_id: required, UUID v4 format
- exit_code: required, 0-255
- error: optional, error message string

---

### POST /jobs/{job_id}/progress
**Purpose:** Post job progress update (agent-initiated)

**Request:**
```json
{
  "percentage": "integer (required, 0-100)",
  "status": "string (optional, pending|in_progress|completed|failed)"
}
```

**Response:** 200 OK
```json
{
  "percentage": "integer",
  "status": "string",
  "updated_at": "string (ISO 8601)"
}
```

**Errors:**
- `400` — Invalid percentage (not 0-100) or invalid status
- `401` — Invalid agent token
- `404` — Job not found
- `500` — Internal server error

**Validation Rules:**
- percentage: required, 0-100
- status: optional, must be pending|in_progress|completed|failed if provided

---

### GET /jobs/{job_id}/progress
**Purpose:** Get current job progress and recent logs

**Request:** No body

**Response:** 200 OK
```json
{
  "job_id": "string",
  "percentage": "integer (0-100)",
  "status": "string",
  "recent_logs": [
    {
      "timestamp": "string (ISO 8601)",
      "percentage": "integer",
      "status": "string"
    }
  ],
  "updated_at": "string (ISO 8601)"
}
```

**Errors:**
- `401` — Unauthenticated
- `404` — Job not found
- `500` — Internal server error

---

### GET /jobs/{job_id}/logs
**Purpose:** Get all job logs with pagination

**Query Params:**
- `page` — Page number (default: 1, min: 1)
- `limit` — Items per page (default: 20, min: 1, max: 100)

**Request:** No body

**Response:** 200 OK
```json
{
  "data": [
    {
      "timestamp": "string (ISO 8601)",
      "percentage": "integer (0-100)",
      "status": "string"
    }
  ],
  "pagination": {
    "page": 1,
    "limit": 20,
    "total": 500,
    "pages": 25
  }
}
```

**Errors:**
- `400` — Invalid pagination params
- `401` — Unauthenticated
- `404` — Job not found
- `500` — Internal server error

---

## Breaking Changes

**None.** All changes in v0.3.0 are additive (new layers) or refactored internals. API surface is unchanged.

---

## Implementation Checklist

- [ ] Phase 1: Request/response type definitions
- [ ] Phase 2: Input validation in handlers
- [ ] Phase 3: Error message standardization
- [ ] Phase 4: Test coverage verification
- [ ] Phase 5: Documentation finalization

---

## Notes for Implementation

1. **Error Messages:** Never reveal which field failed in login (use generic "Invalid credentials")
2. **Passwords:** Never return password hashes in any response
3. **Timestamps:** Always ISO 8601 format with timezone
4. **Pagination:** Always include pagination metadata when listing
5. **IDs:** UUIDs for jobs/runs/agents, integers for users/groups
6. **Status Codes:** Follow HTTP conventions (201 for created, 204 for deleted, 400 for invalid, 401 for auth, 403 for permission, 404 for not found, 500 for server error)

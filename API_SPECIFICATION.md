# ArcVault API Specification v0.5.1

**Version:** v0.5.1  
**Last Updated:** July 1, 2026  
**Status:** Active — tracking coordinator v0.5.1 (deprecated progress endpoints excluded)

---

## Overview

This document defines the complete API contract for ArcVault Coordinator. All endpoints follow REST conventions with JSON request/response bodies.

**Base URL:** `https://<coordinator>/api`  
**Authentication:** JWT Bearer token in `Authorization` header (except public endpoints)

**Roles:**
- `admin` — Full access, all routes
- `operator` — Can create/manage jobs, run templates, cancel jobs
- `viewer` — Read-only (list agents, jobs, alerts, audit, etc.)
- **agent token** — Special auth for agent-initiated endpoints (register, heartbeat, post results)
- **public** — No auth required (`/health`, `/api/auth/login`)

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
  "total": 150,
  "page": 1,
  "pages": 8,
  "limit": 25
}
```

Default limit: **25** (max 100). Page default: 1.

**Note:** Some endpoints (credential profiles, alert rules, alert history) return bare arrays without pagination. These are documented individually in their respective sections.

---

## System Endpoints

### GET /health
**Purpose:** Health check (no auth required)

**Request:** No body

**Response:** 200 OK
```json
{
  "status": "ok"
}
```

---

### GET /api/version
**Purpose:** Get coordinator version and system information (viewer+)

**Request:** No body

**Response:** 200 OK
```json
{
  "version": "v0.5.1",
  "build_time": "2026-06-30T12:00:00Z",
  "os": "windows",
  "arch": "amd64",
  "go_version": "go1.25.0",
  "uptime": "3h12m5s"
}
```

**Errors:**
- `401` — Unauthenticated

---

### GET /ws
**Purpose:** Dashboard WebSocket for real-time job/agent updates (JWT required)

**Request:** WebSocket upgrade with `Authorization: Bearer <JWT>` header (or `Sec-WebSocket-Protocol: bearer.<JWT>` for browser clients)

**Auth:** Valid JWT (admin/operator/viewer)

---

### GET /ws/agent
**Purpose:** Agent WebSocket for receiving commands (agent token)

**Request:** WebSocket upgrade with `Authorization: Bearer <agent-token>` header

**Auth:** Valid agent token or admin token

---

### GET /ws/federation
**Purpose:** Federation peer WebSocket connection (mTLS peer certificate)

**Request:** WebSocket upgrade with mTLS client certificate

**Auth:** Valid peer certificate (mTLS)

---

## Authentication API

### POST /api/auth/login
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
  "role": "string (admin|operator|viewer)",
  "token": "string (JWT)",
  "expires_in": 14400,
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

### POST /api/auth/logout
**Purpose:** Invalidate current session

**Request:** No body

**Response:** 204 No Content

**Errors:**
- `401` — Unauthenticated
- `500` — Internal server error

---

### GET /api/auth/me
**Purpose:** Get current authenticated user claims from JWT

**Request:** No body

**Response:** 200 OK
```json
{
  "id": "integer",
  "username": "string",
  "role": "string (admin|operator|viewer)",
  "must_change_password": "boolean"
}
```

**Errors:**
- `401` — Unauthenticated
- `500` — Internal server error

---

### POST /api/auth/refresh
**Purpose:** Get new JWT token (extends session, re-reads role from DB)

**Request:** No body (uses current token)

**Response:** 200 OK
```json
{
  "token": "string (new JWT)",
  "expires_in": 14400
}
```

**Errors:**
- `401` — Unauthenticated
- `500` — Internal server error

---

### PUT /api/auth/change-password
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

### GET /api/users
**Purpose:** List all users (admin only)

**Query Params:**
- `page` — Page number (default: 1, min: 1)
- `limit` — Items per page (default: 25, min: 1, max: 100)

**Request:** No body

**Response:** 200 OK
```json
{
  "data": [
    {
      "user_id": "integer",
      "username": "string",
      "role": "string (admin|operator|viewer)",
      "must_change_password": "boolean",
      "created_at": "string (ISO 8601)"
    }
  ],
  "total": 50,
  "page": 1,
  "pages": 2,
  "limit": 25
}
```

**Errors:**
- `401` — Unauthenticated
- `403` — Not authorized (not admin)
- `500` — Internal server error

---

### POST /api/users
**Purpose:** Create new user (admin only)

**Request:**
```json
{
  "username": "string (required, 1-255 chars, unique)",
  "password": "string (required, 8+ chars)",
  "role": "string (required, admin|operator|viewer)"
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
- Role: must be "admin", "operator", or "viewer"
- New users always have `must_change_password = true`

---

### DELETE /api/users/{user_id}
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

### PUT /api/users/{user_id}/role
**Purpose:** Update user role (admin only)

**Request:**
```json
{
  "role": "string (required, admin|operator|viewer)"
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
- Role must be "admin", "operator", or "viewer"

---

## Group Management API

### GET /api/groups
**Purpose:** List all agent groups with member counts (viewer+)

**Query Params:**
- `page` — Page number (default: 1, min: 1)
- `limit` — Items per page (default: 25, min: 1, max: 100)

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
  "total": 12,
  "page": 1,
  "pages": 1,
  "limit": 25
}
```

**Errors:**
- `401` — Unauthenticated
- `500` — Internal server error

---

### POST /api/groups
**Purpose:** Create new agent group (admin)

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

### GET /api/groups/{group_id}
**Purpose:** Get single group with agent count (viewer+)

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

### PUT /api/groups/{group_id}
**Purpose:** Update group name/description (admin)

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

### DELETE /api/groups/{group_id}
**Purpose:** Delete group (admin)

**Request:** No body

**Response:** 204 No Content

**Errors:**
- `401` — Unauthenticated
- `404` — Group not found
- `500` — Internal server error

---

### POST /api/groups/{group_id}/agents
**Purpose:** Add agent to group (admin)

**Request:**
```json
{
  "agent_id": "string (required, UUID)"
}
```

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

### DELETE /api/groups/{group_id}/agents/{agent_id}
**Purpose:** Remove agent from group (admin)

**Request:** No body

**Response:** 204 No Content

**Errors:**
- `401` — Unauthenticated
- `404` — Group or agent not found
- `500` — Internal server error

---

### GET /api/groups/{group_id}/agents
**Purpose:** List all agents in group (viewer+)

**Query Params:**
- `page` — Page number (default: 1, min: 1)
- `limit` — Items per page (default: 25, min: 1, max: 100)

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
  "total": 5,
  "page": 1,
  "pages": 1,
  "limit": 25
}
```

**Errors:**
- `401` — Unauthenticated
- `404` — Group not found
- `500` — Internal server error

---

## Agent Management API

### POST /api/agents/register
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

### POST /api/agents/{agent_id}/heartbeat
**Purpose:** Agent heartbeat / status update (agent-initiated)

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

### GET /api/agents
**Purpose:** List all agents with pagination and filters (viewer+)

**Query Params:**
- `page` — Page number (default: 1, min: 1)
- `limit` — Items per page (default: 25, min: 1, max: 100)
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
  "total": 45,
  "page": 1,
  "pages": 2,
  "limit": 25
}
```

**Errors:**
- `400` — Invalid pagination params
- `401` — Unauthenticated
- `500` — Internal server error

---

### GET /api/agents/{agent_id}
**Purpose:** Get single agent details (viewer+)

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

### DELETE /api/agents/{agent_id}
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

## Credential Profiles API

### POST /api/credential-profiles
**Purpose:** Create a new encrypted credential profile (admin)

**Request:**
```json
{
  "name": "string (required, 1-255 chars)",
  "type": "string (required, SMB|SSH|AWS|Database)",
  "data": {
    "key1": "value1",
    "key2": "value2"
  }
}
```

**Response:** 201 Created
```json
{
  "id": "string (cred-xxxxxxxx)",
  "name": "string",
  "type": "string",
  "created_at": "string (ISO 8601)"
}
```

**Errors:**
- `400` — Missing required fields (name, type, data)
- `401` — Unauthenticated
- `403` — Not authorized (not admin)
- `503` — Encryption key not configured
- `500` — Internal server error

**Validation Rules:**
- name: required, 1-255 characters
- type: required, must be a valid credential type
- data: required, must be a JSON object (will be encrypted)
- Encryption key must be configured (via config or env var)

---

### GET /api/credential-profiles
**Purpose:** List all credential profiles (viewer+)  
**Note:** Encrypted data is NOT returned — only metadata

**Request:** No body

**Response:** 200 OK
```json
[
  {
    "id": "string",
    "name": "string",
    "type": "string",
    "created_at": "string (ISO 8601)"
  }
]
```

**Errors:**
- `401` — Unauthenticated
- `500` — Internal server error

---

### DELETE /api/credential-profiles/{id}
**Purpose:** Delete a credential profile (admin)

**Request:** No body

**Response:** 204 No Content

**Errors:**
- `401` — Unauthenticated
- `403` — Not authorized (not admin)
- `404` — Credential profile not found
- `409` — Credential profile is referenced by one or more jobs
- `500` — Internal server error

---

## Templates API

### GET /api/templates
**Purpose:** List all backup templates with pagination and search (viewer+)

**Query Params:**
- `page` — Page number (default: 1, min: 1)
- `limit` — Items per page (default: 25, min: 1, max: 100)
- `search` — Search name or agent_id (optional, substring match)

**Request:** No body

**Response:** 200 OK
```json
{
  "data": [
    {
      "id": "string",
      "name": "string",
      "agent_id": "string",
      "command": "string",
      "schedule": "string (cron expression)",
      "enabled": "boolean",
      "created_at": "string (ISO 8601)",
      "next_run": "string (ISO 8601, nullable)"
    }
  ],
  "total": 10,
  "page": 1,
  "pages": 1,
  "limit": 25
}
```

**Errors:**
- `401` — Unauthenticated
- `500` — Internal server error

---

### POST /api/templates
**Purpose:** Create a new backup template (admin)

**Request:**
```json
{
  "id": "string (required, unique identifier)",
  "name": "string (required, 1-255 chars)",
  "agent_id": "string (required, UUID of target agent)",
  "command": "string (required, backup command to execute)",
  "schedule": "string (required, cron expression)",
  "enabled": "boolean (optional, default: true)"
}
```

**Response:** 201 Created
```json
{
  "id": "string",
  "name": "string",
  "agent_id": "string",
  "command": "string",
  "schedule": "string",
  "enabled": "boolean",
  "created_at": "string (ISO 8601)",
  "next_run": "string (ISO 8601, nullable)"
}
```

**Errors:**
- `400` — Missing required fields or invalid cron expression
- `400` — Agent not found
- `401` — Unauthenticated
- `403` — Not authorized (not admin)
- `409` — Template id already exists
- `500` — Internal server error

**Validation Rules:**
- id: required, must be unique
- name: required, 1-255 characters
- agent_id: required, must reference existing agent
- command: required
- schedule: required, must be valid cron expression

---

### GET /api/templates/{id}
**Purpose:** Get single template details (viewer+)

**Request:** No body

**Response:** 200 OK
```json
{
  "id": "string",
  "name": "string",
  "agent_id": "string",
  "command": "string",
  "schedule": "string",
  "enabled": "boolean",
  "created_at": "string (ISO 8601)",
  "next_run": "string (ISO 8601, nullable)"
}
```

**Errors:**
- `401` — Unauthenticated
- `404` — Template not found
- `500` — Internal server error

---

### PUT /api/templates/{id}
**Purpose:** Update template fields (admin, partial update)

**Request:**
```json
{
  "name": "string (optional)",
  "agent_id": "string (optional)",
  "command": "string (optional)",
  "schedule": "string (optional, cron expression)",
  "enabled": "boolean (optional)"
}
```

**Response:** 200 OK
```json
{
  "id": "string",
  "name": "string",
  "agent_id": "string",
  "command": "string",
  "schedule": "string",
  "enabled": "boolean",
  "created_at": "string (ISO 8601)",
  "next_run": "string (ISO 8601, nullable)"
}
```

**Errors:**
- `400` — Invalid JSON or invalid cron expression
- `401` — Unauthenticated
- `403` — Not authorized (not admin)
- `404` — Template not found
- `500` — Internal server error

---

### DELETE /api/templates/{id}
**Purpose:** Delete a template (admin)

**Request:** No body

**Response:** 204 No Content

**Errors:**
- `401` — Unauthenticated
- `403` — Not authorized (not admin)
- `404` — Template not found
- `500` — Internal server error

---

### POST /api/templates/{id}/run
**Purpose:** Execute a template immediately (operator+)

**Request:** No body

**Response:** 202 Accepted

**Errors:**
- `401` — Unauthenticated
- `403` — Not authorized (not operator+)
- `404` — Template not found
- `500` — Internal server error

---

## Job Management API

### POST /api/jobs
**Purpose:** Create new job — single agent or group dispatch (operator+)

**Request (Single Agent):**
```json
{
  "name": "string (required, 1-255 chars)",
  "agent_id": "string (required, UUID format) [use this OR group_id]",
  "source_path": "string (required, 1-4096 chars)",
  "dest_path": "string (required, 1-4096 chars)",
  "schedule": "string (optional, cron format)",
  "sync_flags": "object (optional, rsync flags)",
  "credentials": "object (optional, credential profile refs)"
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
  "sync_flags": "object (optional)",
  "credentials": "object (optional)"
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
- `403` — Not authorized (not operator+)
- `404` — Agent or group not found
- `500` — Internal server error

**Validation Rules:**
- name: required, 1-255 characters
- source_path: required, 1-4096 characters
- dest_path: required, 1-4096 characters
- Must provide agent_id OR group_id, not both
- Group must have at least 1 member (for group dispatch)
- schedule: optional, must be valid cron expression if provided

---

### GET /api/jobs
**Purpose:** List all jobs with pagination and filters (agent or viewer+)

**Query Params:**
- `page` — Page number (default: 1, min: 1)
- `limit` — Items per page (default: 25, min: 1, max: 100)
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
  "total": 150,
  "page": 1,
  "pages": 6,
  "limit": 25
}
```

**Errors:**
- `400` — Invalid pagination params
- `401` — Unauthenticated
- `500` — Internal server error

---

### GET /api/jobs/{job_id}
**Purpose:** Get single job details (viewer+)

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
  "progress": {
    "percentage": "integer (0-100)",
    "status": "string",
    "updated_at": "string (ISO 8601)"
  },
  "created_at": "string (ISO 8601)"
}
```

**Errors:**
- `401` — Unauthenticated
- `404` — Job not found
- `500` — Internal server error

---

### PATCH /api/jobs/{id}/status
**Purpose:** Update job status (agent or operator+)

**Request:**
```json
{
  "status": "string (required, pending|running|completed|failed|canceling|cancelled)"
}
```

**Response:** 200 OK
```json
{
  "id": "string (UUID)",
  "agent_id": "string",
  "name": "string",
  "source_path": "string",
  "dest_path": "string",
  "schedule": "string (nullable)",
  "sync_flags": "object (nullable)",
  "status": "string",
  "created_at": "string (ISO 8601)"
}
```

**Errors:**
- `400` — Invalid status value
- `400` — Invalid JSON
- `401` — Unauthenticated
- `404` — Job not found
- `500` — Internal server error

**Validation Rules:**
- Status must be one of: pending, running, completed, failed, canceling, cancelled

---

### DELETE /api/jobs/{job_id}
**Purpose:** Delete job (admin)

**Request:** No body

**Response:** 204 No Content

**Errors:**
- `401` — Unauthenticated
- `403` — Not authorized (not admin)
- `404` — Job not found
- `500` — Internal server error

---

### POST /api/jobs/{job_id}/cancel
**Purpose:** Cancel running job (operator+)

**Request:** No body

**Response:** 200 OK
```json
{
  "status": "canceling",
  "job_id": "string"
}
```

**Errors:**
- `400` — Job is not running
- `401` — Unauthenticated
- `403` — Not authorized (not operator+)
- `404` — Job not found
- `500` — Internal server error

---

### GET /api/jobs/{job_id}/runs
**Purpose:** List job execution history (viewer+)

**Query Params:**
- `page` — Page number (default: 1, min: 1)
- `limit` — Items per page (default: 25, min: 1, max: 100)

**Request:** No body

**Response:** 200 OK
```json
{
  "data": [
    {
      "id": "string (UUID)",
      "job_id": "string",
      "exit_code": "integer (nullable)",
      "output": "string (nullable)",
      "started_at": "string (ISO 8601)",
      "finished_at": "string (ISO 8601, nullable)"
    }
  ],
  "total": 42,
  "page": 1,
  "pages": 2,
  "limit": 25
}
```

**Errors:**
- `400` — Invalid pagination params
- `401` — Unauthenticated
- `404` — Job not found
- `500` — Internal server error

---

### GET /api/job-runs
**Purpose:** List all job runs across all jobs with filters (viewer+)

**Query Params:**
- `page` — Page number (default: 1, min: 1)
- `limit` — Items per page (default: 25, min: 1, max: 100)
- `job_id` — Filter by job (optional)
- `agent_id` — Filter by agent (optional)
- `status` — Filter by status (optional)

**Request:** No body

**Response:** 200 OK
```json
{
  "data": [
    {
      "id": "string (UUID)",
      "job_id": "string",
      "job_name": "string",
      "source_path": "string",
      "dest_path": "string",
      "agent_id": "string",
      "agent_hostname": "string",
      "exit_code": "integer (nullable)",
      "output": "string (nullable)",
      "status": "string",
      "started_at": "string (ISO 8601)",
      "finished_at": "string (ISO 8601, nullable)"
    }
  ],
  "total": 200,
  "page": 1,
  "pages": 8,
  "limit": 25
}
```

**Errors:**
- `400` — Invalid pagination params
- `401` — Unauthenticated
- `500` — Internal server error

---

### POST /api/jobs/{job_id}/results
**Purpose:** Post job execution results (agent-initiated)

**Request:**
```json
{
  "exit_code": "integer (required, 0-255)",
  "output": "string (optional)",
  "started_at": "string (optional, ISO 8601)"
}
```

**Response:** 201 Created
```json
{
  "id": "string (UUID)",
  "job_id": "string",
  "exit_code": "integer",
  "output": "string",
  "status": "string (completed|failed)",
  "started_at": "string (ISO 8601)",
  "finished_at": "string (ISO 8601)"
}
```

**Errors:**
- `400` — Missing required fields or validation failed
- `401` — Invalid agent token
- `404` — Job not found
- `500` — Internal server error

**Validation Rules:**
- exit_code: required, 0-255
- output: optional, command output string

---

### GET /api/jobs/{job_id}/logs
**Purpose:** Get all job logs with pagination (viewer+)

**Query Params:**
- `page` — Page number (default: 1, min: 1)
- `limit` — Items per page (default: 25, min: 1, max: 100)

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
  "total": 500,
  "page": 1,
  "pages": 20,
  "limit": 25
}
```

**Errors:**
- `400` — Invalid pagination params
- `401` — Unauthenticated
- `404` — Job not found
- `500` — Internal server error

---

## Federation API

### GET /api/federation
**Purpose:** List all federation peers (admin)

**Request:** No body

**Response:** 200 OK
```json
[
  {
    "id": "string",
    "name": "string",
    "url": "string",
    "status": "string (online|offline)",
    "last_seen": "string (ISO 8601, nullable)",
    "version": "string"
  }
]
```

**Errors:**
- `401` — Unauthenticated
- `403` — Not authorized (not admin)
- `500` — Internal server error

---

### POST /api/federation
**Purpose:** Create a new federation peer connection (admin)

**Request:**
```json
{
  "name": "string (required, 1-255 chars)",
  "url": "string (required, valid URL)",
  "token": "string (required, authentication token)"
}
```

**Response:** 201 Created
```json
{
  "id": "string",
  "name": "string",
  "url": "string",
  "status": "string (offline)",
  "last_seen": "string (ISO 8601, nullable)",
  "version": "string"
}
```

**Errors:**
- `400` — Missing required fields (name, url, token)
- `401` — Unauthenticated
- `403` — Not authorized (not admin)
- `500` — Internal server error

---

### GET /api/federation/{id}
**Purpose:** Get single federation peer details (admin)

**Request:** No body

**Response:** 200 OK
```json
{
  "id": "string",
  "name": "string",
  "url": "string",
  "status": "string (online|offline)",
  "last_seen": "string (ISO 8601, nullable)",
  "version": "string"
}
```

**Errors:**
- `401` — Unauthenticated
- `403` — Not authorized (not admin)
- `404` — Federation peer not found
- `500` — Internal server error

---

### PUT /api/federation/{id}
**Purpose:** Update federation peer (admin, partial update)

**Request:**
```json
{
  "name": "string (optional)",
  "url": "string (optional)",
  "token": "string (optional)"
}
```

**Response:** 200 OK
```json
{
  "id": "string",
  "name": "string",
  "url": "string",
  "status": "string",
  "last_seen": "string (ISO 8601, nullable)",
  "version": "string"
}
```

**Errors:**
- `401` — Unauthenticated
- `403` — Not authorized (not admin)
- `404` — Federation peer not found
- `500` — Internal server error

---

### DELETE /api/federation/{id}
**Purpose:** Delete federation peer (admin)

**Request:** No body

**Response:** 204 No Content

**Errors:**
- `401` — Unauthenticated
- `403` — Not authorized (not admin)
- `404` — Federation peer not found
- `500` — Internal server error

---

### POST /api/federation/{id}/sync
**Purpose:** Trigger a full resync with federation peer (admin)

**Request:** No body

**Response:** 202 Accepted

**Errors:**
- `401` — Unauthenticated
- `403` — Not authorized (not admin)
- `404` — Federation peer not found
- `500` — Internal server error

---

### GET /api/federation/{id}/agents
**Purpose:** Get cached agent list from federation peer (admin)

**Request:** No body

**Response:** 200 OK
```json
{
  "agents": [ /* agent objects */ ],
  "stale": "boolean",
  "as_of": "string (ISO 8601)"
}
```

**Errors:**
- `401` — Unauthenticated
- `403` — Not authorized (not admin)
- `404` — Federation peer not found
- `500` — Internal server error

---

### GET /api/federation/{id}/jobs
**Purpose:** Get cached job list from federation peer (admin)

**Request:** No body

**Response:** 200 OK
```json
{
  "jobs": [ /* job objects */ ],
  "stale": "boolean",
  "as_of": "string (ISO 8601)"
}
```

**Errors:**
- `401` — Unauthenticated
- `403` — Not authorized (not admin)
- `404` — Federation peer not found
- `500` — Internal server error

---

### GET /api/federation/{id}/history
**Purpose:** Get cached job run history from federation peer (admin)

**Request:** No body

**Response:** 200 OK
```json
{
  "history": [ /* job run objects */ ],
  "stale": "boolean",
  "as_of": "string (ISO 8601)"
}
```

**Errors:**
- `401` — Unauthenticated
- `403` — Not authorized (not admin)
- `404` — Federation peer not found
- `500` — Internal server error

---

### GET /api/federation/sync
**Purpose:** Get federation event log since sequence number (admin, root coordinator)

**Query Params:**
- `since` — Sequence number to start from (required)
- `coordinator` — Coordinator ID to get events for (required)

**Request:** No body

**Response:** 200 OK
```json
{
  "events": [
    {
      "seq": "integer",
      "event_type": "string",
      "payload": "string (JSON)"
    }
  ],
  "latest_seq": "integer"
}
```

**Errors:**
- `400` — Missing or invalid query parameters
- `401` — Unauthenticated
- `403` — Not authorized (not admin)
- `500` — Internal server error

---

### POST /api/federation/sync/ack
**Purpose:** Acknowledge applied events up to sequence number (admin, spoke coordinator)

**Request:**
```json
{
  "seq": "integer (required, non-negative)"
}
```

**Query Params:**
- `coordinator` — Coordinator ID (required)

**Response:** 200 OK
```json
{
  "status": "ack",
  "seq": "integer"
}
```

**Errors:**
- `400` — Missing or invalid parameters
- `401` — Unauthenticated
- `403` — Not authorized (not admin)
- `500` — Internal server error

---

### GET /api/federation/health
**Purpose:** Get health status of all federation peers (viewer+)

**Request:** No body

**Response:** 200 OK
```json
[
  {
    "id": "string",
    "name": "string",
    "status": "string (online|offline|reconnecting)",
    "last_seen": "string (ISO 8601, nullable)",
    "lag_events": "integer",
    "agent_count": "integer",
    "last_seq": "integer",
    "max_seq": "integer",
    "version": "string (nullable)"
  }
]
```

**Errors:**
- `401` — Unauthenticated
- `500` — Internal server error

---

## Alert Rules & History API

### GET /api/alert-rules
**Purpose:** List all alert rules (viewer+)

**Request:** No body

**Response:** 200 OK
```json
[
  {
    "id": "integer",
    "job_id": "string (empty = applies to all jobs)",
    "rule_type": "string (on_failure|duration_exceeded|missed_schedule)",
    "threshold": "integer (seconds; 0 for on_failure)",
    "enabled": "boolean",
    "created_at": "string (ISO 8601)"
  }
]
```

**Errors:**
- `401` — Unauthenticated
- `500` — Internal server error

---

### POST /api/alert-rules
**Purpose:** Create a new alert rule (admin)

**Request:**
```json
{
  "job_id": "string (optional, empty = all jobs)",
  "rule_type": "string (required, on_failure|duration_exceeded|missed_schedule)",
  "threshold": "integer (required, seconds; 0 for on_failure)",
  "enabled": "boolean (optional, default: false)"
}
```

**Response:** 201 Created
```json
{
  "id": "integer",
  "job_id": "string",
  "rule_type": "string",
  "threshold": "integer",
  "enabled": "boolean",
  "created_at": "string (ISO 8601)"
}
```

**Errors:**
- `400` — Invalid JSON or invalid rule_type
- `401` — Unauthenticated
- `403` — Not authorized (not admin)
- `500` — Internal server error

**Validation Rules:**
- rule_type must be one of: on_failure, duration_exceeded, missed_schedule
- threshold: required, in seconds (0 for on_failure)

---

### PUT /api/alert-rules/{id}
**Purpose:** Update an alert rule (admin)

**Request:**
```json
{
  "job_id": "string (optional)",
  "rule_type": "string (optional)",
  "threshold": "integer (optional)",
  "enabled": "boolean (optional)"
}
```

**Response:** 200 OK
```json
{
  "id": "integer",
  "job_id": "string",
  "rule_type": "string",
  "threshold": "integer",
  "enabled": "boolean",
  "created_at": "string (ISO 8601)"
}
```

**Errors:**
- `400` — Invalid JSON or invalid id
- `401` — Unauthenticated
- `403` — Not authorized (not admin)
- `500` — Internal server error

---

### DELETE /api/alert-rules/{id}
**Purpose:** Delete an alert rule (admin)

**Request:** No body

**Response:** 204 No Content

**Errors:**
- `400` — Invalid id
- `401` — Unauthenticated
- `403` — Not authorized (not admin)
- `500` — Internal server error

---

### GET /api/alert-history
**Purpose:** List alert notification history (viewer+)

**Request:** No body

**Response:** 200 OK
```json
[
  {
    "id": "integer",
    "rule_id": "integer",
    "job_id": "string",
    "run_id": "string",
    "rule_type": "string",
    "fired_at": "string (ISO 8601)",
    "channel": "string",
    "status": "string",
    "attempts": "integer",
    "last_error": "string"
  }
]
```

**Errors:**
- `401` — Unauthenticated
- `500` — Internal server error

---

### POST /api/alert-history/{id}/retry
**Purpose:** Retry a failed alert notification (admin)

**Request:** No body

**Response:** 200 OK
```json
{
  "id": "integer",
  "status": "retry_requested"
}
```

**Errors:**
- `400` — Invalid id
- `401` — Unauthenticated
- `403` — Not authorized (not admin)
- `500` — Internal server error

---

## Update & Rollback API

### GET /api/update/check
**Purpose:** Check for coordinator updates (admin, cached for 5 minutes)

**Request:** No body

**Response:** 200 OK
```json
{
  "current": "string (current version)",
  "latest": "string (latest available version)",
  "update_available": "boolean",
  "release_url": "string",
  "asset_url": "string",
  "checksum_url": "string"
}
```

**Errors:**
- `401` — Unauthenticated
- `403` — Not authorized (not admin)
- `500` — Failed to check for updates

---

### POST /api/update/apply
**Purpose:** Apply coordinator update (admin)

**Request:** No body

**Response:** 202 Accepted
```json
{
  "status": "update started"
}
```

**Errors:**
- `401` — Unauthenticated
- `403` — Not authorized (not admin)
- `409` — Update already in progress
- `500` — Internal server error

**Note:** Progress is broadcast via WebSocket as `update_progress` events.

---

### POST /api/agents/{id}/update
**Purpose:** Trigger remote agent update (admin)

**Request:** No body

**Response:** 202 Accepted
```json
{
  "status": "update_initiated",
  "agent_id": "string"
}
```

**Errors:**
- `400` — Missing agent id or arch unknown
- `401` — Unauthenticated
- `403` — Not authorized (not admin)
- `404` — Agent not found
- `409` — Update already in progress for this agent
- `500` — Internal server error

---

### GET /api/rollback-available
**Purpose:** Check if coordinator rollback is available (admin)

**Request:** No body

**Response:** 200 OK
```json
{
  "available": "boolean"
}
```

**Errors:**
- `401` — Unauthenticated
- `403` — Not authorized (not admin)
- `500` — Internal server error

---

### POST /api/rollback
**Purpose:** Roll back coordinator to previous version (admin)

**Request:** No body

**Response:** 200 OK (streaming JSON events)
```json
{"type":"rollback_progress","step":"starting","pct":5,"message":"Starting rollback..."}
```

**Errors:**
- `401` — Unauthenticated
- `403` — Not authorized (not admin)
- `409` — No backup available for rollback
- `500` — Internal server error

---

### POST /api/agents/{id}/rollback
**Purpose:** Roll back agent to previous version (admin)

**Request:** No body

**Response:** 202 Accepted
```json
{
  "status": "rollback_initiated",
  "agent_id": "string"
}
```

**Errors:**
- `400` — Missing agent id
- `401` — Unauthenticated
- `403` — Not authorized (not admin)
- `404` — Agent not found
- `409` — No backup available for agent rollback
- `500` — Internal server error

---

## Audit API

### GET /api/audit/commands
**Purpose:** List command audit logs with filters (viewer+)

**Query Params:**
- `program` — Filter by program name (optional, exact match)
- `whitelisted` — Filter by whitelist status (optional, "true"|"false")
- `agent_id` — Filter by agent ID (optional)
- `from` — Start timestamp (optional, RFC3339)
- `to` — End timestamp (optional, RFC3339)
- `limit` — Results per page (default: 100, max: 10000)
- `offset` — Pagination offset (optional)

**Request:** No body

**Response:** 200 OK
```json
{
  "logs": [ /* audit log entries */ ],
  "total": "integer",
  "limit": "integer",
  "offset": "integer"
}
```

**Errors:**
- `401` — Unauthenticated
- `500` — Internal server error

---

### GET /api/audit/non-whitelisted-programs
**Purpose:** Get distinct non-whitelisted programs and execution counts (viewer+)

**Request:** No body

**Response:** 200 OK
```json
{
  "programs": [ /* program objects */ ],
  "count": "integer"
}
```

**Errors:**
- `401` — Unauthenticated
- `500` — Internal server error

---

### GET /api/audit/stats
**Purpose:** Get audit statistics for a time range (viewer+)

**Query Params:**
- `from` — Start timestamp (optional, RFC3339, default: 24h ago)
- `to` — End timestamp (optional, RFC3339, default: now)

**Request:** No body

**Response:** 200 OK
```json
{
  /* stats fields from DB */
}
```

**Errors:**
- `401` — Unauthenticated
- `500` — Internal server error

---

### GET /api/audit/user-actions
**Purpose:** List user action audit log entries (viewer+)

**Query Params:**
- `page` — Page number (default: 1, min: 1)
- `limit` — Items per page (default: 50, min: 1, max: 100)
- `action` — Filter by action type (optional)
- `user_id` — Filter by user ID (optional)
- `username` — Filter by username (optional)
- `resource_type` — Filter by resource type (optional)
- `resource_id` — Filter by resource ID (optional)
- `from_date` — Start timestamp (optional, RFC3339)
- `to_date` — End timestamp (optional, RFC3339)
- `success` — Filter by success status (optional, "true"|"false")

**Request:** No body

**Response:** 200 OK
```json
{
  "data": [
    {
      "id": "integer",
      "user_id": "integer (nullable)",
      "username": "string",
      "user_role": "string",
      "action": "string",
      "resource_type": "string (nullable)",
      "resource_id": "string (nullable)",
      "details": "string (nullable)",
      "ip_address": "string",
      "success": "boolean",
      "request_method": "string (nullable)",
      "request_path": "string (nullable)",
      "status_code": "integer (nullable)",
      "latency_ms": "integer (nullable)",
      "created_at": "string (ISO 8601)"
    }
  ],
  "total": "integer",
  "page": "integer",
  "pages": "integer",
  "limit": "integer"
}
```

**Errors:**
- `401` — Unauthenticated
- `500` — Internal server error

---

## Admin Utility API

### GET /api/admin/token
**Purpose:** Get the admin token for agent registration (admin)

**Request:** No body

**Response:** 200 OK
```json
{
  "token": "string"
}
```

**Errors:**
- `401` — Unauthenticated
- `403` — Not authorized (not admin)
- `500` — Internal server error

---

### GET /api/admin/bootstrap.ps1
**Purpose:** Generate and download a PowerShell bootstrap script for agent setup (admin)

**Query Params:**
- `hostname` — Per-machine token scope (optional)

**Request:** No body

**Response:** 200 OK
Content-Type: `text/plain` (PowerShell script)

**Errors:**
- `401` — Unauthenticated
- `403` — Not authorized (not admin)
- `500` — Internal server error

---

### GET /downloads/installer
**Purpose:** Download the ArcVault Setup installer (admin)

**Request:** No body

**Response:** 200 OK
Content-Type: `application/octet-stream`
Content-Disposition: `attachment; filename=ArcVault-Setup-{version}-windows-amd64.exe`

**Errors:**
- `401` — Unauthenticated
- `403` — Not authorized (not admin)
- `404` — Installer not found
- `500` — Internal server error

---

### GET /downloads/agent.exe
**Purpose:** Download the agent binary (agent token or admin)

**Request:** No body

**Response:** 200 OK
Content-Type: `application/octet-stream`
Content-Disposition: `attachment; filename=agent.exe`

**Errors:**
- `401` — Unauthenticated
- `404` — agent.exe not available
- `500` — Internal server error

---

## Notes for Implementation

1. **Error Messages:** Never reveal which field failed in login (use generic "Invalid username or password")
2. **Passwords:** Never return password hashes in any response
3. **Timestamps:** Always ISO 8601 format with timezone
4. **Pagination:** Always include pagination metadata when listing (`total`, `page`, `pages`, `limit`)
5. **IDs:** UUIDs for jobs/runs/agents, integers for users/groups, `cred-` prefix for credential profiles, `fed-` prefix for federation peers
6. **Status Codes:** Follow HTTP conventions (201 for created, 202 for accepted, 204 for deleted, 400 for invalid, 401 for auth, 403 for permission, 404 for not found, 409 for conflict, 500 for server error)

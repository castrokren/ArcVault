# ArcVault 2.0 API Functions Inventory

**Purpose**: Complete list of HTTP endpoints and their handlers. Prevents accidental deletion or misconfiguration during refactors.

## Endpoint Registry

Format: `METHOD /path → handler() — auth, role, description`

### Authentication & Tokens
- `POST /api/login` → `handleLogin()` — public, returns JWT
- `POST /api/refresh-token` → `handleRefreshToken()` — viewer+, refreshes JWT
- `POST /api/change-password` → `handleChangePassword()` — viewer+, user self-service
- `POST /api/agents/register` → `handleRegister()` — agent-token, agent startup registration

### Agents (coordinator → agent lifecycle)
- `GET /api/agents` → `handleListAgents()` — admin/viewer (paginated)
- `GET /api/agents/{id}` — N/A (use list)
- `DELETE /api/agents/{id}` → `handleDeleteAgent()` — admin
- `POST /api/agents/{id}/heartbeat` → `handleHeartbeat()` — agent-token
- `POST /api/agents/{id}/update` → `handleAgentUpdate()` — admin, sends `update_command` over WS
- `POST /api/agents/{id}/rollback` → `handleAgentRollback()` — admin, sends `rollback_command` over WS
- `POST /api/agents/{id}/token` → `handleCreateAgentToken()` — admin, generates per-agent token for new installs

### Coordinator Self-Update
- `GET /api/update/check` → `handleCheckUpdate()` — admin
- `POST /api/update/apply` → `handleApplyUpdate()` — admin, self-updates from GitHub release
- `GET /api/rollback-available` → `handleRollbackAvailable()` — admin
- `POST /api/rollback` → `handleRollback()` — admin, coordinator rollback

### Jobs
- `GET /api/jobs` → `handleListJobs()` — admin/operator/viewer (paginated)
- `POST /api/jobs` → `handleCreateJob()` — admin/operator, creates job template
- `PUT /api/jobs/{id}` → `handleUpdateJob()` — admin/operator
- `DELETE /api/jobs/{id}` → `handleDeleteJob()` — admin
- `POST /api/jobs/{id}/run` → `handleRunJob()` — admin/operator, triggers immediate run
- `POST /api/jobs/{id}/cancel` → `handleCancelJob()` — admin/operator, cancels in-progress run
- `GET /api/job-runs` → `handleListJobRuns()` — admin/operator/viewer (paginated), returns run history
- `GET /api/job-runs/{id}` — N/A (use list)

### Groups
- `GET /api/groups` → `handleListGroups()` — admin (paginated)
- `POST /api/groups` → `handleCreateGroup()` — admin
- `PUT /api/groups/{id}` → `handleUpdateGroup()` — admin
- `DELETE /api/groups/{id}` → `handleDeleteGroup()` — admin
- `POST /api/groups/{id}/agents` → `handleAddAgentToGroup()` — admin
- `DELETE /api/groups/{id}/agents/{agentId}` → `handleRemoveAgentFromGroup()` — admin

### Credentials
- `GET /api/credentials` → `handleListCredentials()` — admin (paginated)
- `POST /api/credentials` → `handleCreateCredential()` — admin, stores encrypted secrets
- `PUT /api/credentials/{id}` → `handleUpdateCredential()` — admin
- `DELETE /api/credentials/{id}` → `handleDeleteCredential()` — admin

### Templates
- `GET /api/templates` → `handleListTemplates()` — viewer (paginated)
- `POST /api/templates` → `handleCreateTemplate()` — admin/operator
- `PUT /api/templates/{id}` → `handleUpdateTemplate()` — admin/operator
- `DELETE /api/templates/{id}` → `handleDeleteTemplate()` — admin

### Users (Admin only)
- `GET /api/users` → `handleListUsers()` — admin (paginated)
- `POST /api/users` → `handleCreateUser()` — admin, creates local user
- `PUT /api/users/{id}/role` → `handleChangeUserRole()` — admin
- `DELETE /api/users/{id}` → `handleDeleteUser()` — admin

### Federation (Multi-coordinator failover)
- `GET /api/federation/status` → `handleFederationStatus()` — admin
- `POST /api/federation/peer` → `handleAddFederationPeer()` — admin
- `DELETE /api/federation/peer/{peerId}` → `handleRemoveFederationPeer()` — admin

### Status & Health
- `GET /health` → `handleHealth()` — public, no auth (load balancer health check)
- `GET /api/version` → `handleVersion()` — viewer+, returns coordinator version

### Downloads
- `GET /downloads/agent.exe` → `handleDownloadAgent()` — agent/admin, serves agent binary
- `GET /downloads/installer` → `handleDownloadInstaller()` — agent/admin, serves Windows installer .exe
- `GET /api/admin/bootstrap.ps1` → `handleBootstrapScript()` — admin, generates per-agent bootstrap script

### WebSocket
- `GET /ws/agent` → `handleAgentWS()` — agent-token, persistent connection for agent ← coordinator commands
- `GET /ws/dashboard` → `handleWS()` — viewer+, persistent connection for dashboard ← live events

## Middleware Chain

All routes protected by role-based middleware (except `/health`, `/ws/agent`, `POST /api/login`, `GET /ws/dashboard`):

- `adminRoute(handler)` — requires admin role JWT
- `operatorRoute(handler)` — requires admin or operator role JWT
- `viewerRoute(handler)` — requires admin, operator, or viewer role JWT
- `adminTokenViewerRoute(handler)` — requires admin token OR viewer JWT (for scripting)
- `agentOrAdminRoute(handler)` — requires agent token OR admin JWT (for downloads)

## Handler File Map

| File | Handlers |
|------|----------|
| `agents.go` | `handleListAgents`, `handleDeleteAgent`, `handleRegister`, `handleHeartbeat` |
| `agent_update.go` | `handleAgentUpdate` |
| `agent_token.go` | `handleCreateAgentToken` |
| `agent_rollback.go` | `handleAgentRollback` |
| `jobs.go` | `handleListJobs`, `handleCreateJob`, `handleUpdateJob`, `handleDeleteJob`, `handleRunJob`, `handleCancelJob`, `handleListJobRuns` |
| `groups.go` | `handleListGroups`, `handleCreateGroup`, `handleUpdateGroup`, `handleDeleteGroup`, `handleAddAgentToGroup`, `handleRemoveAgentFromGroup` |
| `credentials.go` | `handleListCredentials`, `handleCreateCredential`, `handleUpdateCredential`, `handleDeleteCredential` |
| `templates.go` | `handleListTemplates`, `handleCreateTemplate`, `handleUpdateTemplate`, `handleDeleteTemplate` |
| `users.go` | `handleListUsers`, `handleCreateUser`, `handleChangeUserRole`, `handleDeleteUser` |
| `auth.go` | `handleLogin`, `handleRefreshToken`, `handleChangePassword` |
| `federation.go` | `handleFederationStatus`, `handleAddFederationPeer`, `handleRemoveFederationPeer` |
| `downloads.go` | `handleDownloadAgent`, `handleDownloadInstaller`, `handleBootstrapScript` |
| `health.go` | `handleHealth` |
| `version.go` | `handleVersion` |
| `hub.go` | `handleAgentWS`, `handleWS` |
| `update.go` | `handleCheckUpdate`, `handleApplyUpdate` |
| `rollback.go` | `handleRollbackAvailable`, `handleRollback` |

## Route Registration

All routes registered in `server.go:registerRoutes()` (~line 340-430). Each route specifies:
- HTTP method + path
- Handler function
- Auth middleware wrapper

To add a route:
1. Write handler in appropriate file (e.g., `handleNewThing` in `new_thing.go`)
2. Register in `server.go:registerRoutes()`: `s.router.HandleFunc("METHOD /path", s.middleware(s.handleNewThing))`
3. Document in this file
4. Add to FEATURES.md if user-visible

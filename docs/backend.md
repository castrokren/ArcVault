# ArcVault Backend (Go)

How the Go backend is structured, and the **test-enforced** inventory of every HTTP route.
The `CONTRACT:routes` block below is checked by `internal/docs` (`TestBackendDoc_routesMatchRegistered`)
against the routes actually registered in `coordinator/server/server.go`. Add or remove a route
and this doc must change with it — the pre-commit hook blocks a drifting commit.

> Supersedes the old `docs/FUNCTIONS.md` (now a redirect). That file was hand-maintained and had
> already drifted (e.g. it listed `POST /api/login`; the real route is `POST /api/auth/login`).

## Two programs, one module

- **Coordinator** (`coordinator/`, `package main`) — the hub. HTTPS on 443 serving the REST API
  (`/api/*`), the embedded Vue dashboard (`static/dist`), and WebSocket hubs for agents and the
  dashboard. SQLite (WAL) storage.
- **Agent** (`agent/`, `package main`) — the spoke, a Windows service per backup host. Registers,
  heartbeats, runs robocopy/rsync jobs, self-updates, and fails over across a coordinator list.

Service lifecycle (install, `run-service`, error 1067, logs) lives in **[service.md](service.md)**.
The Vue dashboard lives in **[frontend.md](frontend.md)**.

## Coordinator layers

Requests flow through three layers — never skip one:

1. **`coordinator/server/`** — HTTP handlers + middleware. Routes registered in
   `server.go:registerRoutes()`. Role middleware wraps every handler (below).
2. **`coordinator/business/`** — services holding domain logic (e.g. `AgentService.RegisterAgent`).
   Handlers call business; business calls db.
3. **`coordinator/db/`** — SQLite query methods (`db/agents.go`, `db/db.go`, …). WAL mode with
   `_busy_timeout=5000`; no `MaxOpenConns(1)`.

Cross-cutting: JWT RBAC (viewer/operator/admin) + agent-token auth, cron scheduler with templates
and missed-schedule detection, alert-rules engine with webhook/email/Slack/Teams notifiers,
coordinator↔coordinator federation failover, user-action + command audit logs, and self-update
from GitHub releases (`coordinator/updater/`).

## Auth middleware

Every route below is wrapped by exactly one guard (except the public ones — `/health`,
`POST /api/auth/login`, and the agent/WS token routes):

- `adminRoute` — admin-role JWT (+ password-change completed).
- `operatorRoute` — admin or operator JWT.
- `viewerRoute` — any authenticated JWT.
- `adminTokenViewerRoute` — admin token (for local ops scripts) OR viewer+ JWT.
- `authMiddleware` — agent token OR admin token (agent register/heartbeat).
- `agentOrAdminRoute` / `agentOrOperatorRoute` / `agentOrViewerRoute` — agent token OR the named JWT role.

All four machine-token guards share `acceptMachineToken` (`server/server.go`). Add an
agent-facing route → route it through that helper rather than re-inlining the token checks.

## Agent credentials: two kinds, and the exchange between them

There are two token shapes in the `tokens` table, distinguished by `db.IsEnrollmentToken`:

| Kind | `agent_id` | Expiry | Purpose |
|---|---|---|---|
| Enrollment | `bootstrap` / `bootstrap:<hostname>` | **1 hour** | Authenticate one new machine's first registration |
| Per-agent | the real agent ID | never | The agent's ongoing credential |

**`POST /api/agents/register` exchanges the first for the second.** When the caller
authenticated with an enrollment token, the response gains a `token` field holding a fresh
per-agent token; the agent adopts it and writes it into `agent-config.yaml`
(`agent/config.TokenStore`). This exists because `bootstrap.ps1` writes the enrollment token
into that file as the agent's `auth_token` — without the exchange the agent stopped
authenticating one hour after the install script was generated.

Details that matter if you touch this:

- The field is **additive**. Agents older than 2026-07-25 ignore it and keep their own token.
- No exchange happens for a caller already holding a per-agent token, or for the admin token
  (ops scripts) — otherwise every agent restart would mint a new credential.
- The enrollment token is **not** deleted on exchange. If the response were lost in transit,
  deleting it would leave the machine with no working credential and no recovery but
  re-enrollment. Its 1-hour expiry bounds the exposure instead.
- Minting a per-agent token **supersedes** that agent's previous ones in the same transaction
  (`db.CreateAgentToken`). Agent tokens never expire and nothing prunes them, so without this
  every mint left a permanently valid credential behind — `rebuild-and-restart.ps1` mints on
  each deploy, which had accumulated 12 live tokens for one agent.
- Consumers must read the token through `TokenStore.Get()` at request time. Copying it at
  construction is what left the WS client and job runner on a dead enrollment token.

## Route inventory (test-enforced)

Format: `METHOD /path`. To change: edit the handler + `registerRoutes()`, then update this block
(a failing test prints the corrected block to paste).

<!-- CONTRACT:routes — auto-checked by internal/docs/doc_test.go; do not hand-drift -->
- `DELETE /api/agents/{id}`
- `DELETE /api/alert-rules/{id}`
- `DELETE /api/credential-profiles/{id}`
- `DELETE /api/federation/{id}`
- `DELETE /api/groups/{id}`
- `DELETE /api/groups/{id}/agents/{agentID}`
- `DELETE /api/jobs/{id}`
- `DELETE /api/templates/{id}`
- `DELETE /api/users/{id}`
- `GET /api/admin/bootstrap.ps1`
- `GET /api/agents`
- `GET /api/alert-history`
- `GET /api/alert-rules`
- `GET /api/audit/commands`
- `GET /api/audit/non-whitelisted-programs`
- `GET /api/audit/stats`
- `GET /api/audit/user-actions`
- `GET /api/auth/me`
- `GET /api/credential-profiles`
- `GET /api/federation`
- `GET /api/federation/health`
- `GET /api/federation/sync`
- `GET /api/federation/{id}`
- `GET /api/federation/{id}/agents`
- `GET /api/federation/{id}/history`
- `GET /api/federation/{id}/jobs`
- `GET /api/groups`
- `GET /api/groups/{id}`
- `GET /api/groups/{id}/agents`
- `GET /api/job-runs`
- `GET /api/jobs`
- `GET /api/jobs/{id}`
- `GET /api/jobs/{id}/logs`
- `GET /api/jobs/{id}/progress`
- `GET /api/jobs/{id}/runs`
- `GET /api/rollback-available`
- `GET /api/templates`
- `GET /api/templates/{id}`
- `GET /api/update/check`
- `GET /api/users`
- `GET /api/version`
- `GET /downloads/agent.exe`
- `GET /downloads/installer`
- `GET /health`
- `GET /ws`
- `GET /ws/agent`
- `GET /ws/federation`
- `PATCH /api/jobs/{id}/status`
- `POST /api/agents/register`
- `POST /api/agents/{id}/heartbeat`
- `POST /api/agents/{id}/rollback`
- `POST /api/agents/{id}/token`
- `POST /api/agents/{id}/update`
- `POST /api/alert-history/{id}/retry`
- `POST /api/alert-rules`
- `POST /api/auth/login`
- `POST /api/auth/logout`
- `POST /api/auth/refresh`
- `POST /api/credential-profiles`
- `POST /api/federation`
- `POST /api/federation/sync/ack`
- `POST /api/federation/{id}/sync`
- `POST /api/groups`
- `POST /api/groups/{id}/agents`
- `POST /api/jobs`
- `POST /api/jobs/{id}/cancel`
- `POST /api/jobs/{id}/progress`
- `POST /api/jobs/{id}/results`
- `POST /api/rollback`
- `POST /api/templates`
- `POST /api/templates/{id}/run`
- `POST /api/update/apply`
- `POST /api/users`
- `PUT /api/alert-rules/{id}`
- `PUT /api/auth/change-password`
- `PUT /api/federation/{id}`
- `PUT /api/groups/{id}`
- `PUT /api/templates/{id}`
- `PUT /api/users/{id}/role`
<!-- /CONTRACT:routes -->

## Agent-side features (not HTTP routes)

Prose only — these live in the agent binary and have no coordinator route to anchor a contract:

- **Register + heartbeat** (`agent/heartbeat/`) — POSTs os/arch/version on startup, then heartbeats
  every 30s. Registration retries in the background; heartbeat starts only after it succeeds.
- **Job execution** (`agent/runner/`) — pulls jobs, runs robocopy/rsync, captures output/exit code,
  honors `cancel_command` over WS.
- **Self-update / rollback** (`agent/updater/`) — downloads agent.exe from the release URL the
  coordinator supplies, verifies SHA256, swaps binary (agent.exe→agent.previous→agent.exe), restarts
  the service, and streams `update_progress` / `rollback_progress` over WS.

## Agent version / out-of-date detection

How the coordinator knows which agents are behind:

- **Source of truth for an agent's version** is the `version` column of the `agents` table. It is
  written only by registration (`POST /api/agents/register` → `db.RegisterAgent`, an upsert with
  `version=excluded.version`). Heartbeats carry **no** version and never touch it.
- **A version change always re-registers.** Self-update restarts the agent service, and every startup
  re-registers, so the new version reaches the coordinator on the next register — no heartbeat path is
  needed for version freshness.
- **The "latest available" baseline** is the newest published GitHub release, cached by the
  coordinator (`GetUpdateCache().Latest`, surfaced to admins via `GET /api/update/check`). This is
  also the version the coordinator installs when it push-updates an agent
  (`POST /api/agents/{id}/update` sends `update_command` with that version). The dashboard flags an
  agent as out-of-date when its reported version is **strictly older** (semver) than this baseline,
  falling back to the coordinator's own running version (`GET /api/version`) for non-admins who can't
  reach the admin-only update check.
- **Federated (remote) agents** are cached on the root coordinator per sub (`SubCache.Agents`). Their
  version arrives via the replicated `FedAgentRegistered` delta (and full snapshots on reconnect). A
  `FedAgentHeartbeat` deliberately carries no version and must never overwrite the cached one — doing
  so previously blanked remote agents' versions every 30s, making remote staleness undetectable
  (regression guard: `TestFedHub_ApplyDelta_AgentHeartbeat`).
- **Not reported at all = never connected.** A host whose agent fails TLS to the coordinator (e.g.
  `tls: bad certificate` — the agent's CA bundle doesn't trust the coordinator cert) never registers,
  so it has no row and no version. That is a connectivity/cert-distribution problem, not a version
  bug; fix it by getting the coordinator CA cert onto the agent (`CACertFile`).

## Adding a route

1. Write the handler in the appropriate `coordinator/server/*.go` file.
2. Register it in `server.go:registerRoutes()` with the right guard middleware.
3. Add the `METHOD /path` line to the `CONTRACT:routes` block above (or run the test — it prints it).
4. If user-visible, describe it in [frontend.md](frontend.md).

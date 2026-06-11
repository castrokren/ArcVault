# ArcVault v1.0.0 — IT Architecture Approval Document

**Document type:** Architecture Review  
**Version:** 1.0.0  
**Date:** 2026-06-11  
**Audience:** IT Architecture Team  

---

## 1. Overview

ArcVault is a self-hosted, cross-platform backup orchestrator. A single Go binary — the **coordinator** — serves the REST API, WebSocket hub, and Vue 3 dashboard from one process. Lightweight **agents** run as OS services on managed hosts (Windows, macOS, Linux), connect to the coordinator over persistent encrypted WebSocket connections, and execute backup jobs as directed. All data remains on-premises. There is no cloud dependency and no external call-home.

---

## 2. Component Diagram

```
┌─────────────────────────────────────────────────────┐
│  BROWSER / ADMIN LAYER                              │
│  Vue 3 SPA (served from coordinator binary)         │
│  HTTPS on :443                                     │
└───────────────────┬─────────────────────────────────┘
                    │ HTTPS / WSS
┌───────────────────▼─────────────────────────────────┐
│  COORDINATOR LAYER                                  │
│  ┌──────────────┐  ┌────────────┐  ┌─────────────┐ │
│  │  REST API    │  │  WS Hub    │  │  Scheduler  │ │
│  │ (Gorilla Mux)│  │(Gorilla WS)│  │(robfig/cron)│ │
│  └──────┬───────┘  └─────┬──────┘  └──────┬──────┘ │
│         └────────────────┼─────────────────┘        │
│                          │                          │
│  ┌───────────────────────▼─────────────────────┐   │
│  │  SQLite (WAL mode, embedded via modernc.org) │   │
│  └──────────────────────────────────────────────┘   │
└───────────────────┬─────────────────────────────────┘
                    │ WSS (per-agent token auth)
┌───────────────────▼─────────────────────────────────┐
│  AGENT LAYER                                        │
│  Agent A (Windows SCM)                              │
│  Agent B (macOS LaunchD)                            │
│  Agent C (Linux systemd)                            │
│  Each agent: outbound WSS only, no inbound port     │
└─────────────────────────────────────────────────────┘
```

Federation adds a secondary coordinator tier:

```
Root Coordinator  ──WSS──►  Spoke Coordinator A
                  ──WSS──►  Spoke Coordinator B
```

---

## 3. Coordinator Design

**Single binary rationale.** The coordinator is compiled to a single Go binary with no runtime dependencies. The Vue 3 dashboard is embedded at build time using Go's `//go:embed` directive, eliminating the need for a separate web server (nginx, Apache, etc.), a Node.js runtime in production, or a reverse proxy for serving static assets. Deployment is: copy binary, write config.json, start service.

**Embedded dashboard.** Dashboard assets (HTML, JS, CSS) are compiled into the binary via `//go:embed`. The coordinator serves them directly from memory. There is no separate static file directory to manage or secure.

**SQLite as embedded datastore.** SQLite was chosen for zero external dependency (no database server to install, configure, or maintain), strong reliability on single-host deployments, and native support for WAL (Write-Ahead Logging) mode, which allows concurrent reads while a write is in progress. `busy_timeout` is set to 5000 ms to handle brief lock contention. The pure-Go `modernc.org/sqlite` driver is used — no CGO, no C toolchain required.

**HTTP router.** Gorilla Mux provides pattern-based routing with middleware support. All routes require JWT authentication middleware except `/api/login` and `/health`.

**Scheduler.** `robfig/cron` (v3) drives scheduled job execution. Cron expressions are stored in the database; the scheduler is updated at runtime when jobs are created or modified — no restart required.

---

## 4. Agent Design

Agents are small Go binaries installed as native OS services:

| Platform | Service mechanism | Config path |
|---|---|---|
| Windows | Service Control Manager (SCM) | `C:\ArcVault-Agent\agent-config.yaml` |
| macOS | LaunchD (`launchd.plist`) | `/etc/arcvault-agent/agent-config.yaml` |
| Linux | systemd unit | `/etc/arcvault-agent/agent-config.yaml` |

Each agent:
- Reads `agent-config.yaml` at startup (coordinator URL, agent ID, token)
- Establishes an outbound WSS connection to the coordinator; does **not** open any inbound listening port
- Authenticates with its 64-char hex token on connect
- Executes job commands dispatched by the coordinator over the WebSocket
- Streams job output and status back to the coordinator in real time
- Implements reconnect-with-backoff if the connection drops

---

## 5. Authentication & Authorization

**JWT.** On successful login, the coordinator issues a signed JWT (algorithm: HS256). The JWT is required as a Bearer token on every API route. Expiry is configurable in `config.json`. Token validation and role extraction occur server-side in middleware before the handler runs.

**RBAC roles.**

| Role | Permissions |
|---|---|
| `admin` | Full access: manage users, agents, jobs, tokens, alert rules, federation config |
| `operator` | Run jobs, view job history, view agents and alert history |
| `viewer` | Read-only: view jobs, agents, runs, alerts — no write operations |

Role is embedded in the JWT claim and enforced server-side on every route. There is no client-side-only enforcement.

**Per-agent tokens.** Each agent is assigned a unique 64-character hex token at registration. Tokens are stored hashed in the `tokens` table (not in plaintext). Tokens are scoped to a single agent ID and can be revoked via the dashboard without restarting the coordinator. There are no shared tokens between agents.

---

## 6. Federation HA

**Topology.** One root coordinator manages one or more spoke coordinators. The root is the write-authoritative node. Spokes receive state updates and can serve read and execution requests from their local agents.

**State sync.** State is replicated via an append-only `federation_events` table using a monotonic sequence. Spoke coordinators consume events from the root. The event log is never truncated; new events are always appended.

**Failover.** The root coordinator exposes a `/health` endpoint. If the root becomes unreachable, spokes can continue local operations autonomously. Manual or automatic promotion of a spoke to root is supported.

**Health monitoring.** Root polls spoke health endpoints on a configurable interval. Alert rules fire on federation node failures the same way they fire on job failures.

---

## 7. Self-Update

The coordinator can deliver binary updates to itself and to connected agents over the existing WebSocket connection. Update flow:

1. Admin uploads or points the coordinator to a new binary package via the dashboard.
2. Coordinator distributes the package to agents over WSS with live progress reporting.
3. Agent or coordinator replaces its binary and restarts.
4. One previous version is retained; rollback can be triggered from the dashboard without re-uploading.

---

## 8. Data Architecture

**Schema summary.**

| Table | Purpose |
|---|---|
| `agents` | Registered agents: ID, hostname, platform, last-seen |
| `jobs` | Job definitions: name, command, schedule, target agent, enabled flag |
| `job_runs` | Execution history: job ID, start/end time, exit code, output, status |
| `tokens` | Per-agent tokens (hashed), agent binding, revocation status |
| `alert_rules` | Alert rule definitions: type, threshold, notification channel config |
| `alert_history` | Fired alerts with timestamps, rule ID, job run reference |
| `federation_events` | Append-only monotonic event log for state replication |

**WAL mode.** SQLite is opened with `PRAGMA journal_mode=WAL` and `PRAGMA busy_timeout=5000`. WAL allows multiple concurrent readers without blocking on an active write.

**Retention.** `alert_history` rows older than 30 days are pruned by the scheduler on a nightly run. `job_runs` are not auto-pruned (this is an operational decision).

**Migrations.** Schema changes are additive only. No column removal or rename is performed in migrations, ensuring safe rollback to a prior binary version.

---

## 9. Dependencies

| Package | Purpose | Notable property |
|---|---|---|
| `modernc.org/sqlite` | SQLite driver | Pure Go — no CGO, no C toolchain required |
| `github.com/gorilla/mux` | HTTP router | Stable, no generics requirement |
| `github.com/gorilla/websocket` | WebSocket (server + client) | RFC 6455 compliant |
| `github.com/robfig/cron/v3` | Cron scheduler | POSIX cron syntax, thread-safe |
| `github.com/golang-jwt/jwt/v5` | JWT issue and validation | HS256; v5 addresses prior security issues in v4 |
| `golang.org/x/crypto` | bcrypt password hashing | Standard Go extended library |

Build toolchain: Go 1.21+. No CGO. Dashboard build (Node 18+, Vite) is a compile-time step only — Node is not required in production.

---

## 10. Scalability Considerations

SQLite is a single-writer database. For the target use case — a backup orchestrator managing dozens of agents running sequential jobs — this is not a practical constraint. Write contention is low because job runs are short-burst writes separated by idle periods.

**Where SQLite becomes a constraint:**
- Hundreds of agents with very high-frequency job runs generating concurrent writes
- Multi-host coordinator clustering requiring a shared writable datastore

**Horizontal scale path.** Federation HA distributes agent load across spoke coordinators, each with its own SQLite instance. State synchronisation uses the event log. This is the supported scale-out mechanism for ArcVault.

**If a different datastore is required:** The data access layer is isolated in the `db` package. Migration to PostgreSQL or another driver would be scoped to that package.

---

## 11. Integration Points

| Integration | Direction | Protocol | Auth |
|---|---|---|---|
| REST API | Inbound (browser, tooling) | HTTPS | JWT Bearer |
| WebSocket (agent protocol) | Outbound from agent | WSS | Per-agent hex token |
| Federation WebSocket | Outbound from spoke | WSS | Coordinator-level shared secret |
| Notification webhooks | Outbound from coordinator | HTTPS POST | HMAC-SHA256 signature on payload |
| SMTP alerts | Outbound from coordinator | SMTP/TLS | SMTP credentials in config.json |
| Slack alerts | Outbound from coordinator | HTTPS POST | Incoming webhook URL |
| Teams alerts | Outbound from coordinator | HTTPS POST | Adaptive Card webhook URL |

All integrations are outbound from the coordinator or agent. No external system needs to make inbound connections to an agent port. The coordinator exposes exactly one port (:443 HTTPS).

---

## Appendix Reference

See `APPENDIX.md` in this directory for:
- Full port matrix
- Complete REST API endpoint list
- `config.json` schema reference

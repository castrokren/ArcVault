# ArcVault

**A self-hosted, cross-platform backup orchestrator with a real-time web dashboard.**

ArcVault coordinates backup jobs across multiple machines from a single pane of glass. A lightweight agent runs on each machine and reports to a central coordinator, which provides live status, alerting, scheduling, and a full audit trail — all through an embedded Vue.js dashboard served from a single Go binary.

---

## Why ArcVault?

Most backup tools are either cloud-dependent, Windows-only, or invisible once they're set up. ArcVault is built for teams that want:

- **Full visibility** — real-time job status, history charts, and alert feeds across every agent
- **Self-hosted** — no SaaS, no cloud dependency, runs on your own infrastructure
- **Cross-platform** — coordinator and agents run on Windows, macOS, and Linux
- **Single binary** — the coordinator ships with the dashboard embedded; no separate web server needed

---

## Features

| Category | Capability |
|---|---|
| **Agents** | Lightweight Go agents on Windows/macOS/Linux; register with a per-agent token |
| **Dashboard** | Embedded Vue.js UI served from the coordinator binary on `:8080` |
| **Job scheduling** | Cron-based backup templates; missed-schedule detection |
| **Monitoring** | Real-time job status, timeline visualization, per-agent history charts |
| **Alerting** | Configurable alert rules (on_failure, duration_exceeded, missed_schedule); 30-day alert history |
| **Notifications** | Webhook (HMAC-signed), Email (SMTP), Slack, and Microsoft Teams |
| **RBAC** | JWT authentication; three roles: `admin`, `operator`, `viewer` |
| **User management** | Create/edit/delete users and agent groups via the dashboard |
| **Federation HA** | Multi-coordinator failover with state sync and health monitoring |
| **Self-update** | Coordinator and agents self-update via WebSocket with live progress and one-version rollback |
| **Pagination** | Server-side pagination and filtering on all list endpoints |
| **Installers** | Native installers for Windows (NSIS), macOS (pkg), and Linux (deb/rpm) |

---

## Architecture

```
┌─────────────────────────────────────┐
│         Coordinator (Go)            │
│  ┌──────────────┐  ┌─────────────┐  │
│  │  REST API    │  │  Vue Dashboard│ │
│  │  WebSocket   │  │  (embedded) │  │
│  └──────────────┘  └─────────────┘  │
│         │ SQLite DB                 │
└─────────┼───────────────────────────┘
          │ WebSocket + Token Auth
   ┌──────┴──────┐
   │             │
Agent-1       Agent-N
(Win/Mac/Linux)
```

Each agent connects to the coordinator over WebSocket, authenticates with a per-agent token, and streams job results in real time. The coordinator stores everything in a local SQLite database and serves the Vue dashboard from the same binary.

---

## Quick Start

### 1. Download

Grab the latest release from [GitHub Releases](https://github.com/castrokren/ArcVault/releases).

Or build from source:
```bash
git clone https://github.com/castrokren/ArcVault.git
cd ArcVault

# Build coordinator
go build -o coordinator ./coordinator

# Build agent
go build -o agent ./agent
```

### 2. Start the coordinator

```bash
coordinator start
# Dashboard available at http://localhost:8080
```

### 3. Register an agent

On the coordinator machine, generate a token for each agent:
```bash
coordinator create-agent-token agent-01
# Copy the token output
```

On each agent machine, create `agent-config.yaml`:
```yaml
coordinator_url: ws://your-coordinator:8080
auth_token: <token from above>
agent_id: agent-01
```

Then start the agent:
```bash
agent start
```

### 4. Install as a system service (optional)

```bash
# Windows (run as Administrator)
coordinator install-service
agent install-service

# Linux / macOS (run as root)
sudo coordinator install-service
sudo agent install-service
```

---

## Configuration

### Notifications (`coordinator/config.json`)

```json
{
  "notifications": {
    "on_failure": true,
    "webhook": {
      "url": "https://hooks.example.com/arcvault",
      "secret": "your-hmac-secret"
    },
    "email": {
      "smtp_host": "smtp.example.com",
      "smtp_port": 587,
      "from": "arcvault@example.com",
      "to": ["ops@example.com"],
      "username": "user",
      "password": "pass"
    },
    "slack_webhook_url": "https://hooks.slack.com/services/...",
    "teams_webhook_url": "https://your-org.webhook.office.com/..."
  }
}
```

Webhook requests are HMAC-SHA256 signed (`X-ArcVault-Signature: sha256=<hex>`) and retried up to 3 times with exponential backoff.

---

## Development

```bash
# Run all tests
go test ./...

# Build dashboard (requires Node.js)
cd dashboard
npm install
npm run build

# The built dashboard is embedded into the coordinator binary at compile time
# via coordinator/static/
```

**Go:** 1.21+  
**Node:** 18+ (dashboard only)  
**Database:** SQLite (embedded, no setup required)

---

## Releases

| Version | Highlights |
|---|---|
| v1.0.0 | Enhanced alerting, Slack/Teams, webhook retry, alert history |
| v0.9.0 | Federation HA, multi-coordinator failover, health monitoring |
| v0.8.0 | RBAC, JWT authentication, user/group management |
| v0.5.0 | Failure notifications (webhook + email) |
| v0.4.0 | Job history visualization |
| v0.3.0 | Bidirectional rollback |
| v0.1.0 | Initial release — single binary, embedded dashboard, agent registration |

---

## License

MIT

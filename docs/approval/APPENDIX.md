# ArcVault — Technical Appendix

**Version:** 1.0.0  
**Purpose:** Shared technical reference for all ArcVault audience documents

---

## Port Matrix

| Port | Protocol | Direction | Purpose | Required | Notes |
|---|---|---|---|---|---|
| 443 | TCP (HTTPS + WSS) | Inbound | Dashboard UI, REST API, WebSocket (agent comms, self-update) | Required | All browser, agent, and federation traffic uses this port |
| 587 | TCP | Outbound | SMTP email notifications | Optional | STARTTLS; configure in `config.json` under `notifications.email` |
| 443 | TCP | Outbound | Slack incoming webhook | Optional | Only required if Slack notifications are enabled |
| 443 | TCP | Outbound | Microsoft Teams Adaptive Card webhook | Optional | Only required if Teams notifications are enabled |
| 443 | TCP | Outbound | Federation spoke → root coordinator | Optional | Spoke connects to root on the same port as the inbound API; only required in federation deployments |

Direction is from the coordinator's perspective unless otherwise noted.

---

## RBAC Permission Table

| Action | Admin | Operator | Viewer |
|---|---|---|---|
| View jobs | Yes | Yes | Yes |
| Run jobs manually | Yes | Yes | No |
| View agents | Yes | Yes | Yes |
| Manage agents (add/remove/edit) | Yes | Yes | No |
| Manage users (create/delete/edit roles) | Yes | No | No |
| Create alert rules | Yes | No | No |
| Delete alert rules | Yes | No | No |
| View alert history | Yes | Yes | Yes |
| Retry failed alerts | Yes | No | No |
| Manage federation (add/remove spokes) | Yes | No | No |
| Perform self-update (coordinator/agents) | Yes | No | No |

---

## config.json Schema

Full schema with all top-level fields. Restart the coordinator after any change.

```jsonc
{
  // Unique identifier for this coordinator instance; used in federation event log
  "coordinator_id": "prod-coordinator-01",

  // Secret used to sign and verify JWT tokens; use a long random string
  "jwt_secret": "change-me-use-64-char-random-string",

  // Path to TLS certificate (PEM format)
  "tls_cert_path": "C:\\ArcVault\\tls\\cert.pem",

  // Path to TLS private key (PEM format)
  "tls_key_path": "C:\\ArcVault\\tls\\key.pem",

  // Number of days to retain alert history in the database (default: 30)
  "alert_history_retention_days": 30,

  "notifications": {
    // Send alert notifications on job failure
    "on_failure": true,

    "webhook": {
      // Destination URL for custom webhook notifications
      "url": "https://ops.example.com/hooks/arcvault",

      // Secret used to compute HMAC-SHA256 signature in X-ArcVault-Signature header
      "secret": "webhook-signing-secret"
    },

    "email": {
      // SMTP server hostname
      "smtp_host": "smtp.example.com",

      // SMTP port; 587 for STARTTLS
      "smtp_port": 587,

      // Sender address
      "from": "arcvault@example.com",

      // Recipient address(es); single address or comma-separated list
      "to": "ops@example.com",

      // SMTP authentication username
      "username": "arcvault@example.com",

      // SMTP authentication password
      "password": "smtp-password"
    },

    "slack": {
      // Slack incoming webhook URL
      "webhook_url": "https://hooks.slack.com/services/T.../B.../..."
    },

    "teams": {
      // Microsoft Teams Adaptive Card webhook URL
      "webhook_url": "https://outlook.office.com/webhook/..."
    }
  }
}
```

---

## agent-config.yaml Schema

Stored at the agent install path. Restart the agent service after any change.

```yaml
# HTTPS base URL of the coordinator (include port)
coordinator_url: "https://coordinator.example.com"

# 64-character hex authentication token generated on the coordinator
auth_token: "a3f1c2d4...c9d2"

# Unique agent identifier; must match the ID used when generating the token
agent_id: "agent-hostname-01"

# Optional: list of coordinator URLs for failover (federation HA deployments)
# The agent tries each URL in order if the primary is unreachable
coordinators:
  - "https://coordinator-primary.example.com"
  - "https://coordinator-spoke.example.com"
```

---

## REST API Endpoints

All authenticated endpoints require a `Bearer` JWT token in the `Authorization` header obtained from `POST /api/login`.

| Method | Path | Auth Required | Role Required | Description |
|---|---|---|---|---|
| GET | `/health` | No | — | Health check; returns 200 OK when coordinator is up |
| POST | `/api/login` | No | — | Authenticate; returns JWT token |
| GET | `/api/jobs` | Yes | Viewer+ | List all configured jobs |
| POST | `/api/jobs` | Yes | Operator+ | Create a new job |
| PUT | `/api/jobs/{id}` | Yes | Operator+ | Update an existing job |
| DELETE | `/api/jobs/{id}` | Yes | Admin | Delete a job |
| POST | `/api/jobs/{id}/run` | Yes | Operator+ | Manually trigger a job run |
| GET | `/api/agents` | Yes | Viewer+ | List all registered agents and status |
| POST | `/api/agents` | Yes | Operator+ | Register a new agent |
| DELETE | `/api/agents/{id}` | Yes | Operator+ | Remove an agent |
| GET | `/api/alert-rules` | Yes | Viewer+ | List all alert rules |
| POST | `/api/alert-rules` | Yes | Admin | Create an alert rule |
| PUT | `/api/alert-rules/{id}` | Yes | Admin | Update an alert rule |
| DELETE | `/api/alert-rules/{id}` | Yes | Admin | Delete an alert rule |
| GET | `/api/alert-history` | Yes | Viewer+ | List alert history (paginated) |
| POST | `/api/alert-history/{id}/retry` | Yes | Admin | Retry a failed alert notification |
| GET | `/api/users` | Yes | Admin | List all users |
| POST | `/api/users` | Yes | Admin | Create a user |
| PUT | `/api/users/{id}` | Yes | Admin | Update a user (role, password) |
| DELETE | `/api/users/{id}` | Yes | Admin | Delete a user |
| GET | `/api/federation` | Yes | Admin | List federation spokes |
| POST | `/api/federation` | Yes | Admin | Register a federation spoke |
| DELETE | `/api/federation/{id}` | Yes | Admin | Remove a federation spoke |

---

## Database Schema Summary

ArcVault uses SQLite in WAL mode (`busy_timeout=5000ms`). The database file is at the coordinator install path (e.g., `C:\ArcVault\arcvault.db`).

| Table | Purpose | Key Columns | Retention |
|---|---|---|---|
| `agents` | Registered agents and their current status | `id`, `name`, `status`, `last_seen`, `coordinator_url` | Permanent (until agent deleted) |
| `jobs` | Job definitions (schedule, source, destination, options) | `id`, `name`, `schedule`, `source`, `destination`, `enabled` | Permanent (until job deleted) |
| `job_runs` | Historical record of individual job executions | `id`, `job_id`, `agent_id`, `status`, `started_at`, `completed_at`, `error` | No automatic pruning; managed by operator |
| `tokens` | Agent authentication tokens | `id`, `agent_id`, `token_hash`, `created_at` | Wiped on coordinator reinstall |
| `alert_rules` | Alert rule definitions | `id`, `job_id`, `rule_type`, `threshold_seconds`, `channel`, `enabled` | Permanent (until rule deleted) |
| `alert_history` | Record of alert deliveries and their status | `id`, `rule_id`, `job_run_id`, `channel`, `status`, `attempted_at`, `response` | `alert_history_retention_days` (default 30 days) |
| `federation_events` | Event log used for state sync between root and spoke coordinators | `id`, `event_type`, `payload`, `created_at`, `synced` | No automatic pruning |
| `users` | Dashboard user accounts and roles | `id`, `email`, `password_hash`, `role`, `created_at` | Permanent (until user deleted) |

---

## System Requirements

| Component | Supported OS | CPU | RAM | Disk | Runtime |
|---|---|---|---|---|---|
| Coordinator | Windows 10+, macOS 12+, Ubuntu 20.04+, RHEL 8+ | 2 cores min | 512 MB min; 1 GB recommended | 1 GB+ (SQLite DB) | None (statically linked binary) |
| Agent | Windows 10+, macOS 12+, Ubuntu 20.04+, RHEL 8+ | 1 core | < 50 MB | 100 MB | None (statically linked binary) |
| Source build | Same as above | — | — | — | Go 1.21+ |
| Dashboard build (from source) | Build machine only | — | — | — | Node.js 18+ |

The dashboard is compiled into the coordinator binary at build time. End users do not require Node.js.

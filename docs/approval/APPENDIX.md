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
| Manage agents (delete) | Yes | No | No |
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
  // Port the coordinator listens on (default: 443 for HTTPS, or 80/8080 for HTTP)
  "port": 443,

  // Path to the SQLite database file
  "database_path": "C:\\ArcVault\\arcvault.db",

  // Static admin bearer token for CLI operations and initial setup — treat as high-privilege
  "admin_token": "change-me-64-char-hex",

  // Secret used to sign and verify JWT tokens; use a long random string
  "jwt_secret": "change-me-use-64-char-random-string",

  // Unique identifier for this coordinator instance; used in federation event log
  "coordinator_id": "prod-coordinator-01",

  // Path to TLS certificate (PEM format); omit to run plain HTTP (not recommended for production)
  "cert_file": "C:\\ArcVault\\tls\\cert.pem",

  // Path to TLS private key (PEM format)
  "key_file": "C:\\ArcVault\\tls\\key.pem",

  // Set to true if TLS is terminated by a reverse proxy; coordinator will serve plain HTTP
  "external_tls": false,

  // Hostname or IP the coordinator binds to; omit to bind to all interfaces
  "host": "192.168.1.100",

  // CORS: list of allowed origins for dashboard access; use "*" for dev only
  "allowed_origins": ["https://arcvault.example.com"],

  // Number of days to retain alert history in the database (default: 30)
  "alert_history_retention_days": 30,

  // Environment tag; set to "production" to enable stricter startup warnings
  "environment": "production",

  // Path to the directory containing the installer binary for /downloads/installer
  "installer_dir": "C:\\ArcVault\\installer",

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

      // Recipient address(es)
      "to": ["ops@example.com"],

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
  },

  "federation": {
    // URL of the root coordinator; set this to make the coordinator a spoke
    "root_url": "https://coordinator-root.example.com",

    // Shared token for federation authentication between root and spoke
    "token": "federation-shared-secret"
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

# Optional: path to a CA certificate file (PEM) for validating the coordinator's TLS cert
# Required when coordinator uses a self-signed or internal CA certificate
ca_cert_file: "C:\\ArcVault-Agent\\ca.pem"
```

---

## REST API Endpoints

All authenticated endpoints require a `Bearer` JWT token in the `Authorization` header obtained from `POST /api/auth/login`. Agent-facing endpoints also accept a per-agent bearer token.

| Method | Path | Auth Required | Role Required | Description |
|---|---|---|---|---|
| GET | `/health` | No | — | Health check; returns `{"status":"ok"}` |
| POST | `/api/auth/login` | No | — | Authenticate; returns JWT token and role |
| POST | `/api/auth/logout` | Yes | Any | Revoke the current JWT |
| GET | `/api/auth/me` | Yes | Any | Return current user info from JWT claims |
| POST | `/api/auth/refresh` | Yes | Any | Issue a new JWT without re-entering credentials |
| PUT | `/api/auth/change-password` | Yes | Any | Change the authenticated user's password |
| GET | `/api/templates` | Yes | Viewer+ | List all backup job templates |
| POST | `/api/templates` | Yes | Admin | Create a backup job template |
| GET | `/api/templates/{id}` | Yes | Viewer+ | Get a single template |
| PUT | `/api/templates/{id}` | Yes | Admin | Update a template |
| DELETE | `/api/templates/{id}` | Yes | Admin | Delete a template |
| POST | `/api/templates/{id}/run` | Yes | Operator+ | Manually trigger a template run |
| GET | `/api/jobs` | Yes | Viewer+ | List job run instances |
| GET | `/api/jobs/{id}` | Yes | Viewer+ | Get a single job run |
| GET | `/api/jobs/{id}/runs` | Yes | Viewer+ | List all runs for a job |
| GET | `/api/job-runs` | Yes | Viewer+ | List all job runs across all jobs |
| POST | `/api/jobs/{id}/cancel` | Yes | Operator+ | Cancel a running job |
| GET | `/api/agents` | Yes | Viewer+ | List all registered agents and status |
| POST | `/api/agents/register` | Yes | Agent token | Register an agent with the coordinator |
| DELETE | `/api/agents/{id}` | Yes | Admin | Remove an agent |
| GET | `/api/alert-rules` | Yes | Viewer+ | List all alert rules |
| POST | `/api/alert-rules` | Yes | Admin | Create an alert rule |
| PUT | `/api/alert-rules/{id}` | Yes | Admin | Update an alert rule |
| DELETE | `/api/alert-rules/{id}` | Yes | Admin | Delete an alert rule |
| GET | `/api/alert-history` | Yes | Viewer+ | List alert history |
| POST | `/api/alert-history/{id}/retry` | Yes | Admin | Retry a failed alert notification |
| GET | `/api/users` | Yes | Admin | List all users |
| POST | `/api/users` | Yes | Admin | Create a user |
| PUT | `/api/users/{id}/role` | Yes | Admin | Update a user's role |
| DELETE | `/api/users/{id}` | Yes | Admin | Delete a user |
| GET | `/api/federation` | Yes | Admin | List federation spokes |
| POST | `/api/federation` | Yes | Admin | Register a federation spoke |
| DELETE | `/api/federation/{id}` | Yes | Admin | Remove a federation spoke |
| GET | `/api/federation/health` | Yes | Viewer+ | Federation health status |
| GET | `/api/update/check` | Yes | Admin | Check for available coordinator update |
| POST | `/api/update/apply` | Yes | Admin | Apply a coordinator update |
| POST | `/api/agents/{id}/update` | Yes | Admin | Push update to a specific agent |
| GET | `/api/rollback-available` | Yes | Admin | Check if a rollback version is available |
| POST | `/api/rollback` | Yes | Admin | Roll back coordinator to prior version |

---

## Database Schema Summary

ArcVault uses SQLite in WAL mode (`busy_timeout=5000ms`). The database file is at the coordinator install path (e.g., `C:\ArcVault\arcvault.db`).

| Table | Purpose | Key Columns | Retention |
|---|---|---|---|
| `agents` | Registered agents and their current status | `id`, `name`, `status`, `last_seen` | Permanent (until agent deleted) |
| `backup_templates` | Backup job definitions (schedule, command, target agent) | `id`, `name`, `agent_id`, `command`, `schedule`, `enabled` | Permanent (until template deleted) |
| `job_runs` | Historical record of individual job executions | `id`, `job_id`, `agent_id`, `status`, `started_at`, `finished_at`, `exit_code`, `output` | No automatic pruning; managed by operator |
| `tokens` | Agent authentication tokens (stored plaintext) | `id`, `agent_id`, `token`, `role`, `expires_at` | Wiped on coordinator reinstall |
| `revoked_tokens` | Revoked user JWTs (by JTI, until expiry) | `jti`, `expires_at` | Auto-pruned when expired |
| `alert_rules` | Alert rule definitions | `id`, `job_id`, `rule_type`, `threshold`, `enabled` | Permanent (until rule deleted) |
| `alert_history` | Record of alert deliveries and their status | `id`, `rule_id`, `job_id`, `run_id`, `channel`, `status`, `attempts`, `last_error` | `alert_history_retention_days` (default 30 days) |
| `federation_events` | Event log used for state sync between root and spoke coordinators | `id`, `event_type`, `payload`, `created_at` | No automatic pruning |
| `users` | Dashboard user accounts and roles | `id`, `username`, `password_hash`, `role`, `must_change_password` | Permanent (until user deleted) |

---

## System Requirements

| Component | Supported OS | CPU | RAM | Disk | Runtime |
|---|---|---|---|---|---|
| Coordinator | Windows 10+, macOS 12+, Ubuntu 20.04+, RHEL 8+ | 2 cores min | 512 MB min; 1 GB recommended | 1 GB+ (SQLite DB) | None (statically linked binary) |
| Agent | Windows 10+, macOS 12+, Ubuntu 20.04+, RHEL 8+ | 1 core | < 50 MB | 100 MB | None (statically linked binary) |
| Source build | Same as above | — | — | — | Go 1.21+ |
| Dashboard build (from source) | Build machine only | — | — | — | Node.js 18+ |

The dashboard is compiled into the coordinator binary at build time. End users do not require Node.js.

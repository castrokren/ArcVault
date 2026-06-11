# ArcVault — Cloud Operations Guide

**Version:** 1.0.0  
**Audience:** Cloud Operations  
**Purpose:** Deploy, operate, monitor, and maintain ArcVault in production

---

## Overview

ArcVault is a self-hosted, cross-platform backup orchestrator consisting of a central coordinator service and one or more lightweight agent services installed on managed hosts. The coordinator exposes an HTTPS API and Vue 3 dashboard, stores state in a local SQLite database, and delivers alerts via webhook, email, Slack, or Teams. Agents register with the coordinator using a pre-shared token, receive job instructions, and report results over WebSocket. No external database, web server, or cloud dependency is required. The entire stack ships as a single binary per component. See APPENDIX.md for port matrix, full API reference, and config schema.

---

## System Requirements

| Component | OS | CPU | RAM | Disk | Network |
|---|---|---|---|---|---|
| Coordinator | Windows 10+, macOS 12+, Ubuntu 20.04+, RHEL 8+ | 2 cores | 512 MB min, 1 GB recommended | 1 GB+ (SQLite DB growth) | Inbound 443/TCP; outbound 587/TCP (SMTP), 443/TCP (Slack/Teams) |
| Agent | Same OS support as coordinator | 1 core | 50 MB | 100 MB | Outbound to coordinator:443/TCP |

Go 1.21+ required for source builds. Pre-built installers do not require Go on the target system.

---

## Installation

### Coordinator

Admin or root privileges are required for all install steps.

**Using the installer (recommended):**

1. Download the appropriate package for the target OS:
   - Windows: `arcvault-coordinator-setup.exe` (NSIS)
   - macOS: `arcvault-coordinator.pkg`
   - Linux (Debian/Ubuntu): `arcvault-coordinator.deb`
   - Linux (RHEL/Fedora): `arcvault-coordinator.rpm`
2. Run the installer as Administrator (Windows) or with `sudo` (Linux/macOS).
3. The installer places files at `C:\ArcVault` (Windows) or the equivalent platform path.
4. The service is registered automatically and started at the end of installation.
5. Navigate to `https://<coordinator-host>:443` to verify the dashboard loads.
6. Complete initial setup (create admin account) on first login.

**Post-install:**

- Edit `C:\ArcVault\config.json` to configure TLS certificates, SMTP, webhook URLs, and JWT secret.
- Restrict config file permissions: only the service account should have read access.
- Verify the health endpoint responds: `curl -k https://localhost/health` should return `200 OK`.

### Agent

1. Download the agent installer for the target OS.
2. Run as Administrator / `sudo`.
3. Agent files are installed to `C:\ArcVault-Agent` (Windows) or platform equivalent.
4. Before starting the agent, edit `agent-config.yaml` with the coordinator URL and auth token (see Agent Token Management).
5. Start the agent service (see Service Management).
6. Confirm the agent appears as online in the dashboard under **Agents**.

---

## Configuration

### coordinator — config.json

Stored at the coordinator install path. Restart the coordinator service after any change.

| Field | Purpose | Example |
|---|---|---|
| `coordinator_id` | Unique identifier for this coordinator (used in federation) | `"prod-coordinator-01"` |
| `jwt_secret` | Secret used to sign JWT tokens; keep long and random | `"change-me-64-char-random-string"` |
| `tls_cert_path` | Path to TLS certificate file (PEM) | `"C:\\ArcVault\\tls\\cert.pem"` |
| `tls_key_path` | Path to TLS private key file (PEM) | `"C:\\ArcVault\\tls\\key.pem"` |
| `alert_history_retention_days` | How many days alert history is retained in DB | `30` |
| `notifications.email.smtp_host` | SMTP server hostname | `"smtp.example.com"` |
| `notifications.email.smtp_port` | SMTP port (TLS STARTTLS) | `587` |
| `notifications.email.from` | Sender address | `"arcvault@example.com"` |
| `notifications.email.to` | Recipient address(es) | `"ops@example.com"` |
| `notifications.email.username` | SMTP auth username | `"arcvault@example.com"` |
| `notifications.email.password` | SMTP auth password | `"smtp-password"` |
| `notifications.slack.webhook_url` | Slack incoming webhook URL | `"https://hooks.slack.com/..."` |
| `notifications.teams.webhook_url` | Teams Adaptive Card webhook URL | `"https://outlook.office.com/webhook/..."` |
| `notifications.webhook.url` | Custom webhook endpoint | `"https://ops.example.com/hooks/arcvault"` |
| `notifications.webhook.secret` | HMAC-SHA256 signing secret for webhook requests | `"webhook-secret"` |

**Security note:** Restrict `config.json` to read-only for the service account. On Windows, remove all user ACEs and grant only the service account and Administrators.

### agent — agent-config.yaml

Stored at the agent install path.

| Field | Purpose | Example |
|---|---|---|
| `coordinator_url` | HTTPS base URL of the coordinator | `"https://coordinator.example.com"` |
| `auth_token` | 64-char hex token generated on the coordinator | `"a3f1...c9d2"` |
| `agent_id` | Unique identifier for this agent; must match token registration | `"agent-hostname-01"` |

See APPENDIX.md for full schema including optional federation failover fields.

---

## Service Management

### Windows

```powershell
# Coordinator
sc.exe start ArcVaultCoordinator
sc.exe stop ArcVaultCoordinator
sc.exe query ArcVaultCoordinator

# Agent
sc.exe start ArcVaultAgent
sc.exe stop ArcVaultAgent
sc.exe query ArcVaultAgent
```

Services can also be managed via **Services MMC** (`services.msc`). Both services are configured for automatic startup.

### Linux (systemd)

```bash
# Coordinator
sudo systemctl start arcvault-coordinator
sudo systemctl stop arcvault-coordinator
sudo systemctl status arcvault-coordinator

# Agent
sudo systemctl start arcvault-agent
sudo systemctl stop arcvault-agent
sudo systemctl status arcvault-agent
```

### macOS (LaunchD)

```bash
# Coordinator
sudo launchctl start com.arcvault.coordinator
sudo launchctl stop com.arcvault.coordinator

# Agent
sudo launchctl start com.arcvault.agent
sudo launchctl stop com.arcvault.agent
```

---

## Health Monitoring

**Health endpoint:**

```
GET https://<coordinator-host>:443/health
```

- No authentication required.
- Returns `200 OK` when the coordinator is up and accepting requests.
- Returns a non-200 status or connection refused when the service is down.

**Dashboard:**

```
https://<coordinator-host>:443
```

Accessible via any modern browser. Requires login (JWT-based).

**Recommended monitoring integration:**

Configure your existing uptime monitoring tool (Datadog, Prometheus blackbox exporter, UptimeRobot, etc.) to poll `GET /health` every 60 seconds. Alert on two consecutive non-200 responses to avoid false positives from brief restarts.

Example Prometheus blackbox target:

```yaml
- job_name: arcvault_health
  metrics_path: /probe
  params:
    module: [http_2xx]
  static_configs:
    - targets:
        - https://coordinator.example.com/health
```

---

## Alert Configuration

Alert rules are created by admin users in the dashboard under **Alerts → Rules**.

### Rule Types

| Type | Trigger | Required Parameter |
|---|---|---|
| `on_failure` | Job exits with a non-zero status | None |
| `duration_exceeded` | Job run time exceeds a threshold | `threshold_seconds` |
| `missed_schedule` | Scheduled job does not start within a threshold | `threshold_seconds` |

**Example:** Create an `on_failure` rule for all jobs with Slack notification — navigate to Alerts → Rules → New Rule, set type to `on_failure`, select notification channel `slack`, save.

### Notification Channel Setup

1. Add the relevant fields to `config.json` (see Configuration section).
2. Restart the coordinator service.
3. In the dashboard, test the channel by creating a rule and triggering a test job failure, or use a webhook testing tool (e.g., `ngrok` + `requestbin`) to confirm the HMAC-signed payload arrives correctly.

**Webhook security:** Outbound webhooks are signed with HMAC-SHA256 using `notifications.webhook.secret`. Validate the `X-ArcVault-Signature` header on the receiving end.

**Retry behavior:** Failed notifications are retried 3 times with exponential backoff. Retry status is visible in the dashboard Alerts tab. Admin users can manually retry via the dashboard or `POST /api/alert-history/{id}/retry`.

**Alert history retention:** Controlled by `alert_history_retention_days` in `config.json` (default: 30 days).

---

## Agent Token Management

### Token Lifecycle

1. On the coordinator, generate a token for the agent:
   ```
   coordinator create-agent-token <agent-id> --token-only
   ```
   This outputs a 64-character hex token.
2. Copy the token into the agent's `agent-config.yaml` under `auth_token`.
3. Set `agent_id` in `agent-config.yaml` to the same `<agent-id>` used in the generate command.
4. Start (or restart) the agent service. The agent registers with the coordinator on startup.
5. Confirm the agent appears as online in the dashboard.

### Token Regeneration After Coordinator Reinstall

**Tokens are stored in the coordinator's SQLite database. A full coordinator reinstall wipes the database and invalidates all existing tokens.** After any reinstall:

1. For every registered agent, run the generate command again with the same agent ID.
2. Update `agent-config.yaml` on each agent host with the new token.
3. Restart each agent service.

Treat token regeneration as a required step in any coordinator reinstall runbook.

---

## Updates

### Self-Update via Dashboard (Recommended)

1. Navigate to **Settings → Updates** in the dashboard (admin role required).
2. If a new version is available, click **Update Coordinator** or **Update Agent** for the target agent.
3. The update is delivered over WebSocket. The coordinator or agent restarts automatically.
4. Verify the health endpoint returns 200 after restart.

**One-version rollback** is available in the same Settings → Updates panel if a new version causes issues.

### Manual Update (Coordinator — Windows)

Use `rebuild-and-restart.ps1` when deploying a locally built binary:

```powershell
.\rebuild-and-restart.ps1
```

The script builds the coordinator binary, stops the service, copies the new binary, starts the service, and polls `https://localhost/health` until a 200 is returned. TLS cert validation is skipped during the health check.

For a fully manual update:

```powershell
sc.exe stop ArcVaultCoordinator
Copy-Item coordinator.exe C:\ArcVault\coordinator.exe -Force
sc.exe start ArcVaultCoordinator
```

---

## Backup & Recovery

### Database Backup

ArcVault stores all state in a SQLite database at the coordinator install path (e.g., `C:\ArcVault\arcvault.db`). There is no built-in automated backup for the database — this is the operator's responsibility.

**Recommended:** Schedule a daily file copy of the DB to a separate volume or remote store.

```powershell
# Example: daily copy (run via Task Scheduler)
Copy-Item C:\ArcVault\arcvault.db "\\backup-server\arcvault\arcvault-$(Get-Date -Format yyyyMMdd).db"
```

When copying a live database, use SQLite's online backup API or stop the coordinator service first to ensure consistency. A stopped-service copy is the safest approach for daily backups during a maintenance window.

### Restore Procedure

1. Stop the coordinator service.
2. Replace `arcvault.db` with the backup copy.
3. Start the coordinator service.
4. Verify the health endpoint responds and the dashboard shows expected data.

**Note:** Restoring an older DB backup means agent tokens from that backup point are active again. Tokens added after the backup date will be lost and must be regenerated.

---

## Federation HA Setup

Federation allows a second coordinator (spoke) to connect to a primary coordinator (root) for high-availability and multi-site deployments. State is synchronized via an event log over HTTPS.

### Setup Steps

1. Install a second coordinator on a separate host using the standard installation procedure.
2. In the spoke coordinator's `config.json`, set the root coordinator URL.
3. Restart the spoke coordinator. It connects to the root and begins syncing state.
4. Verify both coordinators appear in the dashboard under **Settings → Federation**.

### Failover Behaviour

- Agents can be configured with a `coordinators` list in `agent-config.yaml` for automatic failover to the spoke if the root is unreachable.
- The spoke operates in read-mostly mode during normal operation; job dispatch continues if the root is offline.

### When to Use

Deploy federation when: the coordinator host is a single point of failure for critical backup jobs; you manage multiple sites and want a unified dashboard; or compliance requires geographic redundancy.

---

## Day 2 Operations Runbook

| Scenario | Action |
|---|---|
| Coordinator won't start | Check service logs (Event Viewer on Windows, `journalctl` on Linux). Verify `config.json` is valid JSON. Confirm TLS cert paths exist and are readable by the service account. Check port 443 is not in use by another process. |
| Agent shows offline in dashboard | Verify agent service is running on the host. Check `agent-config.yaml` has correct `coordinator_url` and `auth_token`. Confirm network connectivity from agent host to coordinator:443. Check if coordinator was reinstalled (token regeneration required). |
| Alert not firing | Confirm an alert rule exists for the job/condition in the dashboard. Verify notification channel config in `config.json` (restart coordinator after changes). Check alert history for failed delivery attempts. Test notification channel with a manual webhook call. |
| Disk space growing | The SQLite DB grows as job run history and alert history accumulate. Reduce `alert_history_retention_days` or archive/delete old job runs via the API. Monitor `C:\ArcVault\arcvault.db` size; plan for 1 GB per 6–12 months of heavy use. |
| Token regeneration after coordinator reinstall | Run `coordinator create-agent-token <agent-id> --token-only` for each agent. Update `auth_token` in each agent's `agent-config.yaml`. Restart each agent service. |
| Health endpoint returns non-200 | Check coordinator service status. Review logs for panic or startup errors. If recently updated, consider rolling back via dashboard or restoring previous binary. |

---

## Logging

**Log output:** Both coordinator and agent write logs to stdout. The service manager captures stdout.

- **Windows:** Logs are captured by the Windows Service Control Manager and accessible in **Event Viewer → Windows Logs → Application**, source `ArcVaultCoordinator` or `ArcVaultAgent`.
- **Linux:** Use `journalctl -u arcvault-coordinator -f` or `journalctl -u arcvault-agent -f`.
- **macOS:** Use `log stream --predicate 'subsystem == "com.arcvault"'` or check Console.app.

**Log format:** Plain text, not structured JSON. No log rotation is performed by ArcVault itself; rely on the OS service manager or a log shipper for rotation and aggregation.

---

## Reference

See [APPENDIX.md](APPENDIX.md) for:
- Full port matrix
- RBAC permission table
- config.json and agent-config.yaml full schema
- REST API endpoint reference
- Database schema summary
- System requirements table

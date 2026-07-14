# ArcVault Runbook

**Comprehensive guide to understanding, installing, operating, and troubleshooting ArcVault.**

**Version:** v0.6.0  
**Last updated:** July 13, 2026

---

## Table of Contents

1. [What Is ArcVault](#1-what-is-arcvault)
2. [Architecture Overview](#2-architecture-overview)
3. [Installation](#3-installation)
4. [Configuration](#4-configuration)
5. [Security & Encryption](#5-security--encryption)
6. [Operation](#6-operation)
7. [Maintenance](#7-maintenance)
8. [Troubleshooting](#8-troubleshooting)
9. [FAQ](#9-faq)
10. [Reference](#10-reference)

---

## 1. What Is ArcVault

ArcVault is a **self-hosted, cross-platform backup orchestrator** with a real-time web dashboard. It coordinates backup jobs across multiple machines from a single pane of glass.

### Key Capabilities

| Capability | Description |
|------------|-------------|
| **Centralized management** | One coordinator manages all agents; dashboard shows everything |
| **Cross-platform agents** | Agents run on Windows, macOS, and Linux |
| **Job scheduling** | Cron-based schedules with a visual ScheduleBuilder (Interval/Daily/Weekly/Monthly/Custom) |
| **Real-time monitoring** | Live WebSocket updates for job status, agent health, and alerts |
| **RBAC** | Three roles: admin (full), operator (manage jobs), viewer (read-only) |
| **Notifications** | Webhook (HMAC-signed), Email (SMTP), Slack, Microsoft Teams |
| **Credential profiles** | AES-256-GCM encrypted storage for SMB/SSH credentials |
| **Federation HA** | Multi-coordinator failover with state sync and health monitoring |
| **Self-update** | Coordinator and agents update via WebSocket with rollback support |
| **Audit logging** | Full user action audit trail for compliance |
| **Job cancellation** | Cancel pending jobs instantly or running jobs via WebSocket kill signal |
| **Alert engine** | Configurable rules: on_failure, duration_exceeded, missed_schedule |

### What ArcVault Is NOT

- It is **not** a backup tool itself — it orchestrates existing tools (robocopy on Windows, rsync on Unix)
- It is **not** cloud-dependent — runs entirely on your infrastructure
- It is **not** an encryption tool for backup data -- that is handled by the underlying tools or network

---

## 2. Architecture Overview

### High-Level Architecture

```
+------------------------------------------------------------------+
|                    Coordinator (Go 1.25.0)                        |
|  +------------------------------------------------------------+  |
|  |               HTTP Handler Layer                            |  |
|  |  REST API (78 endpoints) + WebSocket hub                    |  |
|  |  Auth: JWT (HS256) / agent tokens                           |  |
|  |  Middleware: admin / operator / viewer                      |  |
|  +---------------------------+--------------------------------+  |
|  |         Service Layer (DTOs)                                |  |
|  |  AgentService | JobService | UserService                    |  |
|  |  GroupService | AuditService                                |  |
|  +---------------------------+--------------------------------+  |
|  |         DB Interface Layer                                   |  |
|  |  AgentQueries | JobQueries | UserQueries                    |  |
|  |  SQLite (WAL mode, 5s busy timeout)                         |  |
|  +------------------------------------------------------------+  |
|              | WebSocket + TLS                                    |
+--------------+-----------------------------------------------------+
               |
    +----------+-----------+
    |                      |
  Agent-1               Agent-N
(Win/Mac/Linux)    (Win/Mac/Linux)
    |                      |
 Backup targets      Backup targets
(SMB / rsync SSH)   (SMB / rsync SSH)
```

### Components

#### Coordinator
The central server. A single Go binary that:
- Serves the Vue 3 dashboard (embedded in the binary)
- Hosts the REST API (78 endpoints) and WebSocket hub
- Stores all data in a local SQLite database (WAL mode, no external DB needed)
- Manages authentication (JWT + agent tokens)
- Evaluates alert rules and sends notifications
- Handles federation with other coordinators (HA)

#### Agent
A lightweight Go binary installed on each machine to be backed up:
- Registers with the coordinator using a per-agent token
- Sends heartbeat every 30 seconds
- Polls for pending jobs every 30 seconds
- Executes backup jobs: robocopy (Windows) or rsync (Unix)
- Streams real-time progress via WebSocket
- Reports results back to coordinator
- Supports self-update and rollback

#### Dashboard
A Vue 3 single-page application embedded in the coordinator binary:
- Real-time job status and history
- Agent monitoring with online/offline status
- User and group management
- Credential profile management
- Alert rule configuration and history
- Federation health monitoring
- Schedule builder UI

### Communication Protocols

| Path | Protocol | Encryption | Auth |
|------|----------|------------|------|
| Browser -> Coordinator | HTTPS (443) | TLS 1.2+ | JWT |
| Browser <-> Coordinator (live) | WebSocket over TLS | TLS 1.2+ (WSS) | JWT + origin validation |
| Agent -> Coordinator | HTTPS | TLS 1.2+ (pinned cert) | Bearer token |
| Agent <-> Coordinator (events) | WebSocket over TLS | TLS 1.2+ (WSS, pinned) | Bearer token |
| Agent -> Backup target | Local FS / SMB | SMB encryption (optional) | Windows credentials |
| Coordinator <-> Coordinator | HTTPS + WebSocket | TLS 1.2+ (mTLS) | Peer certificates |
| Coordinator -> Alert channels | HTTPS (webhooks) | TLS 1.2+ | API key / token |

### Job Lifecycle

1. **Create** -- Dashboard or API creates a job (single agent or group dispatch)
2. **Queue** -- Coordinator stores as pending; broadcasts via WebSocket
3. **Dispatch** -- Agent polls or receives push notification
4. **Execute** -- Agent runs robocopy/rsync with progress streaming
5. **Report** -- Agent posts exit code and output back to coordinator
6. **Alert** -- Coordinator evaluates alert rules; sends notifications if triggered
7. **Cancel** -- (Optional) User cancels via API; WebSocket kill signal terminates agent process

---

## 3. Installation

### Prerequisites

| Requirement | Coordinator | Agent |
|-------------|-------------|-------|
| OS | Windows 10+, Linux, macOS | Windows 10+, Linux, macOS |
| RAM | 256 MB minimum | 128 MB minimum |
| Disk | 500 MB + backup storage | 100 MB |
| Network | Static IP or hostname | Connectivity to coordinator |

### Method 1: Installer (Windows, Recommended)

Download the latest installer from [GitHub Releases](https://github.com/castrokren/ArcVault/releases).

```powershell
# Run the installer as Administrator
ArcVault-Setup-<version>-windows-amd64.exe
```

The installer:
1. Prompts you to choose **Coordinator**, **Agent**, or **Both**
2. Installs coordinator to `C:/ArcVault/`
3. Installs agent to `C:/ArcVault-Agent/`
4. Creates Windows services (`arcvault-coordinator`, `arcvault-agent`)
5. Generates TLS certificates and configuration
6. When installing both, automatically configures the agent to connect to the coordinator

### Method 2: Manual Installation (All Platforms)

#### Build from Source

```bash
# Clone the repository
git clone https://github.com/castrokren/ArcVault.git
cd ArcVault

# Build coordinator
go build -o coordinator -ldflags "-X main.Version=$(cat VERSION)" ./coordinator

# Build agent
go build -o agent -ldflags "-X main.Version=$(cat VERSION)" ./agent

# Verify version matches VERSION file
./coordinator --version
# Expected: v0.6.0
```

**Note:** Go 1.25.0+ is required. The dashboard is embedded in the coordinator binary at build time. If you modify the dashboard, rebuild via `scripts/rebuild-and-restart.ps1` (Windows) or the equivalent build steps.

**Release integrity:** Every release ships a `SHA256SUMS` file alongside the
binaries. The agent's self-updater verifies the downloaded binary against this
checksum before applying the update. The checksum and binary share the same
download origin (GitHub Releases), so this protects against transit corruption
or MITM — it does NOT protect against a compromised GitHub account or release.

#### Start the Coordinator

```bash
# On first run, initialize:
./coordinator init

# Start the coordinator
./coordinator start

# Dashboard available at https://localhost
# Or http://localhost:<port> if TLS is disabled
```

#### Install as a System Service

```bash
# Windows (run as Administrator)
coordinator install-service
agent install-service

# Linux / macOS (run as root)
sudo coordinator install-service
sudo agent install-service
```

### Method 3: Docker (Linux)

```bash
# Pull and run coordinator
docker run -d   --name arcvault-coordinator   -p 443:443   -v /data/arcvault:/data   ghcr.io/castrokren/arcvault-coordinator:latest
```

---

## 4. Configuration

### Coordinator Configuration (config.json)

Location: `C:/ArcVault/config.json` (Windows) or `/etc/arcvault/config.json` (Linux/macOS)

```json
{
  "port": 443,
  "database_path": "C:/ArcVault/arcvault.db",
  "admin_token": "your-admin-token-here",
  "jwt_secret": "your-jwt-secret-here",
  "jwt_expiry": "4h",
  "credential_key": "64-char-hex-string-for-AES-encryption",
  "environment": "production",
  "allowed_origins": ["https://dashboard.example.com"],
  "coordinator_id": "coord-01",
  "installer_dir": "C:/ArcVault/installers",
  "alert_history_retention_days": 30,
  "notifications": {
    "on_failure": true,
    "webhook": {
      "url": "https://hooks.example.com/arcvault",
      "secret": "hmac-signing-secret"
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
  },
  "federation": {
    "enabled": false,
    "peers": ["https://coordinator-02:443"]
  }
}
```

| Field | Required | Description |
|-------|----------|-------------|
| `port` | Yes | HTTP listener port (443 for TLS, 8080 for dev) |
| `database_path` | Yes | Path to SQLite database file |
| `admin_token` | Yes | Machine credential for local ops scripts (build/deploy, sanity checks). Allowlisted to a few read endpoints + agent endpoints — **not** a general admin master key. |
| `jwt_secret` | Yes | Secret for signing JWT tokens (change on first login) |
| `jwt_expiry` | No | JWT token TTL (default: 4h) |
| `credential_key` | No | 64-char hex for AES-256-GCM credential encryption |

> **Production secret sourcing (2026-07):** `admin_token`, `jwt_secret`, and `credential_key` are read from the environment variables `ARCVAULT_ADMIN_TOKEN`, `ARCVAULT_JWT_SECRET`, and `ARCVAULT_CREDENTIAL_KEY` (on the Windows service, these live in the registry key `HKLM\SYSTEM\CurrentControlSet\Services\arcvault-coordinator\Environment`). The coordinator **blanks all three from `config.json` on write**, so they no longer persist to disk. Set them in the environment, not the file.
| `environment` | No | "development" or "production" (affects defaults) |
| `allowed_origins` | No | CORS origins (default: localhost) |
| `coordinator_id` | No | Unique ID for federation |
| `installer_dir` | No | Directory for /downloads/installer endpoint |
| `alert_history_retention_days` | No | Days to keep alert history (default: 30) |

### Agent Configuration (agent-config.yaml)

Location: `C:/ArcVault-Agent/agent-config.yaml` (Windows) or `/etc/arcvault-agent/agent-config.yaml`

```yaml
coordinator_url: wss://your-coordinator:443
auth_token: <agent-token-from-coordinator>
agent_id: agent-01
ca_cert_file: C:/ArcVault-Agent/coordinator.crt
```

| Field | Required | Description |
|-------|----------|-------------|
| `coordinator_url` | Yes | Coordinator WebSocket URL (wss:// for TLS, ws:// for dev) |
| `auth_token` | Yes | Per-agent token generated by coordinator |
| `agent_id` | Yes | Unique identifier for this agent |
| `ca_cert_file` | For TLS | Path to pinned coordinator certificate |

### Managing Agent Tokens

```bash
# Generate a new agent token
coordinator create-agent-token agent-01

# List all agent tokens
coordinator list-agent-tokens

# Revoke an agent token
coordinator revoke-agent-token agent-01
```

---

## 5. Security & Encryption

### TLS Certificate Management

ArcVault generates a self-signed TLS certificate on first `coordinator init`.

```bash
# Regenerate certificate (if hostname/IP changes)
coordinator rekey-cert

# View certificate details (Windows)
certutil -text C:/ArcVault/cert.pem | findstr /C:"Subject" /C:"Issuer" /C:"Validity"
```

**Certificate locations:**
- Coordinator: `C:/ArcVault/cert.pem` and `C:/ArcVault/key.pem`
- Agent (pinned copy): `C:/ArcVault-Agent/coordinator.crt`

### Agent Certificate Pinning

Agents download and pin the coordinator certificate during registration. Every subsequent HTTPS/WSS connection validates against this pinned copy, preventing MITM attacks even if the system root CA is compromised.

### Credential Encryption

Backup credentials (passwords, SSH keys) are encrypted at rest using:
- **Algorithm:** AES-256-GCM (authenticated encryption)
- **Key source:** `credential_key` in `config.json` (64-char hex string)
- **Fallback:** Environment variable `ARCVAULT_CREDENTIAL_KEY`

### Password Policy

ArcVault enforces password complexity server-side via `validatePasswordStrength()` in `auth.go`:

| Requirement | Detail |
|-------------|--------|
| Minimum length | 8 characters |
| Uppercase | At least one uppercase letter (A-Z) |
| Lowercase | At least one lowercase letter (a-z) |
| Digit | At least one digit (0-9) |
| Special character | At least one special character (`!@#$%^&*()_+-=[]{}|;':\",./<>?~`) |

This validation runs on:
- **Login** (`POST /api/auth/login`): weak passwords are rejected even if the user exists
- **Password change** (`PUT /api/auth/change-password`): new password must meet all requirements
- **User creation** (`POST /api/users`): new user passwords must meet all requirements
- **Business layer** (`business/users.go`): minimum 8-char check as a secondary guard

The dashboard password strength meter checks all four character classes and blocks form submission on weak passwords. The meter is purely UX — the server enforces the policy regardless of what the client sends.

### Route-Level Role Guards (Admin Pages)

The dashboard enforces role-based access at the router level. Routes with `meta: { requiresRole: 'admin' }` are guarded by a `beforeEach` navigation guard that checks the JWT payload. The following pages are admin-only:

- Federation (`/federation`)
- Users (`/users`)
- Groups (`/groups`)
- Alerts (`/alerts`)
- Credentials (`/credentials`)

The credentials nav link is also conditionally rendered with `v-if="isAdmin"`. The credentials page itself performs an additional JWT decode + admin role check and redirects non-admins.

### Pagination Limits

All paginated endpoints enforce:
- **Max page:** 10,000 (`MaxPage` constant)
- **Max limit per page:** 100 (`MaxLimit` constant)
- **Default limit:** 25 (`DefaultLimit` constant)

These caps prevent resource exhaustion through unbounded pagination parameters. Combined, the maximum rows retrievable in a single paginated request is 1,000,000 (10000 pages × 100 per page).

### Production Security Checklist

```
[ ] TLS certificate generated with correct SANs for hostname/IP
[ ] environment set to "production" in config.json
[ ] WebSocket origin validation enabled (default in production)
[ ] Agents have pinned coordinator certificate
[ ] credential_key backed up in secure location
[ ] Full-disk encryption enabled on coordinator machine
[ ] SMB3 signing/encryption enabled on backup targets
[ ] TLS cert rotation planned before expiry
[ ] Regular backups of config.json
[ ] Audit logging enabled and monitored
[ ] Password complexity policy enforced for all users
[ ] Admin route middleware confirmed active on user management
```

### Development Mode

```powershell
# Skip TLS verification for local testing (NEVER in production)
$env:ARCVAULT_INSECURE_SKIP_VERIFY = "1"
```

---

## 6. Operation

### Starting and Stopping

```powershell
# Windows services
sc.exe stop arcvault-agent
sc.exe stop arcvault-coordinator
sc.exe start arcvault-coordinator
sc.exe start arcvault-agent

# Always stop agent first, start coordinator first
```

### Managing Jobs

| Action | API Endpoint | Role |
|--------|-------------|------|
| Create job | POST /api/jobs | operator+ |
| List jobs | GET /api/jobs | viewer+ |
| Get job details | GET /api/jobs/{id} | viewer+ |
| Cancel job | POST /api/jobs/{id}/cancel | operator+ |
| Delete job | DELETE /api/jobs/{id} | admin |
| View job runs | GET /api/jobs/{id}/runs | viewer+ |
| Create template | POST /api/templates | admin |
| Run template now | POST /api/templates/{id}/run | operator+ |

### Managing Agents

| Action | API Endpoint | Role |
|--------|-------------|------|
| List agents | GET /api/agents | viewer+ |
| Delete agent | DELETE /api/agents/{id} | admin |
| Update agent | POST /api/agents/{id}/update | admin |
| Rollback agent | POST /api/agents/{id}/rollback | admin |

### Managing Users & Groups

| Action | API Endpoint | Role |
|--------|-------------|------|
| List users | GET /api/users | admin |
| Create user | POST /api/users | admin |
| Delete user | DELETE /api/users/{id} | admin |
| Change role | PUT /api/users/{id}/role | admin |
| List groups | GET /api/groups | viewer+ |
| Create group | POST /api/groups | admin |
| Add agent to group | POST /api/groups/{id}/agents | admin |

### Authentication & Roles

| Role | Permissions |
|------|------------|
| **admin** | Full access -- manage users, agents, credentials, federation, system updates |
| **operator** | Create and manage jobs, run templates, cancel jobs |
| **viewer** | Read-only access to dashboards, job history, alerts |

### Notifications

ArcVault can send alerts when jobs fail:
- **Webhook:** HMAC-SHA256 signed, retries 3x with exponential backoff (5s -> 15s -> 45s)
- **Email:** SMTP with optional TLS
- **Slack:** Incoming webhook
- **Microsoft Teams:** Incoming webhook

### Monitoring

- **Dashboard:** Real-time WebSocket updates for job status and agent health
- **Alert rules:** Per-job configurable rules (on_failure, duration_exceeded, missed_schedule)
- **Alert history:** 30-day retention by default
- **Audit log:** Full user action trail for compliance

---

## 7. Maintenance

### Updating the Coordinator

```bash
# Check for updates
coordinator check-update

# Apply update (downloads and restarts)
coordinator update-apply
```

Or via the dashboard: Admin -> Check for Updates -> Apply Update.

The coordinator self-updates by:
1. Downloading the new binary from GitHub Releases
2. Validating the binary
3. Swapping the binary and restarting
4. One-version rollback available via `coordinator rollback`

### Updating Agents

```bash
# Via coordinator API — requires an admin session JWT (obtain from POST /api/auth/login).
# The admin token no longer works here: it is not accepted on admin routes.
curl -X POST https://coordinator:443/api/agents/agent-01/update   -H "Authorization: Bearer <admin-jwt>"
```

Or via the dashboard: Agents -> Select agent -> Update.

### Rotating TLS Certificate

```bash
coordinator rekey-cert
# Restart coordinator service
# Agents will re-register with new cert automatically
```

### Rotating Credential Key

```bash
coordinator rekey
# This re-encrypts all stored credentials with the new key
```

### Backing Up the Coordinator

Essential items to back up:
1. `config.json` (non-secret settings; the admin token, JWT secret, and credential key are **not** here anymore — see below)
2. The three secrets from the environment (`ARCVAULT_ADMIN_TOKEN`, `ARCVAULT_JWT_SECRET`, `ARCVAULT_CREDENTIAL_KEY` — on Windows, exported from the service registry `Environment` key). **Losing `ARCVAULT_CREDENTIAL_KEY` makes all stored credentials unrecoverable.**
3. `arcvault.db` (all job history, configuration, encrypted credentials)
4. `cert.pem` and `key.pem` (TLS certificate)

```powershell
# Example backup script
$date = Get-Date -Format "yyyy-MM-dd"
$backupDir = "D:\ArcVault-Backups\$date"
New-Item -ItemType Directory -Path $backupDir -Force

Copy-Item C:/ArcVault/config.json $backupDirCopy-Item C:/ArcVault/arcvault.db $backupDirCopy-Item C:/ArcVault/cert.pem $backupDirCopy-Item C:/ArcVault/key.pem $backupDir
Write-Host "Backup complete: $backupDir"
```

### Restoring from Backup

```powershell
# Stop services
sc.exe stop arcvault-agent
sc.exe stop arcvault-coordinator

# Restore files
Copy-Item D:\ArcVault-Backups6-07-01\* C:/ArcVault/ -Force

# Start services
sc.exe start arcvault-coordinator
sc.exe start arcvault-agent
```

---

## 8. Troubleshooting

### Dashboard Issues

#### Dashboard Shows Blank Page
**Cause:** Stale JWT in localStorage or wrong port.  
**Fix:** 
```javascript
// In browser DevTools console:
localStorage.clear(); location.reload();
```

#### Dashboard Shows "Cannot Connect"
**Cause:** Coordinator not running or wrong port.  
**Fix:**
```powershell
# Verify coordinator is running
Invoke-RestMethod -Uri "https://localhost/health" -SkipCertificateCheck
# Expected: {"status":"ok"}
```

### Agent Issues

#### Agent Not Showing in Dashboard
**Cause:** Authentication issue or agent not registered.  
**Fix:**
```powershell
# Check agent logs
Get-Content C:/ArcVault-Agent/logs/arcvault-agent.log -Tail 20

# Verify token in agent-config.yaml matches coordinator
```

#### Agent Shows 401 Unauthorized
**Cause:** Token mismatch between agent config and coordinator.  
**Fix:** Regenerate the agent token:
```powershell
coordinator create-agent-token agent-01
# Update agent-config.yaml with new token
sc.exe restart arcvault-agent
```

#### Agent TLS Error (x509: certificate signed by unknown authority)
**Cause:** Agent missing or has stale pinned certificate.  
**Fix:**
```powershell
# Re-pull the coordinator certificate
# The bootstrap script handles this, or manually:
# Copy C:/ArcVault/cert.pem to C:/ArcVault-Agent/coordinator.crt
```

#### Agent Shows Error 1067 (Windows Service)
**Cause:** Invalid configuration or connection failure.  
**Fix:**
```powershell
# Check the agent log for details
Get-Content C:/ArcVault-Agent/logs/arcvault-agent.log -Tail 30
# Common causes: wrong coordinator_url, invalid token, TLS mismatch
```

### Connection Issues

#### Port Already in Use
**Fix:** Change port in config.json, or kill the process using the port:
```powershell
netstat -ano | findstr :8080
taskkill /PID <pid> /F
```

#### Federation Connection Errors (Non-Critical)
If you see errors like `GET http://localhost:443/api/rollback-available` in the browser console, these are harmless for single-coordinator deployments. They occur because the dashboard probes for federation endpoints that do not exist in standalone mode.

### Job Issues

#### Jobs Stuck in "Pending"
**Cause:** Agent not polling or coordinator not dispatching.  
**Fix:**
```powershell
# Check agent is online
Invoke-RestMethod -Uri "https://localhost/api/agents" `
  -Headers @{Authorization = "Bearer <token>"}

# Check job status
Invoke-RestMethod -Uri "https://localhost/api/jobs" `
  -Headers @{Authorization = "Bearer <token>"}
```

#### Jobs Complete with Exit Code 9 (Robocopy)
This is **not an error**. Exit code 9 means "partial success" -- some files were copied, some were skipped due to matching criteria. ArcVault normalizes codes 1-15 to 0 (success).

### Service Management

```powershell
# Check service status
sc.exe query arcvault-coordinator
sc.exe query arcvault-agent

# View service logs (Windows Event Viewer)
Get-WinEvent -LogName Application | Where-Object { $_.ProviderName -like "*arcvault*" }

# Restart both services (coordinator first, agent first on stop)
sc.exe stop arcvault-agent
sc.exe stop arcvault-coordinator
sc.exe start arcvault-coordinator
sc.exe start arcvault-agent
```

### Recovery

#### Lost credential_key
If `ARCVAULT_CREDENTIAL_KEY` (service registry `Environment`) is lost, all stored credentials become unrecoverable. Create new credentials via the dashboard and re-authorize backup jobs.

**Prevention:** Back up the `ARCVAULT_CREDENTIAL_KEY` value in a secure location (a secrets manager, not next to `arcvault.db`).

#### Lost admin_token
The admin token is a machine credential for local ops scripts; agents do **not** use it (they authenticate with their own per-agent tokens). To rotate:
1. Stop the coordinator service
2. Set a new value in `ARCVAULT_ADMIN_TOKEN` (service registry `Environment`)
3. Restart the coordinator service
4. Update the token in any local ops scripts that read it

---

## 9. FAQ

**Q: Can I run ArcVault without TLS/HTTPS?**  
A: Yes, for development. Set `"environment": "development"` in config.json and use `http://` and `ws://` URLs. Never do this in production.

**Q: How many agents can one coordinator handle?**  
A: This depends on hardware, but the coordinator has been tested with 50+ concurrent agents. SQLite WAL mode handles concurrent reads well, with a 5-second busy timeout for writes.

**Q: Can I use an existing SQLite database?**  
A: The coordinator creates and manages its own database schema via auto-migration on startup. Point it at an existing file and it will add any missing tables.

**Q: How do I move the coordinator to a new machine?**  
A: Back up config.json, arcvault.db, and cert.pem/key.pem from the old machine. Restore them on the new machine at the same paths. Reinstall agents or update their coordinator_url.

**Q: Does ArcVault support IPv6?**  
A: Yes, Go net/http stack handles IPv6 natively. Use bracketed addresses in URLs: `https://[::1]:443`.

**Q: Can I run the dashboard on a different port than the API?**  
A: No. The dashboard is embedded in the coordinator binary and served from the same port.

**Q: What happens if the coordinator goes offline while a job is running?**  
A: The agent continues running the job locally. When the coordinator comes back online, the agent reports the result. Running jobs are not lost.

**Q: How do I contribute?**  
A: See the project on GitHub. Pull requests are welcome. The architecture uses a three-layer pattern (Handler -> Service -> DB Interface) with Zod schema validation on the frontend.

---

## 10. Reference

### Key File Paths

| Item | Windows | Linux/macOS |
|------|---------|-------------|
| Coordinator binary | `C:/ArcVault/coordinator.exe` | `/usr/local/bin/coordinator` |
| Agent binary | `C:/ArcVault-Agent/agent.exe` | `/usr/local/bin/agent` |
| Coordinator config | `C:/ArcVault/config.json` | `/etc/arcvault/config.json` |
| Agent config | `C:/ArcVault-Agent/agent-config.yaml` | `/etc/arcvault-agent/agent-config.yaml` |
| Database | `C:/ArcVault/arcvault.db` | `/var/lib/arcvault/arcvault.db` |
| TLS certificate | `C:/ArcVault/cert.pem` | `/etc/arcvault/cert.pem` |
| TLS key | `C:/ArcVault/key.pem` | `/etc/arcvault/key.pem` |
| Agent cert pin | `C:/ArcVault-Agent/coordinator.crt` | `/etc/arcvault-agent/coordinator.crt` |
| Agent logs | `C:/ArcVault-Agent/logs/arcvault-agent.log` | `/var/log/arcvault-agent.log` |
| Dev project | `C:/Projects/ArcVault2.0/` | `~/ArcVault/` |

### Service Names

| Service | Windows | Linux |
|---------|---------|-------|
| Coordinator | `arcvault-coordinator` | `arcvault-coordinator` |
| Agent | `arcvault-agent` | `arcvault-agent` |

### Default Ports

| Traffic | Port |
|---------|------|
| HTTPS (TLS) | 443 |
| HTTP (development) | 8080 |

### Default Credentials

| Field | Value |
|-------|-------|
| Username | `admin` |
| Password | `changeme` (must change on first login) |

### API Base Url

- Production: `https://<coordinator-host>/api`
- Development: `http://localhost:8080/api`

### Useful Commands

```bash
# Coordinator
coordinator init              # First-time setup (generates cert + config)
coordinator start             # Start the server
coordinator stop              # Stop the server
coordinator --version         # Show version
coordinator rekey-cert        # Regenerate TLS certificate
coordinator rekey             # Rotate credential encryption key
coordinator check-update      # Check for updates
coordinator create-agent-token <id>   # Generate agent token
coordinator list-agent-tokens         # List all agent tokens
coordinator revoke-agent-token <id>   # Revoke agent token
coordinator install-service   # Install as Windows/system service
coordinator uninstall-service # Remove service

# Agent
agent start                   # Start the agent
agent stop                    # Stop the agent
agent --version               # Show version
agent install-service         # Install as Windows/system service
agent uninstall-service       # Remove service
```

### Related Documentation

| Document | Location | Content |
|----------|----------|---------|
| API Specification | `API_SPECIFICATION.md` | All REST/WS endpoints with schemas |
| Architecture | `docs/architecture/` | Diagrams and security documentation |
| Codebase Primer | `CODEBASE.md` | Developer architecture reference |
| Encryption Details | `docs/architecture/ENCRYPTION.md` | TLS, cert pinning, credential encryption |
| Documentation Runbook | `docs/DOCUMENTATION_RUNBOOK.md` | How to keep docs in sync |
| Build Guides | `docs/BUILD_*.md` | Platform-specific installer builds |

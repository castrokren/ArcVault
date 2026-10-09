# ArcVault — Deployment Guide

This guide covers deploying ArcVault in a production Windows environment. It walks through installing the backup agent as a Windows Service, deploying the controller, securing the stack with SSL and JWT authentication, and ongoing maintenance.

---

## Table of Contents

- [Prerequisites](#prerequisites)
- [Architecture Decision: Where to Deploy](#architecture-decision-where-to-deploy)
- [Option A: Automated Deployment (Recommended)](#option-a-automated-deployment-recommended)
- [Option B: Manual Deployment](#option-b-manual-deployment)
  - [Step 1 — Install Python](#step-1--install-python)
  - [Step 2 — Deploy the Agent](#step-2--deploy-the-agent)
  - [Step 3 — Generate SSL Certificates](#step-3--generate-ssl-certificates)
  - [Step 4 — Generate JWT Key Pair](#step-4--generate-jwt-key-pair)
  - [Step 5 — Configure the Agent](#step-5--configure-the-agent)
  - [Step 6 — Install the Windows Service](#step-6--install-the-windows-service)
  - [Step 7 — Deploy the Controller](#step-7--deploy-the-controller)
  - [Step 8 — Configure the Controller](#step-8--configure-the-controller)
  - [Step 9 — Register the Agent in the Controller](#step-9--register-the-agent-in-the-controller)
  - [Step 10 — Verify the Installation](#step-10--verify-the-installation)
- [Upgrading](#upgrading)
- [Rollback](#rollback)
- [Using the Desktop GUI](#using-the-desktop-gui)
- [nginx Reverse Proxy (Optional)](#nginx-reverse-proxy-optional)
- [Firewall Rules](#firewall-rules)
- [Maintenance](#maintenance)
- [Troubleshooting](#troubleshooting)

---

## Prerequisites

| Requirement | Minimum | Notes |
|-------------|---------|-------|
| OS | Windows 10 / Windows Server 2019 | 64-bit required |
| Python | 3.9+ (3.11 recommended) | Agent and controller each use separate venvs |
| PowerShell | 5.1+ | For deployment scripts |
| Privileges | Local Administrator | Required for Windows Service installation |
| Disk space | 2 GB free on install drive | For Python venv + dependencies |
| Network | Agent port reachable from controller | Default: TCP 8443 |

Python is auto-installed by the deployment script if not present. To install manually:

```powershell
winget install Python.Python.3.11
# or
choco install python311
```

---

## Architecture Decision: Where to Deploy

**Backup Agent** — install on every machine whose files need to be backed up. Each agent runs independently and writes to its own local SQLite database.

**Backup Controller** — install on any machine with network access to all agents. This is typically a dedicated ops/management host, but it can run on the same machine as an agent in smaller environments.

```
[Controller Host]          [Machine A]          [Machine B]
 Streamlit :8501  ──────►  Agent :8443          Agent :8443
                           (backs up A)         (backs up B)
```

---

## Option A: Automated Deployment (Recommended)

The `Deploy-ArcVault.ps1` script handles everything: Python installation, virtual environment, dependencies, SSL cert generation, JWT key generation, Windows Service registration, and post-install verification.

**Run as Administrator:**

```powershell
# Standard installation to default paths
.\deployment\Deploy-ArcVault.ps1

# Custom install path and port
.\deployment\Deploy-ArcVault.ps1 -InstallPath "D:\ArcVault" -Port 8443

# Non-interactive (CI/CD, remote deployment)
.\deployment\Deploy-ArcVault.ps1 -AutoInstallPython -Silent -LogPath "C:\Logs\arcvault_install.log"

# Development mode (no Windows Service, console run only)
.\deployment\Deploy-ArcVault.ps1 -Development
```

**Script parameters:**

| Parameter | Default | Description |
|-----------|---------|-------------|
| `-InstallPath` | `C:\Program Files\ArcVault` | Application install directory |
| `-DataPath` | `C:\ProgramData\ArcVault` | Keys, database, and config directory |
| `-Port` | `8443` | Agent listen port |
| `-SourcePath` | `..\apps\backup-agent` | Path to ArcVault source files (relative to script) |
| `-Development` | false | Skip service install; run in console mode |
| `-Silent` | false | Suppress interactive prompts and console output |
| `-AutoInstallPython` | false | Download and install Python 3.11 automatically |
| `-SkipBackup` | false | Skip backup of existing installation before upgrade |
| `-LogPath` | `%TEMP%\arcvault_deployment.log` | Installation log path |

After completion, check the log at `%TEMP%\arcvault_deployment.log` and verify the service is running:

```powershell
Get-Service ArcVaultAgent
# Status should be: Running
```

---

## Option B: Manual Deployment

Follow these steps if you need fine-grained control or are deploying in an environment where the automated script cannot run.

### Step 1 — Install Python

```powershell
# Verify Python is available
python --version   # Need 3.9+

# If not installed:
winget install Python.Python.3.11
# Close and reopen PowerShell after installation to refresh PATH
```

---

### Step 2 — Deploy the Agent

```powershell
# Create the install and data directories
New-Item -ItemType Directory -Force -Path "C:\Program Files\ArcVault"
New-Item -ItemType Directory -Force -Path "C:\ProgramData\ArcVault\keys"
New-Item -ItemType Directory -Force -Path "C:\ProgramData\ArcVault\logs"

# Copy application files
Copy-Item -Recurse "apps\backup-agent\*" "C:\Program Files\ArcVault\"

# Create and populate the virtual environment
cd "C:\Program Files\ArcVault"
python -m venv .venv
.venv\Scripts\Activate.ps1
pip install --upgrade pip
pip install -r requirements_unified.txt

# Run pywin32 post-install (required for Windows Service support)
python .venv\Scripts\pywin32_postinstall.py -install
```

---

### Step 3 — Generate SSL Certificates

ArcVault ships a self-signed certificate generator. Run it from the deployment directory or via the GUI.

**Via script:**

```powershell
cd "C:\Program Files\ArcVault"
.venv\Scripts\Activate.ps1
python scripts\generate_self_signed_cert.py `
    --output-dir "C:\ProgramData\ArcVault\keys" `
    --hostname (hostname)
```

This creates three files in `C:\ProgramData\ArcVault\keys\`:

| File | Purpose |
|------|---------|
| `server.crt` | Agent TLS certificate |
| `server.key` | Agent TLS private key |
| `ca-bundle.crt` | CA bundle — distribute to controller hosts |

**Permissions:** Restrict the key file so only the service account can read it:

```powershell
$acl = Get-Acl "C:\ProgramData\ArcVault\keys\server.key"
$acl.SetAccessRuleProtection($true, $false)
$rule = New-Object System.Security.AccessControl.FileSystemAccessRule(
    "SYSTEM", "FullControl", "Allow")
$acl.AddAccessRule($rule)
Set-Acl "C:\ProgramData\ArcVault\keys\server.key" $acl
```

---

### Step 4 — Generate JWT Key Pair

```powershell
cd "C:\Program Files\ArcVault"
.venv\Scripts\Activate.ps1
python scripts\generate_jwt_keys.py `
    --output-dir "C:\ProgramData\ArcVault\keys"
```

This creates:

| File | Where it goes | Purpose |
|------|---------------|---------|
| `controller_private.pem` | Controller host only | Controller uses this to mint JWTs |
| `controller_public.pem` | Agent host | Agent uses this to verify JWTs |

**Important:** Copy `controller_public.pem` to every agent host. Keep `controller_private.pem` on the controller host only and never commit it to source control.

```powershell
# Restrict private key permissions
icacls "C:\ProgramData\ArcVault\keys\controller_private.pem" /inheritance:r
icacls "C:\ProgramData\ArcVault\keys\controller_private.pem" /grant:r "SYSTEM:(F)"
icacls "C:\ProgramData\ArcVault\keys\controller_private.pem" /grant:r "${env:USERNAME}:(F)"
```

---

### Step 5 — Configure the Agent

Create `C:\ProgramData\ArcVault\.env` (or set as system environment variables via Group Policy / the Windows registry):

```ini
# Agent bind settings
AGENT_BIND_HOST=0.0.0.0
AGENT_BIND_PORT=8443

# TLS
AGENT_CERTFILE=C:\ProgramData\ArcVault\keys\server.crt
AGENT_KEYFILE=C:\ProgramData\ArcVault\keys\server.key

# JWT authentication (production — required)
AGENT_REQUIRE_JWT=true
AGENT_JWT_PUBLIC_KEY_PATH=C:\ProgramData\ArcVault\keys\controller_public.pem
AGENT_JWT_ISSUER=backup-controller
AGENT_JWT_AUDIENCE=backup-agent
AGENT_JWT_LEEWAY=60

# Data paths
ARCVAULT_DB_PATH=C:\ProgramData\ArcVault\jobs.db

# Concurrency
AGENT_MAX_CONCURRENT_JOBS=3

# Public URL reported to controller (replace with actual hostname)
AGENT_PUBLIC_HOST=MYMACHINE.corp.example.com
AGENT_PUBLIC_SCHEME=https
```

Create `C:\Program Files\ArcVault\agent\service_config.json` to set the service host/port:

```json
{
  "host": "0.0.0.0",
  "port": 8443
}
```

**Loading the `.env` file:** Set the variables as system environment variables so the Windows Service process inherits them:

```powershell
# Read the .env file and set as system-level environment variables
Get-Content "C:\ProgramData\ArcVault\.env" |
  Where-Object { $_ -notmatch '^\s*#' -and $_ -match '=' } |
  ForEach-Object {
    $key, $val = $_ -split '=', 2
    [System.Environment]::SetEnvironmentVariable($key.Trim(), $val.Trim(), 'Machine')
  }
```

---

### Step 6 — Install the Windows Service

```powershell
cd "C:\Program Files\ArcVault"
.venv\Scripts\Activate.ps1

# Install and configure the service
python agent\win_service.py install
python agent\win_service.py --startup auto

# Start the service
Start-Service ArcVaultAgent

# Verify it's running
Get-Service ArcVaultAgent
```

The service will be listed as **ArcVault Agent Service** in `services.msc`. It logs to `%TEMP%\arcvault_service.log`.

**Service management commands:**

```powershell
Start-Service ArcVaultAgent
Stop-Service ArcVaultAgent
Restart-Service ArcVaultAgent

# Remove the service entirely
python agent\win_service.py remove
```

---

### Step 7 — Deploy the Controller

The controller runs on a separate host (or the same host in smaller setups).

```powershell
# Copy controller files to the controller host
# (adjust path as needed)
New-Item -ItemType Directory -Force -Path "C:\Program Files\ArcVaultController"
Copy-Item -Recurse "apps\backup-control\*" "C:\Program Files\ArcVaultController\"

cd "C:\Program Files\ArcVaultController"
python -m venv .venv
.venv\Scripts\Activate.ps1
pip install --upgrade pip
pip install -r requirements.txt   # or: pip install -e .
```

---

### Step 8 — Configure the Controller

Create `C:\ProgramData\ArcVault\.env` on the controller host:

```ini
# JWT signing
CTRL_JWT_PRIVATE_KEY_PATH=C:\ProgramData\ArcVault\keys\controller_private.pem
CTRL_JWT_ISSUER=backup-controller
CTRL_JWT_AUDIENCE=backup-agent

# SSL verification — use false for self-signed agent certs
AGENT_VERIFY_SSL=false
# Or provide your CA bundle:
# AGENT_CA_BUNDLE=C:\ProgramData\ArcVault\keys\ca-bundle.crt

# Default agent URL (shown in the UI on first launch)
AGENT_URL=https://MYMACHINE.corp.example.com:8443
```

**Start the controller:**

```powershell
cd "C:\Program Files\ArcVaultController"
.venv\Scripts\Activate.ps1

# Load environment
Get-Content "C:\ProgramData\ArcVault\.env" |
  Where-Object { $_ -notmatch '^\s*#' -and $_ -match '=' } |
  ForEach-Object {
    $key, $val = $_ -split '=', 2
    $env:($key.Trim()) = $val.Trim()
  }

streamlit run controller\app.py --server.port 8501 --server.address 0.0.0.0
```

To run the controller as a service, use NSSM (included in the repo as `nssm.exe`):

```powershell
.\nssm.exe install ArcVaultController "C:\Program Files\ArcVaultController\.venv\Scripts\streamlit.exe"
.\nssm.exe set ArcVaultController AppParameters "run controller\app.py --server.port 8501 --server.address 0.0.0.0"
.\nssm.exe set ArcVaultController AppDirectory "C:\Program Files\ArcVaultController"
.\nssm.exe set ArcVaultController Start SERVICE_AUTO_START
Start-Service ArcVaultController
```

---

### Step 9 — Register the Agent in the Controller

1. Open the controller dashboard at `http://<controller-host>:8501`
2. In the left sidebar, expand **Manage agents**
3. Enter the agent **Name** (e.g., `MYMACHINE`) and **URL** (e.g., `https://MYMACHINE.corp.example.com:8443`)
4. Click **Add agent**
5. The agent should appear as 🟢 if reachable

Agent URLs are persisted to `controller/agents.json` and survive restarts.

---

### Step 10 — Verify the Installation

```powershell
# 1. Check the agent health endpoint
Invoke-RestMethod -Uri "https://MYMACHINE.corp.example.com:8443/health" `
    -SkipCertificateCheck
# Expected: {"status":"ok","version":"0.1.0"}

# 2. Check the agent info endpoint
Invoke-RestMethod -Uri "https://MYMACHINE.corp.example.com:8443/info" `
    -SkipCertificateCheck
# Expected: {hostname, public_url, job_counts, ...}

# 3. Verify the service log
Get-Content "$env:TEMP\arcvault_service.log" -Tail 20

# 4. Verify the job database was created
Test-Path "C:\ProgramData\ArcVault\jobs.db"   # Should be True

# 5. Run the built-in verification script
.\deployment\Verify-ArcVault.ps1
```

---

## Upgrading

Use the upgrade script, which backs up the current installation before applying changes:

```powershell
# Run as Administrator
.\deployment\Upgrade-ArcVault.ps1 -InstallPath "C:\Program Files\ArcVault"
```

The script:
1. Stops the `ArcVaultAgent` Windows Service
2. Creates a timestamped backup of the current install
3. Copies new application files
4. Updates Python dependencies (`pip install -r requirements_unified.txt`)
5. Restarts the service
6. Runs post-upgrade verification

For a manual upgrade without the script:

```powershell
Stop-Service ArcVaultAgent

# Update files
Copy-Item -Recurse "apps\backup-agent\*" "C:\Program Files\ArcVault\" -Force

# Update dependencies
cd "C:\Program Files\ArcVault"
.venv\Scripts\pip.exe install -r requirements_unified.txt

Start-Service ArcVaultAgent
```

---

## Rollback

If an upgrade causes issues:

```powershell
# Run as Administrator
.\deployment\Rollback-ArcVault.ps1 -InstallPath "C:\Program Files\ArcVault"
```

The rollback script lists available backups (timestamped directories created by `Upgrade-ArcVault.ps1`) and restores the selected one. The SQLite database at `C:\ProgramData\ArcVault\jobs.db` is **not** touched by rollback — job history is preserved.

---

## Using the Desktop GUI

The bundled `arcvault.exe` (or `arcvault_unified.py` during development) provides a PyQt5 GUI for setup tasks that don't require command-line access.

```powershell
# Launch the GUI
.\arcvault.exe
# or in development:
python arcvault_unified.py
```

**GUI tabs:**

- **⚙️ Service** — Install, start, stop, and uninstall the Windows Service. Enter the desired host and port, then click **Save & Install Service**.
- **🔐 SSL Certificates** — Generate a self-signed certificate. Enter the hostname, output directory, and click **Generate**. Files are created at the specified path.
- **🔑 JWT Keys** — Generate the RSA-2048 key pair. Click **Generate** to create `controller_private.pem` and `controller_public.pem` in the specified directory.
- **ℹ️ About** — Version and build information.

**Headless (CLI) modes:**

```powershell
# Generate SSL cert without GUI
arcvault.exe --gen-cert --hostname MYMACHINE --output-dir C:\ProgramData\ArcVault\keys

# Generate JWT keys without GUI
arcvault.exe --gen-jwt --output-dir C:\ProgramData\ArcVault\keys

# Install as Windows Service without GUI
arcvault.exe install
```

---

## nginx Reverse Proxy (Optional)

If you want to front the agent with nginx (for example, to terminate TLS at the proxy and use a proper CA-signed cert), a sample config is included in the `nginx/` directory.

Key points:
- Proxy `https://<nginx-host>/` → `http://localhost:8443`
- Set `proxy_read_timeout 3600;` to support long-lived SSE log streams
- Pass `X-API-Key` and `Authorization` headers through to the agent

```nginx
location / {
    proxy_pass         http://127.0.0.1:8443;
    proxy_http_version 1.1;
    proxy_set_header   Host              $host;
    proxy_set_header   X-Real-IP         $remote_addr;
    proxy_set_header   X-Forwarded-For   $proxy_add_x_forwarded_for;
    proxy_set_header   X-API-Key         $http_x_api_key;
    proxy_set_header   Authorization     $http_authorization;
    proxy_buffering    off;               # Required for SSE
    proxy_cache        off;
    proxy_read_timeout 3600s;
    chunked_transfer_encoding on;
}
```

When using nginx for TLS termination, set `AGENT_CERTFILE` / `AGENT_KEYFILE` on the **agent** to empty (so uvicorn serves plain HTTP on localhost) and let nginx handle the certificate.

---

## Firewall Rules

Open the agent port on every machine running the agent:

```powershell
# Allow inbound TCP on the agent port (default 8443)
New-NetFirewallRule `
    -DisplayName "ArcVault Agent" `
    -Direction Inbound `
    -Protocol TCP `
    -LocalPort 8443 `
    -Action Allow `
    -Profile Domain, Private

# If running the controller on a separate port
New-NetFirewallRule `
    -DisplayName "ArcVault Controller" `
    -Direction Inbound `
    -Protocol TCP `
    -LocalPort 8501 `
    -Action Allow `
    -Profile Domain, Private
```

The controller only needs outbound access to agent hosts on port 8443 — no additional inbound rules required on the controller host unless operators access the dashboard remotely.

---

## Maintenance

### Clean Up Old Job Records

Old completed jobs accumulate in the SQLite database and `#backup_logs` directories. Purge them via the agent API:

```powershell
$token = $env:AGENT_TOKEN   # or mint a JWT
Invoke-RestMethod `
    -Method Post `
    -Uri "https://MYMACHINE.corp.example.com:8443/maintenance/cleanup?days=30" `
    -Headers @{"X-API-Key" = $token} `
    -SkipCertificateCheck
```

Schedule this as a weekly Scheduled Task:

```powershell
$action = New-ScheduledTaskAction -Execute "powershell.exe" -Argument `
    "-NonInteractive -Command `"Invoke-RestMethod -Method Post -Uri 'https://localhost:8443/maintenance/cleanup?days=30' -Headers @{'X-API-Key'=`$env:AGENT_TOKEN} -SkipCertificateCheck`""

$trigger = New-ScheduledTaskTrigger -Weekly -DaysOfWeek Sunday -At "02:00"
Register-ScheduledTask -TaskName "ArcVault Cleanup" -Action $action -Trigger $trigger -RunLevel Highest
```

### Rotate JWT Keys

```powershell
# 1. Generate new key pair
python scripts\generate_jwt_keys.py --output-dir "C:\ProgramData\ArcVault\keys\new"

# 2. Copy new public key to all agent hosts
# (agents will accept new tokens immediately after key file is replaced)
Copy-Item "C:\ProgramData\ArcVault\keys\new\controller_public.pem" `
          "C:\ProgramData\ArcVault\keys\controller_public.pem" -Force

# 3. Replace the private key on the controller host
Copy-Item "C:\ProgramData\ArcVault\keys\new\controller_private.pem" `
          "C:\ProgramData\ArcVault\keys\controller_private.pem" -Force

# 4. Restart the controller to pick up the new key
Restart-Service ArcVaultController   # or restart the Streamlit process
```

The 60-second JWT leeway (`AGENT_JWT_LEEWAY=60`) means there is a brief window during rollout where old tokens from the controller are still accepted by agents that have received the new public key. Complete the key swap within that window.

### Renew SSL Certificates

Self-signed certificates generated by ArcVault are valid for 1 year by default. Renew before expiry:

```powershell
# Stop the agent service first
Stop-Service ArcVaultAgent

# Regenerate the certificate
python scripts\generate_self_signed_cert.py `
    --output-dir "C:\ProgramData\ArcVault\keys" `
    --hostname (hostname)

# Restart the agent
Start-Service ArcVaultAgent

# Distribute the new ca-bundle.crt to the controller host
```

---

## Troubleshooting

### Service fails to start

1. Check the service log: `Get-Content "$env:TEMP\arcvault_service.log" -Tail 50`
2. Verify port availability: `Test-NetConnection -ComputerName localhost -Port 8443`
3. Confirm environment variables are set at the machine level: `[System.Environment]::GetEnvironmentVariable("AGENT_REQUIRE_JWT", "Machine")`
4. Confirm the certificate files exist and are readable by the SYSTEM account

### `401 Unauthorized` errors

- **Static token mode:** Confirm `AGENT_TOKEN` is identical on both agent and controller.
- **JWT mode:** Confirm `CTRL_JWT_PRIVATE_KEY_PATH` and `AGENT_JWT_PUBLIC_KEY_PATH` point to the matching key pair. Check that `AGENT_JWT_ISSUER` and `CTRL_JWT_ISSUER` match, and likewise `AGENT_JWT_AUDIENCE` / `CTRL_JWT_AUDIENCE`.
- Check the agent log for the specific JWT error: `Invalid JWT: Signature verification failed` indicates a key mismatch; `Invalid JWT: Signature has expired` indicates a clock skew issue (increase `AGENT_JWT_LEEWAY`).

### SSL certificate errors

- If the controller shows SSL errors against a self-signed cert, set `AGENT_VERIFY_SSL=false` or point `AGENT_CA_BUNDLE` to the `ca-bundle.crt` distributed from the agent host.
- Confirm the cert's CN or SAN matches the hostname in the agent URL registered in the controller.

### Jobs stuck in `running` after agent restart

On restart, the agent automatically marks any `queued`/`running` jobs as `failed` with the reason "Service restarted - job interrupted". If jobs are still showing `running` after the service restarts, the database file may be locked by another process. Check with:

```powershell
# Check for locked database
& "C:\Program Files\ArcVault\.venv\Scripts\python.exe" -c "
import sqlite3
conn = sqlite3.connect('C:/ProgramData/ArcVault/jobs.db')
print(list(conn.execute('SELECT id, state FROM jobs WHERE state IN (\"running\", \"queued\")')))
conn.close()
"
```

### Low disk space warning in logs

The agent checks for at least 1 GB free on the destination before running robocopy. Ensure the destination volume has sufficient space or redirect logs elsewhere via `ARCVAULT_DB_PATH`.

### Controller can't reach agent (🔴 in dashboard)

1. Confirm the agent service is running: `Get-Service ArcVaultAgent`
2. Test direct connectivity: `Invoke-RestMethod "https://<agent-host>:8443/health" -SkipCertificateCheck`
3. Check firewall: `Test-NetConnection -ComputerName <agent-host> -Port 8443`
4. Confirm the URL in `agents.json` / the controller sidebar matches what the agent is listening on
5. Use the **Refresh** button in the controller sidebar to trigger auto-discovery via `/info`

# ArcVault Backup Agent - Windows Deployment Guide

## Overview

The ArcVault Backup Agent is a lightweight FastAPI service that runs on Windows machines to execute backup jobs remotely. It provides a REST API for receiving backup commands and streams live logs back to the controller.

## 🚀 Quick Start (Windows)

### Prerequisites
- Windows 10/11 or Windows Server 2016+
- PowerShell 5.1 or later
- Administrator privileges

### Option 1: Use Pre-built Executable (Recommended)
```powershell
# Download or build ArcVaultAgent.exe to dist/ directory
# Run PowerShell as Administrator
.\Install-ArcVaultAgent.ps1 -AgentToken "your-secure-token" -PublicUrl "http://your-ip:8443"
```

### Option 2: Run from Source (Development)
```powershell
# Install Python 3.9+ first
python -m venv .venv
.\.venv\Scripts\Activate.ps1
pip install -e .

# Set environment variables
$env:AGENT_TOKEN = "dev-token"
$env:AGENT_BIND_HOST = "0.0.0.0"
$env:AGENT_BIND_PORT = "8443"

# Run the agent
python agent/run_agent.py
```

## 📦 Installation Methods

### Method 1: Automated Installer (Production)

The automated installer creates a scheduled task that runs the agent headlessly:

```powershell
# Basic installation
.\Install-ArcVaultAgent.ps1 -AgentToken "secure-token-123" -PublicUrl "http://192.168.1.100:8443"

# Interactive setup with prompts
.\Install-ArcVaultAgent.ps1 -Interactive

# Custom configuration
.\Install-ArcVaultAgent.ps1 -AgentToken "token" -ListenHost "0.0.0.0" -Port 8443 -PublicHost "192.168.1.100"
```

**What gets installed:**
- Scheduled Task: "ArcVault Agent" (auto-starts with Windows)
- Installation Directory: `C:\ProgramData\ArcVault\`
- Startup Script: `ArcVaultAgent-Startup.ps1`
- Log File: `agent.log`

### Method 2: Manual Installation

1. **Download/Build the executable:**
   ```powershell
   # If building from source
   python -m venv .venv_build
   .\.venv_build\Scripts\Activate.ps1
   pip install pyinstaller
   pip install -e .
   pyinstaller --onefile --name ArcVaultAgent agent/run_agent.py
   ```

2. **Configure environment variables:**
   ```powershell
   $env:AGENT_TOKEN = "your-secure-token"
   $env:AGENT_BIND_HOST = "0.0.0.0"
   $env:AGENT_BIND_PORT = "8443"
   $env:AGENT_PUBLIC_URL = "http://your-public-ip:8443"
   ```

3. **Run the agent:**
   ```powershell
   .\dist\ArcVaultAgent.exe
   ```

### Method 3: Development Setup

For development and testing:

```powershell
# Clone and setup
git clone <repository>
cd ArcVault/apps/backup-agent

# Create virtual environment
python -m venv .venv
.\.venv\Scripts\Activate.ps1

# Install dependencies
pip install -e .

# Run with development settings
$env:AGENT_TOKEN = "dev-token"
uvicorn agent.app:app --host 0.0.0.0 --port 8443 --reload
```

## ⚙️ Configuration

### Environment Variables

| Variable | Description | Default | Required |
|----------|-------------|---------|----------|
| `AGENT_TOKEN` | Authentication token | | ✅ Yes |
| `AGENT_BIND_HOST` | Host to bind to | `0.0.0.0` | No |
| `AGENT_BIND_PORT` | Port to listen on | `8443` | No |
| `AGENT_PUBLIC_URL` | Complete public URL | Auto-detect | No |
| `AGENT_PUBLIC_HOST` | Public hostname/IP | Auto-detect | No |
| `AGENT_PUBLIC_SCHEME` | URL scheme | `https` | No |
| `AGENT_CERTFILE` | SSL certificate file | | No |
| `AGENT_KEYFILE` | SSL private key file | | No |

### JWT Authentication (Production)

For enhanced security, use JWT tokens:

```powershell
# On the agent
$env:AGENT_REQUIRE_JWT = "true"
$env:AGENT_JWT_PUBLIC_KEY_PATH = "C:\ProgramData\ArcVault\controller_public.pem"
$env:AGENT_JWT_ISSUER = "backup-controller"
$env:AGENT_JWT_AUDIENCE = "backup-agent"
```

### SSL/TLS Configuration

For HTTPS support:

```powershell
$env:AGENT_CERTFILE = "C:\certs\agent.crt"
$env:AGENT_KEYFILE = "C:\certs\agent.key"
```

## 🔧 Management

### Using the Management Helper

```powershell
# Check status
.\Manage-ArcVaultAgent.ps1 -Action Status

# View logs
.\Manage-ArcVaultAgent.ps1 -Action Logs

# Restart agent
.\Manage-ArcVaultAgent.ps1 -Action Restart

# Test installation
.\Manage-ArcVaultAgent.ps1 -Action Test

# Follow logs in real-time
.\Manage-ArcVaultAgent.ps1 -Action Logs -Follow
```

### Manual Commands

```powershell
# Start the scheduled task
Start-ScheduledTask -TaskName "ArcVault Agent"

# Stop the agent process
Stop-Process -Name "ArcVaultAgent" -Force

# Check if agent is running
Get-Process -Name "ArcVaultAgent" -ErrorAction SilentlyContinue

# View recent logs
Get-Content "C:\ProgramData\ArcVault\agent.log" -Tail 20

# Test connectivity
Invoke-WebRequest -Uri "http://localhost:8443/health" -UseBasicParsing
```

### Service Status Check

```powershell
# Check scheduled task
Get-ScheduledTask -TaskName "ArcVault Agent"
Get-ScheduledTaskInfo -TaskName "ArcVault Agent"

# Check agent process
Get-Process -Name "ArcVaultAgent" -ErrorAction SilentlyContinue

# Test HTTP endpoints
Invoke-WebRequest -Uri "http://localhost:8443/health"
Invoke-WebRequest -Uri "http://localhost:8443/info"
```

## 🌐 API Endpoints

The agent exposes these REST endpoints:

### Health Check
```http
GET /health
```
Returns: `{"status": "ok", "version": "0.1.0"}`

### Agent Information
```http
GET /info
```
Returns agent details including public URL and job statistics.

### Create Backup Job
```http
POST /jobs
Content-Type: application/json
X-API-Key: your-token

{
  "sources": ["C:\\Data", "C:\\Documents"],
  "destination": "D:\\Backup",
  "options": {
    "mirror": true,
    "copy_subdirs": true
  }
}
```

### Get Job Status
```http
GET /jobs/{job_id}
X-API-Key: your-token
```

### Stream Job Logs
```http
GET /jobs/{job_id}/logs
X-API-Key: your-token
```
Returns Server-Sent Events stream of live logs.

### List All Jobs
```http
GET /jobs
X-API-Key: your-token
```

### Shutdown Agent
```http
POST /shutdown
X-API-Key: your-token
```

## 🔍 Troubleshooting

### Common Issues

**Agent won't start:**
```powershell
# Check scheduled task status
Get-ScheduledTask -TaskName "ArcVault Agent"

# View startup logs
Get-Content "C:\ProgramData\ArcVault\agent.log"

# Test manually
& "C:\ProgramData\ArcVault\ArcVaultAgent-Startup.ps1"

# Check for port conflicts
netstat -ano | findstr :8443
```

**Connection refused:**
```powershell
# Check Windows Firewall
New-NetFirewallRule -DisplayName "ArcVault Agent" -Direction Inbound -Protocol TCP -LocalPort 8443 -Action Allow

# Test local connectivity
Invoke-WebRequest -Uri "http://localhost:8443/health"

# Check binding
netstat -an | findstr :8443
```

**Authentication errors:**
```powershell
# Verify token matches controller
$env:AGENT_TOKEN = "correct-token"

# For JWT issues, check key paths
Test-Path $env:AGENT_JWT_PUBLIC_KEY_PATH
```

**Permission issues:**
```powershell
# Run PowerShell as Administrator
# Check executable permissions
Get-Acl "C:\ProgramData\ArcVault\ArcVaultAgent.exe"

# Verify scheduled task runs as SYSTEM
Get-ScheduledTask -TaskName "ArcVault Agent" | Select-Object Principal
```

### Log Analysis

**Log locations:**
- Agent logs: `C:\ProgramData\ArcVault\agent.log`
- Windows Event Log: Application > "ArcVault Agent"
- Task Scheduler logs: Task Scheduler Library > ArcVault Agent

**Common log patterns:**
```powershell
# Search for errors
Select-String -Path "C:\ProgramData\ArcVault\agent.log" -Pattern "ERROR"

# Check startup sequence
Get-Content "C:\ProgramData\ArcVault\agent.log" | Select-Object -First 20

# Monitor in real-time
Get-Content "C:\ProgramData\ArcVault\agent.log" -Wait -Tail 10
```

### Network Diagnostics

```powershell
# Test local binding
Test-NetConnection -ComputerName localhost -Port 8443

# Test from remote machine
Test-NetConnection -ComputerName your-agent-ip -Port 8443

# Check firewall rules
Get-NetFirewallRule -DisplayName "*ArcVault*"

# Verify public URL detection
Invoke-WebRequest -Uri "http://localhost:8443/info" | ConvertFrom-Json | Select-Object public_url
```

## 🔒 Security Considerations

### Production Deployment

1. **Use strong authentication:**
   ```powershell
   # Generate secure token
   $token = [System.Web.Security.Membership]::GeneratePassword(32, 8)
   $env:AGENT_TOKEN = $token
   ```

2. **Enable HTTPS:**
   ```powershell
   # Use SSL certificates
   $env:AGENT_CERTFILE = "C:\certs\agent.crt"
   $env:AGENT_KEYFILE = "C:\certs\agent.key"
   ```

3. **Restrict network access:**
   ```powershell
   # Bind to specific interface
   $env:AGENT_BIND_HOST = "192.168.1.100"
   
   # Configure firewall rules
   New-NetFirewallRule -DisplayName "ArcVault Agent" -Direction Inbound -Protocol TCP -LocalPort 8443 -RemoteAddress "192.168.1.0/24" -Action Allow
   ```

4. **Use JWT authentication:**
   ```powershell
   $env:AGENT_REQUIRE_JWT = "true"
   $env:AGENT_JWT_PUBLIC_KEY_PATH = "C:\keys\controller_public.pem"
   ```

### File System Permissions

The agent runs with SYSTEM privileges and can access:
- All local drives and network shares
- System files and hidden directories
- Files locked by other processes

**Security implications:**
- Secure the agent token/keys
- Restrict network access to the agent port
- Monitor backup job requests
- Review log files for sensitive information

## 📊 Monitoring and Maintenance

### Health Monitoring

```powershell
# Create monitoring script
$healthCheck = @"
try {
    `$response = Invoke-WebRequest -Uri "http://localhost:8443/health" -TimeoutSec 5
    if (`$response.StatusCode -eq 200) {
        Write-Host "Agent is healthy"
        exit 0
    }
} catch {
    Write-Host "Agent health check failed: `$_"
    exit 1
}
"@

# Save and schedule health checks
Set-Content -Path "C:\Scripts\agent-health-check.ps1" -Value $healthCheck
```

### Log Rotation

```powershell
# Create log rotation script
$logRotation = @"
`$logFile = "C:\ProgramData\ArcVault\agent.log"
`$maxSize = 10MB

if ((Get-Item `$logFile -ErrorAction SilentlyContinue).Length -gt `$maxSize) {
    Move-Item `$logFile "`$logFile.old" -Force
    New-Item `$logFile -ItemType File
}
"@

# Schedule daily log rotation
$action = New-ScheduledTaskAction -Execute "powershell.exe" -Argument "-Command & { $logRotation }"
$trigger = New-ScheduledTaskTrigger -Daily -At "02:00"
Register-ScheduledTask -TaskName "ArcVault Log Rotation" -Action $action -Trigger $trigger
```

### Performance Monitoring

```powershell
# Monitor agent performance
Get-Process -Name "ArcVaultAgent" | Select-Object CPU, WorkingSet, VirtualMemorySize

# Check job statistics
Invoke-WebRequest -Uri "http://localhost:8443/info" | ConvertFrom-Json | Select-Object job_counts
```

## 🔄 Updates and Maintenance

### Updating the Agent

```powershell
# Stop current agent
.\Manage-ArcVaultAgent.ps1 -Action Stop

# Replace executable
Copy-Item "new-ArcVaultAgent.exe" "C:\ProgramData\ArcVault\ArcVaultAgent.exe" -Force

# Start agent
.\Manage-ArcVaultAgent.ps1 -Action Start

# Verify update
.\Manage-ArcVaultAgent.ps1 -Action Test
```

### Backup Configuration

```powershell
# Export scheduled task
Export-ScheduledTask -TaskName "ArcVault Agent" | Out-File "agent-task-backup.xml"

# Backup configuration
Copy-Item -Path "C:\ProgramData\ArcVault" -Destination "backup-location" -Recurse
```

### Complete Removal

```powershell
# Use the installer's uninstall function
.\Install-ArcVaultAgent.ps1 -Uninstall

# Or manual cleanup
Stop-Process -Name "ArcVaultAgent" -Force -ErrorAction SilentlyContinue
Unregister-ScheduledTask -TaskName "ArcVault Agent" -Confirm:$false -ErrorAction SilentlyContinue
Remove-Item -Path "C:\ProgramData\ArcVault" -Recurse -Force -ErrorAction SilentlyContinue
```

## 🤝 Integration with Controller

After installing the agent, register it with your controller:

1. **Note the agent URL** displayed during installation
2. **Add to controller** via web interface or API
3. **Test connection** by creating a backup job

Example controller registration:
```
Agent Name: Windows-Server-01
Agent URL: http://192.168.1.100:8443
Token: your-secure-token
```

## 📝 Development

### Building from Source

```powershell
# Setup development environment
git clone <repository>
cd ArcVault/apps/backup-agent
python -m venv .venv
.\.venv\Scripts\Activate.ps1
pip install -e .

# Run tests
python -m pytest tests/

# Build executable
pip install pyinstaller
pyinstaller --onefile --name ArcVaultAgent agent/run_agent.py
```

### API Testing

```powershell
# Test with curl (if available)
curl -X POST http://localhost:8443/jobs -H "X-API-Key: dev-token" -H "Content-Type: application/json" -d '{\"sources\":[\"C:/temp\"],\"destination\":\"D:/backup\"}'

# Test with PowerShell
$headers = @{"X-API-Key" = "dev-token"; "Content-Type" = "application/json"}
$body = @{sources = @("C:\temp"); destination = "D:\backup"} | ConvertTo-Json
Invoke-RestMethod -Uri "http://localhost:8443/jobs" -Method POST -Headers $headers -Body $body
```

---

## 📞 Support

For issues and questions:
1. Check the troubleshooting section above
2. Review log files for error details
3. Test with the management helper scripts
4. Verify network connectivity and authentication

**Quick diagnostics:**
```powershell
.\Manage-ArcVaultAgent.ps1 -Action Status
.\Manage-ArcVaultAgent.ps1 -Action Test
.\Manage-ArcVaultAgent.ps1 -Action Logs -LogLines 50
```

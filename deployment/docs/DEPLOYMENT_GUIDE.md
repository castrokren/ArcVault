# 📖 ArcVault Deployment Guide

Comprehensive guide for deploying ArcVault in production environments.

## Table of Contents

1. [System Requirements](#system-requirements)
2. [Pre-Deployment Planning](#pre-deployment-planning)
3. [Installation Methods](#installation-methods)
4. [Post-Installation Configuration](#post-installation-configuration)
5. [Service Management](#service-management)
6. [Security Hardening](#security-hardening)
7. [Monitoring and Maintenance](#monitoring-and-maintenance)

---

## System Requirements

### Hardware Requirements

**Minimum:**
- **CPU**: 2 cores
- **RAM**: 4 GB
- **Disk**: 10 GB free space
- **Network**: 100 Mbps

**Recommended:**
- **CPU**: 4+ cores
- **RAM**: 8+ GB
- **Disk**: 50+ GB free space (depends on backup volume)
- **Network**: 1 Gbps

### Software Requirements

- **OS**: Windows 10/11 or Windows Server 2016+
- **PowerShell**: 5.1 or later
- **Python**: 3.9 or later (can be installed automatically)
- **.NET Framework**: 4.7.2+ (for Windows service)

### Network Requirements

- **Inbound Port**: 8443 (default, configurable)
- **Firewall**: Allow incoming connections on agent port
- **DNS**: Hostname resolution for SSL certificates
- **TLS**: Support for TLS 1.2+

---

## Pre-Deployment Planning

### 1. Architecture Planning

**Single Server Deployment:**
```
┌─────────────────────┐
│   ArcVault Agent    │
│   (Backup Server)   │
│                     │
│  - Agent Service    │
│  - Backup Jobs      │
│  - Local Storage    │
└─────────────────────┘
```

**Multi-Agent Deployment:**
```
┌──────────────┐     ┌──────────────┐     ┌──────────────┐
│  Agent #1    │     │  Agent #2    │     │  Agent #3    │
│  (Client 1)  │     │  (Client 2)  │     │  (Client 3)  │
└──────┬───────┘     └──────┬───────┘     └──────┬───────┘
       │                    │                    │
       └────────────────────┼────────────────────┘
                            │
                   ┌────────┴────────┐
                   │   Controller    │
                   │   (Management)  │
                   └─────────────────┘
```

### 2. Installation Path Planning

**Default Paths:**
- **Installation**: `C:\Program Files\ArcVault`
- **Data**: `C:\ProgramData\ArcVault`
- **Logs**: `C:\ProgramData\ArcVault\logs`
- **Backups**: Configurable destination

**Custom Paths Considerations:**
- Use fast storage (SSD) for application files
- Use large storage for backup destinations
- Ensure adequate permissions on all paths
- Consider backup and disaster recovery

### 3. Network Planning

**Port Configuration:**
- Default port: `8443`
- Choose unused port (check with `netstat -ano | findstr :8443`)
- Configure firewall rules
- Document port assignments

**SSL/TLS Planning:**
- Use self-signed certificates for internal networks
- Use trusted certificates for external access
- Plan certificate renewal process
- Configure certificate validation

### 4. Security Planning

**Authentication:**
- JWT-based authentication (recommended)
- Token-based authentication (simpler)
- API key management
- User access control

**Encryption:**
- TLS/SSL for transport encryption
- Optional backup data encryption
- Key management and rotation
- Secure key storage

---

## Installation Methods

### Method 1: Standard Installation (Recommended)

**Step 1: Download Deployment Package**
```powershell
# Extract deployment package to a folder
cd C:\Temp\ArcVault-Deployment
```

**Step 2: Review Configuration**
```powershell
# Edit default configurations (optional)
notepad config\default_service_config.json
notepad config\default_backup_config.json
```

**Step 3: Run Deployment Script**
```powershell
# Open PowerShell as Administrator
.\Deploy-ArcVault.ps1 -Verbose
```

**Step 4: Verify Installation**
```powershell
.\Verify-ArcVault.ps1 -Detailed
```

### Method 2: Custom Installation Path

```powershell
# Install to custom location
.\Deploy-ArcVault.ps1 `
    -InstallPath "D:\Applications\ArcVault" `
    -DataPath "E:\Data\ArcVault" `
    -Port 9443 `
    -Verbose
```

### Method 3: Development Mode

```powershell
# Install without Windows service (for testing)
.\Deploy-ArcVault.ps1 -Development -Verbose
```

### Method 4: Silent Installation

```powershell
# Unattended installation
.\Deploy-ArcVault.ps1 `
    -Silent `
    -LogPath "C:\Logs\arcvault_install.log"
```

---

## Post-Installation Configuration

### 1. Service Configuration

**Edit Service Config:**
```powershell
notepad "C:\ProgramData\ArcVault\service_config.json"
```

**Key Settings:**
```json
{
  "host": "0.0.0.0",
  "port": 8443,
  "log_level": "info",
  "require_jwt": true,
  "jwt_public_key_path": "C:\\ProgramData\\ArcVault\\keys\\jwt_public.pem",
  "jwt_issuer": "arcvault-controller",
  "ssl_enabled": true
}
```

**Apply Changes:**
```powershell
Restart-Service ArcVaultAgent
```

### 2. Backup Job Configuration

**Edit Backup Config:**
```powershell
notepad "C:\ProgramData\ArcVault\backup_config.json"
```

**Example Backup Job:**
```json
{
  "name": "Daily Documents Backup",
  "enabled": true,
  "sources": [
    "C:\\Users\\*\\Documents"
  ],
  "destination": "\\\\nas\\backups\\documents",
  "schedule": {
    "type": "daily",
    "time": "02:00"
  },
  "options": {
    "compression": true,
    "encryption": true,
    "verify": true,
    "retain_days": 30
  }
}
```

### 3. SSL Certificate Configuration

**Using Self-Signed Certificates (Internal):**
```powershell
# Certificates are generated automatically during installation
# Located at: C:\ProgramData\ArcVault\keys\
```

**Using Trusted Certificates:**
```powershell
# Replace auto-generated certificates
Copy-Item "C:\Certs\mycert.crt" "C:\ProgramData\ArcVault\keys\server.crt"
Copy-Item "C:\Certs\mykey.key" "C:\ProgramData\ArcVault\keys\server.key"

# Restart service
Restart-Service ArcVaultAgent
```

### 4. Firewall Configuration

**Allow Inbound Port:**
```powershell
# Create firewall rule
New-NetFirewallRule `
    -DisplayName "ArcVault Agent" `
    -Direction Inbound `
    -Protocol TCP `
    -LocalPort 8443 `
    -Action Allow `
    -Profile Domain,Private

# Verify rule
Get-NetFirewallRule -DisplayName "ArcVault Agent"
```

---

## Service Management

### Starting and Stopping

```powershell
# Start service
Start-Service ArcVaultAgent

# Stop service
Stop-Service ArcVaultAgent

# Restart service
Restart-Service ArcVaultAgent

# Check status
Get-Service ArcVaultAgent
```

### Service Properties

```powershell
# View service details
Get-Service ArcVaultAgent | Format-List *

# Change startup type
Set-Service ArcVaultAgent -StartupType Automatic

# View service account
Get-WmiObject Win32_Service -Filter "Name='ArcVaultAgent'" | 
    Select-Object Name, StartName
```

### Service Logs

```powershell
# View service log
Get-Content "$env:TEMP\arcvault_service.log" -Tail 50

# Monitor log in real-time
Get-Content "$env:TEMP\arcvault_service.log" -Wait -Tail 20

# View application logs
Get-Content "C:\ProgramData\ArcVault\logs\agent.log" -Tail 50
```

---

## Security Hardening

### 1. Service Account Configuration

**Run as Dedicated Service Account:**
```powershell
# Create service account
$password = ConvertTo-SecureString "YourSecurePassword" -AsPlainText -Force
New-LocalUser -Name "ArcVaultService" `
    -Password $password `
    -PasswordNeverExpires `
    -UserMayNotChangePassword

# Grant necessary permissions
icacls "C:\Program Files\ArcVault" /grant "ArcVaultService:(OI)(CI)RX"
icacls "C:\ProgramData\ArcVault" /grant "ArcVaultService:(OI)(CI)F"

# Change service account
$credential = New-Object System.Management.Automation.PSCredential `
    (".\ArcVaultService", $password)
$service = Get-WmiObject Win32_Service -Filter "Name='ArcVaultAgent'"
$service.Change($null, $null, $null, $null, $null, $null, 
    $credential.UserName, $credential.GetNetworkCredential().Password)

# Restart service
Restart-Service ArcVaultAgent
```

### 2. File System Permissions

**Restrict Installation Directory:**
```powershell
# Remove inherited permissions
icacls "C:\Program Files\ArcVault" /inheritance:r

# Grant access to Administrators and SYSTEM
icacls "C:\Program Files\ArcVault" /grant "BUILTIN\Administrators:(OI)(CI)F"
icacls "C:\Program Files\ArcVault" /grant "NT AUTHORITY\SYSTEM:(OI)(CI)F"
```

**Restrict Data Directory:**
```powershell
# Secure data directory
icacls "C:\ProgramData\ArcVault" /inheritance:r
icacls "C:\ProgramData\ArcVault" /grant "BUILTIN\Administrators:(OI)(CI)F"
icacls "C:\ProgramData\ArcVault" /grant "NT AUTHORITY\SYSTEM:(OI)(CI)F"

# Extra protection for keys directory
icacls "C:\ProgramData\ArcVault\keys" /inheritance:r
icacls "C:\ProgramData\ArcVault\keys" /grant "BUILTIN\Administrators:(OI)(CI)F"
```

### 3. Network Security

**IP Whitelisting (if needed):**
```powershell
# Allow only specific IPs
New-NetFirewallRule `
    -DisplayName "ArcVault Agent - Restricted" `
    -Direction Inbound `
    -Protocol TCP `
    -LocalPort 8443 `
    -Action Allow `
    -RemoteAddress "192.168.1.0/24"
```

### 4. JWT Key Rotation

**Rotate JWT Keys:**
```powershell
# Backup old keys
Copy-Item "C:\ProgramData\ArcVault\keys\jwt_*.pem" "C:\Backup\JWT_Keys_$(Get-Date -Format 'yyyyMMdd')"

# Generate new keys
cd "C:\Program Files\ArcVault"
.\venv\Scripts\python.exe arcvault_unified.py --gen-jwt --output-dir "C:\ProgramData\ArcVault\keys"

# Restart service
Restart-Service ArcVaultAgent
```

---

## Monitoring and Maintenance

### 1. Health Checks

**Basic Health Check:**
```powershell
# Check service status
$service = Get-Service ArcVaultAgent
if ($service.Status -ne "Running") {
    Write-Warning "Service is not running!"
}

# Check port connectivity
Test-NetConnection -ComputerName localhost -Port 8443
```

**Automated Health Check Script:**
```powershell
# Save as Check-ArcVaultHealth.ps1
$issues = @()

# Check service
if ((Get-Service ArcVaultAgent).Status -ne "Running") {
    $issues += "Service not running"
}

# Check disk space
$drive = (Get-Item "C:\ProgramData\ArcVault").PSDrive
$freeGB = [math]::Round($drive.Free / 1GB, 2)
if ($freeGB -lt 5) {
    $issues += "Low disk space: $freeGB GB"
}

# Check log file size
$logFile = Get-Item "$env:TEMP\arcvault_service.log" -ErrorAction SilentlyContinue
if ($logFile -and ($logFile.Length / 1MB) -gt 100) {
    $issues += "Log file too large: $([math]::Round($logFile.Length / 1MB, 2)) MB"
}

if ($issues.Count -eq 0) {
    Write-Host "✓ All checks passed" -ForegroundColor Green
} else {
    Write-Host "⚠ Issues found:" -ForegroundColor Yellow
    $issues | ForEach-Object { Write-Host "  - $_" -ForegroundColor Yellow }
}
```

### 2. Log Management

**Rotate Service Log:**
```powershell
# Archive old log
$logPath = "$env:TEMP\arcvault_service.log"
if (Test-Path $logPath) {
    $archivePath = "$env:TEMP\arcvault_service_$(Get-Date -Format 'yyyyMMdd').log"
    Move-Item $logPath $archivePath -Force
    Restart-Service ArcVaultAgent
}
```

**Clean Old Backup Logs:**
```powershell
# Remove logs older than 90 days
$logDir = "C:\ProgramData\ArcVault\logs\#backup_logs"
Get-ChildItem $logDir -Recurse -File | 
    Where-Object { $_.LastWriteTime -lt (Get-Date).AddDays(-90) } |
    Remove-Item -Force
```

### 3. Performance Monitoring

**Monitor Resource Usage:**
```powershell
# Get process info
Get-Process | Where-Object { $_.ProcessName -like "*python*" } |
    Select-Object ProcessName, CPU, WorkingSet, Handles
```

### 4. Backup and Recovery

**Backup Configuration:**
```powershell
# Create configuration backup
$timestamp = Get-Date -Format "yyyyMMdd_HHmmss"
$backupPath = "C:\Backups\ArcVault_Config_$timestamp"

# Backup data directory
Copy-Item "C:\ProgramData\ArcVault" $backupPath -Recurse
```

**Test Recovery:**
```powershell
# Test rollback capability
.\Rollback-ArcVault.ps1 -BackupPath "C:\Backups\ArcVault_Backup_20241201"
```

---

## Troubleshooting

For troubleshooting guidance, see [TROUBLESHOOTING.md](TROUBLESHOOTING.md)

## Next Steps

1. ✅ Complete post-installation configuration
2. ✅ Set up backup jobs
3. ✅ Configure monitoring and alerts
4. ✅ Test backup and restore procedures
5. ✅ Document your deployment

---

**Need Help?** Check the [System Requirements](SYSTEM_REQUIREMENTS.md) or [Troubleshooting Guide](TROUBLESHOOTING.md)


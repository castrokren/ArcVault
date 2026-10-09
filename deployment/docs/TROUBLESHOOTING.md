# 🔧 Troubleshooting Guide

Common issues and solutions for ArcVault deployment.

## Table of Contents

1. [Installation Issues](#installation-issues)
2. [Service Issues](#service-issues)
3. [Network Issues](#network-issues)
4. [Configuration Issues](#configuration-issues)
5. [Performance Issues](#performance-issues)
6. [Backup Job Issues](#backup-job-issues)
7. [Diagnostic Tools](#diagnostic-tools)

---

## Installation Issues

### Issue: "Administrator privileges required"

**Symptoms:**
- Installation script fails immediately
- Error message about administrator rights

**Solution:**
```powershell
# Right-click PowerShell and select "Run as Administrator"
# Or from elevated prompt:
Start-Process powershell -Verb RunAs -ArgumentList "-File Deploy-ArcVault.ps1"
```

---

### Issue: "Python not found"

**Symptoms:**
- Installation script cannot find Python
- Error: "Python 3.9+ required"

**Solution:**
```powershell
# Option 1: Install Python manually
# Download from: https://www.python.org/downloads/
# Ensure "Add Python to PATH" is checked

# Option 2: Install via chocolatey
choco install python39 -y

# Verify installation
python --version
refreshenv
```

---

### Issue: "Port already in use"

**Symptoms:**
- Installation completes but service won't start
- Error about port binding

**Solution:**
```powershell
# Find process using port
netstat -ano | findstr :8443

# Kill process (if safe)
Stop-Process -Id <PID> -Force

# Or use different port
.\Deploy-ArcVault.ps1 -Port 9443
```

---

### Issue: "Insufficient disk space"

**Symptoms:**
- Installation fails during file copy
- Error about disk space

**Solution:**
```powershell
# Check available space
Get-PSDrive C | Select-Object Used, Free

# Free up space or use different drive
.\Deploy-ArcVault.ps1 -InstallPath "D:\ArcVault"
```

---

### Issue: "Source files not found"

**Symptoms:**
- Cannot find arcvault_unified.py
- Source validation fails

**Solution:**
```powershell
# Specify correct source path
.\Deploy-ArcVault.ps1 -SourcePath "..\apps\backup-agent"

# Or absolute path
.\Deploy-ArcVault.ps1 -SourcePath "C:\ArcVault\apps\backup-agent"
```

---

## Service Issues

### Issue: Service won't start

**Symptoms:**
- `Get-Service ArcVaultAgent` shows "Stopped"
- Service starts then immediately stops

**Diagnosis:**
```powershell
# Check service log
Get-Content "$env:TEMP\arcvault_service.log" -Tail 50

# Check Windows Event Log
Get-EventLog -LogName Application -Source "ArcVaultAgent" -Newest 10
```

**Common Causes:**

#### 1. Port Already in Use
```powershell
# Check if port is available
Test-NetConnection -ComputerName localhost -Port 8443

# Change port in config
notepad "C:\ProgramData\ArcVault\service_config.json"
# Change "port": 8443 to different port
# Restart service
Restart-Service ArcVaultAgent
```

#### 2. Configuration File Invalid
```powershell
# Validate JSON syntax
Get-Content "C:\ProgramData\ArcVault\service_config.json" | ConvertFrom-Json

# If error, restore default
Copy-Item "C:\Program Files\ArcVault\deployment\config\default_service_config.json" `
    "C:\ProgramData\ArcVault\service_config.json" -Force
```

#### 3. Missing Dependencies
```powershell
# Reinstall dependencies
cd "C:\Program Files\ArcVault"
.\venv\Scripts\pip.exe install -r requirements_unified.txt --force-reinstall
```

#### 4. Python Environment Issues
```powershell
# Test Python environment
cd "C:\Program Files\ArcVault"
.\venv\Scripts\python.exe arcvault_unified.py --help

# If fails, recreate venv
Remove-Item venv -Recurse -Force
python -m venv venv
.\venv\Scripts\pip.exe install -r requirements_unified.txt
```

---

### Issue: Service starts but crashes

**Symptoms:**
- Service starts successfully
- Crashes after a few seconds/minutes

**Diagnosis:**
```powershell
# Watch log in real-time
Get-Content "$env:TEMP\arcvault_service.log" -Wait -Tail 20

# Look for error patterns
Select-String -Path "$env:TEMP\arcvault_service.log" -Pattern "ERROR|Exception|Traceback"
```

**Solutions:**

#### Check for Module Import Errors
```powershell
# Test imports
cd "C:\Program Files\ArcVault"
.\venv\Scripts\python.exe -c "import agent.app; print('OK')"
```

#### Check File Permissions
```powershell
# Verify service can read files
icacls "C:\Program Files\ArcVault"
icacls "C:\ProgramData\ArcVault"

# Fix permissions if needed
icacls "C:\Program Files\ArcVault" /grant "NT AUTHORITY\SYSTEM:(OI)(CI)RX"
icacls "C:\ProgramData\ArcVault" /grant "NT AUTHORITY\SYSTEM:(OI)(CI)F"
```

---

### Issue: Service won't stop

**Symptoms:**
- `Stop-Service` command hangs
- Service shows "Stopping" indefinitely

**Solution:**
```powershell
# Force stop
Stop-Service ArcVaultAgent -Force

# If still stuck, kill process
$svc = Get-WmiObject Win32_Service -Filter "Name='ArcVaultAgent'"
$processId = $svc.ProcessId
Stop-Process -Id $processId -Force

# Restart service
Start-Service ArcVaultAgent
```

---

## Network Issues

### Issue: Cannot connect to agent API

**Symptoms:**
- Port is open but API doesn't respond
- Connection timeout or refused

**Diagnosis:**
```powershell
# Test port connectivity
Test-NetConnection -ComputerName localhost -Port 8443

# Test HTTP endpoint
Invoke-WebRequest -Uri "http://localhost:8443/" -UseBasicParsing
```

**Solutions:**

#### 1. Firewall Blocking
```powershell
# Check firewall rules
Get-NetFirewallRule -DisplayName "*ArcVault*"

# Create rule if missing
New-NetFirewallRule `
    -DisplayName "ArcVault Agent" `
    -Direction Inbound `
    -Protocol TCP `
    -LocalPort 8443 `
    -Action Allow
```

#### 2. Wrong Host Binding
```powershell
# Check service config
notepad "C:\ProgramData\ArcVault\service_config.json"

# Ensure "host": "0.0.0.0" for all interfaces
# Or "host": "127.0.0.1" for localhost only
```

#### 3. SSL/TLS Issues
```powershell
# Test with HTTPS
Invoke-WebRequest -Uri "https://localhost:8443/" -SkipCertificateCheck

# If SSL cert issue, regenerate
cd "C:\Program Files\ArcVault"
.\venv\Scripts\python.exe arcvault_unified.py --gen-cert
Restart-Service ArcVaultAgent
```

---

### Issue: "SSL certificate verification failed"

**Symptoms:**
- Cannot connect over HTTPS
- Certificate errors

**Solution:**
```powershell
# For self-signed certificates (internal use):
# Option 1: Accept in client application

# Option 2: Install certificate to trusted root
$cert = New-Object System.Security.Cryptography.X509Certificates.X509Certificate2
$cert.Import("C:\ProgramData\ArcVault\keys\server.crt")
$store = New-Object System.Security.Cryptography.X509Certificates.X509Store "Root", "LocalMachine"
$store.Open("ReadWrite")
$store.Add($cert)
$store.Close()

# Option 3: Use trusted certificate
# Replace with certificate from trusted CA
Copy-Item "C:\Certs\trusted.crt" "C:\ProgramData\ArcVault\keys\server.crt"
Copy-Item "C:\Certs\trusted.key" "C:\ProgramData\ArcVault\keys\server.key"
Restart-Service ArcVaultAgent
```

---

## Configuration Issues

### Issue: Configuration changes not applied

**Symptoms:**
- Modified config file
- Changes don't take effect

**Solution:**
```powershell
# Restart service to apply changes
Restart-Service ArcVaultAgent

# Verify service picked up changes
Get-Content "$env:TEMP\arcvault_service.log" -Tail 20 | 
    Select-String "Configuration"
```

---

### Issue: "Invalid JSON in configuration"

**Symptoms:**
- Service won't start after config change
- JSON parsing errors in log

**Solution:**
```powershell
# Validate JSON syntax
try {
    Get-Content "C:\ProgramData\ArcVault\service_config.json" | ConvertFrom-Json
    Write-Host "✓ JSON is valid" -ForegroundColor Green
} catch {
    Write-Host "✗ JSON is invalid: $($_.Exception.Message)" -ForegroundColor Red
}

# Use online validator: https://jsonlint.com/

# Restore default if needed
Copy-Item "C:\Program Files\ArcVault\deployment\config\default_service_config.json" `
    "C:\ProgramData\ArcVault\service_config.json" -Force
```

---

### Issue: Environment variables not loaded

**Symptoms:**
- .env file changes ignored
- Service uses wrong settings

**Solution:**
```powershell
# Note: .env is secondary to service_config.json
# Use service_config.json for service configuration

# Or ensure .env is in correct location
Test-Path "C:\ProgramData\ArcVault\.env"

# Check service reads .env
Get-Content "$env:TEMP\arcvault_service.log" | 
    Select-String "environment|config"
```

---

## Performance Issues

### Issue: Slow backup performance

**Symptoms:**
- Backups take longer than expected
- Low throughput

**Diagnosis:**
```powershell
# Check CPU usage
Get-Process | Where-Object { $_.ProcessName -like "*python*" } | 
    Select-Object ProcessName, CPU, WorkingSet

# Check disk I/O
Get-Counter '\PhysicalDisk(_Total)\Disk Reads/sec', '\PhysicalDisk(_Total)\Disk Writes/sec'

# Check network usage
Get-NetAdapterStatistics
```

**Solutions:**

#### 1. Increase Concurrent Workers
```json
// In service_config.json or backup_config.json
{
  "max_workers": 4  // Increase based on CPU cores
}
```

#### 2. Optimize Compression
```json
// In backup job configuration
{
  "compression": false  // Disable if CPU-bound
}
```

#### 3. Check Disk Performance
```powershell
# Test disk speed
winsat disk -drive c
winsat disk -drive d

# Use faster disks for active operations
```

---

### Issue: High memory usage

**Symptoms:**
- Service uses excessive RAM
- System becomes slow

**Diagnosis:**
```powershell
# Check memory usage
Get-Process | Where-Object { $_.ProcessName -like "*python*" } | 
    Select-Object ProcessName, WorkingSet, PM

# Monitor over time
while ($true) {
    Get-Process python* | Select ProcessName, @{N='Memory(MB)';E={$_.WS / 1MB}}
    Start-Sleep -Seconds 5
}
```

**Solutions:**

#### 1. Reduce Concurrent Jobs
```json
{
  "max_concurrent_jobs": 2  // Reduce from default
}
```

#### 2. Restart Service Periodically
```powershell
# Create scheduled task to restart service
$action = New-ScheduledTaskAction -Execute 'PowerShell.exe' `
    -Argument 'Restart-Service ArcVaultAgent'
$trigger = New-ScheduledTaskTrigger -Daily -At "03:00"
Register-ScheduledTask -TaskName "Restart ArcVault" `
    -Action $action -Trigger $trigger
```

---

## Backup Job Issues

### Issue: Backup job fails

**Symptoms:**
- Job starts but fails
- Error in backup logs

**Diagnosis:**
```powershell
# Check backup logs
Get-ChildItem "C:\ProgramData\ArcVault\logs\#backup_logs" -Recurse | 
    Sort-Object LastWriteTime -Descending | 
    Select-Object -First 1 | 
    Get-Content -Tail 50
```

**Common Causes:**

#### 1. Permission Denied
```powershell
# Grant service account access
$serviceAccount = (Get-WmiObject Win32_Service -Filter "Name='ArcVaultAgent'").StartName

# Grant read access to source
icacls "C:\SourcePath" /grant "$serviceAccount:(OI)(CI)RX"

# Grant write access to destination
icacls "D:\BackupPath" /grant "$serviceAccount:(OI)(CI)F"
```

#### 2. Destination Full
```powershell
# Check destination space
$dest = "D:\Backups"
$drive = (Get-Item $dest).PSDrive
$freeGB = [math]::Round($drive.Free / 1GB, 2)
Write-Host "Free space: $freeGB GB"

# Clean old backups
# Implement retention policy
```

#### 3. Network Path Unavailable
```powershell
# Test network path
Test-Path "\\nas\backups"

# Map drive if needed
New-PSDrive -Name "Z" -PSProvider FileSystem -Root "\\nas\backups" -Persist
```

---

## Diagnostic Tools

### Comprehensive Health Check

```powershell
# Save as Check-ArcVaultHealth.ps1

Write-Host "`n=== ArcVault Health Check ===" -ForegroundColor Cyan

# Service Status
Write-Host "`n[Service Status]" -ForegroundColor Yellow
$service = Get-Service ArcVaultAgent -ErrorAction SilentlyContinue
if ($service) {
    Write-Host "  Status: $($service.Status)"
    Write-Host "  StartType: $($service.StartType)"
} else {
    Write-Host "  Service not installed!" -ForegroundColor Red
}

# Port Check
Write-Host "`n[Network]" -ForegroundColor Yellow
$config = Get-Content "C:\ProgramData\ArcVault\service_config.json" | ConvertFrom-Json
$port = $config.port
Write-Host "  Port: $port"
$portTest = Test-NetConnection -ComputerName localhost -Port $port -WarningAction SilentlyContinue
Write-Host "  Listening: $($portTest.TcpTestSucceeded)"

# Log Files
Write-Host "`n[Logs]" -ForegroundColor Yellow
$logFile = "$env:TEMP\arcvault_service.log"
if (Test-Path $logFile) {
    $log = Get-Item $logFile
    Write-Host "  Service Log: $($log.Length / 1MB) MB"
    Write-Host "  Last Write: $($log.LastWriteTime)"
    
    # Check for recent errors
    $errors = Get-Content $logFile -Tail 100 | 
        Where-Object { $_ -match "ERROR|Exception" }
    if ($errors) {
        Write-Host "  Recent Errors: $($errors.Count)" -ForegroundColor Red
    } else {
        Write-Host "  Recent Errors: 0" -ForegroundColor Green
    }
} else {
    Write-Host "  Log file not found" -ForegroundColor Red
}

# Disk Space
Write-Host "`n[Disk Space]" -ForegroundColor Yellow
$drives = @("C", "D")
foreach ($driveLetter in $drives) {
    $drive = Get-PSDrive $driveLetter -ErrorAction SilentlyContinue
    if ($drive) {
        $freeGB = [math]::Round($drive.Free / 1GB, 2)
        $usedPercent = [math]::Round(($drive.Used / ($drive.Used + $drive.Free)) * 100, 2)
        Write-Host "  ${driveLetter}: $freeGB GB free ($usedPercent% used)"
    }
}

# Configuration
Write-Host "`n[Configuration]" -ForegroundColor Yellow
if (Test-Path "C:\ProgramData\ArcVault\service_config.json") {
    Write-Host "  service_config.json: ✓" -ForegroundColor Green
} else {
    Write-Host "  service_config.json: ✗" -ForegroundColor Red
}

if (Test-Path "C:\ProgramData\ArcVault\keys\jwt_public.pem") {
    Write-Host "  JWT keys: ✓" -ForegroundColor Green
} else {
    Write-Host "  JWT keys: ✗" -ForegroundColor Red
}

if (Test-Path "C:\ProgramData\ArcVault\keys\server.crt") {
    Write-Host "  SSL cert: ✓" -ForegroundColor Green
} else {
    Write-Host "  SSL cert: ✗" -ForegroundColor Red
}

Write-Host "`n=== End Health Check ===" -ForegroundColor Cyan
```

---

### Log Analysis Tool

```powershell
# Save as Analyze-Logs.ps1

param([string]$LogPath = "$env:TEMP\arcvault_service.log")

if (-not (Test-Path $LogPath)) {
    Write-Host "Log file not found: $LogPath" -ForegroundColor Red
    exit
}

Write-Host "`n=== Log Analysis ===" -ForegroundColor Cyan

# Read log
$logs = Get-Content $LogPath

# Statistics
Write-Host "`n[Statistics]" -ForegroundColor Yellow
Write-Host "  Total Lines: $($logs.Count)"

$errors = $logs | Where-Object { $_ -match "\[ERROR\]|ERROR:" }
Write-Host "  Errors: $($errors.Count)" -ForegroundColor $(if ($errors.Count -gt 0) { "Red" } else { "Green" })

$warnings = $logs | Where-Object { $_ -match "\[WARNING\]|WARNING:" }
Write-Host "  Warnings: $($warnings.Count)" -ForegroundColor $(if ($warnings.Count -gt 0) { "Yellow" } else { "Green" })

# Recent errors
if ($errors.Count -gt 0) {
    Write-Host "`n[Recent Errors]" -ForegroundColor Red
    $errors | Select-Object -Last 10 | ForEach-Object {
        Write-Host "  $_" -ForegroundColor Gray
    }
}

# Service starts/stops
$starts = $logs | Where-Object { $_ -match "service starting|SERVICE_RUNNING" }
$stops = $logs | Where-Object { $_ -match "service stopped|SERVICE_STOPPED" }
Write-Host "`n[Service Activity]" -ForegroundColor Yellow
Write-Host "  Starts: $($starts.Count)"
Write-Host "  Stops: $($stops.Count)"

Write-Host "`n=== End Analysis ===" -ForegroundColor Cyan
```

---

## Getting More Help

### Collect Diagnostic Information

```powershell
# Create diagnostic bundle
$timestamp = Get-Date -Format "yyyyMMdd_HHmmss"
$bundlePath = "C:\Temp\ArcVault_Diagnostics_$timestamp"
New-Item -Path $bundlePath -ItemType Directory -Force

# Copy logs
Copy-Item "$env:TEMP\arcvault_service.log" $bundlePath -ErrorAction SilentlyContinue
Copy-Item "C:\ProgramData\ArcVault\logs\*" $bundlePath -Recurse -ErrorAction SilentlyContinue

# Export service info
Get-Service ArcVaultAgent | Out-File "$bundlePath\service_info.txt"

# Export configuration (sanitized)
Get-Content "C:\ProgramData\ArcVault\service_config.json" | Out-File "$bundlePath\service_config.txt"

# System information
Get-ComputerInfo | Out-File "$bundlePath\system_info.txt"

# Compress bundle
Compress-Archive -Path $bundlePath -DestinationPath "$bundlePath.zip"

Write-Host "Diagnostic bundle created: $bundlePath.zip"
```

### Enable Debug Logging

```json
// In service_config.json
{
  "log_level": "debug"  // Change from "info" to "debug"
}
```

```powershell
# Restart service to apply
Restart-Service ArcVaultAgent
```

---

**Still need help?** Check the [DEPLOYMENT_GUIDE.md](DEPLOYMENT_GUIDE.md) or create an issue on GitHub.


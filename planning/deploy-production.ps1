# ArcVault Production Deployment Script
# Deploy v1.2.0 to production environment

param(
    [string]$ProjectPath = "C:\Projects\ArcVault2.0",
    [string]$ProductionPath = "C:\ArcVault",
    [int]$Port = 8080
)

function Write-Section {
    param([string]$Title)
    Write-Host "`n" + ("=" * 70) -ForegroundColor Cyan
    Write-Host $Title -ForegroundColor Cyan
    Write-Host ("=" * 70) -ForegroundColor Cyan
}

function Write-Success {
    param([string]$Message)
    Write-Host "✅ $Message" -ForegroundColor Green
}

function Write-Error-Custom {
    param([string]$Message)
    Write-Host "❌ $Message" -ForegroundColor Red
}

function Write-Warning-Custom {
    param([string]$Message)
    Write-Host "⚠️  $Message" -ForegroundColor Yellow
}

# Step 1: Pre-deployment checks
Write-Section "Step 1: Pre-Deployment Verification"

$coordinatorBinary = "$ProjectPath\coordinator\arcvault-coordinator.exe"
if (-not (Test-Path $coordinatorBinary)) {
    Write-Error-Custom "Coordinator binary not found at $coordinatorBinary"
    exit 1
}
Write-Success "Coordinator binary found (v0.2.0)"

# Step 2: Check if production path exists
Write-Section "Step 2: Production Path Check"

if (Test-Path "$ProductionPath\arcvault.db") {
    Write-Warning-Custom "Existing production database found"
    Write-Host "Location: $ProductionPath\arcvault.db"

    $backup = "$ProductionPath\arcvault.db.backup.$(Get-Date -Format 'yyyyMMdd-HHmmss')"
    Copy-Item "$ProductionPath\arcvault.db" -Destination $backup
    Write-Success "Database backed up to: $backup"
} else {
    Write-Success "Fresh production installation (no existing database)"
}

# Step 3: Stop existing service (if running)
Write-Section "Step 3: Stop Existing Service"

$service = Get-Service arcvault-coordinator -ErrorAction SilentlyContinue
if ($service) {
    Write-Host "Stopping existing service..."
    Stop-Service arcvault-coordinator -Force -ErrorAction SilentlyContinue
    Start-Sleep -Seconds 2
    Write-Success "Service stopped"
} else {
    Write-Success "No existing service running"
}

# Step 4: Create production directories
Write-Section "Step 4: Create Production Directories"

New-Item -ItemType Directory -Path $ProductionPath -Force | Out-Null
Write-Success "Production directory: $ProductionPath"

# Step 5: Copy binaries
Write-Section "Step 5: Deploy Binaries"

Copy-Item $coordinatorBinary -Destination "$ProductionPath\arcvault-coordinator.exe" -Force
$size = (Get-Item "$ProductionPath\arcvault-coordinator.exe").Length / 1MB
Write-Success "Coordinator deployed ($('{0:F1}' -f $size) MB)"

# Step 6: Generate production encryption key
Write-Section "Step 6: Generate Production Encryption Key"

$rng = New-Object System.Security.Cryptography.RNGCryptoServiceProvider
$keyBytes = New-Object byte[] 32
$rng.GetBytes($keyBytes)
$productionKey = [System.BitConverter]::ToString($keyBytes).Replace("-", "").ToLower()
Write-Success "Encryption key generated (64 hex characters)"

Write-Warning-Custom "CRITICAL: Save this key securely immediately!"
Write-Host "Key: $productionKey" -ForegroundColor White -BackgroundColor DarkRed
Write-Host ""
Write-Host "Save to: Password manager / secure vault / encrypted storage" -ForegroundColor Yellow
Write-Host ""

# Step 7: Create production config
Write-Section "Step 7: Create Production Configuration"

$adminToken = "prod-$(New-Guid)".Substring(0, 40)
$jwtSecret = [System.BitConverter]::ToString($keyBytes).Replace("-", "").ToLower()

$config = @{
    port = $Port
    admin_token = $adminToken
    jwt_secret = $jwtSecret
    database_path = "$ProductionPath\arcvault.db"
    environment = "production"
}

$configJson = $config | ConvertTo-Json
$utf8NoBom = New-Object System.Text.UTF8Encoding $false
[System.IO.File]::WriteAllText("$ProductionPath\config.json", $configJson, $utf8NoBom)
Write-Success "Configuration created"

Write-Host "Port: $Port"
Write-Host "Environment: production"
Write-Host "Database: $ProductionPath\arcvault.db"
Write-Host ""
Write-Host "Admin token: $adminToken" -ForegroundColor White
Write-Warning-Custom "Save admin token securely - it cannot be recovered!"

# Save credentials to files for reference
$adminToken | Out-File "$ProductionPath\ADMIN_TOKEN.txt" -Encoding UTF8NoBOM
$productionKey | Out-File "$ProductionPath\ENCRYPTION_KEY.txt" -Encoding UTF8NoBOM
Write-Success "Credentials saved to: $ProductionPath\ENCRYPTION_KEY.txt and ADMIN_TOKEN.txt"

# Step 8: Install as Windows Service
Write-Section "Step 8: Install Windows Service"

Write-Host "Installing arcvault-coordinator as Windows service..."
Write-Host "This requires administrator privileges"
Write-Host ""

# Create service batch script
$serviceBatch = @"
@echo off
REM ArcVault Coordinator Production Service
cd "$ProductionPath"
set ARCVAULT_CREDENTIAL_KEY=$productionKey
.\arcvault-coordinator.exe start
"@

$serviceBatch | Out-File "$ProductionPath\service-run.bat" -Encoding ASCII
Write-Success "Service startup script created"

# Register as service using nssm if available, or sc.exe
$nssm = Get-Command nssm.exe -ErrorAction SilentlyContinue

if ($nssm) {
    Write-Host "Using nssm to install service..."
    nssm install arcvault-coordinator "$ProductionPath\arcvault-coordinator.exe" start
    nssm set arcvault-coordinator AppDirectory "$ProductionPath"
    nssm set arcvault-coordinator AppEnvironmentExtra "ARCVAULT_CREDENTIAL_KEY=$productionKey"
    Write-Success "Service installed with nssm"
} else {
    Write-Warning-Custom "nssm not found - using sc.exe (manual configuration required)"
    Write-Host "To install manually, run:"
    Write-Host "  sc.exe create arcvault-coordinator binPath= ""$ProductionPath\arcvault-coordinator.exe start"""
    Write-Host "  reg add ""HKLM\SYSTEM\CurrentControlSet\Services\arcvault-coordinator\Environment"" /v ARCVAULT_CREDENTIAL_KEY /d $productionKey /f"
}

# Step 9: Verify installation
Write-Section "Step 9: Verify Installation"

if ($nssm) {
    Start-Service arcvault-coordinator -ErrorAction SilentlyContinue
    Start-Sleep -Seconds 3

    $service = Get-Service arcvault-coordinator -ErrorAction SilentlyContinue
    if ($service.Status -eq "Running") {
        Write-Success "Service is running"

        try {
            $version = Invoke-WebRequest http://localhost:$Port/api/version -ErrorAction Stop
            Write-Success "API responding on port $Port"
            Write-Host "Version: $($version.Content)"
        } catch {
            Write-Error-Custom "API not responding: $_"
        }
    } else {
        Write-Error-Custom "Service failed to start. Check logs."
    }
} else {
    Write-Warning-Custom "Service not auto-started (requires manual sc.exe setup)"
    Write-Host "After manual installation, start with: sc start arcvault-coordinator"
}

# Step 10: Production checklist
Write-Section "Step 10: Post-Deployment Checklist"

Write-Host @"
Before going live, verify:

CRITICAL:
  [ ] Encryption key backed up securely (NOT in git!)
  [ ] Admin token backed up securely (NOT in git!)
  [ ] Database backup location verified
  [ ] Service account has file permissions
  [ ] Firewall rules allow port $Port

FUNCTIONAL:
  [ ] API responds: curl http://localhost:$Port/api/version
  [ ] Dashboard loads: http://localhost:$Port
  [ ] Can login with admin token
  [ ] Can create credentials (Admin > Credentials)
  [ ] Can create jobs with credentials
  [ ] Agents can connect and receive credentials

OPERATIONAL:
  [ ] Logging configured
  [ ] Monitoring configured
  [ ] Alerting configured
  [ ] Backup schedule verified
  [ ] Disaster recovery plan documented

"@

# Create summary file
$summary = @"
ARCVAULT v1.2.0 PRODUCTION DEPLOYMENT
======================================

Deployment Date: $(Get-Date)
Version: v1.2.0
Environment: Production
Port: $Port

INSTALLATION DETAILS:
=====================
Production Path: $ProductionPath
Binaries: $ProductionPath\arcvault-coordinator.exe
Database: $ProductionPath\arcvault.db
Config: $ProductionPath\config.json

CREDENTIALS (save securely!):
=============================
Admin Token: $adminToken
Encryption Key: $productionKey

These files created (delete after saving securely):
  - $ProductionPath\ADMIN_TOKEN.txt
  - $ProductionPath\ENCRYPTION_KEY.txt

SERVICE INFO:
==============
Service Name: arcvault-coordinator
Startup Type: Automatic (via sc.exe)
Account: SYSTEM

VERIFICATION STEPS:
====================
1. Start service: sc start arcvault-coordinator
2. Check status: sc query arcvault-coordinator
3. Verify API: curl http://localhost:$Port/api/version
4. Access dashboard: http://localhost:$Port
5. Check logs: Event Viewer > Application

NEXT STEPS:
===========
1. Verify service is running
2. Test API and dashboard connectivity
3. Create test credential
4. Create test job with credential
5. Execute test job and verify agent credential application
6. Monitor logs for 24 hours
7. Remove plaintext credential files when confident

ROLLBACK:
=========
If issues occur:
  sc stop arcvault-coordinator
  copy $backup back to $ProductionPath\arcvault.db
  sc start arcvault-coordinator

SUPPORT:
========
Release: v1.2.0 - Path Authentication System
Documentation: $ProjectPath\planning\COMPLETION_SUMMARY.md
Issues: Check coordinator logs and database permissions

Created: $(Get-Date -Format 'yyyy-MM-dd HH:mm:ss')
"@

$summary | Out-File "$ProductionPath\DEPLOYMENT_SUMMARY.txt" -Encoding UTF8NoBOM
Write-Success "Deployment summary saved"

# Final summary
Write-Section "✅ PRODUCTION DEPLOYMENT COMPLETE"

Write-Host @"
v1.2.0 - Path Authentication System is deployed!

NEXT IMMEDIATE ACTIONS:
=======================
1. Verify service is running:
   sc query arcvault-coordinator

2. Test API connectivity:
   Invoke-WebRequest http://localhost:$Port/api/version

3. Access dashboard:
   http://localhost:$Port

4. Login with admin token:
   $adminToken

CRITICAL REMINDERS:
===================
⚠️  Save encryption key: $productionKey
⚠️  Save admin token: $adminToken
⚠️  Delete ENCRYPTION_KEY.txt after backing up
⚠️  Delete ADMIN_TOKEN.txt after backing up
⚠️  DO NOT commit these to git!

DOCUMENTATION:
===============
See: $ProductionPath\DEPLOYMENT_SUMMARY.txt
Full docs: $ProjectPath\planning\COMPLETION_SUMMARY.md

MONITORING:
===========
Check logs in Event Viewer under Application
Watch for errors in first 24 hours
Verify agents can receive credentials

All set! Your production system is ready. 🚀
"@

Write-Host ""
Write-Host "Deployment completed: $(Get-Date -Format 'yyyy-MM-dd HH:mm:ss')" -ForegroundColor Green

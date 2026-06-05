# Automated Staging Environment Setup
# Run this once to set up complete staging environment

param(
    [string]$ProjectPath = "C:\Projects\ArcVault2.0",
    [string]$StagingPath = "C:\ArcVault-Staging",
    [int]$Port = 8081
)

function Write-Section {
    param([string]$Title)
    Write-Host "`n" + ("=" * 60) -ForegroundColor Cyan
    Write-Host $Title -ForegroundColor Cyan
    Write-Host ("=" * 60) -ForegroundColor Cyan
}

function Write-Success {
    param([string]$Message)
    Write-Host "✅ $Message" -ForegroundColor Green
}

function Write-Error-Custom {
    param([string]$Message)
    Write-Host "❌ $Message" -ForegroundColor Red
}

# Step 1: Verify binaries exist
Write-Section "Step 1: Verify Build Artifacts"

$coordinatorBinary = "$ProjectPath\coordinator\arcvault-coordinator.exe"
if (-not (Test-Path $coordinatorBinary)) {
    Write-Error-Custom "Coordinator binary not found at $coordinatorBinary"
    Write-Host "Build binaries first: cd $ProjectPath\coordinator && go build -o arcvault-coordinator.exe ."
    exit 1
}
Write-Success "Coordinator binary found"

# Step 2: Create staging directories
Write-Section "Step 2: Create Staging Directories"

if (Test-Path $StagingPath) {
    Write-Host "Staging directory already exists, cleaning..." -ForegroundColor Yellow
    Remove-Item "$StagingPath\arcvault.db" -Force -ErrorAction SilentlyContinue
}

New-Item -ItemType Directory -Path $StagingPath -Force | Out-Null
Write-Success "Staging directory created: $StagingPath"

# Step 3: Copy binaries
Write-Section "Step 3: Copy Binaries"

Copy-Item $coordinatorBinary -Destination "$StagingPath\arcvault-coordinator.exe" -Force
$size = (Get-Item "$StagingPath\arcvault-coordinator.exe").Length / 1MB
Write-Success "Coordinator copied ($('{0:F1}' -f $size) MB)"

# Step 4: Generate encryption key
Write-Section "Step 4: Generate Encryption Key"

$rng = New-Object System.Security.Cryptography.RNGCryptoServiceProvider
$keyBytes = New-Object byte[] 32
$rng.GetBytes($keyBytes)
$encryptionKey = [System.BitConverter]::ToString($keyBytes).Replace("-", "").ToLower()
Write-Success "Encryption key generated (64 hex characters)"

# Save key to file for reference
$encryptionKey | Out-File "$StagingPath\ENCRYPTION_KEY.txt" -Encoding UTF8
Write-Host "⚠️  Key saved to: $StagingPath\ENCRYPTION_KEY.txt" -ForegroundColor Yellow
Write-Host "⚠️  SAVE THIS KEY SECURELY BEFORE DELETING THE FILE!" -ForegroundColor Yellow

# Step 5: Create config.json
Write-Section "Step 5: Create Configuration"

$adminToken = "staging-test-token-$(New-Guid)".Substring(0, 40)

$config = @{
    port = $Port
    admin_token = $adminToken
    database_path = "$StagingPath\arcvault.db"
    environment = "staging"
}

$config | ConvertTo-Json | Out-File "$StagingPath\config.json" -Encoding UTF8
Write-Success "Configuration created"
Write-Host "  Port: $Port"
Write-Host "  Admin Token: $adminToken"
Write-Host "  Database: $StagingPath\arcvault.db"
Write-Host "  Admin token saved to: $StagingPath\ADMIN_TOKEN.txt" -ForegroundColor Yellow
$adminToken | Out-File "$StagingPath\ADMIN_TOKEN.txt" -Encoding UTF8

# Step 6: Create startup script
Write-Section "Step 6: Create Startup Script"

$startupScript = @"
@echo off
REM ArcVault Staging Startup Script
cd "$StagingPath"
set ARCVAULT_CREDENTIAL_KEY=$encryptionKey
echo Starting ArcVault Coordinator (Staging)...
echo Port: $Port
echo.
.\arcvault-coordinator.exe start
"@

$startupScript | Out-File "$StagingPath\start-staging.bat" -Encoding ASCII
Write-Success "Startup script created: $StagingPath\start-staging.bat"

# Step 7: Create summary file
Write-Section "Step 8: Creating Summary"

$summary = @"
ARCVAULT STAGING ENVIRONMENT SUMMARY
====================================

Staging Directory: $StagingPath
Coordinator Port: $Port
Binary: $StagingPath\arcvault-coordinator.exe
Database: $StagingPath\arcvault.db

CREDENTIALS STORED IN PLAIN TEXT (for staging only):
====================================================
❌ WARNING: Save these securely BEFORE deleting the files!

Admin Token: $adminToken
  File: $StagingPath\ADMIN_TOKEN.txt

Encryption Key: $encryptionKey
  File: $StagingPath\ENCRYPTION_KEY.txt

NEXT STEPS:
===========
1. Start coordinator: .\start-staging.bat
2. Wait ~2 seconds for startup
3. Verify: Invoke-WebRequest http://localhost:$Port/api/version
4. Run tests: .\test-staging.ps1

Created: $(Get-Date)
"@

$summary | Out-File "$StagingPath\STAGING_SETUP_SUMMARY.txt" -Encoding UTF8
Write-Success "Summary created: $StagingPath\STAGING_SETUP_SUMMARY.txt"

# Step 8: Display next steps
Write-Section "Setup Complete!"

Write-Host "
Your staging environment is ready!

NEXT STEPS:
-----------
1. Start the coordinator:
   .\start-staging.bat

2. Wait for startup (watch for port $Port listening)

3. Access API:
   Invoke-WebRequest http://localhost:$Port/api/version

IMPORTANT:
----------
⚠️  Save your encryption key and admin token securely!
⚠️  Files with credentials are in: $StagingPath

After testing, proceed with merge to main.
" -ForegroundColor Green

Write-Host "Setup summary saved to: $StagingPath\STAGING_SETUP_SUMMARY.txt" -ForegroundColor Yellow

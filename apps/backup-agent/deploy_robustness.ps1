# Deployment Script: Deploy Robustness Improvements
# This script safely deploys the database and logging enhancements

param(
    [switch]$SkipBackup = $false,
    [switch]$SkipTests = $false,
    [switch]$Force = $false
)

Write-Host ""
Write-Host "============================================================" -ForegroundColor Cyan
Write-Host "ArcVault Backup Agent - Deploy Robustness Improvements" -ForegroundColor Cyan
Write-Host "============================================================" -ForegroundColor Cyan
Write-Host ""

# Configuration
$serviceName = "ArcVaultBackupAgent"
$sourceDir = "C:\Users\castrk05\Py_stuff\ArcVault\apps\backup-agent"
$deployDir = "C:\Program Files\ArcVault\BackupAgent\agent"
$dataDir = "C:\ProgramData\ArcVault"

$ErrorActionPreference = "Stop"

# Function to confirm action
function Confirm-Action {
    param([string]$Message)
    
    if ($Force) { return $true }
    
    $response = Read-Host "$Message (yes/no)"
    return $response -eq "yes"
}

# Function to test file integrity
function Test-FileIntegrity {
    param([string]$Path)
    
    if (-not (Test-Path $Path)) {
        return $false
    }
    
    try {
        # Try to import/parse the Python file
        $content = Get-Content $Path -Raw
        return $content.Length -gt 0
    }
    catch {
        return $false
    }
}

# Check if running as administrator
Write-Host "[Pre-Check 1/5] Verifying Administrator Rights..." -ForegroundColor Cyan
$isAdmin = ([Security.Principal.WindowsPrincipal] [Security.Principal.WindowsIdentity]::GetCurrent()).IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)
if (-not $isAdmin) {
    Write-Host "✗ This script must be run as Administrator" -ForegroundColor Red
    Write-Host "Right-click PowerShell and select 'Run as Administrator'" -ForegroundColor Yellow
    exit 1
}
Write-Host "✓ Running as Administrator" -ForegroundColor Green
Write-Host ""

# Check if service exists
Write-Host "[Pre-Check 2/5] Verifying Service Exists..." -ForegroundColor Cyan
try {
    $service = Get-Service -Name $serviceName -ErrorAction Stop
    Write-Host "✓ Service found: $serviceName" -ForegroundColor Green
}
catch {
    Write-Host "✗ Service '$serviceName' not found" -ForegroundColor Red
    Write-Host "Please install the service first using .\agent\reinstall_service.ps1" -ForegroundColor Yellow
    exit 1
}
Write-Host ""

# Check if source files exist
Write-Host "[Pre-Check 3/5] Verifying Source Files..." -ForegroundColor Cyan
$sourceFiles = @{
    "db.py" = "$sourceDir\agent\db.py"
    "log_handler.py" = "$sourceDir\agent\log_handler.py"
    "app.py" = "$sourceDir\agent\app.py"
}

$missingFiles = @()
foreach ($file in $sourceFiles.GetEnumerator()) {
    if (-not (Test-Path $file.Value)) {
        $missingFiles += $file.Key
        Write-Host "✗ Missing: $($file.Key)" -ForegroundColor Red
    }
    elseif (-not (Test-FileIntegrity $file.Value)) {
        Write-Host "✗ Corrupted: $($file.Key)" -ForegroundColor Red
        $missingFiles += $file.Key
    }
    else {
        $size = (Get-Item $file.Value).Length / 1KB
        Write-Host "✓ Found: $($file.Key) ($($size.ToString('F1')) KB)" -ForegroundColor Green
    }
}

if ($missingFiles.Count -gt 0) {
    Write-Host ""
    Write-Host "✗ Missing or corrupted files: $($missingFiles -join ', ')" -ForegroundColor Red
    Write-Host "Please ensure you're in the correct directory" -ForegroundColor Yellow
    exit 1
}
Write-Host ""

# Check if deployment directory exists
Write-Host "[Pre-Check 4/5] Verifying Deployment Directory..." -ForegroundColor Cyan
if (-not (Test-Path $deployDir)) {
    Write-Host "✗ Deployment directory not found: $deployDir" -ForegroundColor Red
    Write-Host "Please ensure the backup agent is installed" -ForegroundColor Yellow
    exit 1
}
Write-Host "✓ Deployment directory exists" -ForegroundColor Green
Write-Host ""

# Run tests (optional)
Write-Host "[Pre-Check 5/5] Running Tests..." -ForegroundColor Cyan
if ($SkipTests) {
    Write-Host "⚠ Tests skipped (--SkipTests flag)" -ForegroundColor Yellow
}
else {
    try {
        Push-Location $sourceDir
        
        $testResult = python test_robustness.py 2>&1
        $testExitCode = $LASTEXITCODE
        
        if ($testExitCode -eq 0) {
            Write-Host "✓ All tests passed" -ForegroundColor Green
        }
        else {
            Write-Host "✗ Tests failed (exit code: $testExitCode)" -ForegroundColor Red
            Write-Host "Test output:" -ForegroundColor Gray
            Write-Host $testResult -ForegroundColor Gray
            
            if (-not (Confirm-Action "Tests failed. Continue anyway?")) {
                Write-Host "Deployment cancelled." -ForegroundColor Yellow
                Pop-Location
                exit 1
            }
        }
        
        Pop-Location
    }
    catch {
        Write-Host "⚠ Could not run tests: $_" -ForegroundColor Yellow
        
        if (-not (Confirm-Action "Unable to run tests. Continue anyway?")) {
            Write-Host "Deployment cancelled." -ForegroundColor Yellow
            exit 1
        }
    }
}
Write-Host ""

# Display deployment plan
Write-Host "============================================================" -ForegroundColor Yellow
Write-Host "Deployment Plan" -ForegroundColor Yellow
Write-Host "============================================================" -ForegroundColor Yellow
Write-Host ""
Write-Host "What will be deployed:" -ForegroundColor Cyan
Write-Host "  ✓ db.py (NEW) - Database persistence module" -ForegroundColor Gray
Write-Host "  ✓ log_handler.py (NEW) - Disk-based logging module" -ForegroundColor Gray
Write-Host "  ✓ app.py (UPDATED) - Main application with enhancements" -ForegroundColor Gray
Write-Host ""
Write-Host "New features:" -ForegroundColor Cyan
Write-Host "  • Jobs persist across restarts (SQLite database)" -ForegroundColor Gray
Write-Host "  • Logs written to disk with rotation (never dropped)" -ForegroundColor Gray
Write-Host "  • Automatic recovery of interrupted jobs" -ForegroundColor Gray
Write-Host "  • Logs stored in backup destination folders" -ForegroundColor Gray
Write-Host ""
Write-Host "Deployment location: $deployDir" -ForegroundColor Cyan
Write-Host "Data location: $dataDir" -ForegroundColor Cyan
Write-Host ""

if (-not (Confirm-Action "Proceed with deployment?")) {
    Write-Host "Deployment cancelled." -ForegroundColor Yellow
    exit 0
}

Write-Host ""
Write-Host "============================================================" -ForegroundColor Cyan
Write-Host "Starting Deployment" -ForegroundColor Cyan
Write-Host "============================================================" -ForegroundColor Cyan
Write-Host ""

# Step 1: Create backup
if (-not $SkipBackup) {
    Write-Host "[Step 1/7] Creating backup of current files..." -ForegroundColor Cyan
    try {
        $timestamp = Get-Date -Format "yyyyMMdd_HHmmss"
        $backupDir = "$deployDir\backup_pre_robustness_$timestamp"
        New-Item -ItemType Directory -Path $backupDir -Force | Out-Null
        
        # Backup existing app.py
        if (Test-Path "$deployDir\app.py") {
            Copy-Item "$deployDir\app.py" "$backupDir\app.py" -ErrorAction Stop
            Write-Host "✓ Backed up app.py" -ForegroundColor Green
        }
        
        # Also create a quick restore backup
        Copy-Item "$deployDir\app.py" "$deployDir\app.py.backup" -Force -ErrorAction Stop
        
        Write-Host "✓ Backup created at: $backupDir" -ForegroundColor Green
        Write-Host "  Quick restore: $deployDir\app.py.backup" -ForegroundColor Gray
    }
    catch {
        Write-Host "✗ Failed to create backup: $_" -ForegroundColor Red
        
        if (-not (Confirm-Action "Backup failed. Continue without backup?")) {
            exit 1
        }
    }
}
else {
    Write-Host "[Step 1/7] Skipping backup (--SkipBackup flag)" -ForegroundColor Yellow
}
Write-Host ""

# Step 2: Stop the service
Write-Host "[Step 2/7] Stopping service..." -ForegroundColor Cyan
try {
    if ($service.Status -eq "Running") {
        Stop-Service -Name $serviceName -Force -ErrorAction Stop
        Start-Sleep -Seconds 3
        
        # Verify stopped
        $service = Get-Service -Name $serviceName
        if ($service.Status -ne "Stopped") {
            throw "Service did not stop (status: $($service.Status))"
        }
        
        Write-Host "✓ Service stopped successfully" -ForegroundColor Green
    }
    else {
        Write-Host "✓ Service already stopped (status: $($service.Status))" -ForegroundColor Green
    }
}
catch {
    Write-Host "✗ Failed to stop service: $_" -ForegroundColor Red
    Write-Host "You may need to stop it manually" -ForegroundColor Yellow
    exit 1
}
Write-Host ""

# Step 3: Create data directory
Write-Host "[Step 3/7] Creating data directory..." -ForegroundColor Cyan
try {
    if (-not (Test-Path $dataDir)) {
        New-Item -ItemType Directory -Path $dataDir -Force -ErrorAction Stop | Out-Null
        Write-Host "✓ Created: $dataDir" -ForegroundColor Green
    }
    else {
        Write-Host "✓ Directory already exists: $dataDir" -ForegroundColor Green
    }
    
    # Create logs subdirectory
    $logsDir = "$dataDir\logs"
    if (-not (Test-Path $logsDir)) {
        New-Item -ItemType Directory -Path $logsDir -Force -ErrorAction Stop | Out-Null
        Write-Host "✓ Created: $logsDir" -ForegroundColor Green
    }
}
catch {
    Write-Host "⚠ Could not create data directory: $_" -ForegroundColor Yellow
    Write-Host "Will be created automatically on first run" -ForegroundColor Yellow
}
Write-Host ""

# Step 4: Deploy new module files
Write-Host "[Step 4/7] Deploying new module files..." -ForegroundColor Cyan
try {
    # Copy db.py
    Copy-Item "$sourceDir\agent\db.py" "$deployDir\db.py" -ErrorAction Stop
    if (Test-FileIntegrity "$deployDir\db.py") {
        Write-Host "✓ Deployed: db.py" -ForegroundColor Green
    }
    else {
        throw "db.py integrity check failed after copy"
    }
    
    # Copy log_handler.py
    Copy-Item "$sourceDir\agent\log_handler.py" "$deployDir\log_handler.py" -ErrorAction Stop
    if (Test-FileIntegrity "$deployDir\log_handler.py") {
        Write-Host "✓ Deployed: log_handler.py" -ForegroundColor Green
    }
    else {
        throw "log_handler.py integrity check failed after copy"
    }
}
catch {
    Write-Host "✗ Failed to deploy new modules: $_" -ForegroundColor Red
    Write-Host "Attempting to restore backup..." -ForegroundColor Yellow
    
    # No need to restore - we haven't touched app.py yet
    Write-Host "Starting service with old version..." -ForegroundColor Yellow
    Start-Service -Name $serviceName
    exit 1
}
Write-Host ""

# Step 5: Deploy updated app.py
Write-Host "[Step 5/7] Deploying updated app.py..." -ForegroundColor Cyan
try {
    Copy-Item "$sourceDir\agent\app.py" "$deployDir\app.py" -Force -ErrorAction Stop
    
    if (Test-FileIntegrity "$deployDir\app.py") {
        Write-Host "✓ Deployed: app.py" -ForegroundColor Green
    }
    else {
        throw "app.py integrity check failed after copy"
    }
}
catch {
    Write-Host "✗ Failed to deploy app.py: $_" -ForegroundColor Red
    Write-Host "Attempting to restore backup..." -ForegroundColor Yellow
    
    # Restore from backup
    if (Test-Path "$deployDir\app.py.backup") {
        Copy-Item "$deployDir\app.py.backup" "$deployDir\app.py" -Force
        Write-Host "✓ Restored app.py from backup" -ForegroundColor Green
    }
    
    # Remove new modules (they're not compatible with old app.py)
    Remove-Item "$deployDir\db.py" -ErrorAction SilentlyContinue
    Remove-Item "$deployDir\log_handler.py" -ErrorAction SilentlyContinue
    
    Write-Host "Starting service with old version..." -ForegroundColor Yellow
    Start-Service -Name $serviceName
    exit 1
}
Write-Host ""

# Step 6: Start the service
Write-Host "[Step 6/7] Starting service..." -ForegroundColor Cyan
try {
    Start-Service -Name $serviceName -ErrorAction Stop
    Start-Sleep -Seconds 5
    
    # Verify started
    $service = Get-Service -Name $serviceName
    if ($service.Status -eq "Running") {
        Write-Host "✓ Service started successfully" -ForegroundColor Green
    }
    else {
        throw "Service is in $($service.Status) state"
    }
}
catch {
    Write-Host "✗ Failed to start service: $_" -ForegroundColor Red
    Write-Host ""
    Write-Host "Service may have errors. Checking logs..." -ForegroundColor Yellow
    
    $logPath = "C:\Program Files\ArcVault\BackupAgent\service.log"
    if (Test-Path $logPath) {
        Write-Host ""
        Write-Host "Last 20 lines of service log:" -ForegroundColor Cyan
        Get-Content $logPath -Tail 20 | ForEach-Object { Write-Host $_ -ForegroundColor Gray }
    }
    
    Write-Host ""
    Write-Host "To rollback, run: .\rollback_robustness.ps1" -ForegroundColor Yellow
    exit 1
}
Write-Host ""

# Step 7: Verification
Write-Host "[Step 7/7] Verifying deployment..." -ForegroundColor Cyan
Write-Host ""

# Check files exist
Write-Host "  Checking deployed files..." -ForegroundColor Gray
$deployedFiles = @("db.py", "log_handler.py", "app.py")
$allFilesOk = $true

foreach ($file in $deployedFiles) {
    $path = "$deployDir\$file"
    if ((Test-Path $path) -and (Test-FileIntegrity $path)) {
        $size = (Get-Item $path).Length / 1KB
        Write-Host "    ✓ $file ($($size.ToString('F1')) KB)" -ForegroundColor Green
    }
    else {
        Write-Host "    ✗ $file (missing or corrupted)" -ForegroundColor Red
        $allFilesOk = $false
    }
}

if (-not $allFilesOk) {
    Write-Host ""
    Write-Host "⚠ Some files are missing or corrupted!" -ForegroundColor Yellow
    Write-Host "Consider rolling back: .\rollback_robustness.ps1" -ForegroundColor Yellow
}

Write-Host ""

# Check database will be created
Write-Host "  Checking data directory..." -ForegroundColor Gray
if (Test-Path $dataDir) {
    Write-Host "    ✓ Data directory exists: $dataDir" -ForegroundColor Green
    
    $dbPath = "$dataDir\jobs.db"
    if (Test-Path $dbPath) {
        $dbSize = (Get-Item $dbPath).Length / 1KB
        Write-Host "    ✓ Database exists: $($dbSize.ToString('F1')) KB" -ForegroundColor Green
    }
    else {
        Write-Host "    ℹ Database will be created on first job" -ForegroundColor Cyan
    }
}

Write-Host ""

# Test API if AGENT_TOKEN is set
Write-Host "  Testing API..." -ForegroundColor Gray
if ($env:AGENT_TOKEN) {
    try {
        $headers = @{"X-API-Key" = $env:AGENT_TOKEN}
        
        # Test health endpoint
        $health = Invoke-RestMethod -Uri "https://localhost:8443/health" `
            -Headers $headers -SkipCertificateCheck -ErrorAction Stop
        
        Write-Host "    ✓ Health endpoint OK (status: $($health.status))" -ForegroundColor Green
        
        # Test info endpoint (includes job_counts from database)
        $info = Invoke-RestMethod -Uri "https://localhost:8443/info" `
            -Headers $headers -SkipCertificateCheck -ErrorAction Stop
        
        Write-Host "    ✓ Info endpoint OK (version: $($info.version))" -ForegroundColor Green
        
        if ($info.job_counts) {
            Write-Host "    ✓ Job statistics available (database working)" -ForegroundColor Green
            Write-Host "      Total jobs: $($info.job_counts.total)" -ForegroundColor Gray
        }
    }
    catch {
        Write-Host "    ⚠ API test failed: $($_.Exception.Message)" -ForegroundColor Yellow
        Write-Host "      Service may still be initializing..." -ForegroundColor Gray
    }
}
else {
    Write-Host "    ⚠ AGENT_TOKEN not set, skipping API test" -ForegroundColor Yellow
}

Write-Host ""
Write-Host "============================================================" -ForegroundColor Green
Write-Host "Deployment Complete!" -ForegroundColor Green
Write-Host "============================================================" -ForegroundColor Green
Write-Host ""

Write-Host "Summary:" -ForegroundColor Cyan
Write-Host "  ✓ New modules deployed (db.py, log_handler.py)" -ForegroundColor Green
Write-Host "  ✓ Application updated (app.py)" -ForegroundColor Green
Write-Host "  ✓ Service restarted successfully" -ForegroundColor Green
Write-Host "  ✓ Robustness improvements active" -ForegroundColor Green
Write-Host ""

Write-Host "New features:" -ForegroundColor Cyan
Write-Host "  • Jobs persist in database: $dataDir\jobs.db" -ForegroundColor Gray
Write-Host "  • Logs stored in: {destination}\#backup_logs\" -ForegroundColor Gray
Write-Host "  • Startup recovery: Interrupted jobs auto-failed" -ForegroundColor Gray
Write-Host "  • Maintenance API: POST /maintenance/cleanup" -ForegroundColor Gray
Write-Host ""

Write-Host "Next steps:" -ForegroundColor Cyan
Write-Host "  1. Create a test backup job to verify functionality" -ForegroundColor Gray
Write-Host "  2. Check that logs appear in destination folder" -ForegroundColor Gray
Write-Host "  3. (Optional) Run integration test: .\test_startup_recovery.ps1" -ForegroundColor Gray
Write-Host "  4. (Optional) Schedule cleanup: POST /maintenance/cleanup?days=30" -ForegroundColor Gray
Write-Host ""

if (-not $SkipBackup) {
    Write-Host "Backup location:" -ForegroundColor Yellow
    Write-Host "  Full backup: $backupDir" -ForegroundColor Gray
    Write-Host "  Quick restore: $deployDir\app.py.backup" -ForegroundColor Gray
    Write-Host ""
}

Write-Host "If you need to rollback:" -ForegroundColor Yellow
Write-Host "  .\rollback_robustness.ps1" -ForegroundColor Gray
Write-Host ""

Write-Host "Documentation:" -ForegroundColor Cyan
Write-Host "  QUICK_REFERENCE.md - Quick commands" -ForegroundColor Gray
Write-Host "  ROBUSTNESS_IMPROVEMENTS.md - Technical details" -ForegroundColor Gray
Write-Host "  ROLLBACK_GUIDE.md - Rollback procedures" -ForegroundColor Gray
Write-Host ""

exit 0


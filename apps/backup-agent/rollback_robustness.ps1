# Rollback Script: Revert Robustness Improvements
# This script rolls back the database and logging enhancements

param(
    [switch]$RemoveData = $false,
    [switch]$UseGit = $true,
    [switch]$Force = $false
)

Write-Host ""
Write-Host "============================================================" -ForegroundColor Yellow
Write-Host "ArcVault Backup Agent - Rollback Robustness Improvements" -ForegroundColor Yellow
Write-Host "============================================================" -ForegroundColor Yellow
Write-Host ""

# Configuration
$serviceName = "ArcVaultBackupAgent"
$agentDir = "C:\Program Files\ArcVault\BackupAgent\agent"
$dataDir = "C:\ProgramData\ArcVault"
$sourceDir = "C:\Users\castrk05\Py_stuff\ArcVault\apps\backup-agent"

# Function to confirm action
function Confirm-Action {
    param([string]$Message)
    
    if ($Force) { return $true }
    
    $response = Read-Host "$Message (yes/no)"
    return $response -eq "yes"
}

# Check if running as administrator
$isAdmin = ([Security.Principal.WindowsPrincipal] [Security.Principal.WindowsIdentity]::GetCurrent()).IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)
if (-not $isAdmin) {
    Write-Host "✗ This script must be run as Administrator" -ForegroundColor Red
    Write-Host "Right-click PowerShell and select 'Run as Administrator'" -ForegroundColor Yellow
    exit 1
}

# Confirm rollback
Write-Host "This will rollback to the version before robustness improvements." -ForegroundColor Yellow
Write-Host ""
Write-Host "What will be rolled back:" -ForegroundColor Cyan
Write-Host "  - app.py (restored to previous version)" -ForegroundColor Gray
Write-Host "  - db.py (removed)" -ForegroundColor Gray
Write-Host "  - log_handler.py (removed)" -ForegroundColor Gray
Write-Host ""
Write-Host "What will be affected:" -ForegroundColor Cyan
Write-Host "  - Job database will no longer be used" -ForegroundColor Gray
Write-Host "  - Disk-based logging will be disabled" -ForegroundColor Gray
Write-Host "  - Jobs will be stored in memory only (as before)" -ForegroundColor Gray
Write-Host ""

if (-not (Confirm-Action "Continue with rollback?")) {
    Write-Host "Rollback cancelled." -ForegroundColor Yellow
    exit 0
}

Write-Host ""

# Step 1: Stop the service
Write-Host "[Step 1/6] Stopping service..." -ForegroundColor Cyan
try {
    $service = Get-Service -Name $serviceName -ErrorAction SilentlyContinue
    if (-not $service) {
        Write-Host "✗ Service '$serviceName' not found" -ForegroundColor Red
        exit 1
    }
    
    if ($service.Status -eq "Running") {
        Stop-Service -Name $serviceName -Force
        Start-Sleep -Seconds 2
        Write-Host "✓ Service stopped" -ForegroundColor Green
    } else {
        Write-Host "✓ Service already stopped" -ForegroundColor Green
    }
}
catch {
    Write-Host "✗ Failed to stop service: $_" -ForegroundColor Red
    exit 1
}

Write-Host ""

# Step 2: Backup current files (just in case)
Write-Host "[Step 2/6] Creating backup of current files..." -ForegroundColor Cyan
try {
    $timestamp = Get-Date -Format "yyyyMMdd_HHmmss"
    $backupDir = "$agentDir\backup_$timestamp"
    New-Item -ItemType Directory -Path $backupDir -Force | Out-Null
    
    Copy-Item "$agentDir\app.py" "$backupDir\app.py" -ErrorAction SilentlyContinue
    Copy-Item "$agentDir\db.py" "$backupDir\db.py" -ErrorAction SilentlyContinue
    Copy-Item "$agentDir\log_handler.py" "$backupDir\log_handler.py" -ErrorAction SilentlyContinue
    
    Write-Host "✓ Backup created at: $backupDir" -ForegroundColor Green
}
catch {
    Write-Host "⚠ Could not create backup: $_" -ForegroundColor Yellow
    Write-Host "Continuing anyway..." -ForegroundColor Yellow
}

Write-Host ""

# Step 3: Restore old app.py
Write-Host "[Step 3/6] Restoring previous app.py..." -ForegroundColor Cyan

if ($UseGit) {
    # Try git method first
    if (Test-Path "$sourceDir\.git") {
        try {
            Push-Location $sourceDir
            
            # Find the commit before robustness changes
            $logMessage = git log --oneline --grep="robustness" -n 1
            if ($logMessage) {
                # Get the commit hash before the robustness commit
                $commitHash = ($logMessage -split ' ')[0]
                git show "${commitHash}^:agent/app.py" > "$sourceDir\agent\app.py.old"
                
                if (Test-Path "$sourceDir\agent\app.py.old") {
                    Copy-Item "$sourceDir\agent\app.py.old" "$agentDir\app.py" -Force
                    Remove-Item "$sourceDir\agent\app.py.old"
                    Write-Host "✓ Restored app.py from git" -ForegroundColor Green
                } else {
                    throw "Could not extract old version from git"
                }
            } else {
                Write-Host "⚠ Could not find robustness commit in git" -ForegroundColor Yellow
                $UseGit = $false
            }
            
            Pop-Location
        }
        catch {
            Write-Host "⚠ Git restore failed: $_" -ForegroundColor Yellow
            $UseGit = $false
        }
    } else {
        Write-Host "⚠ Git repository not found at $sourceDir" -ForegroundColor Yellow
        $UseGit = $false
    }
}

# Fallback to backup method
if (-not $UseGit) {
    $backupFile = "$agentDir\app.py.backup"
    if (Test-Path $backupFile) {
        Copy-Item $backupFile "$agentDir\app.py" -Force
        Write-Host "✓ Restored app.py from backup" -ForegroundColor Green
    } else {
        Write-Host "✗ No backup found and git method failed" -ForegroundColor Red
        Write-Host "Please manually restore app.py from a backup" -ForegroundColor Yellow
        
        # Start service with current version
        Start-Service -Name $serviceName
        exit 1
    }
}

Write-Host ""

# Step 4: Remove new module files
Write-Host "[Step 4/6] Removing new module files..." -ForegroundColor Cyan
try {
    if (Test-Path "$agentDir\db.py") {
        Remove-Item "$agentDir\db.py" -Force
        Write-Host "✓ Removed db.py" -ForegroundColor Green
    } else {
        Write-Host "  db.py not found (already removed)" -ForegroundColor Gray
    }
    
    if (Test-Path "$agentDir\log_handler.py") {
        Remove-Item "$agentDir\log_handler.py" -Force
        Write-Host "✓ Removed log_handler.py" -ForegroundColor Green
    } else {
        Write-Host "  log_handler.py not found (already removed)" -ForegroundColor Gray
    }
}
catch {
    Write-Host "⚠ Error removing module files: $_" -ForegroundColor Yellow
}

Write-Host ""

# Step 5: Handle data directory
Write-Host "[Step 5/6] Handling data directory..." -ForegroundColor Cyan
if (Test-Path $dataDir) {
    $dbPath = "$dataDir\jobs.db"
    $logsPath = "$dataDir\logs"
    
    Write-Host "Data directory found: $dataDir" -ForegroundColor Gray
    
    if (Test-Path $dbPath) {
        $dbSize = (Get-Item $dbPath).Length / 1KB
        Write-Host "  Database: $dbSize KB" -ForegroundColor Gray
    }
    
    if (Test-Path $logsPath) {
        $logCount = (Get-ChildItem $logsPath -Filter "*.log").Count
        Write-Host "  Log files: $logCount files" -ForegroundColor Gray
    }
    
    if ($RemoveData) {
        Write-Host ""
        Write-Host "⚠ You specified -RemoveData flag" -ForegroundColor Yellow
        Write-Host "This will delete the job database and all log files!" -ForegroundColor Yellow
        
        if (Confirm-Action "Are you sure you want to delete all data?") {
            Remove-Item $dataDir -Recurse -Force
            Write-Host "✓ Data directory removed" -ForegroundColor Green
        } else {
            Write-Host "Data directory preserved" -ForegroundColor Yellow
        }
    } else {
        Write-Host "✓ Data directory preserved (use -RemoveData to delete)" -ForegroundColor Green
    }
} else {
    Write-Host "✓ No data directory found" -ForegroundColor Green
}

Write-Host ""

# Step 6: Start the service
Write-Host "[Step 6/6] Starting service..." -ForegroundColor Cyan
try {
    Start-Service -Name $serviceName
    Start-Sleep -Seconds 5
    
    $service = Get-Service -Name $serviceName
    if ($service.Status -eq "Running") {
        Write-Host "✓ Service started successfully" -ForegroundColor Green
    } else {
        Write-Host "✗ Service is in $($service.Status) state" -ForegroundColor Red
        Write-Host "Check service logs for errors" -ForegroundColor Yellow
    }
}
catch {
    Write-Host "✗ Failed to start service: $_" -ForegroundColor Red
    Write-Host "Check service logs: C:\Program Files\ArcVault\BackupAgent\service.log" -ForegroundColor Yellow
}

Write-Host ""

# Verification
Write-Host "============================================================" -ForegroundColor Cyan
Write-Host "Post-Rollback Verification" -ForegroundColor Cyan
Write-Host "============================================================" -ForegroundColor Cyan
Write-Host ""

# Check service status
$service = Get-Service -Name $serviceName
Write-Host "Service Status: $($service.Status)" -ForegroundColor $(if ($service.Status -eq "Running") { "Green" } else { "Red" })

# Test API if service is running
if ($service.Status -eq "Running") {
    Write-Host ""
    Write-Host "Testing API endpoint..." -ForegroundColor Cyan
    
    try {
        $headers = @{"X-API-Key" = $env:AGENT_TOKEN}
        $info = Invoke-RestMethod -Uri "https://localhost:8443/health" `
            -Headers $headers -SkipCertificateCheck -ErrorAction Stop
        
        Write-Host "✓ API responding correctly" -ForegroundColor Green
        Write-Host "  Status: $($info.status)" -ForegroundColor Gray
        Write-Host "  Version: $($info.version)" -ForegroundColor Gray
    }
    catch {
        Write-Host "⚠ API test failed: $_" -ForegroundColor Yellow
        Write-Host "Service may still be starting up..." -ForegroundColor Yellow
    }
}

Write-Host ""
Write-Host "============================================================" -ForegroundColor Green
Write-Host "Rollback Complete" -ForegroundColor Green
Write-Host "============================================================" -ForegroundColor Green
Write-Host ""

Write-Host "Summary:" -ForegroundColor Cyan
Write-Host "  ✓ Service reverted to previous version" -ForegroundColor Green
Write-Host "  ✓ New module files removed" -ForegroundColor Green
Write-Host "  ✓ Service restarted" -ForegroundColor Green

if (Test-Path $dataDir) {
    Write-Host ""
    Write-Host "Note: Data directory preserved at:" -ForegroundColor Yellow
    Write-Host "  $dataDir" -ForegroundColor Gray
    Write-Host "  Run with -RemoveData flag to delete it" -ForegroundColor Gray
}

Write-Host ""
Write-Host "If you want to re-deploy the robustness improvements later," -ForegroundColor Cyan
Write-Host "simply run the deployment steps again." -ForegroundColor Cyan

Write-Host ""


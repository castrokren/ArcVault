<#
.SYNOPSIS
    Upgrade script for ArcVault installation

.DESCRIPTION
    Safely upgrades ArcVault installation to a new version:
    - Performs pre-upgrade checks
    - Backs up current installation
    - Stops services
    - Deploys new version
    - Migrates configurations
    - Restarts services
    - Verifies upgrade

.PARAMETER SourcePath
    Path to new version source files

.PARAMETER InstallPath
    Installation directory (default: C:\Program Files\ArcVault)

.PARAMETER DataPath
    Data directory (default: C:\ProgramData\ArcVault)

.PARAMETER Force
    Skip confirmation prompts

.PARAMETER SkipBackup
    Skip backup (not recommended)

.PARAMETER Verify
    Run verification after upgrade

.EXAMPLE
    .\Upgrade-ArcVault.ps1 -SourcePath "..\apps\backup-agent"
    Upgrade to new version

.EXAMPLE
    .\Upgrade-ArcVault.ps1 -SourcePath "C:\ArcVault-v2.0" -Verify
    Upgrade and verify

.NOTES
    Version: 1.0.0
    Requires: Administrator privileges
#>

[CmdletBinding()]
param(
    [Parameter(Mandatory=$true)]
    [string]$SourcePath,
    [string]$InstallPath = "C:\Program Files\ArcVault",
    [string]$DataPath = "C:\ProgramData\ArcVault",
    [switch]$Force,
    [switch]$SkipBackup,
    [switch]$Verify
)

$ErrorActionPreference = "Stop"
$script:UpgradeSuccess = $false
$script:BackupPath = $null

#region Logging

function Write-Log {
    param(
        [string]$Message,
        [ValidateSet("Info", "Success", "Warning", "Error")]
        [string]$Level = "Info"
    )
    
    $timestamp = Get-Date -Format "yyyy-MM-dd HH:mm:ss"
    
    switch ($Level) {
        "Info"    { Write-Host "[$timestamp] $Message" -ForegroundColor Cyan }
        "Success" { Write-Host "[$timestamp] ✓ $Message" -ForegroundColor Green }
        "Warning" { Write-Host "[$timestamp] ⚠ $Message" -ForegroundColor Yellow }
        "Error"   { Write-Host "[$timestamp] ✗ $Message" -ForegroundColor Red }
    }
}

function Write-Section {
    param([string]$Title)
    Write-Host "`n═══ $Title ═══" -ForegroundColor Cyan
}

#endregion

#region Pre-checks

function Test-Administrator {
    Write-Log "Checking administrator privileges..." -Level Info
    $currentPrincipal = New-Object Security.Principal.WindowsPrincipal([Security.Principal.WindowsIdentity]::GetCurrent())
    $isAdmin = $currentPrincipal.IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)
    
    if (-not $isAdmin) {
        Write-Log "Administrator privileges required" -Level Error
        exit 1
    }
    Write-Log "Administrator privileges confirmed" -Level Success
}

function Test-ExistingInstallation {
    Write-Log "Checking for existing installation..." -Level Info
    
    if (-not (Test-Path $InstallPath)) {
        Write-Log "No existing installation found at: $InstallPath" -Level Error
        Write-Log "Use Deploy-ArcVault.ps1 for new installations" -Level Info
        return $false
    }
    
    $appFile = Join-Path $InstallPath "arcvault_unified.py"
    if (-not (Test-Path $appFile)) {
        Write-Log "Existing installation appears corrupted" -Level Error
        return $false
    }
    
    Write-Log "Existing installation found" -Level Success
    return $true
}

function Test-SourceFiles {
    Write-Log "Validating source files..." -Level Info
    
    if (-not (Test-Path $SourcePath)) {
        Write-Log "Source path not found: $SourcePath" -Level Error
        return $false
    }
    
    # Check for required files
    $requiredFiles = @(
        "arcvault_unified.py",
        "requirements_unified.txt",
        "agent",
        "service"
    )
    
    foreach ($file in $requiredFiles) {
        $filePath = Join-Path $SourcePath $file
        if (-not (Test-Path $filePath)) {
            Write-Log "Required file/folder missing: $file" -Level Error
            return $false
        }
    }
    
    Write-Log "Source files validated" -Level Success
    return $true
}

function Get-CurrentVersion {
    Write-Log "Detecting current version..." -Level Info
    
    # Try to read version from a version file
    $versionFile = Join-Path $InstallPath "VERSION.txt"
    if (Test-Path $versionFile) {
        $version = Get-Content $versionFile -Raw
        Write-Log "Current version: $version" -Level Info
        return $version.Trim()
    }
    
    # Try to parse from Python file
    $appFile = Join-Path $InstallPath "arcvault_unified.py"
    if (Test-Path $appFile) {
        $content = Get-Content $appFile
        $versionLine = $content | Where-Object { $_ -match "version\s*=\s*['\"]([^'\"]+)['\"]" }
        if ($versionLine) {
            $version = $matches[1]
            Write-Log "Current version: $version" -Level Info
            return $version
        }
    }
    
    Write-Log "Version unknown" -Level Warning
    return "unknown"
}

function Get-NewVersion {
    Write-Log "Detecting new version..." -Level Info
    
    # Try to read version from source
    $versionFile = Join-Path $SourcePath "VERSION.txt"
    if (Test-Path $versionFile) {
        $version = Get-Content $versionFile -Raw
        Write-Log "New version: $version" -Level Info
        return $version.Trim()
    }
    
    # Try to parse from Python file
    $appFile = Join-Path $SourcePath "arcvault_unified.py"
    if (Test-Path $appFile) {
        $content = Get-Content $appFile
        $versionLine = $content | Where-Object { $_ -match "version\s*=\s*['\"]([^'\"]+)['\"]" }
        if ($versionLine) {
            $version = $matches[1]
            Write-Log "New version: $version" -Level Info
            return $version
        }
    }
    
    Write-Log "New version: unknown" -Level Warning
    return "unknown"
}

#endregion

#region Backup

function Backup-CurrentInstallation {
    if ($SkipBackup) {
        Write-Log "Skipping backup (SkipBackup flag set)" -Level Warning
        return $null
    }
    
    Write-Log "Creating backup of current installation..." -Level Info
    
    try {
        $timestamp = Get-Date -Format "yyyyMMdd_HHmmss"
        $backupDir = "$(Split-Path $InstallPath -Parent)\ArcVault_Backup_$timestamp"
        
        # Copy installation
        Write-Log "Backing up installation directory..." -Level Info
        Copy-Item -Path $InstallPath -Destination $backupDir -Recurse -Force -ErrorAction Stop
        
        # Copy data directory
        if (Test-Path $DataPath) {
            Write-Log "Backing up data directory..." -Level Info
            $dataBackup = "$backupDir\data"
            Copy-Item -Path $DataPath -Destination $dataBackup -Recurse -Force -ErrorAction Stop
        }
        
        Write-Log "Backup created: $backupDir" -Level Success
        return $backupDir
    }
    catch {
        Write-Log "Backup failed: $($_.Exception.Message)" -Level Error
        
        if (-not $Force) {
            $continue = Read-Host "Continue without backup? (Y/N)"
            if ($continue -ne "Y") {
                throw "Upgrade cancelled: Backup failed"
            }
        }
        
        return $null
    }
}

#endregion

#region Service Management

function Stop-ArcVaultService {
    Write-Log "Stopping ArcVault service..." -Level Info
    
    $service = Get-Service -Name "ArcVaultAgent" -ErrorAction SilentlyContinue
    
    if (-not $service) {
        Write-Log "Service not found" -Level Warning
        return $true
    }
    
    if ($service.Status -eq "Stopped") {
        Write-Log "Service already stopped" -Level Info
        return $true
    }
    
    try {
        Stop-Service -Name "ArcVaultAgent" -Force -ErrorAction Stop
        
        # Wait for service to stop
        $timeout = 30
        $elapsed = 0
        while ((Get-Service -Name "ArcVaultAgent").Status -ne "Stopped" -and $elapsed -lt $timeout) {
            Start-Sleep -Seconds 1
            $elapsed++
        }
        
        if ((Get-Service -Name "ArcVaultAgent").Status -eq "Stopped") {
            Write-Log "Service stopped successfully" -Level Success
            return $true
        }
        else {
            Write-Log "Service did not stop within timeout" -Level Warning
            return $false
        }
    }
    catch {
        Write-Log "Failed to stop service: $($_.Exception.Message)" -Level Error
        return $false
    }
}

function Start-ArcVaultService {
    Write-Log "Starting ArcVault service..." -Level Info
    
    $service = Get-Service -Name "ArcVaultAgent" -ErrorAction SilentlyContinue
    
    if (-not $service) {
        Write-Log "Service not found" -Level Warning
        return $true
    }
    
    try {
        Start-Service -Name "ArcVaultAgent" -ErrorAction Stop
        
        # Wait for service to start
        Start-Sleep -Seconds 3
        
        $service = Get-Service -Name "ArcVaultAgent"
        if ($service.Status -eq "Running") {
            Write-Log "Service started successfully" -Level Success
            return $true
        }
        else {
            Write-Log "Service did not start (Status: $($service.Status))" -Level Warning
            return $false
        }
    }
    catch {
        Write-Log "Failed to start service: $($_.Exception.Message)" -Level Error
        Write-Log "Check service log: $env:TEMP\arcvault_service.log" -Level Info
        return $false
    }
}

#endregion

#region Upgrade Operations

function Copy-NewFiles {
    param([string]$Source, [string]$Destination)
    
    Write-Log "Deploying new version..." -Level Info
    Write-Log "Source: $Source" -Level Info
    Write-Log "Destination: $Destination" -Level Info
    
    try {
        # Preserve certain directories/files
        $preserveDirs = @("logs", "venv")
        $preserveFiles = @("service_config.json", ".env")
        
        # Remove old application files (but preserve data)
        Write-Log "Removing old application files..." -Level Info
        $items = Get-ChildItem -Path $Destination -ErrorAction Stop
        
        foreach ($item in $items) {
            $shouldPreserve = $false
            
            foreach ($preserve in $preserveDirs) {
                if ($item.Name -eq $preserve) {
                    $shouldPreserve = $true
                    Write-Log "Preserving: $($item.Name)" -Level Info
                    break
                }
            }
            
            if (-not $shouldPreserve) {
                Remove-Item -Path $item.FullName -Recurse -Force -ErrorAction Stop
            }
        }
        
        # Copy new files
        Write-Log "Copying new application files..." -Level Info
        $excludePatterns = @("__pycache__", "*.pyc", ".pytest_cache", "dist", "build", ".venv", "venv")
        
        Get-ChildItem -Path $Source -Recurse | ForEach-Object {
            $relativePath = $_.FullName.Substring($Source.Length)
            $destPath = Join-Path $Destination $relativePath
            
            # Check if should exclude
            $exclude = $false
            foreach ($pattern in $excludePatterns) {
                if ($_.FullName -like "*$pattern*") {
                    $exclude = $true
                    break
                }
            }
            
            if (-not $exclude) {
                if ($_.PSIsContainer) {
                    if (-not (Test-Path $destPath)) {
                        New-Item -Path $destPath -ItemType Directory -Force | Out-Null
                    }
                }
                else {
                    Copy-Item -Path $_.FullName -Destination $destPath -Force
                }
            }
        }
        
        Write-Log "New files deployed successfully" -Level Success
        return $true
    }
    catch {
        Write-Log "Failed to deploy new files: $($_.Exception.Message)" -Level Error
        return $false
    }
}

function Update-PythonEnvironment {
    param([string]$InstallDir)
    
    Write-Log "Updating Python environment..." -Level Info
    
    $venvPath = Join-Path $InstallDir "venv"
    $pythonPath = Join-Path $venvPath "Scripts\python.exe"
    $pipPath = Join-Path $venvPath "Scripts\pip.exe"
    
    if (-not (Test-Path $pythonPath)) {
        Write-Log "Python virtual environment not found" -Level Warning
        Write-Log "Creating new virtual environment..." -Level Info
        
        try {
            & python -m venv $venvPath
        }
        catch {
            Write-Log "Failed to create virtual environment: $($_.Exception.Message)" -Level Error
            return $false
        }
    }
    
    # Update dependencies
    $requirementsFile = Join-Path $InstallDir "requirements_unified.txt"
    if (Test-Path $requirementsFile) {
        Write-Log "Installing/updating dependencies..." -Level Info
        
        try {
            # Upgrade pip
            & $pipPath install --upgrade pip --quiet
            
            # Install requirements
            & $pipPath install -r $requirementsFile --quiet
            
            Write-Log "Dependencies updated successfully" -Level Success
            return $true
        }
        catch {
            Write-Log "Failed to update dependencies: $($_.Exception.Message)" -Level Warning
            return $false
        }
    }
    else {
        Write-Log "Requirements file not found" -Level Warning
        return $true
    }
}

function Update-ServiceConfiguration {
    Write-Log "Checking service configuration..." -Level Info
    
    # Copy service config to installation directory (needed by Windows service)
    $dataConfig = "$DataPath\service_config.json"
    $installConfig = "$InstallPath\service_config.json"
    
    if ((Test-Path $dataConfig) -and (Test-Path $InstallPath)) {
        try {
            Copy-Item -Path $dataConfig -Destination $installConfig -Force -ErrorAction Stop
            Write-Log "Service configuration synchronized" -Level Success
        }
        catch {
            Write-Log "Failed to sync service configuration: $($_.Exception.Message)" -Level Warning
        }
    }
    
    return $true
}

function Save-VersionInfo {
    param([string]$Version, [string]$InstallDir)
    
    Write-Log "Saving version information..." -Level Info
    
    try {
        $versionFile = Join-Path $InstallDir "VERSION.txt"
        $version | Out-File -FilePath $versionFile -Encoding UTF8 -Force
        
        # Also save upgrade history
        $historyFile = Join-Path $InstallDir "UPGRADE_HISTORY.txt"
        $timestamp = Get-Date -Format "yyyy-MM-dd HH:mm:ss"
        $entry = "[$timestamp] Upgraded to version $Version"
        Add-Content -Path $historyFile -Value $entry -ErrorAction SilentlyContinue
        
        Write-Log "Version information saved" -Level Success
        return $true
    }
    catch {
        Write-Log "Failed to save version info: $($_.Exception.Message)" -Level Warning
        return $false
    }
}

#endregion

#region Main Upgrade

function Start-Upgrade {
    Write-Host ""
    Write-Host "╔══════════════════════════════════════════════════════════════╗" -ForegroundColor Cyan
    Write-Host "║              ArcVault Upgrade Utility v1.0.0                 ║" -ForegroundColor Cyan
    Write-Host "╚══════════════════════════════════════════════════════════════╝" -ForegroundColor Cyan
    Write-Host ""
    
    # Pre-checks
    Write-Section "Pre-Upgrade Checks"
    Test-Administrator
    
    if (-not (Test-ExistingInstallation)) {
        Write-Log "Cannot proceed with upgrade" -Level Error
        exit 1
    }
    
    if (-not (Test-SourceFiles)) {
        Write-Log "Source files validation failed" -Level Error
        exit 1
    }
    
    # Version info
    $currentVersion = Get-CurrentVersion
    $newVersion = Get-NewVersion
    
    Write-Host ""
    Write-Host "  Current Version: " -NoNewline
    Write-Host $currentVersion -ForegroundColor Yellow
    Write-Host "  New Version:     " -NoNewline
    Write-Host $newVersion -ForegroundColor Green
    Write-Host ""
    
    # Confirmation
    if (-not $Force) {
        Write-Host "═══════════════════════════════════════════════════════════════" -ForegroundColor Yellow
        Write-Host " This will upgrade your ArcVault installation" -ForegroundColor Yellow
        Write-Host "═══════════════════════════════════════════════════════════════" -ForegroundColor Yellow
        Write-Host ""
        
        $confirm = Read-Host "Continue with upgrade? (Y/N)"
        
        if ($confirm -ne "Y") {
            Write-Log "Upgrade cancelled by user" -Level Warning
            exit 0
        }
    }
    
    # Perform upgrade
    try {
        # Backup
        Write-Section "Creating Backup"
        $script:BackupPath = Backup-CurrentInstallation
        if ($script:BackupPath) {
            Write-Log "Backup location: $script:BackupPath" -Level Info
        }
        
        # Stop service
        Write-Section "Stopping Service"
        if (-not (Stop-ArcVaultService)) {
            throw "Failed to stop service"
        }
        
        # Deploy new files
        Write-Section "Deploying New Version"
        if (-not (Copy-NewFiles -Source $SourcePath -Destination $InstallPath)) {
            throw "Failed to deploy new files"
        }
        
        # Update Python environment
        Write-Section "Updating Python Environment"
        Update-PythonEnvironment -InstallDir $InstallPath
        
        # Update service configuration
        Write-Section "Updating Configuration"
        Update-ServiceConfiguration
        
        # Save version info
        Save-VersionInfo -Version $newVersion -InstallDir $InstallPath
        
        # Start service
        Write-Section "Starting Service"
        if (-not (Start-ArcVaultService)) {
            Write-Log "Service did not start automatically" -Level Warning
            Write-Log "You may need to start it manually" -Level Info
        }
        
        # Success
        $script:UpgradeSuccess = $true
        
        Write-Host ""
        Write-Host "═══════════════════════════════════════════════════════════════" -ForegroundColor Green
        Write-Host " ✓ UPGRADE COMPLETED SUCCESSFULLY" -ForegroundColor Green
        Write-Host "═══════════════════════════════════════════════════════════════" -ForegroundColor Green
        Write-Host ""
        Write-Host "  Upgraded from: " -NoNewline
        Write-Host $currentVersion -ForegroundColor Yellow
        Write-Host "  To:            " -NoNewline
        Write-Host $newVersion -ForegroundColor Green
        Write-Host ""
        
        if ($script:BackupPath) {
            Write-Host "A backup of your previous installation was saved to:" -ForegroundColor Gray
            Write-Host "  $script:BackupPath" -ForegroundColor White
            Write-Host ""
            Write-Host "To rollback: .\Rollback-ArcVault.ps1 -BackupPath '$script:BackupPath'" -ForegroundColor Cyan
            Write-Host ""
        }
        
        # Run verification if requested
        if ($Verify) {
            Write-Section "Running Verification"
            $verifyScript = Join-Path (Split-Path $PSCommandPath) "Verify-ArcVault.ps1"
            if (Test-Path $verifyScript) {
                & $verifyScript
            }
            else {
                Write-Log "Verification script not found" -Level Warning
            }
        }
        else {
            Write-Host "Run verification: .\Verify-ArcVault.ps1" -ForegroundColor Cyan
        }
        
        exit 0
    }
    catch {
        Write-Host ""
        Write-Host "═══════════════════════════════════════════════════════════════" -ForegroundColor Red
        Write-Host " ✗ UPGRADE FAILED" -ForegroundColor Red
        Write-Host "═══════════════════════════════════════════════════════════════" -ForegroundColor Red
        Write-Host ""
        Write-Log "Error: $($_.Exception.Message)" -Level Error
        
        if ($script:BackupPath) {
            Write-Host ""
            Write-Host "A backup was created before upgrade:" -ForegroundColor Yellow
            Write-Host "  $script:BackupPath" -ForegroundColor White
            Write-Host ""
            Write-Host "To rollback: .\Rollback-ArcVault.ps1 -BackupPath '$script:BackupPath'" -ForegroundColor Cyan
            Write-Host ""
        }
        
        exit 1
    }
}

#endregion

# Run upgrade
Start-Upgrade


<#
.SYNOPSIS
    Rollback script for ArcVault installation

.DESCRIPTION
    Safely rolls back ArcVault installation to a previous backup.
    - Stops services
    - Restores files from backup
    - Restarts services
    - Verifies rollback

.PARAMETER BackupPath
    Path to backup directory (if not specified, shows available backups)

.PARAMETER InstallPath
    Installation directory (default: C:\Program Files\ArcVault)

.PARAMETER DataPath
    Data directory (default: C:\ProgramData\ArcVault)

.PARAMETER Force
    Skip confirmation prompts

.PARAMETER Verify
    Run verification after rollback

.EXAMPLE
    .\Rollback-ArcVault.ps1
    Interactive rollback with backup selection

.EXAMPLE
    .\Rollback-ArcVault.ps1 -BackupPath "C:\Program Files\ArcVault_Backup_20241202_143022"
    Rollback to specific backup

.EXAMPLE
    .\Rollback-ArcVault.ps1 -Force
    Rollback to latest backup without confirmation

.NOTES
    Version: 1.0.0
    Requires: Administrator privileges
#>

[CmdletBinding()]
param(
    [string]$BackupPath = "",
    [string]$InstallPath = "C:\Program Files\ArcVault",
    [string]$DataPath = "C:\ProgramData\ArcVault",
    [switch]$Force,
    [switch]$Verify
)

$ErrorActionPreference = "Stop"
$script:RollbackSuccess = $false

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

#endregion

#region Backup Management

function Get-AvailableBackups {
    Write-Log "Searching for backups..." -Level Info
    
    $searchPaths = @(
        (Split-Path $InstallPath -Parent),
        "C:\Backups",
        "D:\Backups",
        $env:USERPROFILE
    )
    
    $backups = @()
    foreach ($searchPath in $searchPaths) {
        if (Test-Path $searchPath) {
            $found = Get-ChildItem -Path $searchPath -Directory -Filter "ArcVault_Backup_*" -ErrorAction SilentlyContinue
            $backups += $found
        }
    }
    
    if ($backups.Count -eq 0) {
        Write-Log "No backups found in common locations" -Level Warning
        return $null
    }
    
    # Sort by creation time (newest first)
    $backups = $backups | Sort-Object CreationTime -Descending
    
    Write-Log "Found $($backups.Count) backup(s)" -Level Success
    return $backups
}

function Select-Backup {
    param([array]$Backups)
    
    if ($Backups.Count -eq 0) {
        return $null
    }
    
    Write-Host "`nAvailable Backups:" -ForegroundColor Cyan
    Write-Host "═══════════════════════════════════════════════════════════════" -ForegroundColor Cyan
    
    for ($i = 0; $i -lt $Backups.Count; $i++) {
        $backup = $Backups[$i]
        $sizeGB = [math]::Round((Get-ChildItem $backup.FullName -Recurse -ErrorAction SilentlyContinue | 
                                 Measure-Object -Property Length -Sum).Sum / 1GB, 2)
        
        Write-Host "  [$($i + 1)] " -NoNewline -ForegroundColor Yellow
        Write-Host "$($backup.Name)" -ForegroundColor White
        Write-Host "      Created: $($backup.CreationTime.ToString('yyyy-MM-dd HH:mm:ss'))" -ForegroundColor Gray
        Write-Host "      Size: $sizeGB GB" -ForegroundColor Gray
        Write-Host "      Path: $($backup.FullName)" -ForegroundColor Gray
        Write-Host ""
    }
    
    Write-Host "═══════════════════════════════════════════════════════════════" -ForegroundColor Cyan
    
    $selection = Read-Host "`nSelect backup number (1-$($Backups.Count)) or 'Q' to quit"
    
    if ($selection -eq "Q" -or $selection -eq "q") {
        Write-Log "Rollback cancelled by user" -Level Warning
        exit 0
    }
    
    $index = $null
    if ([int]::TryParse($selection, [ref]$index) -and $index -ge 1 -and $index -le $Backups.Count) {
        return $Backups[$index - 1].FullName
    }
    else {
        Write-Log "Invalid selection" -Level Error
        return $null
    }
}

function Test-BackupValidity {
    param([string]$Path)
    
    Write-Log "Validating backup..." -Level Info
    
    if (-not (Test-Path $Path)) {
        Write-Log "Backup path does not exist: $Path" -Level Error
        return $false
    }
    
    # Check if backup contains expected files
    $requiredItems = @(
        "arcvault_unified.py",
        "agent",
        "service"
    )
    
    $valid = $true
    foreach ($item in $requiredItems) {
        $itemPath = Join-Path $Path $item
        if (-not (Test-Path $itemPath)) {
            Write-Log "Backup missing required item: $item" -Level Warning
            $valid = $false
        }
    }
    
    if ($valid) {
        Write-Log "Backup validation passed" -Level Success
    }
    else {
        Write-Log "Backup validation failed (may not be complete)" -Level Warning
        
        if (-not $Force) {
            $continue = Read-Host "Continue with rollback anyway? (Y/N)"
            if ($continue -ne "Y") {
                Write-Log "Rollback cancelled" -Level Warning
                return $false
            }
        }
    }
    
    return $true
}

#endregion

#region Rollback Operations

function Stop-ArcVaultService {
    Write-Log "Stopping ArcVault service..." -Level Info
    
    $service = Get-Service -Name "ArcVaultAgent" -ErrorAction SilentlyContinue
    
    if (-not $service) {
        Write-Log "Service not found (may not be installed)" -Level Warning
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

function Backup-CurrentInstallation {
    param([string]$PreRollbackBackupPath)
    
    Write-Log "Creating pre-rollback backup..." -Level Info
    
    if (-not (Test-Path $InstallPath)) {
        Write-Log "No current installation to backup" -Level Warning
        return $null
    }
    
    try {
        $timestamp = Get-Date -Format "yyyyMMdd_HHmmss"
        $backupDir = "$PreRollbackBackupPath\ArcVault_PreRollback_$timestamp"
        
        Copy-Item -Path $InstallPath -Destination $backupDir -Recurse -Force
        
        if (Test-Path $DataPath) {
            $dataBackup = "$backupDir\data"
            Copy-Item -Path $DataPath -Destination $dataBackup -Recurse -Force
        }
        
        Write-Log "Pre-rollback backup created: $backupDir" -Level Success
        return $backupDir
    }
    catch {
        Write-Log "Failed to create pre-rollback backup: $($_.Exception.Message)" -Level Warning
        return $null
    }
}

function Restore-Installation {
    param([string]$BackupSource, [string]$InstallDest)
    
    Write-Log "Restoring installation from backup..." -Level Info
    Write-Log "Source: $BackupSource" -Level Info
    Write-Log "Destination: $InstallDest" -Level Info
    
    try {
        # Remove current installation
        if (Test-Path $InstallDest) {
            Write-Log "Removing current installation..." -Level Info
            Remove-Item -Path $InstallDest -Recurse -Force -ErrorAction Stop
        }
        
        # Restore from backup
        Write-Log "Copying files from backup..." -Level Info
        Copy-Item -Path $BackupSource -Destination $InstallDest -Recurse -Force -ErrorAction Stop
        
        Write-Log "Installation restored" -Level Success
        return $true
    }
    catch {
        Write-Log "Failed to restore installation: $($_.Exception.Message)" -Level Error
        return $false
    }
}

function Restore-DataDirectory {
    param([string]$BackupSource, [string]$DataDest)
    
    Write-Log "Restoring data directory..." -Level Info
    
    $dataBackupPath = Join-Path $BackupSource "data"
    
    if (-not (Test-Path $dataBackupPath)) {
        Write-Log "No data directory in backup" -Level Warning
        return $true
    }
    
    try {
        # Backup current data (if exists)
        if (Test-Path $DataDest) {
            $tempBackup = "$DataDest`_temp_$(Get-Date -Format 'yyyyMMddHHmmss')"
            Move-Item -Path $DataDest -Destination $tempBackup -Force
        }
        
        # Restore data from backup
        Copy-Item -Path $dataBackupPath -Destination $DataDest -Recurse -Force -ErrorAction Stop
        
        # Remove temp backup if restore successful
        if ($tempBackup -and (Test-Path $tempBackup)) {
            Remove-Item -Path $tempBackup -Recurse -Force -ErrorAction SilentlyContinue
        }
        
        Write-Log "Data directory restored" -Level Success
        return $true
    }
    catch {
        Write-Log "Failed to restore data directory: $($_.Exception.Message)" -Level Error
        
        # Try to restore from temp backup
        if ($tempBackup -and (Test-Path $tempBackup)) {
            Write-Log "Attempting to restore previous data..." -Level Warning
            Move-Item -Path $tempBackup -Destination $DataDest -Force -ErrorAction SilentlyContinue
        }
        
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

#region Main Rollback

function Start-Rollback {
    Write-Host ""
    Write-Host "╔══════════════════════════════════════════════════════════════╗" -ForegroundColor Cyan
    Write-Host "║              ArcVault Rollback Utility v1.0.0                ║" -ForegroundColor Cyan
    Write-Host "╚══════════════════════════════════════════════════════════════╝" -ForegroundColor Cyan
    Write-Host ""
    
    # Pre-checks
    Test-Administrator
    
    # Find or select backup
    if (-not $BackupPath) {
        Write-Section "Finding Backups"
        $backups = Get-AvailableBackups
        
        if (-not $backups -or $backups.Count -eq 0) {
            Write-Log "No backups found. Cannot proceed with rollback." -Level Error
            exit 1
        }
        
        $BackupPath = Select-Backup -Backups $backups
        
        if (-not $BackupPath) {
            Write-Log "No backup selected. Rollback cancelled." -Level Warning
            exit 0
        }
    }
    
    # Validate backup
    Write-Section "Validating Backup"
    if (-not (Test-BackupValidity -Path $BackupPath)) {
        Write-Log "Backup validation failed. Cannot proceed." -Level Error
        exit 1
    }
    
    # Confirmation
    if (-not $Force) {
        Write-Host ""
        Write-Host "═══════════════════════════════════════════════════════════════" -ForegroundColor Yellow
        Write-Host " WARNING: This will replace your current installation" -ForegroundColor Yellow
        Write-Host "═══════════════════════════════════════════════════════════════" -ForegroundColor Yellow
        Write-Host ""
        Write-Host "  Backup: " -NoNewline
        Write-Host $BackupPath -ForegroundColor White
        Write-Host "  Install Path: " -NoNewline
        Write-Host $InstallPath -ForegroundColor White
        Write-Host "  Data Path: " -NoNewline
        Write-Host $DataPath -ForegroundColor White
        Write-Host ""
        
        $confirm = Read-Host "Are you sure you want to proceed? (Type 'YES' to continue)"
        
        if ($confirm -ne "YES") {
            Write-Log "Rollback cancelled by user" -Level Warning
            exit 0
        }
    }
    
    # Perform rollback
    try {
        # Stop service
        Write-Section "Stopping Service"
        if (-not (Stop-ArcVaultService)) {
            throw "Failed to stop service"
        }
        
        # Create pre-rollback backup
        Write-Section "Creating Safety Backup"
        $preRollbackBackup = Backup-CurrentInstallation -PreRollbackBackupPath (Split-Path $InstallPath -Parent)
        if ($preRollbackBackup) {
            Write-Log "Safety backup created: $preRollbackBackup" -Level Info
        }
        
        # Restore installation
        Write-Section "Restoring Installation"
        if (-not (Restore-Installation -BackupSource $BackupPath -InstallDest $InstallPath)) {
            throw "Failed to restore installation"
        }
        
        # Restore data directory (optional)
        Write-Section "Restoring Data"
        Restore-DataDirectory -BackupSource $BackupPath -DataDest $DataPath
        
        # Start service
        Write-Section "Starting Service"
        Start-ArcVaultService
        
        # Success
        $script:RollbackSuccess = $true
        
        Write-Host ""
        Write-Host "═══════════════════════════════════════════════════════════════" -ForegroundColor Green
        Write-Host " ✓ ROLLBACK COMPLETED SUCCESSFULLY" -ForegroundColor Green
        Write-Host "═══════════════════════════════════════════════════════════════" -ForegroundColor Green
        Write-Host ""
        
        if ($preRollbackBackup) {
            Write-Host "A backup of your previous installation was saved to:" -ForegroundColor Gray
            Write-Host "  $preRollbackBackup" -ForegroundColor White
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
        Write-Host " ✗ ROLLBACK FAILED" -ForegroundColor Red
        Write-Host "═══════════════════════════════════════════════════════════════" -ForegroundColor Red
        Write-Host ""
        Write-Log "Error: $($_.Exception.Message)" -Level Error
        
        if ($preRollbackBackup) {
            Write-Host ""
            Write-Host "A backup was created before rollback attempt:" -ForegroundColor Yellow
            Write-Host "  $preRollbackBackup" -ForegroundColor White
            Write-Host ""
        }
        
        exit 1
    }
}

#endregion

# Run rollback
Start-Rollback


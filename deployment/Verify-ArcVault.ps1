<#
.SYNOPSIS
    Post-installation verification script for ArcVault

.DESCRIPTION
    Comprehensive verification script that checks:
    - Installation integrity
    - Service status and health
    - Network connectivity
    - Configuration validity
    - File permissions
    - Log file analysis

.PARAMETER InstallPath
    Installation directory (default: C:\Program Files\ArcVault)

.PARAMETER DataPath
    Data directory (default: C:\ProgramData\ArcVault)

.PARAMETER Port
    Agent service port (default: 8443)

.PARAMETER Detailed
    Show detailed verification output

.PARAMETER ExportReport
    Export verification report to file

.EXAMPLE
    .\Verify-ArcVault.ps1
    Standard verification

.EXAMPLE
    .\Verify-ArcVault.ps1 -Detailed -ExportReport "C:\Reports\arcvault_verify.txt"
    Detailed verification with report export

.NOTES
    Version: 1.0.0
#>

[CmdletBinding()]
param(
    [string]$InstallPath = "C:\Program Files\ArcVault",
    [string]$DataPath = "C:\ProgramData\ArcVault",
    [int]$Port = 8443,
    [switch]$Detailed,
    [string]$ExportReport = ""
)

$ErrorActionPreference = "Continue"
$script:VerificationResults = @()
$script:TotalChecks = 0
$script:PassedChecks = 0
$script:FailedChecks = 0
$script:WarningChecks = 0

#region Helper Functions

function Write-Check {
    param(
        [string]$Name,
        [bool]$Result,
        [string]$Message = "",
        [string]$Severity = "Error"
    )
    
    $script:TotalChecks++
    
    $status = if ($Result) { "PASS" } else { if ($Severity -eq "Warning") { "WARN" } else { "FAIL" } }
    $color = switch ($status) {
        "PASS" { "Green" }
        "WARN" { "Yellow" }
        "FAIL" { "Red" }
    }
    
    if ($Result) {
        $script:PassedChecks++
    }
    elseif ($Severity -eq "Warning") {
        $script:WarningChecks++
    }
    else {
        $script:FailedChecks++
    }
    
    $icon = switch ($status) {
        "PASS" { "✓" }
        "WARN" { "⚠" }
        "FAIL" { "✗" }
    }
    
    $output = "  $icon $Name`: $status"
    Write-Host $output -ForegroundColor $color
    
    if ($Message -and ($Detailed -or -not $Result)) {
        Write-Host "    → $Message" -ForegroundColor Gray
    }
    
    $script:VerificationResults += [PSCustomObject]@{
        Check = $Name
        Status = $status
        Message = $Message
        Timestamp = Get-Date
    }
}

function Write-Section {
    param([string]$Title)
    Write-Host "`n═══ $Title ═══" -ForegroundColor Cyan
}

#endregion

#region Verification Tests

function Test-InstallationFiles {
    Write-Section "Installation Files"
    
    # Check installation directory
    $exists = Test-Path $InstallPath
    Write-Check -Name "Installation directory exists" -Result $exists -Message $InstallPath
    
    if ($exists) {
        # Check main application file
        $appFile = Join-Path $InstallPath "arcvault_unified.py"
        $exists = Test-Path $appFile
        Write-Check -Name "Main application file" -Result $exists -Message $appFile
        
        # Check virtual environment
        $venvPath = Join-Path $InstallPath "venv"
        $venvPython = Join-Path $venvPath "Scripts\python.exe"
        $exists = Test-Path $venvPython
        Write-Check -Name "Python virtual environment" -Result $exists -Message $venvPath
        
        if ($exists -and $Detailed) {
            # Check Python version
            try {
                $pythonVersion = & $venvPython --version 2>&1
                Write-Check -Name "Python version" -Result $true -Message $pythonVersion
            }
            catch {
                Write-Check -Name "Python version" -Result $false -Message "Unable to determine version"
            }
        }
        
        # Check required modules
        $requiredDirs = @("agent", "service", "gui", "cli")
        foreach ($dir in $requiredDirs) {
            $dirPath = Join-Path $InstallPath $dir
            $exists = Test-Path $dirPath
            Write-Check -Name "Module: $dir" -Result $exists -Message $dirPath -Severity "Warning"
        }
    }
}

function Test-DataDirectory {
    Write-Section "Data Directory"
    
    # Check data directory
    $exists = Test-Path $DataPath
    Write-Check -Name "Data directory exists" -Result $exists -Message $DataPath
    
    if ($exists) {
        # Check subdirectories
        $subdirs = @("keys", "logs", "logs\#backup_logs", "config")
        foreach ($subdir in $subdirs) {
            $subdirPath = Join-Path $DataPath $subdir
            $exists = Test-Path $subdirPath
            Write-Check -Name "Directory: $subdir" -Result $exists -Message $subdirPath -Severity "Warning"
        }
        
        # Check configuration files
        $configFiles = @("service_config.json", ".env")
        foreach ($configFile in $configFiles) {
            $configPath = Join-Path $DataPath $configFile
            $exists = Test-Path $configPath
            Write-Check -Name "Config: $configFile" -Result $exists -Message $configPath
        }
        
        # Check SSL certificates
        $certPath = "$DataPath\keys\server.crt"
        $keyPath = "$DataPath\keys\server.key"
        $hasCert = (Test-Path $certPath) -and (Test-Path $keyPath)
        Write-Check -Name "SSL certificate" -Result $hasCert -Message "server.crt/server.key" -Severity "Warning"
        
        # Check JWT keys
        $privateKeyPath = "$DataPath\keys\jwt_private.pem"
        $publicKeyPath = "$DataPath\keys\jwt_public.pem"
        $hasJWT = (Test-Path $privateKeyPath) -and (Test-Path $publicKeyPath)
        Write-Check -Name "JWT key pair" -Result $hasJWT -Message "jwt_private.pem/jwt_public.pem" -Severity "Warning"
    }
}

function Test-WindowsService {
    Write-Section "Windows Service"
    
    # Check if service exists
    $service = Get-Service -Name "ArcVaultAgent" -ErrorAction SilentlyContinue
    $exists = $null -ne $service
    Write-Check -Name "Service installed" -Result $exists -Message "ArcVaultAgent"
    
    if ($exists) {
        # Check service status
        $running = $service.Status -eq "Running"
        Write-Check -Name "Service running" -Result $running -Message $service.Status
        
        # Check startup type
        $autoStart = $service.StartType -eq "Automatic"
        Write-Check -Name "Automatic startup" -Result $autoStart -Message $service.StartType -Severity "Warning"
        
        if ($Detailed) {
            # Check service account
            $serviceInfo = Get-WmiObject Win32_Service -Filter "Name='ArcVaultAgent'"
            if ($serviceInfo) {
                $account = $serviceInfo.StartName
                Write-Check -Name "Service account" -Result $true -Message $account
            }
        }
    }
}

function Test-ServiceHealth {
    Write-Section "Service Health"
    
    # Check if service is responding
    $service = Get-Service -Name "ArcVaultAgent" -ErrorAction SilentlyContinue
    if ($service -and $service.Status -eq "Running") {
        # Check service log file
        $logPath = "$env:TEMP\arcvault_service.log"
        $logExists = Test-Path $logPath
        Write-Check -Name "Service log file" -Result $logExists -Message $logPath
        
        if ($logExists) {
            # Check for recent activity
            $logFile = Get-Item $logPath
            $recentActivity = $logFile.LastWriteTime -gt (Get-Date).AddMinutes(-10)
            Write-Check -Name "Recent log activity" -Result $recentActivity -Message "Last write: $($logFile.LastWriteTime)" -Severity "Warning"
            
            # Check for errors in log
            try {
                $logContent = Get-Content $logPath -Tail 100 -ErrorAction Stop
                $errors = $logContent | Where-Object { $_ -match "\[ERROR\]|\[CRITICAL\]|ERROR:|Exception|Traceback" }
                $hasErrors = $errors.Count -gt 0
                
                if ($hasErrors) {
                    $errorMsg = "Found $($errors.Count) error(s) in recent logs"
                    Write-Check -Name "Error-free operation" -Result $false -Message $errorMsg -Severity "Warning"
                    
                    if ($Detailed) {
                        Write-Host "`n    Recent errors:" -ForegroundColor Yellow
                        $errors | Select-Object -First 5 | ForEach-Object {
                            Write-Host "      $_" -ForegroundColor Gray
                        }
                    }
                }
                else {
                    Write-Check -Name "Error-free operation" -Result $true -Message "No errors in recent logs"
                }
            }
            catch {
                Write-Check -Name "Log file readable" -Result $false -Message "Unable to read log file"
            }
        }
    }
    else {
        Write-Check -Name "Service health check" -Result $false -Message "Service not running" -Severity "Warning"
    }
}

function Test-NetworkConnectivity {
    Write-Section "Network Connectivity"
    
    # Check if port is listening
    $listening = $false
    try {
        $connection = Test-NetConnection -ComputerName localhost -Port $Port -WarningAction SilentlyContinue -InformationLevel Quiet
        $listening = $connection
    }
    catch {
        $listening = $false
    }
    
    Write-Check -Name "Port $Port listening" -Result $listening -Message "localhost:$Port"
    
    if ($listening) {
        # Try to connect to the API
        try {
            $uri = "http://localhost:$Port/health"
            $response = Invoke-WebRequest -Uri $uri -TimeoutSec 5 -UseBasicParsing -ErrorAction Stop
            $reachable = $response.StatusCode -eq 200
            Write-Check -Name "API endpoint reachable" -Result $reachable -Message "/health returned $($response.StatusCode)"
        }
        catch {
            # Health endpoint might not exist, try root
            try {
                $uri = "http://localhost:$Port/"
                $response = Invoke-WebRequest -Uri $uri -TimeoutSec 5 -UseBasicParsing -ErrorAction Stop
                $reachable = $response.StatusCode -in @(200, 404, 401)
                Write-Check -Name "API endpoint reachable" -Result $reachable -Message "Root endpoint returned $($response.StatusCode)" -Severity "Warning"
            }
            catch {
                Write-Check -Name "API endpoint reachable" -Result $false -Message $_.Exception.Message -Severity "Warning"
            }
        }
    }
}

function Test-Configuration {
    Write-Section "Configuration"
    
    # Check service configuration
    $configPath = "$DataPath\service_config.json"
    if (Test-Path $configPath) {
        try {
            $config = Get-Content $configPath | ConvertFrom-Json
            Write-Check -Name "service_config.json valid" -Result $true -Message "JSON parsed successfully"
            
            if ($Detailed) {
                # Check important settings
                if ($config.port) {
                    $portMatch = $config.port -eq $Port
                    Write-Check -Name "Port configuration" -Result $portMatch -Message "Configured: $($config.port), Expected: $Port" -Severity "Warning"
                }
                
                if ($config.host) {
                    Write-Check -Name "Host configuration" -Result $true -Message $config.host
                }
            }
        }
        catch {
            Write-Check -Name "service_config.json valid" -Result $false -Message "Invalid JSON: $($_.Exception.Message)"
        }
    }
    else {
        Write-Check -Name "service_config.json exists" -Result $false -Message $configPath
    }
    
    # Check environment file
    $envPath = "$DataPath\.env"
    if (Test-Path $envPath) {
        Write-Check -Name ".env file exists" -Result $true -Message $envPath
        
        if ($Detailed) {
            try {
                $envContent = Get-Content $envPath
                $portLine = $envContent | Where-Object { $_ -match "AGENT_PORT=(\d+)" }
                if ($portLine) {
                    Write-Check -Name "Environment port setting" -Result $true -Message $portLine
                }
            }
            catch {
                Write-Check -Name "Environment file readable" -Result $false -Message $_.Exception.Message
            }
        }
    }
    else {
        Write-Check -Name ".env file exists" -Result $false -Message $envPath -Severity "Warning"
    }
}

function Test-Permissions {
    Write-Section "File Permissions"
    
    # Check if we can read installation directory
    try {
        $null = Get-ChildItem $InstallPath -ErrorAction Stop
        Write-Check -Name "Installation directory readable" -Result $true -Message $InstallPath
    }
    catch {
        Write-Check -Name "Installation directory readable" -Result $false -Message $_.Exception.Message
    }
    
    # Check if we can read/write data directory
    try {
        $null = Get-ChildItem $DataPath -ErrorAction Stop
        Write-Check -Name "Data directory readable" -Result $true -Message $DataPath
        
        # Try to write a test file
        $testFile = "$DataPath\test_write.tmp"
        "test" | Out-File $testFile -ErrorAction Stop
        Remove-Item $testFile -ErrorAction Stop
        Write-Check -Name "Data directory writable" -Result $true -Message $DataPath
    }
    catch {
        Write-Check -Name "Data directory read/write" -Result $false -Message $_.Exception.Message
    }
}

#endregion

#region Reporting

function Export-VerificationReport {
    param([string]$Path)
    
    $report = @"
╔══════════════════════════════════════════════════════════════╗
║         ArcVault Installation Verification Report            ║
╚══════════════════════════════════════════════════════════════╝

Generated: $(Get-Date -Format "yyyy-MM-dd HH:mm:ss")
Installation Path: $InstallPath
Data Path: $DataPath
Port: $Port

═══════════════════════════════════════════════════════════════

SUMMARY:
  Total Checks:   $script:TotalChecks
  Passed:         $script:PassedChecks
  Failed:         $script:FailedChecks
  Warnings:       $script:WarningChecks
  Success Rate:   $([math]::Round(($script:PassedChecks / $script:TotalChecks) * 100, 2))%

═══════════════════════════════════════════════════════════════

DETAILED RESULTS:

"@
    
    foreach ($result in $script:VerificationResults) {
        $report += "$($result.Status) - $($result.Check)`n"
        if ($result.Message) {
            $report += "      $($result.Message)`n"
        }
    }
    
    $report += @"

═══════════════════════════════════════════════════════════════

RECOMMENDATIONS:

"@
    
    if ($script:FailedChecks -gt 0) {
        $report += "⚠ There are $script:FailedChecks failed checks that need attention.`n"
        $report += "  Review the failed checks above and take corrective action.`n`n"
    }
    
    if ($script:WarningChecks -gt 0) {
        $report += "⚠ There are $script:WarningChecks warnings.`n"
        $report += "  These may not be critical but should be reviewed.`n`n"
    }
    
    if ($script:PassedChecks -eq $script:TotalChecks) {
        $report += "✓ All checks passed! Installation appears healthy.`n"
    }
    
    $report += "`n═══════════════════════════════════════════════════════════════`n"
    
    $report | Out-File -FilePath $Path -Encoding UTF8
    Write-Host "`n✓ Report exported to: $Path" -ForegroundColor Green
}

#endregion

#region Main

function Start-Verification {
    Write-Host ""
    Write-Host "╔══════════════════════════════════════════════════════════════╗" -ForegroundColor Cyan
    Write-Host "║         ArcVault Installation Verification v1.0.0            ║" -ForegroundColor Cyan
    Write-Host "╚══════════════════════════════════════════════════════════════╝" -ForegroundColor Cyan
    Write-Host ""
    
    # Run all verification tests
    Test-InstallationFiles
    Test-DataDirectory
    Test-WindowsService
    Test-ServiceHealth
    Test-NetworkConnectivity
    Test-Configuration
    Test-Permissions
    
    # Summary
    Write-Host ""
    Write-Host "═══════════════════════════════════════════════════════════════" -ForegroundColor Cyan
    Write-Host " VERIFICATION SUMMARY" -ForegroundColor Cyan
    Write-Host "═══════════════════════════════════════════════════════════════" -ForegroundColor Cyan
    Write-Host ""
    Write-Host "  Total Checks:   $script:TotalChecks" -ForegroundColor White
    Write-Host "  Passed:         $script:PassedChecks" -ForegroundColor Green
    Write-Host "  Failed:         $script:FailedChecks" -ForegroundColor Red
    Write-Host "  Warnings:       $script:WarningChecks" -ForegroundColor Yellow
    
    $successRate = [math]::Round(($script:PassedChecks / $script:TotalChecks) * 100, 2)
    Write-Host "  Success Rate:   $successRate%" -ForegroundColor $(if ($successRate -gt 90) { "Green" } elseif ($successRate -gt 70) { "Yellow" } else { "Red" })
    Write-Host ""
    
    # Overall status
    if ($script:FailedChecks -eq 0) {
        Write-Host "✓ VERIFICATION PASSED" -ForegroundColor Green
        Write-Host "  Installation appears healthy and operational." -ForegroundColor Green
    }
    elseif ($script:FailedChecks -le 2) {
        Write-Host "⚠ VERIFICATION PASSED WITH WARNINGS" -ForegroundColor Yellow
        Write-Host "  Installation is mostly functional but has some issues." -ForegroundColor Yellow
    }
    else {
        Write-Host "✗ VERIFICATION FAILED" -ForegroundColor Red
        Write-Host "  Installation has significant issues that need attention." -ForegroundColor Red
    }
    
    Write-Host ""
    
    # Export report if requested
    if ($ExportReport) {
        Export-VerificationReport -Path $ExportReport
    }
    
    # Exit code
    if ($script:FailedChecks -eq 0) {
        exit 0
    }
    else {
        exit 1
    }
}

#endregion

# Run verification
Start-Verification


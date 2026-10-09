# ArcVault Agent Service Diagnostic Script
# Run as Administrator

Write-Host "==================================" -ForegroundColor Cyan
Write-Host "ArcVault Agent Service Diagnostics" -ForegroundColor Cyan
Write-Host "==================================" -ForegroundColor Cyan
Write-Host ""

# Check if running as admin
$isAdmin = ([Security.Principal.WindowsPrincipal] [Security.Principal.WindowsIdentity]::GetCurrent()).IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)
if (-not $isAdmin) {
    Write-Host "WARNING: Not running as Administrator. Some checks may fail." -ForegroundColor Yellow
    Write-Host ""
}

# 1. Check service status
Write-Host "1. Service Status:" -ForegroundColor Green
try {
    $service = Get-Service -Name "ArcVaultAgent" -ErrorAction Stop
    Write-Host "   Status: $($service.Status)" -ForegroundColor $(if ($service.Status -eq 'Running') { 'Green' } else { 'Yellow' })
    Write-Host "   Start Type: $($service.StartType)"
} catch {
    Write-Host "   Service not found or not installed!" -ForegroundColor Red
}
Write-Host ""

# 2. Check service configuration
Write-Host "2. Service Configuration:" -ForegroundColor Green
Write-Host "   Looking for service_config.json..."
$scriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
$configPath = Join-Path $scriptDir "service_config.json"
if (Test-Path $configPath) {
    Write-Host "   Found: $configPath" -ForegroundColor Green
    try {
        $config = Get-Content $configPath | ConvertFrom-Json
        Write-Host "   Host: $($config.host)"
        Write-Host "   Port: $($config.port)"
    } catch {
        Write-Host "   ERROR: Invalid JSON in config file!" -ForegroundColor Red
    }
} else {
    Write-Host "   Not found (will use environment variables or defaults)" -ForegroundColor Yellow
}
Write-Host ""

# 3. Check environment variables
Write-Host "3. Environment Variables:" -ForegroundColor Green
$envHost = [System.Environment]::GetEnvironmentVariable('AGENT_LISTEN_HOST', 'Machine')
$envPort = [System.Environment]::GetEnvironmentVariable('AGENT_PORT', 'Machine')
if ($envHost) {
    Write-Host "   AGENT_LISTEN_HOST: $envHost"
} else {
    Write-Host "   AGENT_LISTEN_HOST: Not set" -ForegroundColor Yellow
}
if ($envPort) {
    Write-Host "   AGENT_PORT: $envPort"
} else {
    Write-Host "   AGENT_PORT: Not set" -ForegroundColor Yellow
}
Write-Host ""

# 4. Check if port is in use
Write-Host "4. Port Status:" -ForegroundColor Green
$port = 8443
if ($config -and $config.port) {
    $port = $config.port
} elseif ($envPort) {
    $port = $envPort
}
Write-Host "   Checking if port $port is in use..."
$portInUse = Get-NetTCPConnection -LocalPort $port -ErrorAction SilentlyContinue
if ($portInUse) {
    Write-Host "   Port $port is IN USE by:" -ForegroundColor Yellow
    foreach ($conn in $portInUse) {
        $proc = Get-Process -Id $conn.OwningProcess -ErrorAction SilentlyContinue
        Write-Host "     - $($proc.Name) (PID: $($conn.OwningProcess))"
    }
} else {
    Write-Host "   Port $port is available" -ForegroundColor Green
}
Write-Host ""

# 5. Check log file
Write-Host "5. Service Log File:" -ForegroundColor Green
$logPath = Join-Path $env:TEMP "arcvault_service.log"
Write-Host "   Location: $logPath"
if (Test-Path $logPath) {
    $logSize = (Get-Item $logPath).Length
    Write-Host "   Size: $([math]::Round($logSize/1KB, 2)) KB"
    $lastModified = (Get-Item $logPath).LastWriteTime
    Write-Host "   Last Modified: $lastModified"
    Write-Host ""
    Write-Host "   Last 20 lines:" -ForegroundColor Cyan
    Write-Host "   " + ("-" * 70) -ForegroundColor DarkGray
    Get-Content $logPath -Tail 20 | ForEach-Object { Write-Host "   $_" }
    Write-Host "   " + ("-" * 70) -ForegroundColor DarkGray
} else {
    Write-Host "   Log file not found (service may not have started yet)" -ForegroundColor Yellow
}
Write-Host ""

# 6. Check recent events
Write-Host "6. Recent Windows Events:" -ForegroundColor Green
try {
    $events = Get-EventLog -LogName Application -Source "ArcVaultAgent" -Newest 5 -ErrorAction Stop
    foreach ($evt in $events) {
        $color = switch ($evt.EntryType) {
            'Error' { 'Red' }
            'Warning' { 'Yellow' }
            default { 'White' }
        }
        Write-Host "   [$($evt.TimeGenerated)] $($evt.EntryType): $($evt.Message)" -ForegroundColor $color
    }
} catch {
    Write-Host "   No events found or unable to read event log" -ForegroundColor Yellow
}
Write-Host ""

# 7. Recommendations
Write-Host "==================================" -ForegroundColor Cyan
Write-Host "Recommendations:" -ForegroundColor Cyan
Write-Host "==================================" -ForegroundColor Cyan

if ($service.Status -ne 'Running') {
    Write-Host "- Service is not running. Try: sc start ArcVaultAgent" -ForegroundColor Yellow
}

if (-not (Test-Path $configPath) -and -not $envHost -and -not $envPort) {
    Write-Host "- No configuration found. Create service_config.json or set environment variables" -ForegroundColor Yellow
}

if ($portInUse -and $service.Status -ne 'Running') {
    Write-Host "- Port $port is in use by another process. Stop that process or change the port" -ForegroundColor Yellow
}

if (Test-Path $logPath) {
    $recentLog = (Get-Item $logPath).LastWriteTime -gt (Get-Date).AddMinutes(-5)
    if (-not $recentLog -and $service.Status -eq 'Running') {
        Write-Host "- Log file hasn't been updated recently. Service might be stuck" -ForegroundColor Yellow
    }
}

Write-Host ""
Write-Host "For more help, see SERVICE_TROUBLESHOOTING.md" -ForegroundColor Cyan
Write-Host ""

# Option to export full diagnostic
Write-Host "Export full diagnostic report? (y/n): " -NoNewline -ForegroundColor Green
$response = Read-Host
if ($response -eq 'y' -or $response -eq 'Y') {
    $reportPath = Join-Path $env:USERPROFILE "Desktop\arcvault_diagnostic_$(Get-Date -Format 'yyyyMMdd_HHmmss').txt"
    
    @"
ArcVault Agent Service Diagnostic Report
Generated: $(Get-Date)
========================================

SERVICE STATUS:
$($service | Format-List | Out-String)

SERVICE CONFIGURATION (sc qc):
$(sc.exe qc ArcVaultAgent)

CONFIG FILE ($configPath):
$(if (Test-Path $configPath) { Get-Content $configPath } else { "Not found" })

ENVIRONMENT VARIABLES:
AGENT_LISTEN_HOST: $envHost
AGENT_PORT: $envPort

PORT STATUS:
$(if ($portInUse) { $portInUse | Format-Table | Out-String } else { "Port $port is available" })

LOG FILE:
$("-" * 80)
$(if (Test-Path $logPath) { Get-Content $logPath } else { "Log file not found" })
$("-" * 80)

RECENT EVENTS:
$("-" * 80)
$(try { Get-EventLog -LogName Application -Source "ArcVaultAgent" -Newest 10 | Format-List | Out-String } catch { "No events found" })
$("-" * 80)
"@ | Out-File $reportPath

    Write-Host "Report saved to: $reportPath" -ForegroundColor Green
}


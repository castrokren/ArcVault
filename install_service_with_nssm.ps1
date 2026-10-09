# Install ArcVault Agent Service using NSSM
# Run as Administrator

param(
    [string]$InstallPath = "C:\Program Files (x86)\ArcVault",
    [string]$ListenHost = "0.0.0.0",
    [int]$Port = 8443
)

$ErrorActionPreference = 'Stop'

Write-Host "========================================"
Write-Host "ArcVault Agent Service Installation"
Write-Host "========================================"
Write-Host ""

# Check for admin
$isAdmin = ([Security.Principal.WindowsPrincipal] [Security.Principal.WindowsIdentity]::GetCurrent()).IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)
if (-not $isAdmin) {
    Write-Host "ERROR: Must run as Administrator!" -ForegroundColor Red
    exit 1
}

Write-Host "Running as Administrator - OK"
Write-Host ""

# Find NSSM
$scriptDir = $PSScriptRoot
$nssmPath = Join-Path $scriptDir "nssm.exe"
if (-not (Test-Path $nssmPath)) {
    Write-Host "ERROR: nssm.exe not found at: $nssmPath" -ForegroundColor Red
    exit 1
}

Write-Host "Found NSSM: $nssmPath"

# Find agent executable
$agentExe = Join-Path $scriptDir "dist\ArcVaultAgent.exe"
if (-not (Test-Path $agentExe)) {
    Write-Host "ERROR: ArcVaultAgent.exe not found at: $agentExe" -ForegroundColor Red
    exit 1
}

Write-Host "Found Agent: $agentExe"
Write-Host ""

# Create installation directory
Write-Host "Creating installation directory..."
New-Item -Path $InstallPath -ItemType Directory -Force | Out-Null

# Copy files
Write-Host "Copying files..."
Copy-Item $agentExe "$InstallPath\ArcVaultAgent.exe" -Force

# Create config file
Write-Host "Creating config file..."
$config = @{
    host = $ListenHost
    port = $Port
}
$config | ConvertTo-Json | Out-File "$InstallPath\service_config.json" -Encoding UTF8 -Force

Write-Host "Files copied to: $InstallPath"
Write-Host ""

# Stop existing service
Write-Host "Checking for existing service..."
$existingSvc = Get-Service -Name "ArcVaultAgent" -ErrorAction SilentlyContinue
if ($existingSvc) {
    Write-Host "Stopping existing service..."
    Stop-Service -Name "ArcVaultAgent" -Force -ErrorAction SilentlyContinue
    Start-Sleep -Seconds 3
    
    # Remove with NSSM
    Write-Host "Removing existing service..."
    & $nssmPath remove ArcVaultAgent confirm 2>&1 | Out-Null
    Start-Sleep -Seconds 2
}

Write-Host ""
Write-Host "Installing service with NSSM..."

# Install service
$agentExePath = "$InstallPath\ArcVaultAgent.exe"
& $nssmPath install ArcVaultAgent $agentExePath

if ($LASTEXITCODE -ne 0) {
    Write-Host "ERROR: NSSM install failed" -ForegroundColor Red
    exit 1
}

Write-Host "Service installed successfully"
Write-Host ""
Write-Host "Configuring service..."

# Configure service
& $nssmPath set ArcVaultAgent AppDirectory $InstallPath | Out-Null
& $nssmPath set ArcVaultAgent DisplayName "ArcVault Agent Service" | Out-Null
& $nssmPath set ArcVaultAgent Description "Runs the ArcVault agent" | Out-Null

# Set environment variables
& $nssmPath set ArcVaultAgent AppEnvironmentExtra "AGENT_LISTEN_HOST=$ListenHost" "AGENT_PORT=$Port" | Out-Null

# Set logs
& $nssmPath set ArcVaultAgent AppStdout "$InstallPath\agent_stdout.log" | Out-Null
& $nssmPath set ArcVaultAgent AppStderr "$InstallPath\agent_stderr.log" | Out-Null

# Auto-restart on failure
& $nssmPath set ArcVaultAgent AppExit Default Restart | Out-Null
& $nssmPath set ArcVaultAgent AppRestartDelay 5000 | Out-Null

# Auto-start
& $nssmPath set ArcVaultAgent Start SERVICE_AUTO_START | Out-Null

Write-Host "Service configured"
Write-Host ""

# Start service
Write-Host "Starting service..."
& $nssmPath start ArcVaultAgent

Start-Sleep -Seconds 5

# Check status
Write-Host ""
Write-Host "========================================"
Write-Host "Service Status:"
Write-Host "========================================"
sc.exe query ArcVaultAgent

Write-Host ""

# Check port
Write-Host "Checking port $Port..."
$conn = Get-NetTCPConnection -LocalPort $Port -ErrorAction SilentlyContinue
if ($conn) {
    Write-Host "SUCCESS! Service is listening on port $Port!" -ForegroundColor Green
    $proc = Get-Process -Id $conn.OwningProcess -ErrorAction SilentlyContinue
    Write-Host "Process: $($proc.Name) (PID: $($proc.Id))"
} else {
    Write-Host "Service not listening yet. Check logs:" -ForegroundColor Yellow
    Write-Host "  $InstallPath\agent_stdout.log"
    Write-Host "  $InstallPath\agent_stderr.log"
}

Write-Host ""
Write-Host "Installation complete!"
Write-Host ""
Write-Host "Logs are at:"
Write-Host "  $InstallPath\agent_stdout.log"
Write-Host "  $InstallPath\agent_stderr.log"
Write-Host ""

param(
    [Parameter(Mandatory=$true)] [string]$ControllerPublicKeyPath,
    [string]$ListenHost = '0.0.0.0',
    [int]$Port = 8443,
    [string]$Issuer = 'backup-controller'
)

$ErrorActionPreference = 'Stop'

# Resolve repo root relative to this script
$repoRoot = Join-Path (Split-Path -Parent $PSScriptRoot) '..' | Resolve-Path
$distExe = Join-Path $repoRoot 'dist\ArcVaultAgentSvc.exe'
if (-not (Test-Path $distExe)) {
  throw "Service EXE not found: $distExe. Build it first with scripts/build_agent_service_exe.bat"
}

# Ensure ProgramData dir exists and copy key
$pgmDataDir = Join-Path $env:ProgramData 'BackupAgent'
New-Item -Path $pgmDataDir -ItemType Directory -Force | Out-Null
$destKey = Join-Path $pgmDataDir 'controller_public.pem'
Copy-Item -Path $ControllerPublicKeyPath -Destination $destKey -Force

# Tighten ACLs: Administrators and SYSTEM only
icacls $destKey /inheritance:r | Out-Null
icacls $destKey /grant:r "Administrators:F" "SYSTEM:F" | Out-Null

# Prepare environment for the service. Use registry-based env for the service context via sc.exe config.
$envMap = @{
  'AGENT_LISTEN_HOST' = $ListenHost
  'AGENT_PORT' = "$Port"
  'AGENT_REQUIRE_JWT' = 'true'
  'AGENT_JWT_PUBLIC_KEY_PATH' = $destKey
  'AGENT_JWT_ISSUER' = $Issuer
}

# Build the binPath with environment variables set before launching the EXE using cmd /c set ... &&
$envPrefix = ($envMap.GetEnumerator() | ForEach-Object { "set $_=$($envMap[$_])" }) -join ' && '
$binPath = "cmd /c $envPrefix && `"$($distExe)`""

# Install the service
sc.exe create ArcVaultAgent binPath= "$binPath" start= auto DisplayName= "ArcVault Agent Service" | Write-Verbose
if ($LASTEXITCODE -ne 0) {
  throw "Failed to create service."
}

# Start the service
Start-Service ArcVaultAgent
Write-Host ("ArcVaultAgent service installed and started. Listening on {0}:{1}" -f $ListenHost, $Port)

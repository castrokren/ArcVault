param(
    [Parameter(Mandatory=$true)]
    [string]$PrivateKeyPath,
    [string]$AgentUrl = "http://127.0.0.1:8443"
)

$ErrorActionPreference = 'Stop'

# Resolve repo root relative to this script
$repoRoot = Join-Path (Split-Path -Parent $PSScriptRoot) '..' | Resolve-Path
$ctrlDir = Join-Path $repoRoot 'apps\backup-control'
$defaultPriv = Join-Path $ctrlDir 'keys\jwt_private.pem'

# Prefer embedded Python in virtualenv if present, else system Python
$venvPy = Join-Path $ctrlDir '.venv\Scripts\python.exe'
if (-not (Test-Path $venvPy)) { $venvPy = 'python' }

if (-not (Test-Path $ctrlDir)) { throw "Controller directory not found: $ctrlDir" }

Push-Location $ctrlDir
try {
    $env:AGENT_URL = $AgentUrl
    $pathToPriv = if ($PrivateKeyPath) { $PrivateKeyPath } elseif (Test-Path $defaultPriv) { $defaultPriv } else { '' }
    if (-not $pathToPriv -or -not (Test-Path $pathToPriv)) {
        throw "JWT private key not found. Provide -PrivateKeyPath or create: $defaultPriv"
    }
    $env:CTRL_JWT_PRIVATE_KEY_PATH = $pathToPriv
    Remove-Item Env:AGENT_TOKEN -ErrorAction SilentlyContinue | Out-Null
    Write-Host "Starting controller with JWT auth (private key): $pathToPriv"

    & $venvPy -m streamlit run controller\app.py
}
finally {
    Pop-Location
}



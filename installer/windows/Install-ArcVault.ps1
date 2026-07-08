#Requires -RunAsAdministrator
<#
.SYNOPSIS
    ArcVault installer wrapper.
    Extract the zip to your desired install location, then run this script as Administrator.
    Passes --install-dir explicitly so configs are always written to the correct location
    regardless of working directory.
#>

$ErrorActionPreference = "Stop"

$scriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
$wizard = Join-Path $scriptDir "arcvault-setup.exe"

if (-not (Test-Path $wizard)) {
    Write-Error "arcvault-setup.exe not found in $scriptDir. Make sure you extracted the full zip."
    exit 1
}

Write-Host "ArcVault Setup" -ForegroundColor Cyan
Write-Host "Install directory: $scriptDir" -ForegroundColor Cyan
Write-Host ""

& $wizard --install-dir $scriptDir

if ($LASTEXITCODE -ne 0) {
    Write-Error "Setup wizard exited with code $LASTEXITCODE"
    exit $LASTEXITCODE
}

# ArcVault full build script
# Usage: .\build.ps1
# Optional: .\build.ps1 -SkipDeploy to build without restarting the service

param(
    [switch]$SkipDeploy
)

$ErrorActionPreference = "Stop"
$root = $PSScriptRoot

Write-Host "==> Building dashboard..." -ForegroundColor Cyan
Set-Location "$root\dashboard"
npm run build
if ($LASTEXITCODE -ne 0) { Write-Host "Dashboard build failed" -ForegroundColor Red; exit 1 }

Write-Host "==> Syncing dist to embed directory..." -ForegroundColor Cyan
Remove-Item -Recurse -Force "$root\coordinator\static\dist" -ErrorAction SilentlyContinue
Copy-Item -Recurse "$root\dashboard\dist" "$root\coordinator\static\dist"

Write-Host "==> Building coordinator..." -ForegroundColor Cyan
Set-Location $root
$version = git describe --tags --always
Write-Host "  Version: $version" -ForegroundColor Gray
go build -ldflags "-X main.Version=$version" -o coordinator/arcvault-coordinator.exe coordinator/main.go
if ($LASTEXITCODE -ne 0) { Write-Host "Go build failed" -ForegroundColor Red; exit 1 }

if ($SkipDeploy) {
    Write-Host "==> Build complete (deploy skipped)" -ForegroundColor Green
    exit 0
}

Write-Host "==> Deploying..." -ForegroundColor Cyan
Stop-Service arcvault-coordinator
Copy-Item "coordinator\arcvault-coordinator.exe" "installer\windows\coordinator.exe" -Force
Start-Service arcvault-coordinator

Write-Host "==> Done" -ForegroundColor Green
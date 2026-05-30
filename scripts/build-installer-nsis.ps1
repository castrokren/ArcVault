#!/usr/bin/env pwsh
<#
.SYNOPSIS
    Build ArcVault Windows installer (.exe) using NSIS
    Final v0.2.1 release build
#>

param(
    [switch]$SkipBinaries,
    [switch]$CleanBuild
)

$ErrorActionPreference = "Stop"

Write-Host ""
Write-Host "=== ArcVault Windows Installer Build (NSIS) ===" -ForegroundColor Cyan
Write-Host ""

# Clean if requested
if ($CleanBuild) {
    Write-Host "Cleaning previous build..." -ForegroundColor Yellow
    Remove-Item -Recurse -Force "installer\windows\*.exe" -ErrorAction SilentlyContinue
    Write-Host "[OK] Clean complete" -ForegroundColor Green
}

# Build Go binaries
if (-not $SkipBinaries) {
    Write-Host ""
    Write-Host "Building Go binaries..." -ForegroundColor Cyan

    go build -o installer/windows/coordinator.exe .\coordinator
    Write-Host "[OK] Built coordinator.exe" -ForegroundColor Green

    go build -o installer/windows/agent.exe .\agent
    Write-Host "[OK] Built agent.exe" -ForegroundColor Green

    go build -o installer/windows/arcvault-setup.exe .\cmd\setup
    Write-Host "[OK] Built arcvault-setup.exe" -ForegroundColor Green
}

# Verify binaries exist
Write-Host ""
Write-Host "Verifying binaries..." -ForegroundColor Cyan
$binaries = @("coordinator.exe", "agent.exe", "arcvault-setup.exe")
foreach ($binary in $binaries) {
    if (Test-Path "installer\windows\$binary") {
        $size = [math]::Round((Get-Item "installer\windows\$binary").Length / 1MB, 1)
        Write-Host "[OK] $binary ($size MB)" -ForegroundColor Green
    } else {
        Write-Host "[FAIL] Missing: $binary" -ForegroundColor Red
        exit 1
    }
}

# Check if NSIS is installed
Write-Host ""
Write-Host "Checking for NSIS installation..." -ForegroundColor Cyan

$nsisPath = "C:\Program Files (x86)\NSIS\makensis.exe"
if (-not (Test-Path $nsisPath)) {
    $nsisPath = "C:\Program Files\NSIS\makensis.exe"
}

if (-not (Test-Path $nsisPath)) {
    Write-Host "[FAIL] NSIS not found at expected locations" -ForegroundColor Red
    Write-Host "  Install NSIS from: https://nsis.sourceforge.io/Download" -ForegroundColor Yellow
    exit 1
}

Write-Host "[OK] NSIS found at: $nsisPath" -ForegroundColor Green

# Build installer with NSIS
Write-Host ""
Write-Host "Building installer with NSIS..." -ForegroundColor Cyan

Push-Location "installer\windows"
try {
    & $nsisPath arcvault.nsi
    if ($LASTEXITCODE -ne 0) {
        Write-Host "[FAIL] NSIS build failed with exit code $LASTEXITCODE" -ForegroundColor Red
        exit 1
    }
    Write-Host "[OK] NSIS compilation complete" -ForegroundColor Green
} finally {
    Pop-Location
}

# Verify final installer
Write-Host ""
Write-Host "Verifying final installer..." -ForegroundColor Cyan
$installerPath = "installer\windows\ArcVault-Setup-0.2.1-windows-amd64.exe"

if (Test-Path $installerPath) {
    $size = [math]::Round((Get-Item $installerPath).Length / 1MB, 1)
    Write-Host ""
    Write-Host "========================================" -ForegroundColor Green
    Write-Host "BUILD COMPLETE!" -ForegroundColor Green
    Write-Host "========================================" -ForegroundColor Green
    Write-Host ""
    Write-Host "[OK] Installer: $installerPath" -ForegroundColor Green
    Write-Host "[OK] Size: $size MB" -ForegroundColor Green
    Write-Host ""
    Write-Host "Next steps:" -ForegroundColor Cyan
    Write-Host "  1. Test installer on fresh Windows system" -ForegroundColor White
    Write-Host "  2. Verify service installation" -ForegroundColor White
    Write-Host "  3. Check dashboard access on port 8080" -ForegroundColor White
    Write-Host "  4. Commit and tag v0.2.1 release" -ForegroundColor White
    Write-Host ""
} else {
    Write-Host "[FAIL] Installer not found!" -ForegroundColor Red
    exit 1
}

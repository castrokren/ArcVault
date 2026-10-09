# Build ArcVault Unified Application
# Run this script to build the all-in-one executable

$ErrorActionPreference = 'Stop'

Write-Host "========================================" -ForegroundColor Cyan
Write-Host "ArcVault Unified Application Builder" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan
Write-Host ""

# Check Python
Write-Host "Checking Python..." -ForegroundColor Green
try {
    $pythonVersion = python --version
    Write-Host "  Found: $pythonVersion" -ForegroundColor Green
} catch {
    Write-Host "  ERROR: Python not found!" -ForegroundColor Red
    exit 1
}
Write-Host ""

# Check/Install dependencies
Write-Host "Installing dependencies..." -ForegroundColor Green
pip install -r requirements_unified.txt
if ($LASTEXITCODE -ne 0) {
    Write-Host "  ERROR: Failed to install dependencies" -ForegroundColor Red
    exit 1
}
Write-Host "  Dependencies installed" -ForegroundColor Green
Write-Host ""

# Clean previous build
if (Test-Path "dist") {
    Write-Host "Cleaning previous build..." -ForegroundColor Green
    Remove-Item -Recurse -Force dist, build -ErrorAction SilentlyContinue
}
Write-Host ""

# Build with PyInstaller
Write-Host "Building executable with PyInstaller..." -ForegroundColor Green
pyinstaller build_unified.spec
if ($LASTEXITCODE -ne 0) {
    Write-Host "  ERROR: Build failed!" -ForegroundColor Red
    exit 1
}
Write-Host ""

# Check result
if (Test-Path "dist\ArcVault.exe") {
    $size = (Get-Item "dist\ArcVault.exe").Length / 1MB
    Write-Host "========================================" -ForegroundColor Green
    Write-Host "SUCCESS!" -ForegroundColor Green
    Write-Host "========================================" -ForegroundColor Green
    Write-Host ""
    Write-Host "Executable created:" -ForegroundColor Green
    Write-Host "  Location: dist\ArcVault.exe"
    Write-Host "  Size: $([math]::Round($size, 2)) MB"
    Write-Host ""
    Write-Host "To run GUI:" -ForegroundColor Cyan
    Write-Host "  .\dist\ArcVault.exe"
    Write-Host ""
    Write-Host "To install as service:" -ForegroundColor Cyan
    Write-Host "  .\dist\ArcVault.exe install"
    Write-Host ""
} else {
    Write-Host "ERROR: Executable not found after build!" -ForegroundColor Red
    exit 1
}


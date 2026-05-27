#!/usr/bin/env pwsh
<#
.SYNOPSIS
    Build ArcVault Windows installer (.exe) using Python and PyInstaller
    Simpler version with fewer formatting issues
#>

param(
    [switch]$SkipBinaries,
    [switch]$CleanBuild
)

$ErrorActionPreference = "Stop"

Write-Host ""
Write-Host "=== ArcVault Windows Installer Build ===" -ForegroundColor Cyan
Write-Host ""

# Clean if requested
if ($CleanBuild) {
    Write-Host "Cleaning previous build..." -ForegroundColor Yellow
    Remove-Item -Recurse -Force "build", "dist", "arcvault.spec" -ErrorAction SilentlyContinue
    Write-Host "✓ Clean complete" -ForegroundColor Green
}

# Create dist directory
Write-Host "Creating dist directory..." -ForegroundColor Cyan
New-Item -ItemType Directory -Path "dist" -Force | Out-Null
Write-Host "✓ Created dist/" -ForegroundColor Green

# Build Go binaries
if (-not $SkipBinaries) {
    Write-Host ""
    Write-Host "Building Go binaries..." -ForegroundColor Cyan

    go build -o dist\coordinator.exe .\coordinator
    Write-Host "✓ Built coordinator.exe" -ForegroundColor Green

    go build -o dist\agent.exe .\agent
    Write-Host "✓ Built agent.exe" -ForegroundColor Green

    go build -o dist\arcvault-setup.exe .\cmd\setup
    Write-Host "✓ Built arcvault-setup.exe" -ForegroundColor Green
}

# Verify binaries exist
Write-Host ""
Write-Host "Verifying binaries..." -ForegroundColor Cyan
$binaries = @("coordinator.exe", "agent.exe", "arcvault-setup.exe")
foreach ($binary in $binaries) {
    if (Test-Path "dist\$binary") {
        $size = [math]::Round((Get-Item "dist\$binary").Length / 1MB, 1)
        Write-Host "✓ $binary ($size MB)" -ForegroundColor Green
    } else {
        Write-Host "✗ Missing: $binary" -ForegroundColor Red
        exit 1
    }
}

# Create PyInstaller spec file
Write-Host ""
Write-Host "Creating PyInstaller spec..." -ForegroundColor Cyan

$spec = @'
# -*- mode: python ; coding: utf-8 -*-
block_cipher = None

a = Analysis(
    ['installer\\windows\\arcvault_installer.py'],
    pathex=[],
    binaries=[
        ('dist\\coordinator.exe', '.'),
        ('dist\\agent.exe', '.'),
        ('dist\\arcvault-setup.exe', '.'),
    ],
    datas=[
        ('installer\\windows\\', 'installer\\windows'),
    ],
    hiddenimports=['tkinter'],
    hookspath=[],
    runtime_hooks=[],
    excludedimports=[],
    win_no_prefer_redirects=False,
    win_private_assemblies=False,
    cipher=block_cipher,
    noarchive=False,
)

pyz = PYZ(a.pure, a.zipped_data, cipher=block_cipher)

exe = EXE(
    pyz,
    a.scripts,
    a.binaries,
    a.zipfiles,
    a.datas,
    [],
    name='ArcVault-Setup-1.1.0-windows-amd64',
    debug=False,
    bootloader_ignore_signals=False,
    strip=False,
    upx=True,
    upx_exclude=[],
    runtime_tmpdir=None,
    console=False,
    disable_windowed_traceback=False,
    target_arch=None,
    codesign_identity=None,
    entitlements_file=None,
)
'@

$spec | Out-File -FilePath "arcvault.spec" -Encoding UTF8
Write-Host "✓ Created arcvault.spec" -ForegroundColor Green

# Compile with PyInstaller
Write-Host ""
Write-Host "Compiling with PyInstaller - this may take 2-3 minutes..." -ForegroundColor Cyan
pyinstaller arcvault.spec --onefile --distpath dist
Write-Host "✓ Compilation complete" -ForegroundColor Green

# Verify final installer
Write-Host ""
Write-Host "Verifying final installer..." -ForegroundColor Cyan
$installerPath = "dist\ArcVault-Setup-1.1.0-windows-amd64.exe"

if (Test-Path $installerPath) {
    $size = [math]::Round((Get-Item $installerPath).Length / 1MB, 1)
    Write-Host ""
    Write-Host "════════════════════════════════════════" -ForegroundColor Green
    Write-Host "BUILD COMPLETE!" -ForegroundColor Green
    Write-Host "════════════════════════════════════════" -ForegroundColor Green
    Write-Host ""
    Write-Host "✓ Installer: $installerPath" -ForegroundColor Green
    Write-Host "✓ Size: $size MB" -ForegroundColor Green
    Write-Host ""
    Write-Host "Next steps:" -ForegroundColor Cyan
    Write-Host "  1. Run installer: .\dist\ArcVault-Setup-1.1.0-windows-amd64.exe" -ForegroundColor White
    Write-Host "  2. Follow the wizard to install" -ForegroundColor White
    Write-Host "  3. Verify services: Get-Service -Name ArcVault*" -ForegroundColor White
    Write-Host "  4. Check dashboard: http://localhost:8080" -ForegroundColor White
    Write-Host ""
} else {
    Write-Host "✗ Installer not found!" -ForegroundColor Red
    exit 1
}

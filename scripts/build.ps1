#!/usr/bin/env pwsh
# Simple ArcVault Windows Installer Build

Write-Host ""
Write-Host "=== ArcVault Installer Build ===" -ForegroundColor Cyan
Write-Host ""

# Clean
Write-Host "Cleaning..." -ForegroundColor Yellow
Remove-Item -Recurse -Force "build", "dist", "arcvault.spec" -ErrorAction SilentlyContinue

# Create dist
New-Item -ItemType Directory -Path "dist" -Force | Out-Null

# Build binaries
Write-Host "Building Go binaries..." -ForegroundColor Cyan
go build -o dist\coordinator.exe .\coordinator
go build -o dist\agent.exe .\agent
go build -o dist\arcvault-setup.exe .\cmd\setup

Write-Host "✓ Binaries built" -ForegroundColor Green
Write-Host ""

# Create spec file
Write-Host "Creating PyInstaller spec file..." -ForegroundColor Cyan

$specLines = @(
    "# -*- mode: python ; coding: utf-8 -*-",
    "block_cipher = None",
    "",
    "a = Analysis(",
    "    ['installer\\windows\\arcvault_installer.py'],",
    "    pathex=[],",
    "    binaries=[",
    "        ('dist\\coordinator.exe', '.'),",
    "        ('dist\\agent.exe', '.'),",
    "        ('dist\\arcvault-setup.exe', '.'),",
    "    ],",
    "    datas=[",
    "        ('installer\\windows\\', 'installer\\windows'),",
    "    ],",
    "    hiddenimports=['tkinter'],",
    "    hookspath=[],",
    "    runtime_hooks=[],",
    "    excludedimports=[],",
    "    win_no_prefer_redirects=False,",
    "    win_private_assemblies=False,",
    "    cipher=block_cipher,",
    "    noarchive=False,",
    ")",
    "",
    "pyz = PYZ(a.pure, a.zipped_data, cipher=block_cipher)",
    "",
    "exe = EXE(",
    "    pyz,",
    "    a.scripts,",
    "    a.binaries,",
    "    a.zipfiles,",
    "    a.datas,",
    "    [],",
    "    name='ArcVault-Setup-1.1.0-windows-amd64',",
    "    debug=False,",
    "    bootloader_ignore_signals=False,",
    "    strip=False,",
    "    upx=True,",
    "    upx_exclude=[],",
    "    runtime_tmpdir=None,",
    "    console=False,",
    "    disable_windowed_traceback=False,",
    "    target_arch=None,",
    "    codesign_identity=None,",
    "    entitlements_file=None,",
    ")"
)

$specLines | Out-File -FilePath "arcvault.spec" -Encoding UTF8
Write-Host "✓ Spec file created" -ForegroundColor Green
Write-Host ""

# Compile
Write-Host "Compiling with PyInstaller..." -ForegroundColor Cyan
Write-Host "(This may take 2-3 minutes)" -ForegroundColor Gray
Write-Host ""

pyinstaller arcvault.spec --distpath dist

Write-Host ""
Write-Host "✓ PyInstaller compilation done" -ForegroundColor Green
Write-Host ""

# Verify
if (Test-Path "dist\ArcVault-Setup-1.1.0-windows-amd64.exe") {
    $size = [math]::Round((Get-Item "dist\ArcVault-Setup-1.1.0-windows-amd64.exe").Length / 1MB, 1)

    Write-Host "════════════════════════════════════" -ForegroundColor Green
    Write-Host "SUCCESS!" -ForegroundColor Green
    Write-Host "════════════════════════════════════" -ForegroundColor Green
    Write-Host ""
    Write-Host "Installer: dist\ArcVault-Setup-1.1.0-windows-amd64.exe" -ForegroundColor Green
    Write-Host "Size: $size MB" -ForegroundColor Green
    Write-Host ""
    Write-Host "Run: .\dist\ArcVault-Setup-1.1.0-windows-amd64.exe" -ForegroundColor Cyan
    Write-Host ""
} else {
    Write-Host "ERROR: Installer not found!" -ForegroundColor Red
    exit 1
}

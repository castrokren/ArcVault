#!/usr/bin/env pwsh
# ArcVault Windows Installer Build

Write-Host ""
Write-Host "=== ArcVault Installer Build ===" -ForegroundColor Cyan
Write-Host ""

# Clean
Write-Host "Cleaning..." -ForegroundColor Yellow
Remove-Item -Recurse -Force "build", "dist", "arcvault.spec" -ErrorAction SilentlyContinue

# Create dist
New-Item -ItemType Directory -Path "dist" -Force | Out-Null

# Version — read from VERSION file (single source of truth).
# NEVER hardcode a version string here; it will drift.
$Version = (Get-Content "$PSScriptRoot\..\VERSION" -Raw).Trim()
if (-not $Version) { Write-Host "ERROR: VERSION file is empty" -ForegroundColor Red; exit 1 }
& "$PSScriptRoot\check-version-sync.ps1" -Expected $Version
if ($LASTEXITCODE -ne 0) { exit 1 }
Write-Host "Building Go binaries at $Version..." -ForegroundColor Cyan

go build -ldflags "-X main.Version=$Version -X arcvault/coordinator/server.Version=$Version" -o dist\coordinator.exe .\coordinator
if ($LASTEXITCODE -ne 0) { Write-Host "ERROR: coordinator build failed" -ForegroundColor Red; exit 1 }
go build -ldflags "-X main.Version=$Version" -o dist\agent.exe .\agent
if ($LASTEXITCODE -ne 0) { Write-Host "ERROR: agent build failed" -ForegroundColor Red; exit 1 }

# ── GUARDRAIL: verify binary reports the correct version ─────────────────────
$binVer = & "dist\coordinator.exe" --version 2>&1
$binVer = ($binVer -join "").Trim()
if ($binVer -ne $Version) {
    Write-Host "ERROR: coordinator.exe reports '$binVer' but expected '$Version'" -ForegroundColor Red
    Write-Host "       ldflags may not have been applied correctly." -ForegroundColor Yellow
    exit 1
}
Write-Host "  Binary version verified: $binVer" -ForegroundColor Green

Write-Host "Binaries built OK" -ForegroundColor Green
Write-Host ""

# Write PyInstaller spec using here-string (avoids quoting/encoding issues)
Write-Host "Creating PyInstaller spec file..." -ForegroundColor Cyan

$spec = @"
# -*- mode: python ; coding: utf-8 -*-
block_cipher = None

a = Analysis(
    ['installer/windows/arcvault_installer.py'],
    pathex=[],
    binaries=[
        ('dist/coordinator.exe', '.'),
        ('dist/agent.exe', '.'),
    ],
    datas=[
        ('installer/windows/', 'installer/windows'),
    ],
    hiddenimports=['tkinter'],
    hookspath=[],
    runtime_hooks=[],
    excludes=[],
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
    name='ArcVault-Setup-$($Version.TrimStart("v"))-windows-amd64',
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
"@

$spec | Out-File -FilePath "arcvault.spec" -Encoding UTF8
Write-Host "Spec file created OK" -ForegroundColor Green
Write-Host ""

# Compile
Write-Host "Compiling with PyInstaller..." -ForegroundColor Cyan
Write-Host "(This may take 2-3 minutes)" -ForegroundColor Gray
Write-Host ""

pyinstaller arcvault.spec --distpath dist
if ($LASTEXITCODE -ne 0) { Write-Host "ERROR: PyInstaller failed" -ForegroundColor Red; exit 1 }

Write-
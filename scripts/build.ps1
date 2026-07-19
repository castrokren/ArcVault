#!/usr/bin/env pwsh
# ArcVault Windows Installer Build

Write-Host ""
Write-Host "=== ArcVault Installer Build ===" -ForegroundColor Cyan
Write-Host ""

# Clean — build into installer\windows\dist (the dir config.json's installer_dir
# serves from). Also nuke any legacy root-level dist\ so it stops confusing people.
Write-Host "Cleaning..." -ForegroundColor Yellow
Remove-Item -Recurse -Force "build", "dist", "installer\windows\dist", "arcvault.spec" -ErrorAction SilentlyContinue

# Create dist
New-Item -ItemType Directory -Path "installer\windows\dist" -Force | Out-Null

# Version — read from VERSION file (single source of truth).
# NEVER hardcode a version string here; it will drift.
$Version = (Get-Content "$PSScriptRoot\..\VERSION" -Raw).Trim()
if (-not $Version) { Write-Host "ERROR: VERSION file is empty" -ForegroundColor Red; exit 1 }
& "$PSScriptRoot\check-version-sync.ps1" -Expected $Version
if ($LASTEXITCODE -ne 0) { exit 1 }
Write-Host "Building Go binaries at $Version..." -ForegroundColor Cyan

go build -ldflags "-X main.Version=$Version -X arcvault/coordinator/server.Version=$Version" -o installer\windows\dist\coordinator.exe .\coordinator
if ($LASTEXITCODE -ne 0) { Write-Host "ERROR: coordinator build failed" -ForegroundColor Red; exit 1 }
go build -ldflags "-X main.Version=$Version" -o installer\windows\dist\agent.exe .\agent
if ($LASTEXITCODE -ne 0) { Write-Host "ERROR: agent build failed" -ForegroundColor Red; exit 1 }

# ── GUARDRAIL: verify binary reports the correct version ─────────────────────
$binVer = & "installer\windows\dist\coordinator.exe" --version 2>&1
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
        ('installer/windows/dist/coordinator.exe', '.'),
        ('installer/windows/dist/agent.exe', '.'),
    ],
    # No datas: the installer reads nothing from installer/windows at runtime
    # (icon is base64-embedded; coordinator.exe/agent.exe come from binaries=
    # above). Bundling the whole folder swept in stale dist/build artifacts and
    # loose exe copies, bloating the installer ~150MB. See git log.
    datas=[],
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
    uac_admin=True,
    icon='installer/windows/icon.ico',
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

pyinstaller arcvault.spec --distpath installer\windows\dist
if ($LASTEXITCODE -ne 0) { Write-Host "ERROR: PyInstaller failed" -ForegroundColor Red; exit 1 }

Write-Host ""
Write-Host "PyInstaller compilation done" -ForegroundColor Green
Write-Host ""

# Verify — name is derived from $Version (no hardcoded strings)
$versionTrimmed = $Version.TrimStart("v")
$outExe = "installer\windows\dist\ArcVault-Setup-$versionTrimmed-windows-amd64.exe"
if (Test-Path $outExe) {
    $size = [math]::Round((Get-Item $outExe).Length / 1MB, 1)
    Write-Host "====================================" -ForegroundColor Green
    Write-Host "SUCCESS!" -ForegroundColor Green
    Write-Host "====================================" -ForegroundColor Green
    Write-Host ""
    Write-Host "Installer: $outExe" -ForegroundColor Green
    Write-Host "Size: $size MB" -ForegroundColor Green
    Write-Host ""
    Write-Host "Run: .\$outExe" -ForegroundColor Cyan
    Write-Host ""
} else {
    Write-Host "ERROR: Installer not found at $outExe" -ForegroundColor Red
    exit 1
}

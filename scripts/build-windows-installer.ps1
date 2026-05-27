param(
    [switch]$SkipBinaries,
    [switch]$CleanBuild
)

Set-Location $PSScriptRoot\..

Write-Host ""
Write-Host "==> Step 1: Verifying prerequisites..." -ForegroundColor Cyan

$pythonVersion = python --version 2>&1
if ($LASTEXITCODE -eq 0) { Write-Host "OK Python: $pythonVersion" -ForegroundColor Green }
else { Write-Host "FAIL Python not found" -ForegroundColor Red; exit 1 }

$goVersion = go version 2>&1
if ($LASTEXITCODE -eq 0) { Write-Host "OK Go: $goVersion" -ForegroundColor Green }
else { Write-Host "FAIL Go not found" -ForegroundColor Red; exit 1 }

$pyiVersion = pyinstaller --version 2>&1
if ($LASTEXITCODE -ne 0) {
    Write-Host "Installing PyInstaller..." -ForegroundColor Cyan
    pip install pyinstaller
    if ($LASTEXITCODE -ne 0) { Write-Host "FAIL Could not install PyInstaller" -ForegroundColor Red; exit 1 }
}
Write-Host "OK PyInstaller found" -ForegroundColor Green

if ($CleanBuild) {
    Write-Host ""
    Write-Host "==> Cleaning previous build..." -ForegroundColor Cyan
    Remove-Item -Recurse -Force "build" -ErrorAction SilentlyContinue
    Remove-Item -Recurse -Force "arcvault.spec" -ErrorAction SilentlyContinue
}

if (-not (Test-Path "dist")) { New-Item -ItemType Directory -Path "dist" -Force | Out-Null }
if (-not (Test-Path "deployment")) { New-Item -ItemType Directory -Path "deployment" -Force | Out-Null }

if (-not $SkipBinaries) {
    Write-Host ""
    Write-Host "==> Step 4: Building Go binaries..." -ForegroundColor Cyan
    foreach ($b in @(@{n="coordinator";p=".\coordinator"}, @{n="agent";p=".\agent"}, @{n="arcvault-setup";p=".\cmd\setup"})) {
        Write-Host "Building $($b.n)..."
        go build -o "dist\$($b.n).exe" $b.p
        if ($LASTEXITCODE -ne 0) { Write-Host "FAIL $($b.n)" -ForegroundColor Red; exit 1 }
        Write-Host "OK $($b.n).exe" -ForegroundColor Green
    }
} else {
    Write-Host ""
    Write-Host "==> Step 4: Skipping Go binaries (already built)" -ForegroundColor Cyan
}

Write-Host ""
Write-Host "==> Step 5: Verifying binaries..." -ForegroundColor Cyan
foreach ($b in @("coordinator.exe", "agent.exe", "arcvault-setup.exe")) {
    if (Test-Path "dist\$b") {
        $size = [math]::Round((Get-Item "dist\$b").Length / 1MB, 1)
        Write-Host "OK $b ($size MB)" -ForegroundColor Green
    } else {
        Write-Host "FAIL Missing dist\$b" -ForegroundColor Red; exit 1
    }
}

Write-Host ""
Write-Host "==> Step 6: Creating PyInstaller spec..." -ForegroundColor Cyan

@'
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
'@ | Out-File -FilePath "arcvault.spec" -Encoding UTF8

Write-Host "OK arcvault.spec created" -ForegroundColor Green

Write-Host ""
Write-Host "==> Step 7: Running PyInstaller (2-3 min)..." -ForegroundColor Cyan

pyinstaller arcvault.spec --distpath deployment

if ($LASTEXITCODE -ne 0) { Write-Host "FAIL PyInstaller failed" -ForegroundColor Red; exit 1 }

Write-Host ""
Write-Host "==> Step 8: Verifying installer..." -ForegroundColor Cyan

$out = "deployment\ArcVault-Setup-1.1.0-windows-amd64.exe"
if (Test-Path $out) {
    $size = [math]::Round((Get-Item $out).Length / 1MB, 1)
    Write-Host ""
    Write-Host "BUILD COMPLETE" -ForegroundColor Green
    Write-Host "Installer: $out ($size MB)" -ForegroundColor Green
} else {
    Write-Host "FAIL Installer not found at $out" -ForegroundColor Red; exit 1
}

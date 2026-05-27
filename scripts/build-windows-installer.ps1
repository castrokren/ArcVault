#!/usr/bin/env pwsh
<#
.SYNOPSIS
    Build ArcVault Windows installer (.exe) using Python and PyInstaller
.DESCRIPTION
    Complete automated build script for the Windows installer
.EXAMPLE
    .\build-windows-installer.ps1
.NOTES
    Requires: Python 3.8+, Go 1.25+, PyInstaller
#>

param(
    [switch]$SkipBinaries,
    [switch]$CleanBuild
)

# Colors for output
$colors = @{
    Success = 'Green'
    Error = 'Red'
    Warning = 'Yellow'
    Info = 'Cyan'
}

function Write-BuildSuccess {
    param([string]$Message)
    Write-Host "✓ $Message" -ForegroundColor Green
}

function Write-BuildError {
    param([string]$Message)
    Write-Host "✗ $Message" -ForegroundColor Red
}

function Write-BuildInfo {
    param([string]$Message)
    Write-Host "ℹ $Message" -ForegroundColor Cyan
}

# ============================================================================
# STEP 1: Verify Prerequisites
# ============================================================================
Write-Host ""
Write-BuildInfo "Step 1: Verifying prerequisites..."
Write-Host ""

# Check Python
$pythonVersion = python --version 2>&1
if ($LASTEXITCODE -eq 0) {
    Write-BuildSuccess "Python found: $pythonVersion"
} else {
    Write-BuildError "Python not found. Install from https://www.python.org/"
    exit 1
}

# Check Go
$goVersion = go version
if ($LASTEXITCODE -eq 0) {
    Write-BuildSuccess "Go found: $goVersion"
} else {
    Write-BuildError "Go not found. Install from https://go.dev/dl/"
    exit 1
}

# Check PyInstaller
$pyinstallerVersion = pyinstaller --version 2>&1
if ($LASTEXITCODE -eq 0) {
    Write-BuildSuccess "PyInstaller found: $pyinstallerVersion"
} else {
    Write-Host ""
    Write-BuildInfo "Installing PyInstaller..."
    pip install pyinstaller
    if ($LASTEXITCODE -ne 0) {
        Write-BuildError "Failed to install PyInstaller"
        exit 1
    }
    Write-BuildSuccess "PyInstaller installed"
}

# ============================================================================
# STEP 2: Clean Build (Optional)
# ============================================================================
if ($CleanBuild) {
    Write-Host ""
    Write-BuildInfo "Step 2: Cleaning previous build..."

    if (Test-Path "build") {
        Write-Host "Removing build directory..."
        Remove-Item -Recurse -Force "build" -ErrorAction SilentlyContinue
    }

    if (Test-Path "dist") {
        Write-Host "Removing dist directory..."
        Remove-Item -Recurse -Force "dist" -ErrorAction SilentlyContinue
    }

    if (Test-Path "arcvault.spec") {
        Write-Host "Removing arcvault.spec..."
        Remove-Item -Force "arcvault.spec" -ErrorAction SilentlyContinue
    }

    Write-BuildSuccess "Clean build ready"
}

# ============================================================================
# STEP 3: Create Build Directories
# ============================================================================
Write-Host ""
Write-BuildInfo "Step 3: Creating build directories..."

if (-not (Test-Path "dist")) {
    New-Item -ItemType Directory -Path "dist" -Force | Out-Null
    Write-BuildSuccess "Created dist directory"
}

# ============================================================================
# STEP 4: Build Go Binaries
# ============================================================================
if (-not $SkipBinaries) {
    Write-Host ""
    Write-BuildInfo "Step 4: Building Go binaries..."

    $binaries = @(
        @{ Name = "coordinator"; Path = ".\coordinator" },
        @{ Name = "agent"; Path = ".\agent" },
        @{ Name = "arcvault-setup"; Path = ".\cmd\setup" }
    )

    foreach ($binary in $binaries) {
        Write-Host "Building $($binary.Name)..."
        go build -o "dist\$($binary.Name).exe" $binary.Path

        if ($LASTEXITCODE -eq 0) {
            $size = (Get-Item "dist\$($binary.Name).exe").Length / 1MB
            Write-BuildSuccess "Built: $($binary.Name).exe ($('{0:F1}' -f $size) MB)"
        } else {
            Write-BuildError "Failed to build $($binary.Name)"
            exit 1
        }
    }
} else {
    Write-Host ""
    Write-BuildInfo "Step 4: Skipping Go binaries (already built)"
}

# ============================================================================
# STEP 5: Verify Go Binaries
# ============================================================================
Write-Host ""
Write-BuildInfo "Step 5: Verifying binaries..."

$requiredBinaries = @("coordinator.exe", "agent.exe", "arcvault-setup.exe")
foreach ($binary in $requiredBinaries) {
    if (Test-Path "dist\$binary") {
        $size = (Get-Item "dist\$binary").Length / 1MB
        Write-BuildSuccess "Found: $binary ($('{0:F1}' -f $size) MB)"
    } else {
        Write-BuildError "Missing: $binary"
        exit 1
    }
}

# ============================================================================
# STEP 6: Create PyInstaller Spec File
# ============================================================================
Write-Host ""
Write-BuildInfo "Step 6: Creating PyInstaller spec file..."

$specContent = @'
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

$specContent | Out-File -FilePath "arcvault.spec" -Encoding UTF8
Write-BuildSuccess "Created: arcvault.spec"

# ============================================================================
# STEP 7: Compile with PyInstaller
# ============================================================================
Write-Host ""
Write-BuildInfo "Step 7: Compiling installer with PyInstaller..."
Write-Host "This may take 2-3 minutes..."

pyinstaller arcvault.spec --distpath deployment

if ($LASTEXITCODE -eq 0) {
    Write-BuildSuccess "PyInstaller compilation successful"
} else {
    Write-BuildError "PyInstaller compilation failed"
    exit 1
}

# ============================================================================
# STEP 8: Verify Final Installer
# ============================================================================
Write-Host ""
Write-BuildInfo "Step 8: Verifying final installer..."

$installerPath = "deployment\ArcVault-Setup-1.1.0-windows-amd64.exe"

if (Test-Path $installerPath) {
    $size = (Get-Item $installerPath).Length / 1MB
    Write-BuildSuccess "Installer created successfully!"
    Write-BuildSuccess "File: $(Split-Path -Leaf $installerPath)"
    Write-BuildSuccess "Size: $('{0:F1}' -f $size) MB"
    Write-BuildSuccess "Path: $(Get-Item $installerPath | Select-Object -ExpandProperty FullName)"
} else {
    Write-BuildError "Installer not found at $installerPath"
    exit 1
}

# ============================================================================
# STEP 9: Summary
# ============================================================================
Write-Host ""
Write-Host "═════════════════════════════════════════════════════════" -ForegroundColor Cyan
Write-Host "BUILD COMPLETE!" -ForegroundColor Green
Write-Host "═════════════════════════════════════════════════════════" -ForegroundColor Cyan
Write-Host ""
Write-Host "Installer ready: $installerPath" -ForegroundColor Green
Write-Host ""
Write-Host "Next steps:"
Write-Host "  1. Test the installer: .\dist\ArcVault-Setup-1.1.0-windows-amd64.exe"
Write-Host "  2. Verify installation works"
Write-Host "  3. Commit to git:"
Write-Host "     git add installer/windows/ arcvault.spec"
Write-Host "     git commit -m 'Phase 18: Python-based Windows installer'"
Write-Host "  4. Release v1.1.0"
Write-Host ""
                       
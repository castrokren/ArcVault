# check-version-sync.ps1
# Verifies VERSION file matches build.ps1, rebuild-and-restart.ps1, and arcvault_installer.py
# Call this at the start of any build script to catch version drift early.

param([string]$Expected = "")

$root = Split-Path -Parent $PSScriptRoot
$versionFile = Join-Path $root "VERSION"

if (-not (Test-Path $versionFile)) {
    Write-Host "ERROR: VERSION file not found at $versionFile" -ForegroundColor Red
    exit 1
}

$canonical = (Get-Content $versionFile -Raw).Trim()
if ($Expected -and $Expected -ne $canonical) {
    Write-Host "ERROR: Script version '$Expected' does not match VERSION file '$canonical'" -ForegroundColor Red
    Write-Host "Update this script or run: Set-Content '$versionFile' '$Expected'" -ForegroundColor Yellow
    exit 1
}

# Check build.ps1
$buildPs1 = Join-Path $root "scripts\build.ps1"
if (Test-Path $buildPs1) {
    $match = Select-String -Path $buildPs1 -Pattern '^\$Version\s*=\s*"(.+)"' | Select-Object -First 1
    if ($match) {
        $v = $match.Matches[0].Groups[1].Value
        if ($v -ne $canonical) {
            Write-Host "VERSION MISMATCH: scripts/build.ps1 has '$v', VERSION file has '$canonical'" -ForegroundColor Red
            Write-Host "Fix: change `$Version in scripts/build.ps1 to '$canonical'" -ForegroundColor Yellow
            exit 1
        }
    }
}

# Check rebuild-and-restart.ps1
$rebuildPs1 = Join-Path $root "scripts\rebuild-and-restart.ps1"
if (Test-Path $rebuildPs1) {
    $match = Select-String -Path $rebuildPs1 -Pattern '^\$Version\s*=\s*"(.+)"' | Select-Object -First 1
    if ($match) {
        $v = $match.Matches[0].Groups[1].Value
        if ($v -ne $canonical) {
            Write-Host "VERSION MISMATCH: scripts/rebuild-and-restart.ps1 has '$v', VERSION file has '$canonical'" -ForegroundColor Red
            Write-Host "Fix: change `$Version in scripts/rebuild-and-restart.ps1 to '$canonical'" -ForegroundColor Yellow
            exit 1
        }
    }
}

# Check arcvault_installer.py
$installerPy = Join-Path $root "installer\windows\arcvault_installer.py"
if (Test-Path $installerPy) {
    $match = Select-String -Path $installerPy -Pattern 'self\.version\s*=\s*"(.+)"' | Select-Object -First 1
    if ($match) {
        $pyVer = "v" + $match.Matches[0].Groups[1].Value  # installer uses "0.5.1", VERSION uses "v0.5.1"
        if ($pyVer -ne $canonical) {
            Write-Host "VERSION MISMATCH: arcvault_installer.py has '$pyVer', VERSION file has '$canonical'" -ForegroundColor Red
            Write-Host "Fix: change self.version in installer/windows/arcvault_installer.py to '$($canonical.TrimStart('v'))'" -ForegroundColor Yellow
            exit 1
        }
    }
}

Write-Host "  Version sync OK: $canonical" -ForegroundColor Green

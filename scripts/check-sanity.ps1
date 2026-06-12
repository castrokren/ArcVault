#!/usr/bin/env pwsh
# check-sanity.ps1
# Smoke-test ArcVault to catch the three recurring regressions:
#
#   1. /downloads/installer serving bootstrap.ps1 instead of the .exe
#   2. Services unable to start (binary missing, wrong version, or bad config)
#   3. Coordinator binary built with wrong version (e.g. "2.0" instead of VERSION)
#
# Usage:
#   .\scripts\check-sanity.ps1                  # offline checks + live if services are up
#   .\scripts\check-sanity.ps1 -SkipLive        # offline/binary checks only
#   .\scripts\check-sanity.ps1 -AdminToken tok  # override admin token for live checks
#
# Run this after every deploy. It is intentionally fast (< 5 seconds).

param(
    [switch]$SkipLive,
    [string]$AdminToken = ""
)

$ErrorActionPreference = "Continue"
$Root      = Split-Path -Parent $PSScriptRoot
$Passed    = 0
$Failed    = 0
$Warnings  = 0

function Pass($msg)    { Write-Host "  [PASS] $msg" -ForegroundColor Green;  $script:Passed++  }
function Fail($msg)    { Write-Host "  [FAIL] $msg" -ForegroundColor Red;    $script:Failed++  }
function Warn($msg)    { Write-Host "  [WARN] $msg" -ForegroundColor Yellow; $script:Warnings++ }
function Section($msg) { Write-Host "`n=== $msg ===" -ForegroundColor Cyan }

# ─────────────────────────────────────────────────────────────────────────────
Section "1. VERSION FILE"
# ─────────────────────────────────────────────────────────────────────────────

$versionFile = Join-Path $Root "VERSION"
if (-not (Test-Path $versionFile)) {
    Fail "VERSION file missing at $versionFile"
} else {
    $canonical = (Get-Content $versionFile -Raw).Trim()
    if ($canonical -match "^v\d+\.\d+\.\d+$") {
        Pass "VERSION file exists: $canonical"
    } else {
        Fail "VERSION file has unexpected format: '$canonical' (expected vX.Y.Z)"
    }
}

# ─────────────────────────────────────────────────────────────────────────────
Section "2. BINARY VERSION CHECK  [REGRESSION: coordinator built as 2.0]"
# ─────────────────────────────────────────────────────────────────────────────
# Guard against: binary built without -ldflags, or with wrong version string.

$coordExe = "C:\ArcVault\coordinator.exe"
if (-not (Test-Path $coordExe)) {
    Warn "coordinator.exe not found at $coordExe — skipping binary version check"
} else {
    try {
        $binVersion = & $coordExe --version 2>&1
        $binVersion = ($binVersion -join "").Trim()

        # Check 1: must not be "2.0" or "v2.0" (project folder name leaked in)
        if ($binVersion -eq "2.0" -or $binVersion -eq "v2.0") {
            Fail "coordinator.exe reports version '$binVersion' — the project FOLDER name was baked in instead of the real version.`n       Fix: rebuild with  -ldflags `"-X main.Version=`$Version`""
        }
        # Check 2: must not be "dev" (ldflags were dropped)
        elseif ($binVersion -eq "dev" -or $binVersion -eq "") {
            Fail "coordinator.exe reports version '$binVersion' — ldflags were not applied at build time.`n       Fix: go build -ldflags `"-X main.Version=`$Version -X arcvault/coordinator/server.Version=`$Version`" -o coordinator.exe .\coordinator"
        }
        # Check 3: must match VERSION file
        elseif ($canonical -and $binVersion -ne $canonical) {
            Fail "coordinator.exe reports '$binVersion' but VERSION file has '$canonical' — version mismatch.`n       Fix: rebuild or update VERSION file"
        }
        else {
            Pass "coordinator.exe version: $binVersion"
        }
    } catch {
        Warn "Could not run coordinator.exe --version: $_"
    }
}

# Same for agent.exe
$agentExe = "C:\ArcVault-Agent\agent.exe"
if (-not (Test-Path $agentExe)) {
    Warn "agent.exe not found at $agentExe — skipping agent binary version check"
} else {
    try {
        $agentVer = & $agentExe --version 2>&1
        $agentVer = ($agentVer -join "").Trim()
        if ($agentVer -eq "2.0" -or $agentVer -eq "v2.0" -or $agentVer -eq "dev" -or $agentVer -eq "") {
            Fail "agent.exe reports bad version '$agentVer' — ldflags dropped or wrong version"
        } elseif ($canonical -and $agentVer -ne $canonical) {
            Fail "agent.exe reports '$agentVer' but VERSION file has '$canonical'"
        } else {
            Pass "agent.exe version: $agentVer"
        }
    } catch {
        Warn "Could not run agent.exe --version: $_"
    }
}

# ─────────────────────────────────────────────────────────────────────────────
Section "3. SERVICE STATUS  [REGRESSION: services unable to start]"
# ─────────────────────────────────────────────────────────────────────────────

foreach ($svc in @("arcvault-coordinator", "arcvault-agent")) {
    $q = sc.exe query $svc 2>&1
    if ($q -match "STATE.*RUNNING") {
        Pass "$svc is RUNNING"
    } elseif ($q -match "STOPPED") {
        Warn "$svc is STOPPED — try: sc.exe start $svc"
    } elseif ($q -match "does not exist|1060") {
        Warn "$svc not installed"
    } else {
        Fail "$svc query failed: $($q -join ' ')"
    }
}

# Config sanity: run-service arg must be present in the service binary path
$scQc = sc.exe qc arcvault-coordinator 2>&1
if ($scQc -match "BINARY_PATH_NAME") {
    $binaryPath = ($scQc | Select-String "BINARY_PATH_NAME").ToString()
    if ($binaryPath -notmatch "run-service") {
        Fail "arcvault-coordinator service is NOT registered with 'run-service' arg.`n       Windows SCM requires this. The service will start then immediately exit (Error 1053).`n       Fix: uninstall and reinstall the service via the ArcVault installer."
    } else {
        Pass "arcvault-coordinator binary path contains 'run-service' arg"
    }
}

# ─────────────────────────────────────────────────────────────────────────────
Section "4. CONFIG FILE CHECKS"
# ─────────────────────────────────────────────────────────────────────────────

$coordCfg = "C:\ArcVault\config.json"
if (-not (Test-Path $coordCfg)) {
    Warn "C:\ArcVault\config.json not found — coordinator will fail to start"
} else {
    try {
        $cfg = Get-Content $coordCfg -Raw | ConvertFrom-Json
        Pass "C:\ArcVault\config.json is valid JSON"

        if (-not $cfg.admin_token) { Fail "config.json missing admin_token" }
        else { Pass "admin_token is set" }

        if (-not $cfg.port)        { Warn "config.json missing port (will default to 8080)" }
        else { Pass "port = $($cfg.port)" }

        # Installer dir check
        if ($cfg.installer_dir) {
            if (-not (Test-Path $cfg.installer_dir)) {
                Fail "installer_dir '$($cfg.installer_dir)' does not exist — /downloads/installer will return 404"
            } else {
                $ver = if ($canonical) { $canonical.TrimStart('v') } else { "?" }
                $installerExe = Join-Path $cfg.installer_dir "ArcVault-Setup-$ver-windows-amd64.exe"
                if (Test-Path $installerExe) {
                    Pass "installer .exe found: $installerExe"
                } else {
                    Warn "installer .exe NOT found at $installerExe — run scripts\build.ps1 to build it"
                }
            }
        } else {
            Warn "config.json missing installer_dir — /downloads/installer will look in C:\ArcVault\ by default"
        }

        # Grab admin token for live checks
        if (-not $AdminToken -and $cfg.admin_token) {
            $AdminToken = $cfg.admin_token
        }
    } catch {
        Fail "config.json is not valid JSON: $_"
    }
}

# ─────────────────────────────────────────────────────────────────────────────
Section "5. LIVE ENDPOINT CHECKS  [REGRESSION: /downloads/installer serves .ps1]"
# ─────────────────────────────────────────────────────────────────────────────

if ($SkipLive) {
    Write-Host "  (skipped — -SkipLive flag set)" -ForegroundColor Gray
} elseif (-not $AdminToken) {
    Warn "No admin token available — skipping live endpoint checks. Pass -AdminToken or ensure config.json is readable."
} else {
    # Derive coordinator URL from config
    $scheme = "http"; $port = 8080
    if ($cfg) {
        if ($cfg.port) { $port = $cfg.port }
        if ($cfg.cert_file -and $cfg.key_file) { $scheme = "https" }
        elseif ($cfg.external_tls) { $scheme = "https" }
    }
    $base = "${scheme}://localhost:${port}"

    # Health check
    try {
        $health = Invoke-RestMethod -Uri "$base/health" -TimeoutSec 5 -SkipCertificateCheck -ErrorAction Stop
        if ($health.status -eq "ok") { Pass "Health endpoint: ok" }
        else { Fail "Health endpoint returned unexpected: $($health | ConvertTo-Json -Compress)" }
    } catch {
        Fail "Health endpoint unreachable at $base/health — is coordinator running? Error: $_"
    }

    # ── THE KEY REGRESSION: /downloads/installer must serve .exe, not .ps1 ──
    try {
        $headers = @{ Authorization = "Bearer $AdminToken" }
        # Use WebRequest so we can inspect headers without downloading the whole file
        $resp = Invoke-WebRequest -Uri "$base/downloads/installer" `
            -Headers $headers -Method Get -TimeoutSec 10 -SkipCertificateCheck `
            -MaximumRedirection 0 -ErrorAction Stop

        $cd   = $resp.Headers["Content-Disposition"]
        $ct   = $resp.Headers["Content-Type"]

        if ($cd -match "bootstrap\.ps1") {
            Fail "[REGRESSION] /downloads/installer is serving bootstrap.ps1!`n       Fix: in server/server.go registerRoutes(), change:`n         GET /downloads/installer → handleDownloadInstaller  (NOT handleBootstrapScript)`n       Current Content-Disposition: $cd"
        } elseif ($cd -match "\.exe") {
            Pass "/downloads/installer serves .exe: $cd"
        } elseif ($resp.StatusCode -eq 404) {
            Warn "/downloads/installer returned 404 — run scripts\build.ps1 to build the installer .exe"
        } else {
            Warn "/downloads/installer Content-Disposition: '$cd' — verify this is correct"
        }

        if ($ct -eq "text/plain") {
            Fail "[REGRESSION] /downloads/installer Content-Type is text/plain — route is wired to the wrong handler (should be application/octet-stream)"
        }
    } catch {
        $statusCode = $_.Exception.Response.StatusCode.value__
        if ($statusCode -eq 404) {
            Warn "/downloads/installer returned 404 — installer .exe not built yet. Run scripts\build.ps1"
        } elseif ($statusCode -eq 401 -or $statusCode -eq 403) {
            Warn "/downloads/installer returned $statusCode — admin token may be wrong"
        } else {
            Warn "Could not check /downloads/installer: $_"
        }
    }

    # /api/version — make sure version matches
    try {
        $versionResp = Invoke-RestMethod -Uri "$base/api/version" `
            -Headers $headers -TimeoutSec 5 -SkipCertificateCheck -ErrorAction SilentlyContinue
        if ($versionResp.version) {
            if ($canonical -and $versionResp.version -ne $canonical) {
                Fail "Running coordinator reports version '$($versionResp.version)' but VERSION file has '$canonical' — needs rebuild"
            } elseif ($versionResp.version -eq "2.0" -or $versionResp.version -eq "dev") {
                Fail "[REGRESSION] Running coordinator reports version '$($versionResp.version)' — wrong version baked into binary"
            } else {
                Pass "Running coordinator version: $($versionResp.version)"
            }
        }
    } catch {
        # Non-fatal — version endpoint requires auth which may not work in all test contexts
    }
}

# ─────────────────────────────────────────────────────────────────────────────
Write-Host ""
Write-Host "══════════════════════════════════════════════" -ForegroundColor Cyan
$total = $Passed + $Failed + $Warnings
Write-Host "Results: $Passed passed, $Failed failed, $Warnings warnings ($total checks)" -ForegroundColor Cyan
if ($Failed -gt 0) {
    Write-Host "SANITY CHECK FAILED — fix the issues above before deploying." -ForegroundColor Red
    exit 1
} elseif ($Warnings -gt 0) {
    Write-Host "Sanity check passed with warnings." -ForegroundColor Yellow
    exit 0
} else {
    Write-Host "All checks passed." -ForegroundColor Green
    exit 0
}

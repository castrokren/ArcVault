#Requires -Version 5.1
<#
.SYNOPSIS
  Repairs the coordinator service's environment so SCM actually injects
  ARCVAULT_JWT_SECRET and ARCVAULT_CREDENTIAL_KEY.

.DESCRIPTION
  Installer builds before 2026-07-24 wrote these secrets with
      reg add ...\<service>\Environment /v NAME /d VALUE
  which creates a SUBKEY named Environment holding REG_SZ values. SCM reads
  something different: a REG_MULTI_SZ value named `Environment` sitting directly
  ON the service key, holding NAME=value strings. The subkey is never read, so:

    * the coordinator generated a random JWT secret on every start, dropping
      every dashboard session and making logout-revocation meaningless
    * loadCredentialKey() fell back to credential_key in config.json, which sits
      in the same directory as arcvault.db

  This script reads the secrets from wherever they currently are, writes them in
  the shape SCM reads, verifies the readback, and restarts the coordinator.

  It does NOT remove credential_key from config.json by default. Run again with
  -RemoveDiskKey only after a restart has proven the environment is being
  injected -- removing it while the env var is not working leaves the coordinator
  unable to decrypt stored credentials (503 on credential endpoints).

  Secrets are never printed. Fingerprints are the first 8 hex of SHA-256.

.PARAMETER RemoveDiskKey
  Second-pass switch: deletes credential_key from config.json. Only proceeds if
  the registry value verifies AND its fingerprint matches the on-disk key.

.PARAMETER SkipRestart
  Write and verify the registry, but leave the service alone.

.EXAMPLE
  .\scripts\repair-service-env.ps1
.EXAMPLE
  .\scripts\repair-service-env.ps1 -RemoveDiskKey
#>
[CmdletBinding()]
param(
    [switch]$RemoveDiskKey,
    [switch]$SkipRestart
)

$ErrorActionPreference = 'Stop'

$ServiceName = 'arcvault-coordinator'
$ServiceKey  = "HKLM:\SYSTEM\CurrentControlSet\Services\$ServiceName"
$LegacyKey   = "$ServiceKey\Environment"
$ConfigPath  = 'C:\ArcVault\config.json'
$LogPath     = 'C:\ArcVault\coordinator-service.log'

function Fingerprint([string]$s) {
    if (-not $s) { return '(none)' }
    $sha = [System.Security.Cryptography.SHA256]::Create()
    $h = $sha.ComputeHash([Text.Encoding]::UTF8.GetBytes($s))
    return (($h | ForEach-Object { $_.ToString('x2') }) -join '').Substring(0, 8)
}

Write-Host ""
Write-Host "=== ArcVault service-environment repair ===" -ForegroundColor Cyan

# --- 0. Elevation. Writing HKLM and restarting a service both require it. ---
$isAdmin = ([Security.Principal.WindowsPrincipal][Security.Principal.WindowsIdentity]::GetCurrent()).IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)
if (-not $isAdmin) {
    Write-Host "ERROR: must run from an elevated (Admin) PowerShell." -ForegroundColor Red
    Write-Host "  Start-Process powershell -Verb RunAs" -ForegroundColor Yellow
    exit 1
}
if (-not (Test-Path $ServiceKey)) {
    Write-Host "ERROR: service key not found: $ServiceKey" -ForegroundColor Red
    Write-Host "       Is $ServiceName installed?" -ForegroundColor Yellow
    exit 1
}

# --- 1. Gather the secrets from every place they might be. ---
Write-Host ""
Write-Host "Step 1: Locating existing secrets..." -ForegroundColor Yellow

# Correct shape, if a previous run already wrote it.
$current = @{}
$svcItem = Get-Item $ServiceKey
if ($svcItem.GetValueNames() -contains 'Environment') {
    foreach ($entry in (Get-ItemProperty -Path $ServiceKey -Name Environment).Environment) {
        $i = $entry.IndexOf('=')
        if ($i -gt 0) { $current[$entry.Substring(0, $i)] = $entry.Substring($i + 1) }
    }
    Write-Host "  Existing REG_MULTI_SZ Environment value found: $($current.Keys -join ', ')" -ForegroundColor Green
} else {
    Write-Host "  No REG_MULTI_SZ Environment value on the service key (this is the bug)." -ForegroundColor Yellow
}

# Legacy wrong shape written by the old installer.
$legacy = @{}
if (Test-Path $LegacyKey) {
    $li = Get-Item $LegacyKey
    foreach ($n in $li.GetValueNames()) { $legacy[$n] = $li.GetValue($n) }
    Write-Host "  Legacy Environment SUBKEY found (never injected by SCM): $($legacy.Keys -join ', ')" -ForegroundColor Yellow
} else {
    Write-Host "  No legacy Environment subkey." -ForegroundColor Gray
}

# config.json — authoritative for the credential key, because loadCredentialKey()
# prefers cfg.CredentialKey, so THIS is the key that encrypted existing rows.
$diskKey = ''
if (Test-Path $ConfigPath) {
    $cfg = Get-Content $ConfigPath -Raw | ConvertFrom-Json
    if ($cfg.PSObject.Properties.Name -contains 'credential_key') { $diskKey = $cfg.credential_key }
}

# --- 2. Decide the authoritative values. ---
Write-Host ""
Write-Host "Step 2: Choosing values..." -ForegroundColor Yellow

# CREDENTIAL KEY: whatever has actually been decrypting data wins. That is
# config.json if present (the coordinator prefers it), else the legacy subkey,
# else an already-correct env value. Never generate a new one -- a different key
# makes every stored credential_profiles row permanently undecryptable.
$credKey = ''
foreach ($candidate in @($diskKey, $legacy['ARCVAULT_CREDENTIAL_KEY'], $current['ARCVAULT_CREDENTIAL_KEY'])) {
    if ($candidate -and $candidate.Length -eq 64) { $credKey = $candidate; break }
}
if (-not $credKey) {
    Write-Host "  ERROR: no 64-char credential key found on disk or in the registry." -ForegroundColor Red
    Write-Host "         Refusing to generate one: a new key cannot decrypt existing" -ForegroundColor Yellow
    Write-Host "         credential_profiles rows. Recover the original key first." -ForegroundColor Yellow
    exit 1
}
Write-Host "  credential key: fingerprint $(Fingerprint $credKey)" -ForegroundColor Green
if ($diskKey -and $legacy['ARCVAULT_CREDENTIAL_KEY'] -and $diskKey -ne $legacy['ARCVAULT_CREDENTIAL_KEY']) {
    Write-Host "  NOTE: on-disk and legacy-registry credential keys DIFFER." -ForegroundColor Yellow
    Write-Host "        Using the on-disk one ($(Fingerprint $diskKey)) - that is the one the" -ForegroundColor Yellow
    Write-Host "        coordinator has been using, so it is the one that encrypted your data." -ForegroundColor Yellow
}

# JWT SECRET: any stable 64-hex value works (nothing persistent has been in use),
# but reuse an existing one so currently-valid sessions survive where possible.
$jwtSecret = ''
foreach ($candidate in @($current['ARCVAULT_JWT_SECRET'], $legacy['ARCVAULT_JWT_SECRET'])) {
    if ($candidate -and $candidate.Length -eq 64) { $jwtSecret = $candidate; break }
}
if (-not $jwtSecret) {
    $bytes = New-Object byte[] 32
    [System.Security.Cryptography.RandomNumberGenerator]::Create().GetBytes($bytes)
    $jwtSecret = (($bytes | ForEach-Object { $_.ToString('x2') }) -join '')
    Write-Host "  jwt secret: generated new, fingerprint $(Fingerprint $jwtSecret)" -ForegroundColor Green
    Write-Host "              (existing sessions will be invalidated once - the last time)" -ForegroundColor Gray
} else {
    Write-Host "  jwt secret: reusing existing, fingerprint $(Fingerprint $jwtSecret)" -ForegroundColor Green
}

# --- 3. Back up before touching anything. ---
Write-Host ""
Write-Host "Step 3: Backing up..." -ForegroundColor Yellow
$stamp = Get-Date -Format 'yyyyMMdd-HHmmss'
if (Test-Path $ConfigPath) {
    Copy-Item $ConfigPath "$ConfigPath.bak-$stamp"
    Write-Host "  config.json -> config.json.bak-$stamp" -ForegroundColor Green
}
$regBackup = "C:\ArcVault\service-key-backup-$stamp.reg"
& reg.exe export "HKLM\SYSTEM\CurrentControlSet\Services\$ServiceName" $regBackup /y | Out-Null
if (Test-Path $regBackup) {
    Write-Host "  service key -> $regBackup" -ForegroundColor Green
    Write-Host "  (contains the secrets in plaintext - delete it once you are done)" -ForegroundColor Yellow
}

# --- 4. Write the REG_MULTI_SZ value SCM reads. ---
Write-Host ""
Write-Host "Step 4: Writing REG_MULTI_SZ Environment value..." -ForegroundColor Yellow
$merged = @{}
foreach ($k in $current.Keys) { $merged[$k] = $current[$k] }
$merged['ARCVAULT_CREDENTIAL_KEY'] = $credKey
$merged['ARCVAULT_JWT_SECRET']     = $jwtSecret
$payload = @($merged.Keys | Sort-Object | ForEach-Object { "$_=$($merged[$_])" })

New-ItemProperty -Path $ServiceKey -Name 'Environment' -PropertyType MultiString -Value $payload -Force | Out-Null
Write-Host "  wrote $($payload.Count) entries: $(($merged.Keys | Sort-Object) -join ', ')" -ForegroundColor Green

# --- 5. Verify the readback. A silent failure here is the whole bug. ---
Write-Host ""
Write-Host "Step 5: Verifying readback..." -ForegroundColor Yellow
$verify = @{}
$svcItem = Get-Item $ServiceKey
if ($svcItem.GetValueNames() -notcontains 'Environment') {
    Write-Host "  FAILED: no Environment value after write." -ForegroundColor Red
    exit 1
}
if ($svcItem.GetValueKind('Environment') -ne 'MultiString') {
    Write-Host "  FAILED: Environment is $($svcItem.GetValueKind('Environment')), expected MultiString." -ForegroundColor Red
    exit 1
}
foreach ($entry in (Get-ItemProperty -Path $ServiceKey -Name Environment).Environment) {
    $i = $entry.IndexOf('=')
    if ($i -gt 0) { $verify[$entry.Substring(0, $i)] = $entry.Substring($i + 1) }
}
$ok = $true
foreach ($name in @('ARCVAULT_CREDENTIAL_KEY', 'ARCVAULT_JWT_SECRET')) {
    if ($verify[$name] -eq $merged[$name]) {
        Write-Host "  [OK]   $name reads back (fingerprint $(Fingerprint $verify[$name]))" -ForegroundColor Green
    } else {
        Write-Host "  [FAIL] $name did not read back correctly" -ForegroundColor Red
        $ok = $false
    }
}
if (-not $ok) {
    Write-Host "  Aborting before restart. Registry restore: reg import $regBackup" -ForegroundColor Red
    exit 1
}

# --- 6. Restart so SCM injects the new environment. ---
if ($SkipRestart) {
    Write-Host ""
    Write-Host "Step 6: SKIPPED (-SkipRestart). The running process still has the old env." -ForegroundColor Yellow
} else {
    Write-Host ""
    Write-Host "Step 6: Restarting $ServiceName..." -ForegroundColor Yellow
    $marker = (Get-Date)
    & sc.exe stop $ServiceName | Out-Null
    Start-Sleep -Seconds 3
    & sc.exe start $ServiceName | Out-Null
    Start-Sleep -Seconds 5

    $status = (Get-Service $ServiceName).Status
    if ($status -ne 'Running') {
        Write-Host "  ERROR: service is $status, not Running. Check $LogPath" -ForegroundColor Red
        Write-Host "  Registry restore if needed: reg import $regBackup" -ForegroundColor Yellow
        exit 1
    }
    Write-Host "  service is Running." -ForegroundColor Green

    # THE actual proof: this line must NOT appear for the new start.
    Write-Host ""
    Write-Host "Step 7: Confirming the secret is now injected..." -ForegroundColor Yellow
    $fresh = @(Get-Content $LogPath -Tail 60 | Select-String -Pattern 'Generated new JWTSecret|loaded from ARCVAULT_JWT_SECRET')
    $generated = @($fresh | Where-Object { $_.Line -match 'Generated new JWTSecret' })
    $loaded    = @($fresh | Where-Object { $_.Line -match 'loaded from ARCVAULT_JWT_SECRET' })

    if ($loaded.Count -gt 0 -and $generated.Count -eq 0) {
        Write-Host "  [PASS] log says the JWT secret came from the environment." -ForegroundColor Green
    } elseif ($generated.Count -gt 0) {
        Write-Host "  [FAIL] log still says 'Generated new JWTSecret' - env not injected." -ForegroundColor Red
        Write-Host "         Do NOT run -RemoveDiskKey. Investigate before proceeding." -ForegroundColor Yellow
        exit 1
    } else {
        Write-Host "  [??]   no matching log line in the last 60 lines. Check manually:" -ForegroundColor Yellow
        Write-Host "         Select-String -Path $LogPath -Pattern JWTSecret | Select-Object -Last 5" -ForegroundColor Gray
    }
}

# --- 8. Optional second pass: take the key off disk. ---
if ($RemoveDiskKey) {
    Write-Host ""
    Write-Host "Step 8: Removing credential_key from config.json..." -ForegroundColor Yellow
    if (-not $diskKey) {
        Write-Host "  Already absent - nothing to do." -ForegroundColor Green
    } elseif ($verify['ARCVAULT_CREDENTIAL_KEY'] -ne $diskKey) {
        Write-Host "  REFUSING: the registry key does not match the on-disk key." -ForegroundColor Red
        Write-Host "  Removing it would leave the coordinator unable to decrypt." -ForegroundColor Yellow
        exit 1
    } else {
        $cfg = Get-Content $ConfigPath -Raw | ConvertFrom-Json
        $cfg.PSObject.Properties.Remove('credential_key')
        ($cfg | ConvertTo-Json -Depth 10) | Set-Content -Path $ConfigPath -Encoding utf8
        Write-Host "  removed. Backup at $ConfigPath.bak-$stamp" -ForegroundColor Green
        Write-Host "  Restart once more, then confirm credential endpoints still work:" -ForegroundColor Yellow
        Write-Host "    a credential-profile POST returning 503 means the key is not reaching the process." -ForegroundColor Gray
    }
} else {
    Write-Host ""
    Write-Host "credential_key is still in config.json (deliberately)." -ForegroundColor Yellow
    Write-Host "Once the PASS above is confirmed, re-run with -RemoveDiskKey to take it off disk." -ForegroundColor Yellow
}

Write-Host ""
Write-Host "=== Done ===" -ForegroundColor Cyan
Write-Host "Delete the plaintext registry backup when finished: $regBackup" -ForegroundColor Yellow

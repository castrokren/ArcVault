#Requires -RunAsAdministrator
# =====================================================================
# ArcVault Agent Bootstrap
# This is the TEMPLATE emitted by coordinator/internal/bootstrap.GenerateScript.
# {{...}} markers are filled by Go at generation time. Everything else is literal.
#
# TARGET: Windows PowerShell 5.1 (powershell.exe) — the default on every box.
#         Also runs on PowerShell 7+ (pwsh) via the Core branch below.
# RUN:    Elevated prompt:  powershell -ExecutionPolicy Bypass -File .\bootstrap.ps1
# =====================================================================

$ErrorActionPreference = 'Stop'

# --- 0. Force TLS 1.2 (PS 5.1 defaults to SSL3/TLS1.0 and will fail HTTPS) ---
[Net.ServicePointManager]::SecurityProtocol = `
    [Net.SecurityProtocolType]::Tls12 -bor [Net.ServicePointManager]::SecurityProtocol

# --- Generation-time values (interpolated by Go) ---
$CoordinatorUrl = '{{CoordinatorURL}}'   # e.g. https://192.168.1.10  (no :443)
$AgentToken     = '{{AgentToken}}'       # role=agent token, minted per script
$PinnedThumb    = '{{CertThumbprint}}'   # SHA-1, UPPERCASE hex, no separators
$ExpectedHash   = '{{AgentExeSHA256}}'   # UPPERCASE SHA-256 of agent.exe

# Coordinator cert — single-quoted here-string so $ and ` never expand.
$CertPem = @'
{{CertPEM}}
'@

# --- Paths ---
$InstallDir  = 'C:\ArcVault-Agent'
$CertPath    = Join-Path $InstallDir 'coordinator.crt'
$AgentExe    = Join-Path $InstallDir 'agent.exe'
$ConfigPath  = Join-Path $InstallDir 'agent-config.yaml'
$ServiceName = 'arcvault-agent'

New-Item -ItemType Directory -Force -Path $InstallDir | Out-Null

# --- 1. Write the pinned cert FIRST, before any download (cert-first order) ---
Set-Content -Path $CertPath -Value $CertPem -Encoding Ascii

# --- 2. Download agent.exe over HTTPS ---
# Integrity is guaranteed by the MANDATORY SHA-256 check in step 3, regardless
# of transport. TLS pinning below is defence-in-depth on Windows PowerShell 5.1.
$dlUri   = "$CoordinatorUrl/downloads/agent.exe"
$headers = @{ Authorization = "Bearer $AgentToken" }

if ($PSVersionTable.PSEdition -eq 'Core') {
    # PS 7+: ServicePointManager callback is ignored by the HttpClient-based
    # cmdlets, so skip the TLS check here. The SHA-256 check is the guarantee.
    Invoke-WebRequest -Uri $dlUri -Headers $headers -OutFile $AgentExe `
        -SkipCertificateCheck -UseBasicParsing
}
else {
    # Windows PowerShell 5.1: pin to the coordinator cert by thumbprint.
    $global:ArcVaultPinnedThumb = $PinnedThumb
    [Net.ServicePointManager]::ServerCertificateValidationCallback = {
        param($s, $cert, $chain, $errors)
        return ($cert.GetCertHashString() -eq $global:ArcVaultPinnedThumb)
    }
    try {
        Invoke-WebRequest -Uri $dlUri -Headers $headers -OutFile $AgentExe `
            -UseBasicParsing
    }
    finally {
        # Always restore default validation — never leave it disabled.
        [Net.ServicePointManager]::ServerCertificateValidationCallback = $null
    }
}

# --- 3. MANDATORY integrity check (blocks tampered or truncated downloads) ---
$actual = (Get-FileHash -Path $AgentExe -Algorithm SHA256).Hash.ToUpper()
if ($actual -ne $ExpectedHash.ToUpper()) {
    throw "agent.exe integrity check failed: expected $ExpectedHash, got $actual"
}

# --- 4. Write agent config ---
# NOTE: agent_id = $env:COMPUTERNAME. Collides on cloned/imaged VMs — acceptable
# for the current dev fleet; revisit if onboarding imaged machines.
$config = @"
agent_id: $env:COMPUTERNAME
coordinator_url: $CoordinatorUrl
auth_token: $AgentToken
ca_cert_file: $CertPath
"@
Set-Content -Path $ConfigPath -Value $config -Encoding Ascii

# --- 5. Idempotent service (re)install ---
# Re-running on an onboarded box: stop + delete + wait for it to truly go away
# (exit 1060 = does not exist) before reinstalling. Mirrors the Session-15
# installer fix for "marked for deletion" (1072).
sc.exe query $ServiceName 2>$null | Out-Null
if ($LASTEXITCODE -eq 0) {
    Write-Host "Existing $ServiceName found - stopping and removing first..."
    sc.exe stop   $ServiceName 2>$null | Out-Null
    sc.exe delete $ServiceName 2>$null | Out-Null
    for ($i = 0; $i -lt 15; $i++) {
        sc.exe query $ServiceName 2>$null | Out-Null
        if ($LASTEXITCODE -eq 1060) { break }   # 1060 = service does not exist
        Start-Sleep -Seconds 1
    }
    Start-Sleep -Seconds 2   # buffer after confirmed gone
}

# --- 6. Install, start, and set SCM failure recovery ---
& $AgentExe install-service
sc.exe start   $ServiceName
sc.exe failure $ServiceName reset=86400 actions=restart/3000/restart/3000/restart/3000

Write-Host ""
Write-Host "ArcVault agent installed and started on $env:COMPUTERNAME."
Write-Host "Confirm it appears online in the coordinator dashboard at $CoordinatorUrl"

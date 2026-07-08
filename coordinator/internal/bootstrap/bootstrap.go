package bootstrap

import "strings"

// Params holds the values to interpolate into the bootstrap script template.
type Params struct {
	CoordinatorURL string // https://192.168.1.10 (port omitted when 443)
	AgentToken     string // role=agent token, minted per generation
	CertPEM        string // coordinator's cert in PEM format
	CertThumbprint string // SHA-1 hex, UPPERCASE, no separators
	AgentExeSHA256 string // UPPERCASE hex
}

// GenerateScript generates the PowerShell bootstrap script by interpolating
// params into the template.
func GenerateScript(p Params) string {
	s := scriptTemplateString()

	// Use simple string replacements
	s = strings.ReplaceAll(s, "COORDINATOR_URL", p.CoordinatorURL)
	s = strings.ReplaceAll(s, "AGENT_TOKEN", p.AgentToken)
	s = strings.ReplaceAll(s, "CERT_THUMBPRINT", p.CertThumbprint)
	s = strings.ReplaceAll(s, "AGENT_EXE_SHA256", p.AgentExeSHA256)
	s = strings.ReplaceAll(s, "CERT_PEM", p.CertPEM)
	return s
}

// scriptTemplateString returns the bootstrap script template with placeholders.
// Uses simple text placeholders (COORDINATOR_URL, etc) for safe substitution.
func scriptTemplateString() string {
	return `#Requires -RunAsAdministrator
# =====================================================================
# ArcVault Agent Bootstrap
# Emitted by coordinator/internal/bootstrap.GenerateScript.
# Placeholders are filled by Go at generation time. Everything else is literal.
#
# TARGET: Windows PowerShell 5.1 (powershell.exe) — the default on every box.
# RUN:    Elevated prompt:  powershell -ExecutionPolicy Bypass -File .\bootstrap.ps1
# =====================================================================

$ErrorActionPreference = 'Stop'

# --- 0. Force TLS 1.2 (PS 5.1 defaults to SSL3/TLS1.0 and will fail HTTPS) ---
[Net.ServicePointManager]::SecurityProtocol = ` + "`" + `
    [Net.SecurityProtocolType]::Tls12 -bor [Net.ServicePointManager]::SecurityProtocol

# --- Generation-time values (interpolated by Go) ---
$CoordinatorUrl = 'COORDINATOR_URL'   # e.g. https://192.168.1.10  (no :443)
$AgentToken     = 'AGENT_TOKEN'       # role=agent token, minted per script
$PinnedThumb    = 'CERT_THUMBPRINT'   # SHA-1, UPPERCASE hex, no separators
$ExpectedHash   = 'AGENT_EXE_SHA256'  # UPPERCASE SHA-256 of agent.exe

# Coordinator cert — single-quoted here-string so $ and ` + "`" + ` never expand.
$CertPem = @'
CERT_PEM
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

# --- 2. Download agent.exe over HTTPS using curl.exe ---
# Invoke-WebRequest on PS 5.1 (.NET HttpWebRequest) cannot survive the TLS
# renegotiation this server performs and drops the connection. curl.exe (present
# on Windows 10+) handles it. --cacert pins trust to the coordinator cert written
# in step 1; --fail turns an auth/error response into a non-zero exit instead of
# saving an error body as agent.exe. Integrity is still enforced by step 3.
$curl = Join-Path $env:SystemRoot 'System32\curl.exe'
if (-not (Test-Path $curl)) { $curl = 'curl.exe' }  # fall back to PATH
& $curl --cacert $CertPath --fail --silent --show-error ` + "`" + `
    -H "Authorization: Bearer $AgentToken" ` + "`" + `
    -o $AgentExe "$CoordinatorUrl/downloads/agent.exe"
if ($LASTEXITCODE -ne 0) {
    throw "agent.exe download failed (curl exit $LASTEXITCODE)"
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
# (exit 1060 = does not exist) before reinstalling.
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
`
}

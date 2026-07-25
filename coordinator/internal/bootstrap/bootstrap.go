package bootstrap

import "strings"

// Params holds the values to interpolate into the bootstrap script template.
//
// There is deliberately no cert thumbprint here. The script pins trust with
// `curl --cacert` against the embedded CertPEM, which validates the certificate
// itself rather than matching a fingerprint; download integrity is enforced
// separately by the mandatory AgentExeSHA256 check.
type Params struct {
	CoordinatorURL string // https://192.168.1.10 (port omitted when 443)
	AgentToken     string // role=agent token, minted per generation
	CertPEM        string // coordinator's cert in PEM format
	AgentExeSHA256 string // UPPERCASE hex
}

// GenerateScript generates the PowerShell bootstrap script by interpolating
// params into the template.
func GenerateScript(p Params) string {
	s := scriptTemplateString()

	// Use simple string replacements
	s = strings.ReplaceAll(s, "COORDINATOR_URL", p.CoordinatorURL)
	s = strings.ReplaceAll(s, "AGENT_TOKEN", p.AgentToken)
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
$ExpectedHash   = 'AGENT_EXE_SHA256'  # UPPERCASE SHA-256 of agent.exe

# Coordinator cert — single-quoted here-string so $ and ` + "`" + ` never expand.
$CertPem = @'
CERT_PEM
'@

# --- Paths ---
$InstallDir  = 'C:\ArcVault-Agent'
$CertPath    = Join-Path $InstallDir 'coordinator.crt'
$AgentExe    = Join-Path $InstallDir 'agent.exe'
$AgentExeNew = Join-Path $InstallDir 'agent.exe.new'
$ConfigPath  = Join-Path $InstallDir 'agent-config.yaml'
$ServiceName = 'arcvault-agent'

New-Item -ItemType Directory -Force -Path $InstallDir | Out-Null

# --- 1. Write the pinned cert FIRST, before any download (cert-first order) ---
Set-Content -Path $CertPath -Value $CertPem -Encoding Ascii

# --- 2. Download agent.exe to a TEMP path over HTTPS using curl.exe ---
# Downloading straight onto $AgentExe fails on any machine that already has the
# agent: Windows locks a running executable, so curl aborts partway with
# "client returned ERROR on write" (exit 23 = CURLE_WRITE_ERROR) and the script
# died before it ever reached the service-stop below. Fetch to .new instead, so a
# failed or tampered download also leaves a working agent untouched.
# Invoke-WebRequest on PS 5.1 (.NET HttpWebRequest) cannot survive the TLS
# renegotiation this server performs and drops the connection. curl.exe (present
# on Windows 10+) handles it. --cacert pins trust to the coordinator cert written
# in step 1; --fail turns an auth/error response into a non-zero exit instead of
# saving an error body as agent.exe.
$curl = Join-Path $env:SystemRoot 'System32\curl.exe'
if (-not (Test-Path $curl)) { $curl = 'curl.exe' }  # fall back to PATH
if (Test-Path $AgentExeNew) { Remove-Item $AgentExeNew -Force }
& $curl --cacert $CertPath --fail --silent --show-error ` + "`" + `
    -H "Authorization: Bearer $AgentToken" ` + "`" + `
    -o $AgentExeNew "$CoordinatorUrl/downloads/agent.exe"
if ($LASTEXITCODE -ne 0) {
    if (Test-Path $AgentExeNew) { Remove-Item $AgentExeNew -Force }
    throw "agent.exe download failed (curl exit $LASTEXITCODE). 23 = could not write to $AgentExeNew (disk full, or the path is locked/denied); 60 = the coordinator cert is not trusted; 22 = HTTP error such as an expired enrollment token."
}

# --- 3. MANDATORY integrity check, on the temp copy, BEFORE it replaces anything ---
$actual = (Get-FileHash -Path $AgentExeNew -Algorithm SHA256).Hash.ToUpper()
if ($actual -ne $ExpectedHash.ToUpper()) {
    Remove-Item $AgentExeNew -Force
    throw "agent.exe integrity check failed: expected $ExpectedHash, got $actual"
}

# --- 4. Stop and remove any existing service BEFORE touching the binary ---
# This has to happen before the swap, not after: Windows locks a running exe, so
# replacing agent.exe underneath a live service is what made re-running this script
# on an already-onboarded machine fail. Waits for the service to truly go away
# (exit 1060 = does not exist).
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

# --- 5. Swap the verified binary into place ---
# The service is stopped, so the file is no longer locked. Retry briefly anyway:
# the SCM can report a service gone while the process is still exiting.
$swapped = $false
for ($i = 0; $i -lt 10; $i++) {
    try {
        Move-Item -Path $AgentExeNew -Destination $AgentExe -Force -ErrorAction Stop
        $swapped = $true
        break
    } catch {
        Start-Sleep -Seconds 1
    }
}
if (-not $swapped) {
    throw "could not replace $AgentExe - it is still locked. Stop the $ServiceName service and any running agent.exe, then re-run. The verified download is waiting at $AgentExeNew."
}

# --- 6. Write agent config ---
# NOTE: agent_id = $env:COMPUTERNAME. Collides on cloned/imaged VMs — acceptable
# for the current dev fleet; revisit if onboarding imaged machines.
$config = @"
agent_id: $env:COMPUTERNAME
coordinator_url: $CoordinatorUrl
auth_token: $AgentToken
ca_cert_file: $CertPath
"@
Set-Content -Path $ConfigPath -Value $config -Encoding Ascii

# --- 7. Install, start, and set SCM failure recovery ---
& $AgentExe install-service
sc.exe start   $ServiceName
sc.exe failure $ServiceName reset=86400 actions=restart/3000/restart/3000/restart/3000

Write-Host ""
Write-Host "ArcVault agent installed and started on $env:COMPUTERNAME."
Write-Host "Confirm it appears online in the coordinator dashboard at $CoordinatorUrl"
`
}

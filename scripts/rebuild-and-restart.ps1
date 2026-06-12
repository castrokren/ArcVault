# ArcVault Rebuild & Restart Script
# Run from an Admin PowerShell in C:\Projects\ArcVault2.0
# Usage: .\rebuild-and-restart.ps1

$ProjectRoot = Split-Path -Parent $PSScriptRoot
Set-Location $ProjectRoot

Write-Host ""
Write-Host "=== ArcVault Rebuild & Restart ===" -ForegroundColor Cyan
Write-Host "Project root: $ProjectRoot"
Write-Host ""

# Step 1: Stop services
Write-Host "Step 1: Stopping services..." -ForegroundColor Yellow

# Disable SCM auto-restart so coordinator doesn't bounce back up before we can replace the binary
sc.exe failure arcvault-coordinator reset= 0 actions= none 2>$null

sc.exe stop arcvault-agent 2>$null
sc.exe stop arcvault-coordinator 2>$null

# Wait for coordinator process to fully exit so the binary file lock is released
Write-Host "  Waiting for coordinator.exe to exit..."
$waitedProc = 0
while ($waitedProc -lt 15) {
    Start-Sleep -Seconds 1
    $proc = Get-Process -Name "coordinator" -ErrorAction SilentlyContinue
    if (-not $proc) {
        Write-Host "  coordinator.exe has exited." -ForegroundColor Green
        break
    }
    $waitedProc++
}
if ($waitedProc -ge 15) {
    Write-Host "  Force-killing coordinator.exe..." -ForegroundColor Yellow
    Stop-Process -Name "coordinator" -Force -ErrorAction SilentlyContinue
    Start-Sleep -Seconds 2
}

# Also wait for port 8080 to be free
$waited = 0
while ($waited -lt 10) {
    Start-Sleep -Seconds 1
    $portInUse = netstat -ano | Select-String ":$coordPort " | Select-String "LISTENING"
    if (-not $portInUse) { break }
    $waited++
}

# Step 2: Build dashboard
Write-Host ""
Write-Host "Step 2: Building dashboard..." -ForegroundColor Yellow
Set-Location "$ProjectRoot\dashboard"

if (-not (Get-Command npm -ErrorAction SilentlyContinue)) {
    Write-Host "  WARNING: npm not found. Skipping dashboard build." -ForegroundColor Red
} else {
    npm run build
    if ($LASTEXITCODE -ne 0) {
        Write-Host "  Dashboard build FAILED." -ForegroundColor Red
        exit 1
    }
    Write-Host "  Dashboard built successfully." -ForegroundColor Green
}

Set-Location $ProjectRoot

# Step 3: Sync dashboard into coordinator embed folder (REQUIRED before go build)
Write-Host ""
Write-Host "Step 3: Syncing dashboard into coordinator embed folder..." -ForegroundColor Yellow
$embedDist = "$ProjectRoot\coordinator\static\dist"
Remove-Item -Recurse -Force $embedDist -ErrorAction SilentlyContinue
New-Item -ItemType Directory -Force -Path $embedDist | Out-Null
Copy-Item -Recurse "$ProjectRoot\dashboard\dist\*" "$embedDist\" -Force
Write-Host "  Synced dashboard\dist -> coordinator\static\dist" -ForegroundColor Green

# Version — read from VERSION file (single source of truth).
# NEVER hardcode a version string here; edit VERSION to bump.
$Version = (Get-Content "$PSScriptRoot\..\VERSION" -Raw).Trim()
if (-not $Version) { Write-Host "ERROR: VERSION file is empty" -ForegroundColor Red; exit 1 }
& "$PSScriptRoot\check-version-sync.ps1" -Expected $Version
if ($LASTEXITCODE -ne 0) { exit 1 }
Write-Host "  Building with version: $Version" -ForegroundColor Cyan

# Step 4: Build coordinator.exe
Write-Host ""
Write-Host "Step 4: Building coordinator.exe..." -ForegroundColor Yellow

if (-not (Get-Command go -ErrorAction SilentlyContinue)) {
    Write-Host "  ERROR: go not found in PATH." -ForegroundColor Red
    exit 1
}

go build -ldflags "-X main.Version=$Version -X arcvault/coordinator/server.Version=$Version" -o coordinator.exe .\coordinator
if ($LASTEXITCODE -ne 0) {
    Write-Host "  coordinator.exe build FAILED." -ForegroundColor Red
    exit 1
}

# ── GUARDRAIL: verify built binary reports the correct version ────────────────
# This catches missing ldflags, wrong version string, or wrong binary being copied.
$binVer = & ".\coordinator.exe" --version 2>&1
$binVer = ($binVer -join "").Trim()
if ($binVer -ne $Version) {
    Write-Host "  ERROR: coordinator.exe reports version '$binVer' but expected '$Version'" -ForegroundColor Red
    Write-Host "         This means ldflags were not applied, or the wrong binary was built." -ForegroundColor Yellow
    Write-Host "         Do NOT deploy this binary. Fix the build command and try again." -ForegroundColor Yellow
    exit 1
}
Write-Host "  coordinator.exe built successfully (version: $binVer)" -ForegroundColor Green

# Step 5: Build agent.exe
Write-Host ""
Write-Host "Step 5: Building agent.exe..." -ForegroundColor Yellow

go build -ldflags "-X main.Version=$Version" -o agent.exe .\agent
if ($LASTEXITCODE -ne 0) {
    Write-Host "  agent.exe build FAILED." -ForegroundColor Red
    exit 1
}
Write-Host "  agent.exe built successfully." -ForegroundColor Green

# Step 6: Deploy to service locations
Write-Host ""
Write-Host "Step 6: Deploying to service locations..." -ForegroundColor Yellow

# Deploy to live service directories (where the Windows services run from)
if (Test-Path "C:\ArcVault") {
    Copy-Item "coordinator.exe" "C:\ArcVault\coordinator.exe" -Force
    Write-Host "  Copied coordinator.exe -> C:\ArcVault\" -ForegroundColor Green
} else {
    Write-Host "  WARNING: C:\ArcVault not found, skipping live deploy" -ForegroundColor Yellow
}

if (Test-Path "C:\ArcVault-Agent") {
    Copy-Item "agent.exe" "C:\ArcVault-Agent\agent.exe" -Force
    Write-Host "  Copied agent.exe -> C:\ArcVault-Agent\" -ForegroundColor Green
} else {
    Write-Host "  WARNING: C:\ArcVault-Agent not found, skipping live deploy" -ForegroundColor Yellow
}

# Also keep installer\windows\ in sync for building the installer
Copy-Item "coordinator.exe" "$ProjectRoot\installer\windows\coordinator.exe" -Force
Copy-Item "agent.exe" "$ProjectRoot\installer\windows\agent.exe" -Force
Write-Host "  Copied binaries -> installer\windows\" -ForegroundColor Green

# Step 7: Start services
Write-Host ""
Write-Host "Step 7: Starting services..." -ForegroundColor Yellow

# Derive coordinator URL from its actual config - never hardcode protocol or port
$coordCfgPath = "C:\ArcVault\config.json"
$coordPort = 8080
$coordScheme = "http"
if (Test-Path $coordCfgPath) {
    $coordCfg = Get-Content $coordCfgPath -Raw | ConvertFrom-Json
    if ($coordCfg.port) { $coordPort = $coordCfg.port }
    if ($coordCfg.cert_file -and $coordCfg.key_file) { $coordScheme = "https" }
}
$coordBase = "${coordScheme}://localhost:${coordPort}"
Write-Host "  Coordinator URL: $coordBase" -ForegroundColor Cyan

sc.exe start arcvault-coordinator

# Re-enable SCM auto-restart now that the new binary is in place
sc.exe failure arcvault-coordinator reset= 86400 actions= restart/3000/restart/3000/restart/3000 2>$null
Write-Host "  SCM failure recovery re-enabled." -ForegroundColor Green

# Wait for service to reach RUNNING state (avoids TLS cert issues with health check)
Write-Host "  Waiting for coordinator service to reach RUNNING state..."
$svcRunning = $false
for ($i = 1; $i -le 15; $i++) {
    Start-Sleep -Seconds 2
    $svcQuery = sc.exe query arcvault-coordinator
    if ($svcQuery -match "STATE.*RUNNING") {
        Write-Host "  Coordinator service is RUNNING." -ForegroundColor Green
        $svcRunning = $true
        break
    }
    if ($i % 3 -eq 0) { Write-Host "  Still starting... ($($i*2)s)" }
}
if (-not $svcRunning) {
    Write-Host "  ERROR: Coordinator service did not reach RUNNING state." -ForegroundColor Red
    Write-Host "  Diagnose: C:\ArcVault\coordinator.exe start" -ForegroundColor Yellow
    exit 1
}

# Step 8: Regenerate agent token (always - coordinator reinstall wipes tokens)
Write-Host ""
Write-Host "Step 8: Regenerating agent token..." -ForegroundColor Yellow

$agentConfigPath = "C:\ArcVault-Agent\agent-config.yaml"
$coordConfigPath = "C:\ArcVault\config.json"

if ((Test-Path $agentConfigPath) -and (Test-Path $coordConfigPath)) {
    $adminToken  = (Get-Content $coordConfigPath | ConvertFrom-Json).admin_token
    $agentConfig = Get-Content $agentConfigPath -Raw

    # Pull agent_id from agent-config.yaml
    $agentId = ($agentConfig | Select-String "agent_id:\s*(.+)").Matches[0].Groups[1].Value.Trim()

    try {
        $resp = Invoke-RestMethod -Uri "$coordBase/api/agent-tokens" `
            -Method POST `
            -Headers @{ Authorization = "Bearer $adminToken"; "Content-Type" = "application/json" } `
            -Body (@{ agent_id = $agentId } | ConvertTo-Json) -TimeoutSec 5

        $newToken = $resp.token
        # Update auth_token in agent-config.yaml in-place.
        # [^\r\n]+ avoids consuming the \r in \r\n line endings
        $agentConfig = $agentConfig -replace '(?m)^(\s*auth_token:\s*)[^\r\n]+', ('$1' + $newToken)
        [System.IO.File]::WriteAllText($agentConfigPath, $agentConfig)
        Write-Host "  Token regenerated and saved to agent-config.yaml." -ForegroundColor Green
    } catch {
        Write-Host "  WARNING: Could not regenerate agent token: $_" -ForegroundColor Red
        Write-Host "  Run manually: coordinator create-agent-token [agent-id]" -ForegroundColor Yellow
    }
} else {
    Write-Host "  Skipping token regeneration (agent or coordinator config not found)." -ForegroundColor Yellow
}

# Step 9: Start agent + verify it registered
Write-Host ""
Write-Host "Step 9: Starting agent..." -ForegroundColor Yellow
sc.exe start arcvault-agent
Start-Sleep -Seconds 4

$token  = (Get-Content $coordConfigPath -ErrorAction SilentlyContinue | ConvertFrom-Json -ErrorAction SilentlyContinue).ad
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

# Wait for port 8080 to be free (up to 20 seconds)
Write-Host "  Waiting for port 8080 to be released..."
$waited = 0
while ($waited -lt 20) {
    Start-Sleep -Seconds 1
    $portInUse = netstat -ano | Select-String ":8080 " | Select-String "LISTENING"
    if (-not $portInUse) {
        Write-Host "  Port 8080 is free." -ForegroundColor Green
        break
    }
    $waited++
    if ($waited % 5 -eq 0) { Write-Host "  Still waiting... ($waited s)" }
}
if ($waited -ge 20) {
    # Force-kill whatever is still on 8080
    $netLines = netstat -ano | Select-String ":8080 " | Select-String "LISTENING"
    foreach ($line in $netLines) {
        $pid8080 = ($line.ToString().Trim() -split '\s+')[-1]
        if ($pid8080 -match '^\d+$' -and $pid8080 -ne "0") {
            taskkill /F /PID $pid8080 2>$null
        }
    }
    Start-Sleep -Seconds 2
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

# Version — must match build.ps1 and arcvault_installer.py
# UPDATE ALL THREE when bumping: build.ps1 $Version, rebuild-and-restart.ps1 $Version, installer py self.version
$Version = "v0.5.1"
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

go build -ldflags "-X main.Version=$Version" -o coordinator.exe .\coordinator
if ($LASTEXITCODE -ne 0) {
    Write-Host "  coordinator.exe build FAILED." -ForegroundColor Red
    exit 1
}
Write-Host "  coordinator.exe built successfully." -ForegroundColor Green

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

# Verify credential_key is present in config.json - warn if missing
$coordConfig = "C:\ArcVault\config.json"
if (Test-Path $coordConfig) {
    $cfg = Get-Content $coordConfig | ConvertFrom-Json
    if (-not $cfg.credential_key) {
        Write-Host ""
        Write-Host "  WARNING: credential_key not found in C:\ArcVault\config.json!" -ForegroundColor Red
        Write-Host "  Credential profiles will not work. Add it manually or re-run the installer." -ForegroundColor Yellow
        Write-Host ""
    }
}

sc.exe start arcvault-coordinator
Start-Sleep -Seconds 3

# Re-enable SCM auto-restart now that the new binary is in place
sc.exe failure arcvault-coordinator reset= 86400 actions= restart/3000/restart/3000/restart/3000 2>$null
Write-Host "  SCM failure recovery re-enabled." -ForegroundColor Green

try {
    $health = Invoke-RestMethod -Uri "https://localhost:8080/health" -SkipCertificateCheck -TimeoutSec 5
    Write-Host "  Coordinator health: $($health.status)" -ForegroundColor Green
} catch {
    Write-Host "  ERROR: Coordinator not responding." -ForegroundColor Red
    exit 1
}

# Step 8: Regenerate agent token (always — coordinator reinstall wipes tokens)
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
        $resp = Invoke-RestMethod -Uri "https://localhost:8080/api/agent-tokens" `
            -Method POST -SkipCertificateCheck `
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

$token  = (Get-Content $coordConfigPath -ErrorAction SilentlyContinue | ConvertFrom-Json -ErrorAction SilentlyContinue).admin_token
$agents = Invoke-RestMethod -Uri "https://localhost:8080/api/agents" -SkipCertificateCheck -Headers @{ Authorization = "Bearer $token" } -TimeoutSec 5 -ErrorAction SilentlyContinue

if ($agents -and $agents.data.Count -gt 0) {
    $agentCount = $agents.data.Count
    Write-Host "  SUCCESS: $agentCount agent(s) registered:" -ForegroundColor Green
    foreach ($a in $agents.data) {
        $aId = $a.id; $aHost = $a.hostname; $aSt = $a.status
        Write-Host "    - $aId  $aHost  status: $aSt" -ForegroundColor Green
    }
} elseif ($agents) {
    Write-Host "  No agents found yet - wait 10s and refresh the dashboard." -ForegroundColor Yellow
} else {
    Write-Host "  Could not query agents API - check coordinator logs." -ForegroundColor Yellow
}

Write-Host ""
Write-Host "=== Done! Open https://localhost:8080 in your browser. ===" -ForegroundColor Cyan
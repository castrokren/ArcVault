# ArcVault Rebuild & Restart Script
# Run from an Admin PowerShell in C:\Projects\ArcVault2.0
# Usage: .\rebuild-and-restart.ps1

$ProjectRoot = Split-Path -Parent $MyInvocation.MyCommand.Path
Set-Location $ProjectRoot

Write-Host ""
Write-Host "=== ArcVault Rebuild & Restart ===" -ForegroundColor Cyan
Write-Host "Project root: $ProjectRoot"
Write-Host ""

# Step 1: Stop services
Write-Host "Step 1: Stopping services..." -ForegroundColor Yellow
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
Remove-Item -Recurse -Force "coordinator\static\dist\*" -ErrorAction SilentlyContinue
Copy-Item -Recurse "dashboard\dist\*" "coordinator\static\dist\" -Force
Write-Host "  Synced dashboard\dist -> coordinator\static\dist" -ForegroundColor Green

# Step 4: Build coordinator.exe
Write-Host ""
Write-Host "Step 4: Building coordinator.exe..." -ForegroundColor Yellow

if (-not (Get-Command go -ErrorAction SilentlyContinue)) {
    Write-Host "  ERROR: go not found in PATH." -ForegroundColor Red
    exit 1
}

go build -o coordinator.exe .\coordinator
if ($LASTEXITCODE -ne 0) {
    Write-Host "  coordinator.exe build FAILED." -ForegroundColor Red
    exit 1
}
Write-Host "  coordinator.exe built successfully." -ForegroundColor Green

# Step 5: Build agent.exe
Write-Host ""
Write-Host "Step 5: Building agent.exe..." -ForegroundColor Yellow

go build -o agent.exe .\agent
if ($LASTEXITCODE -ne 0) {
    Write-Host "  agent.exe build FAILED." -ForegroundColor Red
    exit 1
}
Write-Host "  agent.exe built successfully." -ForegroundColor Green

# Step 6: Deploy to service locations
Write-Host ""
Write-Host "Step 6: Deploying to service locations..." -ForegroundColor Yellow

Copy-Item "coordinator.exe" "C:\ArcVault\coordinator.exe" -Force
Write-Host "  Copied coordinator.exe -> C:\ArcVault\" -ForegroundColor Green

Copy-Item "agent.exe" "C:\ArcVault-Agent\agent.exe" -Force
Write-Host "  Copied agent.exe -> C:\ArcVault-Agent\" -ForegroundColor Green

# Step 7: Start services
Write-Host ""
Write-Host "Step 7: Starting services..." -ForegroundColor Yellow
sc.exe start arcvault-coordinator
Start-Sleep -Seconds 3

try {
    $health = Invoke-RestMethod -Uri "http://localhost:8080/health" -TimeoutSec 5
    Write-Host "  Coordinator health: $($health.status)" -ForegroundColor Green
} catch {
    Write-Host "  ERROR: Coordinator not responding." -ForegroundColor Red
    exit 1
}

sc.exe start arcvault-agent
Start-Sleep -Seconds 4

# Step 8: Verify agent registered
Write-Host ""
Write-Host "Step 8: Verifying agent registered..." -ForegroundColor Yellow

try {
    $token = (Get-Content "C:\ArcVault\config.json" | ConvertFrom-Json).admin_token
    $agents = Invoke-RestMethod -Uri "http://localhost:8080/api/agents" -Headers @{ Authorization = "Bearer $token" } -TimeoutSec 5
    $count = $agents.data.Count
    if ($count -gt 0) {
        Write-Host "  SUCCESS: $count agent(s) registered:" -ForegroundColor Green
        foreach ($a in $agents.data) {
            Write-Host "    - $($a.id) | $($a.hostname) | status: $($a.status)" -ForegroundColor Green
        }
    } else {
        Write-Host "  No agents found yet. Wait 10s and refresh the dashboard." -ForegroundColor Yellow
    }
} catch {
    Write-Host "  Could not query agents API: $_" -ForegroundColor Yellow
}

Write-Host ""
Write-Host "=== Done! Open http://localhost:8080 in your browser. ===" -ForegroundColor Cyan

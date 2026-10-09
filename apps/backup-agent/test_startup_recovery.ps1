# Test Script: Startup Recovery
# This script tests that the backup agent correctly recovers interrupted jobs on restart

Write-Host ""
Write-Host "============================================================" -ForegroundColor Cyan
Write-Host "ArcVault Backup Agent - Startup Recovery Test" -ForegroundColor Cyan
Write-Host "============================================================" -ForegroundColor Cyan
Write-Host ""

# Configuration
$serviceName = "ArcVaultBackupAgent"
$agentUrl = "https://localhost:8443"
$headers = @{"X-API-Key" = $env:AGENT_TOKEN}

# Check if service exists
$service = Get-Service -Name $serviceName -ErrorAction SilentlyContinue
if (-not $service) {
    Write-Host "✗ Service '$serviceName' not found" -ForegroundColor Red
    Write-Host "Please install the service first." -ForegroundColor Yellow
    exit 1
}

# Check if AGENT_TOKEN is set
if (-not $env:AGENT_TOKEN) {
    Write-Host "✗ AGENT_TOKEN environment variable not set" -ForegroundColor Red
    Write-Host "Please set the AGENT_TOKEN environment variable." -ForegroundColor Yellow
    exit 1
}

Write-Host "[Pre-Check] Service found: $serviceName" -ForegroundColor Green
Write-Host "[Pre-Check] Agent URL: $agentUrl" -ForegroundColor Green
Write-Host ""

# Ensure service is running
Write-Host "[Setup] Ensuring service is running..." -ForegroundColor Yellow
if ($service.Status -ne "Running") {
    Start-Service -Name $serviceName
    Start-Sleep -Seconds 5
}

$service = Get-Service -Name $serviceName
if ($service.Status -ne "Running") {
    Write-Host "✗ Failed to start service" -ForegroundColor Red
    exit 1
}
Write-Host "✓ Service is running" -ForegroundColor Green
Write-Host ""

# Test 1: Create a test job
Write-Host "[Test 1] Creating a test backup job..." -ForegroundColor Yellow

# Use a large source to ensure job runs for a while
$config = @{
    sources = @("C:\Windows\System32")  # Large source
    destination = "C:\Backups\recovery-test"
    options = @{ 
        verbose = $true
        retries = 0
        threads = 1  # Slow it down
    }
} | ConvertTo-Json

try {
    $response = Invoke-RestMethod -Uri "$agentUrl/jobs" `
        -Method Post -Body $config -ContentType "application/json" `
        -Headers $headers -SkipCertificateCheck
    
    $jobId = $response.job_id
    Write-Host "✓ Created job: $jobId" -ForegroundColor Green
}
catch {
    Write-Host "✗ Failed to create job: $_" -ForegroundColor Red
    exit 1
}

Write-Host ""

# Test 2: Wait for job to start running
Write-Host "[Test 2] Waiting for job to start running..." -ForegroundColor Yellow
$maxWait = 30
$waited = 0
$jobStarted = $false

while ($waited -lt $maxWait) {
    Start-Sleep -Seconds 2
    $waited += 2
    
    try {
        $job = Invoke-RestMethod -Uri "$agentUrl/jobs/$jobId" `
            -Headers $headers -SkipCertificateCheck
        
        if ($job.state -eq "running") {
            $jobStarted = $true
            Write-Host "✓ Job is now running (waited $waited seconds)" -ForegroundColor Green
            break
        }
        
        Write-Host "  Job state: $($job.state) (waiting...)" -ForegroundColor Gray
    }
    catch {
        Write-Host "  Error checking job status: $_" -ForegroundColor Gray
    }
}

if (-not $jobStarted) {
    Write-Host "✗ Job did not start running within $maxWait seconds" -ForegroundColor Red
    Write-Host "Skipping restart test..." -ForegroundColor Yellow
    exit 1
}

Write-Host ""

# Test 3: Force-stop the service while job is running
Write-Host "[Test 3] Force-stopping service while job is running..." -ForegroundColor Yellow
Write-Host "  This simulates a service crash or system restart" -ForegroundColor Gray

try {
    Stop-Service -Name $serviceName -Force
    Write-Host "✓ Service stopped" -ForegroundColor Green
}
catch {
    Write-Host "✗ Failed to stop service: $_" -ForegroundColor Red
    exit 1
}

Write-Host ""

# Test 4: Restart the service
Write-Host "[Test 4] Restarting service..." -ForegroundColor Yellow

try {
    Start-Service -Name $serviceName
    Start-Sleep -Seconds 8  # Give it time to run startup recovery
    
    $service = Get-Service -Name $serviceName
    if ($service.Status -ne "Running") {
        Write-Host "✗ Service did not start" -ForegroundColor Red
        exit 1
    }
    
    Write-Host "✓ Service restarted successfully" -ForegroundColor Green
}
catch {
    Write-Host "✗ Failed to restart service: $_" -ForegroundColor Red
    exit 1
}

Write-Host ""

# Test 5: Verify job was recovered and marked as failed
Write-Host "[Test 5] Verifying job recovery..." -ForegroundColor Yellow

try {
    $job = Invoke-RestMethod -Uri "$agentUrl/jobs/$jobId" `
        -Headers $headers -SkipCertificateCheck
    
    Write-Host "  Job state: $($job.state)" -ForegroundColor Gray
    Write-Host "  Job error: $($job.error)" -ForegroundColor Gray
    
    $testPassed = $false
    
    if ($job.state -eq "failed") {
        Write-Host "✓ Job state is 'failed'" -ForegroundColor Green
        
        if ($job.error -like "*Service restarted*" -or $job.error -like "*interrupted*") {
            Write-Host "✓ Error message indicates service restart" -ForegroundColor Green
            $testPassed = $true
        }
        else {
            Write-Host "✗ Error message does not indicate service restart" -ForegroundColor Red
            Write-Host "  Expected: Contains 'Service restarted' or 'interrupted'" -ForegroundColor Yellow
            Write-Host "  Actual: $($job.error)" -ForegroundColor Yellow
        }
    }
    else {
        Write-Host "✗ Job state is not 'failed'" -ForegroundColor Red
        Write-Host "  Expected: 'failed'" -ForegroundColor Yellow
        Write-Host "  Actual: '$($job.state)'" -ForegroundColor Yellow
    }
    
    if ($testPassed) {
        Write-Host ""
        Write-Host "============================================================" -ForegroundColor Green
        Write-Host "✓ STARTUP RECOVERY TEST PASSED" -ForegroundColor Green
        Write-Host "============================================================" -ForegroundColor Green
        Write-Host ""
        Write-Host "The backup agent correctly:" -ForegroundColor Green
        Write-Host "  1. Detected the interrupted job on startup" -ForegroundColor Green
        Write-Host "  2. Marked it as 'failed'" -ForegroundColor Green
        Write-Host "  3. Set an appropriate error message" -ForegroundColor Green
        Write-Host ""
    }
    else {
        Write-Host ""
        Write-Host "============================================================" -ForegroundColor Red
        Write-Host "✗ STARTUP RECOVERY TEST FAILED" -ForegroundColor Red
        Write-Host "============================================================" -ForegroundColor Red
        Write-Host ""
        exit 1
    }
}
catch {
    Write-Host "✗ Failed to retrieve job status: $_" -ForegroundColor Red
    exit 1
}

# Test 6: Verify job persistence
Write-Host "[Test 6] Verifying job persistence..." -ForegroundColor Yellow

try {
    $allJobs = Invoke-RestMethod -Uri "$agentUrl/jobs" `
        -Headers $headers -SkipCertificateCheck
    
    $jobFound = $false
    foreach ($j in $allJobs.jobs) {
        if ($j.job_id -eq $jobId) {
            $jobFound = $true
            break
        }
    }
    
    if ($jobFound) {
        Write-Host "✓ Job found in jobs list after restart" -ForegroundColor Green
        Write-Host "  Job persisted successfully to database" -ForegroundColor Green
    }
    else {
        Write-Host "✗ Job not found in jobs list" -ForegroundColor Red
        exit 1
    }
}
catch {
    Write-Host "✗ Failed to retrieve jobs list: $_" -ForegroundColor Red
    exit 1
}

Write-Host ""

# Test 7: Verify database file exists
Write-Host "[Test 7] Verifying database file..." -ForegroundColor Yellow

$dbPath = "C:\ProgramData\ArcVault\jobs.db"
if (Test-Path $dbPath) {
    $dbSize = (Get-Item $dbPath).Length / 1KB
    Write-Host "✓ Database file exists: $dbPath" -ForegroundColor Green
    Write-Host "  Size: $($dbSize.ToString('F2')) KB" -ForegroundColor Gray
}
else {
    Write-Host "✗ Database file not found: $dbPath" -ForegroundColor Red
}

Write-Host ""

# Cleanup
Write-Host "[Cleanup] Test job ID: $jobId" -ForegroundColor Yellow
Write-Host "You can check the job logs at:" -ForegroundColor Gray
Write-Host "  C:\Backups\recovery-test\#backup_logs\$jobId.log" -ForegroundColor Gray
Write-Host "Or fallback location:" -ForegroundColor Gray
Write-Host "  C:\ProgramData\ArcVault\logs\$jobId.log" -ForegroundColor Gray

Write-Host ""
Write-Host "============================================================" -ForegroundColor Cyan
Write-Host "Test Complete" -ForegroundColor Cyan
Write-Host "============================================================" -ForegroundColor Cyan
Write-Host ""

exit 0


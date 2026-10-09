# Migration Guide: Robustness Improvements

## Overview

This guide explains how to deploy the robustness improvements to your ArcVault backup agent.

## What Changed

1. **Database for Job Persistence** - Jobs now persist across restarts
2. **Disk-Based Logging** - Logs written to disk with automatic rotation
3. **Job Recovery** - Interrupted jobs marked as failed on restart
4. **Maintenance Endpoint** - Clean up old jobs and logs

## Pre-Deployment Checklist

### 1. Review Dependencies

No new dependencies required! The changes use only Python standard library modules:
- `sqlite3` (built-in)
- `logging` (built-in)
- `logging.handlers` (built-in)

### 2. Check File System Permissions

Ensure the service account has write access to:
```
C:\ProgramData\ArcVault\
├── jobs.db          (will be created)
└── logs\            (will be created)
```

And for destination folders:
```
{backup_destination}\
└── #backup_logs\    (will be created per job)
```

### 3. Backup Existing Configuration

While this is backward compatible, back up your current setup:
```powershell
# Backup service configuration
Copy-Item "C:\Program Files\ArcVault\BackupAgent\service_config.json" `
          "C:\Program Files\ArcVault\BackupAgent\service_config.json.backup"

# Note: No job history to backup (was in-memory only)
```

## Deployment Steps

### Option 1: Automated Script (Recommended) ⭐

Use the automated deployment script with built-in error handling and verification:

```powershell
cd "C:\Users\castrk05\Py_stuff\ArcVault\apps\backup-agent"

# Standard deployment (with backup and tests)
.\deploy_robustness.ps1

# Quick deployment (skip tests)
.\deploy_robustness.ps1 -SkipTests

# Fast deployment (skip backup and tests)
.\deploy_robustness.ps1 -SkipBackup -SkipTests

# Force deployment (no prompts)
.\deploy_robustness.ps1 -Force
```

**Features:**
- ✅ Pre-deployment validation (5 checks)
- ✅ Automatic backup creation
- ✅ Test suite execution
- ✅ Error handling with rollback
- ✅ Post-deployment verification
- ✅ Service stop/start management
- ✅ Clear status messages

### Option 2: Manual PowerShell Commands

If you prefer manual control:

```powershell
# 1. Stop the service
Stop-Service -Name "ArcVaultBackupAgent"

# 2. Navigate to agent directory
cd "C:\Users\castrk05\Py_stuff\ArcVault\apps\backup-agent"

# 3. Create backup (optional but recommended)
$timestamp = Get-Date -Format "yyyyMMdd_HHmmss"
Copy-Item "C:\Program Files\ArcVault\BackupAgent\agent\app.py" `
          "C:\Program Files\ArcVault\BackupAgent\agent\app.py.backup_$timestamp"

# 4. Copy new modules to deployment location
Copy-Item "agent\db.py" "C:\Program Files\ArcVault\BackupAgent\agent\" -ErrorAction Stop
Copy-Item "agent\log_handler.py" "C:\Program Files\ArcVault\BackupAgent\agent\" -ErrorAction Stop
Copy-Item "agent\app.py" "C:\Program Files\ArcVault\BackupAgent\agent\" -Force -ErrorAction Stop

# 5. Create required directories (if not exist)
New-Item -ItemType Directory -Force -Path "C:\ProgramData\ArcVault"
New-Item -ItemType Directory -Force -Path "C:\ProgramData\ArcVault\logs"

# 6. Set permissions (allow service account to write)
$acl = Get-Acl "C:\ProgramData\ArcVault"
# Add your service account permissions here if needed

# 7. Start the service
Start-Service -Name "ArcVaultBackupAgent"

# 8. Verify service started
$service = Get-Service -Name "ArcVaultBackupAgent"
if ($service.Status -ne "Running") {
    Write-Host "✗ Service failed to start" -ForegroundColor Red
    Get-Content "C:\Program Files\ArcVault\BackupAgent\service.log" -Tail 20
} else {
    Write-Host "✓ Service started successfully" -ForegroundColor Green
}
```

### Option 3: Using Reinstall Script

If you prefer a full reinstall:

```powershell
cd "C:\Users\castrk05\Py_stuff\ArcVault\apps\backup-agent"
.\agent\reinstall_service.ps1
```

This will:
1. Stop and uninstall the old service
2. Copy all updated files
3. Install and start the new service

## Post-Deployment Verification

### 1. Check Service Status

```powershell
Get-Service -Name "ArcVaultBackupAgent"
# Should show: Status = Running
```

### 2. Verify Database Created

```powershell
Test-Path "C:\ProgramData\ArcVault\jobs.db"
# Should return: True
```

### 3. Test API Endpoints

```powershell
# Get agent info (should include job_counts)
$headers = @{"X-API-Key" = $env:AGENT_TOKEN}
Invoke-RestMethod -Uri "https://localhost:8443/info" -Headers $headers -SkipCertificateCheck

# Should return job_counts: { total: 0, queued: 0, running: 0, succeeded: 0, failed: 0 }
```

### 4. Run Test Backup Job

```powershell
# Create a test job
$config = @{
    sources = @("C:\Temp\test")
    destination = "C:\Backups\test"
    options = @{
        verbose = $true
        retries = 1
    }
} | ConvertTo-Json

$response = Invoke-RestMethod -Uri "https://localhost:8443/jobs" `
    -Method Post -Body $config -ContentType "application/json" `
    -Headers $headers -SkipCertificateCheck

$jobId = $response.job_id
Write-Host "Created job: $jobId"

# Wait a moment
Start-Sleep -Seconds 5

# Check job status
Invoke-RestMethod -Uri "https://localhost:8443/jobs/$jobId" `
    -Headers $headers -SkipCertificateCheck

# Verify log file exists
Test-Path "C:\Backups\test\#backup_logs\$jobId.log"
# Should return: True
```

### 5. Test Job Persistence (Restart Test)

```powershell
# 1. Create a job and note the job_id
# 2. Restart the service
Restart-Service -Name "ArcVaultBackupAgent"

# 3. Query jobs endpoint - should still show the job
$jobs = Invoke-RestMethod -Uri "https://localhost:8443/jobs" `
    -Headers $headers -SkipCertificateCheck
$jobs.jobs | Format-Table job_id, state, error
```

### 6. Test Startup Recovery (Optional)

Run the automated startup recovery test:

```powershell
cd C:\Users\castrk05\Py_stuff\ArcVault\apps\backup-agent
.\test_startup_recovery.ps1
```

This test will:
1. Create a long-running backup job
2. Force-stop the service while job is running
3. Restart the service
4. Verify the interrupted job is marked as "failed" with appropriate error message

## Rollback Procedure

If you need to rollback to the previous version, follow these detailed steps:

### Quick Rollback (Using Git)

```powershell
# 1. Stop the service
Stop-Service -Name "ArcVaultBackupAgent" -Force
Write-Host "✓ Service stopped" -ForegroundColor Green

# 2. Navigate to source directory
cd C:\Users\castrk05\Py_stuff\ArcVault\apps\backup-agent

# 3. Restore old app.py from git (before robustness changes)
git checkout HEAD~1 -- agent/app.py
Write-Host "✓ Restored previous app.py" -ForegroundColor Green

# 4. Copy restored version to deployment location
Copy-Item "agent\app.py" "C:\Program Files\ArcVault\BackupAgent\agent\" -Force
Write-Host "✓ Deployed old version" -ForegroundColor Green

# 5. Start the service
Start-Service -Name "ArcVaultBackupAgent"
Start-Sleep -Seconds 3

# 6. Verify service is running
$service = Get-Service -Name "ArcVaultBackupAgent"
if ($service.Status -eq "Running") {
    Write-Host "✓ Service running - Rollback complete" -ForegroundColor Green
} else {
    Write-Host "✗ Service failed to start" -ForegroundColor Red
}

# 7. Reset git changes (don't commit the rollback)
git checkout -- agent/app.py
```

### Manual Rollback (Using Backup)

If you created a backup before deployment:

```powershell
# 1. Stop the service
Stop-Service -Name "ArcVaultBackupAgent" -Force

# 2. Restore from backup
$backupPath = "C:\Program Files\ArcVault\BackupAgent\agent\app.py.backup"
$currentPath = "C:\Program Files\ArcVault\BackupAgent\agent\app.py"

if (Test-Path $backupPath) {
    Copy-Item $backupPath $currentPath -Force
    Write-Host "✓ Restored from backup" -ForegroundColor Green
} else {
    Write-Host "✗ Backup not found at $backupPath" -ForegroundColor Red
    Write-Host "Use git rollback method instead" -ForegroundColor Yellow
    exit 1
}

# 3. Start the service
Start-Service -Name "ArcVaultBackupAgent"

# 4. Verify
Get-Service -Name "ArcVaultBackupAgent"
```

### Complete Rollback (Remove New Files)

To completely remove the robustness improvements:

```powershell
# 1. Stop the service
Stop-Service -Name "ArcVaultBackupAgent" -Force

# 2. Remove new module files
$agentDir = "C:\Program Files\ArcVault\BackupAgent\agent"

Remove-Item "$agentDir\db.py" -ErrorAction SilentlyContinue
Remove-Item "$agentDir\log_handler.py" -ErrorAction SilentlyContinue
Write-Host "✓ Removed new modules" -ForegroundColor Green

# 3. Restore old app.py (use git or backup method above)

# 4. Optionally remove data files (if you want to start fresh)
$dataDir = "C:\ProgramData\ArcVault"
if (Test-Path $dataDir) {
    Write-Host "Data directory: $dataDir" -ForegroundColor Yellow
    Write-Host "This contains the job database and logs." -ForegroundColor Yellow
    
    $confirm = Read-Host "Remove data directory? (yes/no)"
    if ($confirm -eq "yes") {
        Remove-Item $dataDir -Recurse -Force
        Write-Host "✓ Removed data directory" -ForegroundColor Green
    } else {
        Write-Host "Data directory preserved" -ForegroundColor Yellow
    }
}

# 5. Start the service
Start-Service -Name "ArcVaultBackupAgent"

# 6. Verify
Get-Service -Name "ArcVaultBackupAgent"
Write-Host ""
Write-Host "Rollback complete. Service is running old version." -ForegroundColor Green
```

### Post-Rollback Verification

```powershell
# Check service is running
Get-Service -Name "ArcVaultBackupAgent"

# Test basic functionality
$headers = @{"X-API-Key" = $env:AGENT_TOKEN}
$info = Invoke-RestMethod -Uri "https://localhost:8443/info" `
    -Headers $headers -SkipCertificateCheck

Write-Host "Service Info:" -ForegroundColor Cyan
$info | ConvertTo-Json

# Create a test job to ensure basic operations work
$testConfig = @{
    sources = @("C:\Temp\test")
    destination = "C:\Backups\rollback-test"
    options = @{ verbose = $true }
} | ConvertTo-Json

try {
    $job = Invoke-RestMethod -Uri "https://localhost:8443/jobs" `
        -Method Post -Body $testConfig `
        -ContentType "application/json" `
        -Headers $headers -SkipCertificateCheck
    
    Write-Host "✓ Test job created: $($job.job_id)" -ForegroundColor Green
    Write-Host "✓ Rollback successful - service working normally" -ForegroundColor Green
}
catch {
    Write-Host "✗ Error creating test job: $_" -ForegroundColor Red
}
```

### Important Notes About Rollback

⚠️ **Data Loss:**
- Rolling back will lose access to jobs stored in the database
- Jobs in memory (before rollback) will be lost
- Log files on disk will remain but won't be managed by the old version

⚠️ **After Rollback:**
- Database file (`jobs.db`) will remain but won't be used
- Log files in `#backup_logs/` folders will remain
- New module files (`db.py`, `log_handler.py`) can be deleted or left in place (harmless)

✅ **Safe to Rollback:**
- No schema changes to existing systems
- Old version will work exactly as before
- Can re-deploy new version later without issues

## Troubleshooting

### Issue: Service won't start

**Symptoms**: Service fails to start after deployment

**Check**:
1. Review service logs:
   ```powershell
   Get-Content "C:\Program Files\ArcVault\BackupAgent\service.log" -Tail 50
   ```

2. Check for import errors:
   ```powershell
   cd "C:\Program Files\ArcVault\BackupAgent"
   python -c "from agent import db, log_handler; print('OK')"
   ```

3. Verify file permissions:
   ```powershell
   Test-Path "C:\ProgramData\ArcVault" -PathType Container
   ```

**Solution**: Ensure all three new/updated files are in place:
- `agent\app.py` (updated)
- `agent\db.py` (new)
- `agent\log_handler.py` (new)

### Issue: Database errors

**Symptoms**: "unable to open database file" or similar

**Check**: Permissions on `C:\ProgramData\ArcVault\`

**Solution**:
```powershell
# Grant NETWORK SERVICE write access (if running as NETWORK SERVICE)
$path = "C:\ProgramData\ArcVault"
$acl = Get-Acl $path
$rule = New-Object System.Security.AccessControl.FileSystemAccessRule(
    "NETWORK SERVICE", "FullControl", "ContainerInherit,ObjectInherit", "None", "Allow")
$acl.AddAccessRule($rule)
Set-Acl $path $acl
```

### Issue: Logs not appearing in destination

**Symptoms**: Logs in `C:\ProgramData\ArcVault\logs\` but not in `{destination}\#backup_logs\`

**Possible Causes**:
1. Destination path not accessible when job starts
2. Permission issues on destination

**Solution**: Check that destination exists and service account can write:
```powershell
Test-Path "D:\Backups\" -PathType Container
# Manually create #backup_logs folder and set permissions if needed
```

### Issue: Old jobs still in memory

**Symptoms**: Jobs from before upgrade not showing after restart

**Explanation**: This is expected! Old jobs were in memory only. Only jobs created after the upgrade will persist.

### Issue: Maintenance endpoint returns 401

**Symptoms**: Cleanup endpoint returns "Unauthorized"

**Solution**: Ensure you're passing the API key:
```powershell
$headers = @{"X-API-Key" = $env:AGENT_TOKEN}
Invoke-RestMethod -Uri "https://localhost:8443/maintenance/cleanup?days=30" `
    -Method Post -Headers $headers -SkipCertificateCheck
```

## Configuration Options

### Change Database Location

```powershell
# Set environment variable before starting service
[System.Environment]::SetEnvironmentVariable(
    "ARCVAULT_DB_PATH", 
    "D:\ArcVault\jobs.db", 
    [System.EnvironmentVariableTarget]::Machine
)
```

### Change Default Logs Directory

```powershell
[System.Environment]::SetEnvironmentVariable(
    "ARCVAULT_LOGS_DIR", 
    "D:\ArcVault\logs", 
    [System.EnvironmentVariableTarget]::Machine
)
```

## Maintenance Tasks

### Schedule Regular Cleanup

Create a scheduled task to clean up old jobs:

```powershell
# Create scheduled task to run cleanup monthly
$action = New-ScheduledTaskAction -Execute "PowerShell.exe" -Argument @"
-Command "
    `$headers = @{'X-API-Key' = [System.Environment]::GetEnvironmentVariable('AGENT_TOKEN')}
    Invoke-RestMethod -Uri 'https://localhost:8443/maintenance/cleanup?days=30' ``
        -Method Post -Headers `$headers -SkipCertificateCheck
"
"@

$trigger = New-ScheduledTaskTrigger -Monthly -At 2am -DaysOfMonth 1

Register-ScheduledTask -TaskName "ArcVault-Cleanup" `
    -Action $action -Trigger $trigger `
    -User "SYSTEM" -RunLevel Highest
```

### Monitor Database Size

```powershell
# Check database size
$dbPath = "C:\ProgramData\ArcVault\jobs.db"
$size = (Get-Item $dbPath).Length / 1MB
Write-Host "Database size: $size MB"

# Query job count
# (Use sqlite3.exe or Python script to query)
```

### Query Database Directly

```powershell
# Install sqlite3.exe if needed, then:
sqlite3 C:\ProgramData\ArcVault\jobs.db "SELECT COUNT(*) FROM jobs;"
sqlite3 C:\ProgramData\ArcVault\jobs.db "SELECT state, COUNT(*) FROM jobs GROUP BY state;"
```

## Support

If you encounter issues:

1. **Check logs**: `C:\Program Files\ArcVault\BackupAgent\service.log`
2. **Review documentation**: `ROBUSTNESS_IMPROVEMENTS.md`
3. **Verify permissions**: Ensure service account can write to data directories
4. **Test manually**: Run Python interpreter as service account and import modules

## Next Steps

After successful deployment:

1. ✅ Monitor first few backup jobs to ensure logs are written correctly
2. ✅ Verify job persistence by restarting service after a job completes
3. ✅ Set up scheduled maintenance task for cleanup
4. ✅ Update monitoring/alerting to check database file and log directory
5. ✅ Review `ROBUSTNESS_IMPROVEMENTS.md` for additional features

## Questions?

- Check `ROBUSTNESS_IMPROVEMENTS.md` for detailed technical information
- Review `SERVICE_TROUBLESHOOTING.md` for general service issues
- Examine source code in `agent\db.py` and `agent\log_handler.py`


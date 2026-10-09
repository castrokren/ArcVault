# Rollback Guide - Robustness Improvements

## Quick Rollback

### Using Automated Script (Recommended)

```powershell
# Simple rollback (keeps data)
.\rollback_robustness.ps1

# Rollback and remove all data
.\rollback_robustness.ps1 -RemoveData

# Force rollback without prompts
.\rollback_robustness.ps1 -Force

# Rollback using backup instead of git
.\rollback_robustness.ps1 -UseGit:$false
```

---

## What Gets Rolled Back

| Item | Action | Effect |
|------|--------|--------|
| `app.py` | Restored to previous version | Database/logging features removed |
| `db.py` | Deleted | Database module removed |
| `log_handler.py` | Deleted | Disk logging module removed |
| `jobs.db` | Preserved by default | Can be deleted with `-RemoveData` |
| Log files | Preserved by default | Can be deleted with `-RemoveData` |
| Service | Restarted | Runs old version |

---

## Manual Rollback Commands

### Option 1: Git-Based Rollback

```powershell
# Stop service
Stop-Service -Name "ArcVaultBackupAgent" -Force

# Navigate to source
cd C:\Users\castrk05\Py_stuff\ArcVault\apps\backup-agent

# Restore old app.py from git
git show HEAD~1:agent/app.py > agent\app.py.old

# Deploy old version
Copy-Item "agent\app.py.old" "C:\Program Files\ArcVault\BackupAgent\agent\app.py" -Force
Remove-Item "agent\app.py.old"

# Remove new modules
Remove-Item "C:\Program Files\ArcVault\BackupAgent\agent\db.py" -Force
Remove-Item "C:\Program Files\ArcVault\BackupAgent\agent\log_handler.py" -Force

# Start service
Start-Service -Name "ArcVaultBackupAgent"

# Verify
Get-Service -Name "ArcVaultBackupAgent"
```

### Option 2: Backup-Based Rollback

```powershell
# Stop service
Stop-Service -Name "ArcVaultBackupAgent" -Force

# Restore from backup
$backupPath = "C:\Program Files\ArcVault\BackupAgent\agent\app.py.backup"
$currentPath = "C:\Program Files\ArcVault\BackupAgent\agent\app.py"

Copy-Item $backupPath $currentPath -Force

# Remove new modules
Remove-Item "C:\Program Files\ArcVault\BackupAgent\agent\db.py" -Force
Remove-Item "C:\Program Files\ArcVault\BackupAgent\agent\log_handler.py" -Force

# Start service
Start-Service -Name "ArcVaultBackupAgent"

# Verify
Get-Service -Name "ArcVaultBackupAgent"
```

### Option 3: Complete Uninstall & Reinstall

```powershell
# Stop and uninstall service
cd "C:\Program Files\ArcVault\BackupAgent"
python agent\win_service.py remove

# Remove agent directory
Remove-Item "C:\Program Files\ArcVault\BackupAgent" -Recurse -Force

# Remove data directory (optional)
Remove-Item "C:\ProgramData\ArcVault" -Recurse -Force

# Checkout old version from git
cd C:\Users\castrk05\Py_stuff\ArcVault\apps\backup-agent
git checkout HEAD~1 -- agent/

# Reinstall service with old version
.\agent\reinstall_service.ps1

# Restore to current git state
git checkout -- agent/
```

---

## Rollback Scenarios

### Scenario 1: Service Won't Start After Deployment

**Problem:** Service fails to start with new version

**Solution:**
```powershell
# Quick rollback
.\rollback_robustness.ps1 -Force

# Check service logs
Get-Content "C:\Program Files\ArcVault\BackupAgent\service.log" -Tail 50
```

### Scenario 2: Database Errors

**Problem:** Getting database-related errors

**Solution:**
```powershell
# Rollback and remove database
.\rollback_robustness.ps1 -RemoveData -Force
```

### Scenario 3: Performance Issues

**Problem:** New version is slower than expected

**Solution:**
```powershell
# Rollback but keep data for analysis
.\rollback_robustness.ps1

# Analyze database size
$dbPath = "C:\ProgramData\ArcVault\jobs.db"
if (Test-Path $dbPath) {
    $size = (Get-Item $dbPath).Length / 1MB
    Write-Host "Database size: $size MB"
}
```

### Scenario 4: Need to Test Both Versions

**Problem:** Want to compare old vs new behavior

**Solution:**
```powershell
# Create a snapshot of current data
$timestamp = Get-Date -Format "yyyyMMdd_HHmmss"
Copy-Item "C:\ProgramData\ArcVault" "C:\ProgramData\ArcVault_backup_$timestamp" -Recurse

# Rollback
.\rollback_robustness.ps1 -Force

# Test old version...

# Re-deploy new version when ready
# (follow deployment steps from MIGRATION_GUIDE.md)
```

---

## Post-Rollback Verification

### 1. Check Service Status

```powershell
$service = Get-Service -Name "ArcVaultBackupAgent"
Write-Host "Status: $($service.Status)"

# Should show: Running
```

### 2. Test API Endpoints

```powershell
$headers = @{"X-API-Key" = $env:AGENT_TOKEN}

# Test health endpoint
$health = Invoke-RestMethod -Uri "https://localhost:8443/health" `
    -Headers $headers -SkipCertificateCheck

Write-Host "Health: $($health.status)"

# Test info endpoint
$info = Invoke-RestMethod -Uri "https://localhost:8443/info" `
    -Headers $headers -SkipCertificateCheck

$info | ConvertTo-Json
```

### 3. Create Test Job

```powershell
$config = @{
    sources = @("C:\Temp\rollback-test")
    destination = "C:\Backups\rollback-test"
    options = @{ verbose = $true; retries = 1 }
} | ConvertTo-Json

$job = Invoke-RestMethod -Uri "https://localhost:8443/jobs" `
    -Method Post -Body $config `
    -ContentType "application/json" `
    -Headers $headers -SkipCertificateCheck

Write-Host "Test job created: $($job.job_id)"

# Check job status
Start-Sleep -Seconds 5
$status = Invoke-RestMethod -Uri "https://localhost:8443/jobs/$($job.job_id)" `
    -Headers $headers -SkipCertificateCheck

$status | ConvertTo-Json
```

### 4. Check Service Logs

```powershell
# View recent service logs
Get-Content "C:\Program Files\ArcVault\BackupAgent\service.log" -Tail 30

# Check for errors
Select-String -Path "C:\Program Files\ArcVault\BackupAgent\service.log" `
    -Pattern "ERROR|CRITICAL" | Select-Object -Last 10
```

---

## Data Handling After Rollback

### What Happens to the Database?

After rollback, the database file remains but is **not used** by the old version:

```powershell
# Check if database exists
$dbPath = "C:\ProgramData\ArcVault\jobs.db"
if (Test-Path $dbPath) {
    Write-Host "Database file exists (not used by old version)"
    
    # View database size
    $size = (Get-Item $dbPath).Length / 1KB
    Write-Host "Size: $size KB"
}
```

### What Happens to Log Files?

Log files in `#backup_logs/` folders remain intact:

```powershell
# Find all backup log directories
Get-ChildItem -Path "C:\Backups" -Recurse -Directory | 
    Where-Object { $_.Name -eq "#backup_logs" } | 
    ForEach-Object {
        $logCount = (Get-ChildItem $_.FullName -Filter "*.log").Count
        Write-Host "$($_.FullName): $logCount log files"
    }
```

### Cleaning Up Old Data

```powershell
# Remove database
Remove-Item "C:\ProgramData\ArcVault\jobs.db" -Force

# Remove log files
Remove-Item "C:\ProgramData\ArcVault\logs" -Recurse -Force

# Remove all #backup_logs directories
Get-ChildItem -Path "C:\Backups" -Recurse -Directory |
    Where-Object { $_.Name -eq "#backup_logs" } |
    ForEach-Object { Remove-Item $_.FullName -Recurse -Force }

Write-Host "✓ All robustness data removed"
```

---

## Re-Deploying After Rollback

If you want to re-deploy the robustness improvements after rolling back:

```powershell
# 1. Ensure you're on the latest version
cd C:\Users\castrk05\Py_stuff\ArcVault\apps\backup-agent
git pull

# 2. Follow deployment steps
# See MIGRATION_GUIDE.md for detailed instructions

# Quick deploy:
Stop-Service -Name "ArcVaultBackupAgent"

Copy-Item "agent\db.py" "C:\Program Files\ArcVault\BackupAgent\agent\"
Copy-Item "agent\log_handler.py" "C:\Program Files\ArcVault\BackupAgent\agent\"
Copy-Item "agent\app.py" "C:\Program Files\ArcVault\BackupAgent\agent\" -Force

Start-Service -Name "ArcVaultBackupAgent"

# 3. Verify deployment
.\test_robustness.py
```

---

## Troubleshooting Rollback Issues

### Issue: Service Won't Start After Rollback

**Check:**
```powershell
# View service logs
Get-Content "C:\Program Files\ArcVault\BackupAgent\service.log" -Tail 50

# Check if app.py was restored correctly
$appPath = "C:\Program Files\ArcVault\BackupAgent\agent\app.py"
Select-String -Path $appPath -Pattern "db.py|log_handler"
# Should return empty (no matches) after rollback
```

**Solution:**
```powershell
# Ensure new modules are removed
Remove-Item "C:\Program Files\ArcVault\BackupAgent\agent\db.py" -ErrorAction SilentlyContinue
Remove-Item "C:\Program Files\ArcVault\BackupAgent\agent\log_handler.py" -ErrorAction SilentlyContinue

# Try starting again
Start-Service -Name "ArcVaultBackupAgent"
```

### Issue: Import Errors in Logs

**Error:** `ModuleNotFoundError: No module named 'agent.db'`

**Solution:**
```powershell
# Old app.py still has imports for new modules
# Restore from a known good backup or git

cd C:\Users\castrk05\Py_stuff\ArcVault\apps\backup-agent
git checkout HEAD~1 -- agent/app.py
Copy-Item "agent\app.py" "C:\Program Files\ArcVault\BackupAgent\agent\" -Force

Restart-Service -Name "ArcVaultBackupAgent"
```

### Issue: API Endpoints Not Working

**Check:**
```powershell
# Test health endpoint
$headers = @{"X-API-Key" = $env:AGENT_TOKEN}
try {
    Invoke-RestMethod -Uri "https://localhost:8443/health" `
        -Headers $headers -SkipCertificateCheck
} catch {
    Write-Host "Error: $($_.Exception.Message)"
}
```

**Common causes:**
- Service not fully started (wait 10 seconds)
- Certificate issues (use `-SkipCertificateCheck`)
- Wrong API token

---

## Prevention: Before Deploying

To make rollback easier, create backups before deploying:

```powershell
# Create backup before deployment
$timestamp = Get-Date -Format "yyyyMMdd_HHmmss"
$backupDir = "C:\Program Files\ArcVault\BackupAgent_backup_$timestamp"

Copy-Item "C:\Program Files\ArcVault\BackupAgent" $backupDir -Recurse
Write-Host "Backup created: $backupDir"

# Now deploy new version...

# If rollback needed:
# Stop-Service -Name "ArcVaultBackupAgent"
# Remove-Item "C:\Program Files\ArcVault\BackupAgent" -Recurse
# Copy-Item $backupDir "C:\Program Files\ArcVault\BackupAgent" -Recurse
# Start-Service -Name "ArcVaultBackupAgent"
```

---

## Summary

| Method | Speed | Ease | Data Preserved | Recommended For |
|--------|-------|------|----------------|-----------------|
| Automated Script | ⚡ Fast | 😊 Easy | ✅ Yes (default) | Most users |
| Git Rollback | ⚡ Fast | 😐 Medium | ✅ Yes | Git users |
| Backup Rollback | ⚡ Fast | 😐 Medium | ✅ Yes | Non-git users |
| Complete Reinstall | 🐌 Slow | 😓 Hard | ❌ Optional | Fresh start |

**Recommendation:** Use the automated script (`rollback_robustness.ps1`) for easiest rollback.

---

## Support

If rollback fails or you encounter issues:

1. Check service logs: `C:\Program Files\ArcVault\BackupAgent\service.log`
2. Review this guide
3. Try complete reinstall method as last resort
4. Refer to `MIGRATION_GUIDE.md` for deployment help

---

**Last Updated:** December 2, 2025


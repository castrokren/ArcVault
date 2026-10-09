# Quick Reference - Robustness Improvements

## 🎯 What Was Fixed

1. **Job Persistence** - Jobs now survive service restarts (SQLite database)
2. **Log Management** - Logs never dropped, automatic rotation (RotatingFileHandler)

## 📦 Files Changed

```
✅ NEW: agent/db.py                    # Database module
✅ NEW: agent/log_handler.py           # Logging module  
✅ UPDATED: agent/app.py                # Main app with persistence
```

## 🚀 Quick Deploy

### Automated (Recommended)
```powershell
# One-command deployment with safety checks
.\deploy_robustness.ps1

# Quick deploy (skip tests)
.\deploy_robustness.ps1 -SkipTests -Force
```

### Manual
```powershell
# Stop → Copy → Start
Stop-Service -Name "ArcVaultBackupAgent"

Copy-Item "agent\db.py" "C:\Program Files\ArcVault\BackupAgent\agent\"
Copy-Item "agent\log_handler.py" "C:\Program Files\ArcVault\BackupAgent\agent\"
Copy-Item "agent\app.py" "C:\Program Files\ArcVault\BackupAgent\agent\" -Force

New-Item -ItemType Directory -Force -Path "C:\ProgramData\ArcVault"

Start-Service -Name "ArcVaultBackupAgent"
```

## 📍 File Locations

| What | Where |
|------|-------|
| Database | `C:\ProgramData\ArcVault\jobs.db` |
| Logs (fallback) | `C:\ProgramData\ArcVault\logs\{job_id}.log` |
| Logs (primary) | `{destination}/#backup_logs\{job_id}.log` |

## 🔧 Key Features

- ✅ Jobs persist across restarts
- ✅ Interrupted jobs marked as failed on recovery
- ✅ All logs written to disk (never dropped)
- ✅ Automatic log rotation (10MB files, 3 backups)
- ✅ Logs stored in backup destination
- ✅ Cleanup API: `POST /maintenance/cleanup?days=30`

## ✅ Verify Deployment

```powershell
# 1. Service running?
Get-Service -Name "ArcVaultBackupAgent"

# 2. Database exists?
Test-Path "C:\ProgramData\ArcVault\jobs.db"

# 3. Create test job
$config = @{sources=@("C:\test"); destination="D:\backup"} | ConvertTo-Json
$headers = @{"X-API-Key"=$env:AGENT_TOKEN}
Invoke-RestMethod -Uri "https://localhost:8443/jobs" -Method Post `
    -Body $config -ContentType "application/json" -Headers $headers `
    -SkipCertificateCheck

# 4. Check logs created
# Should see: D:\backup\#backup_logs\{job-id}.log
```

## 🧪 Test Before Deploy

```powershell
cd apps\backup-agent
python test_robustness.py
```

## 🔄 Rollback

If you need to revert the changes:

```powershell
# Simple rollback
.\rollback_robustness.ps1

# Rollback + remove data
.\rollback_robustness.ps1 -RemoveData -Force
```

**Full guide:** `ROLLBACK_GUIDE.md`

## 📚 Full Documentation

- `QUICK_REFERENCE.md` - This file
- `ROBUSTNESS_SUMMARY.md` - Complete overview
- `ROBUSTNESS_IMPROVEMENTS.md` - Technical details
- `MIGRATION_GUIDE.md` - Deployment guide
- `ROLLBACK_GUIDE.md` - Rollback procedures

## 🆘 Quick Troubleshooting

| Problem | Solution |
|---------|----------|
| Service won't start | Check `C:\Program Files\ArcVault\BackupAgent\service.log` |
| Database error | Verify `C:\ProgramData\ArcVault\` permissions |
| Logs not in destination | Check destination path accessible |

## 🔑 Key Code Changes

### Before
```python
# Lost on restart
jobs: Dict[str, Dict] = {}
log_buffers: Dict[str, "queue.Queue"] = {}

# Drops logs at 950 items
if q.qsize() >= 950:
    # Drop oldest logs
```

### After
```python
# Persisted to database
@app.on_event("startup")
async def startup_event():
    db.init_db()
    active_jobs = db.load_active_jobs()
    # Mark interrupted jobs as failed

# Never drops logs
def _enqueue_log(job_id, text):
    log_handler.log_to_job(job_id, text)  # Disk
    # Also queue for streaming
```

## 💡 Tips

- Logs automatically stored in `{destination}/#backup_logs/`
- Schedule monthly cleanup: `POST /maintenance/cleanup?days=30`
- Query jobs anytime: `GET /jobs` (persisted!)
- Check job stats: `GET /info` (includes counts)

## ⚡ Zero New Dependencies

Uses only Python standard library:
- `sqlite3` ✅
- `logging` ✅
- `logging.handlers` ✅

No `pip install` needed!

---

**Status**: ✅ Ready for Production

**Backward Compatible**: ✅ Yes

**Migration Required**: ❌ No (automatic)


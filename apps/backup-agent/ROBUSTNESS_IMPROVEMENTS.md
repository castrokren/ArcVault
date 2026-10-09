# ArcVault Agent: Robustness Improvements

## Overview

This document describes the robustness improvements made to the ArcVault Backup Agent to address job persistence and log management issues.

## Changes Implemented

### 1. Database-Backed Job Persistence

**Problem**: Jobs were stored in memory (`jobs: Dict[str, Dict] = {}`), causing all job history to be lost on service restart.

**Solution**: Implemented SQLite database for persistent job storage.

#### New Module: `agent/db.py`

Key features:
- **Persistent storage**: Jobs are saved to SQLite database at `C:/ProgramData/ArcVault/jobs.db`
- **Automatic recovery**: On startup, interrupted jobs are marked as failed
- **Fast access**: In-memory cache with write-through to database
- **Job statistics**: Query job counts by state without loading all jobs
- **Cleanup utilities**: Remove old job records (default: 30 days)

#### Database Schema

```sql
CREATE TABLE jobs (
    id TEXT PRIMARY KEY,
    state TEXT NOT NULL,
    progress INTEGER DEFAULT 0,
    error TEXT,
    sources TEXT,
    config TEXT,
    created_at REAL NOT NULL,
    started_at REAL,
    finished_at REAL,
    updated_at REAL NOT NULL
)
```

#### Key Functions

- `init_db()`: Initialize database schema
- `save_job(job_id, job_data)`: Save/update a job
- `load_job(job_id)`: Load a specific job
- `load_all_jobs()`: Load all jobs
- `load_active_jobs()`: Load only queued/running jobs
- `delete_old_jobs(days)`: Clean up old completed jobs
- `get_job_stats()`: Get job counts by state

### 2. Disk-Based Log Management with Rotation

**Problem**: Log queue size was limited to 1000 entries, silently dropping oldest logs when capacity was reached.

**Solution**: Implemented disk-based logging with automatic rotation using Python's `RotatingFileHandler`.

#### New Module: `agent/log_handler.py`

Key features:
- **Persistent logs**: All logs written to disk immediately
- **Automatic rotation**: 10MB per file, keeps 3 backup files
- **Destination-aware**: Logs stored in `destination/#backup_logs/` folder [[memory:4155247]]
- **Never drops logs**: Disk storage grows as needed
- **Streaming support**: In-memory queue kept for live log streaming
- **Cleanup utilities**: Remove old log files (default: 30 days)

#### Log File Locations

1. **Primary location** (preferred): `{destination}/#backup_logs/{job_id}.log`
   - Stored alongside backed-up data
   - Easy to find when reviewing backups

2. **Fallback location**: `C:/ProgramData/ArcVault/logs/{job_id}.log`
   - Used when destination is unavailable or not specified

#### Key Functions

- `get_job_logger(job_id, destination)`: Get logger for a job
- `log_to_job(job_id, message, level)`: Write log message
- `get_job_log_path(job_id, destination)`: Get path to log file
- `read_job_logs(job_id, destination, tail_lines)`: Read recent logs
- `follow_job_logs(job_id, destination, start_pos)`: Stream logs (tail -f style)
- `close_job_logger(job_id)`: Clean up logger resources
- `cleanup_old_logs(days)`: Remove old log files

### 3. Enhanced Application Logic

#### Startup Recovery

```python
@app.on_event("startup")
async def startup_event():
    """Initialize database and recover any interrupted jobs."""
    db.init_db()
    
    # Recover jobs that were running when the service stopped
    active_jobs = db.load_active_jobs()
    for job_id, job_data in active_jobs.items():
        if job_data.get("state") in ["queued", "running"]:
            job_data["state"] = "failed"
            job_data["error"] = "Service restarted - job interrupted"
            job_data["finished_at"] = time.time()
            db.save_job(job_id, job_data)
```

#### Dual Logging Strategy

All log messages are written to **both**:
1. **Disk** (via `RotatingFileHandler`): Persistent, never dropped
2. **Memory queue** (limited to 1000 entries): For live streaming, old entries dropped when full

```python
def _enqueue_log(job_id: str, text: str, level: int = logging.INFO) -> None:
    """
    Log to both disk (persistent) and memory queue (for streaming).
    """
    # Always write to disk first (persistent, never dropped)
    log_handler.log_to_job(job_id, text, level)
    
    # Also put in queue for live streaming (if queue exists)
    if job_id in log_buffers:
        # ... queue management ...
```

#### Job State Management

- **Write-through cache**: Jobs stored in memory cache and immediately written to database
- **Thread-safe**: All job updates protected by locks
- **Consistent state**: Database is source of truth; cache rebuilt on restart

### 4. New Maintenance Endpoint

```http
POST /maintenance/cleanup?days=30
X-API-Key: your-token-here
```

Cleans up:
- Job records older than N days (only completed/failed)
- Log files older than N days
- Cache is refreshed after cleanup

Response:
```json
{
  "status": "cleanup_complete",
  "jobs_deleted": 42,
  "log_files_deleted": 126,
  "days": 30
}
```

## Configuration

### Environment Variables

- `ARCVAULT_DB_PATH`: Override default database path (default: `C:/ProgramData/ArcVault/jobs.db`)
- `ARCVAULT_LOGS_DIR`: Override default logs directory (default: `C:/ProgramData/ArcVault/logs`)
- `AGENT_MAX_CONCURRENT_JOBS`: Maximum concurrent backup jobs (default: 3)

### File System Layout

```
C:/ProgramData/ArcVault/
├── jobs.db                    # SQLite database
└── logs/
    ├── {job_id_1}.log
    ├── {job_id_1}.log.1       # Rotated backup
    ├── {job_id_1}.log.2
    ├── {job_id_2}.log
    └── ...

{backup_destination}/
└── #backup_logs/              # Logs stored with backup data
    ├── {job_id}.log
    ├── {job_id}.log.1
    └── ...
```

## Benefits

### Before
- ❌ All job history lost on restart
- ❌ Logs silently dropped when queue full (>950 items)
- ❌ No way to recover interrupted jobs
- ❌ No persistent audit trail

### After
- ✅ Jobs persisted in SQLite database
- ✅ All logs written to disk with rotation
- ✅ Interrupted jobs automatically recovered and marked as failed
- ✅ Complete audit trail survives restarts
- ✅ Logs stored in destination folder for easy access
- ✅ Old data can be cleaned up with maintenance endpoint
- ✅ Statistics available without loading all jobs into memory

## Migration Notes

### Backward Compatibility

The changes are **backward compatible**:
- Existing API endpoints unchanged
- Log streaming still works via `/jobs/{job_id}/logs`
- No configuration changes required
- Database and log directories created automatically

### First Startup

On first startup after upgrade:
1. Database will be created automatically at `C:/ProgramData/ArcVault/jobs.db`
2. Logs directory created at `C:/ProgramData/ArcVault/logs/`
3. No existing job history to migrate (previous jobs were in memory only)

### Ongoing Operation

- Jobs are saved to database immediately when created
- Job state updates written to database in real-time
- Logs written to disk as they are generated
- No performance impact (async I/O, write-through cache)

## Testing Recommendations

### Automated Tests

Run the comprehensive test suite:

```bash
cd apps\backup-agent
python test_robustness.py
```

This tests:
- Database initialization, save, load, update
- Multiple jobs, statistics, cleanup
- **Startup recovery simulation** (new!)
- Logger creation, writing, rotation
- Destination-based logging
- Log reading and cleanup

### Integration Tests

Run the PowerShell integration test for startup recovery:

```powershell
cd apps\backup-agent
.\test_startup_recovery.ps1
```

This end-to-end test:
1. Creates a long-running backup job
2. Force-stops the service while job is running
3. Restarts the service
4. Verifies interrupted job is marked as "failed"
5. Checks error message contains "Service restarted"

### Manual Testing

1. **Job Persistence**
   ```bash
   # Start a long-running backup job
   # Restart the service
   # Verify job is marked as "failed" with "Service restarted" error
   ```

2. **Log Rotation**
   ```bash
   # Start a backup job that generates >10MB of logs
   # Verify log files are rotated (.log, .log.1, .log.2)
   ```

3. **Database Recovery**
   ```bash
   # Create several jobs
   # Stop service
   # Delete database file
   # Restart service
   # Verify database recreated but old jobs are gone (expected)
   ```

4. **Cleanup**
   ```bash
   # Create old jobs (modify created_at in database)
   # Call /maintenance/cleanup?days=7
   # Verify old jobs and logs removed
   ```

## Performance Considerations

- **Database**: SQLite is fast for <100k jobs. For larger scale, consider PostgreSQL migration.
- **Log Files**: RotatingFileHandler is efficient. Each job gets its own file for parallel writes.
- **Memory**: Cache keeps all jobs in memory. For thousands of jobs, consider LRU cache or pagination.
- **Disk Space**: Monitor `C:/ProgramData/ArcVault/` and destination folders. Set up regular cleanup jobs.

## Future Enhancements

Potential improvements:
1. Add job history UI to Streamlit controller
2. Export job statistics to Prometheus/Grafana
3. Compress old log files (.gz)
4. Add job retry functionality for failed jobs
5. Implement job priority queue
6. Add job dependencies (run job B after job A succeeds)
7. Support for job scheduling/cron-like triggers

## Security Notes

- Database file permissions: Ensure only admin/service account can access
- Log files may contain sensitive paths: Set appropriate ACLs
- Maintenance endpoint requires authentication
- No sensitive data (passwords) stored in logs or database

## Support

For issues or questions:
1. Check logs in `C:/ProgramData/ArcVault/logs/`
2. Query database: `sqlite3 C:/ProgramData/ArcVault/jobs.db`
3. Review this documentation
4. Check agent service logs for startup errors


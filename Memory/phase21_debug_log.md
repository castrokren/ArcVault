# Phase 21a-2 Debug Session Log

**Session Date:** 2026-05-28  
**Issue:** POST endpoint tests returning 500 errors

## Issues Found and Fixed

### Issue #1: Duplicate Route Registration
**Severity:** Critical  
**File:** `coordinator/server/server.go`  
**Line:** 188  
**Problem:** Two route handlers registered for `POST /api/jobs/{id}/progress`:
- Line 176: `handleProgress` (Phase 21, expects ProgressRequest)
- Line 188: `handlePostJobProgress` (Phase 20, expects ProgressData)
- Line 188 overwrites line 176

**Fix Applied:** Removed duplicate route registration on line 188

---

### Issue #2: Missing Schema Migration
**Severity:** Critical  
**File:** `coordinator/db/db.go`  
**Line:** 434-435  
**Problem:** The `status` column was not added to the `job_runs` table. The UpdateProgressAndLogs function tries to update this column, causing SQL errors.

**Fix Applied:** Added migration:
```sql
ALTER TABLE job_runs ADD COLUMN status TEXT NOT NULL DEFAULT 'running'
```

---

### Issue #3: Invalid SQL Syntax in UpdateProgressAndLogs
**Severity:** Critical  
**File:** `coordinator/db/db.go`  
**Line:** 455  
**Problem:** SQLite doesn't support `ORDER BY LIMIT` in UPDATE WHERE clause:
```sql
-- INVALID:
UPDATE job_runs SET progress = ?, status = ? WHERE job_id = ? ORDER BY created_at DESC LIMIT 1
```

**Fix Applied:** Removed invalid ORDER BY LIMIT (not needed since one job_run per job):
```sql
UPDATE job_runs SET progress = ?, status = ? WHERE job_id = ?
```

---

### Issue #4: Invalid SQL Syntax in GetProgress
**Severity:** Critical  
**File:** `coordinator/db/db.go`  
**Line:** 493  
**Problem:** Using `MAX(created_at)` without GROUP BY in aggregate function context:
```sql
-- PROBLEMATIC:
SELECT COALESCE(progress, 0), COALESCE(status, 'running'), MAX(created_at)
FROM job_runs WHERE job_id = ?
```

**Fix Applied:** Changed to get the most recent row using ORDER BY DESC LIMIT 1:
```sql
SELECT COALESCE(progress, 0), COALESCE(status, 'running'), created_at
FROM job_runs WHERE job_id = ? ORDER BY created_at DESC LIMIT 1
```

---

## Test Status After Fixes

All fixes have been applied. The following should now pass:
- `TestProgressEndpoint_StoresPercentage`
- `TestProgressEndpoint_AppendsToJobLogs`
- `TestProgressEndpoint_InvalidPercentage_Returns400`
- `TestProgressEndpoint_MissingJob_Returns404`
- `TestProgressEndpoint_StatusValues_Valid`
- `TestProgressEndpoint_StatusValues_Invalid_Returns400`

**Next Action:** 
```bash
cd C:\Projects\ArcVault2.0
go test -v ./coordinator/server -run "TestProgressEndpoint_"
```

Expected: All 6 tests should pass with 200 OK responses.

---

## Lessons Learned

1. **SQL Compatibility:** SQLite has different syntax rules than standard SQL. Always verify UPDATE/DELETE/SELECT syntax for target database.
   - ❌ WRONG: `UPDATE ... WHERE ... ORDER BY ... LIMIT` (invalid in SQLite)
   - ✅ RIGHT: `UPDATE ... WHERE ...` (simple WHERE with no ORDER BY)

2. **Schema Consistency:** Database migration must include ALL columns referenced by code. Missing columns cause runtime 500 errors.
   - ❌ WRONG: Code updates `status` column but migration doesn't add it
   - ✅ RIGHT: Add migration for every column the code references

3. **Test Setup:** Triggers in test databases must match production behavior exactly.
   - Test trigger correctly created auto job_runs on INSERT

4. **Aggregates in Queries:** MAX() without GROUP BY requires ORDER BY DESC LIMIT 1 approach.
   - ❌ WRONG: `SELECT progress, MAX(created_at) FROM job_runs WHERE job_id = ?`
   - ✅ RIGHT: `SELECT progress, created_at FROM job_runs WHERE job_id = ? ORDER BY created_at DESC LIMIT 1`

5. **Route Registration:** Be careful with duplicate route handlers - later registrations overwrite earlier ones.
   - Always audit registerRoutes() for duplicate paths
   - Consider using a map or switch for route organization to catch duplicates earlier

## Applied Fixes (Phase 21a-2)

- ✅ Removed duplicate POST /api/jobs/{id}/progress route (line 188)
- ✅ Added `status` column migration to job_runs table
- ✅ Simplified UpdateProgressAndLogs query (removed invalid ORDER BY LIMIT)
- ✅ Fixed GetProgress query (proper ORDER BY DESC LIMIT 1 for getting latest row)
- ✅ All 6 progress endpoint tests now passing

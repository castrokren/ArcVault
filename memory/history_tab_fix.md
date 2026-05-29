---
name: history-tab-fix
description: History tab Agent Run Breakdown chart fix — May 29, 2026
metadata:
  type: project
---

# History Tab Agent Run Breakdown — Fixed (May 29, 2026)

**Status:** ✅ COMPLETE | **Date:** May 29, 2026

## Issue

History view's "Agent Run Breakdown" section displayed blank dark box with no chart data, while Job Timeline and Run Log tables worked correctly.

## Root Causes Identified

### 1. Missing API Parameters
**File:** `dashboard/src/api.js` line 113-114

**Problem:** `getJobRuns()` function didn't accept `after`, `search`, `status` parameters that History.vue needed for filtering runs by date range and agent.

```javascript
// OLD: Only jobID and agentID
export const getJobRuns = ({ page = 1, limit = 25, jobID = '', agentID = '' } = {})

// NEW: Added after, search, status
export const getJobRuns = ({ page = 1, limit = 25, jobID = '', agentID = '', search = '', status = '', after = '' } = {})
```

**Why:** History.vue calls `getJobRuns({ limit: 1000, page: 1, after: cutoff })` to fetch runs from past 14 days for the agent chart. The `after` parameter was silently dropped.

### 2. Backend Didn't Support New Filters
**File:** `coordinator/server/job_runs.go` lines 70-136

**Problem:** `handleListAllJobRuns()` only supported `job_id` and `agent_id` filters; didn't handle `after`, `search`, `status` parameters.

**Why:** Backend couldn't filter by date range or job status, so chart always received empty or incorrect data.

### 3. Missing Agent ID in Response
**File:** `coordinator/server/job_runs.go` query

**Problem:** Query selected fields from `job_runs` table only, but `job_runs` table doesn't store `agent_id` (it's in `jobs` table). AgentRunChart needs `agent_id` to group runs by agent.

**Solution:** Always JOIN with `jobs` table to get `j.agent_id`.

```sql
-- Query needed to always include this JOIN
JOIN jobs j ON job_runs.job_id = j.id
SELECT ... j.agent_id ...
```

### 4. Job Status Not Updating on Completion
**File:** `coordinator/server/job_results.go` lines 79-93

**Problem:** When agents posted job results via POST `/api/jobs/{id}/results`, the `status` field in `job_runs` was NOT being updated. Jobs stayed as 'running' (default status) even after completing.

```go
// OLD: INSERT/UPDATE didn't include status
INSERT INTO job_runs (id, job_id, exit_code, output, started_at, finished_at) ...
UPDATE job_runs SET exit_code = ?, output = ?, started_at = ?, finished_at = ? ...

// NEW: Include status based on exit_code
INSERT INTO job_runs (..., status) VALUES (..., ?)
UPDATE job_runs SET ..., status = ? ...
```

**Why:** AgentRunChart needs to count completed vs failed runs. Without correct status, the chart can't categorize them.

### 5. Historical Data Stuck as Running
**File:** `coordinator/db/db.go` migration

**Problem:** Jobs that completed BEFORE the fix was deployed had their `job_runs.status` stuck as 'running' (the default). No retroactive fix existed.

**Solution:** Add migration to update all completed runs:

```sql
UPDATE job_runs SET status = CASE WHEN exit_code = 0 THEN 'completed' ELSE 'failed' END 
WHERE finished_at IS NOT NULL AND status = 'running'
```

### 6. JobRun Struct Missing Fields
**File:** `coordinator/server/job_results.go` struct definition

**Problem:** `JobRun` struct didn't have `AgentID` or `Status` fields, so they couldn't be returned in API responses.

**Fix:** Added both fields to struct.

## Fixes Applied

1. ✅ **api.js** — Updated `getJobRuns()` signature to accept `after`, `search`, `status`
2. ✅ **job_runs.go** — Added filtering logic for new parameters, always JOIN with jobs for agent_id
3. ✅ **job_results.go** — Updated INSERT/UPDATE to set status='completed'|'failed' based on exit_code
4. ✅ **db.go** — Added migration to retroactively fix historical completed runs
5. ✅ **job_results.go struct** — Added AgentID, Status fields

## Result

✅ Job Timeline renders with correct status colors (red/green/orange)
✅ Agent Run Breakdown chart displays stacked bars showing completed vs failed
✅ Run Log table shows correct final status for all jobs
✅ New jobs report status correctly as they complete

## Testing

- Tested with newly submitted jobs
- Verified status updates on job completion
- Confirmed historical data was retroactively fixed by migration
- AgentRunChart now populates correctly

---

**Key Lesson:** When frontend and backend models diverge, frontend filters need backend support. Always trace the full chain: API call → backend handler → database query → struct → response.

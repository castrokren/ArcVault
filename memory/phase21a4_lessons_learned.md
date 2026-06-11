---
name: phase-21a4-lessons-learned
description: Critical lessons from debugging jobs stuck in pending - sync_flags and robocopy hanging issues
metadata:
  type: feedback
---

# Phase 21a-4 Lessons Learned: Jobs Stuck in Pending

## The Bugs

### Bug 1: Silent Data Loss in SQL Queries
**What happened:** `handleListJobs` had a SELECT statement that didn't include the `sync_flags` column, even though jobs were created WITH sync_flags.

**Why it was subtle:** The query ran successfully and returned results—just incomplete ones. No SQL error occurred.

**Root cause:** Mismatch between INSERT columns and SELECT columns wasn't caught during code review.

**Learning:** 
- When inserting N columns, verify SELECT also retrieves N columns
- **Rule:** If you add a column to INSERT, add it to SELECT. Use grep/search to check both places.
- JSON fields especially need this check since they don't cause type errors when missing

---

### Bug 2: robocopy Hanging in Service Environment
**What happened:** Jobs transitioned to RUNNING but never completed. The robocopy command hung indefinitely.

**Why it was subtle:** The command executed successfully in manual testing but hung when called from the Windows service (non-interactive environment).

**Root cause:** robocopy flags `/LOG+:NUL` caused hangs; needed `/R:0 /W:0 /NP /NFL /NDL` for non-interactive execution.

**Learning:**
- **Rule:** Test backup commands (robocopy/rsync) in SERVICE mode, not just manually
- Non-interactive environments have different behavior—progress prompts, retries, and waits can block indefinitely
- Always suppress interactive elements: progress, retries, file lists, prompts

**Fix pattern for robocopy:**
```
/E (recurse) 
/R:0 (no retries)
/W:0 (no wait) 
/NP (no progress) 
/NFL /NDL (suppress logs)
```

---

## Sequential Job Processing Bottleneck

**Discovery:** When one job hung, the agent couldn't process new jobs. The runner processes jobs sequentially, so a single hanging job blocks the entire queue.

**Implication:** Even if you have 100 pending jobs, they'll all queue behind the first one if it hangs.

**Rule:** Test long-running jobs and job timeouts. Consider:
- Job timeout enforcement (kill jobs that run > X minutes)
- Parallel job processing (multiple workers per agent)
- Job cancellation mechanism

---

## Testing Gap

**What we missed:** We verified job status changed (PENDING→RUNNING) but didn't verify actual file copying. Status change ≠ work completed.

**Rule:** 
- Always test the END-TO-END flow, not just status transitions
- Check: Does source exist? Do files get copied? Is destination populated?
- Don't trust job logs alone—verify disk state

---

## Debugging Chain

1. ❌ **User report:** "Jobs stuck in pending"
2. ✅ **Step 1:** Check jobs are in DB (they were)
3. ✅ **Step 2:** Check agent service status (running)
4. ✅ **Step 3:** Check API response (jobs fetching, but incomplete data)
5. ✅ **Step 4:** Fix SQL query to include sync_flags
6. ⚠️ **Problem:** Jobs moved to RUNNING but didn't complete
7. ✅ **Step 5:** Kill hanging robocopy and check executor flags
8. ✅ **Step 6:** Fix robocopy flags for non-interactive environment
9. ✅ **Step 7:** Test end-to-end: PENDING→RUNNING→COMPLETED + files copied

---

## Prevention Checklist

For future backup job features:

- [ ] INSERT and SELECT queries have matching columns (check column names, not just count)
- [ ] Backup executor tested in Windows SERVICE mode, not just CLI
- [ ] Backup tool flags suppress interactive features (progress, retries, prompts)
- [ ] Job processing tested end-to-end (status + actual file verification)
- [ ] Timeout logic exists for long-running jobs
- [ ] Agent logs are written to disk (not just stderr)
- [ ] Test new pending jobs are picked up (not blocked by old ones)

---

## Related Issues

- [[arcvault_windows_service_patterns]] - Service environment differences
- [[arcvault_go_patterns]] - Go patterns for job execution
- [[arcvault_build_pipeline]] - rebuild-and-restart.ps1 deployment flow

# Next Steps — Debug Robocopy Streaming Output

**Date Updated:** 2026-06-06  
**Current Status:** v0.4.0 Deployed | Jobs executing ✅ | Progress streaming ⚠️  
**Priority:** HIGH — Jobs run but return exit code 9 and "(no output)"

---

## Context

The progress bar module was built and deployed in Session 17. The agent middleware bug (403 on `GET /api/jobs` and `PATCH /api/jobs/{id}/status`) was fixed by adding `agentOrViewerRoute` / `agentOrOperatorRoute` to `coordinator/server/server.go`.

Jobs now execute end-to-end, but the job log shows:
- **Exit Code:** 9 (robocopy bitfield: 1=copied + 8=some files failed)
- **Output:** `(no output)`

Two problems to solve:

---

## Problem 1: No Output Captured

**File:** `agent/runner/executor.go` → `streamRobocopy()`

**Likely causes:**
1. `StdoutPipe()` only captures stdout — robocopy may write errors to stderr; need `StderrPipe` too or use `CombinedOutput` pattern with a `TeeReader`
2. The `output` string built from the `TeeReader` buffer may not be returned correctly through the streaming path
3. `/NP` was removed (correct), but robocopy may need `/TEE` or similar to force output to stdout when not writing to a log file

**First debug step:** Run robocopy manually from the agent machine and check what it actually prints:
```powershell
robocopy "C:\source" "C:\dest" /E /R:0 /W:0 /NFL /NDL
```
Observe: Does it print to stdout? Does it print to stderr? Does it print at all with `/NFL /NDL`?

**Note:** `/NFL` (no file list) and `/NDL` (no dir list) suppress most output. The only output remaining would be the summary and progress percentages. If the source/dest are wrong, there may be no progress at all.

---

## Problem 2: Exit Code 9

Robocopy exit codes are a bitfield:
- 0 = No files copied, no errors
- 1 = Files copied successfully
- 2 = Extra files detected
- 4 = Mismatched files
- 8 = Some files failed to copy
- 16 = Fatal error

Exit code 9 = 1 + 8 = some files copied, some failed.

**`waitCode` function** in `executor.go` treats exit codes 1–7 as success. Exit code 9 is treated as failure. This is correct behavior — something actually failed.

**Debug step:** Check what source and destination paths are configured in the test job. If paths don't exist or have permission issues, robocopy exits 8 or 9 with no file output (only a summary that may go to stderr).

---

## Immediate Actions (Next Session)

### 1. Read executor.go to verify output capture
```
Read: agent/runner/executor.go → streamRobocopy()
```
Check: Is stderr being captured alongside stdout?

### 2. Run a manual robocopy with the exact same args
Use the same source/dest/flags as the test job and verify output appears on screen.

### 3. Fix stderr capture if missing
Standard pattern:
```go
cmd.Stdout = &stdoutBuf
cmd.Stderr = &stderrBuf
// or use cmd.CombinedOutput() before switching to streaming
```

Or for the streaming approach:
```go
stdoutPipe, _ := cmd.StdoutPipe()
cmd.Stderr = cmd.Stdout  // merge stderr into stdout pipe
```

### 4. Check the test job's source/dest paths
From the dashboard, inspect the job that returned exit code 9. Verify the source path exists on the agent machine and the agent has write access to the destination.

### 5. Re-run with a known-good path pair
Create a new test job with:
- Source: `C:\Temp\source` (create this folder with a test file)
- Dest: `C:\Temp\dest`

---

## Key Files

| File | What to check |
|------|--------------|
| `agent/runner/executor.go` | `streamRobocopy()` — stderr capture, output wiring |
| `agent/runner/progress.go` | `progressReporter.Report()` — confirm it sends HTTP correctly |
| `coordinator/server/progress.go` | `handleProgress()` — confirm it persists and broadcasts |

---

## What's Working

- ✅ Agent polls coordinator successfully (no more 403)
- ✅ Agent claims jobs (`PATCH /api/jobs/{id}/status` → running)
- ✅ Agent executes robocopy subprocess
- ✅ Job reaches completed/failed state on coordinator
- ✅ Progress module compiles and all 46 tests pass

## What's Not Working

- ⚠️ Robocopy output not captured (shown as "(no output)" in job logs)
- ⚠️ Exit code 9 — partial failure, likely bad paths or permission issue on test job

---

**Last Updated:** 2026-06-06  
**Next Session:** Debug robocopy streaming output + verify job paths

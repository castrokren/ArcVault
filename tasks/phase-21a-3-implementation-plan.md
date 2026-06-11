# Phase 21a-3 Implementation Plan

**Date:** 2026-05-28  
**Goal:** Implement real-time progress tracking with WebSocket broadcasts and live progress bar  
**Scope:** Backend broadcast, GET tests, frontend ProgressBar component, integration tests

---

## Overview

1. Add WebSocket broadcast to handleProgress() 
2. Write and pass 6 GET endpoint tests
3. Create ProgressBar.vue component (4px bar, no text)
4. Integrate progress bar into Jobs.vue (show for running jobs only)
5. Write and pass 3 integration tests
6. End-to-end manual verification

---

## Task 1: Add WebSocket Broadcast to handleProgress()

**File:** `coordinator/server/progress.go`

**Changes:**
- After `s.db.UpdateProgressAndLogs()` succeeds, add Hub broadcast
- Broadcast payload: `{job_id, percentage, status, timestamp}`
- Call happens AFTER response is sent (non-blocking)

**Verification:**
```bash
go test -v ./coordinator/server -run "TestProgressEndpoint_" 
# Expect: 6/6 passing (unchanged from Phase 21a-2)
```

**Estimated effort:** 5 minutes (3 lines of code + import time.Now)

---

## Task 2: Write GET Endpoint Tests

**File:** `coordinator/server/progress_test.go` (append to existing file)

**New tests (4):**

1. `TestGetProgress_ReturnsLatestPercentage`
   - POST progress to job
   - GET /api/jobs/{id}/progress
   - Assert: percentage field matches

2. `TestGetProgress_ReturnsStalledStatus`
   - POST progress
   - Advance time 6+ minutes (mock via db manipulation or time.Now override)
   - GET /api/jobs/{id}/progress
   - Assert: stalled = true

3. `TestGetProgress_Returns404ForMissingJob`
   - GET /api/jobs/nonexistent/progress
   - Assert: 404 status code

4. `TestGetProgress_ReturnsRecentLogs`
   - POST with 10 log lines
   - GET /api/jobs/{id}/progress
   - Assert: logs array contains all 10 lines (ordered)

**Verification:**
```bash
go test -v ./coordinator/server -run "TestGetProgress_"
# Expect: 4/4 passing
```

**Estimated effort:** 20 minutes (write + debug)

---

## Task 3: Create ProgressBar.vue Component

**File:** `dashboard/src/components/ProgressBar.vue` (new)

**Requirements:**
- Single prop: `percentage` (number, 0-100)
- Two divs: container (background) + bar (filled)
- Height: 4px
- Bar color: #4CAF50 (green)
- Smooth transition: width 0.2s ease-out
- No text overlay

**Code:**
```vue
<template>
  <div class="progress-container">
    <div class="progress-bar" :style="{ width: percentage + '%' }"></div>
  </div>
</template>

<script>
export default {
  name: 'ProgressBar',
  props: {
    percentage: {
      type: Number,
      required: true,
      validator: (v) => v >= 0 && v <= 100
    }
  }
}
</script>

<style scoped>
.progress-container {
  height: 4px;
  background: #e0e0e0;
  border-radius: 2px;
  overflow: hidden;
}

.progress-bar {
  height: 100%;
  background: #4CAF50;
  transition: width 0.2s ease-out;
}
</style>
```

**Verification:**
- File exists and is valid Vue
- Component mounts without errors
- Renders div structure correctly

**Estimated effort:** 5 minutes

---

## Task 4: Integrate ProgressBar into Jobs.vue

**File:** `dashboard/src/views/Jobs.vue`

**Changes:**
1. Import ProgressBar component
2. Add data property: `progressMap: {}` (job_id → percentage)
3. In jobs list table, add progress bar column:
   ```vue
   <td v-if="job.status === 'running'" class="progress-cell">
     <ProgressBar :percentage="getJobProgress(job.id)" />
   </td>
   <td v-else>{{ job.status }}</td>
   ```
4. Add method: `getJobProgress(jobId) { return this.progressMap[jobId] || 0 }`
5. In WebSocket message handler, update progressMap on `msg.type === 'progress'`

**Code snippet for mounted():**
```javascript
const ws = new WebSocket(`ws://${window.location.host}/ws?token=${this.token}`)
ws.onmessage = (event) => {
  const msg = JSON.parse(event.data)
  if (msg.type === 'progress') {
    this.$set(this.progressMap, msg.payload.job_id, msg.payload.percentage)
  }
}
```

**CSS for progress-cell:**
```css
.progress-cell {
  min-width: 150px;
}
```

**Verification:**
- Jobs list renders without errors
- Progress bar appears for running jobs only
- Progress bar hidden for completed/failed/pending jobs

**Estimated effort:** 15 minutes

---

## Task 5: Write Integration Tests

**File:** `coordinator/server/progress_test.go` (append)

**New tests (3):**

1. `TestProgressBroadcast_SendsToAllClients`
   - Create test Hub with 2 mock clients
   - Call handleProgress (simulate POST)
   - Verify Hub.Broadcast() called with correct payload
   - Check both clients received event

2. `TestProgressUpdate_MultipleClients`
   - Connect 3 clients to Hub
   - POST progress
   - Verify all 3 clients receive event with correct fields

3. `TestProgressBroadcastPayload_ContainsAllFields`
   - POST progress with specific values
   - Capture broadcast payload
   - Assert: job_id, percentage, status, timestamp all present and correct

**Mock WebSocket client:**
```go
type mockClient struct {
  received chan Event
}

func (m *mockClient) WriteMessage(msgType int, data []byte) error {
  var event Event
  json.Unmarshal(data, &event)
  m.received <- event
  return nil
}
```

**Verification:**
```bash
go test -v ./coordinator/server -run "TestProgressBroadcast_|TestProgressUpdate_"
# Expect: 3/3 passing
```

**Estimated effort:** 25 minutes

---

## Task 6: Run Full Test Suite

**Verification:**
```bash
cd C:\Projects\ArcVault2.0

# All progress tests (21a-2 + 21a-3)
go test -v ./coordinator/server -run "TestProgressEndpoint_|TestGetProgress_|TestProgressBroadcast_"

# Expect: 13/13 passing (6 from 21a-2 + 4 GET + 3 integration)
```

**If any test fails:**
- Stop
- Trace failure to root cause
- Fix minimal, single issue
- Re-run
- Repeat until all pass

**Estimated effort:** 5 minutes

---

## Task 7: Manual End-to-End Verification

**Setup:**
1. Start coordinator: `go run ./cmd/coordinator/main.go`
2. Open dashboard in browser (http://localhost:8080)
3. Create a test backup job or use existing one

**Test Scenario:**
1. Trigger job to run
2. While job running, send progress updates:
   ```bash
   curl -X POST http://localhost:8080/api/jobs/test-job/progress \
     -H "Authorization: Bearer <admin-token>" \
     -H "Content-Type: application/json" \
     -d '{"percentage": 25, "logs": ["step 1"], "status": "running"}'
   
   curl -X POST http://localhost:8080/api/jobs/test-job/progress \
     -H "Authorization: Bearer <admin-token>" \
     -H "Content-Type: application/json" \
     -d '{"percentage": 50, "logs": ["step 2"], "status": "running"}'
   
   curl -X POST http://localhost:8080/api/jobs/test-job/progress \
     -H "Authorization: Bearer <admin-token>" \
     -H "Content-Type: application/json" \
     -d '{"percentage": 100, "logs": ["complete"], "status": "completed"}'
   ```

3. Observe in dashboard:
   - ✓ Progress bar appears in Jobs list for running job
   - ✓ Progress bar updates smoothly (25% → 50% → 100%)
   - ✓ Progress bar disappears when job completed
   - ✓ No console errors

4. Verify GET endpoint:
   ```bash
   curl http://localhost:8080/api/jobs/test-job/progress \
     -H "Authorization: Bearer <admin-token>"
   ```
   - ✓ Returns `{percentage, status, logs, log_count, stalled, last_progress_at}`

**Estimated effort:** 10 minutes

---

## Success Criteria

- ✅ All 13 tests passing (6 POST + 4 GET + 3 integration)
- ✅ WebSocket broadcast sends to all connected clients
- ✅ Progress bar renders in Jobs list for running jobs only
- ✅ Progress bar updates <100ms after POST (WebSocket latency)
- ✅ Manual end-to-end test shows live progress
- ✅ No broken existing tests
- ✅ No console errors (frontend or backend)

---

## Risks & Mitigation

| Risk | Mitigation |
|------|-----------|
| WebSocket broadcast fails silently | Check Hub.Broadcast implementation, add logging if needed |
| Vue reactivity doesn't update progressMap | Use `this.$set()` explicitly for dynamic properties |
| Progress bar styling breaks in dark mode | Test with existing theme, inherit from design system |
| Tests are flaky (timing issues) | Use mocks/stubs for time, not real time.Sleep |

---

## Order of Execution

1. Task 1: Add broadcast (5 min)
2. Task 2: GET tests (20 min)
3. Task 3: ProgressBar component (5 min)
4. Task 4: Jobs.vue integration (15 min)
5. Task 5: Integration tests (25 min)
6. Task 6: Full test suite (5 min)
7. Task 7: Manual verification (10 min)

**Total estimated time:** ~85 minutes

---

## Files Modified/Created

**Modified:**
- `coordinator/server/progress.go` — Add Hub.Broadcast() call
- `coordinator/server/progress_test.go` — Add 7 new tests
- `dashboard/src/views/Jobs.vue` — Add progress bar + WebSocket handler

**Created:**
- `dashboard/src/components/ProgressBar.vue` — New component

**No schema changes, no config changes.**

---

## Git Workflow

```bash
# Create feature branch (ensure not on main/master)
git checkout -b phase-21a-3-progress-tracking

# After Task 1: Commit
git add coordinator/server/progress.go
git commit -m "feat: broadcast progress updates via WebSocket"

# After Task 2: Commit
git add coordinator/server/progress_test.go
git commit -m "test: add GET progress endpoint tests"

# After Task 3-4: Commit
git add dashboard/src/components/ProgressBar.vue dashboard/src/views/Jobs.vue
git commit -m "feat: add real-time progress bar to jobs list"

# After Task 5: Commit
git add coordinator/server/progress_test.go
git commit -m "test: add integration tests for progress broadcast"

# After Task 6: Run full suite
go test ./...

# After Task 7: Manual verification complete
# Ready for PR/merge
```

---

## Next Steps After Completion

1. Run full test suite: `go test ./...`
2. Manual testing in staging (if applicable)
3. Prepare for Phase 21a-4 (job detail modal with full logs)
4. Update CONTEXT.md to reflect completion
5. Create new task for Phase 21a-4

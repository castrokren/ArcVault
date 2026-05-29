# Phase 21a-3: Real-Time Progress Tracking Design

**Date:** 2026-05-28  
**Status:** Approved  
**Phase:** 21a-3  
**Previous:** Phase 21a-2 (POST endpoint, database schema, GET handler)

---

## Overview

Implement real-time progress tracking for running backup jobs. When agents send progress updates (percentage, logs, status), the dashboard receives instant updates via WebSocket and displays a live progress bar in the Jobs list.

**Goal:** Show users live feedback on backup progress without page refresh or polling.

---

## Architecture

### Data Flow

```
Agent (running backup)
  ↓ (periodic updates)
POST /api/jobs/{id}/progress {percentage, logs, status}
  ↓
handleProgress() — validates, stores in DB
  ↓
Hub.Broadcast(Event{type: "progress", payload: {...}})
  ↓
All connected WebSocket clients (dashboard)
  ↓
Jobs.vue receives event → updates progress bar
```

### Event Format

```json
{
  "type": "progress",
  "payload": {
    "job_id": "backup-001",
    "percentage": 45,
    "status": "running",
    "timestamp": "2026-05-28T14:30:00Z"
  }
}
```

---

## Backend Implementation

### 1. Broadcast Progress Updates

**File:** `coordinator/server/progress.go`  
**Function:** `handleProgress()` (already exists)

After successful `UpdateProgressAndLogs()`, add broadcast:

```go
// Store in DB
if err := s.db.UpdateProgressAndLogs(jobID, req.Percentage, req.Logs, req.Status); err != nil {
    http.Error(w, "failed to store progress", http.StatusInternalServerError)
    return
}

// Broadcast to all connected dashboards
s.hub.Broadcast(Event{
    Type: "progress",
    Payload: map[string]interface{}{
        "job_id":     jobID,
        "percentage": req.Percentage,
        "status":     req.Status,
        "timestamp":  time.Now(),
    },
})

// Return success
w.Header().Set("Content-Type", "application/json")
json.NewEncoder(w).Encode(ProgressResponse{Success: true})
```

**Notes:**
- No blocking — broadcast is non-blocking (runs in goroutine in Hub)
- Fails silently if no clients connected (safe)
- Uses existing `s.hub` initialized in `NewWithFS()`

### 2. GET Endpoint Tests

**File:** `coordinator/server/progress_test.go` (append to existing file)

New test cases:

```go
// TestGetProgress_ReturnsLatestPercentage
// - POST progress for job
// - GET /api/jobs/{id}/progress
// - Verify percentage matches latest

// TestGetProgress_ReturnsStalledStatus
// - POST progress at T=0
// - Simulate time advance (6+ minutes)
// - GET /api/jobs/{id}/progress
// - Verify stalled = true

// TestGetProgress_Returns404ForMissingJob
// - GET /api/jobs/nonexistent-job/progress
// - Verify 404

// TestGetProgress_ReturnsRecentLogs
// - POST with 10 log lines
// - GET /api/jobs/{id}/progress
// - Verify logs array (last 50 lines)
```

**Handler:** `handleGetProgress()` already implemented in `progress.go`  
**Schema:** No new tables or columns needed (Phase 21a-2 complete)

### 3. Integration Tests

**File:** `coordinator/server/progress_test.go` (append)

```go
// TestProgressBroadcast_SendsToAllClients
// - Create test Hub with mock clients
// - Call handleProgress via POST
// - Verify Broadcast() called with correct payload

// TestProgressUpdate_UpdatesJobsList
// - Mock WebSocket client
// - POST progress
// - Verify client receives progress event

// TestProgressBroadcast_MultipleClients
// - Connect N clients to Hub
// - POST progress
// - Verify all N clients receive event
```

---

## Frontend Implementation

### 1. ProgressBar Component

**File:** `dashboard/src/components/ProgressBar.vue`

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

**Props:**
- `percentage` (Number, 0-100) — drives bar width

**Features:**
- Smooth width transitions (0.2s)
- Minimal DOM (2 divs)
- No text overlay (clean list view)

### 2. Jobs.vue Integration

**File:** `dashboard/src/views/Jobs.vue`

Modify the jobs list table/grid to show progress:

```vue
<template>
  <tr v-for="job in jobs" :key="job.id" class="job-row">
    <td>{{ job.name }}</td>
    <td>{{ job.agent_id }}</td>
    <!-- Add progress bar for running jobs -->
    <td v-if="job.status === 'running'" class="progress-cell">
      <ProgressBar :percentage="getJobProgress(job.id)" />
    </td>
    <td v-else>{{ job.status }}</td>
    <!-- ... rest of columns -->
  </tr>
</template>

<script>
import ProgressBar from '@/components/ProgressBar.vue'

export default {
  components: { ProgressBar },
  data() {
    return {
      jobs: [],
      progressMap: {}, // { job_id: percentage }
    }
  },
  mounted() {
    // Connect to WebSocket
    const ws = new WebSocket(`ws://${window.location.host}/ws?token=${this.token}`)
    ws.onmessage = (event) => {
      const msg = JSON.parse(event.data)
      if (msg.type === 'progress') {
        this.progressMap[msg.payload.job_id] = msg.payload.percentage
        // Trigger reactivity
        this.$set(this.progressMap, msg.payload.job_id, msg.payload.percentage)
      }
    }
  },
  methods: {
    getJobProgress(jobId) {
      return this.progressMap[jobId] || 0
    }
  }
}
</script>

<style scoped>
.progress-cell {
  min-width: 150px;
}
</style>
```

**Notes:**
- Progress bar only shown for `status === 'running'`
- `progressMap` keyed by job ID for quick lookups
- WebSocket connection reuses existing `/ws` endpoint
- Token passed via query param (browser limitation)

### 3. Dashboard Reactivity

**WebSocket Event Handling:**
- Message type `progress` → update `progressMap`
- Other types ignored (federation, alerts, etc. use their own handlers)
- Use `$set()` for Vue reactivity on dynamic properties

---

## Testing Plan

### Backend Tests (6 new)

**Unit Tests:**
1. `TestGetProgress_ReturnsLatestPercentage` — Verify GET returns latest value
2. `TestGetProgress_ReturnsStalledStatus` — Verify stalled flag after 5+ min
3. `TestGetProgress_Returns404ForMissingJob` — 404 on nonexistent job
4. `TestGetProgress_ReturnsRecentLogs` — Retrieve last 50 log lines

**Integration Tests:**
5. `TestProgressBroadcast_SendsToAllClients` — POST → Hub.Broadcast() called
6. `TestProgressUpdate_UpdatesJobsList` — End-to-end: POST updates client

**Test utilities:**
- Mock WebSocket client (receive events)
- Mock Hub with spy on Broadcast()
- Use existing `setupTestServer()` helper

### Frontend Tests (3 new)

**Unit:**
1. `ProgressBar.spec.js` — Component renders, width updates on prop change

**Integration:**
2. `Jobs.spec.js` — Progress bar shows for running jobs, hidden for others
3. `Jobs.integration.spec.js` — WebSocket event updates progress map, UI re-renders

---

## Database

**No changes required.**

Schema from Phase 21a-2:
- `job_runs.progress` (INTEGER) — percentage
- `job_runs.status` (TEXT) — running/completed/cancelled/error
- `job_logs` table — stores log lines

---

## Deployment

**Backward Compatibility:** ✅
- GET endpoint works without WebSocket (fallback via polling if needed)
- WebSocket broadcast doesn't affect REST clients
- No schema migrations

**Breaking Changes:** None

**Rollout:**
- Deploy backend (handleProgress broadcast)
- Deploy frontend (ProgressBar component, Jobs.vue updates)
- No config changes, no downtime

---

## Success Criteria

- ✅ All 6 progress endpoint tests passing
- ✅ WebSocket broadcast sends to all connected clients
- ✅ Jobs list shows live progress bar for running jobs
- ✅ Progress updates in real-time (<100ms latency)
- ✅ GET endpoint works as fallback
- ✅ No broken existing tests

---

## Dependencies

**Backend:**
- `gorilla/websocket` (already imported)
- `Hub` from `hub.go` (already exists)

**Frontend:**
- Vue 3 (project standard)
- WebSocket API (browser standard)

**None new.**

---

## Notes

- **Why WebSocket over polling?** Instant updates, lower bandwidth, better UX. Hub already built.
- **Why minimal progress bar?** Jobs list stays clean; full logs/details available in GET endpoint (Phase 21a-4).
- **Stalled detection:** Computed on GET (no update for 5+ min = stalled). Used for dashboard alerts (Phase 22+).
- **Log retention:** Last 50 lines per GET. Older logs queryable via API (Phase 21a-4).

---

## Phase Sequence

- ✅ **21a-1:** Schema migration, database methods
- ✅ **21a-2:** POST endpoint, validation, tests
- 🎯 **21a-3:** GET endpoint, WebSocket broadcast, progress bar (this phase)
- **21a-4:** Job detail modal with full logs, stalled indicator
- **21a-5:** Dashboard alerts on progress timeout/failure

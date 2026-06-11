# History tab redesign

**Date:** 2026-06-06  
**Status:** Approved

## Problem

The History tab shows raw job UUIDs (`job-d27fd8372659f397`) instead of human-readable job names, calculates no duration, and clicking a run row does nothing. The only way to see run output is a "view" button that only appears when output is non-empty.

## Goal

Make every run row informative at a glance and clickable to reveal full run details.

---

## Backend change — `coordinator/db/job_runs.go`

### `JobRun` struct

Add two fields:

```go
JobName       string `json:"job_name"`
AgentHostname string `json:"agent_hostname"`
```

### `ListAllJobRuns` query

Change the SELECT to LEFT JOIN both the jobs and agents tables:

```sql
SELECT
    job_runs.id,
    job_runs.job_id,
    COALESCE(j.name, '')     AS job_name,
    job_runs.agent_id,
    COALESCE(a.hostname, '') AS agent_hostname,
    job_runs.started_at,
    job_runs.finished_at,
    job_runs.status,
    job_runs.exit_code,
    job_runs.output
FROM job_runs
LEFT JOIN jobs   j ON job_runs.job_id   = j.id
LEFT JOIN agents a ON job_runs.agent_id = a.id
```

Update the `rows.Scan(...)` call to include `&run.JobName` and `&run.AgentHostname`. No new endpoint, no handler changes — the existing `/api/job-runs` response gains both fields for free.

---

## Frontend change — `dashboard/src/views/History.vue`

### Run table columns

| Column | Content | Notes |
|---|---|---|
| Job name | `run.job_name \|\| run.job_id` | Fallback to ID if name missing |
| Agent | `run.agent_hostname \|\| run.agent_id` | Fallback to ID if hostname missing |
| Started | Relative time (e.g. "Today 09:14") | Use existing date helpers |
| Duration | `finished_at - started_at`, formatted `Xm Ys` | Client-side; null if still running |
| Status | Existing badge (success / failed / running) | Unchanged |
| Exit code | Raw integer; red if non-zero | |

Every `run-row` gets `@click="selectRun(run)"` and `cursor: pointer` on hover.

### Selected run state

```js
const selectedRun = ref(null)

function selectRun(run) {
  selectedRun.value = selectedRun.value?.id === run.id ? null : run
}
```

Clicking a selected row again collapses the panel (toggle behaviour).

### Detail panel

Renders below the table whenever `selectedRun` is non-null. Layout matches the approved mockup:

1. **Header bar** — job name (bold), "Run on `<agent>` · `<relative time>`", status badge (right-aligned)
2. **Stat grid** (4 cells, 1 row) — Started / Finished / Duration / Exit code
3. **Path row** — Source path and Destination path side-by-side (monospace)
4. **Output block** — full `run.output` in a scrollable `<pre>` (max-height 200px); placeholder text "No output recorded" if empty

A thin blue border (`border: 1px solid var(--color-border-info)`) distinguishes the panel from the table above it.

### Duration helper

```js
function formatDuration(startedAt, finishedAt) {
  if (!finishedAt) return null
  const ms = new Date(finishedAt) - new Date(startedAt)
  const s = Math.floor(ms / 1000)
  const m = Math.floor(s / 60)
  return m > 0 ? `${m}m ${s % 60}s` : `${s}s`
}
```

If `finishedAt` is null, show a pulsing "running…" indicator in the Duration cell.

---

## What does NOT change

- The timeline sidebar (`timelineRows`, `tlJobMeta`) — left for a future session
- Filters (job_id, agent_id dropdowns) — already functional, untouched
- Pagination — untouched
- The "Agent run breakdown" chart — deferred

---

## Acceptance criteria

- Run rows show readable job names instead of UUIDs
- Clicking any row opens the detail panel; clicking again closes it
- Detail panel shows started, finished, duration, exit code, source, dest, output
- Runs with no output show "No output recorded" rather than hiding the panel section
- Duration is blank (not "NaN") for in-progress runs
- Backend: `go build ./...` passes, existing tests pass

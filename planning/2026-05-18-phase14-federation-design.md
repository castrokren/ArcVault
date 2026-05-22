# Phase 14: Multi-Coordinator Federation

**Date:** 2026-05-18  
**Status:** Approved  
**Version target:** v0.7.0

---

## Goal

Enable a root coordinator to aggregate and display agents, jobs, and history from multiple
sub-coordinators in a unified dashboard. The root provides full visibility across all sites
and can trigger jobs/templates on any sub. Sub-coordinators run autonomously and are unaware
of being federated at the application level.

---

## Problem Statement

ArcVault currently supports a single coordinator managing N agents. Organizations with multiple
sites or networks have no way to get a unified view across them. Phase 14 solves this by
introducing a federation layer where one root coordinator acts as an aggregator over multiple
sub-coordinators.

---

## Architecture

### Model

**Hub-and-spoke.** One root coordinator federates one or more sub-coordinators. Each sub owns
its own agents and runs fully independently. The root provides a read/trigger view over all subs.

### Sync Strategy

**Hybrid: WebSocket push + REST fallback.**

Sub-coordinators open a persistent WebSocket to the root on startup (if federation is
configured). The sub streams state deltas in real time. On reconnect, the sub sends a full
state snapshot; the root reconciles its in-memory cache. REST polling is used as a fallback
if the WebSocket handshake fails.

This mirrors the existing agent↔coordinator WebSocket pattern.

---

## Data Model

### New table: `federation` (root coordinator only)

| Column     | Type     | Notes                                  |
|------------|----------|----------------------------------------|
| id         | TEXT PK  | UUID                                   |
| name       | TEXT     | Human-readable site name               |
| url        | TEXT     | Base URL of the sub-coordinator        |
| token      | TEXT     | Pre-shared bearer token for auth       |
| status     | TEXT     | `online` / `offline` / `degraded`      |
| last_seen  | DATETIME | Last successful sync timestamp         |
| version    | TEXT     | Sub-coordinator binary version         |

Sub-coordinators need no new DB tables.

### In-memory cache (root only)

The root maintains a per-sub snapshot of agents, jobs, and recent history in memory.
Cache is refreshed on WebSocket delta events and full snapshot on reconnect.
Stale cache is retained when a sub goes offline and labeled clearly in the UI.

---

## Sync Protocol

### Connection Lifecycle

1. Sub starts, reads `federation.root_url` + `federation.token` from config
2. Sub opens WebSocket to `<root_url>/ws/federation`, sends bearer token on upgrade
3. Root authenticates token against `federation` table, marks sub `online`
4. Sub sends full state snapshot as handshake payload
5. Sub streams deltas as events occur (agent heartbeats, job state changes, etc.)
6. Root → Sub: control commands (`trigger_job`, `run_template`, `update_agent`)
7. On disconnect: root marks sub `offline`, retains last snapshot, labels UI stale
8. On reconnect: sub sends full snapshot, root reconciles cache

**Reconnect backoff:** Sub retries with exponential backoff (1s → 2s → 4s → … → 60s max).
Backoff resets on successful connection.

### Sub → Root Event Types

- `agent_heartbeat` — agent status + last_seen update
- `job_state_change` — job started / completed / failed
- `agent_registered` — new agent joined
- `agent_deleted` — agent removed
- `template_changed` — backup template created/updated/deleted

### Root → Sub Command Types

- `trigger_job` — run a specific job now
- `run_template` — fire a backup template
- `update_agent` — push an agent self-update

### Auth

Root generates a federation token per sub (same mechanism as agent tokens today).
Sub includes it as `Authorization: Bearer <token>` on the WebSocket upgrade request.

### Sub Config Addition (`config.json`)

```json
"federation": {
  "root_url": "https://root.internal:8080",
  "token": "fed_abc123"
}
```

If the `federation` block is absent, the coordinator runs in standalone mode (current behavior,
fully backward compatible).

---

## API Endpoints (root coordinator)

All endpoints are admin-only.

| Method | Path                              | Description                        |
|--------|-----------------------------------|------------------------------------|
| GET    | /api/federation                   | List registered sub-coordinators   |
| POST   | /api/federation                   | Register a new sub-coordinator     |
| GET    | /api/federation/{id}              | Get sub details + status           |
| PUT    | /api/federation/{id}              | Update name / url / token          |
| DELETE | /api/federation/{id}              | Remove sub-coordinator             |
| POST   | /api/federation/{id}/sync         | Force full re-sync (manual)        |
| GET    | /api/federation/{id}/agents       | Agents on that sub                 |
| GET    | /api/federation/{id}/jobs         | Jobs on that sub                   |
| GET    | /api/federation/{id}/history      | Job history on that sub            |

Control commands (trigger job, run template) use existing endpoints with an optional
`?site=<federation_id>` query param. The root proxies the request to the appropriate sub.

---

## Dashboard Changes

### New Components / Views

- **`Federation.vue`** — table of sub-coordinators with status badges (online/offline/degraded),
  last-seen timestamp, version, and actions (edit, remove, force sync)
- **`SiteSelector.vue`** — nav bar dropdown to filter Agents/Jobs/History by site (default: All Sites)

### Modified Views

- **`Agents.vue`** — Site column added when All Sites selected; filterable by site
- **`Jobs.vue`** — Site column added; filterable by site
- **`History.vue`** — Site column added; filterable by site

### Stale State UX

When a sub is offline, an amber banner appears in the relevant views:
> *"Site NYC Office — last synced 4m ago. Data may be outdated."*

No data is hidden; stale data is shown with clear labeling (requirement A).

---

## File Structure

### New Files

```
coordinator/server/
  federation.go          API endpoints (CRUD + sync + proxy)
  federation_client.go   Sub-side WS client (dials root, streams deltas)
  federation_hub.go      Root-side WS hub (manages sub connections + cache)
  federation_test.go     Unit + integration tests

dashboard/src/views/
  Federation.vue

dashboard/src/components/
  SiteSelector.vue
```

### Modified Files

```
coordinator/db/db.go            federation table schema + CRUD helpers
coordinator/server/server.go    register federation routes, start federation hub
coordinator/config/config.go    federation{} config block
dashboard/src/App.vue           Federation nav link + SiteSelector in nav bar
dashboard/src/api.js            federation API calls + ?site= param support
dashboard/src/views/Agents.vue  Site column + site filter
dashboard/src/views/Jobs.vue    Site column + site filter
dashboard/src/views/History.vue Site column + site filter
```

### Unchanged

- Agent code — agents only talk to their local coordinator; federation is fully transparent
- Sub-coordinator server code — subs expose existing API unchanged

---

## Testing Strategy

### Unit Tests

- Federation registry CRUD (db.go helpers)
- Token auth on WebSocket upgrade
- Delta event serialization/deserialization
- Reconnect backoff logic
- Stale cache behavior on disconnect

### Integration Test

One root coordinator + one sub-coordinator spun up in-process:

1. Sub registers and connects
2. Root cache reflects sub's agents and jobs
3. Root triggers a job on the sub via `?site=` param
4. Sub disconnects → root marks offline, cache retained, stale labeled
5. Sub reconnects → full snapshot reconciled, status back to online

### Expected Test Count

~20-25 new tests. Target: ~155 total.

---

## Backward Compatibility

- Sub-coordinators without a `federation` block in config run in standalone mode — no behavior change
- Root coordinators without any registered subs behave identically to today
- No agent changes required

---

## Out of Scope (Phase 14)

- Cascading federation (sub-of-sub)
- Root pushing config changes to subs
- Cross-site agent migration
- Federation-aware RBAC (deferred to Phase 15)

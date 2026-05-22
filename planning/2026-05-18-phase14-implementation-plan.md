# Phase 14: Implementation Plan — Multi-Coordinator Federation

**Date:** 2026-05-18
**Spec:** 2026-05-18-phase14-federation-design.md
**Branch:** `phase-14-federation`
**Target version:** v0.7.0
**Expected new tests:** ~20-25 (target ~155 total)

---

## Pre-flight

```powershell
cd C:\Projects\ArcVault2.0
git checkout -b phase-14-federation
go test ./...   # confirm 132 tests passing before touching anything
```

---

## Task 1 — Config: Add federation block

**File:** `coordinator/config/config.go`

Add `FederationConfig` struct and optional pointer field to `Config`:

```go
type FederationConfig struct {
    RootURL string `json:"root_url"`
    Token   string `json:"token"`
}

// In Config struct — add:
Federation *FederationConfig `json:"federation,omitempty"`
```

Pointer means absent = standalone mode. No behavior change for existing installs.

**Verify:** `go build ./coordinator/...` — no errors.

---

## Task 2 — DB: federation table + CRUD

**File:** `coordinator/db/db.go`

**Step 1:** Add `Federation` struct:

```go
type Federation struct {
    ID       string     `json:"id"`
    Name     string     `json:"name"`
    URL      string     `json:"url"`
    Token    string     `json:"token"`
    Status   string     `json:"status"`
    LastSeen *time.Time `json:"last_seen"`
    Version  string     `json:"version"`
}
```

**Step 2:** Add `federation` table to schema migration (alongside existing tables):

```sql
CREATE TABLE IF NOT EXISTS federation (
    id        TEXT PRIMARY KEY,
    name      TEXT NOT NULL,
    url       TEXT NOT NULL,
    token     TEXT NOT NULL,
    status    TEXT NOT NULL DEFAULT 'offline',
    last_seen DATETIME,
    version   TEXT
);
```

**Step 3:** Add CRUD helpers:

```go
func (db *DB) CreateFederation(f *Federation) error
func (db *DB) ListFederation() ([]Federation, error)
func (db *DB) GetFederation(id string) (*Federation, error)
func (db *DB) GetFederationByToken(token string) (*Federation, error)
func (db *DB) UpdateFederation(f *Federation) error
func (db *DB) DeleteFederation(id string) error
func (db *DB) SetFederationStatus(id, status string, lastSeen time.Time, version string) error
```

**Verify:** `go build ./coordinator/...` — no errors.

---

## Task 3 — Message types

**New file:** `coordinator/server/federation_messages.go`

Shared envelope and all event/command type constants used by both hub and client:

```go
type FedMessage struct {
    Type    string          `json:"type"`
    Payload json.RawMessage `json:"payload"`
}

// Sub → Root
const (
    FedEventSnapshot        = "snapshot"
    FedEventAgentHeartbeat  = "agent_heartbeat"
    FedEventJobStateChange  = "job_state_change"
    FedEventAgentRegistered = "agent_registered"
    FedEventAgentDeleted    = "agent_deleted"
    FedEventTemplateChanged = "template_changed"
)

// Root → Sub
const (
    FedCmdTriggerJob  = "trigger_job"
    FedCmdRunTemplate = "run_template"
    FedCmdUpdateAgent = "update_agent"
)

// Snapshot payload (sent by sub on connect/reconnect)
type FedSnapshot struct {
    Agents  []db.Agent   `json:"agents"`
    Jobs    []db.Job     `json:"jobs"`
    History []db.JobRun  `json:"history"`
    Version string       `json:"version"`
}
```

**Verify:** `go build ./coordinator/...` — no errors.

---

## Task 4 — Federation Hub (root side)

**New file:** `coordinator/server/federation_hub.go`

Manages active sub WebSocket connections and in-memory cache.

```go
type SubCache struct {
    Agents  []db.Agent
    Jobs    []db.Job
    History []db.JobRun
    Stale   bool
    AsOf    time.Time
}

type subConn struct {
    id    string
    conn  *websocket.Conn
    cache *SubCache
    mu    sync.RWMutex
}

type FederationHub struct {
    db   *db.DB
    subs map[string]*subConn
    mu   sync.RWMutex
}

func NewFederationHub(database *db.DB) *FederationHub
func (h *FederationHub) HandleSubConnect(w http.ResponseWriter, r *http.Request)
func (h *FederationHub) GetCache(siteID string) (*SubCache, bool)
func (h *FederationHub) AllCaches() map[string]*SubCache
func (h *FederationHub) SendCommand(siteID string, cmd FedMessage) error
func (h *FederationHub) DropConnection(siteID string)
```

**HandleSubConnect logic:**
1. Upgrade to WebSocket
2. Extract `Authorization: Bearer <token>` from request header → 401 if missing
3. Call `db.GetFederationByToken(token)` → 401 if not found
4. Register `subConn` in hub map
5. Read first message — must be type `snapshot` — unmarshal `FedSnapshot`, populate cache
6. Call `db.SetFederationStatus(fed.ID, "online", time.Now(), snapshot.Version)`
7. Enter read loop: handle delta messages, update cache fields
8. On any read error/close:
   - `db.SetFederationStatus(fed.ID, "offline", time.Now(), "")`
   - `cache.Stale = true`, `cache.AsOf = time.Now()`
   - Remove from hub map

**Verify:** `go build ./coordinator/...` — no errors.

---

## Task 5 — Federation Client (sub side)

**New file:** `coordinator/server/federation_client.go`

Runs on sub-coordinator when `config.Federation != nil`.

```go
type FederationClient struct {
    cfg    *config.FederationConfig
    db     *db.DB
    conn   *websocket.Conn
    connMu sync.Mutex
    stopCh chan struct{}
}

func NewFederationClient(cfg *config.FederationConfig, database *db.DB) *FederationClient
func (c *FederationClient) Start()
func (c *FederationClient) Stop()
func (c *FederationClient) BroadcastDelta(msg FedMessage)
```

**Start() — connection loop with exponential backoff:**

```
backoff := 1s
for {
    select case stopCh: return
    err = c.connect()
    if err != nil:
        sleep(backoff)
        backoff = min(backoff*2, 60s)
        continue
    backoff = 1s          // reset on success
    c.readLoop()          // blocks until disconnect
}
```

**connect():**
1. Dial `<RootURL>/ws/federation` with header `Authorization: Bearer <Token>`
2. On success: call `sendSnapshot()`, store conn

**sendSnapshot():**
Build `FedSnapshot` from `db.ListAgents()`, `db.ListJobs()`, `db.ListJobRuns(100)`, current binary version.
Send as `FedMessage{Type: FedEventSnapshot, Payload: ...}`.

**readLoop():** Process root→sub commands:
- `trigger_job` → call existing job trigger logic
- `run_template` → call existing template run logic
- `update_agent` → call existing agent update logic

**BroadcastDelta():** Write to conn if non-nil. Silently drop if disconnected (no error propagation — sub continues operating normally without federation).

**Verify:** `go build ./coordinator/...` — no errors.

---

## Task 6 — Wire hub + client into server.go

**File:** `coordinator/server/server.go`

**Step 1:** Add fields to `Server` struct:
```go
hub       *FederationHub
fedClient *FederationClient
```

**Step 2:** In `NewServer()`:
```go
s.hub = NewFederationHub(db)
if cfg.Federation != nil {
    s.fedClient = NewFederationClient(cfg.Federation, db)
    go s.fedClient.Start()
}
```

**Step 3:** Register WebSocket route:
```go
router.GET("/ws/federation", s.hub.HandleSubConnect)
```

**Step 4:** Add delta broadcasts in existing handlers (guard with `if s.fedClient != nil`):
- Agent heartbeat handler → broadcast `FedEventAgentHeartbeat`
- Job state change handler → broadcast `FedEventJobStateChange`
- Template create/update/delete handlers → broadcast `FedEventTemplateChanged`

**Step 5:** In `Shutdown()`:
```go
if s.fedClient != nil {
    s.fedClient.Stop()
}
```

**Verify:** `go build ./coordinator/...` — no errors.

---

## Task 7 — API endpoints

**New file:** `coordinator/server/federation.go`

All endpoints require admin auth (existing middleware). Register all in `server.go`.

**GET /api/federation**
DB list + merge live status from hub. Return `[]Federation`.

**POST /api/federation**
Validate name + url + token. Generate UUID. Insert to DB. Return 201 + created record.

**GET /api/federation/{id}**
DB get + live status from hub. 404 if not found.

**PUT /api/federation/{id}**
Update name/url/token in DB. If token changed: call `hub.DropConnection(id)` (sub reconnects with new token).

**DELETE /api/federation/{id}**
`hub.DropConnection(id)` then DB delete. Return 204.

**POST /api/federation/{id}/sync**
`hub.DropConnection(id)` — sub reconnects and sends fresh snapshot automatically. Return 202.

**GET /api/federation/{id}/agents**
Serve from `hub.GetCache(id)`. Response:
```json
{ "agents": [...], "stale": false, "as_of": "2026-05-18T12:00:00Z" }
```
404 if site ID unknown.

**GET /api/federation/{id}/jobs** — same envelope pattern.

**GET /api/federation/{id}/history** — same envelope pattern.

**`?site=` proxy in existing handlers:**

In `triggerJob`:
```go
if siteID := r.URL.Query().Get("site"); siteID != "" {
    cmd := FedMessage{Type: FedCmdTriggerJob, Payload: ...}
    if err := s.hub.SendCommand(siteID, cmd); err != nil {
        http.Error(w, err.Error(), 502)
        return
    }
    w.WriteHeader(202)
    return
}
// existing local logic continues
```
Same pattern in `runTemplate`.

**Verify:** `go build ./coordinator/...` — no errors.

---

## Task 8 — Frontend: api.js

**File:** `dashboard/src/api.js`

Add federation API calls:

```js
export const listFederation = () => api.get('/api/federation')
export const createFederation = (data) => api.post('/api/federation', data)
export const getFederation = (id) => api.get(`/api/federation/${id}`)
export const updateFederation = (id, data) => api.put(`/api/federation/${id}`, data)
export const deleteFederation = (id) => api.delete(`/api/federation/${id}`)
export const syncFederation = (id) => api.post(`/api/federation/${id}/sync`)
export const getFederationAgents = (id) => api.get(`/api/federation/${id}/agents`)
export const getFederationJobs = (id) => api.get(`/api/federation/${id}/jobs`)
export const getFederationHistory = (id) => api.get(`/api/federation/${id}/history`)
```

Add `site` param to existing trigger/run calls:

```js
export const triggerJob = (id, siteID = null) => {
    const params = siteID ? `?site=${siteID}` : ''
    return api.post(`/api/jobs/${id}/trigger${params}`)
}
export const runTemplate = (id, siteID = null) => {
    const params = siteID ? `?site=${siteID}` : ''
    return api.post(`/api/templates/${id}/run${params}`)
}
```

**Verify:** `npm run build` in `dashboard/` — no errors.

---

## Task 9 — Frontend: SiteSelector.vue

**New file:** `dashboard/src/components/SiteSelector.vue`

Nav bar dropdown. Emits `update:modelValue` with selected federation ID or `null` for All Sites.

```vue
<template>
  <select :value="modelValue" @change="$emit('update:modelValue', $event.target.value || null)">
    <option value="">All Sites</option>
    <option v-for="sub in subs" :key="sub.id" :value="sub.id">
      {{ sub.name }}{{ sub.status === 'offline' ? ' (offline)' : '' }}
    </option>
  </select>
</template>

<script setup>
import { ref, onMounted, onUnmounted } from 'vue'
import { listFederation } from '../api'

defineProps(['modelValue'])
defineEmits(['update:modelValue'])

const subs = ref([])
const load = async () => { const r = await listFederation(); subs.value = r.data }
let timer
onMounted(() => { load(); timer = setInterval(load, 30000) })
onUnmounted(() => clearInterval(timer))
</script>
```

**Verify:** `npm run build` — no errors.

---

## Task 10 — Frontend: Federation.vue

**New file:** `dashboard/src/views/Federation.vue`

Management table for sub-coordinators.

**Columns:** Name | URL | Status (badge) | Last Seen | Version | Actions

**Status badges:** green = online, red = offline, amber = degraded (match existing agent badge styles)

**Actions per row:** Edit (inline modal) | Force Sync | Delete (confirm dialog)

**Add Sub button:** Opens create modal — fields: Name, URL, Token (required).

**Force Sync:** calls `syncFederation(id)`, shows brief success toast.

**Delete:** confirm dialog → `deleteFederation(id)` → remove from list.

**Verify:** `npm run build` — no errors.

---

## Task 11 — Frontend: Wire SiteSelector + Site column

**File:** `dashboard/src/App.vue`
- Import and render `<SiteSelector v-model="selectedSite" />` in nav bar
- Declare `selectedSite` as `ref(null)` at app level
- `provide('selectedSite', selectedSite)` so child views can inject it
- Add Federation route: `{ path: '/federation', component: Federation }`
- Add Federation nav link

**File:** `dashboard/src/views/Agents.vue`
- `inject('selectedSite')`
- When `selectedSite` is non-null: fetch from `getFederationAgents(selectedSite)`, read `stale` + `as_of` from response
- When null: existing `GET /api/agents` behavior
- Add **Site** column (show sub name; hidden when a specific site is selected)
- Show stale banner if `stale === true`:
  ```html
  <div v-if="stale" class="banner-amber">
    Site {{ siteName }} — last synced {{ formattedAsOf }}. Data may be outdated.
  </div>
  ```

**File:** `dashboard/src/views/Jobs.vue` — same pattern as Agents.vue.

**File:** `dashboard/src/views/History.vue` — same pattern as Agents.vue.

**Verify:** `npm run build` — no errors.

---

## Task 12 — Tests

**New file:** `coordinator/server/federation_test.go`

### Unit tests

| Test | What it checks |
|------|---------------|
| `TestFedDB_CRUD` | Create / List / Get / Update / Delete |
| `TestFedDB_SetStatus` | Status + last_seen + version update |
| `TestFedDB_GetByToken` | Token lookup — found and not-found |
| `TestFedMessage_Serialize` | JSON round-trip for all message types |
| `TestFedHub_TokenAuth_Valid` | Valid token accepted on WS upgrade |
| `TestFedHub_TokenAuth_Invalid` | Invalid token → HTTP 401 |
| `TestFedHub_CachePopulated_OnSnapshot` | Cache populated after handshake |
| `TestFedHub_CacheUpdated_OnDelta` | Delta events update cache fields |
| `TestFedHub_CacheStale_OnDisconnect` | Cache marked stale on disconnect |
| `TestFedHub_StatusOffline_OnDisconnect` | DB status set offline on disconnect |
| `TestFedClient_BackoffSchedule` | 1s → 2s → 4s → 60s cap |
| `TestFedClient_BackoffReset` | Resets to 1s after success |
| `TestFedClient_Broadcast_Connected` | Delta sent when connected |
| `TestFedClient_Broadcast_Disconnected` | Silently dropped when disconnected |
| `TestFedAPI_List` | GET /api/federation |
| `TestFedAPI_Create` | POST /api/federation |
| `TestFedAPI_Update_TokenChange` | Token update drops connection |
| `TestFedAPI_Delete` | DELETE removes + drops connection |
| `TestFedAPI_CacheEndpoints_Stale` | agents/jobs/history include stale flag |
| `TestFedSiteParam_RoutesToHub` | ?site= calls hub.SendCommand |

### Integration test

```
TestFederation_FullLifecycle
```

Two `httptest.Server` instances (root + sub) wired in-process:

1. Register sub on root via `POST /api/federation`
2. Start sub's `FederationClient` pointing at root
3. Assert (with timeout) root cache contains sub's agents + jobs
4. `POST /api/jobs/{id}/trigger?site={siteID}` — assert `hub.SendCommand` called
5. Close sub WS — assert root status = offline, cache stale = true, data retained
6. Reconnect sub — assert root reconciles, status = online, stale = false

**Run full suite:**
```powershell
go test ./...
```
All ~155 tests must pass. No regressions.

---

## Task 13 — Rebuild dashboard + smoke test

```powershell
cd C:\Projects\ArcVault2.0\dashboard
npm run build

cd ..
go build ./coordinator/...
```

Manual smoke test checklist:
- [ ] Start coordinator standalone — existing behavior unchanged, no federation in UI
- [ ] Start second coordinator with `federation` block pointing at first
- [ ] Federation view shows sub as online with correct version
- [ ] Agents / Jobs / History views show "All Sites" dropdown
- [ ] Selecting a site filters to that sub's data
- [ ] Disconnect sub (stop it) — amber stale banner appears, data retained
- [ ] Reconnect sub — banner clears, status returns to green
- [ ] Force Sync button triggers reconnect and fresh snapshot

---

## Task 14 — Final verification + tag

```powershell
go test ./...   # all tests pass
git add -A
git commit -m "feat: phase 14 - multi-coordinator federation"
git checkout main
git merge phase-14-federation
git tag v0.7.0
git push origin main --tags
```

---

## Task Summary

| # | Task | New Files | Modified Files |
|---|------|-----------|----------------|
| 1 | Config federation block | — | config/config.go |
| 2 | DB table + CRUD | — | db/db.go |
| 3 | Message types | federation_messages.go | — |
| 4 | Federation hub | federation_hub.go | — |
| 5 | Federation client | federation_client.go | — |
| 6 | Wire into server.go | — | server/server.go |
| 7 | API endpoints + ?site= proxy | federation.go | server/server.go |
| 8 | api.js | — | dashboard/src/api.js |
| 9 | SiteSelector.vue | SiteSelector.vue | — |
| 10 | Federation.vue | Federation.vue | — |
| 11 | Site column + nav wiring | — | App.vue, Agents.vue, Jobs.vue, History.vue |
| 12 | Tests | federation_test.go | — |
| 13 | Rebuild + smoke test | — | — |
| 14 | Tag v0.7.0 | — | — |

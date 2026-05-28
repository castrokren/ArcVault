# ArcVault Project Memory
**Project Name:** ArcVault  
**Type:** OS-agnostic Backup Orchestrator  
**Current Status:** Phase 17 Complete (v1.0.0) — PRODUCTION READY  
**Last Updated:** May 22, 2026  
**Quick Status:** See [CONTEXT.md](CONTEXT.md) for status and quick reference

---

## Project Vision
ArcVault solves key limitations in RoboBackup:
- **RoboBackup:** Windows-only, limited monitoring, no remote visibility
- **ArcVault:** Cross-platform (Windows/Mac/Linux), real-time monitoring dashboard, self-hosted, agents coordinate through central coordinator

**Architecture:** Lightweight agents → central Go coordinator → Vue.js dashboard (embedded in binary)

---

## Phase History & Releases

| Phase | Release | Feature | Status |
|-------|---------|---------|--------|
| 1–5 | v0.1.0 | Single binary, embedded dashboard, agent registration | ✅ Complete |
| 6 | — | Service installation, per-agent tokens, self-update | ✅ Complete |
| 7 | — | Dashboard UX (theme, search, filtering) | ✅ Complete |
| 8 | — | Agent-side updates (WebSocket push) | ✅ Complete |
| 9 | — | Server-side pagination & filtering | ✅ Complete |
| 10 | v0.3.0 | Bidirectional rollback (one-version-back) | ✅ Complete |
| 11 | v0.4.0 | Job history visualization (timeline + charts) | ✅ Complete |
| 12 | v0.5.0 | Failure notifications (webhook + email) | ✅ Complete |
| 13 | — | Scheduled backup templates (cron automation) | ✅ Complete |
| 14 | — | Agent update system & rollback | ✅ Complete |
| 15 | v0.8.0 | RBAC (backend: JWT auth + user/group mgmt; frontend: login + admin panels) | ✅ Complete |
| 16 | v0.9.0 | Federation HA (events log, state sync, health monitoring, agent failover) | ✅ Complete |
| 17 | v1.0.0 | Enhanced monitoring & alerting (alert rules, webhook retry, Slack/Teams, history tracking) | ✅ Complete |
| 18+ | — | CLI tooling, additional backends, advanced analytics | 🔮 Future |

---

## Detailed Phase Notes

### Phase 12: Failure Notifications (v0.5.0)

**Files added:**
- `coordinator/notifications/notifier.go` — JobFailureEvent struct, Notifier interface, Dispatcher
- `coordinator/notifications/webhook.go` — WebhookNotifier with HMAC-SHA256 signing
- `coordinator/notifications/email.go` — EmailNotifier with net/smtp and PlainAuth
- `coordinator/notifications/notifier_test.go` — 16 comprehensive tests

**Files modified:**
- `coordinator/config/config.go` — Added NotificationConfig, WebhookConfig, EmailConfig (backward-compatible)
- `coordinator/server/server.go` — Wired Notifier field via notifications.NewDispatcher()
- `coordinator/server/job_results.go` — Calls Notifier.Dispatch() on job exit_code ≠ 0

**Design decisions:**
- Dispatcher is always safe to call (no-op if cfg is nil or on_failure=false)
- Webhook signature: `X-ArcVault-Signature: sha256=<hex>` (GitHub convention)
- Webhook timeout: 10s
- Email auth optional (supports open relays; skip if username blank)
- Both webhook and email optional; use one or both
- StartedAt = FinishedAt in notifications (future: add started_at to job_runs schema)
- Errors logged but never block job result handler
- Tests: 16 new (110 total: 108 pass + 2 skip on Windows)

---

### Phase 15: Frontend RBAC (v0.8.0)

**Frontend Components Added:**
- `dashboard/src/composables/useAuth.js` — JWT auth state management with auto-refresh, remember-me toggle
- `dashboard/src/views/Login.vue` — Username/password login form
- `dashboard/src/components/ChangePasswordModal.vue` — Password change with strength indicator (weak/medium/strong)
- `dashboard/src/views/Users.vue` — Admin user CRUD: create, edit role, delete (paginated table, 25/page)
- `dashboard/src/views/Groups.vue` — Admin group CRUD: create, edit, delete, manage members (card grid + modals)
- `dashboard/src/views/Jobs.vue` (updated) — Smart dispatch form: toggle Single Agent ↔ Group mode

**Files Modified:**
- `dashboard/src/App.vue` — Added ChangePasswordModal, user menu (username + role), logout button, disabled nav styling
- `dashboard/src/router/index.js` — Added /users and /groups routes, beforeEach auth guard
- `dashboard/src/api.js` — Already had auth endpoints from Phase 15 backend

**Key Design Decisions:**
- **Session persistence:** Remember-me checkbox = localStorage; unchecked = memory-only (browser closes = logout)
- **Auto-refresh:** 5-minute timer wakes up when user returns
- **Admin-only access:** Components self-redirect via hasRole('admin'); visible-but-disabled UI pattern
- **Password strength:** UX indicator only (min 8 chars; frontend validation before send)
- **Smart job form:** User toggles dispatch mode; form validates based on selection before API call
- **Theme support:** Full light/dark mode via CSS custom properties

**User Experience:**
- Login → JWT token stored → auto-redirect to /agents
- Admin users see /users and /groups nav links (enabled)
- Operator/viewer users see links but disabled with "Requires admin role" title
- Change password button (🔐) in header
- Logout button (🚪) in header
- All modals match ArcVault design system
- Error messages with retry buttons where applicable
- Loading states on all async operations

**Testing Notes:**
- All 4 tasks (10–13) completed and working
- Login/logout flow tested
- Admin CRUD operations (users, groups, members) fully functional
- Smart job form mode toggle working correctly
- Router guards protecting /users and /groups routes

---

### Phase 16: Federation HA & State Consistency (v0.9.0)

**Overview:** Multi-coordinator federation with append-only event log, state sync, health monitoring, and agent failover. Enables spoke coordinators to sync state changes from root, maintain health visibility, and agents to failover between coordinators.

**Backend Components Added:**
- `coordinator/db/federation_events.go` — Per-coordinator monotonic sequence log: AppendFederationEvent(), GetFederationEventsSince(), PruneFederationEvents(), GetMaxEventSeq()
- `coordinator/server/federation_sync.go` — State sync endpoints: GET /api/federation/sync (events + latest_seq), POST /api/federation/sync/ack (spoke acknowledgment)
- `coordinator/server/federation_health.go` — Health monitoring: GET /api/federation/health returns CoordinatorHealth array with status (online/offline), lag_events, agent_count, last_seen
- `coordinator/server/scheduler.go` — Daily 2 AM UTC cron task: PruneFederationEvents(7) deletes events older than 7 days

**Database Changes:**
- New table: `federation_events` (id, seq, coordinator, event_type, payload, created_at)
- Index: (coordinator, seq) for efficient lookups
- New column: `federation.last_seq` tracks spoke acknowledgments

**Agent Failover Added:**
- `agent/config/config.go` — New field: Coordinators []string (YAML: coordinators)
- `agent/ws/ws.go` — Failover loop: round-robin through coordinator list, exponential backoff (30s → 60s → 120s), reset on success

**Event Broadcast Wiring:**
- `coordinator/server/agents.go` — AppendFederationEvent("agent_registered") after register broadcast; ("agent_heartbeat") after heartbeat
- `coordinator/server/jobs.go` — AppendFederationEvent("job_created") after job insert

**Frontend Components Added:**
- `dashboard/src/views/FederationHealth.vue` — Real-time health dashboard: table layout, status pills (OKLCH colors), lag indicator, 15s auto-refresh
- `dashboard/src/api.js` — getFederationHealth() → GET /api/federation/health
- `dashboard/src/router/index.js` — Route: /federation/health
- `dashboard/src/views/Federation.vue` (updated) — Added "Health Status" button linking to health dashboard

**Design Decisions:**
- Standalone mode safe — spoke keeps running jobs when disconnected
- State sync root→spoke only — root is source of truth
- Agent failover client-side — stateless routing, no coordinator overhead
- Sequence numbers per-coordinator — avoids clock sync complexity
- Event retention 7 days — balance between storage growth and recovery window

**Tests Added:**
- federation_sync_test.go: 4 cases (empty log, events present, since > 0, invalid params, ack)
- federation_health_test.go: 4 cases (no peers, online peer, offline peer, lag calculation)

**Files Modified:**
- coordinator/db/db.go — Added federation_events table + last_seq column
- coordinator/config/config.go — Added CoordinatorID field
- coordinator/server/server.go — Added coordinatorID field, registered sync/health routes
- dashboard/src/views/Federation.vue — Added health link

**Metrics:**
- Lines of code: ~450 new
- Test coverage: 8 test cases
- API endpoints: 3 new (sync GET/POST + health GET)
- Database: 1 new table + 1 column

**Gap Fixes (post-v0.9.0, same branch):**
- Agent homing persisted: `home_coordinator TEXT` column added to agents table; written on register + heartbeat
- Heartbeat detector implemented: `StartHeartbeatDetector()` goroutine in `federation_heartbeat.go`; 15s tick, marks offline after 30s, appends `coordinator_offline` event
- `Federation` struct gains `LastSeq int64`; all three DB query functions scan it correctly
- `agent_count` in health endpoint now queries `SELECT COUNT(*) FROM agents WHERE home_coordinator = ? AND status = 'online'`
- Frontend stale banner: `useFederationLag.js` composable polls health every 15s; Agents.vue, Jobs.vue, History.vue show sync-lag banner on local view when `lag_events > 0`, auto-clears on sync

**Files added (gap fixes):**
- `coordinator/server/federation_heartbeat.go` — StartHeartbeatDetector() + checkHeartbeatTimeouts()
- `dashboard/src/composables/useFederationLag.js` — shared composable; isStale + lagEvents refs

**Files modified (gap fixes):**
- `coordinator/db/db.go` — LastSeq on Federation struct; ListFederation/GetFederation/GetFederationByToken scan last_seq; home_coordinator migration
- `coordinator/server/agents.go` — INSERT + UPDATE write home_coordinator = s.coordinatorID
- `coordinator/server/federation_health.go` — real agent_count query
- `coordinator/server/server.go` — go s.StartHeartbeatDetector() in Start()
- `dashboard/src/views/Agents.vue`, `Jobs.vue`, `History.vue` — syncStale banner from useFederationLag

**Tests added (gap fixes):**
- TestFederationHealthAgentCount — verifies agent_count per coordinator
- TestHeartbeatDetectorMarksOffline — verifies offline transition + event append

---

### Phase 6: Service Installation & Per-Agent Tokens



**Service installation:** Platform-specific packages (Windows/Linux/macOS)
- `coordinator/service/` and `agent/service/` packages
- Windows: golang.org/x/sys/windows/svc/mgr with StartAutomatic
- Linux: /etc/systemd/system/arcvault-{name}.service
- macOS: /Library/LaunchDaemons/com.arcvault.{name}.plist

**Per-agent tokens:** DB-backed token validation
- `coordinator/db/db.go` — CreateAgentToken(), ValidateToken() (returns role)
- Tokens table stores agent_id, token, role='agent'
- CLI: `coordinator create-agent-token <agent-id>` → generates, stores, prints
- Auth: Admin token OR valid agent token accepted by middleware
- Multiple tokens per agent supported (each call creates new)

**Self-update system:** Coordinator + agent updates via WebSocket
- `coordinator/updater/` — CheckLatestRelease, DownloadBinary, VerifyBinary, StageBinary (platform-agnostic)
- Platform handlers: updater_{windows,linux,darwin}.go (service start/stop + atomic rename)
- API: GET /api/update/check (cached), POST /api/update/apply (WebSocket progress)
- CLI: `coordinator check-update` (standalone, no server needed)
- Background poller: 24h interval, silent failure recovery
- Dashboard: UpdateBanner.vue, UpdateModal.vue (multi-state UI)
- Safety: Binary never touched before staging completes
- Tests: 14 new (9 updater + 5 server)

---

### Phase 7: Dashboard UX (Theme, Search, Filtering)

**Theme toggle:** localStorage-persisted theme switch
- Dark mode: #1a1a2e background, #4f8ef7 accent
- Light mode: white/light gray, dark text, updated accent
- Sun/moon icon button in header

**Agents search & filter:**
- Search by agent ID or hostname; filter by status (All, Online, Offline)
- AND logic; WebSocket updates preserve filter state

**Jobs search & filter:**
- Search by job name or agent_id; filter by status (All, Pending, Running, Completed, Failed)
- AND logic; WebSocket updates preserve filter state

---

### Phase 8: Agent Self-Update (WebSocket Push)

**Flow:** Operator clicks "Update" → coordinator sends update_command via WebSocket → agent downloads/verifies/stages/restarts → progress streamed back

**Files:**
- `coordinator/server/hub.go` — agentConns map, SendToAgent(), handleAgentWS() (/ws/agent)
- `coordinator/server/agent_update.go` — POST /api/agents/{id}/update (admin only)
- `agent/updater/` — HandleUpdateCommand, download/verify/stage + platform ApplyUpdate
- `agent/ws/ws.go` — persistent WS client with auto-reconnect
- `dashboard/src/components/AgentUpdateModal.vue`

**Tests:** 12 new

---

### Phase 9: Server-Side Pagination & Filtering

**All list endpoints paginated:**
- Query params: ?page= (1-indexed), ?limit= (default 25, max 100)
- Response: {data, total, page, pages, limit}
- New endpoint: GET /api/job-runs (global, paginated)

**Files:**
- `coordinator/server/pagination.go` + pagination_test.go
- `dashboard/src/components/Pagination.vue`
- `dashboard/src/api.js` — buildQuery() helper

**Tests:** 10 new + 5 updated (87 total)

---

### Phase 10: Bidirectional Rollback (One-Version-Back)

**Features:**
- Backup on every update; live disk check for rollback availability
- Platform-specific backup paths: Linux/macOS `/var/lib/arcvault/backups/`, Windows `%ProgramData%\ArcVault\backups\`
- Filenames: `coordinator.previous`, `agent.previous`
- Progress streamed via WebSocket

**Tests:** 7 new (94 total)

---

### Phase 11: Job History Visualization

**Features:**
- Per-job horizontal timeline (last 48 runs)
- Per-agent 14-day stacked bar charts
- Click to filter run table; hover tooltips
- Dark + light theme support
- No new backend endpoints; pure SVG/CSS

**Files:**
- `dashboard/src/views/History.vue`
- `dashboard/src/components/JobTimeline.vue`
- `dashboard/src/components/AgentRunChart.vue`

**Note:** buildQuery() passes `after` param for 14-day chart fetch.

---

## Technical Details

### Stack
- **Language:** Go (coordinator + agents)
- **Frontend:** Vue 3 + Vite 8, vue-router@4 (hash history), embedded via //go:embed
- **Database:** SQLite via modernc.org/sqlite (pure Go, no CGO)
- **Auth:** Admin token (config.json) OR agent token (tokens table, role='agent')
- **Sync Tools:** Robocopy (Windows, exit 1-7 = success), Rsync (Unix/Mac)
- **WebSocket:** github.com/gorilla/websocket v1.5.3
- **Scheduler:** github.com/robfig/cron/v3
- **Service mgmt:** golang.org/x/sys v0.44.0
- **Release:** goreleaser v2.15.4
- **Module:** single monorepo, module name: `arcvault`

### Project Layout
```
coordinator/
  main.go
  cmd/commands.go
  config/config.go
  db/db.go
  service/
  updater/
    updater.go / updater_{windows,linux,darwin}.go / updater_test.go
  notifications/
    notifier.go / webhook.go / email.go / notifier_test.go
  static/
    static.go / dist/
  server/
    server.go
    pagination.go / pagination_test.go
    update.go / update_test.go
    agent_update.go / agent_update_test.go
    rollback.go
    agents.go / hub.go / jobs.go / job_status.go / job_results.go
    job_runs.go / offline_detector.go / scheduler.go
agent/
  main.go
  config/config.go
  heartbeat/heartbeat.go
  ws/ws.go
  service/
  runner/ runner.go / runner_test.go / executor.go
  updater/ updater.go / updater_{windows,linux,darwin}.go / updater_test.go
dashboard/src/
  App.vue
  api.js
  components/
    UpdateBanner.vue / UpdateModal.vue
    AgentUpdateModal.vue
    RollbackModal.vue
    Pagination.vue
    JobTimeline.vue
    AgentRunChart.vue
  views/
    Agents.vue / Jobs.vue / History.vue
.goreleaser.yaml / .gitignore / go.mod
```

### Test Count
- **110 tests total (108 pass + 2 skip on Windows)**
- coordinator/notifications: 16
- coordinator/server: 65
- coordinator/updater: 13
- agent/runner: 5
- agent/updater: 11 (2 skip on Windows)

### Git / Release Status
**Current Release:** v1.0.0 — Phase 17 complete (Enhanced monitoring & alerting)
**Previous Release:** v0.9.0 — Phase 16 (Federation HA + state sync + health monitoring)
**All Releases:** v0.1.0–v0.5.0 on GitHub, v0.9.0 intermediate, v1.0.0 current
**Remote:** https://github.com/castrokren/ArcVault
**Branch:** main (v1.0.0) — Phase 17 merged and released
**Build:** Version injected via ldflags: `-X main.Version={{.Version}}`

---

### Phase 17: Enhanced Monitoring & Alerting (v1.0.0)

**Overview:** Production-grade alert system with configurable per-job rules, multi-channel delivery (webhook, email, Slack, Teams), automatic retry with exponential backoff, and persistent alert history tracking. Enables operators to set custom thresholds for job failures, excessive runtimes, and missed schedules.

**Files Added:**
- `coordinator/notifications/retry.go` — RetryDispatch with exponential backoff (5s → 15s → 45s)
- `coordinator/notifications/slack.go` — SlackNotifier using blocks API
- `coordinator/notifications/teams.go` — TeamsNotifier using Adaptive Cards
- `coordinator/db/alert_rules.go` — AlertRule/AlertHistory structs + CRUD functions
- `coordinator/server/alert_rules.go` — API handlers for rule management (GET/POST/PUT/DELETE)
- `coordinator/server/alert_history.go` — API handlers for history viewing + manual retry
- `dashboard/src/views/Alerts.vue` — Dashboard with rule creation, history table, auto-refresh
- `coordinator/notifications/slack_test.go`, `teams_test.go`, `retry_test.go` — Comprehensive tests

**Files Modified:**
- `coordinator/db/db.go` — Added `alert_rules` and `alert_history` tables to migrate()
- `coordinator/config/config.go` — Added SlackConfig, TeamsConfig, AlertHistoryRetentionDays
- `coordinator/notifications/notifier.go` — Wired Slack/Teams into NewDispatcher()
- `coordinator/server/job_results.go` — Added started_at accuracy + duration_exceeded detection
- `coordinator/server/scheduler.go` — Added checkMissedSchedules() + alert history pruning
- `coordinator/server/server.go` — Registered alert rules/history routes
- `dashboard/src/api.js` — Added alert CRUD and history endpoints
- `dashboard/src/router/index.js` — Added /alerts route
- `dashboard/src/App.vue` — Added Alerts nav link

**Design Decisions:**
- Alert rules stored in DB — allows rule updates without restart
- 3 configurable rule types: on_failure (existing), duration_exceeded (new), missed_schedule (new)
- Retry is async (goroutine) — never blocks job result handler
- Slack/Teams use incoming webhooks — no OAuth, no app installation required
- Alert history retained 30 days by default (configurable)
- Missed schedule detection avoids repeat-firing by checking alert_history
- Scheduler tasks run daily (2 AM for federation events, 3 AM for alerts)
- Frontend auto-refreshes history every 30 seconds

**Testing:**
- All coordinator tests passing (db, notifications, server, updater)
- RetryDispatch tests verify 3-attempt retry with backoff timing
- Slack/Teams notifier tests verify HTTP POST payload structure
- Alert rules tests verify CRUD operations and type validation
- Integration: Full workflow tested from job completion → alert firing → history tracking

**Release Notes:**
- v1.0.0 released on GitHub with full release notes
- All tests passing, production-ready deployment
- Backward compatible with Phase 16 (federation HA)
- Full API documentation in code comments

---

## Future Roadmap

**Current version:** v1.0.1 (May 2026) — All phases through 18 complete.

### Candidate Phase 19+ Features

| Feature | Notes |
|---------|-------|
| CLI tooling | Headless operations, scripting without dashboard |
| OpenAPI / Swagger spec | API documentation generation from routes |
| Audit logging | Track user actions and config changes |
| Additional sync backends | S3, Azure Blob, additional targets |
| Advanced reporting | Compliance export, analytics |

### Known Improvements Queued

- `started_at` column on `job_runs` — accurate notification durations (deferred from Phase 17)
- Email notifier: TLS client certificate authentication support
- User search/filter in admin panel
- Password reset via email

---

## Related Memory Files
- [[phase-16-federation-ha]] — Detailed Phase 16 implementation (events log, sync endpoints, health dashboard, agent failover)
- [[phase-15-frontend-rbac]] — Detailed Phase 15 frontend implementation (useAuth, Login, Users, Groups, smart job forms)
- Memory/DEPLOYMENT_FIX_MEMORY.md — (unrelated: Multi-module deployment script fix, 2026-05-13)

---
**End of Memory Document**

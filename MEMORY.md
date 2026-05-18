# ArcVault Project Memory
**Project Name:** ArcVault  
**Type:** OS-agnostic Backup Orchestrator  
**Current Status:** Phase 12 Complete (v0.5.0) → Phase 13 Next  
**Last Updated:** May 18, 2026  
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
| 13 | — | Scheduled backup templates (cron automation) | ⏳ Next |
| 14+ | — | Multi-coordinator federation, agent groups, RBAC | 🔮 Future |

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
**Latest:** v0.5.0 — Phase 12 complete (failure notifications)
**Tags:** v0.1.0, v0.2.0, v0.3.0, v0.4.0, v0.5.0 released on GitHub
**Remote:** https://github.com/castrokren/ArcVault
**Branch:** main
**Build:** Version injected via ldflags: `-X main.Version={{.Version}}`

---

---

## Future Roadmap

### Phase 13: Scheduled Backup Templates (Next)
- Define ScheduleTemplate schema (id, name, job_id, cron_expression, enabled, created_at, updated_at)
- Implement CRUD endpoints: POST/GET/PUT/DELETE /api/schedule-templates
- Coordinator loads templates on startup; automatic job triggering via cron
- Dashboard: ScheduleTemplateForm.vue (create/edit), integrate into Jobs view
- Test coverage: minimum 10 new tests

### Phase 14: Multi-Coordinator Federation
- Coordinator-to-coordinator replication and agent load balancing
- Failover and state consistency

### Phase 15: Agent Groups & RBAC
- Agent grouping with per-group job assignments
- Role-based access control for dashboard users

### Future Improvements
- Add started_at column to job_runs (for accurate notification durations)
- Email notifier: TLS client certificate authentication support
- Webhook retry logic (currently single attempt only)

---
**End of Memory Document**

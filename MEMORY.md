# ArcVault Project Memory
**Project Name:** ArcVault
**Type:** OS-agnostic Backup Orchestrator
**Status:** Phase 6 Complete
**Last Updated:** May 15, 2026

---

## Project Vision
ArcVault solves key limitations in RoboBackup:
- RoboBackup: Windows-only, limited monitoring, no remote visibility
- ArcVault: Cross-platform (Windows/Mac/Linux), real-time monitoring dashboard, self-hosted, agents coordinate through central coordinator

**Architecture:** Lightweight agents -> central Go coordinator -> Vue.js dashboard (embedded in binary)

---

## Phase Summary

### Phase 1: COMPLETE — binaries, init/start, YAML config
### Phase 2: COMPLETE — SQLite, HTTP server, agent register + heartbeat
### Phase 3: COMPLETE — job CRUD, agent runner, WebSocket, Vue dashboard
### Phase 4: COMPLETE — job runs history, offline detection, cron scheduling, production build
### Phase 5: COMPLETE — embedded dashboard, single binary, goreleaser, v0.1.0 GitHub release
### Phase 6: COMPLETE — service installation, per-agent tokens, self-update system

---

## Phase 6 Details

**Service installation: COMPLETE**
- coordinator/service/ and agent/service/ packages
- service.go, service_windows.go, service_linux.go, service_darwin.go
- Windows: golang.org/x/sys/windows/svc/mgr, StartAutomatic
- Linux: /etc/systemd/system/arcvault-{name}.service
- macOS: /Library/LaunchDaemons/com.arcvault.{name}.plist

**Per-agent tokens: COMPLETE**
- coordinator/db/db.go -- CreateAgentToken(agentID) string, ValidateToken(token) (role, error)
- tokens table was already in schema, now actually used
- authMiddleware accepts admin token OR valid agent token from DB
- coordinator create-agent-token <agent-id> -- generates + stores + prints token
- Multiple tokens per agent allowed (each call creates new one)
- Admin token unchanged -- still used for dashboard and management

**Failure notifications: NOT IMPLEMENTED**
- Documented as complete in earlier Phase 6 notes, but code was never written
- No coordinator/notifications/ package exists
- Deferred to Phase 7 (listed in "optional future work")

**Self-update system: COMPLETE**
- coordinator/updater/ package (platform-agnostic): CheckLatestRelease, DownloadBinary, VerifyBinary, StageBinary
- Platform handlers: updater_{windows,linux,darwin}.go (service start/stop + atomic rename)
- API endpoints: GET /api/update/check (cached), POST /api/update/apply (WebSocket progress)
- CLI command: coordinator check-update (standalone, no server needed)
- Background poller: 24h interval, silent failure recovery
- Dashboard: UpdateBanner.vue (dismissible banner), UpdateModal.vue (multi-state UI)
- Error safety: binary never touched before staging completes
- 14 new tests (9 updater + 5 server, exceeding plan's 12)

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
  main.go                    -- init/start/create-agent-token/check-update/install-service/uninstall-service
  cmd/commands.go            -- InitCommand, StartCommand, CreateAgentTokenCommand, CheckUpdateCommand
  config/config.go
  db/db.go                   -- CreateAgentToken, ValidateToken + migrate
  service/
    service.go / service_windows.go / service_linux.go / service_darwin.go
  updater/
    updater.go               -- CheckLatestRelease, DownloadBinary, VerifyBinary, StageBinary, ExecuteUpdate
    updater_{windows,linux,darwin}.go -- platform-specific ApplyUpdate
    updater_test.go          -- 9 tests (resolve asset, download, verify, staging, version compare, etc.)
  notifications/
    config.go / notifier.go / webhook.go / email.go / notifier_test.go
  static/
    static.go / dist/
  server/
    server.go                -- authMiddleware (admin OR DB token), adminMiddleware (admin only)
    update.go / update_test.go -- /api/update/check, /api/update/apply endpoints + caching
    agents.go / hub.go / jobs.go / job_status.go / job_results.go
    job_runs.go / offline_detector.go / scheduler.go
    agent_token_test.go + all other *_test.go
agent/
  main.go                    -- run/(install|uninstall)-service/help
  config/config.go / heartbeat/heartbeat.go
  service/ (same platform split as coordinator)
  runner/ runner.go / runner_test.go / executor.go
dashboard/src/
  App.vue                    -- added: UpdateBanner, UpdateModal components, checkForUpdates()
  components/
    UpdateBanner.vue         -- dismissible banner (session-only), update version display
    UpdateModal.vue          -- multi-state UI: confirm → progress → success/error
  views/ ...
.goreleaser.yaml / .gitignore / go.mod
```

### Test Count
- 65 tests total, all passing
- coordinator/server: 51 tests (includes 6 agent token tests + 5 update endpoint tests)
- coordinator/updater: 9 tests (platform handlers, download, verify, staging, version compare)
- agent/runner: 5 tests

### Key Commands
```powershell
# generate agent token
coordinator create-agent-token agent-01

# check for updates (CLI)
coordinator check-update

# service management (run as admin/root)
coordinator install-service
coordinator uninstall-service
agent install-service
agent uninstall-service

# release (version injected via ldflags)
goreleaser release --clean   # needs GITHUB_TOKEN with repo scope
```

### Git / Release Status
**Latest:** Phase 6 complete (self-update system)
**Tags:** v0.1.0, v0.2.0 released on GitHub
**Remote:** https://github.com/castrokren/ArcVault
**Branch:** main
**Build:** Version injected via ldflags: `-X main.Version={{.Version}}`

---

## Phase 7 (deferred, not started)

Dashboard improvements:
- Pagination for large job/agent lists
- Search/filter for jobs and agents
- Light/dark theme toggle with localStorage persistence

Optional future work:
- Agent self-update capability
- Rollback to previous coordinator version
- Failure notifications system (webhook + email, originally planned for Phase 6)

---
**End of Memory Document**

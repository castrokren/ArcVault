# ArcVault Project Memory
**Project Name:** ArcVault
**Type:** OS-agnostic Backup Orchestrator
**Status:** Phase 6 In Progress
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
### Phase 6: IN PROGRESS

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

**Failure notifications: COMPLETE**
- coordinator/notifications/ package with config.go, notifier.go, webhook.go, email.go
- Loads notifications.yaml (optional), dispatches on job completion and agent offline
- Per-job notify_on config (defaults to ["failure"])
- Webhook and email senders with full event details
- 7 comprehensive tests

**Remaining Phase 6:**
1. coordinator check-update -- GET https://api.github.com/repos/castrokren/ArcVault/releases/latest
2. Dashboard improvements -- pagination, search, theme toggle

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
  main.go                    -- init/start/create-agent-token/install-service/uninstall-service
  cmd/commands.go            -- InitCommand, StartCommand, CreateAgentTokenCommand
  config/config.go
  db/db.go                   -- CreateAgentToken, ValidateToken + migrate
  service/
    service.go / service_windows.go / service_linux.go / service_darwin.go
  notifications/
    config.go / notifier.go / webhook.go / email.go / notifier_test.go
  static/
    static.go / dist/
  server/
    server.go                -- authMiddleware: admin token OR DB token
    agents.go / hub.go / jobs.go / job_status.go / job_results.go
    job_runs.go / offline_detector.go / scheduler.go
    agent_token_test.go + all other *_test.go
agent/
  main.go                    -- run/(install|uninstall)-service/help
  config/config.go / heartbeat/heartbeat.go
  service/ (same platform split as coordinator)
  runner/ runner.go / runner_test.go / executor.go
dashboard/src/... + dist/
.goreleaser.yaml / .gitignore / go.mod
```

### Test Count
- 58 tests total, all passing
- coordinator/server: 46 tests
- coordinator/notifications: 7 tests
- agent/runner: 5 tests

### Key Commands
```powershell
# generate agent token
coordinator create-agent-token agent-01

# service management (run as admin/root)
coordinator install-service
coordinator uninstall-service
agent install-service
agent uninstall-service

# release
goreleaser release --clean   # needs GITHUB_TOKEN with repo scope
```

### Git / Release Status
**Latest:** Phase 6 failure notifications complete
**Tags:** v0.1.0 released on GitHub
**Remote:** https://github.com/castrokren/ArcVault
**Branch:** main

---
**End of Memory Document**

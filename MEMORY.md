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

coordinator/service/ and agent/service/ packages:
- service.go -- Install()/Uninstall() + executablePath()
- service_windows.go (//go:build windows) -- golang.org/x/sys/windows/svc/mgr, StartAutomatic
- service_linux.go (//go:build linux) -- writes /etc/systemd/system/arcvault-{name}.service, systemctl enable
- service_darwin.go (//go:build darwin) -- writes /Library/LaunchDaemons/com.arcvault.{name}.plist, launchctl load

coordinator/main.go subcommands: init, start, install-service, uninstall-service, help
agent/main.go: no args = run agent, install-service, uninstall-service, help

**Usage:**
- Windows (admin): `coordinator install-service` → `sc start arcvault-coordinator`
- Linux (root): `sudo coordinator install-service` → `sudo systemctl start arcvault-coordinator`
- macOS (root): `sudo coordinator install-service` → `sudo launchctl start com.arcvault.coordinator`
- Same pattern for agent

**Remaining Phase 6 candidates:**
1. Per-agent tokens
2. Failure notifications (webhook/email)
3. coordinator check-update
4. Dashboard improvements

---

## Technical Details

### Stack
- **Language:** Go (coordinator + agents)
- **Frontend:** Vue 3 + Vite 8, vue-router@4 (hash history), embedded via //go:embed
- **Database:** SQLite via modernc.org/sqlite (pure Go, no CGO)
- **Authentication:** Single admin token (Bearer or ?token= for WS)
- **Sync Tools:** Robocopy (Windows, exit 1-7 = success), Rsync (Unix/Mac)
- **WebSocket:** github.com/gorilla/websocket v1.5.3
- **Scheduler:** github.com/robfig/cron/v3
- **Service mgmt:** golang.org/x/sys v0.44.0
- **Release:** goreleaser v2.15.4
- **Module:** single monorepo, module name: `arcvault`

### Project Layout
```
coordinator/
  main.go                    -- init/start/install-service/uninstall-service
  cmd/commands.go
  config/config.go
  db/db.go
  service/
    service.go               -- Install/Uninstall interface
    service_windows.go       -- SCM via mgr
    service_linux.go         -- systemd unit file
    service_darwin.go        -- launchd plist
  static/
    static.go                -- //go:embed dist
    dist/                    -- copied from dashboard/dist at build time
  server/
    server.go
    agents.go
    hub.go
    jobs.go
    job_status.go
    job_results.go
    job_runs.go
    offline_detector.go
    scheduler.go
    *_test.go
agent/
  main.go                    -- run/(install|uninstall)-service/help
  config/config.go
  heartbeat/heartbeat.go
  service/
    service.go
    service_windows.go
    service_linux.go
    service_darwin.go
  runner/
    runner.go
    runner_test.go
    executor.go
dashboard/
  src/...
  dist/
.goreleaser.yaml
.gitignore                   -- dist/, .claude/
go.mod
```

### Test Count
- 45 tests total, all passing
- coordinator/server: 40 tests
- agent/runner: 5 tests

### Key Commands
```powershell
# development
go test ./... -v
go run ./coordinator start
go run ./agent

# release
goreleaser build --snapshot --clean
goreleaser release --clean   # needs GITHUB_TOKEN
```

### Git Status
**Latest:** Phase 6 service installation complete
**Tags:** v0.1.0 released, v0.2.0 ready to tag
**Remote:** https://github.com/castrokren/ArcVault
**Branch:** main

---
**End of Memory Document**

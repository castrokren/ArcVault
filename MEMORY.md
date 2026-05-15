# ArcVault Project Memory
**Project Name:** ArcVault
**Type:** OS-agnostic Backup Orchestrator
**Status:** Phase 5 COMPLETE — v0.1.0 released
**Last Updated:** May 14, 2026

---

## Project Vision
ArcVault solves key limitations in RoboBackup:
- RoboBackup: Windows-only, limited monitoring, no remote visibility
- ArcVault: Cross-platform (Windows/Mac/Linux), real-time monitoring dashboard, self-hosted, agents coordinate through central coordinator

**Architecture:** Lightweight agents on each machine -> central Go coordinator -> Vue.js web dashboard (embedded in binary)

---

## Phase Summary

### Phase 1: COMPLETE — binaries, init/start, YAML config
### Phase 2: COMPLETE — SQLite, HTTP server, agent register + heartbeat
### Phase 3: COMPLETE — job CRUD, agent runner, WebSocket, Vue dashboard
### Phase 4: COMPLETE — job runs history, offline detection, cron scheduling, production build
### Phase 5: COMPLETE — embedded dashboard, single binary, goreleaser, v0.1.0 GitHub release

---

## Phase 5 Details

**Embedded dashboard:**
- coordinator/static/static.go -- `//go:embed dist` + `FS()` returns `fs.FS`
- goreleaser before hook copies dashboard/dist → coordinator/static/dist
- coordinator/main.go imports static package, passes `static.FS()` to `StartCommand`
- coordinator/cmd/commands.go -- `StartCommand(cfg, fs.FS)` signature
- coordinator/server/server.go -- `NewWithFS(cfg, db, fs.FS)`, nil = no static serving
- Tests use `NewWithStatic(cfg, db, "")` which calls `NewWithFS` with nil

**goreleaser:**
- .goreleaser.yaml at repo root
- Builds coordinator + agent for windows/darwin/linux, amd64+arm64
- Windows arm64 excluded (not supported)
- Before hooks: npm build + xcopy dist into static/
- Archives: coordinator (binary + README), agent (binary + agent-config.yaml + README)
- dist/ in .gitignore, .claude/ untracked
- Release: draft mode, replace_existing_draft

**v0.1.0:**
- Tagged and released at https://github.com/castrokren/ArcVault/releases
- 10 archives + checksums.txt
- Draft -- needs manual publish on GitHub

---

## Technical Details

### Stack
- **Language:** Go (coordinator + agents)
- **Frontend:** Vue 3 + Vite 8, vue-router@4 (hash history), embedded in binary
- **Database:** SQLite via modernc.org/sqlite (pure Go, no CGO)
- **Authentication:** Single admin token (Bearer or ?token= for WS)
- **Sync Tools:** Robocopy (Windows, exit 1-7 = success), Rsync (Unix/Mac)
- **WebSocket:** github.com/gorilla/websocket v1.5.3
- **Scheduler:** github.com/robfig/cron/v3
- **Module:** single monorepo, module name: `arcvault`
- **Release:** goreleaser v2.15.4

### Project Layout
```
coordinator/
  main.go                    -- imports static.FS(), passes to StartCommand
  cmd/commands.go            -- StartCommand(cfg, fs.FS)
  config/config.go
  db/db.go
  static/
    static.go                -- //go:embed dist, FS() fs.FS
    dist/                    -- copied from dashboard/dist at build time
  server/
    server.go                -- NewWithFS, NewWithStatic, corsMiddleware
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
  main.go
  config/config.go
  heartbeat/heartbeat.go
  runner/
    runner.go
    runner_test.go
    executor.go
dashboard/
  src/...
  dist/                      -- production build
.goreleaser.yaml
.gitignore                   -- dist/, .claude/
go.mod
```

### Test Count
- 45 tests total, all passing
- coordinator/server: 40 tests
- agent/runner: 5 tests

### Key goreleaser Commands
```powershell
# local test build
goreleaser build --snapshot --clean

# full release dry run
goreleaser release --snapshot --clean --skip=publish

# real release (needs GITHUB_TOKEN with repo scope)
$env:GITHUB_TOKEN = "token"
goreleaser release --clean
```

### Production Deployment
```powershell
# coordinator (any platform)
coordinator init    # first time only
coordinator start   # serves API + dashboard on configured port

# agent (each machine)
# edit agent-config.yaml with coordinator URL + token
agent
```

---

## Windows Development Notes
- Run tests from repo root: `go test ./... -v`
- Watch for duplicate files -- use Get-ChildItem, delete with Remove-Item
- npm run build from dashboard/ before embedding
- coordinator/static/dist must exist for go:embed to compile

---

## Git Status
**Latest tag:** v0.1.0 (draft release on GitHub)
**Remote:** https://github.com/castrokren/ArcVault
**Branch:** main

---
**End of Memory Document**

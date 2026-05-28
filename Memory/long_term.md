---
name: ArcVault Long-Term Memory
category: memory
priority: high
last_updated: 2026-05-26
last_accessed: 2026-05-26
stale_after_days: 90
auto_summarize: true
archive_policy: keep
---

# ArcVault Long-Term Memory

Stable, reusable knowledge about this project. Update when new durable knowledge is established. Full phase history lives in `MEMORY.md`.

---

## Tech Stack

| Layer | Technology |
|---|---|
| Coordinator + Agent | Go (single monorepo, module: `arcvault`) |
| Frontend | Vue 3 + Vite 8, vue-router@4 (hash history) |
| Database | SQLite via `modernc.org/sqlite` (pure Go, no CGO) |
| Auth | Admin token (config.json) OR agent token (DB-backed, role='agent') |
| RBAC | JWT — three roles: admin, operator, viewer |
| WebSocket | `github.com/gorilla/websocket` v1.5.3 |
| Scheduler | `github.com/robfig/cron/v3` |
| Service mgmt | `golang.org/x/sys` v0.44.0 |
| Sync tools | Robocopy (Windows, exit 1–7 = success), Rsync (Unix/macOS) |
| Release | goreleaser v2.15.4 |

## Architecture Facts

- **Single binary:** Coordinator embeds the Vue dashboard via `//go:embed` at compile time
- **Dashboard build output:** `coordinator/static/dist/` → embedded by `coordinator/static/static.go`
- **Agent communication:** Persistent WebSocket to coordinator (`/ws/agent`), auto-reconnect with exponential backoff
- **Agent failover:** Round-robin through `coordinators` list in `agent-config.yaml`; resets on success
- **Federation model:** Root is source of truth; spoke→root sync only; event log per-coordinator (monotonic sequence)
- **Pagination standard:** All list endpoints — `?page=` (1-indexed), `?limit=` (default 25, max 100)
- **Version injection:** `-X main.Version={{.Version}}` via goreleaser ldflags

## Current Project State

- **Release:** v1.0.0 (Phase 17 complete — production ready)
- **Tests:** 110 total (108 pass + 2 skip on Windows)
- **Branch:** main
- **Next:** Phase 18+ — CLI tooling, additional backends, advanced analytics (see `MEMORY.md` Future Roadmap)

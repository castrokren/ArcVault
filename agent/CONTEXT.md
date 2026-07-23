# CONTEXT — agent

Last updated: 2026-07-22

## What happens here
The spoke. A Windows/macOS/Linux service installed on each backup host. Registers with
the coordinator using a per-agent token, sends heartbeats, executes backup jobs
(robocopy/rsync), and reports status back over WebSocket.

## Process — how work flows
1. `service/` — OS service wrapper starts `main.go`.
2. Agent registers with coordinator using its token (`config/`), gets assigned an agent ID.
3. `heartbeat/` — periodic heartbeat to coordinator (status, health).
4. `ws/` — persistent WebSocket connection; receives job/cancel commands from coordinator,
   pushes job output back.
5. `runner/` — executes the actual robocopy/rsync job, captures output, honors cancel.
6. `updater/` — self-updates when coordinator signals a new version; supports one-version
   rollback.
7. On coordinator loss, agent fails over across its configured coordinator list with
   exponential backoff.

## What files live here
- `main.go` — entry point, subcommand dispatch
- `runner/` — job execution (robocopy/rsync)
- `heartbeat/` — heartbeat loop
- `ws/` — WebSocket client, command handling
- `updater/` — self-update logic
- `service/` — Windows/macOS/Linux service install/start/stop
- `config/` — agent config loading (coordinator list, token)
- `honcho/` — (agent-side helper — see source for current scope)

## Standards / rules specific to this workspace
- **Contract-tested doc:** `docs/service.md` locks the agent's service name and `main.go`
  subcommands against `internal/docs` (Go tests). Rename a subcommand or the service
  constant → update the CONTRACT block in the same commit.
- Go packages here: lowercase, single word. Files: snake_case.
- Never hardcode version — it flows from the root `VERSION` file via ldflags at build time
  (`scripts/rebuild-and-restart.ps1`).

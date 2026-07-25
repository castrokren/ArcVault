# CONTEXT — agent

Last updated: 2026-07-25

## The auth token is mutable — read it, never copy it
Registration can **exchange** a short-lived enrollment token for a permanent per-agent one, so
the token is not a constant for the process lifetime. Everything reads it through
`config.TokenStore.Get()` at request time; `TokenStore.Replace()` persists the new value by
rewriting only the `auth_token` line of `agent-config.yaml` (comments and every other key,
notably `ca_cert_file`, survive).

`heartbeat.Config`, `runner.Config` and `ws.Client` all hold `Tokens *config.TokenStore`, not a
string. They used to copy `cfg.AuthToken` at construction, which meant a token replaced during
registration never reached the job runner or the WS client — they kept using the enrollment
token until it expired an hour later, and job execution died silently while the heartbeat
looked fine.

`BuildTLSConfig("")` returns `nil`, meaning **system roots** — so an empty or missing
`ca_cert_file` produces `x509: certificate signed by unknown authority` against a self-signed
coordinator. A *dangling* path is worse: it errors, and since that failure is non-fatal the
agent runs with its heartbeat disabled.

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

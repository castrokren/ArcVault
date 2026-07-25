# ArcVault Services (Windows)

Both the coordinator and the agent run as Windows services under the Service Control Manager (SCM).
This doc explains how they start, where they log, and how to diagnose the classic **error 1067**.

The service names and CLI subcommands below are **test-enforced** by `internal/docs`
(`TestServiceDoc_*`) against the constants in `coordinator/service` / `agent/service` and the
`switch` in each `main.go`. Rename a service or add a subcommand and this doc must change with it.

## The two services

<!-- CONTRACT:service-names — checked against CoordinatorServiceName / AgentServiceName consts -->
- `arcvault-coordinator`
- `arcvault-agent`
<!-- /CONTRACT:service-names -->

The installer registers each with `<binary> run-service` as the service command
(`installer/windows/arcvault_installer.py` → `sc create`, or `<binary> install-service`).

## `run-service` vs `start` — why it matters

The SCM does not run a normal program; it runs one that performs the `svc.Run` handshake
(`golang.org/x/sys/windows/svc`). So the service command is **`run-service`**, never `start`:

- `run-service` → `service.RunService()` → `svc.Run()` → the handler signals `StartPending`, launches
  the real work in a goroutine, then signals `Running`. This is what the SCM requires.
- `start` (coordinator) / no-args (agent) → runs the server directly in the foreground, for local dev.
  If the SCM launched this, it would never get the handshake and would kill the process.

## Error 1067 ("The process terminated unexpectedly")

1067 means the process exited **before** it finished the `StartPending → Running` handshake — i.e.
it died during startup. Under the SCM there is **no console**, so anything written to stdout/stderr is
lost. That is why 1067 historically appeared with no explanation.

Fixed causes (this is what to check first on a fresh install):

- **Coordinator unreachable at agent boot** — the agent used to `log.Fatalf` when its first
  registration failed (firewall, boot order, coordinator still starting, TLS pin). Now registration
  retries in the background and never exits; heartbeat starts only after it succeeds.
- **Bad TLS config in a goroutine** — `heartbeat.Start` used to `log.Fatalf` on TLS-config failure,
  taking the whole process down. Now it logs and disables heartbeat.
- **Corrupt config** — a malformed `agent-config.yaml` / `config.json` fails `config.Load` at startup.
  (A deploy-script `-replace` bug once wrote an invalid `auth_token:` line; fixed.)

## Where startup errors go now

Because the SCM discards the console, both binaries redirect their logger to a file beside the exe:

- **Agent** → `<install-dir>\logs\arcvault-agent.log` (set up in `agent/service` `RunService`).
- **Coordinator** → `<install-dir>\coordinator-service.log` (set up in `coordinator/main.go` `run-service`).

**To diagnose a 1067 on any machine:** open that log. Or run the binary directly from an elevated
prompt in the install dir to see the error live: `agent.exe` (agent) or `coordinator.exe start`
(coordinator).

## CLI subcommands

Dispatched from each `main.go`'s `switch os.Args[1]`.

### Coordinator (`coordinator.exe <cmd>`)

<!-- CONTRACT:coordinator-commands — checked against coordinator/main.go case labels -->
- `--version`
- `version`
- `init`
- `start`
- `create-agent-token`
- `prune-bootstrap-tokens`
- `rekey`
- `rekey-cert`
- `check-update`
- `run-service`
- `install-service`
- `uninstall-service`
- `help`
<!-- /CONTRACT:coordinator-commands -->

### Agent (`agent.exe <cmd>`)

<!-- CONTRACT:agent-commands — checked against agent/main.go case labels -->
- `run-service`
- `install-service`
- `uninstall-service`
- `--version`
- `version`
- `help`
<!-- /CONTRACT:agent-commands -->

(The agent with **no argument** runs the agent in the foreground — the dev/debug path.)

## Install / uninstall

`install-service` and `uninstall-service` register/remove the service via
`golang.org/x/sys/windows/svc/mgr` (`*/service/service_windows.go`). Install auto-deletes any
existing registration first and waits for the SCM to release it (handles "marked for deletion" 1072).
Requires an elevated (admin) prompt.

## Adding a subcommand

1. Add a `case "<cmd>":` to the relevant `main.go`.
2. Add the token to the matching CONTRACT block above (or run the test — it prints the corrected block).

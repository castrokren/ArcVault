# CONTEXT — installer

Last updated: 2026-07-24

## What happens here
A single Windows .exe (Python/Tkinter + PyInstaller) that installs the coordinator and/or
agent as Windows services. Lives under `installer/windows/`.

## Process — how work flows
1. `arcvault_installer.py` — Tkinter GUI/CLI driving the install (choose coordinator/agent,
   collect config, write service).
2. PyInstaller packages it per `ArcVault-Setup-*.spec` / `arcvault.spec` into a single .exe
   in `dist/`.
3. `arcvault.nsi` — NSIS script for the installer wrapper.
4. `Install-ArcVault.ps1` — PowerShell-driven install path (alternative to the GUI).
5. Installed services are the same `coordinator.exe` / `agent.exe` binaries built by
   `scripts/rebuild-and-restart.ps1` — the installer does not rebuild them.

## What files live here
- `arcvault_installer.py` — main installer logic
- `arcvault.nsi` — NSIS packaging script
- `Install-ArcVault.ps1` — PowerShell install path
- `agent-config.yaml` — default agent config template
- `icon.ico` — installer icon
- `build/`, `dist/` — PyInstaller build output (gitignored, not source)

## Install order is load-bearing (do not reorder `do_install`)
```
coordinator config → coordinator service → mint agent token → agent config → agent service
```
The coordinator must be installed **before** the agent, for two reasons:

1. Minting the agent's own token shells out to
   `coordinator.exe create-agent-token <id> --token-only`, which opens `config.json` and the
   DB sitting next to that exe. No running service or network call is needed, but the binary
   has to be in place. (Safe concurrently with the starting service only because the DSN now
   really applies WAL — see `coordinator/CONTEXT.md`.)
2. `write_agent_config` copies the coordinator's `cert.pem`, which the coordinator generates
   on its **first start**. The installer waits up to 15s for it and fails loudly if it never
   appears.

## Standards / rules specific to this workspace
- **Never set `agent_token = admin_token`.** The installer did this until 2026-07-24, so every
  agent it deployed authenticates with a permanent fleet-wide credential that names no machine
  and can't be revoked individually. Co-installs now mint a per-agent token (above);
  agent-only installs take an operator-pasted token, which should come from the dashboard's
  **Enroll Agent** button (1-hour expiry).
- **Only write `ca_cert_file` when a cert was actually copied.** A path to a nonexistent file
  is worse than omitting the key: `BuildTLSConfig` errors, and since the error is non-fatal
  the agent runs with its **heartbeat silently disabled**. Empty means "use system roots".
  Agent-only installs have no local cert to copy, so they warn and point at Enroll Agent,
  whose script embeds the cert.
- Secrets the installer must persist to the service registry `Environment`, not `config.json`:
  `ARCVAULT_CREDENTIAL_KEY` and `ARCVAULT_JWT_SECRET`. Missing the JWT secret makes the
  coordinator generate a random one per start, invalidating every session on restart.
- Never commit `config.json` produced here — it's gitignored and may contain secrets.
- The installer must stay in sync with current service names/subcommands documented in
  `docs/service.md` — if `agent/service/` or `coordinator/service/` change, verify the
  installer still targets the right service name.

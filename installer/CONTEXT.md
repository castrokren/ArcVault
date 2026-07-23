# CONTEXT — installer

Last updated: 2026-07-22

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

## Standards / rules specific to this workspace
- Never commit `config.json` produced here — it's gitignored and may contain secrets.
- The installer must stay in sync with current service names/subcommands documented in
  `docs/service.md` — if `agent/service/` or `coordinator/service/` change, verify the
  installer still targets the right service name.

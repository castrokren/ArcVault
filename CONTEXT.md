# ArcVault2.0 -- Project Status
**Last updated:** May 15, 2026

## Current Phase
Phase 6 in progress. Service installation complete.

## What works
- `coordinator init` / `start` / `install-service` / `uninstall-service`
- `agent` (no args = run) / `install-service` / `uninstall-service`
- Single binary deployment, dashboard embedded
- 45 tests passing
- v0.1.0 released on GitHub, v0.2.0 ready to tag

## Service installation
| Platform | Coordinator | Agent |
|----------|-------------|-------|
| Windows | SCM via golang.org/x/sys/windows/svc/mgr | same |
| Linux | /etc/systemd/system/arcvault-coordinator.service | /etc/systemd/system/arcvault-agent.service |
| macOS | /Library/LaunchDaemons/com.arcvault.coordinator.plist | /Library/LaunchDaemons/com.arcvault.agent.plist |

## Phase 6 remaining candidates
1. Per-agent tokens -- each agent authenticates with its own token
2. Failure notifications -- webhook or email on job failure
3. `coordinator check-update` -- checks GitHub releases API
4. Dashboard improvements -- pagination, search, theme toggle

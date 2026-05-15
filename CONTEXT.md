# ArcVault2.0 -- Project Status
**Last updated:** May 15, 2026

## Current Phase
**Phase 6 complete.** Self-update system fully implemented.

## What works
- `coordinator init` / `start` / `create-agent-token <id>` / `check-update` / `install-service` / `uninstall-service`
- `agent` (no args = run) / `install-service` / `uninstall-service`
- Single binary deployment, dashboard embedded
- Per-agent tokens: each agent gets its own token via `coordinator create-agent-token <agent-id>`
- Admin token still works for dashboard and management
- **Self-update system:**
  - `coordinator check-update` — check for newer releases (CLI)
  - `/api/update/check` — cached version info (dashboard)
  - `/api/update/apply` — initiate update with progress streaming (WebSocket)
  - Background poller: checks GitHub releases every 24h
  - Dashboard banner + modal with multi-state UI (confirm → progress → success/error)
  - Atomic update flow: download → verify → stage → restart (Windows/Linux/macOS)
- **65 tests passing** (51 original + 14 new: 9 updater + 5 server)
  - coordinator/server: 51 (includes agent token tests)
  - coordinator/updater: 9 (new)
  - agent/runner: 5
+ coordinator/updater/ — platform-agnostic download/verify/stage
+ coordinator/updater/{windows,linux,darwin}.go — service control
+ coordinator/server/update.go — REST endpoints + progress streaming
+ dashboard/src/components/{UpdateBanner,UpdateModal}.vue

## Per-agent token workflow
```
# Generate token for an agent
coordinator create-agent-token agent-01
# → prints token, add to agent-config.yaml as auth_token

# Agent authenticates with its own token
# Admin token stays private to dashboard/operator
```

## Service installation
| Platform | Install command | Start command |
|----------|----------------|---------------|
| Windows (admin) | coordinator install-service | sc start arcvault-coordinator |
| Linux (root) | sudo coordinator install-service | sudo systemctl start arcvault-coordinator |
| macOS (root) | sudo coordinator install-service | sudo launchctl start com.arcvault.coordinator |
| Same for agent | agent install-service | (platform equivalent) |

## Phase 7 (not started)
Possible future work:
- Dashboard improvements: pagination, search, theme toggle
- Agent self-update
- Rollback to previous version

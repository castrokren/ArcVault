# ArcVault2.0 -- Project Status
**Last updated:** May 15, 2026

## Current Phase
Phase 6 in progress. Failure notifications complete.

## What works
- `coordinator init` / `start` / `create-agent-token <id>` / `install-service` / `uninstall-service`
- `agent` (no args = run) / `install-service` / `uninstall-service`
- Single binary deployment, dashboard embedded
- Per-agent tokens: each agent gets its own token via `coordinator create-agent-token <agent-id>`
- Admin token still works for dashboard and management
- Failure notifications: webhook and email on job failure/success, agent offline alerts
- 58 tests passing
+ coordinator/notifications/ — webhook + email, per-job notify_on
+ notifications.yaml — optional global config

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

## Phase 6 remaining
- ~~Failure notifications (webhook/email on job failure)~~ ✅ **COMPLETE**
- `coordinator check-update` (GitHub releases API)
- Dashboard improvements (pagination, search, theme toggle)

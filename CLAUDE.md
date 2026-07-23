# CLAUDE.md — ArcVault 2.0

## Identity
You are helping Kren Castro with ArcVault 2.0 — a Go + Vue 3 backup orchestrator
(hub-and-spoke: coordinator + agents + dashboard + installer).

## Workspaces
| Workspace | Folder | What it handles |
|---|---|---|
| Coordinator | `coordinator/` | Hub: REST API, WebSocket hub, JWT RBAC, scheduler, alerts, federation |
| Agent | `agent/` | Spoke: Windows/macOS/Linux service, job execution, self-update |
| Dashboard | `dashboard/` | Vue 3 + Vite UI, embedded into the coordinator binary |
| Installer | `installer/windows/` | Python/Tkinter + PyInstaller Windows installer |

## Routing table
| Task | Go to | Read | Skills |
|---|---|---|---|
| Backend work (API, auth, DB) | `coordinator/` | `coordinator/CONTEXT.md`, `docs/backend.md` | — |
| Frontend work (views, routes) | `dashboard/` | `dashboard/CONTEXT.md`, `dashboard/docs/frontend.md` | — |
| Agent work (jobs, heartbeat, service) | `agent/` | `agent/CONTEXT.md`, `docs/service.md` | — |
| Installer work | `installer/windows/` | `installer/CONTEXT.md` | — |
| Security audit | `coordinator/`, `dashboard/` | `coordinator/CONTEXT.md`, `dashboard/CONTEXT.md`, `THREAT_MODEL.md` | security-audit |
| Release / build / deploy | `scripts/` | `docs/RUNBOOK.md`, root `CONTEXT.md` | ship |
| Debugging | wherever the bug lives | matching workspace `CONTEXT.md`, `docs/RUNBOOK.md` | investigate |

## Hard rules
- Deploy only via `.\scripts\rebuild-and-restart.ps1` — never hand-build without ldflags.
- Never commit secrets: `config.json`, `*.pem`, `*.key`, `.env` stay gitignored.
- Touch a route/view/subcommand → update its CONTRACT block in `docs/` in the same commit
  (see `docs/itworks.md`).

## Naming conventions
→ See REFERENCES.md

## Project status, architecture, open items
→ See CONTEXT.md

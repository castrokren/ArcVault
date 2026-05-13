# Planning Workspace

**Last updated:** May 13, 2026

## What happens here

Deciding what to build next in ArcVault. Breaking phases into ordered tasks. Tracking what's done and what's blocked.

## Phase 2 task order

Implement in this sequence — each step unblocks the next:

1. `coordinator/config/` — Config struct + file persistence
2. `coordinator/db/` — SQLite init and schema (tables: agents, jobs, job_runs, tokens)
3. `coordinator/server/` — Gorilla mux router wired into StartCommand()
4. `POST /api/agents/register` and `POST /api/agents/{id}/heartbeat` endpoints
5. Agent heartbeat loop (every 30 seconds)
6. Job scheduling via robfig/cron
7. WebSocket endpoint for real-time dashboard push
8. Vue 3 + Vite dashboard (new `dashboard/` folder)

## Process

1. Check root CONTEXT.md for current phase and open items
2. Pick the next unblocked task from the order above
3. Break it into concrete steps before writing any code
4. Switch to the Building workspace to implement

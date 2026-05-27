---
name: ArcVault Decisions
category: memory
priority: high
last_updated: 2026-05-26
last_accessed: 2026-05-26
stale_after_days: 90
auto_summarize: true
archive_policy: keep
---

# ArcVault Architecture Decisions

Key decisions made during development. Check here before making architectural choices. Full rationale in `MEMORY.md`.

---

## Auth & Security

- **Admin token OR agent token** — middleware accepts both; enables per-agent scoping without breaking admin workflows
- **JWT RBAC** — three roles (admin, operator, viewer) with fine-grained endpoint access; stored in DB
- **Webhook signature** — `X-ArcVault-Signature: sha256=<hex>` (GitHub convention); 10s timeout
- **Password hashing** — bcrypt; minimum 8 chars enforced on frontend (UX only, not cryptographic constraint)
- **Session persistence** — remember-me = localStorage; unchecked = memory-only (browser close = logout)

## Database

- **No CGO** — `modernc.org/sqlite` (pure Go); enables single binary cross-platform builds
- **Additive migrations only** — never drop columns; add nullable or defaulted columns only
- **Federation events** — per-coordinator monotonic sequence log; 7-day retention; append-only
- **Alert history** — 30-day retention by default (configurable)

## Notifications & Alerting

- **Async retry** — RetryDispatch runs in goroutine; never blocks job result handler
- **Slack/Teams** — incoming webhooks only; no OAuth, no app installation required
- **3 rule types** — `on_failure`, `duration_exceeded`, `missed_schedule`; stored in DB for runtime updates without restart
- **Missed schedule deduplication** — checks alert_history before firing to avoid repeat-firing

## Federation

- **Root is source of truth** — state sync root→spoke only; spokes never push state changes to root
- **Standalone mode safe** — spoke continues running jobs when disconnected from root
- **Sequence numbers per-coordinator** — avoids clock sync complexity
- **Agent failover client-side** — stateless routing; no coordinator overhead; round-robin with exponential backoff (30s → 60s → 120s)
- **Agent homing** — `home_coordinator` written on register + heartbeat; used for `agent_count` in health endpoint

## Dashboard

- **No state management library** — composables + `ref`/`reactive` only; Vuex/Pinia explicitly rejected (overkill for scope)
- **Auto-refresh pattern** — composables poll on interval (e.g. useFederationLag 15s, history 30s); not WebSocket-pushed
- **Visible-but-disabled nav** — operator/viewer see admin links but they're disabled with title text; no hidden routes
- **Smart job form** — user toggles Single Agent ↔ Group mode; validates based on selection before API call

## Updates & Rollback

- **Backup on every update** — `coordinator.previous` / `agent.previous` in platform backup paths
- **One-version-back only** — rollback limited to immediately prior binary
- **Binary never touched before staging completes** — safety: stage first, then swap atomically
- **Background poller** — 24h update check interval; silent failure recovery

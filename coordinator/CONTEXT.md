# CONTEXT — coordinator

Last updated: 2026-07-25

## What happens here
The hub of ArcVault's hub-and-spoke architecture. A Go HTTPS server (default port 443)
that serves the REST API (`/api/*`), the embedded Vue dashboard (`static/dist`), and a
WebSocket hub for connected agents. Storage is SQLite in WAL mode.

## Process — how work flows
Three layers, strictly one direction:

```
server/   (HTTP handlers, auth, routing)
   ↓
business/ (services: AgentService, JobService, UserService, GroupService, AuditService)
   ↓
db/       (query interfaces, SQLite)
```

Handlers never touch `db/` directly — they call `business/`, which calls `db/`.

## What files live here
- `server/` — HTTP handlers, JWT auth, federation, alert rules, credentials, agent tokens
- `business/` — service layer (DTOs), one file per domain (agents, jobs, users, groups, audit)
- `db/` — SQLite query layer, schema, migrations
- `api/` — API type definitions
- `notifications/` — webhook/email/Slack/Teams notifiers
- `updater/` — self-update from GitHub releases
- `service/` — Windows/macOS/Linux service wrapper (install/start/stop)
- `config/` — coordinator config loading
- `cmd/` — CLI entry subcommands
- `static/` — embedded dashboard build output (`dist`)
- `main.go` — entry point
- `tests/` — integration tests

## Standards / rules specific to this workspace
- **The SQLite driver is `modernc.org/sqlite`, whose DSN pragma syntax is
  `?_pragma=name(value)`.** mattn/go-sqlite3-style keys (`_busy_timeout=`, `_journal_mode=`)
  are **silently ignored** — the project ran without WAL or a busy timeout for its entire
  history because of this, which made every concurrent request a SQLITE_BUSY candidate and
  (revocation checks being fail-closed) logged users out on tab clicks. `db/dsn_test.go`
  pins the syntax; do not "tidy" the DSN.
- **TLS material is generated on start, not only by `coordinator init`.**
  `ensureTLSMaterial` (`cmd/commands.go`) defaults `cert_file`/`key_file` to `cert.pem` /
  `key.pem` beside the executable and calls `tlscert.EnsureExists`. It is idempotent on
  purpose — regenerating would break every agent pinning the old cert. Failure is fatal only
  when `environment = production`, because falling through to `ListenAndServe` serves agent
  tokens and JWTs in cleartext (which is exactly what fresh installs used to do).
- **Minting a token must never revoke one.** `CreateAgentToken` is additive on purpose.
  `POST /api/agents/{id}/token` is the dashboard's "Get Token" button — an operator *reading*
  a token — and nothing writes it into the running agent's config. A version that deleted the
  agent's other tokens at mint time took the live agent down the moment that button was
  clicked. Cleanup belongs in `handleRegister` via `SupersedeAgentTokens(agentID, keepToken)`,
  which runs only after the agent has proven which credential it holds.
- **Two token kinds, one exchange.** `bootstrap*` = enrollment, 1-hour expiry, several may be
  pending at once. Anything else = per-agent, never expires. `handleRegister` swaps the first
  for the second and leaves the enrollment token to expire on its own (deleting it would brick
  a machine whose response was lost). See `docs/backend.md`.
- **All machine-token auth goes through `acceptMachineToken`** (`server/server.go`), shared by
  `authMiddleware`, `agentOrViewerRoute`, `agentOrOperatorRoute`, `agentOrAdminRoute`. Add a
  new agent-facing route → route it through that helper, never re-inline the token checks.
  Its `isAdminToken` branch is deprecated; see root `CONTEXT.md` for the removal gate.
- **`tlscert.ReadCertPEM` returns PEM, not DER**, despite what an older doc comment claimed.
  Callers needing DER (`x509.ParseCertificate`, a Windows-style SHA-1 thumbprint) must
  `pem.Decode` first and use `block.Bytes`.
- **`cfg.Host` may be empty — never interpolate it blindly into a URL.** It is optional and the
  installer never writes it, so `fmt.Sprintf("https://%s", cfg.Host)` shipped enrollment
  scripts pointing at the literal string `https://`. Use `coordinatorBaseURL()`, which falls
  back to the request's Host header and refuses a loopback result.
- **`Server.tokenCache` is dead code** — declared and initialised, never read or written. Every
  agent-token check hits the DB through `ValidateToken`. Do not assume caching when reasoning
  about token validity or writing tests.
- **Password complexity:** min 8 chars, uppercase, lowercase, digit, special character —
  enforced in `validatePasswordStrength()` (`server/auth.go`) and mirrored in
  `business/users.go`.
- **Admin UI routes are guarded server-side too** — `adminRoute` middleware requires admin
  role + completed password change for GET/POST/DELETE `/api/users`, PUT `/api/users/{id}/role`.
- **Pagination:** `MaxPage` = 10000, `MaxLimit` = 100, `DefaultLimit` = 25 — enforced on
  every paginated endpoint.
- **Contract-tested docs:** `docs/backend.md` (every route) and `docs/service.md` (service
  names + subcommands) are checked against this code by `internal/docs` (Go tests). Change
  a route or subcommand → update the matching CONTRACT block in the same commit, or the
  pre-commit hook blocks it. See `docs/itworks.md`.
- Go packages here: lowercase, single word. Files: snake_case.

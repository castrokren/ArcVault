# STATE — ArcVault security hardening

## Goal
Close the auth/session vulnerabilities found in the 2026-07-08 audit of the coordinator, verifying each fix against the running production instance rather than tests alone.

## Invariants / decisions
- **Deploy only via `.\scripts\rebuild-and-restart.ps1`.** Never hand-build without ldflags (version flows from `VERSION`).
- **Do not modify dashboard token storage.** Reuse `useAuth.login()`; the localStorage key is `arcvault_token` (mirrored to `arcvault_jwt`).
- **Live prod config is `C:\ArcVault\config.json`**, NOT the repo `config.json`. It has `environment = production`, a 64-char `admin_token`, and **no `jwt_secret`** (so the secret regenerates every restart and all sessions die).
- **Live DB is `C:\ArcVault\arcvault.db`.** `~/.arcvault/arcvault.db` is a stale June snapshot — do not draw conclusions from it.
- `modernc.org/sqlite` binds `time.Time` as Go's `String()` form in **local time**. Never bind a `time.Time` into a `DATETIME` column compared against `datetime('now')` (UTC). Use `db.sqliteTime()`.
- Revocation checks are **fail-closed**: no `jti`, or an unreachable revocation store, means reject.
- Login throttle keys must derive from `r.RemoteAddr`, never `X-Forwarded-For` — a forgeable key defeats the limit.
- Verification means **driving the real API/browser**, not just `go test`. See "Lesson" below.

## Done
All deployed to prod (coordinator.exe built 12:32:47, PID restarted 12:32:51) and verified live:

- **Logout actually revokes.** `GenerateJWT` now issues a random 16-byte `jti`; `tokenRevoked()` is the single fail-closed guard used by both `JWTMiddleware` and `handleWS`. `handleLogout` returns 500 instead of a false `{"ok":true}` if `RevokeToken` fails. Verified: `/api/auth/me`, `/api/jobs`, `/api/agents` all 200 → logout → all **401**, and `wss://localhost/ws` **rejected**.
- **`sqliteTime()` fix (db.go).** Root cause of the above actually failing in prod despite green tests. Applied to both bind sites: `RevokeToken` and `CreateAgentToken`.
- **Empty/constant-time admin token.** New `bearerToken()` / `isAdminToken()` / `isAgentToken()` helpers; all 6 middlewares + `handleAgentWS` route through them. Closes `"" == ""` bypass when `AdminToken` is unset, and the non-constant-time compare.
- **Per-account login lockout.** Throttle now keys on `ip:<addr>` AND `user:<username>` (lowercased, applied whether or not the account exists). 5 burst / 1-per-12s / 10-min lockout at 10 failures. Also fixed IPv6 port-stripping and unbounded `loginLimiters` map growth (prune idle entries past 4096 keys).
- **`X-Forwarded-For` no longer trusted by default.** New `trust_proxy_headers` config field, wired via `business.SetTrustProxyHeaders()` in `NewWithFS`. When enabled, takes the **rightmost** XFF entry (the one your proxy appended), not the client-controlled leftmost.
- **Audit log `Success`** is now `< 400`, not `< 500` — 401/403 no longer recorded as successes.
- **Security headers + CSP** on all responses; HSTS gated on `r.TLS != nil || externalTLS`. Verified no CSP violations in the real dashboard (clean console, WebSocket "Live").
- **Confirmed already fine:** no SQL injection (all parameterized), no `v-html`/`innerHTML`, no path traversal in `downloads.go`, bcrypt cost 10, generic login error (no user enumeration), WebSocket origin validation real and production-gated, `admin`/`changeme` already rotated.

## In-progress
- **NOTHING IS COMMITTED.** 8 modified files + 2 new test files sit in the working tree. Deployed but unversioned — a `git checkout` destroys all of it. **Commit this first, before any new work.**

## Next
Ordered by value:

1. **Commit the working tree.** `git status` shows: modified `business/audit.go`, `business/audit_test.go`, `config/config.go`, `db/db.go`, `server/auth.go`, `server/hub.go`, `server/request_audit.go`, `server/server.go`; untracked `db/token_expiry_test.go`, `server/auth_hardening_test.go`.
2. **Set `ARCVAULT_JWT_SECRET` in prod** (`C:\ArcVault\service-run.bat` already sets `ARCVAULT_CREDENTIAL_KEY`, add it there). Without it the secret regenerates each restart, every session dies, and working revocation is moot. One line, highest value-per-effort.
3. **Wire `PruneExpiredTokens()`.** Defined at `db.go:165`, **never called** by anything outside tests. `revoked_tokens` grows one row per logout, forever. Suggested: fold into the existing offline-detector ticker in `Server.Start()`. (Earlier claim that the malformed row "self-cleans" was WRONG — nothing prunes.)
4. **Rate-limit `handleChangePassword`.** ~10 lines, reuse `loginKeyAllowed()` with a `pwchange:<userid>` key. Currently a stolen token allows unlimited old-password brute force.
5. **Admin-token architecture (#3).** `GET /api/admin/token` (`server.go:368`) hands a permanent, unrevocable, role-bypassing credential to any admin session. One XSS → permanent compromise surviving password rotation. Load-bearing for agent registration + installer, so this is a scoped-token redesign, not a patch. Needs a design conversation.
6. **Plaintext agent tokens (#6).** `tokens.token` stored raw, matched by equality (`db.go:123`). Store `sha256(token)` instead. Touches registration, installer, and every deployed agent — needs a migration path. Lower urgency: requires DB file access, which already implies compromise.
7. **Password policy** is length ≥ 8 only. Recommendation: **skip character-class rules** (they produce `Password1!`). If pursuing, a breached-password check at set time is worth more.

### Known-dirty prod data (cosmetic, non-blocking)
- `revoked_tokens` has 1 malformed row: `expires_at = '2026-07-08 16:24:02 -0400 EDT'`. Harmless (never matches) but won't be removed until (3) lands.
- `tokens` has 2 malformed bootstrap rows dated 2026-06-11 with Go monotonic-clock suffixes (`m=+3632.86...`). These could **never** have validated — if agent bootstrap seemed flaky around then, this was why.
- `tokens` holds **29 agent tokens with `expires_at = NULL`** across only 3 agents. Non-expiring, plaintext, no revocation path. Suggests tokens accumulate on re-registration rather than being replaced. Worth investigating alongside (6).

## Open questions
- Per-account lockout is a **deliberate availability tradeoff**: an attacker can lock out a legitimate user by burning 10 failures against their username. Self-heals in 10 min. Is that acceptable for `admin` during an incident, or do we need a break-glass source-IP exemption?
- `recordLoginSuccess` clears the failure counter and lockout but **not** the token bucket, so 5 rapid *successful* logins from one IP still hit 429. Pre-existing behavior, unchanged. Leave?
- `PUT /api/users/{id}/role` is registered but `handleUpdateUserRole` requires `PATCH` → **always 405**. Dead endpoint. Fix the verb or drop the route?
- Role `operator` can't be created — `CreateUserRequest.Validate()` and `UpdateUserRoleRequest.Validate()` both allow only `admin`|`viewer` — yet user id 3 in prod has `operator`. How did it get set, and is `operator` supposed to be creatable?
- User-management handlers (`handleListUsers` etc.) use raw `JWTMiddleware` + an inline `claims.Role != "admin"` check instead of `adminRoute`, so they skip `RequirePasswordChange`. Consolidate onto `adminRoute`?

## Lesson (do not summarize away)
`TestLogoutRevokesToken` passed while logout was **completely broken in production**. The token lives 4h and the machine's offset is −4h, so the local-time expiry string coincided with the UTC issue time; the test ran in 160ms, inside the same second, and a string prefix tie made the comparison succeed. Green by arithmetic accident.

A negative control proved the test *could* fail. It did not prove the test measured the right thing. The Go-level round trip never exercised SQLite's actual `expires_at > datetime('now')` comparison. **Assert on observable behavior through the real interface, and inspect what's actually stored.** The new `db/token_expiry_test.go` pins `time.Local` to a −4h zone (so it bites on a UTC CI box), reads raw bytes via `expires_at || ''` (defeating the driver's `DATETIME`→`time.Time` conversion), and sleeps 1.1s to force the clock past the prefix tie.

## File map
- `coordinator/server/auth.go` — JWT issue/validate, `newJTI`, `tokenRevoked`, login throttle (`loginKeyAllowed`, `clientIPKey`, `accountKey`, prune), auth handlers
- `coordinator/server/server.go` — `bearerToken`/`isAdminToken`/`isAgentToken`, all middlewares, `securityHeaders`, route table
- `coordinator/server/hub.go` — `handleWS` (JWT + revocation), `handleAgentWS` (admin/agent token)
- `coordinator/server/request_audit.go` — per-request audit middleware (`Success` threshold)
- `coordinator/db/db.go` — `sqliteTime()`, `RevokeToken`, `IsTokenRevoked`, `PruneExpiredTokens` (uncalled), `CreateAgentToken`, `ValidateToken`
- `coordinator/business/audit.go` — `ClientIP()`, `SetTrustProxyHeaders()`
- `coordinator/config/config.go` — `TrustProxyHeaders` field; `Save()` blanks `admin_token`/`jwt_secret` on disk
- `coordinator/server/auth_hardening_test.go` — NEW: jti, logout revocation, empty-admin-token bypass (6 middlewares), per-account throttle, XFF-rotation resistance
- `coordinator/db/token_expiry_test.go` — NEW: canonical UTC storage, clock-tick survival, prune, bootstrap-token validation
- `C:\ArcVault\` — live deployment (config.json, arcvault.db, service-run.bat)

## Pre-existing test failures (NOT caused by this work)
`internal/tlscert` (5 tests, `x509: malformed certificate`) and `internal/bootstrap` (`TestGenerateScript_crossEdition`). Confirmed failing identically on a clean `git stash` of HEAD. Unrelated — do not chase them as regressions.

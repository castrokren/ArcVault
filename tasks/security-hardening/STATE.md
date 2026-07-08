# STATE — ArcVault security hardening

## Goal
Close the auth/session vulnerabilities found in the 2026-07-08 audit of the coordinator, verifying each fix against the running production instance rather than tests alone.

## Invariants / decisions
- **Deploy only via `.\scripts\rebuild-and-restart.ps1`.** Never hand-build without ldflags (version flows from `VERSION`).
- **Do not modify dashboard token storage.** Reuse `useAuth.login()`; the localStorage key is `arcvault_token` (mirrored to `arcvault_jwt`).
- **Live prod config is `C:\ArcVault\config.json`**, NOT the repo `config.json`. It has `environment = production`, a 64-char `admin_token`, and no `jwt_secret` (the secret now comes from the env var — see below).
- **The service's env comes from the registry, not a script.** `HKLM\SYSTEM\CurrentControlSet\Services\arcvault-coordinator\Environment` (REG_MULTI_SZ, one `NAME=value` per entry). SCM injects it; verified empirically. (`C:\ArcVault\service-run.bat` — a dead script that launched the wrong binary and held a plaintext credential key — was **deleted 2026-07-08**; the service `ImagePath` never referenced it.)
- **The coordinator has no log sink.** `run-service` discards stdout; nothing writes a log file or event log. You cannot verify startup behavior by reading logs — drive the API.
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
- **Committed and merged.** `main` fast-forwarded to `cb5145f` (was 2 behind). A rebuild on `main` now deploys the hardened code — the "rebuild silently regresses prod" trap is closed.
- **`ARCVAULT_JWT_SECRET` set in prod.** 64 hex chars, written to the service key's `Environment` (registry only; never printed). Verified by observable behavior: a JWT minted before a restart still returned 200 on `/api/auth/me` and `/api/jobs` from a fresh PID, and again after a full `rebuild-and-restart.ps1`. Before this, every restart invalidated every session and made revocation moot.
- **Credential key moved off disk.** `ARCVAULT_CREDENTIAL_KEY` now lives in the service registry `Environment`; `credential_key` deleted from `C:\ArcVault\config.json` (backup: `config.json.bak-20260708-135629`). `config.Save()` now blanks `CredentialKey` alongside `AdminToken`/`JWTSecret` — it's `omitempty`, so the field drops entirely. Verified in three steps: (1) offline, the registry key decrypts the real prod ciphertext to valid JSON (AES-GCM authenticates, so a wrong key errors rather than yielding garbage); (2) post-restart, `POST /api/credential-profiles` returned **201**, not the 503 that a missing key produces — so `Encrypt()` ran against the env-sourced key; (3) the throwaway profile deleted (204) and the original row is untouched. Threat model: this raises "read one JSON file" to "read the service's registry key". Both need Administrator. It is not a vault.
- **`PruneExpiredTokens()` wired.** New `Server.StartTokenPruner(1h)` in `auth.go`, called from `Server.Start()`; prunes once at startup, then hourly. Verified in prod, both directions: the malformed `81714cf8…` row (`expires_at = '2026-07-08 16:24:02 -0400 EDT'`) was deleted, while a live logout revocation (`a5989f7c…`, expiring 20:34 UTC) survived. An over-eager prune would have un-revoked a logged-out session; it didn't.
- **`handleChangePassword` rate-limited.** A stolen token could brute-force the old password unboundedly; now keyed `pwchange:<userid>` through the existing login limiter — throttle-check before bcrypt (429 on trip), `recordLoginFailure` on wrong old password, `recordLoginSuccess` on success. Same 5-burst / 10-fail-lockout as login, but per-user (independent of login/IP keys). New `change_password_throttle_test.go` proves the 6th wrong guess for one user is 429 while a different user still gets through. **Not yet deployed** as of this entry.
- **`decryptCredentials()` no longer fails silently.** Now returns `(map, error)`, logs each failure (`[credentials]` prefix), and returns the error instead of a bare `nil`. Both callers (`jobs.go` list + get-one) return **500** when a job with a bound profile (`credProfileID != ""`) can't produce credentials — previously the agent got a job with silently-missing creds and ran the backup without them. Tightened the sibling path too: a real DB error from `GetJobCredentialProfileID` (was swallowed by `if err == nil`) now also refuses dispatch; the no-binding case still passes through. New `credentials_test.go` proves right-key succeeds, wrong-key (AES-GCM auth fail) and missing-profile both error. Deployed (binary 17:22:09, service Running). Not behaviorally verifiable live: zero prod jobs are bound to a profile, so there is no path to trigger it without fabricating a bad-key bound job on prod — the test with its AES-GCM negative control is the verification.

## In-progress
- Nothing. Working tree clean, `main` == deployed behavior.

## Next
Ordered by value:

1. **Credential key sits on disk next to the ciphertext it protects.** `loadCredentialKey()` (`server/credentials.go:21`) prefers `cfg.CredentialKey` and only falls back to `credcrypto.LoadKey()`. Prod `C:\ArcVault\config.json` **has** `credential_key` (64 chars, byte-identical to the one in the dead `service-run.bat`), so the env var is never consulted and encryption works fine. **Correction: an earlier note here claimed `LoadKey()` returns `ErrKeyNotSet` in prod. That was wrong — it is never called.**

   The actual defect: the key lives in `config.json`, in the same directory as `arcvault.db`. Anyone who can read the DB can read the key, so encryption-at-rest for the 1 stored `credential_profiles` row (`cred-d077217915c0b069`, SMB, 133 bytes) is decorative. Note `config.Save()` blanks `admin_token` and `jwt_secret` on write but **not** `credential_key` (`config.go:83-86`) — inconsistent.

   Fix: move the key to the service registry `Environment` (machinery now proven by the JWT secret), delete `credential_key` from `config.json`, and blank it in `Save()` alongside the other two. Migration is safe *because the two key values are identical* — same bytes, different source, so existing ciphertext still decrypts. `credcrypto.Rekey()` (`rekey.go`) exists if the key ever needs rotating.

   **DONE 2026-07-08** — see the credential-key entry under Done. Note for whoever reads this next: `GET /api/credential-profiles/{id}` does not exist (only POST/GET-list/DELETE), and `decryptCredentials()` swallows every failure into `nil`, so a broken key silently omits credentials from an agent's job payload rather than erroring. The only observable decrypt path is an agent fetching a job bound to a profile — and **zero jobs are bound to one**. That is why verification went through POST (201-vs-503) instead.

1. **Admin-token architecture** (scoped-token redesign, phased). Trace done this session — the admin token has four jobs: (a) **master key** on every JWT route via the `JWTMiddleware` fallback (`auth.go:173-197`, injects `role:admin`); (b) static bearer on agent endpoints (`authMiddleware`: register/heartbeat) and `adminTokenRoute`/`agentOrViewerRoute`, which have their **own** `isAdminToken` checks independent of the fallback; (c) exposed to the browser via `GET /api/admin/token` → `Users.vue` "Copy Admin Token"; (d) reused as an agent token by the installer (`arcvault_installer.py:447`). **Key discovery:** the scoped per-agent path already exists — `GET /api/admin/bootstrap.ps1` (`handleBootstrapScript`) mints a fresh per-agent token via `CreateAgentToken` and embeds it; agents auth with that (`cfg.AuthToken`), not the admin token.
   - **Phase 2 DONE 2026-07-08** — removed `GET /api/admin/token` (route + `handleGetAdminToken`) and the `Users.vue` "Copy Admin Token" button/handler so the raw token is no longer fetchable by browser JS (the XSS-exfil path). Removed the endpoint, not just the button — XSS could `fetch()` it directly. API spec section deleted. Coordinator + dashboard build clean, server tests green. **Committed, not yet deployed** (dashboard needs the embed rebuild via `rebuild-and-restart.ps1`).
   - **Phase 1 BLOCKED on a decision** — removing the `JWTMiddleware` admin-token master-key fallback would break the sanctioned deploy script: `rebuild-and-restart.ps1:255` uses the admin token as a Bearer on `GET /api/agents` (a `viewerRoute` = `JWTMiddleware`), which only works via that fallback. Need to decide how the deploy smoke-test authenticates once the admin token stops being a master key. (Aside found: the script's `POST /api/agent-tokens` at line 229 hits a route that **does not exist** — already failing into its catch.)
   - **Phases 3-4 (later):** short-lived/expiring enrollment tokens; installer switches off `agent_token = admin_token`; demote/remove admin token. Moves together with Next #2 (plaintext agent tokens).
2. **Plaintext agent tokens.** `tokens.token` stored raw, matched by equality (`db.go:123`). Store `sha256(token)` instead. Touches registration, installer, and every deployed agent — needs a migration path. Lower urgency: requires DB file access, which already implies compromise.
3. **Password policy** is length ≥ 8 only. Recommendation: **skip character-class rules** (they produce `Password1!`). If pursuing, a breached-password check at set time is worth more.

~~`decryptCredentials()` fails silently~~ — **DONE 2026-07-08**, see Done.
~~Rate-limit `handleChangePassword`~~ — **DONE 2026-07-08**, see Done (committed, deploy pending).
~~Delete `C:\ArcVault\service-run.bat`~~ — **DONE 2026-07-08**. Verified dead first: service `ImagePath` is `coordinator.exe run-service` (never referenced the .bat), registry `Environment` already holds `ARCVAULT_CREDENTIAL_KEY` (so its exported key was redundant). Deleted; service still Running. Removed a plaintext credential key from disk — not backed up on purpose (would re-expose the key).

### Known-dirty prod data (cosmetic, non-blocking)
- ~~`revoked_tokens` malformed row~~ — pruned on the 2026-07-08 13:20 deploy, as predicted.
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
- `coordinator/db/db.go` — `sqliteTime()`, `RevokeToken`, `IsTokenRevoked`, `PruneExpiredTokens`, `CreateAgentToken`, `ValidateToken`
- `coordinator/business/audit.go` — `ClientIP()`, `SetTrustProxyHeaders()`
- `coordinator/config/config.go` — `TrustProxyHeaders` field; `Save()` blanks `admin_token`/`jwt_secret`/`credential_key` on disk
- `coordinator/server/credentials.go` — `loadCredentialKey()` (config first, then env), `decryptCredentials()` (returns `(map, error)`, logs + errors on failure)
- `coordinator/server/credentials_test.go` — NEW: right-key decrypt succeeds; wrong-key (AES-GCM auth fail) and missing-profile both error
- `coordinator/server/change_password_throttle_test.go` — NEW: 6th wrong old-password for one user is 429; a different user is unaffected
- `coordinator/server/auth_hardening_test.go` — NEW: jti, logout revocation, empty-admin-token bypass (6 middlewares), per-account throttle, XFF-rotation resistance
- `coordinator/db/token_expiry_test.go` — NEW: canonical UTC storage, clock-tick survival, prune, bootstrap-token validation
- `C:\ArcVault\` — live deployment (config.json, arcvault.db; service-run.bat deleted 2026-07-08)

## Pre-existing test failures (NOT caused by this work)
`internal/tlscert` (5 tests, `x509: malformed certificate`) and `internal/bootstrap` (`TestGenerateScript_crossEdition`). Confirmed failing identically on a clean `git stash` of HEAD. Unrelated — do not chase them as regressions.

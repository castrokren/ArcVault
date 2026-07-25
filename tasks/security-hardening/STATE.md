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
- **`handleChangePassword` rate-limited.** A stolen token could brute-force the old password unboundedly; now keyed `pwchange:<userid>` through the existing login limiter — throttle-check before bcrypt (429 on trip), `recordLoginFailure` on wrong old password, `recordLoginSuccess` on success. Same 5-burst / 10-fail-lockout as login, but per-user (independent of login/IP keys). New `change_password_throttle_test.go` proves the 6th wrong guess for one user is 429 while a different user still gets through. **Deployed 2026-07-08** (binary 18:19:19).
- **`decryptCredentials()` no longer fails silently.** Now returns `(map, error)`, logs each failure (`[credentials]` prefix), and returns the error instead of a bare `nil`. Both callers (`jobs.go` list + get-one) return **500** when a job with a bound profile (`credProfileID != ""`) can't produce credentials — previously the agent got a job with silently-missing creds and ran the backup without them. Tightened the sibling path too: a real DB error from `GetJobCredentialProfileID` (was swallowed by `if err == nil`) now also refuses dispatch; the no-binding case still passes through. New `credentials_test.go` proves right-key succeeds, wrong-key (AES-GCM auth fail) and missing-profile both error. Deployed (binary 17:22:09, service Running). Not behaviorally verifiable live: zero prod jobs are bound to a profile, so there is no path to trigger it without fabricating a bad-key bound job on prod — the test with its AES-GCM negative control is the verification.
- **Password complexity validation (2026-07-13).** New `validatePasswordStrength()` in `auth.go` enforces minimum 8 characters plus at least one uppercase, lowercase, digit, and special character. Applied to login (`POST /api/auth/login`), password change (`PUT /api/auth/change-password`), and user creation (`POST /api/users`). Business layer (`business/users.go`) got a min-length-8 guard as secondary defense. Client-side: password strength meter updated to check all 4 character classes; user creation form validates character classes client-side; weak submissions blocked. Deployed and verified.
- **User management routes consolidated onto `adminRoute` (2026-07-13).** All 4 user management endpoints (GET /api/users, POST /api/users, DELETE /api/users/{id}, PUT /api/users/{id}/role) swapped from raw `s.JWTMiddleware` + inline role check to `s.adminRoute`, which chains `RequirePasswordChange` and `RequireRole("admin")` middleware. No more administrative bypass through un-enforced password-change checks.
- **Route-level role guards on dashboard (2026-07-13).** Added `meta: { requiresRole: 'admin' }` to 5 route definitions (federation, users, groups, alerts, credentials). The Vue router's `beforeEach` guard now checks JWT claims and redirects non-admin users. Credentials page does an additional JWT decode + admin check with redirect. Credentials nav link gated with `v-if="isAdmin"`.
- **Pagination page cap (2026-07-13).** Added `MaxPage = 10000` constant to cap page parameter. Extracted `DefaultLimit = 25` and `MaxLimit = 100` as named constants. Prevents resource-exhaustion DoS via unbounded pagination parameters.
- **PUT vs PATCH fix (2026-07-13).** `handleUpdateUserRole` was checking `MethodPatch` but the route was registered as `PUT`. Fixed the handler to check `MethodPut`. Before this fix, the endpoint always returned 405 (Method Not Allowed).

## In-progress
- Phase 4's removal is gated on the `[auth] DEPRECATED` log going quiet — see Next #1.
  Post-deploy 2026-07-24: **zero** DEPRECATED lines so far, but the only agent that has checked
  in is the local one (which `rebuild-and-restart.ps1` Step 8 re-tokenizes every deploy). The
  two remote agents have not been seen since June, so the log proves nothing about them yet.

## ⚠ REGRESSION found 2026-07-24 post-deploy — the service Environment was NEVER injected
Two "Done" entries above (`ARCVAULT_JWT_SECRET set in prod`, `Credential key moved off disk`)
describe the intended state. **Neither is true on this machine, and the installer could never
have produced it.**

Evidence from the live box:
- `HKLM\SYSTEM\CurrentControlSet\Services\arcvault-coordinator` has **no `Environment` value**
  at all. Its value names are only `Type, Start, ErrorControl, ImagePath, DisplayName,
  ObjectName, Description, FailureActions`.
- There IS an `Environment` **subkey** holding `ARCVAULT_CREDENTIAL_KEY` and
  `ARCVAULT_JWT_SECRET` as `REG_SZ` values (both 64 chars, still recoverable).
- `coordinator-service.log` contains `[config] Generated new JWTSecret` **13 times, first at
  2026-07-17 15:22**, including this deploy at 20:57:11.
- `C:\ArcVault\config.json` again contains `credential_key`, beside `arcvault.db`.

ROOT CAUSE: SCM reads exactly one thing — a **`REG_MULTI_SZ` value named `Environment` directly
ON the service key**, holding `NAME=value` strings. `set_service_environment_variable` ran
`reg add ...\Environment /v NAME /d VALUE`, which creates a *subkey* of `REG_SZ` values. That
shape is never read. So every restart minted a fresh JWT secret (sessions dropped, logout
revocation meaningless) and `loadCredentialKey()` fell back to the on-disk key.

The 2026-07-08 verification was real but was verifying a **hand-set** registry value; re-running
the installer did `sc delete` + reinstall, which dropped it, and the installer recreated only
the wrong shape.

**Correction to commit `04763b4`** ("installer persists ARCVAULT_JWT_SECRET"): that commit wired
a correct intention through this broken helper. It did not work. Fixed properly in the commit
that added this section.

FIXED (installer, for future installs):
- `set_service_environment_variable` now uses `winreg` to write a merged `REG_MULTI_SZ`
  `Environment` value on the service key, and **returns False unless it reads back** — the
  silent failure is what hid this for a week. Merging matters: a replace-style write would let
  the JWT secret evict the credential key.
- `_read_service_env` reads the correct shape AND the legacy subkey, so an upgrade **recovers**
  existing secrets instead of minting new ones. Verified against the live key: both 64-char
  values are readable. This is the data-loss guard — a fresh credential key makes every stored
  `credential_profiles` row permanently undecryptable.
- `write_coordinator_config` no longer writes `credential_key`. If the registry write fails the
  installer puts it back and warns, rather than leaving the coordinator unable to decrypt.
- `installer/windows/test_service_env.py` covers the parse/format round-trip and the merge.

**STILL BROKEN ON THIS MACHINE — needs an elevated shell.** The installer fix does not
retroactively repair the live service; the secrets must be rewritten in the correct shape from
the legacy subkey. Until then: every coordinator restart logs out every session, and the
credential key sits next to the database.

## Next
Ordered by value:

1. **Credential key sits on disk next to the ciphertext it protects.** `loadCredentialKey()` (`server/credentials.go:21`) prefers `cfg.CredentialKey` and only falls back to `credcrypto.LoadKey()`. Prod `C:\ArcVault\config.json` **has** `credential_key` (64 chars, byte-identical to the one in the dead `service-run.bat`), so the env var is never consulted and encryption works fine. **Correction: an earlier note here claimed `LoadKey()` returns `ErrKeyNotSet` in prod. That was wrong — it is never called.**

   The actual defect: the key lives in `config.json`, in the same directory as `arcvault.db`. Anyone who can read the DB can read the key, so encryption-at-rest for the 1 stored `credential_profiles` row (`cred-d077217915c0b069`, SMB, 133 bytes) is decorative. Note `config.Save()` blanks `admin_token` and `jwt_secret` on write but **not** `credential_key` (`config.go:83-86`) — inconsistent.

   Fix: move the key to the service registry `Environment` (machinery now proven by the JWT secret), delete `credential_key` from `config.json`, and blank it in `Save()` alongside the other two. Migration is safe *because the two key values are identical* — same bytes, different source, so existing ciphertext still decrypts. `credcrypto.Rekey()` (`rekey.go`) exists if the key ever needs rotating.

   **DONE 2026-07-08** — see the credential-key entry under Done. Note for whoever reads this next: `GET /api/credential-profiles/{id}` does not exist (only POST/GET-list/DELETE), and `decryptCredentials()` swallows every failure into `nil`, so a broken key silently omits credentials from an agent's job payload rather than erroring. The only observable decrypt path is an agent fetching a job bound to a profile — and **zero jobs are bound to one**. That is why verification went through POST (201-vs-503) instead.

1. **Admin-token architecture** (scoped-token redesign, phased). Trace done this session — the admin token has four jobs: (a) **master key** on every JWT route via the `JWTMiddleware` fallback (`auth.go:173-197`, injects `role:admin`); (b) static bearer on agent endpoints (`authMiddleware`: register/heartbeat) and `adminTokenRoute`/`agentOrViewerRoute`, which have their **own** `isAdminToken` checks independent of the fallback; (c) exposed to the browser via `GET /api/admin/token` → `Users.vue` "Copy Admin Token"; (d) reused as an agent token by the installer (`arcvault_installer.py:447`). **Key discovery:** the scoped per-agent path already exists — `GET /api/admin/bootstrap.ps1` (`handleBootstrapScript`) mints a fresh per-agent token via `CreateAgentToken` and embeds it; agents auth with that (`cfg.AuthToken`), not the admin token.
   - **Phase 2 DONE + DEPLOYED 2026-07-08** — removed `GET /api/admin/token` (route + `handleGetAdminToken`) and the `Users.vue` "Copy Admin Token" button/handler so the raw token is no longer fetchable by browser JS (the XSS-exfil path). Removed the endpoint, not just the button — XSS could `fetch()` it directly. API spec section deleted. Verified live: `GET /api/admin/token` → **404**.
   - **Phase 1 DONE 2026-07-08** — removed the `JWTMiddleware` admin/agent fallback (`auth.go`): the admin token is no longer a master key injecting `role:admin` on every JWT route. Replaced with an explicit allowlist for the two ops scripts' machine reads: `GET /downloads/installer` → `adminTokenRoute` (was `adminRoute`; that dead wrapper is now used), `GET /api/version` + `GET /api/agents` → new `adminTokenViewerRoute` (admin token OR viewer+ JWT). Agent endpoints (`authMiddleware`, `agentOr*`) unchanged — their own `isAdminToken` checks were never the fallback. Net: the admin token can no longer touch user management, roles, credentials, groups, federation writes, or agent delete/update. New `admin_token_allowlist_test.go` pins it (admin token → 401 on `/api/users`, 200 on `/api/agents` + `/api/version`). Fixed ~50 tests that leaned on the old master key: shared `authHeader()` now mints an admin **JWT**; new `machineAuthHeader()` (admin token) for `authMiddleware` routes (results/progress/register); `newTemplateTestServer` was missing `JWTSecret`. Two 403→401 assertion flips (agent token now 401s at `JWTMiddleware` before `RequireRole`). Full coordinator suite green except the two pre-existing tlscert/bootstrap failures. **DEPLOYED + VERIFIED LIVE 2026-07-08** (binary 18:19:19): admin token → `/api/agents` **200**, `/api/version` **200**, `/downloads/installer` **200**, `/api/users` **401** (master key gone). Caught a stale-deploy first (binary hadn't rebuilt; `/api/users` still 200) — re-ran the script, then re-verified. The Lesson holds: check the running process.
      - Aside (FIXED 2026-07-08): `rebuild-and-restart.ps1` Step 8 POSTed `/api/agent-tokens`, a route that **never existed** — silently failing into its catch every deploy. Now mints via the CLI `coordinator create-agent-token <id> --token-only` (exe resolves config.json next to itself). Parse-checked; exercised on next deploy.
   - **Phase 3 DONE 2026-07-24** — the two paths that handed out the admin token as an agent credential are closed.
     - Dashboard: new admin-only "Enroll Agent" button on the Agents view drives `GET /api/admin/bootstrap.ps1` with a hostname, which mints a `bootstrap:<hostname>` token — 1-hour expiry via `CreateAgentToken`, enforced by `ValidateToken`. `?hostname=` is validated against `[A-Za-z0-9.-]`/253 chars server-side (it is persisted as the token's `agent_id`; rejecting `:` stops a hint forging a role prefix). `bootstrap_hostname_test.go`.
     - Installer: `arcvault_installer.py` no longer does `agent_token = admin_token`. On a coordinator+agent install it now mints a real per-agent token by shelling out to `coordinator.exe create-agent-token <id> --token-only` (opens config.json + DB next to the exe; no running service or network needed, 3 retries, output validated as 64 hex chars). This forced a reorder of `do_install`: coordinator config → coordinator service → mint → agent config → agent service. Agent-only installs are unchanged — the operator pastes the token, which is now a bootstrap token rather than the admin token.
   - **Phase 4 NOT DONE — deliberately gated, not forgotten.** Removing the admin token means deleting the `isAdminToken` branch in `acceptMachineToken`, and that branch is the only thing keeping already-deployed agents authenticating. At least 3 registered agents (v0.4.0/v0.5.0/v0.6.0) plus whatever is at 192.168.68.64 predate Phase 3, so deleting it today bricks the fleet. It is a sequencing constraint, not a coding problem.
     - Enabling step DONE 2026-07-24: the four middlewares that accepted the admin token as an agent credential (`authMiddleware`, `agentOrViewerRoute`, `agentOrOperatorRoute`, `agentOrAdminRoute`) were four copies of the same block; they now share `acceptMachineToken`, which logs `[auth] DEPRECATED: <ip> authenticated with the admin token` once per client IP (deduplicated — agents heartbeat every 30s). `legacy_admin_token_test.go` covers the dedup.
     - **Removal criterion: when that line stops appearing in `coordinator-service.log`, every machine has its own token and the branch can be deleted.** Until then, re-enroll each host that shows up via "Enroll Agent". Still moves together with Next #2 (plaintext agent tokens).
2. **Plaintext agent tokens.** `tokens.token` stored raw, matched by equality (`db.go:123`). Store `sha256(token)` instead. Touches registration, installer, and every deployed agent — needs a migration path. Lower urgency: requires DB file access, which already implies compromise.
3. **Password policy** is now full complexity enforcement (8+ chars, uppercase, lowercase, digit, special). **DONE 2026-07-13** — see Done.

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
- ~~`PUT /api/users/{id}/role` is registered but `handleUpdateUserRole` requires `PATCH` → **always 405**. Dead endpoint. Fix the verb or drop the route?~~ **FIXED 2026-07-13** — handler now checks `MethodPut`.
- Role `operator` can't be created — `CreateUserRequest.Validate()` and `UpdateUserRoleRequest.Validate()` both allow only `admin`|`viewer` — yet user id 3 in prod has `operator`. How did it get set, and is `operator` supposed to be creatable?
- ~~User-management handlers (`handleListUsers` etc.) use raw `JWTMiddleware` + an inline `claims.Role != "admin"` check instead of `adminRoute`, so they skip `RequirePasswordChange`. Consolidate onto `adminRoute`?~~ **DONE 2026-07-13** — all 4 user management endpoints now use `adminRoute`.

## Lesson (do not summarize away)
`TestLogoutRevokesToken` passed while logout was **completely broken in production**. The token lives 4h and the machine's offset is −4h, so the local-time expiry string coincided with the UTC issue time; the test ran in 160ms, inside the same second, and a string prefix tie made the comparison succeed. Green by arithmetic accident.

A negative control proved the test *could* fail. It did not prove the test measured the right thing. The Go-level round trip never exercised SQLite's actual `expires_at > datetime('now')` comparison. **Assert on observable behavior through the real interface, and inspect what's actually stored.** The new `db/token_expiry_test.go` pins `time.Local` to a −4h zone (so it bites on a UTC CI box), reads raw bytes via `expires_at || ''` (defeating the driver's `DATETIME`→`time.Time` conversion), and sleeps 1.1s to force the clock past the prefix tie.

## File map
- `coordinator/server/auth.go` — JWT issue/validate, `newJTI`, `tokenRevoked`, login throttle (`loginKeyAllowed`, `clientIPKey`, `accountKey`, prune), auth handlers, `validatePasswordStrength` (NEW)
- `coordinator/server/server.go` — `bearerToken`/`isAdminToken`/`isAgentToken`, all middlewares (incl. `adminTokenRoute`, new `adminTokenViewerRoute`), `securityHeaders`, route table
- `coordinator/server/admin_token_allowlist_test.go` — NEW: admin token 401 on `/api/users`, 200 on allowlisted `/api/agents` + `/api/version`
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
- `coordinator/business/users.go` — min-length-8 check on password (NEW: secondary guard)
- `C:\ArcVault\` — live deployment (config.json, arcvault.db; service-run.bat deleted 2026-07-08)

## Pre-existing test failures — RESOLVED 2026-07-24 (`7ac0537`)
Both suites noted here as "unrelated, do not chase" were stale tests, not code defects, and are now green. The whole Go suite passes (11 packages).

- `internal/tlscert` (4 tests, `x509: malformed certificate`): they fed `ReadCertPEM`'s output straight to `x509.ParseCertificates`. `ReadCertPEM` returns **PEM**; `ParseCertificates` wants **DER**. The missing step was `pem.Decode` → `block.Bytes`. Five copies of the block collapsed into one `parseCertFile` helper. `ReadCertPEM`'s doc comment claimed it returned DER, which is what misled them — corrected, since `server/bootstrap_handler.go` reads the same function.
- `internal/bootstrap` (`TestGenerateScript_crossEdition`): asserted `PSEdition` branching, `-SkipCertificateCheck` and `ServerCertificateValidationCallback`, all from an abandoned `Invoke-WebRequest` design (PS 5.1's HttpWebRequest could not survive this server's TLS renegotiation, so the script moved to `curl.exe` and needs no edition branch). Rewritten as `TestGenerateScript_pinsTrustToEmbeddedCert`: asserts `--cacert`, `--fail`, forced TLS 1.2, cert-written-before-download ordering, and that the verification-bypass strings stay **absent**. The old test protected nothing while looking like it protected the download path.

## Also 2026-07-24 — two security findings outside the numbered phases

- **Fresh installs served cleartext HTTP on 443** (`361e10a`). `tlscert.Generate` was only called by `coordinator init` / `regen-cert`, and nothing in the install path calls either; the installer writes a `config.json` with no `cert_file`/`key_file`, so `Server.Start()` took the empty-path branch and fell through to `ListenAndServe` while the installer, dashboard and agents all addressed it as `https://`. Agent tokens and JWTs would cross the wire in the open. `tlscert.EnsureExists` already existed for exactly this and had **zero callers**; now wired into `StartCommandWithContext` (covers `start` and `run-service`, not just the GUI installer). Idempotent — regenerating would break agents pinning the old cert. Verified live on port 18443 rather than by unit test alone: generated the pair, logged HTTPS, `/health` 200 over TLS, cleartext request rejected 400. No-op on the live box, whose config already has both paths.
- **Bootstrap cert thumbprint was wrong AND dead** (`7ac0537`). Computed as sha1-over-PEM; a Windows thumbprint is sha1-over-DER (measured — PowerShell's `X509Certificate2.Thumbprint` matches the DER value). It was also never compared by the script: `grep -c PinnedThumb` was 1, its own assignment. Real pinning is `curl --cacert` against the embedded PEM, which validates the certificate rather than matching a fingerprint. Deleted rather than repaired — a wrong value that reads as load-bearing is worse than no value.

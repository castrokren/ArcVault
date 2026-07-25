# STATE — Release hygiene & dashboard version bugs

## Goal
Make ArcVault's GitHub release pipeline sane (releases published, tags truthful,
poison tags gone) and make the dashboard's version display/update flow trustworthy.

## Invariants / decisions
- The updater reads `releases/latest` on castrokren/ArcVault — a git tag alone is
  invisible; a release must be published with assets `coordinator_windows_amd64.exe`,
  `agent_windows_amd64.exe`, `SHA256SUMS`.
- Version single source of truth: `VERSION` file → ldflags via
  `scripts/rebuild-and-restart.ps1`. Never hand-build.
- Navbar version comes from `/api/version` (viewer-accessible), never hardcoded.
- Deepseek sessions ignore "do not commit" instructions — isolate them on a
  branch/worktree next time.

## Done
- Published GitHub release v0.6.0 (binaries + SHA256SUMS); `releases/latest` returns it.
- Local coordinator self-updated 0.5.1 → v0.6.0 (verified via /api/version; checksum
  path exercised — first release with SHA256SUMS).
- Dashboard bugs fixed, verified in browser against live coordinator, committed on
  `security/hardening-v0.6.0`:
  - `4890577` (Deepseek) PLAN-2: UpdateModal emits `updated`, App.vue re-fetches.
  - `e715ac0` (Claude) PLAN-1 files (getVersion export, inject defaults neutralized)
    + reconnect poller now polls /api/version until restarted coordinator reports
    target version (was a stub that always timed out in service mode).
- Tests 76/76 green (`cd dashboard; npx vitest run`), `npm run build` clean.

## Done (2026-07-17 session) — all committed on security/hardening-v0.6.0
- Kiln dashboard retheme (Phase 1 app + Phase 2 login) — commit d6c63ef.
- Agents view redesigned to "Fleet Console" (Workbench, inside Kiln) — commit ec07a86.
  Command bar + integrated fleet band + left status rail + table-as-surface + live WS
  activity rail. Honest-data: online-rate spark is placeholder (TODO real endpoint);
  no per-row heartbeat sparkline (no per-agent telemetry). Approved mockup:
  claude.ai/code/artifact/2ab07fac-d440-4338-9af4-986db6b2dd25
- Installer reskinned to Kiln (arcvault_installer.py) + root arcvault.spec — commit 3ec8ea1.
- Deploy TLS probes fixed for Windows PowerShell 5.1 (curl.exe instead of PS7-only
  -SkipCertificateCheck) in check-sanity.ps1 + rebuild-and-restart.ps1 — commit d6e7774.
- DEPLOYED via rebuild-and-restart.ps1 (kren, admin). Redesign LIVE on the HTTPS
  coordinator (verified: /health ok, serving index-CLXBuUXP.js). Sanity check now
  11/0/2. Installer .exe rebuilt (36MB, bundles the redesigned coordinator.exe).
- Resolved STATE open question: untracked `ArcVault/` was a personal Obsidian vault
  accidentally in-tree — now gitignored (with *.bak, __pycache__/).

## Done (2026-07-19 session) — installer build-dir fix + fresh installer
- BUG: `scripts/build.ps1` wrote the installer to root `dist\`, but `config.json`
  `installer_dir` serves from `installer\windows\dist`. A build never reached the served
  dir; two `dist\` dirs existed (root empty, served one held a stale 7/17 installer).
- FIX (build.ps1 + regenerated arcvault.spec): build Go binaries + PyInstaller output all
  into `installer\windows\dist`; clean step now also nukes legacy root `dist\`. One dist dir,
  matches config.json. Changed: clean/New-Item paths, both `go build -o`, spec `binaries=`,
  guardrail path, `--distpath`, `$outExe`.
- Built fresh installer via build.ps1: `installer\windows\dist\ArcVault-Setup-0.6.0-windows-amd64.exe`
  (26.7MB, 7/19), binary version guardrail verified v0.6.0. Root `dist\` confirmed gone.
- NOT committed: build.ps1 + arcvault.spec (kren's call).

## Done (2026-07-19 session) — dashboard "fast timeout / relog on tab click" ROOT CAUSE
- SYMPTOM (kren): coordinator dashboard logs you out almost immediately; clicking a nav
  tab forces a relog.
- FALSE LEAD (real bug, not the cause): installer never set `ARCVAULT_JWT_SECRET`, so
  `config.go` regenerated a random JWT secret each start → restarts invalidated tokens.
  Fixed anyway: installer now persists `ARCVAULT_JWT_SECRET` like the credential key
  (arcvault_installer.py: get_or_create_jwt_secret + set_service_environment_variable),
  set it on the live service, rebuilt the .exe. But coordinator was NOT crash-looping
  (PID stable), so this wasn't the symptom.
- REAL ROOT CAUSE (from `C:\ArcVault\coordinator-service.log`):
  `JWTMiddleware: revocation check failed: database is locked (5) (SQLITE_BUSY)` — bursts
  of 3 on tab click. The token-revocation check (auth.go:164) is fail-closed: ANY error,
  incl. a transient SQLITE_BUSY, returns 401 → dashboard handle401() wipes session → relog.
  Concurrent API calls on tab nav contended on the DB and got instant BUSY.
- WHY BUSY despite a busy_timeout in the DSN: driver is `modernc.org/sqlite`, whose DSN
  pragma syntax is `_pragma=name(value)`. The old DSN used mattn/go-sqlite3's
  `_busy_timeout=5000&_journal_mode=WAL` keys, which modernc SILENTLY IGNORES. So WAL was
  never on and busy_timeout was 0 — every concurrent touch could BUSY.
- ORIGIN (git-traced): NOT a regression. Driver has been modernc.org/sqlite since the
  initial commit (d7c115c); mattn/go-sqlite3 was never a dependency. The mattn-style DSN
  dates to that same commit → WAL + busy_timeout were NEVER in effect in the project's
  whole history. The code comment claiming they "handle all concurrency" was wrong from
  day one. This is why the fast-logout has been chronic, not new.
- FIX (coordinator/db/db.go): DSN → `?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)`.
  DIFFERENTIAL PROOF (throwaway test, both DSNs side by side): OLD → journal_mode=`delete`,
  busy_timeout=`0`, and a writer-vs-writer contention reproduces the exact
  `database is locked (5) (SQLITE_BUSY)`; NEW → `wal`/`5000`, contention waits & succeeds.
  PERMANENT GUARD: coordinator/db/dsn_test.go (TestInit_appliesWALandBusyTimeout) fails if
  the DSN param syntax ever regresses. `go build ./...`, `go test ./db/` + `./server/` green.
- DEPLOYED via rebuild-and-restart.ps1 (sanity 12/0/2, health ok, coordinator v0.6.0
  RUNNING). VERIFIED live: `C:\ArcVault\arcvault.db-wal` + `-shm` sidecars now exist
  (stamped at deploy) — the running binary is in WAL mode at last; no new SQLITE_BUSY
  since restart.
- USER-CONFIRMED (2026-07-19): kren logged in on the live dashboard and it stays up while
  navigating — the fast relog is resolved. (Did NOT run the planned 50-request concurrency
  burst; kren's own click-through was enough.) Regression guard: coordinator/db/dsn_test.go.
- COMMITTED 2026-07-24 as `45e376d` (db.go + dsn_test.go) and `04763b4` (the installer's
  ARCVAULT_JWT_SECRET persistence).
- SEPARATE ISSUE spotted in the log (not the relog bug): a steady flood of
  `http: TLS handshake error from 192.168.68.64:*: remote error: tls: bad certificate`
  every ~10-30s — a LAN client/agent that doesn't trust the self-signed cert. Noise for now.

## Done (2026-07-19 session, cont'd) — remote agent version-staleness detection
- KREN ASK: coordinator can't detect when (esp. remote) agents are out of date. Worked the 3
  queued steps end to end; "actually test the full stack, no fake tests."
- STEP 1 (version transmission) — CONFIRMED WORKING, no code change. Agent POSTs os/arch/version
  on register (`agent/heartbeat/heartbeat.go:90`); coordinator upserts it (`db/agents.go:17`,
  `version=excluded.version`). Heartbeat carries no version by design; self-update restarts →
  re-register refreshes it. Live proof: `/api/agents` returned 3 real agents at v0.4.0 / v0.5.0 /
  v0.6.0.
- STEP 2 (comparison logic) — FOUND + FIXED 2 real bugs:
  - Frontend (dashboard/src/views/Agents.vue + utils/format.js): `updateAvailable` compared
    `agent.version !== updateStore.current` (coordinator running version, `!=` not semver-behind),
    and ALL drift UI was gated behind `!selectedSite` so federated/remote agents showed NO drift.
    Fixed: new `versionBehind(a,b)` semver helper (strictly-older, fail-safe on missing); baseline =
    `updateStore.latest || updateStore.current` (latest release = what a push-update installs; falls
    back to /api/version for non-admins who can't hit admin-only /api/update/check); read-only drift
    badge now shows for remote agents too (labeled "stale"); Update button + modal stay local-only
    (can't push-update another coordinator's agents). Unit test: utils/format.test.js (3 cases).
  - Backend (coordinator/server, THE remote-staleness root cause): `FedAgentHeartbeat` had a
    `Version` field and `federation_hub.go:159` wrote it into the root's SubCache every heartbeat —
    but `handleHeartbeat` never populated it, so each sub-agent heartbeat (30s) BLANKED the remote
    agent's version. Fixed: dropped the field + the clobber; version now comes only from
    register/snapshot deltas. Regression guard: `TestFedHub_ApplyDelta_AgentHeartbeat` now asserts
    version is PRESERVED across a heartbeat.
- STEP 3 (192.168.68.64 TLS flood) — INVESTIGATED, operational not code. `tls: bad certificate`
  every ~15-30s = a 4th remote host's agent whose CA bundle doesn't trust this coordinator's cert;
  it never completes TLS → never registers → has no version at all. Fix is cert distribution
  (`CACertFile` on that agent), not a version bug. (`[::1]` cert errors are just local browser/curl.)
- DOCS: docs/backend.md gained an "Agent version / out-of-date detection" prose section (version
  source of truth, latest-release baseline, federated caveat, never-connected case). Contract test
  `internal/docs` green.
- TESTED (not faked): vitest 80/80; `go test ./coordinator/server ./coordinator/db ./internal/docs`
  all green; full-stack rebuild-and-restart deployed (coordinator v0.6.0 RUNNING, sanity 12/0/2);
  live `/api/agents` verified real versions. REMAINING: visual browser confirm of the drift badges
  needs kren to log in (no creds on file; login screen reached, can't proceed further).
- COMMITTED 2026-07-24 as `4706ef9` (Agents.vue, format.js + format.test.js, backend.md,
  federation_messages.go, federation_hub.go, federation_test.go).

## In-progress
- **Correction 2026-07-25 (later session): the 09:07 deploy is now stale.** `0299671`
  ("stop the service before replacing agent.exe" — makes bootstrap safe to re-run on a
  machine that already has an agent) was committed at **09:43:43**, 36 minutes AFTER the
  coordinator.exe on disk (built 09:07:48). The live coordinator is still serving the
  OLD bootstrap script template. See the dated section below — needs a redeploy.
- Fleet as of 09:49 07-25 (may be stale per above): `DESKTOP-EE77F38` v0.6.0 online,
  `SRB3FLPC010` v0.5.0 online (freshly enrolled), `SMILOW3FLSP001` still offline
  since 06-11 and never re-enrolled.
- Left uncommitted (kren's call): `.agents/` + `skills-lock.json` (hallmark skill
  toolchain), `tasks/security-hardening/PLAN-review-fixes.md` (unrelated prior task).
- BLOCKED (needs kren): `gh release upload v0.6.0 installer/windows/dist/ArcVault-Setup-0.6.0-windows-amd64.exe --clobber`
  — permission classifier blocks the outward upload. Run it via `!` in the prompt.
  Rebuild the .exe first: the one on disk (07-19) predates every token/cert change.

## Done (2026-07-25 later session) — dashboard "Get Token" removal + Update-button root-cause
- **Dashboard "Get Token" button removed.** Per-agent token minting from the UI was the
  trigger for the mint-revokes-live-agent regression fixed earlier today (`984dd1d`), and
  duplicated the safer "Enroll Agent" bootstrap flow. Removed `AgentTokenModal.vue`, its
  wiring in `Agents.vue` (`tokenModalOpen`/`selectedAgentForToken`/`openTokenModal`), and
  the dead `createAgentToken` export from `api.ts`. Backend route
  (`POST /api/agents/{id}/token`) is untouched — still used by bootstrap. CONTRACT block in
  `dashboard/docs/frontend.md` updated. Also logged in `tasks/security-hardening/STATE.md`.
- **"Update agent" button investigated end to end — not a dashboard bug, three real findings:**
  1. **The mechanism itself works.** Read the local agent's own log
     (`C:\ArcVault-Agent\logs\arcvault-agent.log`): its WS thread 401'd from 07:42–07:45
     (caught in the crossfire of the token-mint-revokes-agent bug fixed today), then after
     the 09:07 redeploy re-enrolled with a fresh token and has held `Agent WS: connected to
     coordinator` continuously since — over an hour, no drops. `hub.SendToAgent` /
     `handleAgentWS` are sound.
  2. **`SRB3FLPC010`'s 404 ("agent not connected") is real**: its heartbeat (HTTP) is fine
     but its WS thread (the actual command channel for push-updates) isn't registered in the
     coordinator's `agentConns`. It's on a pre-fix v0.5.0 binary from before today's
     token/enrollment work; a re-enroll (bootstrap re-run) was the agreed next step to get it
     current, matching how the local agent recovered.
  3. **The re-enroll attempt uncovered a stale deploy.** `GET /api/admin/bootstrap.ps1`
     refused with 409 (`host` unset in `C:\ArcVault\config.json` — the existing
     hardening from earlier today working as designed). After a retry that got further, the
     download step failed with curl exit 23 ("client returned ERROR on write") — this is
     the exact failure mode `0299671`'s comment describes for a machine that already has a
     running agent. **`0299671` was committed at 09:43:43, but the live `coordinator.exe`
     was built at 09:07:48 — 36 minutes earlier.** The running coordinator is still serving
     the pre-fix bootstrap template that writes straight over the locked, live `agent.exe`
     instead of `agent.exe.new`. Confirmed by comparing `Get-Item coordinator.exe
     .LastWriteTime` against `git log -1 --format=%ci 0299671`.
  - **Cert/host finding**: the live cert's SAN list is `localhost`, `DESKTOP-EE77F38`,
    `192.168.68.62`, `127.0.0.1` — but this machine's actual current Wi-Fi IP is
    `192.168.68.58` (the `.62` mismatch was already flagged earlier today, session note
    "Certificate issued for wrong IP"). Recommended `host` value: **`DESKTOP-EE77F38`** — it's
    already in the SAN (no cert regen needed, which would otherwise require hand-copying a
    new cert to every already-enrolled agent's `coordinator.crt`), and it's immune to future
    DHCP churn on `.58`.

### Next (for whoever picks this up — kren is doing the actual deploy outside this session)
1. Set `"host": "DESKTOP-EE77F38"` in `C:\ArcVault\config.json`.
2. Redeploy via `.\scripts\rebuild-and-restart.ps1` (picks up `0299671` + everything through
   `f5f8878`; also makes the coordinator advertise the corrected host in bootstrap URLs).
3. Re-run **Enroll Agent** for `SRB3FLPC010`, download the fresh script, run it there
   (elevated PowerShell, `Unblock-File` first — double-clicking a `.ps1` just opens an editor).
4. Confirm via that machine's `logs\arcvault-agent.log`: look for `Agent WS: connected to
   coordinator` with no repeated disconnects, then retry the dashboard Update button.
5. Once confirmed working, re-enroll `SMILOW3FLSP001` the same way (still offline since 06-11).

## Verification debts — what is NOT proven
- `SMILOW3FLSP001` has never been re-enrolled. It has no per-agent token row, so it
  is presumably still on the admin token or a June `bootstrap` token. Re-enroll it
  via Enroll Agent to find out, and watch for `[auth] DEPRECATED` in
  `coordinator-service.log` when it next checks in.
- The dashboard UI itself is still unverified by eye — Enroll Agent and the
  stale/update version badges have only been exercised over the API, never clicked.
  (No dashboard credentials on file.)
- The installer's GUI path (`arcvault_installer.py`) has NOT been run since the
  Phase-3 token minting, cert wait and registry-shape changes. Needs a scratch-box
  test of BOTH a coordinator+agent install and an agent-only install.
- `scripts/repair-service-env.ps1` has never been run. See
  `tasks/security-hardening/STATE.md` — the live service still has no `Environment`
  value, so the coordinator mints a fresh JWT secret on every start and the
  credential key still sits beside the DB.

## Done (2026-07-17 session, cont'd) — agent push-update fixes
- KREN ASK #2 RESOLVED: coordinator→agent push-update pipeline already fully exists (HTTP trigger
  → WS push → download/verify/stage/swap/restart → progress relay → dashboard UI). Two bugs
  prevented it from working:
  - Bug 1 (auth regression): Agent WS auth query-param mismatch after hardening (05b5736). Fixed
    by moving token to Authorization header in agent/ws/ws.go; dropped dead query param.
  - Bug 2 (Windows self-kill race): ApplyUpdate calls 'net stop' from inside the dying process;
    SCM handler os.Exit(0)s before rename/restart execute. Fixed by deferring binary swap +
    restart to detached helper script; fire-and-forget the helper, then call 'net stop'.
  - Commit 2c462c2: Both fixes (agent/ws/ws.go + agent/updater/updater_windows.go).
  - Tests: all 76 agent/updater tests pass; both packages compile clean.
  - Next: manual end-to-end test against real agent service to confirm /ws/agent connects
    and update_progress completes without timeout.

## Done (2026-07-17 session, cont'd) — service error 1067 on other machines
- KREN ASK: agent/coordinator services fail with Windows error 1067 after install.
  Root causes (agent = the "other machines" service):
  - Agent `runAgent()` called `log.Fatalf` on `heartbeat.Register` failure. On a remote
    box the coordinator is often unreachable at service-boot (firewall/boot-order/TLS pin;
    real log: `dial tcp [::1]:443 refused`) → process dies during StartPending → SCM 1067.
    Fix (agent/main.go): registration now retries in a background goroutine, never fatal.
  - `heartbeat.Start` had a Fatalf-from-goroutine on TLS-config failure → also nuked the
    process. Fix (agent/heartbeat/heartbeat.go): log + return (disable heartbeat), don't exit.
  - Deploy-script bug: `rebuild-and-restart.ps1` Step 8 used `-replace '...(auth_token:...)' ('$1'+$token)`;
    a hex token starting with a digit reads as backreference `$1<digit>`, dropping the key and
    writing `$12be8...` → invalid YAML → agent 1067 on config-load. Fixed: no capture group,
    `-replace '^\s*auth_token:.*$', "auth_token: $token"` + `.Trim()`.
  - Coordinator visibility: `run-service` now redirects the logger to
    `coordinator-service.log` beside the exe (agent already had `logs/arcvault-agent.log`).
    Under SCM there's no console, so 1067 previously died with zero trace.
  - Boot-order fix: heartbeat.Start now runs INSIDE the registration goroutine, AFTER the
    first successful Register — you can't heartbeat an agent the coordinator hasn't registered
    yet (it 404s), and racing the token read 401s. Ordering register→heartbeat removes the blip.
  - DUG the transient: the "~2min 400-register window" seen on the first deploy does NOT
    reproduce — it was a one-time artifact of TWO create-agent-token mints seconds apart (manual
    + Step 8) while the coordinator was mid-restart. Confirmed by clean restart. The remaining
    per-boot blip was one 404/401 cycle from heartbeat firing before registration; the boot-order
    fix eliminated it.
  - VERIFIED via rebuild-and-restart.ps1: both services RUNNING, sanity 13/0/0, CLEAN agent boot
    (WS connected → Registered → Heartbeat OK, no 404/401). Agent online in /api/agents. NOT committed.

## Done (2026-07-24 session) — 8 commits on security/hardening-v0.6.0 (all now DEPLOYED 07-25)
Started by committing the backlog above, then followed the admin-token thread it exposed.

- `45e376d` SQLite pragma DSN fix + `db/dsn_test.go` (see the 2026-07-19 entry above).
- `4706ef9` version-staleness detection (see the 2026-07-19 entry above).
- `04763b4` installer persists `ARCVAULT_JWT_SECRET` to the service registry Environment.
- `2ad940a` **Enroll Agent** flow. Replaced an uncommitted "reveal the admin token" button
  (which would have undone security-hardening Phase 2, deliberately removed 2026-07-08 as an
  XSS-exfil path) with `GET /api/admin/bootstrap.ps1` + a hostname prompt → a
  `bootstrap:<hostname>` token with a 1-hour expiry. `?hostname=` validated to `[A-Za-z0-9.-]`
  / 253 chars, since it is persisted as the token's `agent_id` and a `:` could forge a role
  prefix. Restored the per-agent `AgentTokenModal` the admin-token work had deleted.
- `4ac5ce5` **security-hardening Phase 3.** Installer no longer does
  `agent_token = admin_token`; co-installs mint a real per-agent token via
  `coordinator.exe create-agent-token`. Forced an ordering change in `do_install`. Phase 4's
  removal is gated on a new once-per-IP `[auth] DEPRECATED` log — details and the removal
  criterion are in `tasks/security-hardening/STATE.md`.
- `361e10a` **Fresh installs served cleartext HTTP on 443.** `tlscert.Generate` was only
  called by `coordinator init`/`regen-cert`, which nothing in the install path calls, so
  `Server.Start()` saw empty cert paths and fell through to `ListenAndServe` while the
  installer, dashboard and agents all used `https://`. `tlscert.EnsureExists` already existed
  for this with **zero callers**; now wired into `StartCommandWithContext`. Idempotent, fatal
  only in production. Verified live on port 18443: generated the pair, logged HTTPS, `/health`
  200 over TLS, cleartext request rejected 400. No-op on kren's box (cert paths already set).
- `7ac0537` Deleted the bootstrap cert thumbprint. It was computed as sha1-over-PEM (Windows
  thumbprints are sha1-over-DER — measured: PowerShell agrees with the DER value) **and** was
  never compared by the script; real pinning is `curl --cacert` against the embedded PEM.
  Also repaired two stale test suites that were already red on this branch: `tlscert` (4 tests
  fed PEM to a DER parser) and `bootstrap`'s `crossEdition` test (asserted an abandoned
  `Invoke-WebRequest` design; rewritten to assert `--cacert`/`--fail`/TLS 1.2/cert-first
  ordering, and that verification-bypass strings stay ABSENT).
- `9a2bdd3` fallow audit of `dashboard/`: fixed a `vi.mock` path that resolved nowhere and was
  therefore a silent no-op (`Login.test.js` mocked `../../composables/useAuth.js`; proven both
  ways with a throwing probe), and removed `motion-v` — in `dependencies` with zero source
  imports, only two dead `vi.mock` blocks. −128 lines.

- TEST STATUS: whole Go suite green for the first time on this branch (11 packages);
  vitest 80/80; `npm run build` clean. `-race` unavailable locally (no gcc).
- RESOLVED from the 2026-07-19 list: remote agent version-staleness (`4706ef9`). The
  192.168.68.64 `tls: bad certificate` flood is **still open** and is NOT the fresh-install
  cert bug — a coordinator serving plain HTTP cannot emit that client-side TLS alert.

## Done (2026-07-25 session) — DEPLOYED, plus two regressions of my own caught in prod

DEPLOY: `rebuild-and-restart.ps1` ran 07-25 09:07. The Step 9 agent check now uses
curl.exe (`Invoke-RestMethod` on PS 5.1 cannot survive this server's TLS and printed
"Could not query agents API" on every deploy while the agent was fine).

### Corrections to earlier claims in this file — read these before trusting the rest
- **The "restarts invalidate every session" line was overstated.** Measured: 13 startups,
  13 `Generated new JWTSecret`, 0 `loaded from ARCVAULT_JWT_SECRET` — the secret really does
  rotate every start. But token TTL is 4h (`auth.go:81`), the restarts are mostly deploys,
  and kren reports no logout problem. Hygiene, not an incident. The 07-19 entry above already
  flagged this theory as a FALSE LEAD; it got restated as fact anyway.
- **`192.168.68.58` is THIS machine's Wi-Fi IP**, not a remote host. Audit-log entries from it
  are kren's own browser over LAN IP. Do not read them as remote-agent traffic.
- **The credential-key-on-disk finding is real but tiny:** it protects exactly one row,
  `cred-d077217915c0b069` ("testng credentials ", SMB), with 0 jobs bound to it.

### `fda901f` — enrollment tokens are exchanged at registration
`bootstrap.ps1` writes the enrollment token into `agent-config.yaml` as the agent's permanent
`auth_token`, but those tokens expire in 1h and `handleRegister` returned no replacement, so
every machine enrolled by script died after an hour. Registration now returns a per-agent
`token` (additive field); the agent adopts it and rewrites only the `auth_token` line.
Consumers read through `config.TokenStore` at request time — copying the string at
construction had left the job runner and WS client on the dead enrollment token.

### `2247dd5` then `984dd1d` — a regression shipped, then fixed
`2247dd5` made `CreateAgentToken` delete the agent's other tokens. `POST /api/agents/{id}/token`
is the dashboard "Get Token" button — a read — and nothing writes that token into the running
agent's config. Clicking it revoked the live credential:

    11:42:01  POST /api/agents/DESKTOP-EE77F38/token      200
    11:42:08  POST /api/agents/DESKTOP-EE77F38/heartbeat  401   <- and every one after

Minting is additive again; cleanup moved to `handleRegister` via
`SupersedeAgentTokens(agentID, keepToken)`, which runs only after the agent has proven which
token it holds. No-op on empty keepToken, so an admin-token registration prunes nothing rather
than guessing.

### `984dd1d` — enrollment scripts pointed at the literal string `https://`
`host` is optional in config.json and the installer never writes it, so
`fmt.Sprintf("https://%s", cfg.Host)` produced a bare scheme and every curl in the script had
nowhere to go. New `coordinatorBaseURL(cfgHost, cfgPort, requestHost)` falls back to the Host
header and **refuses with 409** on a loopback result rather than emitting an unusable script.
URL resolution also moved ahead of the cert read and token mint, so a refused request no longer
leaves an orphan enrollment token.

### `0299716` — the script could never re-run on a machine that already had an agent
curl wrote straight onto `agent.exe`; Windows locks a running executable and the service was
only stopped further down, so curl aborted first:

    curl: (23) client returned ERROR on write of 14083 bytes

That is why `SRB3FLPC010` sat on its stale June config reporting
`x509: certificate signed by unknown authority` — the script threw before the config write, so
its `ca_cert_file` still pointed at a pre-current cert. Now: download to `agent.exe.new`, verify
the hash on the temp copy, stop+delete the service, swap, then write config and install. A
failed download leaves a working agent untouched. curl exit codes are explained in the error
text (23 = cannot write, 60 = untrusted cert, 22 = HTTP error).

### `7ac0537` / `9a2bdd3` / `db48c07` — smaller items
- Deleted the bootstrap cert thumbprint: computed as sha1-over-PEM (a Windows thumbprint is
  sha1-over-DER; measured, PowerShell agrees with DER) **and** never compared by the script.
  Real pinning is `curl --cacert` against the embedded PEM.
- Repaired two stale suites that were already red: `tlscert` (fed PEM to a DER parser) and
  `bootstrap`'s `crossEdition` test (asserted an abandoned Invoke-WebRequest design).
- fallow audit of `dashboard/`: fixed a `vi.mock` path that resolved nowhere and was therefore
  a silent no-op, and removed `motion-v` (zero source imports). `db48c07` then made that mock
  *correct* — its `login` resolved `undefined` while `handleSubmit` reads `result.success`.

### PROVEN ON THE REAL FLEET (not just tests)
- `SRB3FLPC010` installed via bootstrap.ps1, registered **201**, exchanged its enrollment
  token, heartbeats **200** from 192.168.68.64. Coordinator logged:
  `[register] SRB3FLPC010: exchanged enrollment token "bootstrap:SRB3FLPC010" for a per-agent token`
- Sandbox e2e with real binaries: clicking Get Token against a running agent no longer kills
  it; an agent restart triggers `removed 1 superseded token(s)`; the old pre-fix binary 401s
  under the same conditions where the new one survives (9 vs 0 auth failures).
- `host` was set manually by kren; the working URL before that came from the Host-header
  fallback.

## Next
1. **Re-enroll `SMILOW3FLSP001`** (offline since 06-11). Enroll Agent, then run the script
   elevated on that box. It has no per-agent token row, so it is the last machine that could
   still be on the admin token.
2. **Run `scripts/repair-service-env.ps1`** (elevated). The live service has no `Environment`
   registry value, so `ARCVAULT_JWT_SECRET` / `ARCVAULT_CREDENTIAL_KEY` are never injected.
   Full diagnosis and the two-pass procedure are in `tasks/security-hardening/STATE.md`.
3. **Rebuild the installer .exe** — the one on disk is 07-19 and predates the per-agent token
   minting, the registry-shape fix and the cert handling. Then `gh release upload ... --clobber`
   (needs kren; the permission classifier blocks the outward upload — run it via `!`).
4. **Prune the 26 legacy `bootstrap` tokens** once every machine has been re-enrolled. Left in
   place deliberately: `SMILOW3FLSP001` has no per-agent token, so one of them may be its only
   credential. Per-agent duplicates now clear themselves at registration.
5. `tasks/release-hygiene/PLAN.md` steps 2-5: merge to main, re-point the v0.6.0 tag (currently
   on orphaned commit 637c33b), delete poison tags (`v5.01`, the fake v0.7.0-v1.2.0), add
   `scripts/publish-release.ps1`.
6. KREN ASK (07-17) — DOWNLOAD INSTALLER BUTTON. `installer_dir` is still absent from
   `C:\ArcVault\config.json`, so `/downloads/installer` 404s. It must point at
   `installer\windows\dist` (per `bf4a272`); the older note in this file said root `dist\`,
   which that commit deleted.
7. AGENT TOKEN GENERATOR — **superseded for new machines**, which go through Enroll Agent.
   `POST /api/agents/{id}/token` + `AgentTokenModal.vue` remain for re-tokenizing an agent
   already in the fleet (it 404s on one that is not). Still untested, but it no longer
   disturbs a running agent (see `984dd1d`).
8. Update agents to v0.6.0 — `SRB3FLPC010` is on v0.5.0. Per-agent update button; agents never
   auto-update.

## Open questions
- Should the stale v1.x releases be deleted or archived somewhere first? (destructive —
  needs kren's explicit go-ahead)
- `ensureTLSMaterial` failure is **fatal in production** (`361e10a`) on the reasoning that
  cleartext agent tokens are worse than a refusal to start. That is a new route to a
  service-start failure, on a project that already spent a session on Windows error 1067.
  Downgrade to a warning? Kren's call.
- Dashboard: delete `Sparkline.vue` and `orbit/OrbitField.vue`, or keep them as intentional
  placeholders? `Sparkline` looks like a leftover from the Fleet Console decision to ship
  without per-agent sparklines. Deleting `OrbitField` also means dropping its now-dead stub in
  `Login.test.js:52`. Related: the page-header block is duplicated 4× across views.
- `dashboard/vite.config.js` `emptyOutDir` has been `false` since the initial commit with no
  recorded reason. An uncommitted change flipping it to `true` was reverted this session
  because nothing explains the original choice and no test covers it. Which is correct?

## File map
- tasks/release-hygiene/PLAN.md — 5-step release cleanup plan (step 1 done)
- tasks/dashboard-version-bugs/PLAN-1-version-source.md — done (commit e715ac0)
- tasks/dashboard-version-bugs/PLAN-2-recheck-after-update.md — done (4890577 + e715ac0)
- coordinator/updater/updater.go — release check/download/checksum logic
- dashboard/src/components/UpdateModal.vue — update flow incl. reconnect poller
- **docs/backend.md** — Go backend architecture + test-enforced route inventory (was FUNCTIONS.md)
- **docs/frontend.md** — Vue dashboard architecture + test-enforced route/view inventory (was FEATURES.md)
- **docs/service.md** — Windows service behaviour (install, run-service, error 1067, logs)
- **internal/docs/doc_test.go** + **dashboard/src/docs/frontend.doc.test.js** — doc-drift tests
- **scripts/git-hooks/pre-commit** + **scripts/install-hooks.ps1** — block commits that drift the docs
- ~/.claude/plans/agent-token-generator.md — implementation plan for token generator feature

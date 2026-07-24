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
- NOT committed yet (kren's call): coordinator/db/db.go, coordinator/db/dsn_test.go,
  installer/windows/arcvault_installer.py, and this STATE.md.
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
- NOT committed yet (kren's call): Agents.vue, utils/format.js, utils/format.test.js, backend.md,
  federation_messages.go, federation_hub.go, federation_test.go, this STATE.md.

## In-progress
- BLOCKED (needs kren): `gh release upload v0.6.0 installer/windows/dist/ArcVault-Setup-0.6.0-windows-amd64.exe --clobber`
  — fresh installer built locally (7/19); permission classifier blocks the outward upload.
  Run it via `!` in the prompt. (Installer isn't fetched by the auto-updater — safe to clobber.)
- Left uncommitted (kren's call): `.agents/` + `skills-lock.json` (hallmark skill toolchain),
  `tasks/security-hardening/PLAN-review-fixes.md` (unrelated prior task).

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

## Next chat — NEW ISSUE (kren, 2026-07-19): agent "out of date" detection for other machines
- SYMPTOM: dashboard/coordinator cannot detect that agents on OTHER machines are out of
  date. The local agent shows fine; remote agents' version-staleness isn't surfaced.
- NOT yet investigated. Starting points to check next session:
  - Agents report `version` on register/heartbeat (business/agents.go, agent/heartbeat).
    Confirm remote agents actually send a version and it's stored/updated per heartbeat.
  - How the UI decides "update available" per agent (Agents.vue + /api/agents payload):
    does it compare agent.version against the latest release / coordinator version, or
    only against the local agent? Likely compares wrong baseline or version is empty/stale.
  - The TLS "bad certificate" flood from 192.168.68.64 (below) may be the SAME remote agent
    failing to connect at all — if it can't reach the coordinator, its version never updates,
    so it can't be flagged out-of-date. Check if that IP is the "other" agent first.
  - Agents never auto-update (per-agent update button only); staleness detection is what
    tells the operator to click it.

## Next
- Commit the 1067 fixes (agent/main.go, agent/heartbeat/heartbeat.go, coordinator/main.go,
  scripts/rebuild-and-restart.ps1).
- KREN ASK (2026-07-17) #1 — DOWNLOAD INSTALLER BUTTON. Fixed in config.json:
  set `installer_dir` to `C:\Projects\ArcVault2.0\dist` (where the built
  ArcVault-Setup-0.6.0-windows-amd64.exe lives). Verify after coordinator restart.
- KREN ASK (2026-07-17) #3 — AGENT TOKEN GENERATOR (restoration from pre-5.0).
  Missing UI feature: 'Get Token' button in Agents view was removed. Operators
  need to generate tokens for new machine installations. Skeleton code done
  (commit 46ffa2f): POST /api/agents/{id}/token endpoint + AgentTokenModal.vue +
  button in Agents.vue. Both builds clean (coordinator, dashboard). Plan at
  ~/.claude/plans/agent-token-generator.md. Next: write tests, then manual e2e.
- Update local agents to v0.6.0 (per-agent update button on Agents page — they never
  auto-update).
- Ship the dashboard fixes to users: the coordinator self-update pulls the binary from
  the GitHub release, so the redesign only reaches other installs via `gh release upload
  v0.6.0 --clobber` of a fresh coordinator, or a v0.6.1 cut.
- `tasks/release-hygiene/PLAN.md` steps 2–5: merge branch to main, re-point v0.6.0
  tag (currently on orphaned commit 637c33b), delete poison tags (`v5.01`!) and stale
  v1.x releases, add `scripts/publish-release.ps1`.
- Minor leftover: navbar `v-if` guard for empty version pill (cosmetic flash, skipped
  by Deepseek); `GET /api/agents` returned a one-off 500 during verification — glance
  at coordinator log if it recurs.

## Open questions
- Should the stale v1.x releases be deleted or archived somewhere first? (destructive —
  needs kren's explicit go-ahead)

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

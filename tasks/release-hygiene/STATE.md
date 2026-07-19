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

## In-progress
- BLOCKED (needs kren): `gh release upload v0.6.0 installer/windows/dist/ArcVault-Setup-0.6.0-windows-amd64.exe --clobber`
  — new Kiln installer built locally; permission classifier blocked the outward upload.
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

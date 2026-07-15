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

## In-progress
- (nothing)

## Next
- Update local agents to v0.6.0 (per-agent update button on Agents page — they never
  auto-update).
- Ship the dashboard fixes: rebuild + either `gh release upload v0.6.0 --clobber` or
  cut v0.6.1 (fixes only reach users inside a coordinator binary).
- `tasks/release-hygiene/PLAN.md` steps 2–5: merge branch to main, re-point v0.6.0
  tag (currently on orphaned commit 637c33b), delete poison tags (`v5.01`!) and stale
  v1.x releases, add `scripts/publish-release.ps1`.
- Minor leftover: navbar `v-if` guard for empty version pill (cosmetic flash, skipped
  by Deepseek); `GET /api/agents` returned a one-off 500 during verification — glance
  at coordinator log if it recurs.

## Open questions
- Should the stale v1.x releases be deleted or archived somewhere first? (destructive —
  needs kren's explicit go-ahead)
- What is the untracked `ArcVault/` dir and the `arcvault.spec` working-tree edit?
  Resolve before merging the branch.

## File map
- tasks/release-hygiene/PLAN.md — 5-step release cleanup plan (step 1 done)
- tasks/dashboard-version-bugs/PLAN-1-version-source.md — done (commit e715ac0)
- tasks/dashboard-version-bugs/PLAN-2-recheck-after-update.md — done (4890577 + e715ac0)
- coordinator/updater/updater.go — release check/download/checksum logic
- dashboard/src/components/UpdateModal.vue — update flow incl. reconnect poller

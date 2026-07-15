# Release Hygiene Plan

Context (2026-07-14): v0.6.0 release published with `coordinator_windows_amd64.exe`,
`agent_windows_amd64.exe`, `SHA256SUMS`. The 0.5.1 coordinator now detects it via
`releases/latest`. Remaining cleanup below, in order.

## 1. Upgrade the local install to 0.6.0 — ✅ DONE 2026-07-15

- Coordinator self-updated 0.5.1 → v0.6.0 (verified via /api/version).
- Fallout: dashboard showed stale version + agents looked up-to-date; root-caused and
  fixed in commits `4890577`/`e715ac0` (see tasks/dashboard-version-bugs/ and STATE.md).
- Still open from this step: agents not yet updated — trigger per-agent from the
  Agents page (they never auto-update).

## 2. Finish and merge `security/hardening-v0.6.0`

- Work through `tasks/security-hardening/PLAN-review-fixes.md` (untracked — commit it).
- Decide what `ArcVault/` (untracked dir in repo root) and the `arcvault.spec` edit are;
  commit or discard before merging.
- PR `security/hardening-v0.6.0` → `main`.
- If review fixes change shipped code, rebuild assets and `gh release upload v0.6.0 --clobber`
  (same tag, replace binaries + SHA256SUMS) — or cut v0.6.1.

## 3. Re-point the v0.6.0 tag (after merge)

The tag currently points at orphaned commit `637c33b` ("phase 13") from a rewritten
history — the release's source snapshot is wrong, assets are right.

```powershell
git tag -f v0.6.0 <merge-commit>
git push origin v0.6.0 --force
```

Release assets are keyed to the tag *name*, so they survive the re-point.

## 4. Delete poison tags and stale releases

- `v5.01` tag: **must go** — if anything ever publishes a release from it, no 0.x/1.x
  version will ever register as an update again.
- Stale published releases `v1.0.0`, `v1.0.1`, `v1.1.0` (+ tags, + `v1.2.0` tag):
  delete — they postdate the version reset and confuse `releases/latest` ordering.
- Orphaned tags from the rewritten history (`v0.3.0`, `v0.4.0`, `v0.7.0`, `v0.8.0`,
  `v0.9.0`, `v0.2.4` — none has a release, most point at unreachable commits): delete.

```powershell
gh release delete v1.1.0 -y; gh release delete v1.0.1 -y; gh release delete v1.0.0 -y
git push origin --delete v5.01 v1.2.0 v1.1.0 v1.0.1 v1.0.0 v0.9.0 v0.8.0 v0.7.0 v0.4.0 v0.3.0 v0.2.4
```

Keep: `v0.1.0`–`v0.2.3` (real history), `v0.6.0`.

## 5. Script the release step

One `scripts/publish-release.ps1`: read `VERSION`, run check-version-sync, build
dashboard + both binaries with ldflags, verify `--version`, write SHA256SUMS,
`gh release create $Version` with the three assets. It's exactly the sequence used
for v0.6.0 — codify it so the "tag exists but no release" failure can't recur.
Retire `scripts/upload_v1_assets.ps1` (targets the dead v1.x scheme).

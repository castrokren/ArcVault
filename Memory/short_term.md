---
name: ArcVault Short-Term Memory
category: memory
priority: medium
last_updated: 2026-05-28
last_accessed: 2026-05-28
stale_after_days: 7
auto_summarize: true
archive_policy: archive
---

# ArcVault Short-Term Memory

Active session context. Clear this after each major task completes. Promote anything worth keeping to `decisions.md`, `lessons_learned.md`, or `patterns.md`.

---

## Last Session (2026-05-28) — Schedule Builder UI + v1.0.2/v1.0.3 (COMPLETE)

**Work completed:**
- v1.0.2: Delete agents (`DELETE /api/agents/{id}`, confirmation modal, 6 new tests)
- v1.0.3: `ScheduleBuilder.vue` — Off/Interval/Daily/Weekly/Monthly/Custom modes, live preview, wired into Jobs + Templates
- Fixed truncated working-tree files (api.js, Agents.vue, Jobs.vue, Templates.vue) — restored from `git show HEAD:`
- Confirmed `coordinator/static/dist/` is go:embed source — must sync from `dashboard/dist/` before `go build`; `rebuild-and-restart.ps1` already does this (Step 3)
- All Go tests passing, dashboard build clean (71 modules)

**Next session:**
- Robocopy/rsync flags: `flags` column on jobs table → API passthrough → agent execution → multi-select UI in Jobs + Templates forms

---

## Previous Session (2026-05-27) — Release v1.0.1 + Installer Fix (PARTIALLY COMPLETE)

**Pending from that session:**
- v1.0.1 GitHub release + `ArcVault-Installer.exe` upload still outstanding

---

## Older Session (2026-05-27) — Release v1.0.1 + Installer Fix (IN PROGRESS)

**Work completed:**
- Frontend redesign shipped to production (services redeployed, verified)
- `scripts/rebuild-and-restart.ps1` ProjectRoot path fixed
- Committed + pushed: `feat: comprehensive frontend redesign, agent/coordinator updates, rebuild script fix`
- v1.0.1 release process started — tag pushed, `gh release create` + `upload_v1_assets.ps1` pending
- WinError 32 bug found and fixed in `deployment/arcvault_installer.py` — service is now stopped before download in both `_install_coordinator` and `_install_agent`

**STOPPED HERE — needs to be completed next session:**
1. Rebuild `ArcVault-Installer.exe` with the WinError 32 fix:
   ```
   cd C:\Projects\ArcVault2.0\deployment
   build_exe.bat
   ```
2. Create the v1.0.1 GitHub release and upload all assets:
   ```
   git tag v1.0.1
   git push origin v1.0.1
   gh release create v1.0.1 --title "v1.0.1 - Dashboard Redesign" --notes "Comprehensive frontend redesign + installer fix (WinError 32 on reinstall)." --latest
   C:\Projects\ArcVault2.0\scripts\upload_v1_assets.ps1 -Tag v1.0.1
   gh release upload v1.0.1 "C:\Projects\ArcVault2.0\deployment\ArcVault-Installer.exe" --clobber
   ```
3. Commit installer fix:
   ```
   git add deployment\arcvault_installer.py
   git commit -m "fix: stop service before download to prevent WinError 32 on reinstall"
   git push
   ```
4. Test full install on another machine using the updated `ArcVault-Installer.exe`

**Key files changed this session:**
- `dashboard/src/` — all 21 Vue files redesigned (design token system)
- `scripts/rebuild-and-restart.ps1` — ProjectRoot path fix
- `deployment/arcvault_installer.py` — WinError 32 fix (service stop before download)

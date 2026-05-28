---
name: ArcVault Lessons Learned
category: memory
priority: medium
last_updated: 2026-05-28
last_accessed: 2026-05-28
stale_after_days: 90
auto_summarize: true
archive_policy: keep
---

# ArcVault Lessons Learned

Patterns and fixes worth preserving across sessions. Distilled from `MEMORY.md` "Known Improvements" and phase notes.

---

## Testing

- **Windows test skips are expected** — 2 tests skip on Windows (agent/updater platform constraints); do not treat as a regression, baseline is 108 pass + 2 skip
- **Run `go test ./...` before any commit** — tests catch regressions the type system doesn't; non-negotiable

## Go Patterns

- **Notifications must never block** — any notification or retry logic must run in a goroutine; learned from Phase 12 where blocking the job result handler was a risk
- **`started_at` accuracy** — job start time was initially inaccurate because it was recorded at result time, not start time; fix: record in `job_runs` table at job start

## Vue Patterns

- **WebSocket updates must preserve filter state** — early versions reset search/filter on WebSocket push; fix: check filter state before re-rendering, preserve user selections
- **Auto-refresh composables** — 15–30s intervals work well for dashboard data; avoids WebSocket complexity for non-critical polling
- **`[data-theme]` inside `<style scoped>` NEVER works** — Vue scoped CSS adds `data-v-XXXX` to every selector, so `[data-theme="dark"]` becomes `[data-theme="dark"][data-v-XXXX]` which cannot match `<html data-theme="dark">`. Theme overrides MUST go in the global `style.css`, not in scoped component styles. Multiple components (Users.vue, Groups.vue, ChangePasswordModal.vue) had this silent bug.
- **Hardcoded colors in scoped CSS break theming** — scoped specificity can override global CSS variable definitions. Prefer removing duplicate class definitions from scoped CSS entirely and relying on global `style.css` for shared classes (`.table`, `.badge`, `.chip`, etc.)
- **Old token names linger invisibly** — some components used `--bg-primary`, `--bg-secondary`, `--border-color`, `--accent-color` (never defined in our design system). Always audit for these when touching a component; they silently produce no color (transparent/inherited).

## Federation

- **Missed schedule deduplication is necessary** — without checking alert_history, missed schedule alerts fire repeatedly; always check before appending

## PowerShell Build Scripts

- **Avoid custom function names that shadow PowerShell builtins** — `Write-Error`, `Write-Info`, `Write-Success` conflict with built-in cmdlets and cause garbled output where function bodies get printed instead of executed. Use plain `Write-Host` with `-ForegroundColor` inline, or prefix custom functions clearly (e.g. `Write-BuildInfo`). Even prefixed names can break if the file encoding is corrupted by external tools (e.g. `sed` on Linux).
- **Don't pass `--onefile` to PyInstaller when using a `.spec` file** — the spec file already encodes the output mode; passing `--onefile` alongside it throws `makespec options not valid when a .spec file is given`.
- **Keep build scripts simple and flat** — the cleanest version of `build-windows-installer.ps1` uses no custom functions, inline `Write-Host` calls, and a single `Set-Location` at the top to ensure relative paths resolve correctly regardless of where the script is invoked from.

## PowerShell 5.1 Quirks

- **No `&&` statement separator** — PowerShell 5.1 (Windows built-in) does not support `&&`. Use separate commands or semicolons. `&&` was added in PowerShell 7.
- **Multiline clipboard pastes reverse in PS5.1** — copying a multiline string to the clipboard and pasting into PowerShell 5.1 can result in lines pasting in reverse order. Always use single-line commands when driving PS via clipboard.
- **`$MyInvocation.MyCommand.Path` vs `$PSScriptRoot`** — when a script is in a subfolder (e.g. `scripts/`), `Split-Path -Parent $MyInvocation.MyCommand.Path` returns that subfolder. Use `Split-Path -Parent $PSScriptRoot` to reliably get the parent (project root). `$PSScriptRoot` is always the directory containing the running script.

## Installer (Windows)

- **Stop the service before copying OR downloading binaries** — both `shutil.copy` and `urllib` throw `WinError 32` if `coordinator.exe` or `agent.exe` is locked by a running service. The fix in `arcvault_installer.py` is to run `sc stop <service>` + `time.sleep(2)` at the start of Step 1 (download), before touching the destination file. The old pattern only stopped the service at Step 3 (service install), which was too late.
- **Deployment packages belong in `deployment/`** — the PyInstaller `--distpath` flag controls where the final `.exe` lands; set it to `deployment` not `dist`.

## Dashboard Deploy Pipeline

- **`coordinator/static/dist/` is the go:embed source, NOT `dashboard/dist/`** — Vite outputs to `dashboard/dist/`; Go embeds from `coordinator/static/dist/`. These must be synced before `go build` or the binary will have stale UI. `scripts/rebuild-and-restart.ps1` Step 3 handles this automatically — always use the script for deploys, not manual `go build`.
- **Working-tree files can silently truncate** — detected on 2026-05-28: api.js, Agents.vue, Jobs.vue, Templates.vue all ended mid-line in the working tree but were complete in git HEAD. Restore with `git show HEAD:<path> > <path>`. Check with `wc -l` vs `git show HEAD:<path> | wc -l` when builds fail unexpectedly.
- **PowerShell `copy /Y` is CMD syntax** — use `Copy-Item -Force src dst` in PowerShell. The `/Y` flag is not a valid parameter for `Copy-Item`.

## Known Technical Debt (from MEMORY.md)

- Email notifier does not support TLS client certificate authentication
- User search/filter not implemented in admin panel
- Password reset via email not implemented (users can only change own password)
- `started_at` column added to job_runs for accuracy (Phase 17); notification `StartedAt` still uses workaround

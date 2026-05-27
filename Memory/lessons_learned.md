---
name: ArcVault Lessons Learned
category: memory
priority: medium
last_updated: 2026-05-26
last_accessed: 2026-05-26
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

## Federation

- **Missed schedule deduplication is necessary** — without checking alert_history, missed schedule alerts fire repeatedly; always check before appending

## PowerShell Build Scripts

- **Avoid custom function names that shadow PowerShell builtins** — `Write-Error`, `Write-Info`, `Write-Success` conflict with built-in cmdlets and cause garbled output where function bodies get printed instead of executed. Use plain `Write-Host` with `-ForegroundColor` inline, or prefix custom functions clearly (e.g. `Write-BuildInfo`). Even prefixed names can break if the file encoding is corrupted by external tools (e.g. `sed` on Linux).
- **Don't pass `--onefile` to PyInstaller when using a `.spec` file** — the spec file already encodes the output mode; passing `--onefile` alongside it throws `makespec options not valid when a .spec file is given`.
- **Keep build scripts simple and flat** — the cleanest version of `build-windows-installer.ps1` uses no custom functions, inline `Write-Host` calls, and a single `Set-Location` at the top to ensure relative paths resolve correctly regardless of where the script is invoked from.

## Installer (Windows)

- **Stop the service before copying binaries** — `shutil.copy` throws `WinError 32` if `coordinator.exe` or `agent.exe` is locked by a running service. Always run `sc stop <service>` + `time.sleep(2)` before copying in `arcvault_installer.py`.
- **Deployment packages belong in `deployment/`** — the PyInstaller `--distpath` flag controls where the final `.exe` lands; set it to `deployment` not `dist`.

## Known Technical Debt (from MEMORY.md)

- Email notifier does not support TLS client certificate authentication
- User search/filter not implemented in admin panel
- Password reset via email not implemented (users can only change own password)
- `started_at` column added to job_runs for accuracy (Phase 17); notification `StartedAt` still uses workaround

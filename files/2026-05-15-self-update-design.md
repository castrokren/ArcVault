# ArcVault Self-Update Design
**Date:** 2026-05-15
**Phase:** 6 — remaining item
**Status:** Approved

---

## Overview

ArcVault coordinators need a way to update themselves without manual binary replacement. This design adds an in-dashboard self-update flow: a banner alerts the user when a new version is available, clicking it opens a modal that drives the full download → stage → restart sequence with live progress feedback.

A `coordinator check-update` CLI command is also added as a lightweight scriptable alternative.

---

## Goals

- User can update the coordinator from the dashboard in one click
- Progress is visible step-by-step (no black box)
- Coordinator binary is never modified if anything fails before the point of no return
- Works on Windows, Linux, and macOS in both service mode and direct terminal mode
- `coordinator check-update` works as a standalone CLI command

---

## Non-goals

- Agent self-update (out of scope for this phase)
- Automatic/unattended updates (always user-initiated)
- Rollback to a previous version

---

## Architecture

### New package: `coordinator/updater/`

```
coordinator/updater/
  updater.go          -- platform-agnostic: version check, asset resolution, download, verify, stage
  updater_windows.go  -- stop service → rename → start via Windows SCM
  updater_linux.go    -- stop service → rename → start via systemd
  updater_darwin.go   -- stop service → rename → start via launchctl
  updater_test.go     -- 8 tests
```

Follows the same platform-split pattern already established in `coordinator/service/`.

### New server files

```
coordinator/server/
  update.go           -- /api/update/check and /api/update/apply handlers
  update_test.go      -- 4 tests
```

### CLI addition

`coordinator check-update` added to `coordinator/cmd/commands.go`. Reuses the `updater` package for the GitHub API call; prints current vs latest and exits.

---

## Version Check

### Background polling

On coordinator startup, and every 24 hours thereafter, the coordinator calls:

```
GET https://api.github.com/repos/castrokren/ArcVault/releases/latest
```

The result is cached in memory. If the GitHub API is unreachable or rate-limited, the check is silently skipped and retried on the next tick. The banner never shows an error for a background check failure.

### `GET /api/update/check`

Returns the cached result. Called by the dashboard on app mount.

```json
{
  "current": "v0.2.0",
  "latest": "v0.3.1",
  "update_available": true,
  "release_url": "https://github.com/castrokren/ArcVault/releases/tag/v0.3.1"
}
```

Auth: admin token required (same as all management endpoints).

---

## Update Flow

### `POST /api/update/apply`

Admin token required. Kicks off the update goroutine. Returns 202 immediately. Returns 409 if an update is already in progress.

Progress is streamed to all connected dashboard clients via the existing WebSocket hub using a new event type `update_progress`:

```json
{ "type": "update_progress", "step": "resolving",   "pct": 10, "message": "Resolving release asset..." }
{ "type": "update_progress", "step": "downloading",  "pct": 20, "message": "Downloading coordinator_linux_amd64 (18.2 MB)..." }
{ "type": "update_progress", "step": "downloading",  "pct": 55, "message": "Downloading... 55%" }
{ "type": "update_progress", "step": "verifying",    "pct": 80, "message": "Verifying binary..." }
{ "type": "update_progress", "step": "staging",      "pct": 88, "message": "Staging binary..." }
{ "type": "update_progress", "step": "restarting",   "pct": 95, "message": "Restarting service..." }
{ "type": "update_progress", "step": "done",         "pct": 100, "message": "Restarted. Reconnecting..." }
```

On error:
```json
{ "type": "update_progress", "step": "error", "pct": -1, "message": "failed to rename staging binary: permission denied" }
```

### Step-by-step flow

1. **Resolve asset URL** — detect `runtime.GOOS` + `runtime.GOARCH`, construct asset name (e.g. `coordinator_linux_amd64`, `coordinator_windows_amd64.exe`), find matching asset in the GitHub release JSON.
2. **Download** — stream download to a temp file (`coordinator.download.tmp`) in the same directory as the running binary. Track bytes received for progress percentage.
3. **Verify** — chmod +x (Unix), then run `coordinator.download.tmp --version`. Must exit 0 and print a version string. If it fails, delete the temp file and emit error.
4. **Stage** — rename `coordinator.download.tmp` → `coordinator.new` (atomic on Unix; on Windows this is a plain rename since the file isn't running yet).
5. **Stop service / restart** — platform handler takes over:
   - **Service mode:** stop service → rename `coordinator.new` → `coordinator` → start service
   - **Terminal mode:** rename `coordinator.new` → `coordinator`, emit terminal-mode done event (no restart)
6. **Coordinator exits** (service mode) — the service manager restarts it with the new binary.

### Service mode detection

Check for the presence of a `--service` flag or `ARCVAULT_SERVICE=1` env var set by the service wrapper at launch. If absent, terminal mode fallback applies.

### Terminal mode done event

```json
{ "type": "update_progress", "step": "done_manual", "pct": 100, "message": "Binary updated. Please restart the coordinator manually." }
```

---

## Error Handling

| Step | Failure | Recovery |
|------|---------|----------|
| Version check (background) | GitHub unreachable / rate limited | Silent skip, retry in 24h |
| Asset resolution | No asset for this OS/arch | Error event: "No release asset found for your platform." |
| Download | Network error / partial download | Delete temp file, emit error. Binary untouched. |
| Verify | Binary exits non-zero | Delete temp file, emit error. Binary untouched. |
| Staging rename | Permission denied | Emit error. Binary untouched. |
| Service stop | Service manager error | Emit error. Old binary still running. |
| Service start | Fails after replace | Emit error: "New binary is in place but service failed to start. Restart it manually using your platform's service manager." |
| Dashboard reconnect | No reconnect within 60s | Show: "Coordinator may still be restarting. Try refreshing." |

**The invariant:** if the error step is before "staging rename", the coordinator binary is never touched.

---

## Dashboard UI

### Banner

Rendered below the topbar when `update_available: true`. Dismissible per-session (not persisted).

```
[ • ]  ArcVault v0.3.1 is available — you're on v0.2.0   [Update now]  [✕]
```

### Update modal — states

**Confirm** — release notes link, restart warning, Cancel / Update now buttons.

**In progress** — sequential steps list, spinner on active step, progress bar on download step. Cancel disabled once started.

**Success (service mode)** — dashboard WebSocket disconnects on coordinator restart, polls reconnect every 2s for up to 60s. On reconnect: "Updated to v0.3.1 — reconnected successfully."

**Success (terminal mode)** — "Binary updated. Please restart the coordinator manually."

**Error** — "Update failed — coordinator was not modified." with the raw error message in a monospace box.

### WebSocket integration

New handler in the Vue app watches for `update_progress` events and drives the modal state machine. No new WebSocket connection — reuses the existing hub connection.

### Version check on load

`GET /api/update/check` called on app mount. Result stored in a reactive ref shared between the banner component and the modal component.

---

## CLI Command

`coordinator check-update` — calls the GitHub releases API directly (not via the coordinator server), prints result, exits.

```
$ coordinator check-update
current:  v0.2.0
latest:   v0.3.1
status:   update available
release:  https://github.com/castrokren/ArcVault/releases/tag/v0.3.1
```

If already up to date:
```
$ coordinator check-update
current:  v0.3.1
latest:   v0.3.1
status:   up to date
```

---

## Testing

**`coordinator/updater/updater_test.go`** — 8 tests:
- `TestResolveAssetURL` — correct asset name for all OS/arch combos
- `TestDownloadBinary` — happy path with mock HTTP server
- `TestDownloadBinaryNetworkError` — partial download cleaned up
- `TestVerifyBinary` — mock binary returning exit 0 with version string
- `TestVerifyBinaryFails` — mock binary returning exit 1
- `TestDetectServiceMode` — returns true/false correctly
- `TestUpdateProgressEvents` — events emitted in correct order with correct fields
- `TestVersionComparison` — semver comparison (v0.3.1 > v0.2.0, same = no update)

**`coordinator/server/update_test.go`** — 4 tests:
- `TestCheckUpdateEndpoint` — correct JSON shape returned
- `TestCheckUpdateCached` — second call uses cache, no second HTTP request
- `TestApplyUpdateRejectsNonAdmin` — agent token cannot trigger update
- `TestApplyUpdateAlreadyRunning` — second POST while update running returns 409

**Total:** 12 new tests → 70 tests total.

---

## File Change Summary

| File | Change |
|------|--------|
| `coordinator/updater/updater.go` | New |
| `coordinator/updater/updater_windows.go` | New |
| `coordinator/updater/updater_linux.go` | New |
| `coordinator/updater/updater_darwin.go` | New |
| `coordinator/updater/updater_test.go` | New |
| `coordinator/server/update.go` | New |
| `coordinator/server/update_test.go` | New |
| `coordinator/cmd/commands.go` | Add `CheckUpdateCommand` |
| `coordinator/main.go` | Wire `check-update` subcommand + background version poll |
| `coordinator/server/server.go` | Register `/api/update/check` and `/api/update/apply` routes |
| `dashboard/src/` | New `UpdateBanner.vue`, `UpdateModal.vue`, update store/websocket handler |

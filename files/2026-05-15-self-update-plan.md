# ArcVault Self-Update — Implementation Plan
**Date:** 2026-05-15
**Spec:** 2026-05-15-self-update-design.md

---

## Task order

Tasks are sequenced so each builds on the last. Backend first, then routes, then dashboard, then CLI, then tests.

---

## Task 1 — `coordinator/updater/updater.go`

Platform-agnostic core. No platform build tags.

**Exports:**
```go
type UpdateInfo struct {
    Current         string
    Latest          string
    UpdateAvailable bool
    ReleaseURL      string
    AssetURL        string
}

type ProgressEvent struct {
    Type    string `json:"type"`    // always "update_progress"
    Step    string `json:"step"`
    Pct     int    `json:"pct"`
    Message string `json:"message"`
}

func CheckLatestRelease(currentVersion string) (*UpdateInfo, error)
func ResolveAssetURL(releaseJSON []byte) (string, error)
func DownloadBinary(assetURL string, destPath string, progress func(pct int)) error
func VerifyBinary(path string) error
func StageBinary(tmpPath, stagedPath string) error
func IsServiceMode() bool
```

**Implementation notes:**
- `CheckLatestRelease` — GET `https://api.github.com/repos/castrokren/ArcVault/releases/latest`, parse tag_name and assets array
- `ResolveAssetURL` — match asset name against `coordinator_{GOOS}_{GOARCH}` (+ `.exe` on Windows)
- `DownloadBinary` — stream to file, call `progress(pct)` callback every ~512KB
- `VerifyBinary` — chmod 0755 (Unix only; skip on Windows), exec `path --version`, check exit 0
- `StageBinary` — `os.Rename(tmpPath, stagedPath)` — atomic on Unix
- `IsServiceMode` — check `os.Getenv("ARCVAULT_SERVICE") == "1"`

---

## Task 2 — Platform handlers

Each file implements:
```go
func ApplyUpdate(stagedPath, currentPath string, progress func(ProgressEvent)) error
```

**`updater_linux.go`** (build tag: `//go:build linux`)
1. `progress` → step "restarting", pct 95
2. `exec.Command("systemctl", "stop", "arcvault-coordinator").Run()`
3. `os.Rename(stagedPath, currentPath)`
4. `exec.Command("systemctl", "start", "arcvault-coordinator").Run()`
5. Return nil (coordinator will be killed by systemd stop, new one starts)

**`updater_darwin.go`** (build tag: `//go:build darwin`)
Same shape, use `launchctl stop/start com.arcvault.coordinator`

**`updater_windows.go`** (build tag: `//go:build windows`)
Reuse the existing `coordinator/service` Windows SCM helpers:
1. Open service manager
2. Stop `arcvault-coordinator`
3. `os.Rename(stagedPath, currentPath)` — safe because service is stopped
4. Start service
5. Return nil

**Terminal mode path** (in `updater.go`, not platform files):
```go
if !IsServiceMode() {
    os.Rename(stagedPath, currentPath)
    progress(ProgressEvent{..., Step: "done_manual", Pct: 100, ...})
    return nil
}
```
Called before `ApplyUpdate` — skip the platform handler entirely.

---

## Task 3 — Background version poller

In `coordinator/main.go`, after `StartCommand` launches the server:

```go
go func() {
    checkAndCache() // immediate check on startup
    ticker := time.NewTicker(24 * time.Hour)
    for range ticker.C {
        checkAndCache()
    }
}()
```

`checkAndCache` calls `updater.CheckLatestRelease(currentVersion)` and stores result in a package-level `sync.RWMutex`-protected var in `coordinator/server/update.go`.

---

## Task 4 — `coordinator/server/update.go`

Two handlers + shared cached state.

```go
var (
    updateMu      sync.RWMutex
    cachedInfo    *updater.UpdateInfo
    updateRunning atomic.Bool
)

func SetUpdateCache(info *updater.UpdateInfo)   // called by poller
func handleCheckUpdate(w, r)                    // GET /api/update/check
func handleApplyUpdate(w, r)                    // POST /api/update/apply
```

`handleCheckUpdate`:
- Read lock, return cached info as JSON
- If cache is nil (no check yet), call `updater.CheckLatestRelease` inline and cache it

`handleApplyUpdate`:
- Reject non-admin tokens → 403
- Reject if `updateRunning` is true → 409
- Set `updateRunning = true`, defer `updateRunning = false`
- Launch goroutine: run full update flow, emit `ProgressEvent`s via hub broadcast

Progress emission:
```go
broadcast := func(evt updater.ProgressEvent) {
    b, _ := json.Marshal(evt)
    hub.Broadcast(b)
}
```

---

## Task 5 — Route registration

In `coordinator/server/server.go`, add inside `setupRoutes` (or wherever existing routes are registered):

```go
mux.Handle("GET /api/update/check", adminMiddleware(handleCheckUpdate))
mux.Handle("POST /api/update/apply", adminMiddleware(handleApplyUpdate))
```

---

## Task 6 — `coordinator/updater/updater_test.go`

8 tests. Use `net/http/httptest` for mock GitHub API server.

```
TestResolveAssetURL         — table test: linux/amd64, darwin/arm64, windows/amd64
TestDownloadBinary          — mock server returns 100 bytes, verify file written
TestDownloadBinaryNetworkError — mock server closes mid-stream, verify tmp deleted
TestVerifyBinary            — write a tiny shell script as mock binary, verify passes
TestVerifyBinaryFails       — write script that exits 1, verify error returned
TestDetectServiceMode       — set/unset ARCVAULT_SERVICE env var
TestUpdateProgressEvents    — mock all steps, collect emitted events, assert order+fields
TestVersionComparison       — semver compare: newer/same/older
```

---

## Task 7 — `coordinator/server/update_test.go`

4 tests. Follow existing test patterns in `coordinator/server/*_test.go`.

```
TestCheckUpdateEndpoint       — mock updater, assert JSON shape
TestCheckUpdateCached         — call twice, assert GitHub only hit once
TestApplyUpdateRejectsNonAdmin — use agent token, assert 403
TestApplyUpdateAlreadyRunning  — set updateRunning=true, POST, assert 409
```

---

## Task 8 — `coordinator check-update` CLI command

In `coordinator/cmd/commands.go`, add:

```go
func CheckUpdateCommand(currentVersion string) error {
    info, err := updater.CheckLatestRelease(currentVersion)
    if err != nil {
        return fmt.Errorf("could not check for updates: %w", err)
    }
    fmt.Printf("current:  %s\n", info.Current)
    fmt.Printf("latest:   %s\n", info.Latest)
    if info.UpdateAvailable {
        fmt.Printf("status:   update available\n")
        fmt.Printf("release:  %s\n", info.ReleaseURL)
    } else {
        fmt.Printf("status:   up to date\n")
    }
    return nil
}
```

Wire in `coordinator/main.go`:
```go
case "check-update":
    if err := cmd.CheckUpdateCommand(version); err != nil {
        log.Fatal(err)
    }
```

---

## Task 9 — `dashboard/src/components/UpdateBanner.vue`

```vue
<template>
  <div v-if="updateStore.available && !dismissed" class="update-banner">
    <span class="dot" />
    <span>ArcVault {{ updateStore.latest }} is available — you're on {{ updateStore.current }}</span>
    <button @click="openModal">Update now</button>
    <button @click="dismissed = true" aria-label="Dismiss">✕</button>
  </div>
</template>
```

- `updateStore` — a reactive ref (or Pinia store if already used in project) holding the check result
- `dismissed` — local `ref(false)`, session-only

---

## Task 10 — `dashboard/src/components/UpdateModal.vue`

Modal state machine driven by `update_progress` WebSocket events.

**States:** `confirm` → `progress` → `success` | `success_manual` | `error`

```
confirm       User sees version info + restart warning
progress      Steps list, spinner on active step, progress bar on download
reconnecting  Coordinator restarted, polling WebSocket every 2s for up to 60s
success       Reconnected, show "Updated to vX.X.X"
success_manual Terminal mode, show "Restart manually"
error         Show error message + error detail
```

WebSocket event handler (in existing ws setup file):
```js
case 'update_progress':
  updateStore.handleProgressEvent(event)
  break
```

Reconnect polling (triggered when step === 'restarting'):
```js
const poll = setInterval(() => {
  if (ws.readyState === WebSocket.OPEN) {
    clearInterval(poll)
    state.value = 'success'
  }
}, 2000)
setTimeout(() => clearInterval(poll), 60_000)
```

---

## Task 11 — Wire banner into `App.vue`

```vue
<UpdateBanner />
<RouterView />
```

On `onMounted`:
```js
const info = await fetch('/api/update/check', { headers: authHeaders }).then(r => r.json())
updateStore.setInfo(info)
```

---

## Completion criteria

- [ ] `go test ./coordinator/...` — 70 tests passing, 0 failures
- [ ] `coordinator check-update` prints correct output against live GitHub API
- [ ] Banner appears in dashboard when a newer version exists
- [ ] Full modal flow works end-to-end in dev (can be tested with a mock `/api/update/apply` that emits fake events)
- [ ] Error states render correctly for each failure mode
- [ ] No coordinator binary modification on any pre-staging failure

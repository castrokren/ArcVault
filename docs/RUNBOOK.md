# ArcVault Runbook

Operational reference for building, deploying, and maintaining ArcVault.
Last updated: 2026-05-27

---

## Quick Reference

| Thing | Value |
|---|---|
| Dashboard URL | http://localhost:8080 |
| Default login | `admin` / `changeme` |
| Coordinator service | `arcvault-coordinator` → `C:\ArcVault\coordinator.exe` |
| Agent service | `arcvault-agent` → `C:\ArcVault-Agent\agent.exe` |
| Coordinator config | `C:\ArcVault\config.json` |
| Agent config | `C:\ArcVault-Agent\agent-config.yaml` |
| Database | `C:\ArcVault\arcvault.db` |
| Dev project | `C:\Projects\ArcVault2.0\` |

---

## Rebuilding and Redeploying

Any time you change Go code or dashboard Vue/JS files, run the rebuild script from an **Admin PowerShell**:

```powershell
cd C:\Projects\ArcVault2.0
.\rebuild-and-restart.ps1
```

This script does everything in order:
1. Stops both services (`arcvault-coordinator`, `arcvault-agent`)
2. Builds the Vue dashboard (`npm run build`)
3. Syncs `dashboard\dist\` → `coordinator\static\dist\` *(required for embed)*
4. Compiles `coordinator.exe` with the fresh dashboard embedded
5. Compiles `agent.exe`
6. Copies both binaries to `C:\ArcVault\` and `C:\ArcVault-Agent\`
7. Starts both services
8. Verifies the coordinator is healthy and the agent is registered

> **Must run as Administrator.** Service management requires elevated permissions.

### Why the sync step matters

The Go binary embeds the dashboard via `//go:embed dist` in `coordinator/static/static.go`.
That embed reads from `coordinator/static/dist/` — **not** `dashboard/dist/`.
If you skip the sync, `go build` will bake in an old dashboard and the UI will be wrong.

---

## Changing Only Go Code (No Dashboard Changes)

If you only modified `.go` files and the dashboard hasn't changed, you can skip `npm run build` manually — but running the full script is always safe and takes less than 30 seconds.

---

## Changing Only Dashboard Code

If you only modified Vue/JS files:

```powershell
cd C:\Projects\ArcVault2.0\dashboard
npm run build
cd ..

# Sync to embed folder
Remove-Item -Recurse -Force "coordinator\static\dist\*"
Copy-Item -Recurse "dashboard\dist\*" "coordinator\static\dist\" -Force

# Rebuild coordinator (must re-embed the new dist)
go build -o coordinator.exe .\coordinator

# Deploy and restart
sc.exe stop arcvault-coordinator
Copy-Item coordinator.exe C:\ArcVault\coordinator.exe -Force
sc.exe start arcvault-coordinator
```

---

## Checking Service Status

```powershell
sc.exe query arcvault-coordinator
sc.exe query arcvault-agent
```

Or verify the API is up:

```powershell
Invoke-RestMethod -Uri "http://localhost:8080/health"
```

Check registered agents:

```powershell
$token = (Get-Content C:\ArcVault\config.json | ConvertFrom-Json).admin_token
Invoke-RestMethod -Uri "http://localhost:8080/api/agents" -Headers @{ Authorization = "Bearer $token" }
```

---

## Starting / Stopping Services Manually

```powershell
# Stop
sc.exe stop arcvault-agent
sc.exe stop arcvault-coordinator

# Start
sc.exe start arcvault-coordinator
sc.exe start arcvault-agent
```

> Always stop the **agent first**, start the **coordinator first**.

---

## Updating the Agent Config

Edit `C:\ArcVault-Agent\agent-config.yaml` then restart the agent service:

```powershell
sc.exe stop arcvault-agent
# edit the file
sc.exe start arcvault-agent
```

The `auth_token` must match `admin_token` in `C:\ArcVault\config.json`.

---

## Clearing a Stale Browser Session

If the dashboard shows a blank page after a coordinator update, clear the old JWT:

1. Press `F12` → **Console** tab
2. Run: `localStorage.clear(); location.reload()`

Or open the dashboard in an incognito window.

---

## Releasing a New Version (GitHub)

Push a version tag to trigger the CI/CD workflow. GitHub will build installers for Windows, macOS, and Linux and publish them as a GitHub Release — no local build tooling required for end users.

```powershell
git tag v1.0.0
git push origin v1.0.0
```

The workflow (`.github/workflows/build-installers.yml`) runs the full build:
builds the dashboard → syncs to embed folder → compiles Go binaries → packages installers → creates GitHub Release.

Users can then download directly from: `https://github.com/castrokren/ArcVault/releases`

---

## Common Issues

| Symptom | Likely cause | Fix |
|---|---|---|
| Dashboard shows token input, not login form | Old dashboard binary deployed | Re-run `rebuild-and-restart.ps1` (sync step was skipped) |
| Agent shows 401 on registration | Token mismatch between agent config and coordinator config | Update `auth_token` in `C:\ArcVault-Agent\agent-config.yaml` to match `C:\ArcVault\config.json` |
| Port 8080 already in use after stop | Service SCM is restarting the process | Use `sc.exe stop` from Admin PS, not `taskkill` |
| `Stop-Service` / `Start-Service` errors | PowerShell not elevated | Right-click PowerShell → Run as Administrator |
| Coordinator starts then immediately exits | Port still bound from previous process | Wait for `sc.exe stop` to fully complete before starting again |
| Blank dashboard page after login | Stale JWT in localStorage | Clear localStorage via DevTools console |

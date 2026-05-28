# Phase 14 Smoke Test — Execution Plan for Claude Code

## Context

ArcVault2.0 is a coordinator + agent backup system written in Go + Vue 3.
Phase 14 added multi-coordinator federation. All 111 server tests pass and
both `go build` and `npm run build` are clean on branch `phase-14-federation`.

The coordinator binary is at: `C:\Projects\ArcVault2.0\coordinator`
The dashboard dist is at: `C:\Projects\ArcVault2.0\dashboard\dist`
Config files live at: `%USERPROFILE%\.arcvault\config.json`

---

## Your Task

Run the Phase 14 smoke test checklist. You will:

1. Start two coordinator instances (root + sub)
2. Work through every checklist item below
3. Report pass/fail for each item with any relevant output
4. If anything fails, capture the error and note which checklist item failed

Do NOT modify any source code. This is verification only.

---

## Setup

### Step 1 — Build the binary

```powershell
cd C:\Projects\ArcVault2.0
go build -o coordinator\arcvault-coordinator.exe .\coordinator\cmd\...
```

### Step 2 — Create root coordinator config

Write to `%USERPROFILE%\.arcvault\root-config.json`:

```json
{
  "port": 8090,
  "database_path": "C:\\Temp\\arcvault-root.db",
  "admin_token": "root-admin-token",
  "environment": "test"
}
```

### Step 3 — Create sub-coordinator config

Write to `%USERPROFILE%\.arcvault\sub-config.json`:

```json
{
  "port": 8091,
  "database_path": "C:\\Temp\\arcvault-sub.db",
  "admin_token": "sub-admin-token",
  "environment": "test",
  "federation": {
    "root_url": "http://localhost:8090",
    "token": "fed-smoke-token"
  }
}
```

### Step 4 — Register the sub on the root

Before starting the sub, register it on the root so the token is in the DB:

```powershell
# Start root in background
Start-Process -NoNewWindow coordinator\arcvault-coordinator.exe `
  -ArgumentList "--config", "$env:USERPROFILE\.arcvault\root-config.json"

Start-Sleep -Seconds 2

# Register the sub-coordinator
$body = '{"name":"Smoke Sub","url":"http://localhost:8091","token":"fed-smoke-token"}'
Invoke-RestMethod -Uri "http://localhost:8090/api/federation" `
  -Method POST `
  -Headers @{ Authorization = "Bearer root-admin-token"; "Content-Type" = "application/json" } `
  -Body $body
```

---

## Checklist

Work through each item. For each one, make the API call or observation described
and record the result.

---

### ✅ Item 1 — Root starts clean, no federation clients connected

```powershell
Invoke-RestMethod -Uri "http://localhost:8090/api/federation" `
  -Headers @{ Authorization = "Bearer root-admin-token" }
```

**Expected:** Array with one entry (`Smoke Sub`), status = `offline`

---

### ✅ Item 2 — Sub starts and connects to root

```powershell
# Start sub in background
Start-Process -NoNewWindow coordinator\arcvault-coordinator.exe `
  -ArgumentList "--config", "$env:USERPROFILE\.arcvault\sub-config.json"

Start-Sleep -Seconds 3

# Check status on root
Invoke-RestMethod -Uri "http://localhost:8090/api/federation" `
  -Headers @{ Authorization = "Bearer root-admin-token" }
```

**Expected:** `Smoke Sub` status = `online`, version is populated

---

### ✅ Item 3 — Register an agent on the sub, verify it appears in root cache

```powershell
# Create an agent token on the sub
$tok = Invoke-RestMethod -Uri "http://localhost:8091/api/agents/register" `
  -Method POST `
  -Headers @{ Authorization = "Bearer sub-admin-token"; "Content-Type" = "application/json" } `
  -Body '{"agent_id":"smoke-agent-01","hostname":"smoke-box","os":"windows","arch":"amd64","version":"v0.7.0"}'

Start-Sleep -Seconds 2

# Check root cache
Invoke-RestMethod -Uri "http://localhost:8090/api/federation/<SUB_ID>/agents" `
  -Headers @{ Authorization = "Bearer root-admin-token" }
```

Replace `<SUB_ID>` with the ID returned from Item 1.

**Expected:** `agents` array contains `smoke-agent-01`, `stale = false`

---

### ✅ Item 4 — Site selector appears in dashboard

Open `http://localhost:8090` in a browser (the embedded dashboard).
Enter admin token `root-admin-token`.

**Expected:**
- Federation nav link visible
- Site selector dropdown appears in nav bar showing `Smoke Sub`
- Agents view shows agents from local coordinator by default

---

### ✅ Item 5 — Selecting a site filters to sub's agents

In the dashboard, select `Smoke Sub` from the site selector.
Navigate to Agents view.

**Expected:** Shows `smoke-agent-01` from the sub. No stale banner.

---

### ✅ Item 6 — Stop sub, stale banner appears, data retained

```powershell
# Stop the sub process
Stop-Process -Name "arcvault-coordinator" -ErrorAction SilentlyContinue
# (stop only the sub — port 8091)

Start-Sleep -Seconds 3

# Check root
Invoke-RestMethod -Uri "http://localhost:8090/api/federation" `
  -Headers @{ Authorization = "Bearer root-admin-token" }
```

**Expected:** `Smoke Sub` status = `offline`

In the dashboard with `Smoke Sub` selected:
**Expected:** Amber stale banner visible, `smoke-agent-01` still shown in table

---

### ✅ Item 7 — Restart sub, banner clears

```powershell
Start-Process -NoNewWindow coordinator\arcvault-coordinator.exe `
  -ArgumentList "--config", "$env:USERPROFILE\.arcvault\sub-config.json"

Start-Sleep -Seconds 4

Invoke-RestMethod -Uri "http://localhost:8090/api/federation" `
  -Headers @{ Authorization = "Bearer root-admin-token" }
```

**Expected:** `Smoke Sub` status = `online`

In the dashboard: stale banner gone, data current.

---

### ✅ Item 8 — Force Sync button works

In the dashboard Federation view, click **Sync** on `Smoke Sub`.

```powershell
# Or via API:
Invoke-RestMethod -Uri "http://localhost:8090/api/federation/<SUB_ID>/sync" `
  -Method POST `
  -Headers @{ Authorization = "Bearer root-admin-token" }
```

**Expected:** Returns 202. After a few seconds, sub reconnects and status is `online`.

---

### ✅ Item 9 — Existing behavior unchanged (standalone mode)

```powershell
Invoke-RestMethod -Uri "http://localhost:8090/health"
Invoke-RestMethod -Uri "http://localhost:8090/api/agents" `
  -Headers @{ Authorization = "Bearer root-admin-token" }
Invoke-RestMethod -Uri "http://localhost:8090/api/jobs" `
  -Headers @{ Authorization = "Bearer root-admin-token" }
```

**Expected:** All return 200 with valid JSON. No errors.

---

## Teardown

```powershell
Stop-Process -Name "arcvault-coordinator" -ErrorAction SilentlyContinue
Remove-Item "C:\Temp\arcvault-root.db" -ErrorAction SilentlyContinue
Remove-Item "C:\Temp\arcvault-sub.db" -ErrorAction SilentlyContinue
```

---

## Report Format

Return a report in this format:

```
## Phase 14 Smoke Test Results

| # | Item | Result | Notes |
|---|------|--------|-------|
| 1 | Root starts clean | PASS/FAIL | ... |
| 2 | Sub connects to root | PASS/FAIL | ... |
| 3 | Agent appears in root cache | PASS/FAIL | ... |
| 4 | Site selector in dashboard | PASS/FAIL | ... |
| 5 | Site filter works | PASS/FAIL | ... |
| 6 | Offline: stale banner + data retained | PASS/FAIL | ... |
| 7 | Reconnect: banner clears | PASS/FAIL | ... |
| 8 | Force sync returns 202 | PASS/FAIL | ... |
| 9 | Existing behavior unchanged | PASS/FAIL | ... |

Overall: PASS / FAIL
Blockers: (list any failures with error messages)
```

---

## Notes for Claude Code

- The coordinator binary accepts a `--config` flag pointing to a config file.
  Check `coordinator/cmd/` to confirm the exact flag name if needed.
- If `--config` flag doesn't exist, the coordinator reads from
  `%USERPROFILE%\.arcvault\config.json` — you may need to swap configs between runs.
- `C:\Temp\` must exist — create it if not: `New-Item -ItemType Directory -Force C:\Temp`
- The dashboard is embedded in the binary and served at the root path.
  If not embedded in this build, the dashboard test (Items 4–5) can be skipped
  and verified manually by the developer instead.
- Do not modify any source files. Report failures as-is.

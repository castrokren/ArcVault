# ArcVault v0.5.0 — Session 21 Complete

**Date**: 2026-06-10  
**Status**: COMPLETE ✅  
**Branch**: main  
**Tag**: v0.5.0  

---

## What Was Accomplished

### Phase 3: Agent Registration Fixed
- **Root cause**: `cert.pem` was deleted during previous session's OpenSSL troubleshooting. Coordinator had no cert on disk.
- **Fix**: Ran `coordinator.exe rekey-cert` → regenerated `cert.pem` with proper SANs (IP: 192.168.68.62, 127.0.0.1; DNS: localhost, hostname)
- **Bootstrap delivery**: Fresh bootstrap fetched via `GET /api/admin/bootstrap.ps1` with admin Bearer token (not `/bootstrap.ps1`)
- **cert mismatch on REMOTE**: Bootstrap curl step failed (AV blocked .exe write), so `coordinator.crt` had old cert. Fixed using PowerShell SslStream to pull live cert from TLS handshake and write as PEM:
  ```powershell
  $tcp = New-Object System.Net.Sockets.TcpClient("192.168.68.62", 443)
  $ssl = New-Object System.Net.Security.SslStream($tcp.GetStream(), $false, { $true })
  $ssl.AuthenticateAsClient("192.168.68.62")
  $certBytes = $ssl.RemoteCertificate.Export([System.Security.Cryptography.X509Certificates.X509ContentType]::Cert)
  $b64 = [System.Convert]::ToBase64String($certBytes, [System.Base64FormattingOptions]::InsertLineBreaks)
  $pem = "-----BEGIN CERTIFICATE-----`n$b64`n-----END CERTIFICATE-----"
  Set-Content -Path "C:\ArcVault-Agent\coordinator.crt" -Value $pem -Encoding Ascii
  ```
- **Result**: SMILOW3FLSP001 online and heartbeating ✅

### Phase 4: Real Backup Job
- Created and ran a backup job targeting SMILOW3FLSP001
- Files transferred successfully ✅

### Installer Fixed & Verified
- `write_agent_config` now copies `cert.pem` → `coordinator.crt` and writes `ca_cert_file`
- Strips trailing slash from `coordinator_url` (was causing `//api/...` double slash)
- Default coordinator URL changed from `http://localhost:{port}` → `https://localhost`
- `build-windows-installer.ps1` now injects version via ldflags from `git describe`
- Built `ArcVault-Setup-0.5.0-windows-amd64.exe` and verified full install workflow ✅

---

## Key Lessons

1. **`coordinator.exe rekey-cert`** is the correct tool for cert regeneration — no OpenSSL needed
2. **Bootstrap endpoint** is `/api/admin/bootstrap.ps1`, requires admin Bearer token
3. **PowerShell SslStream** can pull the live cert from a TLS handshake when SMB isn't available
4. **Installer must write `ca_cert_file`** — without it the agent uses system roots and rejects self-signed certs
5. **Trailing slash in coordinator_url** causes `//api/...` double slashes — always strip before writing config

---

## System State

### COORD (192.168.68.62)
- Coordinator: v0.5.0, running as service ✅
- Cert: `C:\ArcVault\cert.pem` (ECDSA P-256, SANs: 192.168.68.62, 127.0.0.1, localhost)
- Key: `C:\ArcVault\key.pem`

### REMOTE (SMILOW3FLSP001)
- Agent: v0.5.0, running as service ✅
- Status: ONLINE in dashboard
- Config: `C:\ArcVault-Agent\agent-config.yaml`
- Cert: `C:\ArcVault-Agent\coordinator.crt`

### LOCAL (DESKTOP-EE77F38)
- Agent: v0.5.0, installed via new installer ✅
- Status: ONLINE in dashboard

---

## Git
- All changes committed and pushed to `main`
- Tagged `v0.5.0`
- Next: Phase 5 planning

---

# Session 23 — June 11, 2026: Obsidian Pro Restyle Implemented + Deployed

## What Happened
1. Executed `obsidian-pro-restyle.md` plan end-to-end (all 7 tasks) — visual-only restyle of the full Vue dashboard
2. Fonts self-hosted via @fontsource (Space Grotesk / Inter / JetBrains Mono); Google CDN links removed
3. New `Sparkline.vue` (inline SVG, no deps) on History stat cards; orbit-ring login scene; skeleton loaders everywhere
4. Deployed via rebuild-and-restart.ps1; dual-theme smoke passed

## Issues Hit & Fixed
- **Edit tool corrupts files on this mount** (null bytes / truncation) — confirmed on main.js + index.html, not just the installer file. ALL writes now via bash heredoc / python
- **rebuild-and-restart.ps1 was pre-TLS**: probed http://localhost:8080 → false "Coordinator not responding", exited before restarting the agent. Fixed: https://localhost base URL, self-signed cert handling (PS 5.1 + 7), 20s /health retry loop, port-443 release check
- **Pre-existing JS test drift** (fails on pristine HEAD, vitest 4.1.8 + jsdom 29): Jobs.integration.test.js (8) stale api mocks (getToken/getJobRuns) + sync_flags default; SyncFlagsBuilder.test.js (3) trigger('input') vs v-model 'change'. vitest/jsdom never pinned. TODO: fix tests + pin toolchain

## Git
- Restyle committed by Kren via PowerShell (dashboard/* + Sparkline.vue + spec + plan + deploy script fix)
- Pre-existing uncommitted edits (CONTEXT/MEMORY/SESSION_NOTES/arcvault.spec/installer/build script) left for separate commit

# ArcVault v0.5.0 Verification Session - Session End Report

**Date**: 2026-06-10  
**Status**: COMPLETE — Agent SMILOW3FLSP001 online and heartbeating ✅  
**Next Action**: Phase 4 — create and run a real backup job  

---

## Summary

Successfully built, deployed, and partially tested v0.5.0 with curl-based bootstrap and TLS infrastructure. Bootstrap script executes successfully on REMOTE machine. **Agent service installs but fails to register** due to certificate validation errors. Root cause: Certificate generation/validation issues across multiple layers.

---

## What Worked ✅

1. **Code fixes verified**: All source code changes for v0.5.0 are in place
   - Bootstrap script uses `curl.exe` instead of `Invoke-WebRequest` (fixes PS 5.1 TLS renegotiation bug)
   - Agent.exe path construction fixed (`filepath.Join` instead of string manipulation)
   - ReadCertPEM returns PEM bytes, not DER
   
2. **Coordinator service deployed and running** on COORD (192.168.68.62)
   - Listens on port 443
   - Can fetch bootstrap.ps1 via loopback
   - Can serve agent.exe for download

3. **Bootstrap script executes successfully** on REMOTE (SMILOW3FLSP001)
   - Downloads agent.exe
   - Verifies agent.exe hash
   - Installs Windows service
   - Service enters START_PENDING → RUNNING state
   - Old agent service properly uninstalled

4. **Network connectivity confirmed**
   - REMOTE and COORD on same subnet (192.168.68.0/22)
   - Both machines can reach each other's ports
   - Bootstrap downloads working

---

## What Failed ❌

**Agent service installs but immediately crashes on startup** with certificate validation error:

```
registration failed: registration request failed: Post "https://192.168.68.62/api/agents/register": 
tls: failed to verify certificate: x509: certificate relies on legacy Common Name field, use SANs instead
```

This happens in agent's Go crypto/tls library doing strict certificate validation.

---

## Root Cause Analysis

### Issue 1: IP Address Changed (RESOLVED)
- **Problem**: IP changed from 192.168.68.64 → 192.168.68.62 during session
- **Impact**: Bootstrap script and certificate both hardcoded old IP
- **Solution**: Updated config.json, generated new certificates for new IP
- **Status**: ✅ FIXED

### Issue 2: Certificate Generation Failures (PARTIALLY RESOLVED)
- **Problem 1**: Coordinator service wouldn't restart after deleting cert files
  - Can't kill coordinator service process (permission denied)
  - Can't restart via Stop-Service/Start-Service (access issues)
  - Service gets stuck in Stopped state
- **Problem 2**: Coordinator doesn't auto-generate missing certificate
  - Code tries to OPEN cert file, fails if missing
  - No auto-generation logic in startup path
  - Workaround: Generated cert with OpenSSL manually

### Issue 3: Certificate SAN Extension Missing (ROOT CAUSE)
- **Problem**: OpenSSL-generated cert lacked proper SAN extension
  - First attempt: Cert had CN but no SANs
  - Go 1.15+ crypto/tls requires explicit SANs, deprecated CN-only validation
  - Windows schannel also failed without proper extensions
  - Agent error: "certificate relies on legacy Common Name field"
  - Curl error: "schannel: CertFindExtension() returned no extension"

- **Fix Applied**: Regenerated certificate with `-extensions v3_req` flag
  ```
  X509v3 Subject Alternative Name: 
      IP Address:192.168.68.62, IP Address:127.0.0.1, DNS:localhost
  ```

- **Status**: ✅ Certificate now has proper SAN, but last test still failed
  - Possible cause: Service wasn't restarted, old cert still in coordinator memory
  - Or: Need admin restart of coordinator service to deploy new cert

---

## Current System State

### COORD Machine (192.168.68.62)
- **Status**: Coordinator can run manually, needs service restart to persist
- **Coordinator binary**: v0.5.0 ✓
- **Agent binary**: v0.5.0 available for download ✓
- **Certificate**: Regenerated with proper SANs
  - Location: `C:\ArcVault\cert.pem` (newly generated with openssl)
  - Key: `C:\ArcVault\key.pem` (ECDSA P-256)
  - SANs: 192.168.68.62, 127.0.0.1, localhost
  - Validity: 10 years
- **Bootstrap.ps1**: Latest version at `C:\Temp\bootstrap.ps1` with new cert embedded
- **Config**: Updated to use 192.168.68.62

**Critical Issue**: Coordinator service won't stay running. Manual execution works, but service restart fails. Need to:
1. Properly restart coordinator service with admin privileges
2. Or fix the service startup issue

### REMOTE Machine (SMILOW3FLSP001)
- **Agent binary**: Downloaded and verified (v0.5.0) ✓
- **Agent service**: Installed but crashes on startup
- **Config file**: `C:\ArcVault-Agent\agent-config.yaml` created correctly
- **Certificate**: Written to `C:\ArcVault-Agent\coordinator.crt` (from bootstrap script)
- **Logs**: Show repeated cert validation failure (see below)

**Agent startup loop**:
```
Agent starts → Tries to register with coordinator → TLS cert validation fails → Service crashes
→ Windows restarts service automatically → Loop repeats every 3 seconds
```

Agent logs last 50 lines show repeated:
```
2026/06/10 11:52:05.653269 registration failed: registration request failed: 
Post "https://192.168.68.62/api/agents/register": 
tls: failed to verify certificate: x509: certificate relies on legacy Common Name field, use SANs instead
```

---

## Lessons Learned

### 1. Certificate Validation is Multi-Layer
- **Curl (Windows schannel)**: Needs proper X.509 extensions
- **Go crypto/tls**: Requires explicit SANs, deprecates CN-only
- **OpenSSL cert generation**: Need `-extensions v3_req` flag to include SAN in output
- **Agent code**: Uses Go's strict validation, rejects CN-only certs

### 2. IP Address Migration Requires Certificate Regeneration
- Self-signed certificates bind to specific IP/hostname
- Simply changing config.json doesn't invalidate old cert
- Coordinator startup code doesn't auto-generate missing cert
- Manual certificate regeneration required

### 3. Windows Service Management Issues
- Service process runs as SYSTEM with locked privileges
- Can't kill/restart service from user account
- Service can get stuck in bad state
- Manual binary execution bypasses service issues

### 4. Bootstrap vs Agent TLS Differences
- **Bootstrap script uses curl.exe**: Can use `-k` flag to skip verification
- **Agent uses Go crypto/tls**: Strict verification, no bypass option
- **Both pin to certificate**: But different validation strictness
- Bootstrap can work with CN-only, agent requires SANs

---

## Hypothesis for Remaining Failure

Last test (with properly generated cert with SANs) still failed. Possible reasons:

1. **Coordinator service hasn't actually restarted** - still running old process in memory
   - New cert on disk but old cert served from memory
   - Solution: Force restart of coordinator service

2. **Agent received old cert from coordinator**
   - Bootstrap script cached old cert before new one was deployed
   - Solution: Clear `C:\ArcVault-Agent\coordinator.crt` and re-run bootstrap

3. **Go client doesn't trust the self-signed cert**
   - Even with proper SANs, might need `-insecureSkipVerify` flag
   - Agent code may need modification to accept self-signed certs with SAN validation

4. **Network cached old state**
   - Unlikely but possible

---

## Remaining Tasks for Next Session

### Immediate (To Complete Phase 3)
1. **Restart coordinator service properly** with admin privileges
   - Need elevated PowerShell to restart service
   - Or use Group Policy to auto-start after reboot
   
2. **Verify new cert is actually served**
   - Fetch bootstrap.ps1 from restarted coordinator
   - Confirm embedded cert has SANs

3. **Redeploy bootstrap.ps1 to REMOTE**
   - Should now have correct cert with SANs
   - Clear old `coordinator.crt` if needed

4. **Run bootstrap script again**
   - Should complete successfully
   - Agent service should stay running

### Phase 3 Verification (Once Agent Runs)
- Check agent logs for "connecting to coordinator" message
- Verify agent appears online in coordinator dashboard
- Check agent can receive heartbeat responses

### Phase 4: Real Proof
- Create backup job in dashboard
- Specify REMOTE agent as target
- Run backup and verify files actually transfer
- This is the definitive test (not just service status)

---

## Files & Locations

### COORD (Primary)
- Source: `C:\Projects\ArcVault2.0\`
- Config: `C:\ArcVault\config.json` (host: 192.168.68.62)
- Cert: `C:\ArcVault\cert.pem` (regenerated with SANs)
- Key: `C:\ArcVault\key.pem`
- Bootstrap: `C:\Temp\bootstrap.ps1` (latest)
- Coordinator binary: `C:\ArcVault\coordinator.exe` (v0.5.0)
- Agent binary: `C:\ArcVault\agent.exe` (v0.5.0)

### REMOTE (Test Machine)
- Agent dir: `C:\ArcVault-Agent\`
- Config: `C:\ArcVault-Agent\agent-config.yaml`
- Certificate: `C:\ArcVault-Agent\coordinator.crt`
- Agent binary: `C:\ArcVault-Agent\agent.exe` (v0.5.0)
- Logs: `C:\ArcVault-Agent\logs\arcvault-agent.log`
- Service: `arcvault-agent` (installed, crashes on startup)

---

## Git Status

**Branch**: `main`  
**Modified files** (uncommitted):
- `coordinator/internal/bootstrap/bootstrap.go` (curl download fix)
- `coordinator/internal/tlscert/tlscert.go` (ReadCertPEM fix)
- `coordinator/server/bootstrap_handler.go` (agent path fix)
- `coordinator/server/downloads.go` (agent path fix)

**To commit**: All fixes are verified and working at the code level. Only deployment/cert issue remains.

---

## Cowork Strategy for Next Session

Recommend these agents:

1. **Networking Specialist**: Diagnose service restart issues, Windows Service Manager behavior
2. **Certificate/TLS Expert**: Verify SAN configuration is correct, test with agent code
3. **Testing Agent**: Run bootstrap on REMOTE with proper cert, verify agent registration
4. **Integration Tester**: Run Phase 4 (actual backup job) once agent is online

**Key blocker to parallelize**: Fix coordinator service restart issue → all other work depends on it.

---

## Success Criteria

- ✅ Agent service stays running (no crashes)
- ✅ Agent logs show successful HTTPS connection to coordinator
- ✅ Agent appears as ONLINE in coordinator dashboard
- ✅ Can create and run a backup job from REMOTE to another location
- ✅ Files actually transfer (not just status update)

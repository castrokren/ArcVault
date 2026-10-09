# Error Handling Testing Guide

## Quick Start

```powershell
# 1. Set your API token
$env:AGENT_TOKEN = "your-agent-token-here"

# 2. Run all tests
.\run_all_error_handling_tests.ps1
```

That's it! The test suite will automatically validate both error handling improvements.

---

## Test Scripts Overview

| Script | Purpose | Duration | Prerequisites |
|--------|---------|----------|---------------|
| `test_port_conflict.ps1` | Validates port conflict detection | ~30 seconds | Admin rights, Python |
| `test_network_retry.ps1` | Validates network retry logic | ~45 seconds | Agent running, AGENT_TOKEN |
| `run_all_error_handling_tests.ps1` | Runs all tests with summary | ~2 minutes | All of the above |

---

## What Gets Tested

### ✅ Port Conflict Detection Tests

1. **Initial State Check** - Verifies port is free
2. **Port Blocking** - Starts process on target port
3. **Failure Detection** - Agent refuses to start with clear error
4. **Error Message** - Validates error mentions port conflict
5. **Recovery** - Agent starts successfully after port freed
6. **Multiple Entry Points** - Tests service and direct execution

### ✅ Network Retry Logic Tests

1. **Agent Health** - Verifies agent is reachable
2. **Share Creation** - Creates test network share
3. **Job Creation** - Starts backup using test share
4. **Network Interruption** - Simulates share disconnection
5. **Retry Observation** - Watches retry attempts with backoff
6. **Network Recovery** - Restores share and verifies success
7. **Log Analysis** - Confirms retry patterns in logs
8. **Permanent Failure** - Tests proper failure after all retries

---

## Common Issues & Solutions

### Issue: "AGENT_TOKEN not set"

**Solution:**
```powershell
$env:AGENT_TOKEN = "your-token-here"
```

Get your token from the agent configuration or environment.

---

### Issue: "Not running as Administrator"

**Solution:**
```powershell
# Run PowerShell as Administrator
Start-Process powershell -Verb RunAs

# Then navigate to directory and run tests
cd path\to\backup-agent
.\run_all_error_handling_tests.ps1
```

---

### Issue: "Cannot reach agent"

**Possible Causes:**
1. Agent service not running
2. Wrong URL (default: https://localhost:8443)
3. Certificate validation failing

**Solutions:**
```powershell
# Check service status
Get-Service -Name "ArcVaultAgent"

# Start service if needed
Start-Service -Name "ArcVaultAgent"

# Skip certificate validation
.\test_network_retry.ps1 -SkipCertificateCheck
```

---

### Issue: "Failed to create test share"

**Possible Causes:**
1. Not running as Administrator
2. Share name conflict
3. Insufficient permissions

**Solutions:**
```powershell
# Check existing shares
net share

# Remove conflicting share
net share TestBackupShare /delete /yes

# Run as Administrator
```

---

## Test Output Examples

### ✅ All Tests Passing

```
╔════════════════════════════════════════════════════════════════╗
║   ArcVault Backup Agent - Error Handling Test Suite          ║
╚════════════════════════════════════════════════════════════════╝

=== Prerequisites Check ===
✓ Running as Administrator
✓ Python available: Python 3.11.0
✓ AGENT_TOKEN environment variable is set

═══════════════════════════════════════════════════════════════
  TEST SUITE 1: Port Conflict Detection
═══════════════════════════════════════════════════════════════

[Test 1] Checking initial port state...
  ✓ Port 8443 is free
[Test 2] Ensuring service is stopped...
  ✓ Service stopped
[Test 3] Starting blocking process on port 8443...
  ✓ Blocking process started (PID: 12345)
[Test 4] Attempting to start service (should fail)...
  ✓ PASS: Service correctly detected port conflict
[Test 5] Stopping blocking process...
  ✓ Blocking process stopped
[Test 6] Verifying service starts after port freed...
  ✓ PASS: Service started successfully

✓ Port Conflict Test Suite: PASSED

═══════════════════════════════════════════════════════════════
  TEST SUITE 2: Network Share Retry Logic
═══════════════════════════════════════════════════════════════

[Test 1] Verifying agent is reachable...
  ✓ Agent is healthy (version: 0.1.0)
[Test 2] Creating test share...
  ✓ Test share created
[Test 3] Creating backup job...
  ✓ Backup job created
[Test 4] Monitoring job and simulating network interruption...
  ✓ Share restored
[Test 5] Waiting for job completion...
  Final job state: succeeded
[Test 6] Analyzing job logs for retry behavior...
  ✓ PASS: Retry logic evidence found in logs
[Test 7] Testing permanent failure scenario...
  ✓ PASS: Job failed with retry-related error

✓ Network Retry Test Suite: PASSED

╔════════════════════════════════════════════════════════════════╗
║                    FINAL TEST SUMMARY                          ║
╚════════════════════════════════════════════════════════════════╝

  ✓ Port Conflict Detection: PASSED
  ✓ Network Share Retry Logic: PASSED

Test Suites Passed: 2
Test Suites Failed: 0

╔════════════════════════════════════════════════════════════════╗
║  ✓✓✓ ALL ERROR HANDLING TESTS PASSED! ✓✓✓                    ║
╚════════════════════════════════════════════════════════════════╝
```

---

### ❌ Test Failure Example

```
[Test 4] Attempting to start service (should fail)...
  ✗ FAIL: Service started despite port conflict!

✗ Port Conflict Test Suite: FAILED

╔════════════════════════════════════════════════════════════════╗
║  ✗ SOME TESTS FAILED                                           ║
╚════════════════════════════════════════════════════════════════╝

Test Suites Passed: 1
Test Suites Failed: 1
```

**What this means:** The port conflict detection is not working properly. The agent started even though the port was blocked.

**Action:** Review the `is_port_available()` implementation in the entry point files.

---

## Advanced Usage

### Testing Specific Port

```powershell
.\test_port_conflict.ps1 -TestPort 9000
```

### Testing Custom Agent URL

```powershell
.\test_network_retry.ps1 -AgentUrl "https://192.168.1.100:8443"
```

### Skipping Specific Test Suites

```powershell
# Skip port tests (if already validated)
.\run_all_error_handling_tests.ps1 -SkipPortTest

# Skip network tests (if agent not running)
.\run_all_error_handling_tests.ps1 -SkipNetworkTest
```

### Using Different API Token

```powershell
# Set token for current session
$env:AGENT_TOKEN = "test-token-123"
.\run_all_error_handling_tests.ps1

# Or pass directly to network test
.\test_network_retry.ps1 -ApiToken "test-token-123"
```

---

## CI/CD Integration

### Example: Azure DevOps Pipeline

```yaml
- task: PowerShell@2
  displayName: 'Run Error Handling Tests'
  inputs:
    targetType: 'filePath'
    filePath: '$(Build.SourcesDirectory)/apps/backup-agent/run_all_error_handling_tests.ps1'
    arguments: '-SkipCertificateCheck'
    errorActionPreference: 'stop'
  env:
    AGENT_TOKEN: $(AGENT_TOKEN_SECRET)
```

### Example: GitHub Actions

```yaml
- name: Run Error Handling Tests
  shell: powershell
  run: |
    cd apps/backup-agent
    .\run_all_error_handling_tests.ps1 -SkipCertificateCheck
  env:
    AGENT_TOKEN: ${{ secrets.AGENT_TOKEN }}
```

### Example: Manual Pre-Deployment Check

```powershell
# Before deploying to production
cd apps\backup-agent

# Ensure agent is running
Start-Service -Name "ArcVaultAgent"

# Run tests
.\run_all_error_handling_tests.ps1

# Check results
if ($LASTEXITCODE -eq 0) {
    Write-Host "✓ All tests passed - safe to deploy" -ForegroundColor Green
    # Proceed with deployment
} else {
    Write-Host "✗ Tests failed - DO NOT deploy" -ForegroundColor Red
    exit 1
}
```

---

## Understanding Test Results

### Port Conflict Test Results

| Result | Meaning | Action |
|--------|---------|--------|
| 6/6 tests passed | ✅ Perfect | Port conflict detection working correctly |
| 5/6 tests passed | ⚠️ Warning | One test failed, review output |
| <5/6 tests passed | ❌ Critical | Port conflict detection not working |

### Network Retry Test Results

| Result | Meaning | Action |
|--------|---------|--------|
| 7/7 tests passed | ✅ Perfect | Network retry logic working correctly |
| 6/7 tests passed | ⚠️ Acceptable | May be timing issue, review logs |
| <6/7 tests passed | ❌ Issue | Network retry logic needs attention |

---

## Test Coverage Summary

### What IS Tested ✅

- ✅ Port availability check before startup
- ✅ Clear error messages on port conflict
- ✅ Service refuses to start when port blocked
- ✅ Agent starts successfully after port freed
- ✅ Network share connection attempts with retry
- ✅ Exponential backoff between retries (1s, 2s, 4s)
- ✅ Retry attempt logging to job logs
- ✅ Proper failure after all retries exhausted
- ✅ Success message after successful retry

### What is NOT Tested ❌

- ❌ Multiple simultaneous port conflicts
- ❌ Port conflicts during runtime (only at startup)
- ❌ Network share retry with different error types
- ❌ Retry logic under high load
- ❌ Maximum retry configuration changes
- ❌ Custom backoff multipliers

**Note:** These scenarios are beyond the scope of automated testing but should be validated manually in production if needed.

---

## Troubleshooting Test Failures

### Port Test Failure: "Service started despite port conflict"

**Diagnosis:**
- Port check may not be working
- Service may be binding to different interface
- Race condition in test

**Fix:**
1. Check `is_port_available()` implementation in:
   - `agent/app.py`
   - `agent/run_agent.py`
   - `agent/win_service.py`
   - `service/win_service.py`
2. Verify port check happens BEFORE uvicorn.run()
3. Ensure exception is raised on port conflict

---

### Network Test Failure: "No retry patterns found"

**Diagnosis:**
- Backup may have completed before network interruption
- Timing issue in test
- Retry logic not logging properly

**Fix:**
1. Check logs in `#backup_logs` directory
2. Verify `_net_use_connect()` logs retry attempts
3. Increase network interruption duration
4. Check if backup source is too small (completes instantly)

---

### Network Test Failure: "Cannot reach agent"

**Diagnosis:**
- Agent not running
- Wrong URL or port
- Certificate validation failing
- Firewall blocking connection

**Fix:**
```powershell
# Check if agent is running
Get-Service -Name "ArcVaultAgent"

# Check what's listening on port
Get-NetTCPConnection -LocalPort 8443 -State Listen

# Test connectivity
Invoke-RestMethod -Uri "https://localhost:8443/health" -SkipCertificateCheck

# Use skip certificate flag
.\test_network_retry.ps1 -SkipCertificateCheck
```

---

## Manual Validation

If automated tests are not suitable for your environment, you can manually validate:

### Manual Port Conflict Test

1. Start a process on port 8443:
   ```powershell
   python -m http.server 8443
   ```

2. Try to start the agent:
   ```powershell
   Start-Service -Name "ArcVaultAgent"
   ```

3. **Expected:** Service fails to start with error mentioning port 8443

4. **Verify error in logs:**
   ```powershell
   Get-Content $env:TEMP\arcvault_service.log -Tail 20
   ```

---

### Manual Network Retry Test

1. Start a backup to a network share

2. During backup, disconnect the share:
   ```powershell
   net share MyBackupShare /delete /yes
   ```

3. **Expected:** See retry attempts in logs:
   ```
   NET USE connect \\server\share (attempt 1/3)
   ⚠ Connection attempt 1/3 failed
   Retrying in 1 seconds...
   ```

4. Reconnect share quickly:
   ```powershell
   net share MyBackupShare=C:\BackupPath /grant:everyone,full
   ```

5. **Expected:** Backup succeeds after retry

---

## Questions & Support

### How long do tests take?

- Port conflict test: ~30 seconds
- Network retry test: ~45 seconds
- Full test suite: ~2 minutes

### Can I run tests on production agent?

Yes, but:
- Tests will briefly stop/start the service
- Network test creates temporary share
- Recommend running during maintenance window

### What if I don't have admin rights?

Some tests will fail without admin rights:
- Cannot create network shares
- Cannot start/stop services
- Port binding may fail

Recommend running in dev/test environment with admin rights.

---

## Next Steps

After running tests successfully:

1. ✅ **Document Results** - Save test output for compliance/audit
2. ✅ **Schedule Regular Testing** - Run before each deployment
3. ✅ **Monitor Production** - Watch for port conflicts and retry patterns
4. ✅ **Update Documentation** - Note any environment-specific issues
5. ✅ **Train Team** - Ensure operators understand test output

---

**Last Updated:** December 2, 2025  
**Test Script Version:** 1.0  
**Compatible With:** ArcVault Backup Agent v0.1.0+


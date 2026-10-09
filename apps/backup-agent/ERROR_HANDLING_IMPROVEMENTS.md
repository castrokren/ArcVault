# Error Handling Improvements

## Overview
This document summarizes the error handling enhancements made to address critical gaps in the backup agent application.

---

## 1. Port Conflict Detection

### Issue
The application would start silently even if port 8443 (or the configured port) was already in use by another process, leading to silent failures and difficult-to-diagnose issues.

### Solution
Added port availability checks before starting uvicorn across all entry points:

#### Implementation

Added `is_port_available()` function that attempts to bind to the port before starting the server:

```python
def is_port_available(port: int) -> bool:
    """Check if a port is available for binding."""
    with socket.socket(socket.AF_INET, socket.SOCK_STREAM) as s:
        try:
            s.bind(('0.0.0.0', port))
            return True
        except OSError:
            return False
```

#### Files Modified

1. **`agent/app.py`** (line ~497)
   - Added `is_port_available()` function
   - Added port check in `__main__` block before `uvicorn.run()`
   - Raises `RuntimeError` with clear message if port is in use

2. **`agent/run_agent.py`** (line ~12)
   - Added `_is_port_available()` function
   - Added socket import
   - Added port check in `main()` before starting uvicorn
   - Raises `RuntimeError` with clear message if port is in use

3. **`agent/win_service.py`** (line ~57)
   - Added `_is_port_available()` function
   - Added socket import
   - Added port check in `_run_server()` before creating uvicorn config
   - Logs error and raises `RuntimeError` for service event log

4. **`service/win_service.py`** (line ~57)
   - Added `_is_port_available()` function
   - Added socket import
   - Added port check in `_run_server()` before creating uvicorn config
   - Logs error and raises `RuntimeError` for service event log

### Benefits

- **Early Detection**: Port conflicts are detected immediately at startup
- **Clear Error Messages**: Users see explicit error message: "Port {port} already in use. Cannot start agent."
- **No Silent Failures**: Application fails fast with clear diagnostic information
- **Service Integration**: Windows service logs port conflicts to event log for troubleshooting

### Testing

To test the port conflict detection:

```powershell
# Start the agent normally
python -m agent.run_agent

# In another terminal, try to start a second instance
python -m agent.run_agent
# Should see: RuntimeError: Port 8443 already in use. Cannot start agent.
```

---

## 2. Network Share Connection Retry Logic

### Issue
Network share connections using `net use` would fail silently without any retry attempts. Transient network issues or temporary unavailability would cause backups to fail immediately.

### Solution
Enhanced `_net_use_connect()` function with exponential backoff retry logic:

#### Implementation

```python
def _net_use_connect(
    share: str, 
    username: Optional[str], 
    password: Optional[str], 
    domain: Optional[str], 
    emit: LogSink, 
    max_retries: int = 3
) -> None:
    """
    Connect to network share with retry logic and exponential backoff.
    
    Args:
        share: UNC path to the share
        username: Username for authentication
        password: Password for authentication
        domain: Domain for authentication
        emit: Logging function
        max_retries: Maximum number of connection attempts (default: 3)
    
    Raises:
        RuntimeError: If connection fails after all retry attempts
    """
    if not share:
        return
    
    user = None
    if username:
        user = f"{domain}\\{username}" if domain else username
    args = ["net", "use", share]
    if password:
        args.append(password)
    if user:
        args.append(f"/user:{user}")
    
    for attempt in range(max_retries):
        emit(f"NET USE connect {share} (attempt {attempt + 1}/{max_retries})")
        result = subprocess.run(args, capture_output=True, text=True)
        
        if result.returncode == 0:
            emit(f"✓ Connected to {share}")
            return
        
        # Log the error
        error_msg = result.stderr.strip() or result.stdout.strip() or "Unknown error"
        emit(f"⚠ Connection attempt {attempt + 1}/{max_retries} failed: {error_msg}")
        
        # If not the last attempt, wait before retrying
        if attempt < max_retries - 1:
            wait_time = 2 ** attempt  # Exponential backoff: 1s, 2s, 4s
            emit(f"Retrying in {wait_time} seconds...")
            time.sleep(wait_time)
    
    # All retries exhausted
    raise RuntimeError(f"Failed to connect to {share} after {max_retries} attempts")
```

#### Files Modified

1. **`agent/engine/runner.py`** (line ~24)
   - Added `time` import for exponential backoff delays
   - Enhanced `_net_use_connect()` with retry logic
   - Added comprehensive error logging at each retry attempt
   - Implemented exponential backoff (1s, 2s, 4s delays)
   - Added error handling in `run_backup()` to catch and report connection failures

### Features

- **Configurable Retries**: Default 3 attempts, can be adjusted via `max_retries` parameter
- **Exponential Backoff**: Wait times increase exponentially (1s, 2s, 4s)
- **Comprehensive Logging**: Each attempt is logged with clear status indicators (✓, ⚠)
- **Error Details**: Captures and logs stderr/stdout from `net use` command
- **Graceful Failure**: After exhausting retries, raises clear RuntimeError with summary

### Retry Sequence

For a network share connection:
1. **Attempt 1**: Immediate connection try
   - On failure: Wait 1 second
2. **Attempt 2**: Second connection try
   - On failure: Wait 2 seconds  
3. **Attempt 3**: Final connection try
   - On failure: Raise RuntimeError with details

Total maximum wait time: 3 seconds before final failure

### Benefits

- **Resilience**: Handles transient network issues automatically
- **Visibility**: Clear logging of each retry attempt
- **Fast Recovery**: Successful connection on early retry avoids backup failure
- **Smart Backoff**: Exponential delays reduce network load during issues
- **Proper Failure**: After retries exhausted, fails with clear error message

### Error Handling Integration

Enhanced the backup runner to properly catch and handle connection failures:

```python
try:
    if cfg.source_credentials and src_share:
        _net_use_connect(
            src_share, 
            cfg.source_credentials.username, 
            cfg.source_credentials.password, 
            cfg.source_credentials.domain, 
            emit_and_log
        )
    if cfg.destination_credentials and dst_share:
        _net_use_connect(
            dst_share, 
            cfg.destination_credentials.username, 
            cfg.destination_credentials.password, 
            cfg.destination_credentials.domain, 
            emit_and_log
        )
except RuntimeError as exc:
    # Network share connection failed after retries
    emit_and_log(f"ERROR: Network share connection failed: {exc}")
    return "failed", str(exc)
```

### Testing

To test the retry logic:

```powershell
# Test with invalid credentials (should retry and fail)
# Monitor logs for retry attempts:
# - "NET USE connect \\share (attempt 1/3)"
# - "⚠ Connection attempt 1/3 failed: ..."
# - "Retrying in 1 seconds..."
# - etc.

# Test with temporarily unavailable share
# - Should see retry attempts
# - If share becomes available during retry window, connection succeeds
```

---

## Log Output Examples

### Successful Connection After Retry

```
[2025-12-02 10:15:30] NET USE connect \\server\backup (attempt 1/3)
[2025-12-02 10:15:31] ⚠ Connection attempt 1/3 failed: The network name cannot be found.
[2025-12-02 10:15:31] Retrying in 1 seconds...
[2025-12-02 10:15:32] NET USE connect \\server\backup (attempt 2/3)
[2025-12-02 10:15:33] ✓ Connected to \\server\backup
```

### Failed Connection After All Retries

```
[2025-12-02 10:15:30] NET USE connect \\server\backup (attempt 1/3)
[2025-12-02 10:15:31] ⚠ Connection attempt 1/3 failed: The network name cannot be found.
[2025-12-02 10:15:31] Retrying in 1 seconds...
[2025-12-02 10:15:32] NET USE connect \\server\backup (attempt 2/3)
[2025-12-02 10:15:33] ⚠ Connection attempt 2/3 failed: The network name cannot be found.
[2025-12-02 10:15:33] Retrying in 2 seconds...
[2025-12-02 10:15:35] NET USE connect \\server\backup (attempt 3/3)
[2025-12-02 10:15:36] ⚠ Connection attempt 3/3 failed: The network name cannot be found.
[2025-12-02 10:15:36] ERROR: Network share connection failed: Failed to connect to \\server\backup after 3 attempts
[2025-12-02 10:15:36] Job failed: Failed to connect to \\server\backup after 3 attempts
```

### Port Conflict Error

```
ERROR: Port 8443 already in use. Cannot start agent.
Traceback (most recent call last):
  File "agent\run_agent.py", line 58, in <module>
    main()
  File "agent\run_agent.py", line 55, in main
    raise RuntimeError(f"Port {port} already in use. Cannot start agent.")
RuntimeError: Port 8443 already in use. Cannot start agent.
```

---

## Summary

### Changes Made

✅ **Port conflict detection** added to 4 entry points:
   - `agent/app.py`
   - `agent/run_agent.py`
   - `agent/win_service.py`
   - `service/win_service.py`

✅ **Network share retry logic** with exponential backoff:
   - Enhanced `_net_use_connect()` in `agent/engine/runner.py`
   - Added comprehensive logging
   - Integrated error handling in backup runner

### Impact

- **Reduced Silent Failures**: Both issues now provide clear, actionable error messages
- **Improved Reliability**: Network share retry logic handles transient failures automatically
- **Better Diagnostics**: Enhanced logging helps troubleshoot connection issues
- **Production Readiness**: Application now handles common failure scenarios gracefully

### Backward Compatibility

All changes are backward compatible:
- Port check happens before any state changes
- Retry logic is transparent to callers (default 3 retries)
- Error messages are clear and actionable
- No API changes to existing functions

---

## Future Enhancements

Consider adding:
1. **Configurable Retry Parameters**: Environment variables for max_retries and backoff multiplier
2. **Health Check Endpoint**: Include port availability check in `/health` endpoint
3. **Metrics**: Track retry success rates and failure reasons
4. **Circuit Breaker**: Skip retries for known-bad shares to fail fast
5. **Port Range Fallback**: Automatically try alternate ports if configured port is in use

---

## Automated Testing

### Test Scripts Provided

Three automated test scripts are included to validate the error handling implementations:

#### 1. `test_port_conflict.ps1`

Comprehensive test for port conflict detection:
- ✅ Verifies port is free initially
- ✅ Blocks port with test process
- ✅ Attempts to start agent (should fail with clear error)
- ✅ Verifies error message mentions port conflict
- ✅ Cleans up and verifies agent can start after port freed
- ✅ Tests both service and direct execution modes

**Usage:**
```powershell
# Run with defaults (port 8443, ArcVaultAgent service)
.\test_port_conflict.ps1

# Specify custom port and service name
.\test_port_conflict.ps1 -TestPort 9000 -ServiceName "MyService"
```

**Expected Output:**
```
=== Port Conflict Detection Test ===
[Test 1] Checking initial port state...
  ✓ Port 8443 is free
[Test 2] Ensuring service is stopped...
  ✓ Service stopped
[Test 3] Starting blocking process on port 8443...
  ✓ Blocking process started (PID: 12345)
[Test 4] Attempting to start service (should fail)...
  ✓ PASS: Service correctly detected port conflict
  Error message: Port 8443 already in use. Cannot start agent.
[Test 5] Stopping blocking process...
  ✓ Blocking process stopped
[Test 6] Verifying service starts after port freed...
  ✓ PASS: Service started successfully

=== Test Results ===
Tests Passed: 6
Tests Failed: 0

✓ ALL TESTS PASSED - Port conflict detection is working correctly!
```

---

#### 2. `test_network_retry.ps1`

Comprehensive test for network share retry logic:
- ✅ Verifies agent is reachable
- ✅ Creates test network share
- ✅ Starts backup job using the share
- ✅ Simulates network interruption (removes share mid-backup)
- ✅ Restores share (simulates network recovery)
- ✅ Analyzes job logs for retry patterns
- ✅ Tests permanent failure scenario
- ✅ Verifies proper error messages after all retries exhausted

**Usage:**
```powershell
# Run with defaults (localhost:8443, uses AGENT_TOKEN env var)
.\test_network_retry.ps1

# Specify custom agent URL and skip certificate validation
.\test_network_retry.ps1 -AgentUrl "https://myagent:8443" -SkipCertificateCheck

# Provide token explicitly
$env:AGENT_TOKEN = "your-token-here"
.\test_network_retry.ps1
```

**Expected Output:**
```
=== Network Share Retry Logic Test ===
[Test 1] Verifying agent is reachable...
  ✓ Agent is healthy (version: 0.1.0)
[Test 2] Creating test share...
  ✓ Test share created: \\localhost\TestBackupShare_abc123
[Test 3] Creating backup job...
  ✓ Backup job created: 1a2b3c4d5e6f7g8h
[Test 4] Monitoring job and simulating network interruption...
  Simulating network interruption (removing share)...
  ⚠ Share disconnected
  Waiting for retry attempts (5 seconds)...
  Restoring share (simulating network recovery)...
  ✓ Share restored
[Test 5] Waiting for job completion...
  Status: running (elapsed: 0s)
  Status: succeeded (elapsed: 8s)
  Final job state: succeeded
[Test 6] Analyzing job logs for retry behavior...
  ✓ Found retry pattern 'attempt \d+/\d+' (2 times)
  ✓ Found retry pattern 'Retrying in \d+ seconds' (2 times)
  ✓ PASS: Retry logic evidence found in logs
[Test 7] Testing permanent failure scenario...
  ✓ PASS: Job failed with retry-related error
    Error: Failed to connect to \\localhost\TestBackupShare_abc123 after 3 attempts

=== Test Results ===
Tests Passed: 7
Tests Failed: 0

✓ TESTS PASSED - Network retry logic is working!
```

---

#### 3. `run_all_error_handling_tests.ps1`

Master test runner that executes all error handling test suites:
- ✅ Checks prerequisites (admin rights, Python, AGENT_TOKEN)
- ✅ Runs port conflict tests
- ✅ Runs network retry tests
- ✅ Provides comprehensive summary with visual formatting
- ✅ Exits with proper status codes for CI/CD integration

**Usage:**
```powershell
# Run all tests
.\run_all_error_handling_tests.ps1

# Skip specific test suites
.\run_all_error_handling_tests.ps1 -SkipPortTest
.\run_all_error_handling_tests.ps1 -SkipNetworkTest

# Skip certificate validation (for self-signed certs)
.\run_all_error_handling_tests.ps1 -SkipCertificateCheck
```

**Expected Output:**
```
╔════════════════════════════════════════════════════════════════╗
║   ArcVault Backup Agent - Error Handling Test Suite          ║
║                                                                ║
║   This suite validates:                                        ║
║   1. Port conflict detection                                   ║
║   2. Network share retry logic with exponential backoff        ║
╚════════════════════════════════════════════════════════════════╝

=== Prerequisites Check ===
✓ Running as Administrator
✓ Python available: Python 3.11.0
✓ AGENT_TOKEN environment variable is set

═══════════════════════════════════════════════════════════════
  TEST SUITE 1: Port Conflict Detection
═══════════════════════════════════════════════════════════════

[... port tests run ...]

✓ Port Conflict Test Suite: PASSED

═══════════════════════════════════════════════════════════════
  TEST SUITE 2: Network Share Retry Logic
═══════════════════════════════════════════════════════════════

[... network tests run ...]

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
║                                                                ║
║  Your error handling implementations are working correctly!    ║
║  - Port conflict detection: ✓                                  ║
║  - Network retry logic: ✓                                      ║
╚════════════════════════════════════════════════════════════════╝
```

---

### Running the Tests

**Quick Start:**
```powershell
# Navigate to backup-agent directory
cd apps\backup-agent

# Set API token (required for network tests)
$env:AGENT_TOKEN = "your-agent-token"

# Run all tests
.\run_all_error_handling_tests.ps1
```

**Individual Tests:**
```powershell
# Test only port conflict detection
.\test_port_conflict.ps1

# Test only network retry logic
.\test_network_retry.ps1 -SkipCertificateCheck
```

**CI/CD Integration:**
```powershell
# Exit code 0 = all tests passed, non-zero = failures
.\run_all_error_handling_tests.ps1
if ($LASTEXITCODE -eq 0) {
    Write-Host "Tests passed, proceeding with deployment"
} else {
    Write-Host "Tests failed, blocking deployment"
    exit 1
}
```

---

### Test Requirements

**Prerequisites:**
- Windows OS with PowerShell 5.1+
- Administrator privileges (for share creation and service management)
- Python 3.x (for port blocking test)
- Agent service installed OR agent source code available
- `AGENT_TOKEN` environment variable set (for network tests)

**Permissions Needed:**
- Create/delete network shares (`net share` command)
- Start/stop Windows services
- Bind to network ports
- Create directories and files

---

**Implementation Date**: December 2, 2025  
**Author**: AI Assistant  
**Status**: ✅ Complete - All changes tested and verified with automated test suite


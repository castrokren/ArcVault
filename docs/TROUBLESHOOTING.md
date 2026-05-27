# ArcVault Troubleshooting Guide

## Dashboard Not Loading / Blank Page

### Root Cause
The dashboard is hardcoded to connect to `localhost:8080`. If the coordinator runs on a different port, the API calls will fail and the page will appear blank.

### Solution
**Ensure the coordinator runs on port 8080** in your `config.json`:

```json
{
  "port": 8080,
  "database_path": "C:/Users/kren/.arcvault/arcvault.db",
  "admin_token": "your_token_here",
  "jwt_secret": "your_jwt_secret",
  "environment": "development",
  "alert_history_retention_days": 30
}
```

### Verification
```bash
curl http://localhost:8080/health
# Should return: {"status":"ok"}
```

---

## Agent Not Showing in Dashboard UI

### Root Cause
The agent is registered and online in the database, but the UI requires authentication to display it.

### Solution
The dashboard requires three localStorage entries to authenticate:

**Option 1: Using Browser DevTools Storage Tab (Easiest)**
1. Open DevTools (F12)
2. Go to **Application** → **Local Storage** → `http://localhost:8080`
3. Add these three entries:

| Key | Value |
|-----|-------|
| `arcvault_jwt` | `eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VyX2lkIjoxLCJ1c2VybmFtZSI6ImFkbWluIiwicm9sZSI6ImFkbWluIiwibXVzdF9jaGFuZ2UiOnRydWUsInN1YiI6IjEiLCJleHAiOjE3Nzk3NDMyMjIsImlhdCI6MTc3OTY1NjgyMn0.NVJ4r2FOlDikaYtYz5beWUOm441SG82hQ3oVQ8_wOFc` |
| `arcvault_remember_me` | `1` |
| `arcvault_user` | `{"username":"admin","role":"admin","must_change_password":true}` |

4. Refresh the page (F5)

**Option 2: Using Login Form**
1. Navigate to `http://localhost:8080/#/login`
2. Enter credentials:
   - Username: `admin`
   - Password: `changeme`
3. You'll be prompted to change your password on first login

### Verification
After authentication:
- Agent should appear in the Agents list
- Status should show as **ONLINE**
- WebSocket indicator should show **● Live** in top-right

---

## Agent Connection Errors

### Runner Fixes Verification
All runner.go fixes are verified working:

| Fix | Status | Details |
|-----|--------|---------|
| SEC-001: URL Injection | ✅ | Using `url.URL` + `url.Values.Encode()` |
| SEC-002: HTTPS Enforcement | ✅ | Validates HTTPS (allows localhost for testing) |
| SEC-003: Stop() Panic | ✅ | Using `sync.Once` for concurrent safety |
| PERF-001: HTTP Timeouts | ✅ | 30-second timeout configured |
| PERF-002: Connection Pooling | ✅ | HTTP transport with idle conn pooling |
| CORR-001: JSON Errors | ✅ | Proper error handling in marshal/unmarshal |
| CORR-002: Response Body Reading | ✅ | Error responses include body text |
| CORR-003: Job Validation | ✅ | Invalid jobs skipped with logging |
| CORR-004: Error Classification | ✅ | 4xx vs 5xx distinguished |

### Test Suite
```bash
go test ./agent/runner/... -v -count=1
# Expected: 13 tests PASS in ~7-10 seconds
```

---

## Federation Connection Errors (Non-Critical)

### Symptoms
Browser console shows:
```
GET http://localhost:443/api/rollback-available net::ERR_CONNECTION_REFUSED
WebSocket connection to 'ws://localhost:443/ws?token=...' failed
```

### Root Cause
The dashboard attempts to connect to a federation coordinator on `localhost:443` for high-availability features. This is optional and non-critical for single-coordinator deployments.

### Solution
These errors can be safely ignored for development/testing. In production:
- Configure federation in `config.json` if using multiple coordinators
- Or disable federation features if not needed

---

## Quick Start

### 1. Start Coordinator
```bash
cd C:\Projects\ArcVault2.0
go build -o coordinator.exe ./coordinator
.\coordinator.exe start
```

### 2. Start Agent
```bash
cd C:\Projects\ArcVault2.0
go build -o agent.exe ./agent
$env:AGENT_CONFIG=.\agent-config.yaml
.\agent.exe
```

### 3. Access Dashboard
```
http://localhost:8080
```

### 4. Get JWT Token for Testing
```bash
curl -X POST http://localhost:8080/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"changeme"}'
```

---

## Common Issues

### Config JSON Invalid JSON Error
**Problem:** `invalid character 'ï' looking for beginning of value`

**Solution:** Ensure JSON file uses UTF-8 without BOM. Use bash to create:
```bash
cat > config.json << 'EOF'
{
  "port": 8080,
  "database_path": "C:/Users/kren/.arcvault/arcvault.db",
  "admin_token": "token",
  "jwt_secret": "secret",
  "environment": "development"
}
EOF
```

### Port Already in Use
**Problem:** `Only one usage of each socket address... normally permitted`

**Solution:** Change to different port in config.json and ensure dashboard URLs match

### Agent Registration Fails with HTTPS Error
**Problem:** `http: server gave HTTP response to HTTPS client`

**Solution:** Update `agent-config.yaml` to use `http://localhost:...` for testing, or use HTTPS in production

---

## Performance Notes

### HTTP Client Configuration
The runner now includes:
- **30-second request timeout** (prevents indefinite hangs)
- **Connection pooling** with 10 idle connections max
- **90-second idle timeout** for keep-alive connections
- **Safe URL encoding** via `url.URL` and `url.Values`

These improvements prevent:
- Hung requests when coordinator becomes unresponsive
- URL injection attacks via unsanitized parameters
- Resource exhaustion from connection leaks

---

## Testing

### Unit Tests
```bash
go test ./agent/runner/... -v
```

### Integration Test (Full Stack)
1. Start coordinator on port 8080
2. Start agent with valid config
3. Verify in API:
```bash
curl http://localhost:8080/api/agents \
  -H "Authorization: Bearer <admin_token>"
```

Should return agent with `"status":"online"`


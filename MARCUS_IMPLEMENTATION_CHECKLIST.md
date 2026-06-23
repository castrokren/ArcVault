# Phase 1 Implementation Checklist for Marcus Chen

## Pre-Implementation Setup

- [ ] Clone/pull latest from main branch
- [ ] Read PHASE_1_SUMMARY.md (2 min overview)
- [ ] Read PHASE_1_IMPLEMENTATION_PLAN.md (detailed context)
- [ ] Read PHASE_1_FILE_CHANGES.md (exact code changes)
- [ ] Set up local environment variables for testing:
  ```bash
  export ARCVAULT_ADMIN_TOKEN=$(openssl rand -hex 32)
  export ARCVAULT_JWT_SECRET=$(openssl rand -hex 32)
  ```

---

## Day 1: Configuration & Validation

### Task 1.1: Update `coordinator/config/config.go`

**Imports**
- [ ] Add `"log"` import (line 7)
- [ ] Add `"strings"` import (line 8)

**Add ValidateAllowedOrigins() Method**
- [ ] Add method after generateSecret() function (after line 127)
- [ ] Method checks for wildcard "*"
- [ ] Method validates HTTPS or localhost
- [ ] Method handles empty list appropriately

**Update Load() Function (lines 89-117)**
- [ ] Check ARCVAULT_ADMIN_TOKEN env var
- [ ] Check ARCVAULT_JWT_SECRET env var
- [ ] Call ValidateAllowedOrigins()
- [ ] Set defaults for development mode
- [ ] Validate production requirements
- [ ] Add appropriate log messages

**Update Save() Function (lines 74-87)**
- [ ] Create sanitized copy
- [ ] Clear AdminToken before saving
- [ ] Clear JWTSecret before saving
- [ ] Add log messages
- [ ] Write sanitized config only

**Verification**
```bash
go fmt coordinator/config/config.go
go build -o /tmp/test coordinator/  # Should compile
```

### Task 1.2: Update `coordinator/cmd/commands.go`

**Update InitCommand() Function (lines 24-99)**
- [ ] Generate adminToken for display (not saved)
- [ ] Generate jwtSecret for display (not saved)
- [ ] Save config with empty AdminToken
- [ ] Save config with empty JWTSecret
- [ ] Display setup instructions with tokens
- [ ] Show warning about not committing to git
- [ ] Show warning about env vars required

**Verification**
```bash
go fmt coordinator/cmd/commands.go
go build -o /tmp/test coordinator/  # Should compile
```

### Task 1.3: Create Tests for Day 1 Work

**Create `coordinator/tests/config_validation_test.go`**
- [ ] Test ValidateAllowedOrigins_RejectsWildcard
- [ ] Test ValidateAllowedOrigins_AllowsHTTPS
- [ ] Test ValidateAllowedOrigins_AllowsLocalhost
- [ ] Test ValidateAllowedOrigins_RejectsHTTPRemote

**Create `coordinator/tests/config_env_test.go`**
- [ ] Test Load_EnvVarOverridesFile
- [ ] Test Load_FailsWithoutTokenInProduction
- [ ] Test Save_NeverWritesSensitiveFields

**Run Tests**
```bash
go test ./coordinator/tests/... -v
# All tests should pass
```

### Task 1.4: Manual Testing - Day 1

**Test Init Command**
```bash
cd /tmp
rm -f config.json  # Clean start
/path/to/coordinator init
# Should display tokens and setup instructions
cat config.json
# Verify: "admin_token": "" (empty)
# Verify: "jwt_secret": "" (empty)
```

**Test Config Loading**
```bash
export ARCVAULT_ADMIN_TOKEN="test-token-123"
export ARCVAULT_JWT_SECRET="test-secret-456"
go run coordinator/main.go start &
# Should start successfully
# Check logs: [config] AdminToken loaded from ARCVAULT_ADMIN_TOKEN env var
pkill -f "coordinator/main.go"
```

**Test Production Mode Validation**
```bash
# Create production config
cat > /tmp/test-config.json <<EOF
{
  "port": 443,
  "environment": "production",
  "allowed_origins": []
}
EOF
# Should fail on startup (no env vars + production)
```

### Day 1 Completion Checklist
- [ ] config.go: ValidateAllowedOrigins added
- [ ] config.go: Load() updated with env var logic
- [ ] config.go: Save() sanitizes tokens
- [ ] commands.go: InitCommand displays setup
- [ ] All tests pass
- [ ] Manual testing successful
- [ ] No compilation errors
- [ ] Code formatted (go fmt)

---

## Day 2: WebSocket & Dashboard

### Task 2.1: Update `coordinator/server/server.go`

**Add wsUpgrader Field**
- [ ] Add `wsUpgrader *websocket.Upgrader` to Server struct (line 53)

**Add initWebSocketUpgrader() Method**
- [ ] Add method after line 90
- [ ] Implement CheckOrigin function
- [ ] Check AllowedOrigins list
- [ ] Log accepted origins
- [ ] Log rejected origins

**Add corsOriginAllowed() Helper**
- [ ] Add method after initWebSocketUpgrader()
- [ ] Return true for empty origin (non-CORS)
- [ ] Check against whitelist
- [ ] Log rejections

**Update NewWithFS() Method**
- [ ] Add call to s.initWebSocketUpgrader() after Server init
- [ ] Call before s.registerRoutes()

**Verification**
```bash
go fmt coordinator/server/server.go
go build -o /tmp/test coordinator/  # Should compile
```

### Task 2.2: Update `coordinator/server/hub.go`

**Update Global Upgrader (lines 128-130)**
- [ ] Add CheckOrigin with fail-safe default
- [ ] Add ReadBufferSize
- [ ] Add WriteBufferSize
- [ ] Add comment about Server override

**Update handleWS() (line 166)**
- [ ] Change `upgrader.Upgrade` to `s.wsUpgrader.Upgrade`

**Update handleAgentWS() (line 214)**
- [ ] Change `upgrader.Upgrade` to `s.wsUpgrader.Upgrade`

**Verification**
```bash
go fmt coordinator/server/hub.go
go build -o /tmp/test coordinator/  # Should compile
```

### Task 2.3: Update `dashboard/src/composables/useWebSocket.js`

**Update connect() Function (lines 18-50)**
- [ ] Add origin logging in onopen
- [ ] Add error code handling in onclose
- [ ] Add code 1006 detection (CORS rejection)
- [ ] Better error messages

**Verification**
```bash
# Format check
cat dashboard/src/composables/useWebSocket.js
# No syntax errors when viewed
```

### Task 2.4: Create Tests for Day 2 Work

**Create `coordinator/tests/websocket_origin_test.go`**
- [ ] Test WebSocketUpgrade_AllowsValidOrigin
- [ ] Test WebSocketUpgrade_RejectsInvalidOrigin
- [ ] Test WebSocketUpgrade_AllowsNoOriginHeader

**Run Tests**
```bash
go test ./coordinator/tests/... -v
# All tests should pass
```

### Task 2.5: Manual Testing - Day 2

**Test WebSocket Origin Validation**
```bash
# Start coordinator with specific AllowedOrigins
export ARCVAULT_ADMIN_TOKEN="test-token-123"
export ARCVAULT_JWT_SECRET="test-secret-456"

# Edit config.json to set:
# "allowed_origins": ["https://localhost:5173"]

go run coordinator/main.go start &

# Test with curl (WebSocket):
curl -i -N \
  -H "Connection: Upgrade" \
  -H "Upgrade: websocket" \
  -H "Origin: https://localhost:5173" \
  -H "Sec-WebSocket-Key: dGhlIHNhbXBsZSBub25jZQ==" \
  -H "Sec-WebSocket-Version: 13" \
  https://localhost:443/ws
# Should return 101 Switching Protocols

# Test from disallowed origin:
curl -i -N \
  -H "Connection: Upgrade" \
  -H "Upgrade: websocket" \
  -H "Origin: https://attacker.com" \
  https://localhost:443/ws
# Should return error (connection rejected)

pkill -f "coordinator/main.go"
```

**Test CORS Headers**
```bash
go run coordinator/main.go start &

curl -i -X OPTIONS \
  -H "Origin: https://dashboard.internal.corp" \
  https://localhost:443/api/jobs
# Should return CORS headers if origin is in AllowedOrigins

curl -i -X OPTIONS \
  -H "Origin: https://attacker.com" \
  https://localhost:443/api/jobs
# Should return 403 or no CORS headers

pkill -f "coordinator/main.go"
```

### Day 2 Completion Checklist
- [ ] server.go: wsUpgrader field added
- [ ] server.go: initWebSocketUpgrader() method added
- [ ] server.go: corsOriginAllowed() helper added
- [ ] server.go: NewWithFS() calls init
- [ ] hub.go: Global upgrader updated
- [ ] hub.go: handleWS() uses s.wsUpgrader
- [ ] hub.go: handleAgentWS() uses s.wsUpgrader
- [ ] useWebSocket.js: Updated with origin logging
- [ ] All tests pass
- [ ] Manual WebSocket testing successful
- [ ] CORS validation working
- [ ] Code formatted (go fmt)

---

## Integration Testing

### Full Build Test
```bash
go build -o /tmp/coordinator ./coordinator/
echo "✓ Build successful"
```

### Full Test Suite
```bash
go test ./...
# All tests should pass
```

### Dashboard Build Test (if needed)
```bash
cd dashboard
npm run build
echo "✓ Dashboard build successful"
```

### Full Integration Test
```bash
# 1. Clean start
rm -f ~/.arcvault/arcvault.db config.json

# 2. Initialize
export ARCVAULT_ADMIN_TOKEN=$(openssl rand -hex 32)
export ARCVAULT_JWT_SECRET=$(openssl rand -hex 32)
./coordinator init

# 3. Start server
./coordinator start &

# 4. Test CORS from allowed origin
curl -H "Origin: https://dashboard.local" \
     -X OPTIONS https://localhost:443/api/agents

# 5. Test WebSocket
# (Connect dashboard, should see "WS connected" in browser console)

# 6. Stop server
pkill -f "coordinator start"

# 7. Verify config.json is clean
cat config.json | grep -E "(admin_token|jwt_secret)"
# Should show empty values: "admin_token": "", "jwt_secret": ""
```

---

## Code Review Preparation

### Before Submitting for Review
- [ ] All files formatted (go fmt ./...)
- [ ] All tests pass (go test ./...)
- [ ] No warnings from linters
- [ ] No hardcoded credentials anywhere
- [ ] No commented-out code
- [ ] Comments added for complex logic
- [ ] Imported new packages are used

### Checklist for Elena (Code Review)
Prepare this summary for Elena:

```markdown
## Phase 1 Implementation Summary

Files Modified:
1. coordinator/config/config.go
   - Added ValidateAllowedOrigins()
   - Updated Load() with env var override
   - Updated Save() to sanitize

2. coordinator/server/server.go
   - Added wsUpgrader field
   - Added initWebSocketUpgrader()
   - Added corsOriginAllowed()
   - Updated NewWithFS()

3. coordinator/server/hub.go
   - Updated global upgrader
   - Updated handleWS()
   - Updated handleAgentWS()

4. coordinator/cmd/commands.go
   - Updated InitCommand()

5. dashboard/src/composables/useWebSocket.js
   - Enhanced connect() with logging

Tests Added:
- cors_validation_test.go
- config_env_test.go
- websocket_origin_test.go

Test Results:
✓ All unit tests pass
✓ All integration tests pass
✓ Manual testing successful

Ready for: Code review, security review, deployment
```

---

## Submission Checklist

### Before Submitting PR
- [ ] All changes committed locally
- [ ] All tests pass
- [ ] Code formatted
- [ ] No merge conflicts
- [ ] Commit messages are clear
- [ ] Documentation files exist and are complete

### Files to Include in PR
- [ ] coordinator/config/config.go (modified)
- [ ] coordinator/server/server.go (modified)
- [ ] coordinator/server/hub.go (modified)
- [ ] coordinator/cmd/commands.go (modified)
- [ ] dashboard/src/composables/useWebSocket.js (modified)
- [ ] coordinator/DEPLOYMENT.md (created)
- [ ] coordinator/tests/*.go (created)

### PR Description Template
```markdown
## Phase 1 Security Remediation Implementation

Fixes P0-001, P0-002, P0-004 from SECURITY_PROPOSAL.md

### Changes
- CORS whitelist validation (P0-001)
- WebSocket origin validation (P0-002)
- Admin token from environment variables (P0-004)

### Testing
- Unit tests: All passing
- Integration tests: All passing
- Manual testing: CORS + WebSocket validated

### Deployment
- New file: DEPLOYMENT.md with production procedures
- Backward compatible
- Requires env vars in production

### Review Checklist
- [x] Code follows style guidelines
- [x] Tests added/updated
- [x] Documentation updated
- [x] No hardcoded credentials
- [x] Security review points addressed
```

---

## Troubleshooting

### Issue: "AdminToken loaded from config file" in logs
**Solution:** Env var not set. Run:
```bash
export ARCVAULT_ADMIN_TOKEN=$(openssl rand -hex 32)
export ARCVAULT_JWT_SECRET=$(openssl rand -hex 32)
```

### Issue: "AllowedOrigins must be explicitly configured"
**Solution:** Set in config.json:
```json
"allowed_origins": ["https://dashboard.internal.corp"]
```

### Issue: WebSocket connection refused
**Solution:** Check:
1. Origin header matches AllowedOrigins
2. Server is running with correct env vars
3. Logs show why origin was rejected

### Issue: "CORS origin not allowed" in logs
**Solution:** Add origin to AllowedOrigins in config.json

### Issue: Compilation error "wsUpgrader undefined"
**Solution:** Make sure:
1. Field added to Server struct
2. initWebSocketUpgrader() called in NewWithFS()
3. s.wsUpgrader used (not global upgrader)

---

## Success Indicators

You'll know it's working when:

✅ `coordinator init` creates config.json with empty tokens  
✅ Server requires ARCVAULT_ADMIN_TOKEN env var  
✅ Server requires ARCVAULT_JWT_SECRET env var  
✅ Logs show env vars loaded at startup  
✅ CORS requests from allowed origins get correct headers  
✅ CORS requests from blocked origins get 403  
✅ WebSocket connections from allowed origins succeed  
✅ WebSocket connections from blocked origins fail  
✅ Dashboard connects and shows "Live" indicator  
✅ All tests pass  

---

## Timeline

- **Start:** Day 1 morning (config.go + commands.go)
- **Mid-point:** Day 1 afternoon (testing + verification)
- **Day 2 morning:** server.go + hub.go
- **Day 2 afternoon:** useWebSocket.js + integration testing
- **End:** Code review ready for Elena

---

## Key Contacts

| Role | Name | For |
|------|------|-----|
| Architect | David Mensah | Architecture questions |
| Code Reviewer | Elena Vasquez | Code review |
| Security | Kwame Asante | Security questions |
| Owner | Kren Castro | Timeline/deployment |

---

## Quick Command Reference

```bash
# Setup
export ARCVAULT_ADMIN_TOKEN=$(openssl rand -hex 32)
export ARCVAULT_JWT_SECRET=$(openssl rand -hex 32)

# Build
go build -o /tmp/coordinator ./coordinator/

# Test
go test ./coordinator/tests/... -v

# Format
go fmt ./...

# Initialize
./coordinator init

# Run
./coordinator start
```

---

**Ready to start?** Begin with PHASE_1_SUMMARY.md, then use PHASE_1_FILE_CHANGES.md for implementation.

**Questions?** Check PHASE_1_IMPLEMENTATION_PLAN.md for detailed context.

Good luck! 🚀

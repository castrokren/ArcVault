# ArcVault 2.0 Security Remediation - Implementation Map

## Quick Reference: Files to Modify by Vulnerability

### P0-001: CORS Wildcard Validation [CRITICAL]

**Status:** Ready to implement  
**Effort:** 2 days  
**Risk:** Low  

#### Files to Modify

1. **`coordinator/server/server.go`** - Line ~414 (corsMiddleware function)
   ```go
   Current: AllowedOrigins validation doesn't reject "*"
   Change: Add startup check to reject wildcard in production
   
   - Reject if o == "*" in production mode
   - Validate origins use https:// or localhost
   - Log warnings
   ```

2. **`coordinator/config/config.go`** - Config struct validation
   ```go
   Current: AllowedOrigins is just []string
   Change: Add validation function
   
   - ValidateAllowedOrigins(origins []string) error
   - Called during config.Load()
   - Rejects "*", enforces https://
   ```

3. **`coordinator/cmd/commands.go`** - Init wizard
   ```go
   Current: No guidance on CORS setup
   Change: Update init wizard
   
   - Prompt for dashboard URL
   - Explain CORS security implications
   - Example: "https://dashboard.internal.corp"
   ```

#### Files to Create

- **`coordinator/tests/cors_test.go`** (new)
  ```go
  Test_CORSValidation_RejectsWildcard()
  Test_CORSValidation_AllowsSpecificOrigin()
  Test_CORSValidation_RejectsNonHTTPSOrigin()
  Test_CORSValidation_AllowsLocalhost()
  ```

---

### P0-002: WebSocket Origin Validation [CRITICAL]

**Status:** Depends on P0-001  
**Effort:** 2 days  
**Risk:** Low  

#### Files to Modify

1. **`coordinator/server/hub.go`** - Line ~64 (upgrader definition)
   ```go
   Current: 
     var upgrader = websocket.Upgrader{
         CheckOrigin: func(r *http.Request) bool { return true },
     }
   
   Change:
     - Create method on Server to initialize upgrader with proper CheckOrigin
     - Inject Server config into upgrader
     - Call during NewWithFS()
   ```

2. **`coordinator/server/server.go`** - NewWithFS() method
   ```go
   Current: upgrader is global, unreachable from Server config
   Change:
     - Move upgrader initialization to Server struct
     - Call s.initWebSocketUpgrader() after config loaded
     - Pass s.cfg.AllowedOrigins to CheckOrigin func
   ```

3. **`coordinator/server/hub.go`** - handleWS() and handleAgentWS()
   ```go
   Current: conn, err := upgrader.Upgrade(w, r, nil)
   Change:
     - Pass upgraded connection through origin-validated upgrader
     - Add logging: log.Printf("WebSocket: Rejected from origin %s", origin)
   ```

#### Files to Create

- **`coordinator/tests/websocket_test.go`** (new)
  ```go
  Test_WebSocketUpgrade_AllowsValidOrigin()
  Test_WebSocketUpgrade_RejectsInvalidOrigin()
  Test_WebSocketUpgrade_AllowsNoOriginHeader()
  Test_WebSocketUpgrade_RejectsWildcardCORS()
  Test_WebSocketUpgrade_LogsRejectedOrigin()
  ```

---

### P0-003: Command Injection Prevention [CRITICAL]

**Status:** Ready to implement (audit mode first)  
**Effort:** 4 days  
**Risk:** Medium  

#### Files to Modify

1. **`agent/runner/executor.go`** - RealExecutor() function
   ```go
   Current:
     args := parseCommandArgs(job.Command)
     cmd := exec.Command(args[0], args[1:]...)
   
   Change:
     - Call ValidateCommand(job.Command) before parsing
     - Return error if validation fails
     - Use exec.LookPath(args[0]) instead of direct args[0]
       (prevents relative path traversal)
   ```

2. **`agent/runner/executor.go`** - Add new validation function
   ```go
   func ValidateCommand(cmd string) error {
       // Validate command format
       // Check program is whitelisted (rsync, robocopy only)
       // Check arguments don't contain shell metacharacters
       // Reject relative paths
   }
   
   var AllowedCommandPrograms = map[string]bool{
       "rsync":    true,
       "robocopy": true,
   }
   ```

3. **`coordinator/server/templates.go`** - handleCreateTemplate()
   ```go
   Current: No validation of template.Command field
   Change:
     - Call agent/runner.ValidateCommand() on input
     - Return 400 BadRequest if validation fails
     - Log template creation for audit
     - Add mode flag: validation_mode: "audit" | "enforce"
   ```

4. **`coordinator/business/jobs.go`** - or create new validation layer
   ```go
   // Add template validation at business logic layer
   // Called both on creation and before execution
   ```

#### Files to Create

- **`agent/runner/executor_validation_test.go`** (new)
  ```go
  Test_ValidateCommand_AllowsRsync()
  Test_ValidateCommand_AllowsRobocopy()
  Test_ValidateCommand_RejectsBash()
  Test_ValidateCommand_RejectsCurl()
  Test_ValidateCommand_RejectsNc()
  Test_ValidateCommand_RejectsRelativePaths()
  Test_ValidateCommand_RejectsShellMetacharacters()
  
  // Error message examples:
  // "program 'bash' not allowed (whitelist: rsync, robocopy)"
  // "argument contains disallowed characters: ';rm -rf /'"
  ```

- **`coordinator/server/templates_validation_test.go`** (new)
  ```go
  Test_CreateTemplate_ValidatesCommand()
  Test_CreateTemplate_RejectsUnauthorizedProgram()
  Test_CreateTemplate_AuditModeLogsViolations()
  Test_CreateTemplate_EnforceModeRejects()
  ```

- **`coordinator/db/templates.go`** (if not exists)
  ```go
  // Audit queries for template validation
  // SELECT id, command FROM templates WHERE ...
  ```

#### Database Audit Query

```sql
-- Find templates with suspicious commands before enforcement
SELECT id, name, command FROM templates
WHERE command LIKE '%bash%'
   OR command LIKE '%curl%'
   OR command LIKE '%nc%'
   OR command LIKE '%powershell%'
   OR command LIKE '%cmd%'
   OR command LIKE '%sh%'
   OR command LIKE '%perl%'
   OR command LIKE '%python%';
```

---

### P0-004: Admin Token from Environment Variables [CRITICAL]

**Status:** Ready to implement  
**Effort:** 3 days  
**Risk:** Low (backward compatible)  

#### Files to Modify

1. **`coordinator/config/config.go`** - Config struct and Load()
   ```go
   Current:
     type Config struct {
         AdminToken string `json:"admin_token"`
         JWTSecret  string `json:"jwt_secret"`
     }
   
   Change:
     - Keep fields but mark as "internal/optional"
     - Update Load() to check environment variables first:
       * ARCVAULT_ADMIN_TOKEN
       * ARCVAULT_JWT_SECRET
     - Env vars override config file values
     - Fail with clear error if neither set and production mode
   ```

2. **`coordinator/config/config.go`** - Save() function
   ```go
   Current: Writes AdminToken and JWTSecret to JSON
   Change:
     - Create sanitized copy of config
     - Set sensitive fields to empty string before marshaling
     - Write only sanitized config to file
     - Log message: "Sensitive fields cleared from config file
                    Set ARCVAULT_ADMIN_TOKEN env var"
   ```

3. **`coordinator/cmd/commands.go`** - Init command
   ```go
   Current: init generates token and saves to config
   Change:
     - Generate tokens (for reference)
     - Display in console: "Your admin token: <token>"
     - Show instructions:
       $ export ARCVAULT_ADMIN_TOKEN=<token>
       $ export ARCVAULT_JWT_SECRET=<secret>
     - Warn: "Do NOT share these tokens or commit to git"
     - Save config.json WITHOUT tokens
   ```

4. **`coordinator/main.go`** - Startup
   ```go
   Current: Loads config silently
   Change:
     - Add validation on startup:
       if cfg.Environment == "production" && cfg.AdminToken == "" {
           return fmt.Errorf("CRITICAL: ARCVAULT_ADMIN_TOKEN env var not set")
       }
     - Log which source was used:
       if os.Getenv("ARCVAULT_ADMIN_TOKEN") != "" {
           log.Printf("Loaded AdminToken from ARCVAULT_ADMIN_TOKEN env var")
       } else {
           log.Printf("WARNING: AdminToken loaded from config file (use env var in prod)")
       }
   ```

#### Files to Create

- **`coordinator/tests/config_test.go`** (update if exists)
  ```go
  Test_LoadConfig_FromEnvVar()
  Test_LoadConfig_EnvVarOverridesConfigFile()
  Test_LoadConfig_FailsIfNoTokenAndProduction()
  Test_SaveConfig_NeverWritesSensitiveFields()
  Test_LoadConfig_LogsSourceOfToken()
  ```

#### Documentation Files

- **`DEPLOYMENT.md`** (create or update)
  ```markdown
  ## Production Deployment
  
  Before starting coordinator:
  
  1. Generate random tokens:
     $ TOKEN=$(openssl rand -hex 32)
     $ SECRET=$(openssl rand -hex 32)
  
  2. Set environment variables:
     $ export ARCVAULT_ADMIN_TOKEN=$TOKEN
     $ export ARCVAULT_JWT_SECRET=$SECRET
  
  3. Start coordinator:
     $ coordinator start
  
  Do NOT include tokens in config.json or commit to git.
  Use secret management system (Vault, AWS Secrets Manager, etc.)
  ```

---

### P0-005: Database Scan Error Handling [HIGH]

**Status:** Ready to implement  
**Effort:** 1 day  
**Risk:** Low  

#### Files to Modify

1. **`coordinator/db/jobs.go`** - ListJobs() function
   ```go
   Current:
     for rows.Next() {
         if err := rows.Scan(&j...); err != nil {
             continue  // ← SILENTLY IGNORED
         }
     }
     return jobs, total, rows.Err()
   
   Change:
     - Track scan errors in slice
     - Log each error with row context
     - Return error from function if scan errors occurred
     - Include count in error message
   ```

2. **`coordinator/server/jobs.go`** - handleListJobs()
   ```go
   Current: 
     result, err := s.jobService.ListJobs(...)
     if err != nil {
         http.Error(w, "failed to list jobs", http.StatusInternalServerError)
     }
   
   Change:
     - Check if error is "partial results"
     - If partial, set response headers:
       * X-Partial-Results: true
       * X-Partial-Error: <error details>
     - Return 200 with partial results + headers
     - Optionally return 503 if critical
   ```

3. **`coordinator/db/db.go`** - Consider adding DataIntegrityChecker
   ```go
   func (d *DB) CheckJobsDataIntegrity() error {
       // Scan all jobs, verify no corrupt rows
       // Return error if any found
   }
   ```

#### Files to Create

- **`coordinator/tests/jobs_db_test.go`** (update if exists)
  ```go
  Test_ListJobs_LogsAndReturnsScanErrors()
  Test_ListJobs_PartialResultsHeaderWhenErrorsOccur()
  Test_ListJobs_CountsSkippedRowsInError()
  ```

- **`coordinator/server/health.go`** (create or update)
  ```go
  // Add database health check endpoint
  // GET /api/admin/database-health
  
  func (s *Server) handleDatabaseHealth(w http.ResponseWriter, r *http.Request) {
      integrity := s.db.CheckJobsDataIntegrity()
      if integrity != nil {
          json.NewEncoder(w).Encode(map[string]interface{}{
              "status": "WARNING",
              "error": integrity.Error(),
          })
      }
  }
  ```

---

### P0-006: SSH Credential Race Condition [HIGH]

**Status:** Ready to implement  
**Effort:** 3 days  
**Risk:** Medium (requires thorough testing)  

#### Files to Modify

1. **`agent/runner/runner.go`** - Add ExecutorContext
   ```go
   type ExecutorContext struct {
       JobID        string
       TempKeyPath  string
       TempKeyDir   string
       CleanupFuncs []func()
   }
   
   // Update RealExecutor signature:
   // func RealExecutor(ctx *ExecutorContext, job Job, ...) (exitCode int, output string)
   ```

2. **`agent/runner/credentials.go`** - Refactor all credential functions
   ```go
   Current:
     func applySSHKeyCredentials(keyData string) (func(), error) {
         os.Setenv("SSH_KEY_PATH", tempPath)  // ← GLOBAL
     }
   
   Change:
     func applySSHKeyCredentials(ctx *ExecutorContext, keyData string) error {
         // Create job-specific temp directory
         jobDir := filepath.Join(os.TempDir(), fmt.Sprintf("arcvault-job-%s", ctx.JobID))
         keyPath := filepath.Join(jobDir, "key.pem")
         
         // Store in context (not global env var)
         ctx.TempKeyPath = keyPath
         ctx.TempKeyDir = jobDir
         ctx.CleanupFuncs = append(ctx.CleanupFuncs, func() {
             os.RemoveAll(jobDir)
         })
     }
   ```

3. **`agent/runner/executor.go`** - Update rsync/robocopy calls
   ```go
   Current:
     cmd := exec.Command("rsync", args...)
   
   Change:
     // Pass SSH key path explicitly:
     args = append(args, "-e", fmt.Sprintf("ssh -i %s", ctx.TempKeyPath))
     cmd := exec.Command("rsync", args...)
   ```

4. **`agent/main.go`** - Update job execution
   ```go
   Current:
     exitCode, output := RealExecutor(job, report)
   
   Change:
     ctx := &ExecutorContext{JobID: job.ID}
     exitCode, output := RealExecutor(ctx, job, report)
     defer func() {
         for _, cleanup := range ctx.CleanupFuncs {
             cleanup()
         }
     }()
   ```

#### Files to Create

- **`agent/tests/credentials_concurrent_test.go`** (new)
  ```go
  Test_ConcurrentJobsWithDifferentSSHKeys()
  // Scenario:
  // - Run 2 jobs simultaneously with different SSH keys
  // - Verify each job uses its own key (rsync logs verify correct user)
  // - Verify temp files cleaned up after both complete
  
  Test_SSHKeyFileCleanedUpAfterJobCompletes()
  Test_SSHKeyFileNotAccessibleToOtherJobs()
  Test_NoOrphanedKeyFilesAfterJobFailure()
  Test_PasswordCredentialNotLeakedToSiblingJob()
  ```

#### Integration Test Scenario

```go
// Run from agent test harness:
// 1. Create 2 SSH credentials (alice-key, bob-key)
// 2. Create 2 jobs on same agent to run simultaneously:
//    - Job A: rsync with alice-key
//    - Job B: rsync with bob-key
// 3. Mock rsync to log SSH_KEY_PATH it received
// 4. Verify:
//    - Job A log contains alice-key path
//    - Job B log contains bob-key path (NOT alice-key)
// 5. After both complete, verify /tmp/ has no orphaned key files
```

---

### P0-007: Scheduler Query Optimization [HIGH]

**Status:** Ready to implement  
**Effort:** 2 days  
**Risk:** Low  

#### Files to Modify

1. **`coordinator/server/scheduler.go`** - checkMissedSchedules()
   ```go
   Current: N+1 queries (1 initial + M jobs + M*K rules)
   Change:
     - Rewrite to single batch query with JOINs
     - Query both jobs and alert_rules and last_run in one call
     - Process results in single pass
     - Add timeout context
   
   Schema: SELECT j.id, j.name, ar.id, ar.threshold, 
                   MAX(jr.finished_at), COUNT(ah.id)
           FROM jobs j
           LEFT JOIN alert_rules ar ON j.id = ar.job_id
           LEFT JOIN job_runs jr ON j.id = jr.job_id
           LEFT JOIN alert_history ah ON j.id = ah.job_id AND ...
   ```

2. **`coordinator/server/scheduler.go`** - Add query timeout
   ```go
   Current: No timeout on scheduler queries
   Change:
     ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
     defer cancel()
     rows, err := s.db.Conn().QueryContext(ctx, ...)
   ```

3. **`coordinator/server/scheduler.go`** - Add performance logging
   ```go
   start := time.Now()
   s.checkMissedSchedules()
   duration := time.Since(start)
   
   log.Printf("Scheduler: checkMissedSchedules took %.2f seconds", duration.Seconds())
   if duration > 5*time.Second {
       log.Printf("WARNING: Scheduler slow, may impact coordinator responsiveness")
   }
   ```

4. **`coordinator/db/schema.go`** or initialization - Add indexes
   ```sql
   CREATE INDEX IF NOT EXISTS idx_jobs_schedule ON jobs(schedule) 
       WHERE schedule IS NOT NULL;
   
   CREATE INDEX IF NOT EXISTS idx_alert_rules_job_enabled ON alert_rules(job_id, enabled);
   
   CREATE INDEX IF NOT EXISTS idx_job_runs_job_finished ON job_runs(job_id, finished_at DESC);
   
   CREATE INDEX IF NOT EXISTS idx_alert_history_job_rule_fired 
       ON alert_history(job_id, rule_type, fired_at DESC);
   ```

#### Files to Create

- **`coordinator/tests/scheduler_test.go`** (update if exists)
  ```go
  Test_CheckMissedSchedules_QueryEfficiency()
  // Run with 1000 jobs, 5 rules each
  // Benchmark: should complete in < 1 second
  
  Test_SchedulerQueryTimeout()
  Test_SchedulerWithIndexes_IsFaster()
  Test_SchedulerLogsPerformanceMetrics()
  ```

---

### P0-008: Agent Polling Jitter [MEDIUM]

**Status:** Ready to implement  
**Effort:** 1 day  
**Risk:** Very Low  

#### Files to Modify

1. **`agent/runner/runner.go`** - Start() method
   ```go
   Current:
     ticker := time.NewTicker(r.cfg.PollInterval)
   
   Change:
     // Add jitter: ±20% variance
     jitterRange := r.cfg.PollInterval / 5
     jitterMs := time.Duration(rand.Int63n(int64(2*jitterRange))) - 
                 time.Duration(int64(jitterRange))
     jitteredInterval := r.cfg.PollInterval + jitterMs
     
     // Add initial delay to scatter polls
     initialDelay := time.Duration(rand.Int63n(int64(r.cfg.PollInterval)))
     time.Sleep(initialDelay)
     
     ticker := time.NewTicker(jitteredInterval)
     
     log.Printf("Agent: Polling with interval %v (jitter applied), next poll in %v",
         jitteredInterval, initialDelay)
   ```

2. **`agent/main.go`** - Config
   ```go
   Current:
     Config: runner.Config{
         PollInterval: 30 * time.Second,
     }
   
   Change:
     Config: runner.Config{
         PollInterval:    30 * time.Second,
         PollJitterPct:   20,  // ±20%
     }
   ```

3. **`agent/runner/runner.go`** - Config struct
   ```go
   type Config struct {
       PollInterval   time.Duration  // existing
       PollJitterPct  int            // NEW: percentage jitter (0-100)
       // ...
   }
   ```

#### Files to Create

- **`agent/tests/polling_test.go`** (new)
  ```go
  Test_PollingIntervalHasJitter()
  // Run 10 agents
  // Record actual poll times
  // Verify NOT all at same time (e.g., T=0s, T=30s, etc.)
  // Verify distributed over time window
  
  Test_InitialDelayScattersPolls()
  // Verify all agents don't poll at T=0s
  // Some at T=5s, T=12s, T=18s, etc.
  
  Test_JitterPercentageConfigurable()
  Test_ZeroJitterDisablesFeature()
  ```

---

## Database Schema Changes

### New Indexes (Critical for P0-007)

```sql
-- For scheduler query optimization
CREATE INDEX IF NOT EXISTS idx_jobs_schedule 
    ON jobs(schedule) WHERE schedule IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_alert_rules_job_enabled 
    ON alert_rules(job_id, enabled);

CREATE INDEX IF NOT EXISTS idx_job_runs_job_finished 
    ON job_runs(job_id, finished_at DESC);

CREATE INDEX IF NOT EXISTS idx_alert_history_job_rule_fired 
    ON alert_history(job_id, rule_type, fired_at DESC);

-- For CORS/WebSocket audit logging
CREATE TABLE IF NOT EXISTS security_audit (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    event_type TEXT NOT NULL,
    origin TEXT,
    ip_address TEXT,
    user_agent TEXT,
    result TEXT,  -- 'allowed' or 'denied'
    timestamp DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_audit_timestamp ON security_audit(timestamp DESC);
```

---

## Environment Variables

### Required (Production)

```bash
# Authentication
export ARCVAULT_ADMIN_TOKEN=<random-32-byte-hex>
export ARCVAULT_JWT_SECRET=<random-32-byte-hex>

# Optional overrides
export ARCVAULT_ALLOWED_ORIGINS="https://dashboard.internal.corp"
export ARCVAULT_ENVIRONMENT="production"
export ARCVAULT_POLL_INTERVAL_SECONDS="30"
export ARCVAULT_POLL_JITTER_PERCENT="20"
```

### For Testing

```bash
# Audit mode for P0-003
export ARCVAULT_COMMAND_VALIDATION_MODE="audit"  # audit | enforce

# Debug logging
export ARCVAULT_DEBUG="true"
export ARCVAULT_SECURITY_AUDIT="true"
```

---

## Configuration Changes

### config.json (Updated Format)

```json
{
  "port": 8080,
  "database_path": "./arcvault.db",
  "admin_token": "",
  "jwt_secret": "",
  "environment": "production",
  "coordinator_id": "primary",
  "alert_history_retention_days": 30,
  "allowed_origins": [
    "https://dashboard.internal.corp",
    "http://localhost:3000"
  ],
  "host": "0.0.0.0",
  "cert_file": "/etc/arcvault/cert.pem",
  "key_file": "/etc/arcvault/key.pem",
  "external_tls": false,
  "installer_dir": "./installers",
  "command_validation_mode": "enforce"
}
```

**Notes:**
- `admin_token` and `jwt_secret` should be EMPTY in file
- Values come from env vars (ARCVAULT_ADMIN_TOKEN, ARCVAULT_JWT_SECRET)
- `allowed_origins` replaces the implicit "*" with explicit list

---

## Testing Infrastructure

### New Test Files Required

```
coordinator/tests/
  ├── cors_test.go           (P0-001)
  ├── websocket_test.go      (P0-002)
  ├── templates_validation_test.go  (P0-003)
  ├── config_test.go         (P0-004)
  ├── jobs_db_test.go        (P0-005)
  ├── scheduler_test.go      (P0-007)
  └── health_check_test.go   (P0-005, P0-007)

agent/tests/
  ├── credentials_concurrent_test.go  (P0-006)
  ├── executor_validation_test.go     (P0-003)
  └── polling_test.go        (P0-008)
```

### Test Categories

**Security Tests** (P0-001, P0-002, P0-003, P0-004)
- CORS origin validation
- WebSocket origin rejection
- Command whitelist enforcement
- Token sourcing (env var vs config)

**Concurrency Tests** (P0-006, P0-008)
- Multiple jobs with different credentials
- Multiple agents polling (verify no synchronization)
- Race condition stress tests

**Performance Tests** (P0-007, P0-008)
- Scheduler query efficiency with 1000+ jobs
- Polling load distribution (no spikes)
- Coordinator responsiveness during scheduler run

**Audit Tests** (P0-005)
- Database error logging
- Partial result reporting
- X-Partial-Results header

---

## Deployment Checklist

### Pre-Deployment (All Phases)

- [ ] Code review of all changes
- [ ] Unit tests passing (100% coverage for security functions)
- [ ] Integration tests passing (staging environment)
- [ ] Load testing completed (500+ agents if applicable)
- [ ] Security testing completed (MITM, credential leaks, race conditions)
- [ ] Rollback plan documented and tested

### Phase 1 Deployment (Auth/Authz)

- [ ] Staging environment deployed with P0-001 + P0-004
- [ ] Verify AllowedOrigins validated
- [ ] Verify env var loading works (ARCVAULT_ADMIN_TOKEN set)
- [ ] Verify config.json no longer contains sensitive fields
- [ ] Smoke test: Admin can login, browser can access dashboard
- [ ] Production deployment window scheduled
- [ ] Rollback: Have old config.json (with tokens) ready in case of failure

### Phase 2 Deployment (Execution)

- [ ] Audit existing templates for violations
- [ ] Deploy in audit_mode (violations logged, not blocked)
- [ ] Monitor logs for 1 week: Any violations reported?
- [ ] Admin reviews and approves/updates flagged templates
- [ ] Enable enforcement_mode: template validation blocks unauthorized programs

### Phase 3 Deployment (Operations)

- [ ] P0-005, P0-007, P0-008: Deploy immediately (low-risk)
  - [ ] Verify scheduler runtime < 1s (check logs)
  - [ ] Verify polling distributed over 30-second window (no spikes)
  - [ ] Monitor DB error logs (should be zero)
- [ ] P0-006: Deploy after concurrent job testing
  - [ ] Run 100x concurrent job test with different SSH keys
  - [ ] Verify no key mixing in logs
  - [ ] Verify /tmp/ clean after all jobs complete

### Post-Deployment Monitoring

- [ ] Set up alerts for security metrics (see THREAT_MODEL.md)
- [ ] Monitor scheduler runtime (alert if > 2s)
- [ ] Monitor coordinator latency (alert if spikes)
- [ ] Monitor CORS/WebSocket rejections (alert if > 10/day)
- [ ] Run database integrity check weekly
- [ ] Review security audit logs daily (first month)

---

## Rollback Plan by Phase

### Phase 1 Rollback (Auth/Authz)

If P0-001 (CORS) causes issues:
- Revert coordinator/server/server.go corsMiddleware
- Restore AllowedOrigins to "*" (dev only)
- Re-deploy

If P0-002 (WebSocket) causes issues:
- Revert coordinator/server/hub.go CheckOrigin to `return true`
- Agents can reconnect without auth

If P0-004 (env vars) causes issues:
- Revert coordinator/config/config.go Load()
- coordinator crashes if AdminToken not in env var → add to config.json temporarily
- Re-deploy with both env var and config file support

### Phase 2 Rollback (Execution)

If P0-003 (command validation) blocks legitimate templates:
- Revert to audit_mode (violations logged but not blocked)
- Update templates manually
- Re-enable enforcement_mode

### Phase 3 Rollback (Operations)

Each fix is independent and reversible:
- P0-005: Remove error logging (safe)
- P0-006: Revert to global env vars (revert agent binary)
- P0-007: Revert to N+1 queries (revert server binary)
- P0-008: Remove jitter (revert agent binary)

---

## Success Criteria

### Code Quality

- ✓ All security functions unit tested (100% coverage)
- ✓ No compiler warnings or errors
- ✓ go fmt, go vet, staticcheck clean
- ✓ Code review approval from Elena Vasquez

### Security Testing

- ✓ CORS wildcard rejected in production
- ✓ WebSocket origin checks working
- ✓ Command validation blocks unauthorized programs
- ✓ Admin tokens loaded from env vars (not config)
- ✓ SSH keys isolated per job (no cross-job leakage)
- ✓ Database errors logged and reported

### Performance

- ✓ Scheduler runtime < 1 second (was 3-5s)
- ✓ Coordinator request latency steady (no 30-second spikes)
- ✓ Agent polling distributed (no synchronized spikes)

### Operational

- ✓ No regression in functionality
- ✓ Agent heartbeats working
- ✓ Job execution succeeding
- ✓ Backups completing normally
- ✓ WebSocket broadcasts working
- ✓ Database healthy (no corruption detected)

---

**Document:** REMEDIATION_IMPLEMENTATION_MAP.md  
**Status:** READY FOR DEVELOPMENT TEAM  
**Next:** Assign tasks to engineers, begin Phase 1 implementation

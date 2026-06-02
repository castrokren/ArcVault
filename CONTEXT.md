# ArcVault2.0 -- Quick Reference
**Last updated:** June 2, 2026 1:40pm EDT | **v0.2.1** | **Release Ready ✅**

## Status
✅ Phase 12: Failure notifications (webhook + email)  
✅ Phase 13: Scheduled backup templates (cron-based job automation)  
✅ Phase 14: Agent update system & rollback  
✅ Phase 15 (backend): RBAC with JWT authentication, user management, agent groups  
✅ Phase 15 (frontend): Login, useAuth composable, user/group CRUD, smart job forms  
✅ Phase 16 (backend): Federation failover, state sync (federation_events log), health monitoring  
✅ Phase 16 (frontend): FederationHealth.vue dashboard with auto-refresh  
✅ Phase 16 (agent): Coordinator list failover with exponential backoff  
✅ Phase 16 (gaps): Agent homing persisted, heartbeat detector live, stale banners wired to lag composable  
✅ Phase 17 (COMPLETE): Enhanced monitoring & alerting
  - Job start time accuracy fix (recorded in job_runs table)
  - Alert rules engine (per-job configurable rules: on_failure, duration_exceeded, missed_schedule)
  - Webhook retry with exponential backoff (3 attempts: 5s → 15s → 45s)
  - Slack & Teams notifiers (via incoming webhooks)
  - Alert history tracking (30-day retention by default)
  - Scheduler: Missed schedule detector + history pruning
  - Frontend: Alerts.vue dashboard with rule creation/deletion & retry controls  
✅ All tests passing | v1.0.0 released on GitHub  
✅ Phase 18 (frontend): Comprehensive dashboard design system overhaul — all 21 Vue files redesigned, new token system, animated login, rebuilt components  
✅ Services redeployed and verified (2026-05-27)  
✅ v1.0.1 bugfixes applied (2026-05-28):
  - Jobs page agent dropdown broken when no groups exist (nil slice JSON null + wrong PaginatedResponse key)
  - Update check endpoint returned plain text on error instead of JSON  
✅ v1.0.3 (2026-05-28): Schedule builder UI — ScheduleBuilder.vue component (Interval/Daily/Weekly/Monthly/Custom modes, live cron preview) wired into Jobs and Templates forms
✅ v1.0.2 (2026-05-28): Delete agents feature
  - DELETE /api/agents/{id} endpoint (admin only, 409 blocks if running jobs)
  - Cleans up tokens + group memberships on delete; historical jobs preserved
  - Dashboard: Delete button + confirmation modal in Agents.vue (admin-only, local view)
  - 6 new tests in agent_delete_test.go; full suite passing
✅ Phase 19 (COMPLETE 2026-05-28): Robocopy/Rsync Advanced Flags
  - Backend: SyncFlags struct (Mirror, MaxAge, MinAge, MaxSize, ExcludeFiles, ExcludeDirs) + validation + ToRobocopyArgs/ToRsyncArgs methods (32 tests, all passing)
  - Frontend: SyncFlagsBuilder.vue component (collapsible Advanced Options, real-time command preview, form validation)
  - Integration: Wired into Jobs form via v-model, API payload includes/omits sync_flags correctly
  - Tests: 70 total (32 backend + 29 component unit + 9 integration)
  - Bug fixes: Undefined array handling, API response structure (agentsData.data), form state reset
✅ Phase 20 (COMPLETE): Job cancellation + progress tracking (initial)
  - Cancel endpoint + status workflow
  - Initial progress column schema
✅ Phase 21a-3 (COMPLETE 2026-05-28): Real-time job progress tracking
  - Backend: POST /api/jobs/{id}/progress (percentage, logs, status) + GET endpoint with log history
  - Database: Auto-create job_runs on job insertion via trigger; UpdateProgressAndLogs writes to job_runs + job_logs
  - Frontend: ProgressBar.vue component (4px green bar, smooth transitions); Jobs.vue WebSocket listener for real-time updates
  - Architecture: Agent → POST progress → DB broadcast → WebSocket → Frontend (no polling needed)
  - Tests: 15/15 progress tests passing; full suite 125+ tests passing
  - PR: phase/21a-3-progress-tracking (ready for review)
✅ Phase 21a-4 (COMPLETE 2026-05-29): Critical hot fix — Jobs stuck in pending
  - Issue: After rebuild, jobs created but never executed (stuck in PENDING or RUNNING)
  - Root cause 1: handleListJobs missing sync_flags in SQL SELECT — agents got incomplete job data
  - Root cause 2: robocopy hanging with /LOG+:NUL flag — blocked agent from processing jobs
  - Fix 1: Added sync_flags to SELECT, Scan, and JSON deserialization in coordinator/server/jobs.go (lines 239, 253, 262-267)
  - Fix 2: Replaced robocopy flags with /R:0 /W:0 /NP /NFL /NDL in agent/runner/executor.go (line 26)
  - Lessons learned documented in memory/phase21a4_lessons_learned.md
  - Testing: Verified jobs transition PENDING→RUNNING→COMPLETED with actual file copying
  - Deployment: Run .\scripts\rebuild-and-restart.ps1
  - Files changed: coordinator/server/jobs.go, agent/runner/executor.go, FIX_SUMMARY.md, memory/phase21a4_lessons_learned.md
✅ v0.2.1 (2026-05-29): Scheduled jobs fix + self-update verification
  - Issue: Scheduled jobs executed immediately instead of waiting for scheduled time
  - Root cause 1: Jobs created with status="pending" even when schedule field set (should be "scheduled")
  - Root cause 2: Jobs created after startup not registered with cron scheduler
  - Fix 1: Set status="scheduled" in handleCreateJob when Schedule != nil (coordinator/server/jobs.go lines 81-96, 171-186)
  - Fix 2: Added late-binding cron registration via registerJobSchedule() called after job insert (lines 127, 216)
  - Fix 3: Added global jobCron + jobCronMu in scheduler.go for post-startup job registration (lines 23-26, 142)
  - Fix 4: Changed reset logic from "pending" → "scheduled" in triggerScheduledJobs (line 50)
  - GitHub Actions: Updated deprecated artifact actions v3 → v4 in build-installers.yml
  - Self-update: Verified coordinator check-update finds v0.2.1 on second PC (asset naming: coordinator_windows_amd64.exe)
  - Tests: Updated 3 scheduler tests to expect "scheduled" status for jobs with schedule
  - Release: v0.2.1 tagged, installer + binary uploaded to GitHub

✅ **Session 6** (June 2, 2026): v0.2.1 Release Finalization — COMPLETE
  - **Task 1c ✅**: Fresh install coordinator service running with Automatic startup
  - **Task 1d ✅**: Health endpoint returns 200 OK {"status":"ok"}
  - **Task 1e ✅**: Browser test: Login works, dashboard loads, agents page shows registered agent
  - **Issue Found & Resolved ✅**: Agent service startup failure (exit code 1067)
    - **Root Cause**: Token mismatch — service loads config from installer/windows directory
    - **Fix**: Regenerated tokens using coordinator from installer directory, synced both configs
    - **Result**: Agent now registered as DESKTOP-EE77F38, online, heartbeat working
    - **Lesson**: Service config paths are relative to executable location (filepath.Join(dir(exe), "config.json"))
  - **Files Changed**: 
    - installer/windows/agent-config.yaml (new valid token)
    - installer/windows/config.json (updated admin token + database path)
  - **Commits**: 
    - 1cf96a9: fix(tokens): regenerate agent and coordinator tokens to resolve service startup failure
    - 9b92289: cleanup(stop): clear agent service stop flag after successful startup
  - **Next**: Task 2 (Tag v0.2.1) and Task 3 (Re-enable GitHub Actions) blocked pending agent service resolution

🎯 Next: Resolve agent service startup issue, complete Tasks 2-3, run full browser checklist

## Core Commands
```bash
# Initialize coordinator
coordinator init

# Start coordinator (runs dashboard on :8080)
coordinator start

# Generate per-agent token
coordinator create-agent-token <agent-id>

# Check for updates
coordinator check-update

# Install as system service
coordinator install-service
agent install-service
```

## What's Implemented
- ✅ Single binary deployment (coordinator) with embedded Vue dashboard
- ✅ Per-agent tokens (in addition to admin token)
- ✅ Self-update system (coordinator + agents, live WebSocket progress)
- ✅ Bidirectional rollback (one-version-back, v0.3.0+)
- ✅ Server-side pagination & filtering (all list endpoints)
- ✅ Job history visualization (timeline + agent charts, v0.4.0+)
- ✅ Failure notifications (webhook + email, v0.5.0+)
- ✅ JWT-based RBAC (v0.8.0): Three roles (admin, operator, viewer) with fine-grained endpoint access
- ✅ User management: Create/list/delete/update roles with bcrypt password hashing
- ✅ Agent groups: Organize agents by environment or function, assign members

## Quick Setup

**Per-agent token:**
```bash
coordinator create-agent-token agent-01
# Copy token to agent-config.yaml as auth_token
```

**Notification config** (`coordinator/config.json`):
```json
{
  "notifications": {
    "on_failure": true,
    "webhook": {
      "url": "https://hooks.example.com/arcvault",
      "secret": "hmac-secret"
    },
    "email": {
      "smtp_host": "smtp.example.com",
      "smtp_port": 587,
      "from": "arcvault@example.com",
      "to": ["ops@example.com"],
      "username": "user",
      "password": "pass"
    }
  }
}
```
*Both webhook and email optional; webhook uses GitHub convention: `X-ArcVault-Signature: sha256=<hex>`*

**Service installation:**
```bash
# Windows (admin PowerShell)
coordinator install-service
sc start arcvault-coordinator

# Linux/macOS (root)
sudo coordinator install-service
sudo systemctl start arcvault-coordinator        # Linux
sudo launchctl start com.arcvault.coordinator   # macOS
```

## Reference
- **Project instructions & routing:** [CLAUDE.md](CLAUDE.md)
- **Phase history & architecture:** [MEMORY.md](MEMORY.md) (detailed design decisions, technical stack, full roadmap)
- **Current branch:** main
- **Latest r
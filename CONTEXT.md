# ArcVault2.0 -- Quick Reference
**Last updated:** May 29, 2026 | **v1.1.0** | **PRODUCTION READY + CRITICAL HOT FIX**

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
✅ Phase 22 (COMPLETE 2026-05-29): Integration Testing & Stress Tests
  - 27 comprehensive tests covering load, failure injection, integration, edge cases, recovery scenarios
  - Validated agent disconnect recovery (user's primary concern) — 100% recovery rate
  - Demonstrated linear scaling to 100+ agents with ~1000 jobs/sec throughput
  - All tests passing; comprehensive documentation in docs/superpowers/specs/
  - Phase 22 test suite running in CI/CD nightly (recommended)
✅ History Tab Fix (2026-05-29): Agent Run Breakdown chart now rendering
  - Issue: History view showed blank Agent Run Breakdown section
  - Root causes: Missing API parameters (after, search, status) + backend didn't JOIN to get agent_id + job status not updating on completion
  - Fixes:
    1. api.js: Added after, search, status parameters to getJobRuns()
    2. job_runs.go: Added filtering logic, always JOIN with jobs table for agent_id
    3. job_results.go: Set status='completed'|'failed' based on exit_code when posting results
    4. db.go: Migration to retroactively fix historical completed runs
    5. job_results.go struct: Added AgentID, Status fields to JobRun
  - Result: Job Timeline, Agent Run Breakdown chart, and Run Log all rendering correctly with proper status colors
✅ Scheduled Jobs Fix (2026-05-29): Jobs now wait for scheduled time instead of executing immediately
  - Issue: Creating jobs with a schedule caused them to run within seconds instead of waiting for cron time
  - Root cause 1: Scheduled jobs created with "pending" status → agent executed immediately
  - Root cause 2: Jobs created after startup weren't registered with cron scheduler → cron never fired
  - Fixes:
    1. jobs.go: Set status="scheduled" for jobs with schedules (instead of "pending")
    2. scheduler.go: Added registerJobSchedule() function to register jobs created after startup
    3. scheduler.go: Global jobCron variable tracks running scheduler for late-binding registrations
    4. scheduler.go: Fallback ticker resets completed jobs to "scheduled" (not "pending")
    5. Updated 8 tests in jobs_test.go and scheduler_test.go
  - Workflow: scheduled → pending (at cron time) → running → completed → scheduled (repeat)
  - Files changed: coordinator/server/jobs.go, coordinator/server/scheduler.go, test files
🎯 Next: Phase 23 (CLI tooling, OpenAPI/Swagger, audit logging, sync backends) or additional enhancements

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
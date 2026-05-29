# ArcVault2.0 -- Quick Reference
**Last updated:** May 28, 2026 | **v1.0.4** | **PRODUCTION READY**

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
🎯 Next: Cancel backups, backup progress indicator

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
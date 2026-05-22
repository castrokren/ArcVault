# ArcVault2.0 -- Quick Reference
**Last updated:** May 22, 2026 | **v0.9.0** | **Phase 16 COMPLETE + gaps closed**

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
📊 All backend tests passing  
🎯 Next: Phase 17 planning

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
- **Current branch:** feature/phase-15-rbac (Phase 16 complete + gaps closed, ready to merge or branch for Phase 17)
- **Latest release:** https://github.com/castrokren/ArcVault/releases/tag/v0.9.0

## Development Tips
- **PowerShell:** Line continuation uses backtick `` ` ``, not backslash
- **Tests:** Run `go test ./...` to verify all tests pass
- **Dashboard:** Embedded at compile time; see `coordinator/static/`

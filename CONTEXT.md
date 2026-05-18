# ArcVault2.0 -- Quick Reference
**Last updated:** May 18, 2026 | **v0.5.0** | **Phase 12 complete** → Phase 13 next

## Status
✅ Phase 12: Failure notifications (webhook + email)  
📊 110 tests passing (108 pass + 2 skip on Windows)  
🎯 Phase 13: Scheduled backup templates (cron-based job automation)

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
- **Latest release:** https://github.com/castrokren/ArcVault/releases/tag/v0.5.0

## Development Tips
- **PowerShell:** Line continuation uses backtick `` ` ``, not backslash
- **Tests:** Run `go test ./...` to verify all 110 tests pass
- **Dashboard:** Embedded at compile time; see `coordinator/static/`

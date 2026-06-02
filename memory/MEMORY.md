# Memory Index

**Updated:** June 2, 2026 12:45pm EDT | **Current Version:** v0.2.1

## Current Status

✅ **v0.2.1 Release Finalization Complete** — Fresh install testing on Windows
  - Coordinator service: ✅ Running, health check passing
  - Agent service: ✅ Running, registered with coordinator
  - Browser test: ✅ Login works, dashboard loads, agents page shows registered agent
  - Agent registration: ✅ DESKTOP-EE77F38 online and responding to heartbeats
  - All systems operational
🎯 **Next:** Final verification, tag v0.2.1, prepare for release

## Memory Files

- [Phase 22 Complete](phase22_complete.md) — Full integration testing suite, stress tests, agent disconnect recovery validation
- [History Tab Fix](history_tab_fix.md) — Bug fix for Agent Run Breakdown chart (root causes + solutions)
- [Phase 21a-4 Implementation](phase21a4_implementation.md) — Jobs stuck in pending hot fix (sync_flags + robocopy flags)
- [Phase 21a-4 Lessons Learned](phase21a4_lessons_learned.md) — Debugging insights from hot fix

## Quick Reference

### Latest Issues Fixed (May 29, 2026)

| Issue | Root Cause | Fix |
|-------|-----------|-----|
| History tab blank | Missing API parameters + no backend filtering + missing agent_id | Added after/search/status params to API, fixed JOIN, updated status field |
| Job status stuck as running | Status field not updated on completion | Updated job_results.go to set status based on exit_code + migration to fix historical data |
| Agent Run Breakdown not grouping | job_runs table lacks agent_id column | Always JOIN with jobs table to get agent_id |

### Phase 22 Key Results

- ✅ Agent disconnect recovery: 100% success rate
- ✅ Linear scaling to 100 agents: ~1000 jobs/sec throughput
- ✅ Memory efficient: 0.3MB for 100 agents
- ✅ Edge cases covered: large paths, high file counts, permissions, disconnects at 50%

### Windows Service Installation Notes (Session 6, June 2)

**Setup Wizard Behavior:**
- Interactive Go-based CLI, not CLI-parameterizable
- Accepts input for: installation type (1=Coordinator, 2=Agent, etc.), port, HTTPS flag, confirmation
- Creates two registry-based Windows services:
  - `arcvault-coordinator` → runs `C:\Projects\ArcVault2.0\installer\windows\coordinator.exe run-service`
  - `arcvault-agent` → runs `C:\Projects\ArcVault2.0\installer\windows\agent.exe run-service`
- Generates config files in same directory: `config.json` (coordinator), `agent-config.yaml` (agent)
- Uses dev binaries, not production install path (no C:\Program Files\ArcVault)

**Agent Service Startup Issue — RESOLVED (Session 6, 13:37 UTC)**
- **Problem:** Agent service exit code 1067, agent couldn't register with coordinator (401 Unauthorized)
- **Root Cause:** Token mismatch between agent config and coordinator database
  - Service loads config from `installer/windows/config.json` (not project root)
  - Agent token was invalid or not in coordinator's tokens table
  - Agent registration failed on startup, causing service crash
- **Solution:** 
  1. Regenerate coordinator and agent tokens using coordinator from installer directory
  2. Ensure both configs use the same database path and valid tokens
  3. Restart services to reload config
- **Test Results:** ✅ Agent now registers as DESKTOP-EE77F38, online, heartbeat working
- **Lesson:** Config file location depends on executable location — paths are relative to exe directory via `filepath.Join(filepath.Dir(exe), "config.json")`

### Production Readiness

ArcVault v0.2.1 is **RELEASE READY** pending:
- ✅ Coordinator functionality verified
- ⚠️ Agent service startup issue (investigation needed)
- ✅ Dashboard 401 fix deployed (redirects to login, no API errors on fresh session)
- ✅ Service naming standardized (arcvault-coordinator, arcvault-agent)
- ⏳ Full browser E2E test pending agent service resolution

---

**See also:** C:\Projects\ArcVault2.0\CONTEXT.md for full version history

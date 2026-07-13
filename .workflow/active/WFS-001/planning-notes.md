# Planning Notes

**Session**: WFS-001
**Created**: 2026-07-13

## User Intent (Phase 1)

- **GOAL**: Fix credential over-sharing — scope credential decryption so agents can only decrypt credentials for their own assigned jobs
- **KEY_CONSTRAINTS**:
  - Must not break existing agent registration or job execution
  - Existing clients expect job list to include credential metadata (not decrypted data)
  - Agent tokens already contain agent identity information
  - The project already has agent_id on jobs and credentials profiles

---

## Context Findings (Phase 2)
(To be filled)

## Conflict Decisions (Phase 3)
(To be filled if conflicts detected)

## Consolidated Constraints (Phase 4 Input)

---

## Task Generation (Phase 4)

See [IMPL_PLAN.md](IMPL_PLAN.md) for full implementation plan.

**Tasks**:
1. `coordinator/db/db.go` — Add `GetAgentIDByToken` function
2. `coordinator/server/server.go` — Add `AgentIDCtxKey`, `getAgentIDFromContext`, store agent_id in middleware
3. `coordinator/server/jobs.go` — Enforce ownership check before decrypting credentials
4. Tests — Add scoping verification tests

## N+1 Context
### Decisions
| Decision | Rationale | Revisit? |
|----------|-----------|----------|

### Deferred
- [ ] (For N+1)

---
name: Phase 18 Planning
status: active
created: 2026-05-26
last_updated: 2026-05-26
---

# Phase 18 Planning

## Objective

Decide what to build next for ArcVault after v1.0.0. The project is production-ready. Phase 18+ is exploratory.

## Constraints

- Maintain single binary deployment model
- No CGO dependencies
- All new features must not regress existing 110 tests
- Favor features that serve the core use case: cross-platform backup orchestration with visibility

## Active Files

- `MEMORY.md` — Future Roadmap section
- `planning/CONTEXT.md` — planning workspace
- `CONTEXT.md` — current project status

## Required Skills

None yet — this is a planning task.

## Success Criteria

- A Phase 18 spec document written to `planning/` or `docs/superpowers/specs/`
- Clear decision on which Phase 18 feature to tackle first

## Current Blockers

None — waiting for decision on Phase 18 direction.

## Recent Decisions

- v1.0.0 released (Phase 17 complete, 2026-05-22)
- Framework integration complete (2026-05-26)

## Candidate Features (from MEMORY.md Future Roadmap)

- CLI tooling for headless operations
- Additional sync backends (S3, Azure Blob, etc.)
- Advanced reporting and compliance export
- User search/filter in admin panel
- Password reset via email
- Email TLS client certificate auth

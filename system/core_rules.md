---
name: ArcVault Core Rules
category: system
priority: critical
last_updated: 2026-05-26
stale_after_days: 180
---

# ArcVault Core Rules

These rules govern all AI-assisted work on this project. They apply regardless of which modules or skills are loaded.

## Development Rules

1. **Run tests before declaring anything done.** `go test ./...` must pass. The current baseline is 110 tests (108 pass + 2 skip on Windows). Never regress this.
2. **Never touch MEMORY.md.** It is the full historical archive. Read it; never overwrite or delete it.
3. **Keep `CONTEXT.md` current.** After any phase completion, update the status section.
4. **Additive DB migrations only.** Never drop columns. Always add new columns with defaults or nullable.
5. **Single binary constraint.** The coordinator must remain a single deployable binary. Dashboard is embedded; never require a separate web server.
6. **No CGO.** Use `modernc.org/sqlite`. If a library requires CGO, find an alternative.
7. **Async for background work.** HTTP handlers must never block on background operations (notifications, retries, etc.). Use goroutines.
8. **Archive before deleting.** For any file, task, or documentation: archive first, delete never without review.

## Context Rules

9. **Load `framework/framework_runtime.md` at session start.** The routing table there governs what else to load.
10. **Prefer `memory/decisions.md` over re-deriving decisions.** Check it before making architectural choices.

## Communication Rules

11. **State the current phase.** When starting work, confirm what phase or feature is being worked on.
12. **Flag test regressions immediately.** If `go test ./...` fails after a change, stop and fix before continuing.
13. **Surface blockers explicitly.** Don't guess around a blocker — state it and ask.

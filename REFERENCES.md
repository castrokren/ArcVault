# REFERENCES — ArcVault 2.0

Last updated: 2026-07-22

## Naming conventions
- Go packages: lowercase, single word (`config`, `server`, `db`)
- Go files: snake_case (`agent_config.go`)
- Vue components: PascalCase
- API routes: kebab-case, prefixed with `/api/`

## Code style conventions observed in the codebase
- Go tests live beside the code they test, named `<file>_test.go` (e.g. `agents.go` /
  `agents_test.go`), one test file per source file — not a separate `tests/` tree, except
  `coordinator/tests/` for cross-cutting integration tests.
- Coordinator layering is enforced by convention, not a compiler boundary: `server/`
  handlers call into `business/` services; `business/` calls into `db/`. Don't have a
  handler call `db/` directly.
- Dashboard API calls go through `src/api.ts` and are validated against `src/schemas/`
  (Zod) — don't call `fetch`/`axios` directly from a component or view.
- PowerShell build/ops scripts live in `scripts/`, one script per concern
  (`rebuild-and-restart.ps1`, `check-sanity.ps1`, `check-version-sync.ps1`, `diagnose.ps1`).

## File naming patterns
- Contract-tested architecture docs: `docs/backend.md`, `docs/frontend.md`,
  `docs/service.md` — each carries a `<!-- CONTRACT:name -->` block checked against source
  by `internal/docs` (Go) or `dashboard/src/docs` (vitest). See `docs/itworks.md`.
- Decision records (if/when added): `YYYY-MM-DD-<decision-title>.md`.
- Task state files: `tasks/<phase>/STATE.md` (schema in the global `~/.claude/STATE.template.md`).

## API naming patterns
- Routes are kebab-case and prefixed `/api/`, e.g. `/api/agent-tokens`,
  `/api/alert-rules`.
- Route inventory is the contract-tested source of truth: `docs/backend.md`'s
  `CONTRACT:routes` block, generated from `coordinator/server/server.go:registerRoutes()`.
  Don't hand-maintain a separate route list — run `go test ./internal/docs/ -run Doc` and
  paste its corrected block.

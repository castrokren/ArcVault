# It Works — the doc-drift workflow

**Principle: a doc that describes the system must prove it is still true, or the change that
broke it does not land.** Prose rots the moment code moves. So ArcVault's architecture docs carry
machine-checked **contract blocks** that a test compares against the real code. Drift fails a test;
the pre-commit hook turns that failure into a blocked commit.

This is part of the standard workflow — not optional cleanup. If you touch a route, a view, or a
CLI subcommand, you update its doc in the same change.

## The contract-tested docs

| Doc | What its contract locks | Checked by |
|-----|-------------------------|------------|
| [`backend.md`](backend.md)  | every `METHOD /path` registered in `coordinator/server/server.go` | `internal/docs` (Go) |
| [`service.md`](service.md)  | the two service names + each `main.go` subcommand | `internal/docs` (Go) |
| [`frontend.md`](../dashboard/docs/frontend.md) | every Vue router route + its role guard | `dashboard/src/docs` (vitest) |

`docs/FUNCTIONS.md` and `docs/FEATURES.md` are retired redirects — their inventories moved into the
docs above precisely because, untested, they had already drifted.

## How a contract block works

Each doc has one or more blocks wrapped in HTML comments (invisible when rendered):

```
<!-- CONTRACT:routes -->
- `GET /api/agents`
- `POST /api/agents/register`
<!-- /CONTRACT:routes -->
```

The test extracts the `- ` bullets, builds a set, and asserts **set-equality** with the live source
of truth. On mismatch it prints exactly what's missing/extra **and the full corrected block to
paste back**. Prose outside the markers is never tested — describe freely.

## One-time setup (per clone)

```powershell
.\scripts\install-hooks.ps1     # points git at scripts/git-hooks (sets core.hooksPath)
```

After this, every `git commit` runs the doc-drift tests first and blocks the commit if a doc is out
of sync.

## The workflow — when you change the system

**Add / change / remove an HTTP route** (`coordinator/server/`):
1. Write the handler and register it in `server.go:registerRoutes()`.
2. Update the `CONTRACT:routes` block in `backend.md`.
3. `go test ./internal/docs/ -run Doc` — if it fails, paste the corrected block it prints.

**Add / change / remove a dashboard view or route** (`dashboard/src/`):
1. Add the component + route (with `meta.requiresRole` if admin-only) in `router/index.js`.
2. Update the `CONTRACT:routes` block in `dashboard/docs/frontend.md`.
3. `cd dashboard && npx vitest run src/docs`.

**Add / change / remove a CLI subcommand or rename a service** (`coordinator/main.go`,
`agent/main.go`, `*/service/`):
1. Add the `case "<cmd>":` (or change the `…ServiceName` const).
2. Update the matching `CONTRACT:` block in `service.md`.
3. `go test ./internal/docs/ -run Doc`.

You do not hand-compute the block. Run the test, copy the corrected block from its failure output,
paste, re-run. Green means the doc and the code agree.

## Running the checks

| Command | Runs |
|---------|------|
| `git commit` | doc-drift tests via the pre-commit hook (blocks on drift) |
| `go test ./internal/docs/ -run Doc` | backend + service contracts |
| `cd dashboard && npx vitest run src/docs` | frontend contract |
| `.\scripts\check-sanity.ps1` | backend + service contracts (section 6), plus deploy smoke tests |
| `.\scripts\rebuild-and-restart.ps1` | full build + `check-sanity.ps1` |

## What is and isn't tested

- **Tested (deterministic):** the inventory contracts — routes, views + role guards, service names,
  subcommands. These have a single source of truth to diff against.
- **Not tested (prose):** architecture explanations, feature descriptions, the 1067 runbook, design
  notes. There is no deterministic anchor for prose, and an AI-judged prose test would fail-false
  constantly. Keep prose accurate by habit; keep the contracts accurate by the test.

## Extending the pattern

To make a new list test-enforced: add a `CONTRACT:<name>` block to the doc, then add a test that
extracts the live set from source and calls the same set-equality assertion
(`assertSetEqual` in `internal/docs/doc_test.go`, or the `parseContract` helper in
`dashboard/src/docs/frontend.doc.test.js`). Wire nothing else — the hook already runs `-run Doc`
and `src/docs`.

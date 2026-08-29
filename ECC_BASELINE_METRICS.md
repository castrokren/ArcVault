# ECC Adoption — Baseline Metrics

**Date:** 2026-07-01
**Branch:** ecc-adoption-baseline
**Commit:** b5786ee (Pre-ECC baseline commit)

## Existing Framework Files

| Count | Location |
|-------|----------|
| 10    | `framework/` (all `*.md`) |

Files:
- framework/framework_runtime.md
- framework/modules/maintenance.md
- framework/modules/memory.md
- framework/modules/routing.md
- framework/modules/skills.md
- framework/modules/tasks.md
- framework/reference/automation_scripts.md
- framework/reference/folder_structure.md
- framework/reference/scaling_stages.md
- framework/reference/examples/vendor_pdf_workflow.md

## Context Size

| Directory | Files | Total Size (bytes) |
|-----------|-------|-------------------|
| `framework/` | 10 | — |
| `system/` | 3 | — |
| `.claude/` | 5 | — |
| **Total** | **18** | **90,957 bytes (~89 KB)** |

## Test Suite Time

| Metric | Value |
|--------|-------|
| Total time | 40.47 seconds |
| `go test ./...` | 40,466 ms |

## Build Time

| Metric | Value |
|--------|-------|
| Total time | 33.05 seconds |
| `go build ./coordinator/` | 33,052 ms |

## Session Startup Time & Token Usage

- **Session startup time:** TBD — will measure during TASK-25
- **Token usage per session:** TBD — will measure during TASK-25

# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Identity

You are helping Kren Castro build ArcVault — a cross-platform backup orchestrator in Go with a Vue 3 dashboard.

## Workspaces

| Workspace | Folder | Purpose |
|---|---|---|
| Planning | planning/ | Deciding what to build next |
| Building | coordinator/, agent/, dashboard/ | Go code, Vue frontend, testing and debugging |

## Routing Table

| Task | Go to | Read | Skills |
|---|---|---|---|
| Deciding what to build next | planning/ | CONTEXT.md, planning/CONTEXT.md | — |
| Writing Go code | coordinator/ or agent/ | CONTEXT.md, src/CONTEXT.md | systematic-debugging |
| Building Vue dashboard | dashboard/ | CONTEXT.md, src/CONTEXT.md | systematic-debugging |
| Debugging or testing | root | src/CONTEXT.md | systematic-debugging |
| Starting a new session | root | CLAUDE.md, CONTEXT.md | — |

## Naming Conventions

- Go packages: lowercase, single word (`config`, `server`, `db`)
- Go files: snake_case (`agent_config.go`)
- Vue components: PascalCase
- API routes: kebab-case, prefixed with `/api/`

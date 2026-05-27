# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Identity

You are helping Kren Castro build ArcVault — a cross-platform backup orchestrator in Go with a Vue 3 dashboard.

## Framework

This project uses a modular AI context framework. On every new session, load:

```
framework/framework_runtime.md
```

That file bootstraps all further context loading via its routing table. Do not skip it.

## Workspaces

| Workspace | Folder | Purpose |
|---|---|---|
| Planning | planning/ | Deciding what to build next |
| Building | coordinator/, agent/, dashboard/ | Go code, Vue frontend, testing and debugging |
| Framework | framework/ | AI context modules, memory, reference docs |
| System | system/ | Project identity, coding standards, core rules |
| Memory | memory/ | Persistent decisions, lessons, long-term knowledge |
| Tasks | tasks/ | Active work, completed and archived tasks |

## Routing Table

| Task | Go to | Read | Skills |
|---|---|---|---|
| Starting a new session | root | CLAUDE.md, CONTEXT.md, framework/framework_runtime.md | — |
| Deciding what to build next | planning/ | CONTEXT.md, planning/CONTEXT.md | — |
| Writing Go code | coordinator/ or agent/ | CONTEXT.md, src/CONTEXT.md | systematic-debugging |
| Building Vue dashboard | dashboard/ | CONTEXT.md, src/CONTEXT.md | systematic-debugging |
| Debugging or testing | root | src/CONTEXT.md | systematic-debugging |
| Storing or recalling decisions | memory/ | memory/decisions.md | — |
| Managing tasks or phases | tasks/ | tasks/active/ | — |
| Framework or context work | framework/ | framework/framework_runtime.md | — |

## Naming Conventions

- Go packages: lowercase, single word (`config`, `server`, `db`)
- Go files: snake_case (`agent_config.go`)
- Vue components: PascalCase
- API routes: kebab-case, prefixed with `/api/`

## Memory Reference

- **Full phase history & design decisions:** `MEMORY.md` (detailed archive, do not overwrite)
- **Lean active memory:** `memory/` (distilled decisions, lessons, patterns — update these)
- **Quick project status:** `CONTEXT.md`

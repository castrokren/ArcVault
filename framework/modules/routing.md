---
name: Routing Engine
category: routing
priority: critical
last_updated: 2026-05-26
stale_after_days: 60
related_files: [framework_runtime.md]
---

# Routing Engine

## Purpose

Classifies incoming tasks, scores file relevance, and assembles the minimal execution context needed to complete the task.

---

## Task Classification

Classify the task by matching keywords against the domain lists below. A task may match multiple domains — load all relevant modules.

### General Domains

| Domain | Keywords |
|---|---|
| Coding | code, function, bug, debug, implement, refactor, test, script, module, class, error, exception |
| Automation | automate, pipeline, workflow, schedule, trigger, batch, run, process, extract, parse |
| Data Analysis | analyze, data, spreadsheet, csv, sql, query, report, chart, metrics, aggregate, filter |
| Memory | remember, decision, preference, lesson, history, context, recall, store, retrieve, log |
| Maintenance | stale, outdated, archive, prune, summarize, cleanup, refresh, update docs, reorganize |
| Skills | skill, template, reusable, pattern, best practice, workflow, encapsulate |
| Tasks | task, todo, objective, goal, complete, block, queue, active, progress |

### ArcVault-Specific Domains

| Domain | Keywords | Also Load |
|---|---|---|
| Coordinator (Go) | coordinator, server, handler, endpoint, api, route, middleware, db, sqlite, migration | `src/CONTEXT.md` |
| Agent (Go) | agent, heartbeat, websocket, ws, runner, executor, updater, service, failover | `src/CONTEXT.md` |
| Dashboard (Vue) | dashboard, vue, component, composable, view, router, frontend, ui, login, auth | `src/CONTEXT.md` |
| Federation | federation, sync, spoke, root, coordinator-to-coordinator, ha, health, lag, replication | `memory/decisions.md` |
| Auth & RBAC | jwt, auth, token, role, admin, operator, viewer, permission, rbac, user, group | `memory/decisions.md` |
| Notifications | notification, webhook, slack, teams, email, alert, retry, dispatch | `memory/decisions.md` |
| Installer / Deployment | installer, build, release, goreleaser, nsis, package, deploy, dist, binary | — |
| Planning | phase, roadmap, next, feature, design, spec, plan, backlog | `planning/CONTEXT.md` |

---

## Relevance Scoring

Score each candidate file before loading. Only load files that meet the threshold.

```text
score = (keyword_matches × 10) + (recent_usage_bonus) − (days_since_accessed × 0.5)

recent_usage_bonus:
  accessed today       → +20
  accessed this week   → +10
  accessed this month  → +5
  older                → +0

Load file if: score ≥ 30
Skip file if: score < 30
```

This formula is illustrative — weights may be tuned per project. The structure (keyword match + recency bonus − staleness penalty, with an explicit load threshold) must be preserved.

**Example:**

| File | Keyword Matches | Last Accessed | Score | Load? |
|---|---|---|---|---|
| `vendor_matching.md` | 3 (vendor, pdf, extract) | 2 days ago | 30 + 10 − 1 = **39** | Yes |
| `sql_analysis.md` | 0 | 14 days ago | 0 + 0 − 7 = **−7** | No |
| `ui_design.md` | 0 | 45 days ago | 0 + 0 − 22.5 = **−22.5** | No |

---

## Context Assembly

Build the execution context in this order:

1. **System rules** — always included (from `framework_runtime.md` mandatory rules)
2. **Active task** — current objective, constraints, success criteria
3. **Required skills** — all skills matched by relevance scoring
4. **Recent decisions** — last 3–5 relevant entries from `modules/memory.md`

Compress redundant context before assembling. Prioritize recent decisions over historical detail.

**Example output:**

```text
EXECUTION CONTEXT

SYSTEM RULES:
- [mandatory rules from framework_runtime.md]

ACTIVE TASK:
- Process vendor spreadsheet and match against PDF database

REQUIRED SKILLS:
- vendor_matching
- pdf_extraction
- excel_cleaning

RECENT DECISIONS:
- Normalize all vendor names to uppercase before matching
```

---

## Routing Decision Log

After each routing decision, log:
- Task keywords matched
- Modules loaded
- Files scored and threshold result
- Context assembled

This log feeds the self-reflection routine in `modules/maintenance.md`.

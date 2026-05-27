---
name: Task Manager
category: system
priority: high
last_updated: 2026-05-26
stale_after_days: 60
related_files: [framework_runtime.md, modules/memory.md, modules/skills.md]
---

# Task Manager

## Purpose

Governs task lifecycle from creation through archival. Defines task file structure, state transitions, size constraints, and the completion protocol.

---

## Task File Structure

Every active task file must include these sections:

```markdown
---
name:
status:         # active | completed | archived | failed
created:        # YYYY-MM-DD
last_updated:   # YYYY-MM-DD
---

# [Task Name]

## Objective
What this task is trying to accomplish.

## Constraints
Hard limits, requirements, or non-negotiables.

## Active Files
Files being read or modified during this task.

## Required Skills
Skill files loaded for this task.

## Success Criteria
How to know the task is done.

## Current Blockers
Anything preventing progress.

## Recent Decisions
Decisions made during this task (last 3–5, most recent first).
```

Task files must stay ≤ 200 lines. If a task grows beyond this, extract detail into memory or a reference file and keep the task file as a lightweight pointer.

---

## Task States

| State | Meaning | Transition |
|---|---|---|
| `active` | Currently in progress | → `completed` when success criteria met; → `failed` if unresolvable blocker |
| `completed` | Done, pending archival | → `archived` after completion protocol runs |
| `archived` | Preserved for reference | Terminal state |
| `failed` | Stopped due to blocker | → `archived` after failure is documented |

---

## Completion Protocol

When a task reaches `completed` or `failed`, run in order:

1. **Summarize** — Write a 3–5 sentence summary of what was accomplished and any key outcomes.
2. **Extract knowledge** — Run the knowledge extraction check from `modules/skills.md`. Create or update skills if applicable.
3. **Update memory** — Promote relevant decisions, lessons, and patterns to the appropriate memory categories per `modules/memory.md`.
4. **Run self-reflection** — If this was a major task (per `framework_runtime.md`), run the 7-question routine in `modules/maintenance.md`.
5. **Archive** — Move the task file from `/tasks/active/` to `/tasks/archived/`. Update status metadata to `archived`.

---

## Task Directories

```text
/tasks/
├── active/       ← Tasks currently in progress
├── completed/    ← Tasks done, pending archival
├── archived/     ← Completed and failed tasks, preserved
└── failed/       ← Optional: failed tasks before archival
```

Keep `/tasks/active/` lean — completed tasks should move promptly to avoid clutter.

---

## Lightweight Task Principle

A task file is a coordination tool, not a document. It should be:
- Short enough to scan in 30 seconds
- Focused on current state, not history
- A pointer to detail, not a container for it

When a task file starts growing, extract the growing content to memory or reference files and replace it with a link.

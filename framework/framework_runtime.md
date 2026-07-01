---
name: AI Framework Runtime
category: system
priority: critical
last_updated: 2026-07-01
stale_after_days: 90
---

# AI Framework Runtime

## Bootstrapping

Read this file completely before taking any action. Use the routing table below to determine which module files to load for the current task. Do not load modules that are not relevant to the task at hand.

---

## Routing Table

| Task Context | Load Module |
|---|---|
| New task begins | `modules/routing.md` |
| Storing or retrieving decisions / memory | `modules/memory.md` |
| Files flagged as stale / session ending | `modules/maintenance.md` |
| Creating or updating a skill file | `modules/skills.md` |
| Managing task lifecycle | `modules/tasks.md` |
| Go development | `rules/golang/` |

Multiple modules may be loaded for a single task. These entries are not mutually exclusive.

---

## Capabilities Check

Before any file-system operation, confirm available tools:

```text
IF file_write_tools available:
    → operate normally — read, write, archive, update files
ELSE:
    → read-only mode — surface recommendations as output only
    → do not attempt writes, deletes, or archives
```

When in read-only mode, state this explicitly at the start of the response.

---

## Universal Mandatory Rules

1. Never load unnecessary context — only load modules required by the current task.
2. Prefer modular files over monolithic prompts — one concern per file.
3. Keep files concise — see size limits in `reference/folder_structure.md`.
4. Archive before deleting — never permanently remove content without archiving first.
5. Detect stale information continuously — use metadata thresholds, not intuition.
6. Summarize repetitive information — compress, preserve critical decisions, archive raw detail.
7. Convert repeated workflows into reusable skills — don't re-prompt what can be encapsulated.
8. Track dependencies between files — use `related_files` and `dependencies` metadata fields.
9. Preserve critical decisions — architecture choices, security rules, user preferences are permanent.
10. Maintain deterministic routing — the same task type must always load the same modules.

---

## Self-Reflection Trigger

A task is **major** if ANY of the following are true:
- It produced a new or modified file
- It involved more than 3 tool calls
- The user explicitly marked it complete

After a major task, run the self-reflection routine defined in `modules/maintenance.md`.

---

## Metadata Standard

Every framework file must include this YAML frontmatter:

```yaml
---
name:
category:           # system | skill | task | memory | routing | reference
priority:           # critical | high | medium | low
last_updated:       # YYYY-MM-DD
last_accessed:      # YYYY-MM-DD (update on each access)
relevance_tags: []
dependencies: []
related_files: []
stale_after_days:
auto_summarize: true
archive_policy:     # archive | delete | keep
---
```

Fields `name`, `category`, `priority`, `last_updated`, and `stale_after_days` are required. All others are optional but recommended.

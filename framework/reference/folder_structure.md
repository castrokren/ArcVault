# Canonical Folder Structure

> **Human reference only.** This file is not loaded at AI runtime.

This is the single authoritative folder layout for projects using this framework. Do not create alternative layouts — if this structure doesn't fit a project, update this file and document the reason.

---

## Full Directory Tree

```text
/project-root
│
├── framework_runtime.md          ← Always-loaded AI parent file
│
├── /framework
│   ├── /modules                  ← On-demand AI runtime modules
│   │   ├── routing.md
│   │   ├── memory.md
│   │   ├── maintenance.md
│   │   ├── skills.md
│   │   └── tasks.md
│   │
│   └── /reference                ← Human-facing documentation only
│       ├── folder_structure.md   ← This file
│       ├── scaling_stages.md
│       ├── automation_scripts.md
│       └── /examples
│           └── vendor_pdf_workflow.md
│
├── /system
│   ├── core_rules.md             ← Universal operational rules
│   ├── identity.md               ← AI identity and role definition
│   ├── communication_rules.md
│   ├── safety_rules.md
│   ├── coding_standards.md
│   └── execution_constraints.md
│
├── /skills
│   ├── /templates
│   ├── /coding
│   ├── /automation
│   ├── /analysis
│   ├── /integrations
│   └── /operations
│
├── /memory
│   ├── short_term.md             ← Active session context
│   ├── long_term.md              ← Stable persistent knowledge
│   ├── decisions.md              ← Architecture and design choices
│   ├── lessons_learned.md        ← Recurring failures and successes
│   └── patterns.md               ← Reusable operational logic
│
├── /tasks
│   ├── /active
│   ├── /completed
│   ├── /archived
│   └── /failed
│
├── /logs
│   ├── /updates
│   ├── /audits
│   ├── /failures
│   └── /optimization
│
└── /automation
    ├── refresh_rules.md
    ├── cleanup_rules.md
    ├── summarization_rules.md
    └── stale_detection.md
```

---

## Naming Decisions

These naming choices resolve inconsistencies between earlier versions of the framework:

| Decision | Chosen | Rejected | Reason |
|---|---|---|---|
| System rules filename | `core_rules.md` | `operating_rules.md` | More precise — "core" signals primacy |
| Short-term memory | `short_term.md` (file) | `short_term/` (directory) | Consistent with other memory files; simpler |
| Memory directory | flat files | nested subdirectories | Easier to scan, consistent metadata |

---

## File Size Limits

| File Type | Recommended Limit |
|---|---|
| `framework_runtime.md` | ≤ 200 lines |
| Module files | ≤ 400 lines |
| System rules | ≤ 300 lines |
| Skill files | ≤ 500 lines |
| Active task files | ≤ 200 lines |
| Memory files | ≤ 400 lines |

Files exceeding these limits are candidates for splitting. See `modules/maintenance.md` for split rules.

---

## Final Principle

This framework exists to ensure AI systems remain organized, context remains relevant, workflows remain modular, and knowledge remains reusable — without requiring heavy enterprise agent frameworks.

The framework should evolve like a disciplined software system: modular, version-controlled, and maintained with the same care as production code. When it stops serving the project, update it. When it grows too large, split it. When it becomes stale, archive it.

The goal is not a perfect system on day one. The goal is a system that gets better every time it's used.

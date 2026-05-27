---
name: Memory Manager
category: memory
priority: high
last_updated: 2026-05-26
stale_after_days: 60
related_files: [framework_runtime.md, modules/maintenance.md]
---

# Memory Manager

## Purpose

Defines memory categories, governs when each is read vs. written, and enforces summarization and archival rules.

---

## Memory Categories

| Category | File | Purpose | Lifespan |
|---|---|---|---|
| Short-Term | `memory/short_term.md` | Active session context — what's in progress right now | Current session only |
| Long-Term | `memory/long_term.md` | Stable, reusable knowledge that persists across sessions | Indefinite |
| Decisions | `memory/decisions.md` | Important architecture and design choices with rationale | Indefinite |
| Lessons Learned | `memory/lessons_learned.md` | Recurring failures and successes worth preserving | Indefinite |
| Patterns | `memory/patterns.md` | Reusable operational logic detected across tasks | Indefinite |

---

## Read vs. Write Conditions

### Short-Term Memory

**Read when:** Starting a new task — check what was in progress.
**Write when:** Task state changes, a blocker appears, or a decision is made mid-task.
**Clear when:** Task completes and knowledge has been promoted to long-term or decisions.

### Long-Term Memory

**Read when:** Starting work in a familiar domain — check for prior knowledge.
**Write when:** A completed task produces stable, reusable knowledge not already captured.
**Never overwrite:** Append new knowledge; archive superseded entries.

### Decisions

**Read when:** About to make an architecture, security, or workflow choice.
**Write when:** A consequential decision is made. Include: what was decided, why, and what alternatives were rejected.
**Never delete:** Archive if superseded, never remove.

### Lessons Learned

**Read when:** Encountering a failure pattern or debugging a recurring issue.
**Write when:** A failure or success is worth preserving as a general lesson.

### Patterns

**Read when:** Routing engine detects a recurring workflow.
**Write when:** The same workflow appears in 3 or more distinct tasks.

---

## Summarization Rules

### Preserve

- Architecture decisions and rationale
- Security rules and constraints
- Successful fixes to recurring problems
- User preferences and working style
- Critical external dependency versions

### Compress

- Repetitive debugging sessions → extract the fix, discard the steps
- Long task conversations → extract decisions and lessons, discard dialogue
- Duplicate examples → keep the clearest one, discard the rest
- Temporary experimentation → discard unless it produced a lesson

### Archive

- Obsolete systems and deprecated APIs
- Decisions that have been superseded (keep the superseding decision)
- Inactive projects and their context

---

## Archive-Before-Delete Rule

No memory content is ever permanently deleted without first being archived.

```text
Before removing any memory entry:
  1. Copy to /tasks/archived/ or /memory/long_term.md archive section
  2. Add archive timestamp and reason
  3. Then remove from active file
```

---

## Memory Update Protocol

After completing a major task (as defined in `framework_runtime.md`):

1. Promote relevant short-term entries to long-term, decisions, or lessons learned
2. Summarize any memory files that have grown beyond their size limit
3. Clear short-term memory of completed task state
4. Update `last_accessed` metadata on any memory file read during the task

---
name: Maintenance Engine
category: system
priority: high
last_updated: 2026-05-26
stale_after_days: 60
related_files: [framework_runtime.md, modules/memory.md]
---

# Maintenance Engine

## Purpose

Keeps the framework healthy. Covers stale detection, pruning, summarization, and the self-reflection routine triggered after major tasks.

---

## Stale Detection

A file is stale when ANY of the following conditions are true:

```text
STALE if:
  - days since last_accessed > stale_after_days (from file metadata)
  - a dependency listed in metadata has changed version
  - a conflicting file exists with overlapping content
  - usage_frequency has been low for 3+ consecutive sessions
  - referenced systems, APIs, or files have been removed
  - the workflow this file describes has changed significantly
```

When a file is detected as stale:
1. Flag it with a `stale: true` metadata field and today's date
2. Do not delete immediately — review in the next maintenance pass
3. Options per file: update, merge into another file, summarize and archive, or archive entirely

---

## Pruning Rules

Remove content from active files when:
- It duplicates content in another file
- It describes a deprecated API or obsolete system
- It captures failed experiments with no extractable lesson
- It is outdated workflow documentation that has been replaced

**Always archive before pruning.** See archive-before-delete rule in `modules/memory.md`.

---

## Optimization Rules

### Merge

```text
IF two skill files have content overlap ≥ 70%:
  → merge into one file
  → update all references
  → archive the merged files
```

### Split

```text
IF a file exceeds its recommended size limit:
  → identify distinct sub-concerns
  → split into focused files by domain
  → update routing table if needed
```

### Compress Repetitive Logs

```text
IF a repeated issue appears in logs 3+ times:
  → summarize the issue and its standard fix
  → preserve the fix
  → archive the raw log entries
```

---

## Self-Reflection Routine

Triggered after any major task (as defined in `framework_runtime.md`). Run through these 7 questions in order:

1. **What changed?** — Which files were created, modified, or deleted?
2. **What new knowledge was created?** — Any decisions, lessons, or patterns worth preserving?
3. **Which skills were useful?** — Which skill files were loaded and did they perform well?
4. **Which files became outdated?** — Did any existing files become stale as a result of this task?
5. **What should be archived?** — Any task files, logs, or temporary context ready for archival?
6. **What patterns repeated?** — Did this task follow a workflow that has appeared before?
7. **What should become a reusable skill?** — Is there logic here worth encapsulating in `modules/skills.md`?

After answering all 7:
- Write any new knowledge to the appropriate memory category (`modules/memory.md`)
- Flag stale files identified in question 4
- Create or update skill files if question 7 has a yes answer
- Archive completed task context

---

## Periodic Maintenance Pass

Run a full maintenance pass at the end of a project phase or when the framework feels slow or cluttered:

1. Scan all files for stale metadata flags
2. Identify merge candidates (overlap ≥ 70%)
3. Identify split candidates (files exceeding size limits)
4. Compress repetitive logs
5. Promote short-term memory to long-term where appropriate
6. Re-rank relevance scores on high-traffic files
7. Refresh examples in skill files that are outdated

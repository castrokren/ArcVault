# Design Spec: AI Framework Restructure
**Date:** 2026-05-26
**Status:** Draft — pending user review
**Scope:** Restructure `universal_ai_self_structuring_skill_framework.md` and `ai_framework_companion_skill_modular_production_architecture.md` into a clean, maintainable multi-file architecture.

---

## Problem Statement

The two existing framework files have four critical issues when used as AI runtime instructions:

1. **Redundancy.** Both files cover stale detection, routing logic, self-reflection routines, maintenance rules, and mandatory rules — sometimes with slight differences. An AI loading both pays token cost twice and faces ambiguity when the files diverge.

2. **Inconsistent structure.** The files describe different folder layouts (e.g. `/system/operating_rules.md` vs `/system/core_rules.md`, `/memory/short_term/` directory vs `/memory/short_term.md` file). There is no single canonical reference.

3. **Bootstrapping gap.** The routing engine determines what files to load, but the routing rules are themselves in a file that must be loaded first. Neither document specifies what the AI reads *first* or how it knows to read anything at all.

4. **Aspirational but not operational.** Key behaviors (relevance scoring 0–100, self-reflection after "major tasks") are described as concepts without concrete algorithms or triggers. An AI following these instructions cannot act on them reliably.

---

## Solution: Hybrid Runtime + Modular Architecture

### Core Principle

Separate what the **AI loads at runtime** from what **humans read as reference**. Within the AI-facing side, the parent file is always minimal — it bootstraps everything else via a routing table.

---

## File Architecture

```text
/framework
│
├── framework_runtime.md          ← Always-loaded parent. Lean (~150–200 lines).
│
├── /modules                      ← Loaded on-demand per routing table.
│   ├── routing.md
│   ├── memory.md
│   ├── maintenance.md
│   ├── skills.md
│   └── tasks.md
│
└── /reference                    ← Human-facing only. Never loaded at AI runtime.
    ├── folder_structure.md
    ├── scaling_stages.md
    ├── automation_scripts.md
    └── /examples
        └── vendor_pdf_workflow.md
```

---

## Parent File: `framework_runtime.md`

This file is loaded first, always. It must stay lean. Its job is to bootstrap everything else.

### Required Sections (in order)

**1. Bootstrapping Declaration**
A single explicit statement: "This file is loaded first. Read it completely before taking any action. Use the routing table below to determine which module files to load."

**2. Routing Table**
Maps task context to module files. Format matches the CLAUDE.md routing table pattern — a simple markdown table. Example:

| Task Context | Load Module |
|---|---|
| New task begins | `modules/routing.md` |
| Storing or retrieving decisions/memory | `modules/memory.md` |
| Files flagged as stale / session ending | `modules/maintenance.md` |
| Creating or updating a skill file | `modules/skills.md` |
| Managing task lifecycle | `modules/tasks.md` |

Multiple modules may be loaded for a single task. The routing table entries are not mutually exclusive.

**3. Capabilities Check**
Before any file-system operation, the AI must confirm it has file-access tools available. If not, it operates in read-only mode: it can recommend actions but cannot write, archive, or update files.

```text
IF file_write_tools available:
    → operate normally
ELSE:
    → surface recommendations as output only, do not attempt writes
```

**4. Universal Mandatory Rules**
One authoritative list (not repeated in modules). Maximum 10 rules. Concise, enforceable.

**5. Self-Reflection Trigger**
Explicit definition of "major task" and the trigger condition:

```text
A task is major if ANY of:
- It produced a new or modified file
- It involved more than 3 tool calls
- The user explicitly marked it complete

After a major task:
  → run self-reflection routine (see modules/maintenance.md)
```

**6. Metadata Standard**
The canonical YAML frontmatter spec. Defined once here, referenced by modules.

```yaml
---
name:
category:
priority:
last_updated:
last_accessed:
relevance_tags: []
dependencies: []
stale_after_days:
auto_summarize: true
---
```

---

## Module Files

Each module is a focused document covering one concern. Modules may reference each other but should not duplicate content from the parent or from other modules.

### `modules/routing.md`

**Purpose:** Concrete routing and relevance-scoring logic.

**Must include:**
- Keyword-matching algorithm for task classification (explicit keyword lists per domain, not abstract descriptions)
- Relevance scoring method — replace the aspirational 0–100 score with a concrete rule-based approach. The formula below is illustrative; the implementation may tune weights, but the structure (keyword match + recency bonus − staleness penalty, with a load threshold) must be present and explicit:
  ```text
  score = (keyword_matches × 10) + (recent_usage_bonus) − (days_since_accessed × 0.5)
  Load file if score ≥ 30
  ```
- Context assembly rules: what gets included in the execution context, in what order

**Must not include:** Memory management rules, maintenance schedules, skill templates.

---

### `modules/memory.md`

**Purpose:** Memory categories, read/write rules, and update protocol.

**Must include:**
- Canonical memory category definitions (short-term, long-term, decisions, lessons learned, patterns)
- When each category is read vs. written
- Concise summarization rules (what to compress, what to preserve)
- Archive-before-delete rule

**Must not include:** Stale detection (that's maintenance), routing logic, skill templates.

---

### `modules/maintenance.md`

**Purpose:** Self-maintenance engine — stale detection, pruning, summarization, self-reflection routine.

**Must include:**
- Stale detection rules with concrete conditions (moved here from both source files — one authoritative version)
- Pruning rules with archive-before-delete enforcement
- Summarization priorities (preserve / compress / archive breakdown)
- The full self-reflection routine (the 7-question checklist), invoked by the trigger defined in the parent

**Must not include:** Routing logic, memory categories, skill templates.

---

### `modules/skills.md`

**Purpose:** Skill file format, creation protocol, and auto-update conditions.

**Must include:**
- The canonical skill file template (one authoritative version)
- Update conditions protocol
- Knowledge extraction rules — how completed tasks become new skills
- Merge/split optimization rules (≥70% overlap → merge; exceeds size threshold → split)

**Must not include:** Memory management, routing logic, maintenance schedules.

---

### `modules/tasks.md`

**Purpose:** Task lifecycle management.

**Must include:**
- Task file structure (objective, constraints, active files, required skills, success criteria, blockers, recent decisions)
- Task states: active / completed / archived / failed
- Completion protocol: summarize → extract knowledge → update memory → archive
- What makes a task "lightweight" (size limit: ≤200 lines)

**Must not include:** Memory management (just references it), skill creation (just references it).

---

## Reference Folder

These files are never loaded at AI runtime. They exist for humans setting up, maintaining, or extending the framework.

| File | Contents |
|---|---|
| `reference/folder_structure.md` | The canonical folder layout (one version, resolving the inconsistency between source files) |
| `reference/scaling_stages.md` | Stage 1–4 production scaling path |
| `reference/automation_scripts.md` | Recommended Python scripts (relevance_scanner, context_builder, summarizer, etc.) |
| `reference/examples/vendor_pdf_workflow.md` | The vendor PDF processing example, fully worked through |

---

## Resolving the Inconsistencies

The two source files must be reconciled on the following before writing the new files:

| Conflict | Source A | Source B | Resolution |
|---|---|---|---|
| System rules filename | `operating_rules.md` | `core_rules.md` | Use `core_rules.md` (more precise) |
| Short-term memory | `/memory/short_term/` (dir) | `/memory/short_term.md` (file) | Use file — simpler, consistent with other memory files |
| Mandatory rules list | 10 rules in companion | 10 rules in base (different wording) | Merge into one authoritative list of 10 in parent file |
| Self-reflection questions | 7 questions (companion) | 7 questions (base, slightly different) | Merge best of both into one canonical 7-question list |

---

## What Gets Cut

The following content from the source files does not survive the restructure:

- Duplicate "Final Principle" sections — collapsed into one and moved to `reference/folder_structure.md` as closing context for human readers; not deleted
- The "Recommended Production Scaling Path" section from the companion — moves entirely to `reference/scaling_stages.md`
- Abstract descriptions of relevance scoring (0–100 without algorithm) — replaced by concrete rule-based scoring in `modules/routing.md`
- Repeated mandatory rules lists — one list, in `framework_runtime.md` only

---

## Success Criteria

The restructure is complete when:

1. `framework_runtime.md` is ≤200 lines
2. Each module file covers exactly one concern with no cross-duplication
3. The routing table in the parent correctly maps to every module
4. The bootstrapping order is unambiguous (AI knows what to read first)
5. Relevance scoring has a concrete, reproducible algorithm
6. The self-reflection trigger defines "major task" explicitly
7. The capabilities check handles no-tool environments gracefully
8. The canonical folder structure exists in exactly one place

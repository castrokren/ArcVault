# Implementation Plan: AI Framework Restructure
**Date:** 2026-05-26
**Spec:** `docs/superpowers/specs/2026-05-26-framework-restructure-design.md`
**Status:** Ready to execute

---

## Pre-flight

- [ ] Source files are available in uploads:
  - `universal_ai_self_structuring_skill_framework.md`
  - `ai_framework_companion_skill_modular_production_architecture.md`
- [ ] Spec has been reviewed and approved ✓
- [ ] Output directory: create `/framework/` at project root (or wherever user designates)

---

## Task 1 — Create folder structure

Create the following empty directories:

```text
/framework/
/framework/modules/
/framework/reference/
/framework/reference/examples/
```

**Verification:** `ls /framework` shows `modules/`, `reference/`, and nothing else.

---

## Task 2 — Write `framework_runtime.md`

Author the lean parent file (target: ≤200 lines). Content in order:

1. **Bootstrapping Declaration** — one paragraph: "Read this file first. Use the routing table below to determine which module files to load before taking any action."

2. **Routing Table** — markdown table:

   | Task Context | Load Module |
   |---|---|
   | New task begins | `modules/routing.md` |
   | Storing or retrieving decisions/memory | `modules/memory.md` |
   | Files flagged as stale / session ending | `modules/maintenance.md` |
   | Creating or updating a skill file | `modules/skills.md` |
   | Managing task lifecycle | `modules/tasks.md` |

3. **Capabilities Check** — short block:
   ```text
   IF file_write_tools available → operate normally
   ELSE → surface recommendations as output only, do not attempt writes
   ```

4. **Universal Mandatory Rules** — merge both source files' 10-rule lists into one authoritative list of 10. Rules from source A that differ slightly from source B: take the more specific wording.

5. **Self-Reflection Trigger** — define "major task" explicitly:
   ```text
   A task is major if ANY of:
   - It produced a new or modified file
   - It involved more than 3 tool calls
   - The user explicitly marked it complete
   After a major task → run self-reflection routine (modules/maintenance.md)
   ```

6. **Metadata Standard** — the canonical YAML frontmatter block (from spec).

**Verification:** Line count ≤200. All 6 sections present. Routing table has 5 rows.

---

## Task 3 — Write `modules/routing.md`

Content:

- Task classification keyword lists per domain (coding, automation, data analysis, memory, maintenance — at minimum)
- Concrete relevance scoring formula (illustrative weights, explicit threshold):
  ```text
  score = (keyword_matches × 10) + (recent_usage_bonus) − (days_since_accessed × 0.5)
  Load file if score ≥ 30
  ```
- Context assembly rules: system rules first, then task, then skills, then recent decisions
- Example routing decision (use the vendor PDF example from source files)

**Verification:** No memory management content. No skill templates. Scoring formula is explicit and has a load threshold.

---

## Task 4 — Write `modules/memory.md`

Content:

- Canonical memory category table (short-term, long-term, decisions, lessons learned, patterns) with purpose for each
- Read vs. write conditions per category
- Summarization rules: what to compress (repetitive debugging, long conversations, duplicate examples), what to preserve (architecture decisions, successful fixes, user preferences)
- Archive-before-delete rule — stated once, authoritatively

**Verification:** No routing logic. No stale detection. No skill templates.

---

## Task 5 — Write `modules/maintenance.md`

Content:

- Stale detection rules — merge both source file versions into one canonical list:
  ```text
  Mark stale if ANY:
  - last_accessed > stale_after_days threshold
  - dependency version changed
  - conflicting file exists
  - usage_frequency == low
  - referenced systems removed
  ```
- Pruning rules with archive-before-delete enforcement
- Summarization priorities (preserve / compress / archive breakdown — from source files)
- Full self-reflection 7-question checklist — merge best of both source versions:
  1. What changed?
  2. What new knowledge was created?
  3. Which skills were useful?
  4. Which files became outdated?
  5. What should be archived?
  6. What patterns repeated?
  7. What should become a reusable skill?

**Verification:** Stale detection is one list with no duplicates. Self-reflection has exactly 7 questions. No routing logic or skill templates.

---

## Task 6 — Write `modules/skills.md`

Content:

- Canonical skill file template (one authoritative version — take from source B which is more complete):
  ```markdown
  ---
  [metadata block]
  ---
  # Purpose
  # Inputs
  # Outputs
  # Dependencies
  # Required Context
  # Workflow
  # Validation
  # Error Handling
  # Examples
  # Anti-Patterns
  # Performance Considerations
  # Security Considerations
  # Update Conditions
  # Changelog
  ```
- Update conditions protocol
- Knowledge extraction rules: how a completed task becomes a new skill
- Merge rule: ≥70% content overlap → merge skills
- Split rule: file exceeds 500 lines → split by domain

**Verification:** Template is complete (all 14 sections). No memory or routing content.

---

## Task 7 — Write `modules/tasks.md`

Content:

- Task file structure (objective, constraints, active files, required skills, success criteria, blockers, recent decisions)
- Task states: active / completed / archived / failed — with transition rules
- Completion protocol: summarize → extract knowledge → update memory → archive
- Size constraint: ≤200 lines for an active task file
- Reference (not repeat) memory and skill modules for post-completion steps

**Verification:** No memory rules duplicated. No skill templates duplicated. Completion protocol references modules, doesn't re-define them.

---

## Task 8 — Write `reference/folder_structure.md`

Content:

- The canonical folder layout — resolve the inconsistency between the two source files using the decisions from the spec:
  - Use `core_rules.md` (not `operating_rules.md`)
  - Use `/memory/short_term.md` (file, not directory)
- Full annotated directory tree
- Brief note on what belongs in each folder
- Consolidated "Final Principle" section (merged from both source files' closing sections)

**Verification:** Exactly one folder layout. No conflicts with module files.

---

## Task 9 — Write `reference/scaling_stages.md`

Content: Stage 1–4 production scaling path from the companion skill source file, verbatim with light cleanup.

- Stage 1 — Manual markdown routing (VS Code, Claude Code, Cursor, ChatGPT Projects)
- Stage 2 — Script-assisted routing (Python relevance scanner, context builder, summarization scripts)
- Stage 3 — Lightweight semantic search (embeddings, vector search)
- Stage 4 — Agentic automation (multi-agent orchestration, autonomous task generation)

**Verification:** 4 stages present. No AI runtime instructions (this is human reference only).

---

## Task 10 — Write `reference/automation_scripts.md`

Content: The recommended Python script table from both source files, merged and deduplicated:

| Script | Purpose |
|---|---|
| `relevance_scanner.py` | Detect relevant files, outdated references, missing metadata |
| `stale_detector.py` | Detect outdated content based on metadata thresholds |
| `context_builder.py` | Assemble minimal execution prompts |
| `summarizer.py` | Compress large markdown files |
| `knowledge_extractor.py` | Create reusable skills from completed tasks |
| `dependency_mapper.py` | Track relationships between files |
| `archive_manager.py` | Handle historical storage |

**Verification:** No duplicates. No AI runtime instructions.

---

## Task 11 — Write `reference/examples/vendor_pdf_workflow.md`

Content: The full vendor PDF processing example from the source files, written out as a complete worked example using the new module structure (which modules get loaded, in what order, what the routing table matches on).

**Verification:** References the new file paths (`modules/routing.md`, etc.), not the old source file structure.

---

## Task 12 — Final verification against spec Success Criteria

Check each of the 8 success criteria from the design spec:

1. `framework_runtime.md` is ≤200 lines → count lines
2. Each module covers exactly one concern with no cross-duplication → scan for repeated content
3. Routing table in parent correctly maps to every module → verify 5 rows, 5 module files
4. Bootstrapping order is unambiguous → confirm opening section in parent
5. Relevance scoring has a concrete algorithm → confirm formula and threshold in `modules/routing.md`
6. Self-reflection trigger defines "major task" explicitly → confirm definition in parent
7. Capabilities check handles no-tool environments → confirm section in parent
8. Canonical folder structure exists in exactly one place → confirm only in `reference/folder_structure.md`

**Done when:** All 8 pass.

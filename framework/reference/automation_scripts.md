# Automation Scripts

> **Human reference only.** This file is not loaded at AI runtime.

These scripts support Stage 2 and Stage 3 of the scaling path (see `scaling_stages.md`). They are optional at Stage 1 — the framework functions without them.

---

## Script Reference

| Script | Stage | Purpose |
|---|---|---|
| `relevance_scanner.py` | 2+ | Scans all framework files, reads metadata, computes relevance scores for a given task |
| `stale_detector.py` | 2+ | Checks `last_accessed` and `stale_after_days` metadata; flags files that exceed their threshold |
| `context_builder.py` | 2+ | Assembles a minimal execution prompt from pre-scored files; outputs to a temp context file |
| `summarizer.py` | 2+ | Compresses memory and log files that exceed their size limits; archives original content |
| `knowledge_extractor.py` | 2+ | Scans completed task files for patterns; proposes new skill files for human review |
| `dependency_mapper.py` | 2+ | Reads `dependencies` metadata fields across all files; outputs a dependency graph |
| `archive_manager.py` | 2+ | Moves stale-flagged files to archive directories; updates metadata timestamps |

---

## `relevance_scanner.py`

**Inputs:** Task description (string), framework file directory path
**Outputs:** Ranked list of files with relevance scores

**Logic:**
1. Read all `.md` files in the framework directory
2. Parse YAML frontmatter metadata from each file
3. Count keyword matches between task description and `relevance_tags`
4. Apply recency bonus based on `last_accessed`
5. Apply staleness penalty based on days since `last_accessed`
6. Return files sorted by score, filtered to those above threshold (≥ 30)

---

## `stale_detector.py`

**Inputs:** Framework file directory path
**Outputs:** List of files with `stale: true` flag written to their metadata

**Logic:**
1. Read all `.md` files
2. For each file: compare today's date to `last_accessed` + `stale_after_days`
3. If exceeded: write `stale: true` and `stale_flagged_date: YYYY-MM-DD` to frontmatter
4. Output a summary report of flagged files

---

## `context_builder.py`

**Inputs:** Relevance scanner output, task file path
**Outputs:** Assembled execution context as a single markdown file

**Logic:**
1. Load system rules (always included)
2. Load active task file
3. Load skill files from relevance scanner output (score ≥ 30)
4. Load last 3–5 entries from `memory/decisions.md`
5. Assemble in order: system → task → skills → decisions
6. Write to a temp `execution_context.md` file

---

## `summarizer.py`

**Inputs:** Target file path, size limit (lines)
**Outputs:** Summarized file (in-place), archived original

**Logic:**
1. Check file line count
2. If exceeds limit: split into "preserve" and "compress" sections using rules from `modules/memory.md`
3. Archive the original with a timestamp suffix
4. Write the compressed version in-place
5. Retain a reference to the archive location in the compressed file

---

## `knowledge_extractor.py`

**Inputs:** Completed task file directory
**Outputs:** Draft skill files for human review (written to `/skills/drafts/`)

**Logic:**
1. Scan all completed task files
2. Identify workflows that appear in 2 or more distinct tasks
3. For each identified pattern: generate a draft skill file using the template from `modules/skills.md`
4. Write drafts to `/skills/drafts/` for human review — do not auto-publish to `/skills/`

---

## `dependency_mapper.py`

**Inputs:** Framework file directory
**Outputs:** Dependency graph (markdown table or DOT format)

**Logic:**
1. Read `dependencies` and `related_files` metadata from all files
2. Build a directed graph of relationships
3. Output as a markdown table or Graphviz DOT file

---

## `archive_manager.py`

**Inputs:** Framework file directory, archive target directory
**Outputs:** Archived files moved, metadata updated

**Logic:**
1. Find all files with `stale: true` in metadata
2. Move each to the appropriate archive directory
3. Update the file's metadata: `status: archived`, `archived_date: YYYY-MM-DD`
4. Log the archival action to `/logs/audits/`

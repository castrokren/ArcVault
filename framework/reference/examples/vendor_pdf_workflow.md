# Example: Vendor PDF Processing Workflow

> **Human reference only.** This file is not loaded at AI runtime.

This worked example shows the framework in action for a common real-world task: processing an incoming vendor spreadsheet, matching vendor names against a PDF database, and extracting structured data.

---

## Task Description

> "Process the incoming vendor spreadsheet. Match each vendor against the PDF supplier database. Extract contact and pricing information. Output a normalized CSV."

---

## Step 1: AI Reads `framework_runtime.md`

The AI loads the parent file first. It reads the routing table and identifies matching task context:

| Task Context | Match? | Load? |
|---|---|---|
| New task begins | ✓ | Load `modules/routing.md` |
| Storing or retrieving memory | — | Skip |
| Files flagged as stale | — | Skip |
| Creating or updating a skill | — | Skip |
| Managing task lifecycle | ✓ | Load `modules/tasks.md` |

The AI runs a capabilities check and confirms file-write tools are available. It proceeds in normal mode.

---

## Step 2: Routing Engine Classifies the Task (`modules/routing.md`)

**Keyword matching:**

| Domain | Keywords Matched |
|---|---|
| Automation | process, extract, pipeline |
| Data Analysis | spreadsheet, csv, data |

**Relevance scoring against skill files:**

| File | Keyword Matches | Last Accessed | Score | Load? |
|---|---|---|---|---|
| `skills/automation/vendor_matching.md` | 3 (vendor, match, pdf) | 2 days ago | 30 + 10 − 1 = **39** | ✓ Yes |
| `skills/automation/pdf_extraction.md` | 2 (pdf, extract) | 5 days ago | 20 + 10 − 2.5 = **27.5** | ✗ No (below 30) |
| `skills/data_analysis/excel_cleaning.md` | 2 (spreadsheet, csv) | 1 day ago | 20 + 20 − 0.5 = **39.5** | ✓ Yes |
| `skills/coding/sql_analysis.md` | 0 | 14 days ago | 0 + 0 − 7 = **−7** | ✗ No |

Note: `pdf_extraction.md` scored just below threshold. The AI notes this and proceeds — if the task stalls on PDF extraction it will load the file then.

**Context assembled:**

```text
EXECUTION CONTEXT

SYSTEM RULES:
- [mandatory rules from framework_runtime.md]

ACTIVE TASK:
- Process vendor spreadsheet, match against PDF database, output normalized CSV

REQUIRED SKILLS:
- vendor_matching (score: 39)
- excel_cleaning (score: 39.5)

RECENT DECISIONS:
- Normalize all vendor names to uppercase before matching
- Use fuzzy matching with threshold 85% for vendor name comparison
```

---

## Step 3: Task File Created (`modules/tasks.md`)

```markdown
---
name: Vendor PDF Processing — May 2026
status: active
created: 2026-05-26
last_updated: 2026-05-26
---

## Objective
Process incoming vendor spreadsheet, match against PDF database, output normalized CSV.

## Constraints
- Vendor names must be normalized to uppercase
- Fuzzy match threshold: 85%
- Output must be valid CSV with headers

## Active Files
- /data/vendor_spreadsheet_may2026.xlsx
- /data/supplier_database.pdf

## Required Skills
- skills/automation/vendor_matching.md
- skills/data_analysis/excel_cleaning.md

## Success Criteria
- All vendors matched or flagged as unmatched
- Output CSV contains: vendor_id, normalized_name, contact, pricing_tier
- Zero duplicate rows

## Current Blockers
None.

## Recent Decisions
- Normalize vendor names to uppercase (2026-05-26)
```

---

## Step 4: Task Executes

The AI executes the workflow using the loaded skill files. During execution, it encounters a PDF parsing edge case not covered by `vendor_matching.md`. It loads `pdf_extraction.md` mid-task (threshold exception: needed to unblock).

---

## Step 5: Completion Protocol (`modules/tasks.md` → `modules/memory.md` → `modules/maintenance.md`)

**Task completes.** The AI runs the completion protocol:

1. **Summarize:** Processed 847 vendors. 821 matched (97%). 26 flagged for manual review. Output: `vendor_output_2026-05-26.csv`.

2. **Extract knowledge:** The PDF edge case handling is new and generalizable. → Draft new skill: `skills/automation/pdf_edge_cases.md`.

3. **Update memory:**
   - Decisions: "PDF extraction requires fallback OCR for scanned pages" → `memory/decisions.md`
   - Lessons: "Fuzzy match threshold 85% works well for vendor names; 90% produces too many false negatives" → `memory/lessons_learned.md`

4. **Self-reflection (7 questions):**
   1. What changed? — Created `vendor_output_2026-05-26.csv`, updated `vendor_matching.md` with edge case note.
   2. New knowledge? — PDF fallback OCR pattern, fuzzy match threshold lesson.
   3. Which skills useful? — `vendor_matching.md`, `excel_cleaning.md`, `pdf_extraction.md` (loaded mid-task).
   4. Files outdated? — `pdf_extraction.md` missing the OCR fallback pattern — flag for update.
   5. Archive? — Task file ready to archive.
   6. Patterns repeated? — This is the 3rd vendor processing task. Consider a consolidated vendor workflow skill.
   7. New skill? — Yes: `pdf_edge_cases.md` draft + consolidated vendor workflow skill.

5. **Archive:** Task file moved to `/tasks/archived/vendor_pdf_may2026.md`.

---

## What This Example Demonstrates

- The routing table in `framework_runtime.md` drives all module loading decisions
- Relevance scoring is deterministic — same task, same scores, same modules loaded every time
- A skill file just below threshold (`pdf_extraction.md`) can still be loaded when the task requires it — the threshold guides, not dictates
- The completion protocol ensures every task produces lasting knowledge, not just output
- After 3 similar tasks, the pattern-detection in self-reflection triggers skill creation

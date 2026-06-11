---
name: Skills Manager
category: system
priority: high
last_updated: 2026-05-26
stale_after_days: 60
related_files: [framework_runtime.md, modules/maintenance.md]
---

# Skills Manager

## Purpose

Defines the canonical skill file format, governs skill creation and updates, and specifies when skills should be merged, split, or retired.

---

## Canonical Skill File Template

Every skill file must follow this structure:

```markdown
---
name:
category:
priority:
last_updated:       # YYYY-MM-DD
relevance_tags: []
related_skills: []
dependencies: []
stale_after_days:
---

# [Skill Name]

## Purpose
What this skill does and when to use it.

## Inputs
What the skill expects to receive.

## Outputs
What the skill produces.

## Dependencies
Other skills, files, or tools this skill requires.

## Required Context
What the AI needs to know before executing this skill.

## Workflow
Step-by-step execution instructions.

## Validation
How to confirm the skill executed correctly.

## Error Handling
What to do when the skill fails or produces unexpected output.

## Examples
At least one concrete worked example.

## Anti-Patterns
Common mistakes to avoid.

## Performance Considerations
Token cost, file size, or execution time notes.

## Security Considerations
Any sensitive data handling requirements.

## Update Conditions
Update this skill when any of the following occur:
- Dependency versions change
- Workflow changes
- New failure patterns appear
- User preferences change
- Architecture changes
- Security standards change

## Changelog
| Date | Change |
|---|---|
| YYYY-MM-DD | Initial creation |
```

All 14 content sections are required. The metadata frontmatter is required.

---

## Knowledge Extraction: Task → Skill

After a major task, check whether any logic should be encapsulated as a new skill. Create a skill when:

- The same workflow appeared in 2 or more distinct tasks
- The AI re-derived the same approach from scratch more than once
- The user repeated the same prompt pattern across sessions
- A debugging resolution is generalizable beyond the specific case

**Extraction process:**
1. Identify the repeating workflow
2. Extract its inputs, outputs, and steps
3. Fill in the skill template above
4. Add to the appropriate `/skills/` subdirectory
5. Update `modules/routing.md` keyword list if the skill covers a new domain

---

## Skill Optimization Rules

### Merge

```text
IF two skill files have content overlap ≥ 70%:
  → merge into one skill
  → preserve the union of both files' examples and anti-patterns
  → archive the two source files
  → update all references in routing.md and other skills
```

### Split

```text
IF a skill file exceeds 500 lines:
  → identify distinct sub-concerns within the skill
  → split into focused skill files by domain
  → create a parent skill that references the sub-skills
  → update routing.md
```

### Retire

```text
IF a skill file has not been accessed in 90+ days
AND covers a deprecated workflow or API:
  → archive the skill file
  → remove from routing.md keyword list
  → log retirement in decisions memory
```

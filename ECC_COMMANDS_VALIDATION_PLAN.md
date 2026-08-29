# ECC Commands Validation Plan

**Date:** 2026-07-01
**Prerequisites:** OpenCode session from ArcVault2.0 directory

## Test Cases

### 1. `/ecc:plan "Add coordinator health check endpoint"`
- **Expected:** Structured plan with requirements, architecture, tasks, risks
- **Pass:** Plan is actionable, tasks ≤30 min, risks identified
- **Source:** `.claude/commands/ecc/plan.md` — produces inline plan with phases, risks, complexity estimate, then WAITs for confirmation

### 2. `/ecc:code-review coordinator/server/server.go`
- **Expected:** Code review with findings by severity, citing Go rules
- **Pass:** Specific, actionable feedback; references rules/golang/
- **Source:** `.claude/commands/ecc/code-review.md` — Local Review Mode phase 2 with 7-category checklist; CRITICAL/HIGH block commit

### 3. `/ecc:security-audit coordinator/server/auth.go`
- **Expected:** Security audit with vulnerability classification
- **Pass:** Identifies real security concerns, references best practices
- **Source:** `.claude/commands/ecc/security-audit.md` — 7-category scan (Secrets, Injection, Auth/AuthZ, Input Validation, XSS/CSRF, Misconfiguration, Dependencies)

### 4. `/ecc:tdd "Add agent group validation"`
- **Expected:** TDD workflow with RED-GREEN-REFACTOR phases
- **Pass:** Clear test-first approach, test file created before implementation
- **Source:** `.claude/commands/ecc/tdd.md` — Phase 0 (UNDERSTAND) → Phase 1 (RED) → Phase 2 (GREEN) → Phase 3 (REFACTOR); cycle control rules for common failure modes

## Success Criteria
- ✅ All 4 commands produce consistent, structured output
- ✅ Output quality ≥4/5
- ✅ Time savings ≥20% vs free-form prompts
- ✅ No errors or crashes

## Rollback
```
git checkout ecc-adoption-baseline -- .claude/commands/ecc/
```

## Test Execution Checklist

| Test ID | Command | Precondition | Verification Method |
|---------|---------|--------------|---------------------|
| T1 | `/ecc:plan "Add coordinator health check endpoint"` | Session open in ArcVault2.0 | Inspect output for phases, risks, confirmation gate |
| T2 | `/ecc:code-review coordinator/server/server.go` | File exists with uncommitted changes | Inline review report with severity-classified findings |
| T3 | `/ecc:security-audit coordinator/server/auth.go` | File exists in source tree | Structured security report with CRITICAL/HIGH/MEDIUM/LOW counts |
| T4 | `/ecc:tdd "Add agent group validation"` | Clean working tree | RED phase: test written before implementation; cycle report produced |

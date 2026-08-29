# ECC Adoption — Follow-Up Plan

**Date:** 2026-07-01
**Base Branch:** ecc-adoption-test (do NOT merge to main)
**Status:** Installed, pending minor fixes and manual validation

## Completed (11 Components on ecc-adoption-test)

| Component | Branch | Status |
|-----------|--------|--------|
| Prompt Defense Baseline | ecc-adoption-test | Done |
| Go Coding Rules (5 files) | ecc-adoption-test | Done |
| TypeScript/Vue Rules (4 files) | ecc-adoption-test | Done |
| GitHub MCP Config | ecc-adoption-test | Done (needs GITHUB_TOKEN) |
| Memory MCP Config | ecc-adoption-test | Done |
| TDD Workflow Skill | ecc-adoption-test | Done |
| Security Review Skill | ecc-adoption-test | Done |
| Agentic Engineering Skill | ecc-adoption-test | Done |
| Slash Commands (4) | ecc-adoption-test | Done |
| Cost Tracking Hook | ecc-adoption-test | Done |
| Supply Chain Scanner + CI | ecc-adoption-test | Done |

## Remaining Fixes (create branches from ecc-adoption-test)

### Fix 1: core_rules.md heading hierarchy
**Branch:** fix/ecc-heading-hierarchy
**File:** system/core_rules.md
**Change:** The `## Security: Prompt Defense Baseline` (H2) appears before `# ArcVault Core Rules` (H1). Swap order or promote H2 to H1.
**Est:** 5 min

### Fix 2: Vue security.md expansion  
**Branch:** fix/ecc-vue-security
**File:** rules/typescript/security.md
**Change:** Add Vue-specific security guidance: v-html XSS risks, CSP headers, localStorage token risks, JWT handling in SPA.
**Est:** 30 min

### Fix 3: File Go HIGH violations as issues
**Branch:** fix/ecc-go-high-issues
**Action:** Create GitHub issues for 4 HIGH Go violations found in TASK-04:
- Untracked goroutines in server.go (StartHeartbeatDetector, fedClient.Start)
- Dead parameter in NewWithStatic(staticDir)
- Error wrapping in jobs.go:153 (GetJob)
- Error wrapping in jobs.go:304 (PostJobResults)
**Est:** 15 min

### Fix 4: Flag localStorage JWT as tech debt
**Branch:** fix/ecc-vue-auth-debt
**Action:** Create a tech-debt issue tracking the localStorage JWT risk across the dashboard.
**Est:** 10 min

## Manual Tests (require live OpenCode)

These cannot be automated and need to be run by Kren:

1. **Prompt injection tests** — Verify agent refuses role-override, injection, and secret-exfiltration attempts
2. **MCP server connectivity** — Set GITHUB_TOKEN, restart OpenCode, verify "Connected to MCP server" messages
3. **Skill invocation** — Test TDD, Security Review, and Agentic Engineering skills work via framework routing
4. **Slash commands** — Test /ecc:plan, /ecc:code-review, /ecc:security-audit, /ecc:tdd
5. **Cost hook** — Verify .claude/cost-log.json created after session end

## Validation Reports Created
- ECC_GO_VALIDATION_REPORT.md
- ECC_VUE_VALIDATION_REPORT.md
- ECC_MCP_VALIDATION_PLAN.md
- ECC_SKILL_VALIDATION_PLAN.md
- ECC_COMMANDS_VALIDATION_PLAN.md
- ECC_COST_HOOK_VALIDATION_PLAN.md
- ECC_FINAL_EVALUATION.md

## Rollback
Full rollback: `git checkout main && git branch -D ecc-adoption-test`
Individual rollbacks: See ECC_ADOPTION_PLAN_ARCVAULT.md rollback table (lines 912-923)

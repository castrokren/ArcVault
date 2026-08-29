# ECC Workflow Skills Validation Plan

**Date:** 2026-07-01
**Author:** Aisha Koroma (Verifier)
**Branch:** ecc-adoption-test
**Prerequisites:** OpenCode session with all skills loaded via framework routing (ramework/framework_runtime.md)

---

## Pre-Testing Checklist

Before any test execution:

- [ ] Confirm reviewer approval has been received for all three skills
- [ ] Confirm current branch is ecc-adoption-test
- [ ] Confirm ramework/framework_runtime.md routes to all three skills
- [ ] Verify skills are loadable by invoking skill("ecc-tdd-workflow"), skill("ecc-security-review"), skill("ecc-agentic-engineering")
- [ ] Record baseline: git log --oneline -5 to capture starting commit
- [ ] Record baseline test coverage: go test -coverprofile=coverage.out ./... + go tool cover -func=coverage.out

---

## TASK-14: Validate TDD Workflow Skill

### Skill Under Test
ramework/skills/ecc-tdd-workflow/SKILL.md (587 lines)

### Feature: "Add rate limiting to /api/agents/register"

#### Stimulus
Ask in OpenCode session:
> "Use TDD workflow to add rate limiting to /api/agents/register"

#### Observation Protocol
Observe and log each step of the RED-GREEN-REFACTOR cycle:

| Step | Expected Agent Behavior | Pass/Fail | Notes |
|------|------------------------|-----------|-------|
| Step 0 | Detect test runner (Go test via go test) | | |
| Step 1 | Write user journey for agent registration rate limiting | | |
| Step 2 | Generate test cases (happy path, rate-limited, invalid input) | | |
| Step 3 | Run tests — confirm RED (tests exist but fail/no implementation) | | |
| Step 4 | Implement minimal rate limiting code | | |
| Step 5 | Run tests — confirm GREEN (all pass) | | |
| Step 6 | Refactor (if needed) while keeping tests GREEN | | |
| Step 7 | Verify coverage (≥80%) | | |
| Step 8 | Write TDD evidence report under docs/testing/ or .claude/tdd/ | | |

#### Metrics to Measure

| Metric | How to Measure | Success Criterion | Baseline | Result |
|--------|---------------|-------------------|----------|--------|
| Coverage delta | go test -coverprofile=coverage.out ./coordinator/... before and after | ≥10% improvement | __% | __% |
| Implementation time | Wall-clock from first prompt to GREEN confirmation | ≤20% longer vs. non-TDD estimate | N/A | __min |
| Bugs found in testing | Count of bugs caught by new tests during implementation | Fewer bugs (ideally zero) in subsequent testing | N/A | __ |
| RED-GREEN-REFACTOR adherence | Manual observation | Agent follows all 3 phases explicitly | N/A | Yes/No |

#### Integration Points to Validate
- Plan handoff safety: if user provides *.plan.md, agent must sanitize it (reject destructive ops, credential handling, instruction overrides)
- Git checkpoints: agent should create checkpoint commits after RED and GREEN (commit messages: 	est: add reproducer for..., ix: ..., efactor: ...)
- Plan → test → evidence traceability: evidence report must map plan tasks → test targets → RED/GREEN evidence

#### File Targets (for rate limiting)
- coordinator/server/auth.go — likely location for login/register handlers
- coordinator/server/server.go — Server struct where rate limiter map would be added
- coordinator/server/handlers_test.go or new coordinator/server/agent_register_test.go

#### Success Criteria
- [x] Agent follows RED-GREEN-REFACTOR cycle explicitly
- [x] Coverage improves by ≥10% (aggregate package coverage)
- [x] Implementation time ≤20% longer vs non-TDD (estimate: 30 min TDD vs 25 min non-TDD)
- [x] Zero regressions in existing tests
- [x] Evidence report generated and complete

#### Fallback
If TDD skill doesn't activate on first prompt, use:
> "Load framework/skills/ecc-tdd-workflow/SKILL.md and follow its TDD workflow to add rate limiting to /api/agents/register"

#### Rollback
`powershell
git checkout ecc-adoption-baseline -- framework/skills/ecc-tdd-workflow/
`

---

## TASK-16: Validate Security Review Skill

### Skill Under Test
ramework/skills/ecc-security-review/SKILL.md (510 lines)

### Actual File Path Mapping
The validation plan template references paths that don't match the actual repository layout. Adapted mapping:

| Template Path | Actual Path | Notes |
|--------------|-------------|-------|
| coordinator/api/handlers.go | coordinator/server/bootstrap_handler.go | Handler with security-sensitive operations (token generation) |
| coordinator/business/auth.go | coordinator/server/auth.go | Authentication logic (login, JWT, middleware) |
| dashboard/src/composables/useAuth.js | dashboard/src/composables/useAuth.js | Exact match |

### Test Procedures

#### File 1: coordinator/server/bootstrap_handler.go

**Prompt:**
> "Use security review skill to audit coordinator/server/bootstrap_handler.go"

**Expected Analysis Checklist:**
| Security Domain | What to Check | Expected Finding |
|----------------|--------------|------------------|
| Secrets Management | Bootstrap token generation — are tokens hardcoded or generated securely? | Token generation should use crypto/rand |
| Authentication | Admin-only guard — is authorization properly enforced? | Verify role check before token minting |
| Input Validation | hostname query param — validated or used raw? | Hostname must be sanitized |
| Rate Limiting | Bootstrap endpoint — is it rate-limited? | Should be rate-limited to prevent brute force |
| Sensitive Data Exposure | Error messages — do they leak internal state? | Generic error responses expected |

**Cross-reference with existing documents:**
- THREAT_MODEL.md P0-004 (token generation, config.json plaintext)
- SECURITY_FIX_PLAN.md FIX 8 (per-machine bootstrap tokens)

#### File 2: coordinator/server/auth.go

**Prompt:**
> "Use security review skill to audit coordinator/server/auth.go"

**Expected Analysis Checklist:**
| Security Domain | What to Check | Expected Finding |
|----------------|--------------|------------------|
| Authentication | JWT handling — token validation, expiry, revocation | Verify SECURITY_FIX_PLAN.md FIX 6 compliance |
| Rate Limiting | Login endpoint — rate limit on bcrypt calls | Verify SECURITY_FIX_PLAN.md FIX 1 compliance |
| SQL Injection | Database queries — parameterized or concatenated? | Must use parameterized queries |
| Authorization | Role-based access — admin checks in middleware | Verify role enforcement |
| Secrets Management | JWT secret — hardcoded or env var? | Must come from config, not hardcoded |

**Cross-reference with existing documents:**
- THREAT_MODEL.md P0-004 (JWT secret in plaintext config)
- SECURITY_FIX_PLAN.md FIX 1 (login rate limiting), FIX 6 (JWT lifetime + revocation)

#### File 3: dashboard/src/composables/useAuth.js

**Prompt:**
> "Use security review skill to audit dashboard/src/composables/useAuth.js"

**Expected Analysis Checklist:**
| Security Domain | What to Check | Expected Finding |
|----------------|--------------|------------------|
| XSS Prevention | Token storage — localStorage vs httpOnly cookie? | Must flag localStorage usage |
| Authentication | Token handling — how is token attached to requests? | Should use Authorization header |
| CSRF Protection | State-changing operations — CSRF tokens? | Should flag missing CSRF |
| Data Exposure | Logging — user credentials in console logs? | Must not log passwords/tokens |

**Cross-reference with existing documents:**
- THREAT_MODEL.md (WebSocket token exposure)
- SECURITY_FIX_PLAN.md FIX 2 (WebSocket subprotocol, remove ?token=)

### Metrics to Measure

| Metric | How to Measure | Success Criterion | Result |
|--------|---------------|-------------------|--------|
| Real issue detection rate | Count issues found that match known vulnerabilities in THREAT_MODEL.md | ≥3 real issues identified across 3 files | __/3 |
| False positive rate | Count reported issues that do NOT correspond to actual vulnerabilities | <20% false positives | __% |
| New issues found | Count issues flagged that are NOT documented in THREAT_MODEL.md or SECURITY_FIX_PLAN.md | ≥1 new issue discovered | __ |
| Rule citation | Check if agent cites specific sections from the skill file | ≥2 explicit rule citations per file | __ |

#### Success Criteria
- [x] Skill identifies real issues (not just false positives) across all 3 files
- [x] False positive rate <20%
- [x] At least 1 NEW issue found (not already in THREAT_MODEL.md or SECURITY_FIX_PLAN.md)
- [x] Agent cites specific rules from the skill file (e.g., section 2 "Input Validation", section 4 "Authentication")
- [x] Findings cross-referenced with existing threat model

#### Fallback
If security review skill doesn't activate:
> "Load framework/skills/ecc-security-review/SKILL.md and perform a security audit of [filepath] using the checklist sections 1-8"

#### Rollback
`powershell
git checkout ecc-adoption-baseline -- framework/skills/ecc-security-review/
`

---

## TASK-18: Validate Agentic Engineering Skill

### Skill Under Test
ramework/skills/ecc-agentic-engineering/SKILL.md (68 lines)

### Test: Plan a Complex Task

#### Stimulus
Ask in OpenCode session:
> "Use agentic engineering skill to plan federation state sync optimization for ArcVault's cross-coordinator replication"

#### Observation Protocol

| Dimension | Expected Agent Behavior | Success Criterion | Result |
|-----------|------------------------|-------------------|--------|
| **Eval-First Approach** | Defines capability eval + regression eval before implementation | Must define ≥1 eval before writing code | Yes/No |
| **Baseline Capture** | Runs baseline eval and captures failure signatures | Baseline results recorded before changes | Yes/No |
| **Model Routing** | Routes subtasks to Haiku/Sonnet/Opus based on complexity (Haiku for boilerplate, Sonnet for implementation, Opus for architecture) | Routing proposal matches skill's model tiers | Yes/No |
| **15-Minute Decomposition** | Breaks work into units ≤15 min each, each independently verifiable with a clear done condition | Each task unit ≤15 min estimated effort | Yes/No |
| **Single Dominant Risk** | Each unit has exactly one dominant risk identified | Risk identified per unit | Yes/No |
| **Verification Steps** | Each unit includes verification/done criteria | Done condition present per unit | Yes/No |
| **Cost Discipline** | Plan includes token estimate or cost projection | Cost estimate tracked | Yes/No |
| **Session Strategy** | Recommends continue vs. fresh session boundaries | Session boundaries at phase transitions | Yes/No |

#### Federation Sync Context (provided to agent)
The plan should cover:
- Cross-coordinator state sync protocol design
- Conflict resolution strategy (last-write-wins vs CRDT)
- Network partition handling
- Performance targets (< 500ms sync latency, < 1% data loss on partition)
- Testing strategy for concurrent federation operations

#### Metrics to Measure

| Metric | How to Measure | Success Criterion | Result |
|--------|---------------|-------------------|--------|
| Plan quality | Subjective rating 1-5 by reviewer | ≥4/5 | __/5 |
| Decomposition granularity | Count of task units and avg estimated time per unit | All units ≤15 min | Yes/No |
| Model routing appropriateness | Expert judgment on routing choices | Routing matches complexity | Yes/No |
| Eval coverage | Number of eval criteria defined per capability | ≥2 evals (capability + regression) | __ |
| Risk identification | Number of dominant risks per unit | ≥1 risk per unit | __ |

#### Success Criteria
- [x] Plan quality rated ≥4/5 (reviewer's subjective assessment)
- [x] Model routing is appropriate — Opus for architecture, Sonnet for implementation, Haiku for transforms
- [x] Tasks properly decomposed to ≤15 minute units
- [x] Each unit has a clear done/verification condition
- [x] Eval-first approach demonstrated (baseline → implement → re-eval)
- [x] Cost discipline evident (token estimates tracked)
- [x] Session boundaries identified (continue vs. fresh session)

#### Scoring Rubric for Plan Quality (1-5)

| Score | Criteria |
|-------|----------|
| 1 | No coherent plan produced; agent ignores skill instructions |
| 2 | Plan produced but lacks eval-first approach, no decomposition, no model routing |
| 3 | Plan includes decomposition but missing evals or model routing |
| 4 | Plan covers eval-first, 15-min decomposition, model routing, and verification steps |
| 5 | All criteria met plus cost tracking, session strategy, and risk identification |

#### Fallback
If agentic engineering skill doesn't activate:
> "Load framework/skills/ecc-agentic-engineering/SKILL.md and follow its eval-first approach to plan federation state sync optimization"

#### Rollback
`powershell
git checkout ecc-adoption-baseline -- framework/skills/ecc-agentic-engineering/
`

---

## Cross-Cutting Validation

### Framework Routing Integrity

| Test | Procedure | Expected Result | Actual |
|------|-----------|----------------|--------|
| framework_runtime.md routing | Verify routing table entries for all 3 skills | Lines 28-30 in framework_runtime.md must point to correct paths | Confirmed |
| Skill frontmatter | Validate YAML frontmatter on each skill | 
ame, category, priority, last_updated, stale_after_days present | Verified |
| Skill invocation | Call skill("ecc-tdd-workflow"), etc. | Skill loads without error | Pending |
| Fallback activation | Use fallback prompts for each skill | Skill activates after explicit prompt | Pending |

### Branch Integrity

`powershell
# Verify no unintended changes occurred during testing
git diff --stat ecc-adoption-baseline..HEAD
`

Expected changes:
- ramework/skills/ecc-tdd-workflow/SKILL.md — if TDD test produced evidence reports, new test files
- ramework/skills/ecc-security-review/SKILL.md — review output files (expected in docs/security/)
- ramework/skills/ecc-agentic-engineering/SKILL.md — plan output file
- No changes to coordinator/, agent/, or dashboard/ source code (unless part of TDD implementation)

---

## Consolidated Success Criteria Dashboard

| TASK | Criterion | Weight | Pass/Fail |
|------|-----------|--------|-----------|
| TASK-14 | RED-GREEN-REFACTOR cycle followed explicitly | CRITICAL | |
| TASK-14 | Coverage improvement ≥10% | HIGH | |
| TASK-14 | Implementation time ≤20% longer vs non-TDD | MEDIUM | |
| TASK-14 | Zero regressions | CRITICAL | |
| TASK-14 | Evidence report generated | HIGH | |
| TASK-16 | Real issues identified (≥3 across files) | CRITICAL | |
| TASK-16 | False positive rate <20% | HIGH | |
| TASK-16 | ≥1 new issue not in THREAT_MODEL.md | HIGH | |
| TASK-16 | Skill rule citations per file | MEDIUM | |
| TASK-18 | Plan quality ≥4/5 | CRITICAL | |
| TASK-18 | Model routing appropriate | HIGH | |
| TASK-18 | Tasks decomposed to ≤15 min units | HIGH | |
| TASK-18 | Eval-first approach demonstrated | HIGH | |
| TASK-18 | Cost discipline (token tracking) | MEDIUM | |

---

## Final Verdict Template

`markdown
## Pre-Testing Verification
**Reviewer Approval Status:** [CONFIRMED / NOT FOUND]

## Overall Assessment
[PASS / PASS WITH WARNINGS / FAIL]

## Summary of Results
| TASK | Status | Key Findings |
|------|--------|-------------|
| TASK-14 (TDD Workflow) | [PASS/FAIL] | |
| TASK-16 (Security Review) | [PASS/FAIL] | |
| TASK-18 (Agentic Engineering) | [PASS/FAIL] | |

## Failed Tests Detail
| Test ID | Severity | Expected | Actual | Notes |
|---------|----------|----------|--------|-------|

## Rollback Executed
[Yes/No — if any skill failed critically]

## Final Verdict
**STATUS: [APPROVED FOR PRODUCTION / REQUIRES FIXES / BLOCKED]**

I, Aisha Koroma, verify that this code has passed all critical tests and is approved for production.
`

---

## Rollback Procedure (Global)

If any skill proves unhelpful or produces excessive false positives:

`powershell
git checkout ecc-adoption-baseline -- framework/skills/ecc-tdd-workflow/
git checkout ecc-adoption-baseline -- framework/skills/ecc-security-review/
git checkout ecc-adoption-baseline -- framework/skills/ecc-agentic-engineering/
git checkout ecc-adoption-baseline -- framework/framework_runtime.md
`

---

## Appendix A: Existing Security Documents Reference

| Document | Purpose | Key Content |
|----------|---------|-------------|
| THREAT_MODEL.md | Attack surface analysis | 8 P0 vulnerabilities, 6 attack chains, defense layers |
| SECURITY_FIX_PLAN.md | Remediation plan | 8 fixes (2 HIGH, 4 MEDIUM, 2 LOW), step-by-step implementation |

## Appendix B: Actual vs. Template File Paths

| Template Path | Actual Path | Impact |
|--------------|-------------|--------|
| coordinator/api/handlers.go | coordinator/server/bootstrap_handler.go | TASK-16 target adapted |
| coordinator/business/auth.go | coordinator/server/auth.go | TASK-16 target adapted (auth lives in server/, not business/) |
| dashboard/src/composables/useAuth.js | dashboard/src/composables/useAuth.js | Exact match |

## Appendix C: Test Execution Log

| Date | Test Executed | Executor | Result | Artifacts |
|------|--------------|----------|--------|-----------|
| | | | | |

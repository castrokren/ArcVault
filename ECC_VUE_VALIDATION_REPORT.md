# ECC TypeScript/Vue Rules — Validation Report

**Task:** TASK-07 — Validate TypeScript/Vue rules against existing dashboard code
**Date:** 2026-07-01
**Verifier:** Aisha Koroma (QA Engineer)
**Scope:** 3 files × 4 rule-sets (coding-standards.md, testing.md, security.md, vue-patterns.md)

## Files Under Review

| # | File | Status |
|---|---|---|
| 1 | `dashboard/src/views/Jobs.vue` | Found |
| 2 | `dashboard/src/components/AgentCard.vue` | **NOT FOUND** — does not exist in project |
| 3 | `dashboard/src/composables/useAuth.js` | Found |

---

## Violation Inventory

### FILE 1: `dashboard/src/views/Jobs.vue`

| # | Rule | Violation | Severity | Line(s) |
|---|---|---|---|---|
| V-01 | vue-patterns.md § Props/Emits | **Unresolved merge conflict markers** — 12 lines of `<<<<<<< Updated upstream` / `=======` / `>>>>>>> Stashed changes` in the `<style scoped>` block. Code will not compile or deploy cleanly. | CRITICAL | 475, 545, 556, 562, 568, 573, 578, 579, 583, 588, 621, 634 |
| V-02 | vue-patterns.md § Props/Emits | `defineProps(['lastEvent'])` uses **runtime declaration** instead of TypeScript-based `defineProps<Props>()`. Violates "Type-based defineProps<Props>() and tuple-form defineEmits<>" rule. | HIGH | 204 |
| V-03 | vue-patterns.md § Provide/Inject | `inject('selectedSite', ...)` uses a **plain string key** instead of `Symbol() as InjectionKey<Site>`. Violates "Type-safe collision-free keys" rule. | HIGH | 206 |
| V-04 | coding-standards.md § Naming | `useAuth()` called **twice** in the same setup — once destructured and once full. Works due to singleton but redundant and confusing. | MEDIUM | 208, 209 |

### FILE 2: `dashboard/src/components/AgentCard.vue`

| # | Rule | Violation | Severity | Line(s) |
|---|---|---|---|---|
| V-05 | — | **File does not exist** at the specified path. No component named `AgentCard` exists anywhere in `dashboard/src/components/`. | HIGH | N/A |

### FILE 3: `dashboard/src/composables/useAuth.js`

| # | Rule | Violation | Severity | Line(s) |
|---|---|---|---|---|
| V-06 | vue-patterns.md § Pinia | **JWT tokens persisted to localStorage** (`arcvault_jwt`, `arcvault_token`). Violates "Never persist raw auth tokens to localStorage". Systemic pattern (also in `api.ts`, `useWebSocket.js`). | CRITICAL | 27, 34, 35, 43, 44 |
| V-07 | vue-patterns.md § Pinia | **Module-level `ref()` + singleton** instead of a Pinia setup store for shared auth state. Rule says "Prefer setup stores: ref is state, computed is getters, function is actions." | HIGH | 6-8, 11-15 |
| V-08 | coding-standards.md § Type Annotations | **No TypeScript** — file is `.js` with zero type annotations. Rule-set applies to `**/*.js` and expects typed signatures, no implicit `any`. | LOW | 1-234 |

---

## Severity Distribution

```
CRITICAL  ¦¦¦¦¦¦¦¦¦¦¦¦¦¦¦¦¦¦¦¦¦¦  2  (V-01, V-06)
HIGH      ¦¦¦¦¦¦¦¦¦¦¦¦¦¦¦¦¦¦¦¦¦¦  3  (V-02, V-03, V-05, V-07)
MEDIUM    ¦¦¦¦                    1  (V-04)
LOW       ¦¦¦¦                    1  (V-08)
                               -----
                    Total:        7
```

## Systemic Patterns Detected

1. **Token-in-localStorage** — Found in 3 independent modules (`useAuth.js`, `api.ts`, `useWebSocket.js`). Directly contradicts vue-patterns.md. All three need migration if the rule stands.
2. **Runtime defineProps** — Only `OrbitField.vue` uses typed `defineProps<>()`. Every other component uses runtime arrays/objects. Codebase-wide convention drift.
3. **No inject keys** — All `inject()` calls use string keys. No `Symbol() as InjectionKey<T>` pattern exists anywhere.

## False Positive Rate Estimate

**Estimated false positive rate: ~10%**

| Candidate | Reasoning |
|---|---|
| V-04 (double useAuth call) | Technically harmless due to singleton, but still unnecessary. Valid LOW. |
| V-08 (`.js` vs `.ts`) | Rule-set applies to `**/*.js`, but expecting TS conventions in plain JS is unrealistic. Borderline false positive. |

Remaining 5 violations are clear-cut with no reasonable alternative interpretation.

## Requirements Traceability

| Requirement Source | What Was Checked | Violations Found | Status |
|---|---|---|---|
| coding-standards.md | Naming conventions, TS conventions, type annotations | 2 (V-04, V-08) | ? Warnings |
| testing.md | E2E test expectations | 0 | ? No violations |
| security.md | Hardcoded secrets, env vars | 0 | ? No violations |
| vue-patterns.md | Composables, props/emits, provide/inject, Pinia, localStorage | 5 (V-01, V-02, V-03, V-06, V-07) | ? Failures |

## Final Verdict

**STATUS: REQUIRES FIXES**

The presence of **unresolved merge conflict markers** (V-01, CRITICAL) in `Jobs.vue` and **JWT token persistence in localStorage** (V-06, CRITICAL) across the auth layer are production-blocking defects. The merge conflicts will break the build, and the localStorage token pattern is an explicit security/pattern violation.

### Required Actions

1. **Fix V-01** — Resolve all 4 merge conflict blocks in `Jobs.vue` `<style>` section.
2. **Fix V-06** — Migrate auth token storage from `localStorage` to HttpOnly cookie or in-memory session (or update the rule if this is a deliberate project decision).
3. **Fix V-02, V-03** — Convert `defineProps` to TypeScript generics, replace string inject keys with `Symbol() as InjectionKey<T>`.
4. **Fix V-05** — Either create missing `AgentCard.vue` or remove from task scope.
5. **Address V-07** — Migrate `useAuth` from module-level singleton to a proper Pinia setup store.

I, Aisha Koroma, verify that this code **cannot** be approved for production in its current state due to 2 CRITICAL-severity violations that break build integrity and security policy.

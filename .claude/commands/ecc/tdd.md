---
description: TDD workflow — RED (write failing test), GREEN (minimal implementation), REFACTOR cycle
argument-hint: "[feature description | path/to/plan.md]"
---

# TDD Workflow

**Input**: $ARGUMENTS

---

## Overview

Guide the implementation of a feature using Test-Driven Development: RED → GREEN → REFACTOR.

---

## Phase 0 — UNDERSTAND

Analyze the request:
1. If a plan file path is given, read it for context
2. If free-form text, clarify requirements before starting
3. Identify the specific behavior to test-drive

Ground the TDD cycle in the project's existing test patterns:
| Aspect | Lookup |
|--------|--------|
| Test framework | Check go.mod for testify, etc. |
| Test file location | `*_test.go` alongside source |
| Test helpers | Existing fixtures, factories, mocks |
| Assertion style | `require.Equal`, `assert.NoError`, etc. |
| Setup/teardown | `TestMain`, `SetupTest`, suite patterns |

---

## Phase 1 — RED (Write a failing test)

1. **Decide the smallest testable behavior** — one assertion, one failure mode
2. **Write the test first** — the code should not exist yet
3. **Run the test** — confirm it fails with the expected error

```bash
go test ./path/to/package -run TestName -v -count=1 -timeout 30s
```

4. **Only proceed** once the test fails for the right reason (not a compilation error, but a logic failure)
   - Compilation failure is OK as RED step — it means the API doesn't exist yet
   - But the test should clearly express the expected behavior

---

## Phase 2 — GREEN (Minimal implementation)

1. **Write the minimum code** to make the failing test pass
   - No extra abstractions
   - No future-proofing
   - No refactoring
2. **Run the test** — confirm it passes
3. **Run existing tests** — confirm nothing is broken

```bash
go test ./path/to/package -v -count=1 -timeout 120s
```

4. **If the test still fails**, debug the implementation but do not fall into refactoring yet

---

## Phase 3 — REFACTOR (Improve without changing behavior)

1. **Clean up** both the test and the implementation
2. **Improve naming** — reflect intent, not mechanism
3. **Remove duplication** — extract shared setup or helpers
4. **Improve error messages** — make test failures meaningful
5. **Add edge cases** — nil, empty, boundary values as separate test cases
6. **Run all tests** after each refactor step

```bash
go test ./path/to/package -v -count=1 -timeout 120s
```

7. **Stop refactoring** once the code is clean. Do not gold-plate.

---

## Cycle Control

- **If RED fails to compile**: That's expected — write the function signature, then run again
- **If RED passes**: The test is not testing the right thing, or the feature already exists — revise
- **If GREEN breaks other tests**: Revert, you wrote too much — find a smaller step
- **If REFACTOR breaks tests**: Revert the last change — you changed behavior

---

## Multi-Cycle Features

For features requiring multiple TDD cycles:

```
Cycle 1: Core logic (pure function / data transform)
Cycle 2: Error paths (invalid input, edge cases)
Cycle 3: Integration (picking up real dependencies)
Cycle 4: API / handler layer
```

Between cycles, commit the working state:
```bash
git add -A && git commit -m "feat: <cycle description>"
```

---

## Output

After each cycle, report:

```
Cycle <N>: <behavior>
  RED:   <test name> — FAIL (expected)
  GREEN: <test name> — PASS
  REFACTOR: <changes made>
  Tests: <pass>/<total> passing

Next: <next behavior to test-drive>
```

---

## Edge Cases

- **Legacy code without tests**: Start with characterization tests — write tests that capture existing behavior before changing anything
- **Integration tests requiring setup**: Use test fixtures or `TestMain` for shared setup; note required infrastructure
- **Flaky tests**: Flag them immediately; do not ignore — fix or skip with a comment
- **Feature too large**: Break it down; each cycle should be completable in under 5 minutes of coding

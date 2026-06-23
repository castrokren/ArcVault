# Code Integrity Guardrails

## Goal
Add static analysis, linting, and test enforcement so regressions are caught automatically — not just by manual smoke tests.

## Tasks

- [ ] **Task 1: Add `.golangci.yml`** — Configure golangci-lint with `vet`, `staticcheck`, `errcheck`, `unused`, `gocritic`. → Verify: `golangci-lint run ./...` passes locally.

- [ ] **Task 2: Add ESLint to dashboard** — `npm install --save-dev eslint @eslint/js eslint-plugin-vue`, create `eslint.config.js`, add `"lint": "eslint src"` to `package.json`. → Verify: `npm run lint` passes on `dashboard/src/`.

- [ ] **Task 3: Create CI lint+test workflow** — Add `.github/workflows/ci.yml` that triggers on push/PR to any branch (not just tags). Steps: checkout → Go setup → `go vet ./...` → `golangci-lint run` → `go test ./...` → Node setup → `npm ci && npm run lint`. → Verify: workflow appears in GitHub Actions on next push.

- [ ] **Task 4: Wire Go tests into existing build workflow** — In `build-installers.yml`, add `go test ./...` step before the build steps so a failing test aborts the release build. → Verify: step present in the job definition.

- [ ] **Task 5: Verify** — Run `golangci-lint run ./...` and `go test ./...` locally; confirm zero errors on current HEAD. Run `npm run lint` in `dashboard/`. Fix any pre-existing issues before merging.

## Done When
- [ ] `go vet`, `golangci-lint`, and `go test` all run in CI on every push
- [ ] `npm run lint` runs in CI on every push
- [ ] Release builds in `build-installers.yml` are gated on passing tests
- [ ] Zero lint/test failures on current HEAD

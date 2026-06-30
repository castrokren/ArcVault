# STATE — Plan C: Agent Credential Injection + Dashboard

## Goal
Audit and verify completion of Plan C (the credential injection pipeline: agent credential application, installer key generation, dashboard credential UI, job form integration).

## Invariants / decisions
- Plan A (credcrypto module) and Plan B (DB migrations, credential CRUD, job integration) are complete
- This is an audit/verification task — no new implementation needed
- The PLAN_STATUS.md in planning/ says "Not started" but all 8 tasks were completed in prior sessions

## Done
- ✅ Plan C Task 1: `JobCredentials` struct and `Credentials *JobCredentials` field on `Job` — exists in `agent/runner/runner.go`
- ✅ Plan C Task 2: `applyCredentials()` with SMB (cmdkey), SSH key (tmpfile), SSH password (sshpass) — exists in `agent/runner/credentials.go`
- ✅ Plan C Task 3: Wired into `process()` with defer cleanup — exists in `agent/runner/runner.go:186`
- ✅ Plan C Task 4: Tests in `agent/runner/credentials_test.go`
- ✅ Plan C Task 5: Installer generates 32-byte hex key, preserves on re-run — exists in `installer/windows/arcvault_installer.py`
- ✅ Plan C Task 6: `Credentials.vue` admin page with create/delete/409 handling — exists in `dashboard/src/views/admin/Credentials.vue`
- ✅ Plan C Task 7: Route `/admin/credentials` in `router/index.js` + nav link in `App.vue`
- ✅ Plan C Task 8: Job form credential dropdown filtered by agent OS — exists in `dashboard/src/views/Jobs.vue`

## Verification results
- ✅ `go test ./agent/...` — all pass (3 packages)
- ✅ `vite build` — 0 errors, 0 warnings
- ⚠️ `go test ./coordinator/...` — 2 pre-existing failures in `internal/bootstrap` and `internal/tlscert` (unrelated to credentials)

## In-progress
- (none — epic verified complete)

## Next
- (none — epic verified complete)

## Open questions
- Should credential API functions be added to `api.ts`? Currently `Credentials.vue` and `Jobs.vue` use raw `fetch()` calls to `/api/credential-profiles` instead of going through `api.ts`.

## File map
- STATE.md — This file (epic tracker)

# Path Auth — Plan C: Agent & Dashboard

## Goal
Add `applyCredentials` to the agent, update the setup wizard, and build the Credentials admin page + job form changes in the dashboard.

## Prereq
Plan B complete and merged.

## Tasks
- [ ] 1. Add `JobCredentials` struct and `Credentials *JobCredentials` field to `Job` struct in `agent/job.go` → Verify: `go build ./agent/...` passes
- [ ] 2. Implement `applyCredentials(job Job) (func(), error)` in `agent/credentials.go` — nil guard (no-op), SMB via cmdkey, SSH key via tmpfile, SSH password via sshpass → Verify: compiles; nil path returns no-op cleanup
- [ ] 3. Wire `applyCredentials` into `agent/process.go` — call before executor, defer cleanup, mark job failed if error returned → Verify: `go build ./agent/...` passes
- [ ] 4. Write `agent/credentials_test.go` — nil credentials (no-op), SMB happy path (mock exec), SSH key happy path (mock exec + tmpfile cleanup), error path (job marked failed) → Verify: `go test ./agent/...` passes
- [ ] 5. Update `arcvault-setup.exe` wizard — generate 32-byte hex key on fresh install (display once with save warning, write to service env block); detect existing key on re-run and skip regeneration → Verify: fresh install sets key; re-run preserves existing key
- [ ] 6. Create `dashboard/src/views/admin/Credentials.vue` — list profiles, New Credential form with conditional fields by type, delete with confirmation and 409 error toast → Verify: page loads, create/delete flows work in browser
- [ ] 7. Add `/admin/credentials` route to `dashboard/src/router/index.js` and link in Admin nav → Verify: route navigates correctly, nav link visible to admin users
- [ ] 8. Update job creation form in `dashboard/src/views/Jobs.vue` — add Path Authentication dropdown after agent selection, fetch and filter profiles by agent OS, default to None → Verify: Windows agent shows only SMB profiles; Linux agent shows only SSH profiles; None submits job without credential_profile_id

## Done When
- [ ] `go test ./agent/...` — all pass
- [ ] `go build ./agent/...` — no errors
- [ ] `npm run build` in `dashboard/` — no errors
- [ ] Manual end-to-end: create SMB profile → create job on Windows agent → agent executes with cmdkey auth → job run record shows profile name
- [ ] Manual: delete profile referenced by job → 409 toast shown in dashboard

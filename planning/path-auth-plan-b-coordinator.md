# Path Auth — Plan B: Coordinator DB, API & Dispatch

## Goal
Add `credential_profiles` table, all CRUD endpoints, job validation, and credential injection into agent-facing job responses.

## Prereq
Plan A complete and merged.

## Tasks
- [ ] 1. Add `credential_profiles` table migration to `coordinator/db.go` — id, name, type, encrypted_data, created_at → Verify: fresh DB has table; existing DB migrates cleanly
- [ ] 2. Add `credential_profile_id` column to `jobs` table migration → Verify: migration runs without error on existing DB
- [ ] 3. Add `credential_profile_id` + `credential_profile_name` columns to `job_runs` table migration → Verify: migration runs without error
- [ ] 4. Implement `POST /api/credential-profiles` handler — validate fields, call `credcrypto.Encrypt`, insert row; return 503 if key unset → Verify: curl creates profile, secrets not in response; 503 when key env unset
- [ ] 5. Implement `GET /api/credential-profiles` handler — return id, name, type, created_at only → Verify: curl returns list, no secret fields present
- [ ] 6. Implement `DELETE /api/credential-profiles/{id}` handler — 409 if any job references it, 404 if not found → Verify: delete referenced profile returns 409; unreferenced returns 204
- [ ] 7. Update `POST /api/jobs` — validate `credential_profile_id` against all target agent OS values; 422 with named agents on mismatch → Verify: smb profile + linux agent returns 422 naming the agent; compatible pair creates job
- [ ] 8. Update `GET /api/jobs` (agent-facing) — when request carries agent bearer token and job has credential_profile_id, decrypt and inject `credentials` object → Verify: agent token request includes credentials field; JWT request does not
- [ ] 9. Snapshot `credential_profile_id` + `credential_profile_name` into job run record at execution start → Verify: completed job run row has both fields populated

## Done When
- [ ] `go test ./coordinator/...` — all pass
- [ ] `go build ./coordinator/...` — no errors
- [ ] Manual: create profile → create job → fetch as agent → credentials present; fetch as dashboard user → credentials absent

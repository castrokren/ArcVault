# PLAN — v0.6.0 review fixes

Branch: `security/hardening-v0.6.0`. One commit per item, in this order.
DoD per item: listed test passes + `go build ./...` clean.

## 1. HIGH — legacy-user login lockout (must fix before merge)

`LoginRequest.Validate()` runs full complexity on the login password; users with
pre-policy passwords get 400 at login forever. `ChangePasswordRequest` also
complexity-checks `OldPassword`, so they can't migrate either.

- `coordinator/server/auth.go`:
  - `LoginRequest.Validate()` → non-empty checks only (username, password). Delete the
    `validatePasswordStrength` call. Kills the policy-leak in login errors too.
  - `ChangePasswordRequest.Validate()` → `OldPassword`: non-empty only.
    `NewPassword`: keep `validatePasswordStrength`.
  - `CreateUserRequest.Validate()`: unchanged (keeps complexity).
- Tests (`auth_hardening_test.go`): login with a legacy 8-char no-special password →
  401 (bad creds path), NOT 400; change-password weak-old → strong-new succeeds;
  change-password to weak new → 400.

## 2. MEDIUM — updater checksum fails open

Verified: latest release (v0.2.3) has no SHA256SUMS asset, so the check is a no-op today.
Two steps, in order:

1. **Release process**: generate SHA256SUMS in the release/ship script and upload it as
   an asset alongside the binaries. (Where the release is cut — installer/ship script;
   locate with `grep -r "gh release" .` — plus docs/RUNBOOK.md note.)
2. **Fail closed** (`agent/updater/updater.go` `VerifyChecksum`):
   - `checksumURL == ""` → return error, not nil.
   - Add `resp.StatusCode != http.StatusOK` → error.
   - Do NOT deploy this agent build until step 1 has shipped at least one release with
     SHA256SUMS, or every subsequent update bricks.
- Test: `VerifyChecksum("", ...)` errors; non-200 errors; existing match/mismatch tests keep passing.
- Out of scope (note in THREAT_MODEL.md): checksum shares origin with binary — protects
  transit only. Release signing is the upgrade path if compromised-release is in scope.

## 3. MEDIUM — SMB validation rejects legitimate passwords

Blacklist bans space/quote/semicolon/pipe/ampersand in passwords while the new policy
*requires* special chars. `exec.Command` passes argv without a shell, so these aren't
injectable anyway.

- `agent/runner/credentials.go` `validateCredField`:
  - host: keep `^[a-zA-Z0-9._-]+$` (this is the real guard).
  - username/password: reject only control chars — `[\x00-\x1f\x7f]`.
- Test: password `P@ss w0rd; "x" & |y` accepted; host `evil;host` rejected; embedded `\n` / NUL rejected.

## 4. LOW — Credentials.vue atob bug locks out real admins

JWT payloads are base64url; `atob` throws on `-`/`_` → catch → false → admin bounced.
Router guard (`meta.requiresRole` + `auth.hasRole`) already protects the route.

- `dashboard/src/views/admin/Credentials.vue`: delete `hasAdminAccess()` and the
  `mounted()` guard block. Pure deletion, no replacement.

## 5. LOW — whitelist is basename-only

`C:\anything\rsync.exe` passes. Acceptable ceiling for Phase 2B (jobs come from the
authenticated coordinator). Comment only:

- `agent/runner/command_whitelist.go`: `// ponytail: basename-only allowlist; full-path
  pinning or binary hashes if job sources are ever untrusted.`

## 6. LOW — client/server password-policy mismatch

Server requires all 4 character classes; `Users.vue` accepts 3; `ChangePasswordModal.vue`
blocks only ≤2.

- `dashboard/src/views/Users.vue`: `classes < 3` → `classes < 4`.
- `dashboard/src/components/ChangePasswordModal.vue`: `classes <= 2` weak → `classes < 4`
  weak (label already names all four).
- Check: enter a 3-class password in both forms → blocked client-side with the message.

## 7. Cleanup (one commit)

- Delete dead `isAgentTokenRequest` (`coordinator/server/credentials.go:186-…`).
- Extract the 4× duplicated agent-ID injection block in `server.go` middlewares into
  `func (s *Server) withAgentID(r *http.Request, token string) *http.Request`.
- `go build ./... && go test ./coordinator/server/...` — no behavior change expected.

## 8. Pre-deploy check (no code)

Fail-closed credential injection means agents whose token has no `agent_id` row get no
job credentials. Before deploying:
`sqlite3 <db> "SELECT token, agent_id FROM tokens WHERE agent_id IS NULL OR agent_id=''"`
— regenerate any hits via the CLI (same path as the deploy-script fix in 27cbfa9).

## Not doing

- Token hashing at rest, release signing, unicode-aware `containsSpecial` — pre-existing
  / out of scope for this round; noted in THREAT_MODEL.md if wanted.

## Known-broken elsewhere

`tlscert` + `bootstrap` tests fail on files this branch never touched (pre-existing).
Full `go test ./...` will not be green; per-package DoD above instead.

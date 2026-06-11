# ArcVault Security Hardening

## Goal
Fix 9 confirmed vulnerabilities across coordinator, agent, and dashboard — all code-level fixes that work for both local and corporate deployment.

## Tasks

### Critical / High

- [ ] **H1 — Template job command allowlist**
  - Add `allowed_commands []string` to agent `Config` struct (`agent/config/config.go`)
  - In `executor.go` `RealExecutor`, if `job.Command != ""`, check it against the allowlist before exec — fail the job (exit code 1, descriptive output) if no match
  - If `allowed_commands` is empty/unset, **block all template commands** (secure by default)
  - → Verify: `go test ./agent/runner/...` passes; unallowed command returns non-zero exit + error message

- [ ] **H3 — JWT out of localStorage, into HTTP-only cookie**
  - Backend (`coordinator/server/auth.go`): on successful login, set `Set-Cookie: arcvault_session=<token>; HttpOnly; Secure; SameSite=Strict; Path=/`
  - Backend: update `requireAuth` middleware to also accept the `arcvault_session` cookie (alongside existing Bearer header, for backward compat with agents/CLI)
  - Frontend (`useAuth.js`): remove all `localStorage.setItem('arcvault_jwt', ...)` and `arcvault_token` writes; token no longer stored client-side
  - Frontend: keep `arcvault_user` (username, role) in localStorage — not a secret
  - Frontend: remove `Authorization: Bearer` header from dashboard API calls — cookie is sent automatically
  - → Verify: Login sets cookie; page reload restores session; localStorage has no JWT keys; agents (Bearer token) still work

- [ ] **H4 — Enforce SHA256 checksum on update**
  - In `coordinator/updater/updater.go` `VerifyChecksum`: return `fmt.Errorf("no SHA256SUMS file in release — update aborted")` if `checksumURL == ""`  (remove the silent skip)
  - Add a note in the GitHub Actions release workflow (or `docs/`) that `SHA256SUMS` must be included in every release asset
  - → Verify: `go test ./coordinator/updater/...` — empty checksumURL now returns error

### Medium

- [ ] **M4 — WebSocket CheckOrigin restriction**
  - In `coordinator/server/hub.go`, replace `CheckOrigin: func(*http.Request) bool { return true }` with a function that validates `r.Header.Get("Origin")` against `cfg.AllowedOrigins` — same logic as `corsMiddleware`
  - If `AllowedOrigins` is empty, allow same-host only (compare origin host to `r.Host`)
  - → Verify: WS upgrade from an unlisted origin returns 403; dashboard WS still connects

- [ ] **M5 — Security headers middleware**
  - Add `securityHeadersMiddleware` in `coordinator/server/server.go` that sets:
    - `Content-Security-Policy: default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'`
    - `X-Frame-Options: DENY`
    - `X-Content-Type-Options: nosniff`
    - `Referrer-Policy: strict-origin-when-cross-origin`
  - Chain it after `corsMiddleware` in `Serve()`
  - → Verify: `curl -I https://localhost` returns all four headers

- [ ] **M6 — Remove `GET /api/admin/token` endpoint**
  - Delete `handleGetAdminToken` from `auth.go`
  - Remove the route `GET /api/admin/token` from `server.go`
  - Admin token is in `config.json` — ops reads it there; no need for an API that echoes it
  - → Verify: `curl /api/admin/token` returns 404; `go build ./coordinator` passes

- [ ] **M7 — Log SSHPASS visibility warning**
  - In `credentials.go` `applySSHPasswordCredentials`, add `log.Printf("WARNING: SSHPASS environment variable is set — SSH password is briefly visible to other processes on this machine")` before `os.Setenv`
  - → Verify: SSH password job logs the warning; cleanup still unsets the var

- [ ] **M3 — Env var overrides for secrets**
  - In config loading (`coordinator/config/config.go`), after unmarshalling `config.json`, check:
    - `ARCVAULT_JWT_SECRET` → overrides `cfg.JWTSecret`
    - `ARCVAULT_ADMIN_TOKEN` → overrides `cfg.AdminToken`
    - `ARCVAULT_CREDENTIAL_KEY` → overrides `cfg.CredentialKey`
  - Log `"[config] JWT secret loaded from ARCVAULT_JWT_SECRET env var"` (no value) when env var wins
  - → Verify: `ARCVAULT_JWT_SECRET=test ./coordinator` starts and uses the env var value

- [ ] **H2 — Production TLS warning + example config**
  - In `coordinator/server/server.go` `Serve()`, if `cfg.Environment == "production" && !cfg.ExternalTLS`, log: `WARNING: production environment is using a self-signed certificate. Set external_tls: true and terminate TLS at a reverse proxy with a CA-signed certificate.`
  - Add `config.example.json` to repo root with production-safe defaults and inline comments (external_tls, allowed_origins, environment)
  - → Verify: Starting coordinator with `"environment": "production"` and no `external_tls` prints the warning

## Done When
- [ ] All 9 tasks above pass their verification steps
- [ ] `go build ./coordinator` and `go build ./agent` succeed
- [ ] `go test ./coordinator/...` and `go test ./agent/...` pass
- [ ] Dashboard login/logout/session works end-to-end with cookie auth

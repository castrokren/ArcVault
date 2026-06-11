# TLS Cert & Agent Bootstrap

## Goal
Coordinator auto-generates a self-signed TLS cert, serves HTTPS, and produces a one-click PowerShell bootstrap script that fully configures a remote agent install.

## Tasks

- [ ] **Task 1: `tlscert` module** — create `coordinator/internal/tlscert/cert.go` with `Generate()`, `EnsureExists()`, `Load()`, `ReadCertPEM()`. ECDSA P-256, 10-year validity, IP/DNS SAN. Write `cert_test.go` (4 tests).
  → Verify: `go test ./coordinator/internal/tlscert/...` passes

- [ ] **Task 2: `bootstrap` module** — create `coordinator/internal/bootstrap/bootstrap.go` with `GenerateScript(Params)`. Outputs PS1 that creates dir, downloads agent.exe with `-SkipCertificateCheck`, writes `coordinator.crt`, writes `agent-config.yaml`, installs + starts service, sets SCM recovery. Write `bootstrap_test.go` (4 tests).
  → Verify: `go test ./coordinator/internal/bootstrap/...` passes

- [ ] **Task 3: Config changes** — add `Host`, `CertFile`, `KeyFile` to `coordinator/config/config.go`. Default `CertFile`/`KeyFile` to `<exe-dir>/cert.pem` and `<exe-dir>/key.pem` when empty.
  → Verify: existing config tests still pass; new fields marshal/unmarshal correctly

- [ ] **Task 4: `coordinator init` changes** — in `coordinator/cmd/commands.go`, prompt for host IP after port/db prompts, save to config, call `tlscert.Generate(host, certFile, keyFile)`, print cert path.
  → Verify: `coordinator init` creates `cert.pem` + `key.pem` in exe dir

- [ ] **Task 5: `coordinator start` → HTTPS** — in `coordinator/cmd/commands.go` `StartCommand`, call `tlscert.EnsureExists()`, load cert, switch `srv.ListenAndServe` to `srv.ListenAndServeTLS(certFile, keyFile)`. Fall back to HTTP with warning log if cert files unresolvable.
  → Verify: `https://<host>:8080/health` returns 200; browser shows untrusted cert warning (expected)

- [ ] **Task 6: New coordinator routes** — add two admin-only handlers in `coordinator/server/`:
  - `GET /downloads/agent.exe` — reads `agent.exe` from exe dir, streams as `application/octet-stream`
  - `GET /api/admin/bootstrap.ps1` — calls `bootstrap.GenerateScript()` with live config values, returns as `text/plain` with `Content-Disposition: attachment; filename="bootstrap.ps1"`
  → Verify: curl both endpoints as admin, get expected responses

- [ ] **Task 7: Agent CA cert support** — add `CACertFile string` to `agent/config/config.go`. In `runner.go`, `heartbeat.go`, `ws.go`: when `ca_cert_file` set, load PEM into `x509.CertPool` and inject into HTTP client `TLSClientConfig.RootCAs`.
  → Verify: `go test ./agent/...` passes; agent with `ca_cert_file` set connects to HTTPS coordinator

- [ ] **Task 8: Dashboard button** — add "Download Agent Installer" button to `Users.vue` header (next to Copy Admin Token). On click, fetch `GET /api/admin/bootstrap.ps1`, trigger file download via blob URL.
  → Verify: button appears for admin users; clicking downloads `bootstrap.ps1`

- [ ] **Task 9: Installer simplification** — remove agent-only card from `installer/windows/arcvault_installer.py`. Keep coordinator-only and coordinator+agent options.
  → Verify: installer launches, shows only two component options

- [ ] **Task 10: End-to-end verification** — rebuild with `.\scripts\rebuild-and-restart.ps1`, download bootstrap script from dashboard, run on test PC, verify agent service starts and registers in coordinator.
  → Verify: test PC agent appears in Agents list in dashboard

## Done When
- [ ] Remote agent connects to coordinator over HTTPS using cert from bootstrap script
- [ ] All Go tests pass
- [ ] Bootstrap script runs on a clean Windows machine without manual config

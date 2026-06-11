# TLS Certificate & Agent Bootstrap — Implementation Plan
**Date:** 2026-06-07
**Targets:** Coordinator + Agent **v0.5.0**
**Supersedes parts of:** `2026-06-07-tls-cert-bootstrap-design.md` (approved)
**Execution:** Claude Code, sequential Plans A→E, TDD per task. PowerShell is the local shell.

---

## Design Amendments (deltas from the approved spec)

These are the changes agreed after design review. Where they conflict with the original spec, **these win.**

1. **Port 443.** HTTPS serves on **443**, not 8080. Port becomes a config field (`port`, default 443) — not hardcoded. Bootstrap URL omits the port when it's 443 (`https://192.168.1.10`), includes it otherwise.
2. **Embed an agent token, not the admin token.** `/api/admin/bootstrap.ps1` mints a fresh `role='agent'` token (via existing `CreateAgentToken`) and embeds *that*. A compromised agent box can no longer own the fleet.
3. **Download-route auth, resolved.** `/api/admin/bootstrap.ps1` is **admin-only** (only an admin mints tokens). `/downloads/agent.exe` accepts an **agent token or admin token** — the script authenticates with its embedded agent token. (Original spec marked the download admin-only, which the embedded agent token could not satisfy.)
4. **Cert-first, then download.** Script writes the pinned cert before any download. Download integrity is guaranteed by a **mandatory SHA-256 check** against a hash stamped into the script at generation time. TLS pinning is defence-in-depth on top.
5. **PS 5.1 is the target.** No `-SkipCertificateCheck` on 5.1 (doesn't exist there). Force TLS 1.2 explicitly. Cross-edition: 5.1 pins via `ServicePointManager` callback; PS 7+ uses `-SkipCertificateCheck` and relies on the mandatory SHA-256. See the script template file.
6. **Cert correctness.** Self-signed **leaf** (`IsCA:false`, no `CertSign`). EKU = **ServerAuth**. ECDSA P-256 KeyUsage = **DigitalSignature only** (drop `KeyEncipherment` — it's an RSA-key-transport usage and wrong for ECDSA). SAN set = `host` + `localhost` + `127.0.0.1` + machine hostname (CN-only matching is dead; multi-SAN keeps browser/localhost/same-machine installs working).
7. **No silent HTTP fallback.** BYO-cert / external-terminator is an **explicit** `external_tls: true` config flag. If `external_tls` is false and HTTPS can't start, the coordinator **fails loudly and exits** — it does not quietly degrade to HTTP (that's the exact failure this feature exists to prevent).
8. **`coordinator rekey-cert`.** New CLI subcommand (mirrors the existing `rekey` for credcrypto) that force-regenerates the cert. `EnsureExists` is a no-op when files exist, so a host/IP change has no recovery without this.
9. **Idempotent script.** Re-running on an onboarded box stops + deletes the existing service with a wait loop (handles exit 1060 "does not exist" / "marked for deletion") before reinstalling — same pattern as the installer fix in Session 15.
10. **Dashboard download is an authed fetch→blob**, not a plain `<a href>` (the route requires the JWT).
11. **Federation deferred.** Spoke↔root TLS trust is a documented follow-up (federation is built but unused). See "Out of Scope."

---

## Out of Scope (this plan)
- Federation spoke↔root TLS trust (federation dormant). When spokes deploy: spoke needs the root's CA cert and the `federation_sync` HTTP client needs `RootCAs`. Add then.
- Cert renewal / ACME / revocation / mTLS (unchanged from spec).
- `agent_id` clone-collision hardening — dev-only fleet; `$env:COMPUTERNAME` is acceptable. Noted as a known limitation in the script.

---

# Plan A — `tlscert` module

**Goal:** A self-contained cert lifecycle module with zero knowledge of config/HTTP.

**Files:**
- New: `coordinator/internal/tlscert/tlscert.go`
- New: `coordinator/internal/tlscert/tlscert_test.go`

**Tasks (TDD — write the test, watch it fail, then implement):**
- [ ] A1: `TestGenerate_filesAndParse` — `Generate("192.168.1.10", cert, key)` writes both files; cert PEM parses via `x509.ParseCertificate`. → Verify: `go test ./coordinator/internal/tlscert/ -run TestGenerate_filesAndParse`
- [ ] A2: `TestGenerate_SANs` — generated cert's `IPAddresses`/`DNSNames` contain `127.0.0.1`, `localhost`, the host, and `os.Hostname()`. → Verify: test passes
- [ ] A3: `TestGenerate_leafEKU` — cert has `ExtKeyUsageServerAuth`, `IsCA == false`, `KeyUsage == DigitalSignature` (no CertSign/KeyEncipherment). → Verify: test passes
- [ ] A4: `TestTLSHandshake` — start `httptest.NewUnstartedServer`, assign `tlscert.Load(...)` cert, `StartTLS`; client with a `CertPool` containing only the generated cert + URL host in SAN → `GET` returns 200. → Verify: test passes (**this is the test that proves the whole module actually works**)
- [ ] A5: `TestTLSHandshake_wrongHost` — same server, client dials a hostname NOT in the SAN set → handshake/verify fails. → Verify: test passes
- [ ] A6: `TestEnsureExists_createsAndNoops` — absent → generates; present → mtime unchanged. → Verify: test passes
- [ ] A7: Implement `Generate`, `EnsureExists`, `Load`, `ReadCertPEM` to green all of A1–A6. ECDSA P-256, 10y validity, PKCS#8 private key PEM (`PRIVATE KEY` block). → Verify: `go test ./coordinator/internal/tlscert/` all pass

**Done When:** All A tests green; `go vet ./coordinator/internal/tlscert/` clean.
**Hard Stop:** If A4 or A5 cannot be made to pass, STOP — the SAN/EKU model is wrong; do not proceed to Plan C.

---

# Plan B — `bootstrap` module

**Goal:** Pure script generator. Interpolates params into the PS 5.1 template. No HTTP/TLS/config knowledge.

**Files:**
- New: `coordinator/internal/bootstrap/bootstrap.go`
- New: `coordinator/internal/bootstrap/bootstrap_test.go`
- Reference: `bootstrap-script-template.ps1` (the template this module emits — see companion file)

**Params struct (amended):**
```go
type Params struct {
    CoordinatorURL string // https://192.168.1.10  (port omitted when 443)
    AgentToken     string // role=agent token, minted per generation
    CertPEM        string // written to disk on the agent as ca_cert_file
    CertThumbprint string // SHA-1 hex, UPPERCASE, no separators — PS pinning
    AgentExeSHA256 string // UPPERCASE hex — mandatory post-download integrity
}
```

**Tasks (TDD):**
- [ ] B1: `TestGenerateScript_containsURL` / `_containsToken` / `_containsComputername` — output references each. → Verify: `go test ./coordinator/internal/bootstrap/`
- [ ] B2: `TestGenerateScript_certInSingleQuotedHeredoc` — the PEM sits inside a single-quoted here-string (`@'` … `'@`), so `$`/backtick in PEM never expand. → Verify: assert the `@'`/`'@` delimiters bracket the PEM
- [ ] B3: `TestGenerateScript_forcesTls12` — output contains the `SecurityProtocol`/`Tls12` line. → Verify: substring present
- [ ] B4: `TestGenerateScript_mandatorySha256` — output contains the SHA-256 compare + `throw` on mismatch. → Verify: substring present
- [ ] B5: `TestGenerateScript_crossEdition` — output branches on `$PSVersionTable.PSEdition` (callback pin on Desktop, `-SkipCertificateCheck` on Core). → Verify: both branches present
- [ ] B6: Implement `GenerateScript(p Params) string` to green B1–B5. → Verify: all B tests pass

**Done When:** All B tests green; emitted script byte-for-byte matches the template with placeholders filled.
**Hard Stop:** If the PEM round-trips with `$`-expansion corruption, STOP and fix the here-string before wiring into the coordinator.

---

# Plan C — Coordinator integration

**Goal:** Config fields, `init` prompt, HTTPS on 443, two new routes, token minting, `rekey-cert`.

**Files:**
- Modify: `coordinator/config/config.go`
- Modify: `coordinator/cmd/commands.go`
- Modify: `coordinator/server/server.go`
- New: `coordinator/server/bootstrap_handler.go`
- New: `coordinator/server/downloads.go`

**Tasks:**
- [ ] C1: Config — add `Host`, `Port` (default 443), `CertFile`/`KeyFile` (default `<exe-dir>/cert.pem`,`key.pem`), `ExternalTLS bool`. Backward-compatible (`omitempty`). → Verify: `go test ./coordinator/config/`
- [ ] C2: `coordinator init` — after admin token, prompt for `host` (IP/hostname), save to config, call `tlscert.Generate(host, certFile, keyFile)`, print cert path. → Verify: run `coordinator init` in a temp dir → `cert.pem`/`key.pem` exist, `config.json` has `host` + `port:443`
- [ ] C3: `coordinator start` — if `ExternalTLS` → plain HTTP + warn loudly. Else `tlscert.EnsureExists` → `Load` → `ListenAndServeTLS(":"+port, ...)`. On TLS start error: `log.Fatal` (no HTTP fallback). → Verify: `coordinator start` → `Invoke-RestMethod https://<host>/api/...` succeeds; confirm nothing answers on 8080
- [ ] C4: `coordinator rekey-cert` subcommand — force `tlscert.Generate` regardless of existing files; print new path. → Verify: run twice, confirm cert serial/mtime changes
- [ ] C5: `GET /downloads/agent.exe` — serve `agent.exe` from the coordinator's exe dir; 500 with a clear message if absent. **Auth: agent token OR admin token** (reuse the agent-token-accepting middleware). → Verify: `curl -H "Authorization: Bearer <agent-token>"` returns the binary; no header → 401
- [ ] C6: `GET /api/admin/bootstrap.ps1` — **admin-only**. Mint a fresh agent token (`CreateAgentToken`), read cert PEM (`tlscert.ReadCertPEM`), compute cert SHA-1 thumbprint + `agent.exe` SHA-256, build `CoordinatorURL` (omit `:443`), call `bootstrap.GenerateScript`, return as `Content-Disposition: attachment; filename=bootstrap.ps1`. → Verify: authed `curl` returns a valid PS script containing a freshly-minted token
- [ ] C7: Register both routes in `server.go`. → Verify: routes resolve; `go test ./coordinator/server/`

**Done When:** Coordinator serves HTTPS on 443; both routes behave per auth rules; `coordinator rekey-cert` works; full `go test ./coordinator/...` green.
**Hard Stop:** If 443 is already bound on the dev box (IIS / other service), STOP — resolve the conflict or temporarily set `port` before continuing; do not silently pick another port.

---

# Plan D — Agent CA trust

**Goal:** Agent verifies the self-signed coordinator cert via a pinned CA file; system roots when unset.

**Files:**
- Modify: `agent/config/config.go`
- Modify: `agent/runner/runner.go`, `agent/heartbeat/heartbeat.go`, `agent/ws/ws.go`
- New: `agent/config/tlsclient.go` (shared `*http.Client` / `tls.Config` builder)

**Tasks (TDD where practical):**
- [ ] D1: Config — add `CACertFile string \`yaml:"ca_cert_file,omitempty"\``. → Verify: `go test ./agent/config/`
- [ ] D2: `tlsclient.go` — `BuildTLSConfig(caCertFile string) (*tls.Config, error)`: if set, read PEM into a fresh `x509.CertPool`, set `RootCAs`; if empty, return nil (system roots). → Verify: unit test — pool contains 1 cert when file set, nil config when empty
- [ ] D3: `TestAgentTrustsPinnedCert` — stand up an `httptest` TLS server using a `tlscert`-generated cert; build the agent client with `ca_cert_file` = that cert → request succeeds; with empty `ca_cert_file` → fails (untrusted). → Verify: test passes
- [ ] D4: Wire `BuildTLSConfig` into the HTTP clients in `runner.go` + `heartbeat.go`, and into `gorilla/websocket` `Dialer.TLSClientConfig` in `ws.go`. → Verify: agent connects to the live HTTPS coordinator and registers
- [ ] D5: Confirm the existing HTTPS-required-for-non-localhost guard in `runner.go` is unchanged. → Verify: read the guard, no edit needed

**Done When:** A real agent with `ca_cert_file` set connects to the live coordinator over HTTPS and appears online; `go test ./agent/...` green.
**Hard Stop:** If D3 only passes with `InsecureSkipVerify`, STOP — pinning is broken; the SAN/host in the URL must match the cert (revisit Plan A SANs).

---

# Plan E — Dashboard, installer, verification & release

**Goal:** Download button, installer cleanup, end-to-end proof, tag v0.5.0.

**Files:**
- Modify: `dashboard/src/views/Users.vue`, `dashboard/src/api.ts`
- Modify: `installer/windows/arcvault_installer.py`
- Modify: `CONTEXT.md`, `MEMORY.md`

**Tasks:**
- [ ] E1: `api.ts` — `downloadBootstrapScript()`: `fetch('/api/admin/bootstrap.ps1', { headers: { Authorization: 'Bearer ' + getToken() } })` → `blob()` → object URL → trigger `bootstrap.ps1` download. (NOT a plain `<a href>` — route is authed.) → Verify: button downloads a non-empty `.ps1`
- [ ] E2: `Users.vue` — add "Download Agent Installer" button beside "Copy Admin Token" (admin-only). → Verify: visible to admin, hidden for viewer/operator
- [ ] E3: Installer — remove the agent-only card; keep "Coordinator only" + "Coordinator + Agent". (Use `bash` heredoc `cat > … << 'PYEOF'`, not the Edit tool — null-byte truncation, per Session 20.) → Verify: installer launches, two cards only
- [ ] E4: **Build pipeline** — `.\scripts\rebuild-and-restart.ps1` (Vue build → clear+copy `coordinator/static/dist` → Go build with ldflags `-X main.Version=v0.5.0` → deploy → restart). Confirm `agent.exe` lands in the same dir as `coordinator.exe`. → Verify: dashboard loads over HTTPS; `/api/update/check` reports `v0.5.0`
- [ ] E5: **End-to-end (manual)** — on a second machine: download `bootstrap.ps1`, run elevated in **Windows PowerShell 5.1**, confirm: cert written → SHA-256 check passes → service starts → agent shows online in dashboard. → Verify: agent online + a test backup runs
- [ ] E6: Update `CONTEXT.md` + `MEMORY.md` (close-of-phase discipline). Note v0.4.0→v0.5.0; correct the stale version table in `MEMORY.md`. Reissue both as downloadable files. → Verify: files written
- [ ] E7: **Release (PowerShell, backtick continuation):**
  ```powershell
  cd C:\Projects\ArcVault2.0
  git add -A
  git commit -m "feat: HTTPS/TLS serving + agent bootstrap script (v0.5.0)"
  git tag v0.5.0
  git push origin main `
    --tags
  ```
  → Verify: `gh release view v0.5.0` shows the tag

**Done When:** E5 passes on a clean second machine; v0.5.0 tagged and pushed; memory files updated.
**Hard Stop (Verification Gate — always last):** Do not tag v0.5.0 until E5 shows a real agent online + a real backup completing. Status-only ("service started") is not proof — copied files on disk are the proof (Phase 21a-4 lesson).

---

## Risks / watch-items
- **443 conflict** on the dev box (IIS, Skype-legacy, other). Pre-check: `Get-NetTCPConnection -LocalPort 443 -State Listen -ErrorAction SilentlyContinue`.
- **PS 7 (`pwsh`) vs 5.1** — script auto-branches, but the cert-pinning path only runs on 5.1. The SHA-256 check is the integrity guarantee on both; document "run in Windows PowerShell."
- **Browser cert warning** every visit to `https://<host>` (self-signed). Expected; optionally trust the cert in the OS store. Document, don't fix.
- **`agent.exe` staleness** — coordinator self-update only swaps `coordinator.exe`; `/downloads/agent.exe` can serve an older agent that immediately flags for update. `rebuild-and-restart.ps1` co-deploys both, so it's in lockstep at deploy time — just don't expect coordinator self-update alone to refresh the served agent.

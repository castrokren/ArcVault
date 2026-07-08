# TLS Certificate & Agent Bootstrap Design
**Date:** 2026-06-07  
**Status:** Approved  
**Author:** Kren Castro  

---

## Overview

ArcVault coordinators currently serve over plain HTTP, which blocks agent connections from remote machines (the agent enforces HTTPS for non-localhost URLs). This design adds:

1. A standalone `tlscert` module that generates and manages self-signed TLS certificates
2. A standalone `bootstrap` module that generates a pre-configured PowerShell installer script
3. Coordinator changes to serve HTTPS and expose the bootstrap script + agent binary
4. Agent changes to load a CA cert file for verifying self-signed coordinator certs
5. A dashboard button to download the bootstrap script
6. Installer simplification (remove agent-only option)

---

## Goals

- Remote agents can connect to the coordinator over HTTPS out of the box
- No manual cert handling by the admin — cert is generated automatically
- New machines can be onboarded by running a single PowerShell script
- Admins can bring their own trusted cert if desired
- Both new modules are fully isolated and independently testable

---

## New Modules

### `coordinator/internal/tlscert/`

Handles certificate lifecycle only. No knowledge of config, HTTP server, or other packages.

**API:**
```go
// Generate creates a new self-signed cert valid for 10 years.
// host is used as the Common Name and SAN (IP or hostname).
// Writes cert PEM to certPath and key PEM to keyPath.
func Generate(host, certPath, keyPath string) error

// EnsureExists generates the cert if either file is missing, otherwise no-ops.
func EnsureExists(host, certPath, keyPath string) error

// Load reads cert and key from disk and returns a tls.Certificate.
func Load(certPath, keyPath string) (tls.Certificate, error)

// ReadCertPEM returns the raw PEM bytes of the cert file (for embedding in bootstrap scripts).
func ReadCertPEM(certPath string) ([]byte, error)
```

**Cert properties:**
- Algorithm: ECDSA P-256
- Validity: 10 years from generation date
- Subject: CN=`host`
- SANs: IP address (if host is an IP) or DNS name
- Key usage: KeyEncipherment, DigitalSignature, CertSign
- Self-signed (CA: true)

### `coordinator/internal/bootstrap/`

Generates a self-contained PowerShell installer script. No knowledge of HTTP, config, or TLS internals.

**API:**
```go
type Params struct {
    CoordinatorURL string // https://192.168.1.10:8080
    AdminToken     string
    CertPEM        string // PEM content to embed in script
}

// GenerateScript returns a complete PowerShell script as a string.
func GenerateScript(p Params) string
```

**Generated script behavior (runs on target Windows machine):**
1. Creates `C:\ArcVault-Agent\`
2. Downloads `agent.exe` from `<CoordinatorURL>/downloads/agent.exe` using `-SkipCertificateCheck` (cert not yet trusted at download time)
3. Writes embedded cert PEM to `C:\ArcVault-Agent\coordinator.crt`
4. Writes `C:\ArcVault-Agent\agent-config.yaml`:
   ```yaml
   agent_id: <$env:COMPUTERNAME>
   coordinator_url: <CoordinatorURL>
   auth_token: <AdminToken>
   ca_cert_file: C:\ArcVault-Agent\coordinator.crt
   ```
5. Installs `arcvault-agent` Windows service: `agent.exe install-service`
6. Starts the service: `sc.exe start arcvault-agent`
7. Sets SCM failure recovery: `sc.exe failure arcvault-agent reset=86400 actions=restart/3000/restart/3000/restart/3000`

---

## Coordinator Changes

### Config (`coordinator/config/config.go`)

Three new optional fields:
```go
Host     string `json:"host,omitempty"`      // IP or hostname for cert SAN + bootstrap URL (e.g. "192.168.1.10")
CertFile string `json:"cert_file,omitempty"` // default: <exe-dir>/cert.pem
KeyFile  string `json:"key_file,omitempty"`  // default: <exe-dir>/key.pem
```

`Host` is required for cert generation and bootstrap script generation. If not set, `coordinator init` prompts for it. `CertFile` and `KeyFile` default to `<exe-dir>/cert.pem` and `<exe-dir>/key.pem` if empty.

### `coordinator init`

After generating the admin token, prompts for the coordinator's host IP or hostname (e.g. `192.168.1.10`), saves it to `config.json` as `host`, calls `tlscert.Generate(host, certFile, keyFile)`, and prints the cert path.

### `coordinator start`

On startup:
1. Calls `tlscert.EnsureExists(host, certFile, keyFile)` — generates cert if missing
2. Loads cert via `tlscert.Load(certFile, keyFile)`
3. Switches from `ListenAndServe` to `ListenAndServeTLS`
4. **Fallback:** If cert/key paths are not resolvable (e.g. external TLS terminator), logs a warning and falls back to plain HTTP — supports bring-your-own-cert setups

### New Routes

Both are admin-only:

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/downloads/agent.exe` | Serves the embedded agent binary |
| `GET` | `/api/admin/bootstrap.ps1` | Generates and returns the bootstrap script as a file download |

The agent binary is served by reading `agent.exe` from the coordinator's own directory (the same directory as `coordinator.exe`). This avoids bloating the coordinator binary — both binaries are already co-located after every deploy via `rebuild-and-restart.ps1`.

---

## Agent Changes

### `agent/config/config.go`

One new optional field:
```yaml
ca_cert_file: C:\ArcVault-Agent\coordinator.crt
```

```go
CACertFile string `yaml:"ca_cert_file,omitempty"`
```

### HTTP Client Trust

In `runner.go`, `heartbeat.go`, and `ws.go`: when `ca_cert_file` is set, load the PEM file and add it to a custom `x509.CertPool`. Inject into the HTTP client's `TLSClientConfig.RootCAs`. This verifies the coordinator's self-signed cert without disabling verification entirely.

If `ca_cert_file` is empty, the agent uses system root CAs (for bring-your-own-cert with a trusted CA).

### URL Validation

The existing restriction in `runner.go` (HTTPS required for non-localhost) stays unchanged — HTTPS is now the enforced path for remote connections.

---

## Dashboard Changes

**Users page** (`dashboard/src/views/Users.vue`) — add a "Download Agent Installer" button next to the existing "Copy Admin Token" button in the header. On click, calls `GET /api/admin/bootstrap.ps1` and triggers a `bootstrap.ps1` file download in the browser.

---

## Installer Changes

Remove the agent-only install card from `installer/windows/arcvault_installer.py`. The coordinator now owns agent distribution via the bootstrap script. Two remaining options:

- **Coordinator only** — for installing the coordinator service
- **Coordinator + Agent** — for same-machine installs (e.g. dev/test)

---

## Testing

### `tlscert` package
- `TestGenerate` — generates cert, verifies files exist, cert parses without error
- `TestEnsureExists_creates` — files absent → generates
- `TestEnsureExists_noop` — files present → no change to mtime
- `TestLoad` — round-trips generate → load, verifies cert is valid

### `bootstrap` package
- `TestGenerateScript_contains_url` — output contains the coordinator URL
- `TestGenerateScript_contains_token` — output contains the admin token
- `TestGenerateScript_contains_cert` — output contains the cert PEM
- `TestGenerateScript_contains_computername` — output references `$env:COMPUTERNAME`

### Integration (manual)
1. Run `coordinator init` → verify `cert.pem` and `key.pem` created in exe dir
2. Run `coordinator start` → verify HTTPS on configured port
3. Click "Download Agent Installer" → verify `bootstrap.ps1` downloads
4. Run script on a second machine → verify agent service starts and registers

---

## Bring Your Own Cert

If an admin wants to use a trusted CA cert (e.g. from an internal PKI or a public CA):

1. Place `cert.pem` and `key.pem` in the coordinator's exe directory (or configure `cert_file`/`key_file` in `config.json`)
2. The coordinator will use those files instead of generating a self-signed cert
3. The bootstrap script will embed that cert as the CA cert on the agent side
4. Agents with `ca_cert_file` set to a trusted CA cert will verify normally

No code changes needed for this path — it falls out of the design naturally.

---

## Out of Scope

- Automatic cert renewal (10-year validity makes this a non-issue for now)
- ACME / Let's Encrypt integration
- Cert revocation
- mTLS (client certificates for agents)

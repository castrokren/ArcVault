# ArcVault 2.0 — Encryption & Security

## Data Encryption In Transit

### ✅ All Network Traffic Encrypted (TLS 1.2+)

| Communication Path | Protocol | Encryption | Certificate | Notes |
|---|---|---|---|---|
| Browser → Coordinator | HTTPS (REST) | ✅ TLS 1.2+ | Self-signed | Origin-validated on handshake |
| Browser ↔ Coordinator | WebSocket | ✅ WSS (TLS) | Self-signed | Origin header validation in production |
| Agent → Coordinator | HTTPS | ✅ TLS 1.2+ | **Pinned** | Agent verifies coordinator.crt during connection |
| Agent ← Coordinator | HTTPS | ✅ TLS 1.2+ | **Pinned** | Same pinned cert validation |
| Agent ↔ Coordinator | WebSocket | ✅ WSS (TLS) | **Pinned** | TLS pinning extends to WebSocket connections |
| Coordinator ↔ Coordinator | HTTPS + WS | ✅ TLS 1.2+ (mTLS) | Mutual certs | Federation: peer-to-peer with cert validation |
| Coordinator → Alerts | HTTPS | ✅ TLS 1.2+ | System CA | Webhooks to Slack/Teams/custom endpoints |
| Coordinator → GitHub | HTTPS | ✅ TLS 1.2+ | System CA | Release update checks |

### ⚠️ Network Paths NOT Encrypted by ArcVault

| Path | Encryption | Notes |
|---|---|---|
| Agent → Backup Target (SMB) | ❌ Optional | Controlled by Windows SMB policy; use SMB3 with signing/encryption enabled on production |
| Agent → Backup Target (rsync) | ❌ No encryption | rsync uses SSH for encryption; ArcVault does not configure SSH |
| Coordinator → SQLite | ❌ No encryption | Database on local disk; use full-disk encryption for production |

---

## Certificate Management

### TLS Certificate Generation

```bash
# Generate on first setup
coordinator init

# Regenerate if hostname/IP changes
coordinator rekey-cert
```

**Location:** `C:\ArcVault\cert.pem` and `C:\ArcVault\key.pem`

**SANs (Subject Alternate Names) included:**
- IP: 127.0.0.1, localhost
- IP: Your configured coordinator IP
- DNS: localhost, your hostname

### Agent Certificate Pinning

1. **During registration:** Agent downloads coordinator's certificate from `/api/agents/register` response
2. **Stored as:** `C:\ArcVault-Agent\coordinator.crt`
3. **Validation:** Every HTTPS request validates coordinator's cert against pinned copy
4. **Benefit:** Prevents MITM attacks even if system root CA is compromised

**Automatic setup via installer:**
- `ArcVault-Setup-0.5.1-windows-amd64.exe` copies coordinator cert to agents during combined install
- Bootstrap script retrieves and pins cert on agent-only installs

---

## Credential Encryption

### At Rest

Backup credentials (passwords, SSH keys) encrypted using:
- **Algorithm:** AES-256-GCM (authenticated encryption)
- **Key source:** `credential_key` in `config.json` (64-char hex string)
- **Fallback:** Environment variable `ARCVAULT_CREDENTIAL_KEY` if config key not found

**Retrieval on reinstall:**
- Installer reads existing `credential_key` from `config.json`
- If upgrading, existing credentials remain valid (same key used)
- If fresh install, new random key generated (credentials unrecoverable if lost)

### In Database

SQLite database (WAL mode) stores encrypted credentials in `credential_profiles` table. Database file itself is NOT encrypted — use **full-disk encryption** on production coordinator.

---

## Security Configuration

### Production Checklist

- [ ] TLS certificate generated with correct SANs for your hostname/IP
- [ ] Coordinator running with `"environment": "production"` in config.json
- [ ] WebSocket origin validation enabled (default in production mode)
- [ ] Agents have pinned coordinator certificate
- [ ] Credential key backed up in secure location (hex string from config.json)
- [ ] Full-disk encryption enabled on coordinator machine
- [ ] SMB3 signing/encryption enabled on backup targets
- [ ] TLS cert rotation planned before expiry
- [ ] Regular backups of `C:\ArcVault\config.json` (contains credential key and admin token)

### Development / Testing

If running locally without proper DNS:
```powershell
# Allow self-signed cert on agents
$env:ARCVAULT_INSECURE_SKIP_VERIFY = "1"

# Or use IP-based SAN in certificate
coordinator init  # and specify your IP address
```

**⚠️ Never use `ARCVAULT_INSECURE_SKIP_VERIFY` in production.**

---

## Compliance Notes

- ✅ TLS 1.2+ (no weak protocols)
- ✅ AES-256 encryption for credentials
- ✅ HTTPS required for dashboard
- ✅ JWT tokens over encrypted connection
- ✅ WebSocket origin validation (prevents CSRF)
- ✅ Certificate pinning on agents (prevents MITM)

**Gaps:**
- ❌ No built-in database encryption (add full-disk encryption)
- ❌ No end-to-end encryption of backup data itself (ArcVault transports, does not encrypt backups)
- ❌ SMB encryption optional (configure on network level)

---

## Common Tasks

### Rotate TLS Certificate

```bash
coordinator rekey-cert
# Restart coordinator service
# Agents will re-register with new cert automatically
```

### Recover from Lost Credential Key

If `credential_key` in config.json is lost:
1. All stored credentials become unrecoverable
2. Create new credentials via dashboard
3. Re-authorize backup jobs

**Prevention:** Back up `config.json` in secure location.

### Inspect Certificate

```powershell
# View cert details
certutil -text C:\ArcVault\cert.pem | findstr /C:"Subject" /C:"Issuer" /C:"Validity"
```

### Agent-Coordinator TLS Troubleshooting

```powershell
# Agent logs show certificate validation errors
Get-Content C:\ArcVault-Agent\logs\arcvault-agent.log | Select-String "certificate"

# Test TLS connection manually
[System.Net.ServicePointManager]::ServerCertificateValidationCallback = {$true}
$response = Invoke-WebRequest -Uri "https://coordinator-ip:8443/api/health" -SkipCertificateCheck
```


# ArcVault v1.0.0 — IT Security Approval Document

**Document type:** Security Review  
**Version:** 1.0.0  
**Date:** 2026-06-11  
**Audience:** IT Security Team  

---

## 1. Overview

ArcVault is a fully self-hosted backup orchestrator. It has no cloud dependency and makes no call-home connections. All data, credentials, keys, and logs remain within the organisation's perimeter. The attack surface is limited to a single HTTPS port on the coordinator host. Every security control described in this document is enforced within the organisation's own infrastructure.

---

## 2. Transport Security

All connections to and from ArcVault use TLS. There is no plaintext fallback.

| Connection | Protocol | Notes |
|---|---|---|
| Browser → Coordinator | HTTPS (TLS) | Port :443; cert path set in config.json |
| Agent → Coordinator | WSS (TLS over WebSocket) | Same port and certificate |
| Coordinator → Spoke (federation) | WSS (TLS) | Certificate must be valid on the spoke host |

**Certificate configuration.** The TLS certificate and private key paths are specified in `config.json`. The operator is responsible for provisioning a valid certificate (self-signed, internal CA, or public CA). ArcVault does not auto-generate certificates.

**No plaintext fallback.** The coordinator does not serve HTTP. If TLS configuration is missing or invalid, the process will not start.

---

## 3. Authentication

### User authentication (JWT)

- On successful login (`POST /api/login`), the coordinator issues a signed JWT.
- Algorithm: **HS256**. Signing secret is stored in `config.json` (see Section 5).
- The JWT must be presented as a `Bearer` token in the `Authorization` header on every API request.
- Token expiry is configurable. The coordinator validates expiry server-side on every request.
- There is no refresh token mechanism; re-authentication is required after expiry.

### Agent authentication (per-agent tokens)

- Each agent is issued a unique 64-character hex token at registration.
- The token is transmitted once (during registration) and stored hashed in the `tokens` table using a one-way hash. The plaintext token is never stored.
- On every WebSocket connection attempt, the agent presents its token; the coordinator verifies it against the stored hash.
- Tokens are scoped to a single agent ID. A token from Agent A cannot authenticate Agent B.
- Tokens can be revoked via the dashboard at any time without restarting the coordinator.
- There are no shared secrets between agents.

---

## 4. Authorization (RBAC)

Three roles are defined. Role assignment is managed by an admin. The role is embedded in the JWT claim on login and validated server-side on every route.

| Role | Capabilities |
|---|---|
| `admin` | Full access: manage users, agents, jobs, tokens, alert rules, federation, system config |
| `operator` | Run jobs, view job history, view agents, view alert history — no write to config or users |
| `viewer` | Read-only: view jobs, agents, runs, alerts — no execution, no write operations |

**Server-side enforcement.** Every API route checks the role from the validated JWT before executing the handler. There is no client-side-only access control. A `viewer` JWT that is replayed against a write endpoint will be rejected by the server.

---

## 5. Credential Storage

| Credential | Storage method | Location |
|---|---|---|
| User passwords | bcrypt hash (`golang.org/x/crypto/bcrypt`) | `users` table in SQLite |
| Agent tokens | One-way hash | `tokens` table in SQLite |
| JWT signing secret | Plaintext in config file | `config.json` on coordinator host |
| SMTP credentials | Plaintext in config file | `config.json` on coordinator host |
| Slack/Teams webhook URLs | Plaintext in config file | `config.json` on coordinator host |

**No credentials in source code.** All secrets are runtime configuration; none are hardcoded or committed to source.

**config.json file permissions.** Restricting read access to `config.json` is the operator's responsibility (see Section 12). ArcVault does not enforce OS-level file permissions, but the signing secret and SMTP credentials are exposed if the file is world-readable.

---

## 6. Data at Rest

- All coordinator data is stored in a single SQLite file on the coordinator host.
- **ArcVault does not encrypt the SQLite file at rest.** OS-level disk encryption (BitLocker, FileVault, LUKS) is recommended and is the operator's responsibility.
- Data never leaves the coordinator host except via configured notification channels (see Section 7).
- Agent hosts do not store job data; they execute commands and stream output to the coordinator.

---

## 7. Notification Security

| Channel | Outbound method | Auth / verification |
|---|---|---|
| Generic webhook | HTTPS POST | Payload signed with HMAC-SHA256; signature sent in `X-ArcVault-Signature` header |
| Slack | HTTPS POST to incoming webhook URL | Webhook URL is the credential; treat as a secret |
| Microsoft Teams | HTTPS POST to Adaptive Card webhook URL | Webhook URL is the credential; treat as a secret |
| Email (SMTP) | SMTP/TLS | SMTP credentials stored in config.json |

**Webhook signing.** Generic webhook consumers can verify payload authenticity by computing `HMAC-SHA256(payload, shared_secret)` and comparing to the `X-ArcVault-Signature` header. The shared secret is configured per alert rule.

**Slack and Teams webhook URLs.** These URLs are bearer credentials. Any party with the URL can post to the channel. URLs must be treated as secrets and rotated if compromised.

**Retry behavior.** Failed webhook deliveries are retried up to 3 times with exponential backoff. Repeated failures are logged to `alert_history`.

---

## 8. Audit Trail

| Log type | Storage | Retention |
|---|---|---|
| Alert firings | `alert_history` table in SQLite | 30 days (auto-pruned nightly) |
| Job executions | `job_runs` table in SQLite | Indefinite (no auto-prune) |

Each `job_runs` record includes: job ID, agent ID, start time, end time, exit code, stdout/stderr output capture, and final status. There is no API endpoint to delete job run records. Alert history older than 30 days is pruned by the internal scheduler.

**Limitation.** Audit logs are stored in the same SQLite file as operational data. There is no separate, tamper-evident audit log or SIEM integration built in. If tamper-evident logging or SIEM forwarding is required, it must be implemented at the OS level (e.g., forwarding the SQLite file or coordinator process logs to a centralised log system).

---

## 9. Threat Model

| Threat | Likelihood | Mitigation in ArcVault | Residual risk |
|---|---|---|---|
| Unauthenticated API access | Low | JWT required on all routes except `/api/login` and `/health`; TLS prevents credential interception | `/health` exposes coordinator liveness (low sensitivity) |
| JWT theft / replay | Low–Medium | Short configurable expiry; HTTPS-only; no token storage in localStorage recommended (implementation detail) | No token revocation for user JWTs; expiry is the only invalidation mechanism |
| Agent token theft | Low | Tokens stored hashed; scoped per agent; revocable immediately via dashboard | Compromised token usable until revoked; operator must act promptly |
| MITM on agent connection | Low | All agent connections are WSS (TLS); no plaintext fallback | Risk increases if self-signed certs are used without pinning |
| SQLite file exfiltration | Medium | File access requires OS-level access to coordinator host; not exposed via any API | No at-rest encryption built in; mitigated by OS disk encryption |
| Notification endpoint abuse | Low | Webhook payloads are HMAC-signed; consumer can verify authenticity | Slack/Teams URLs are unprotected bearer credentials if leaked |
| Self-update supply chain | Medium | Update delivered over authenticated WSS channel; one-version rollback available | No cryptographic signature verification on update packages in v1.0.0 (see Section 11) |
| Brute-force login | Medium | No built-in rate limiting or account lockout in v1.0.0 | Mitigate at network layer (WAF, firewall, fail2ban on coordinator host) |

---

## 10. Network Exposure

The coordinator exposes **one port: :443 (HTTPS)**. No other ports are opened by ArcVault.

Agents make **outbound connections only** to the coordinator's WSS endpoint. Agents do not open any listening port. No inbound firewall rules are required on agent hosts for ArcVault traffic.

Federation spokes connect outbound to the root coordinator over WSS. The root coordinator must be reachable from spokes on :443.

See `APPENDIX.md` for the full port matrix.

---

## 11. Self-Update Security

Update packages are distributed from the coordinator to agents over the existing authenticated WSS channel. The channel is encrypted (TLS) and the agent must present a valid token to receive updates.

**Rollback.** One prior version is retained after each update. Rollback can be triggered from the dashboard without re-uploading a package.

**Known limitation — v1.0.0.** Update packages are not cryptographically signed in this release. An operator who can write a malicious binary to the update source location could deliver it to agents. Mitigations:

- Restrict filesystem access to the coordinator host (standard OS hardening)
- Verify binary checksums manually before staging an update
- Post-v1.0.0: package signing is on the roadmap

---

## 12. Recommendations

The following operational steps are recommended before production deployment:

1. **Enable OS-level disk encryption** on the coordinator host (BitLocker on Windows, FileVault on macOS, LUKS on Linux) to protect the SQLite file at rest.
2. **Restrict `config.json` permissions** to the service account that runs the coordinator (e.g., `chmod 600` on Linux/macOS; restrict ACL on Windows). The JWT signing secret, SMTP credentials, and notification webhook URLs are stored here in plaintext.
3. **Use a certificate from an internal CA or public CA** rather than a self-signed certificate where possible. If a self-signed certificate is used, distribute the CA to browsers and agent hosts so TLS validation is enforced.
4. **Firewall the coordinator port** (:443) to known source ranges — admin workstations and agent subnets only. Do not expose the coordinator to the public internet unless explicitly required.
5. **Rotate agent tokens periodically** and immediately on suspected compromise. Token rotation is a dashboard operation; no coordinator restart required.
6. **Implement login rate limiting** at the network layer (WAF, reverse proxy, or firewall rule) until built-in rate limiting is available in a future release.
7. **Treat Slack and Teams webhook URLs as secrets.** Store them in a secrets manager or password vault and rotate them if exposure is suspected.
8. **Forward coordinator host system logs to your SIEM** if tamper-evident audit logging or centralised log retention beyond 30 days is required.

---

## Appendix Reference

See `APPENDIX.md` in this directory for:
- Full RBAC permission table (per-route breakdown)
- Port matrix (coordinator, agent, federation)

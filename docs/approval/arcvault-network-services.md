# ArcVault v1.0.0 — Network Services Review

**To:** Network Services
**From:** IT Infrastructure
**Date:** June 11, 2026
**Subject:** Firewall rules and network requirements for ArcVault deployment

---

## Overview

ArcVault consists of a central coordinator process and lightweight agents deployed on managed machines. Agents initiate outbound WebSocket connections to the coordinator — no inbound connectivity is required on agent machines. The coordinator exposes a single HTTPS port used by both browser-based dashboard clients and agents. Outbound traffic from the coordinator is limited to notification channels (email and webhooks); this is optional and can be restricted without affecting core functionality.

---

## Inbound Traffic — Coordinator Host

| Port | Protocol | Source | Purpose |
|---|---|---|---|
| 443 | HTTPS | Browser clients (internal) | Dashboard access |
| 443 | WSS (WebSocket over TLS) | Agent machines (internal) | Agent registration, job control, telemetry |

No other inbound ports are required.

---

## Outbound Traffic — Coordinator Host

| Port | Protocol | Destination | Purpose | Required |
|---|---|---|---|---|
| 587 | SMTP/TLS | Internal or external mail relay | Alert email notifications | Optional |
| 443 | HTTPS | api.slack.com | Slack alert notifications | Optional |
| 443 | HTTPS | org-name.webhook.office.com | Microsoft Teams alert notifications | Optional |

All outbound notification channels are optional. Core backup orchestration functions without any outbound access from the coordinator.

---

## WebSocket Details

Agents connect to the coordinator over a persistent WSS connection on port 443. Key characteristics:

- Connection is always agent-initiated (outbound from the agent host)
- Agents authenticate using a per-agent bearer token on connection
- The connection is kept alive for the duration of the agent's session; reconnect logic handles transient network interruptions
- No inbound firewall rules are required on agent machines
- Agents do not listen on any port

---

## TLS Configuration

- The coordinator terminates TLS on port 443
- Certificates may be self-signed or issued by an internal CA; the certificate path is configurable in `config.json`
- Agents and browsers connect over TLS; agents can be configured to trust a specific CA certificate or to skip verification in controlled environments (not recommended for production)
- No plaintext fallback — TLS is mandatory

---

## Federation HA Networking

When multiple coordinators are deployed in federation mode:

- Spoke coordinators connect to the root coordinator over HTTPS on port 443 (same port as agents and browsers)
- State sync traffic travels over this same connection — no additional port is required
- All coordinator-to-coordinator communication is TLS-encrypted
- Internal DNS resolution or explicit IP routing between coordinator nodes is required; each spoke must be able to reach the root coordinator hostname

---

## Recommended Firewall Rules

| Direction | Source | Destination | Port | Protocol | Purpose |
|---|---|---|---|---|---|
| Inbound | Internal network | Coordinator host | 443 | HTTPS/WSS | Dashboard and agent traffic |
| Outbound | Coordinator host | Mail relay | 587 | SMTP/TLS | Alert emails (if enabled) |
| Outbound | Coordinator host | Internet | 443 | HTTPS | Slack/Teams webhooks (if enabled) |
| Outbound | Agent hosts | Coordinator host | 443 | WSS | Agent-to-coordinator connection |
| Inbound (block) | Any | Agent hosts | Any | Any | Not required — agents open no listening ports |

---

## DNS Requirements

- The coordinator must be reachable by a consistent hostname from all agent machines and browser clients
- An internal DNS A record pointing to the coordinator host is recommended
- If agents are configured with an IP address rather than a hostname, TLS certificate CN/SAN validation must account for this
- Federation spokes require DNS resolution of the root coordinator hostname

---

## Cloud Egress

No cloud egress is required for core functionality. ArcVault is entirely self-contained. Outbound connections to Slack (443) and Teams (443) are needed only if those alert channels are configured. SMTP outbound (587) is needed only if email alerting is enabled. All three can be blocked without affecting backup operations.

---

See APPENDIX.md for the full port matrix and configuration reference.

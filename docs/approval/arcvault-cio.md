# ArcVault v1.0.0 — Deployment Approval Request

**To:** Chief Information Officer
**From:** IT Infrastructure
**Date:** June 11, 2026
**Subject:** Approval to deploy ArcVault backup orchestration platform

---

## The Problem

The current backup tool, RoboBackup, is Windows-only and provides no central visibility into backup state across the environment. Operators must log into each machine individually to verify job outcomes. There is no alerting on failure, no audit trail, and no way to confirm backup coverage without manual checks. As the environment has grown to include macOS and Linux systems, RoboBackup no longer covers the full estate.

---

## What ArcVault Delivers

ArcVault is a self-hosted backup orchestrator built in-house. It runs on any platform, provides a web dashboard with real-time job status, and sends alerts when jobs fail or are missed. There is no SaaS subscription, no cloud dependency, and no data leaves the organisation's infrastructure.

| Capability | Detail |
|---|---|
| Cross-platform agents | Windows, macOS, Linux — managed from one dashboard |
| Web dashboard | Real-time job status, history, and metrics via browser |
| Alerting | Failure, duration, and missed-schedule notifications via email, Slack, Teams, or webhook |
| Access control | Three roles: admin, operator, viewer |
| High availability | Federation mode: multiple coordinators with automatic failover |
| Self-update | Coordinator and agents update remotely — no per-machine maintenance |

---

## Risk Posture

ArcVault is entirely self-hosted. No backup data, credentials, or telemetry is transmitted to external services. All internal communication uses TLS. Access requires JWT authentication. Role-based access control limits what each user can see and do.

The system has no external dependencies for core functionality. Optional alerting channels (Slack, Teams, email) use standard outbound HTTPS and SMTP — these can be disabled if not required.

Detailed technical review documentation has been prepared for IT Architecture, IT Security, Network Services, and Cloud Operations. See APPENDIX.md for the full technical reference.

---

## Current Status

ArcVault v1.0.0 is complete, built, and tested. It is ready for production deployment.

---

## Approval Request

**Approval is requested to proceed with production deployment of ArcVault v1.0.0.**

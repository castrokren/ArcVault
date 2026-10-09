# ArcVault

ArcVault is a backup management system for Windows that lets you schedule, run, and monitor file backups across multiple machines from a single web dashboard.

It wraps Windows' built-in `robocopy` command in a simple client-server interface — no third-party backup software, no cloud dependency, no per-seat licensing. Just fast, reliable file copies to wherever you point it.

---

## What it does

You install a small **backup agent** on each machine you want to back up. The agent runs as a Windows Service in the background and waits for instructions.

You open the **controller dashboard** in your browser. From there you can:

- Point any agent at a source folder and a destination (local path or network share)
- Start a backup job with one click
- Watch the backup progress in real time as logs stream live to your browser
- See whether jobs succeeded or failed, and why
- Manage multiple machines from the same screen

![ArcVault dashboard showing agent status and live log output](assets/screenshot.png)

---

## How it works

```
Your browser
    │
    ▼
Controller dashboard (Streamlit web app)
    │  sends backup jobs over HTTPS
    ▼
Backup agent (runs as a Windows Service on each machine)
    │  executes
    ▼
robocopy — copies files to the destination
```

The agent handles everything locally: it copies the files, manages network share credentials, writes detailed logs, and reports status back to the dashboard. If the agent restarts mid-job, it records the interruption and you can resubmit.

---

## Key features

- **One dashboard, many machines** — add as many agents as you need and switch between them in the sidebar
- **Live log streaming** — watch robocopy output line by line as the backup runs
- **Network share support** — back up to or from UNC paths (`\\server\share`) with optional credentials, with automatic retry on connection failures
- **Job history** — all jobs are persisted to a local SQLite database so history survives restarts
- **Conflict detection** — the agent rejects a new job if it overlaps with a source path already being backed up
- **Secure by default** — HTTPS with self-signed certificates and JWT-based authentication between controller and agents
- **No cloud required** — everything runs on your own infrastructure

---

## Tech stack

| Layer | Technology |
|-------|-----------|
| Backup engine | Windows `robocopy` |
| Agent API | Python / FastAPI |
| Dashboard | Python / Streamlit |
| Authentication | RS256 JWT (asymmetric keys) |
| Job storage | SQLite |
| Windows Service | pywin32 |
| Setup GUI | PyQt5 |

---

## Getting started

**Requirements:** Windows 10 or Server 2019+, Python 3.9+

```powershell
# 1. Start a backup agent (on the machine you want to back up)
cd apps\backup-agent
pip install -e .
$env:AGENT_TOKEN = "dev-token"
python agent\run_agent.py
# Agent is now listening at http://localhost:8443

# 2. Start the controller dashboard (any machine with network access to the agent)
cd apps\backup-control
pip install -e .
$env:AGENT_TOKEN = "dev-token"
streamlit run controller\app.py
# Open http://localhost:8501 in your browser
```

For production deployment — Windows Service installation, SSL certificates, JWT key generation, and multi-machine setup — see the [Deployment Guide](docs/DEPLOYMENT.md).

---

## Documentation

| | |
|-|-|
| [Deployment Guide](docs/DEPLOYMENT.md) | How to install and configure ArcVault in production |
| [API Reference](docs/AGENT_API.md) | Full REST API documentation for the backup agent |
| [Architecture](docs/ARCHITECTURE.md) | How the components fit together under the hood |
| [Environment Variables](docs/ENVIRONMENT_CONFIGURATION.md) | Complete configuration reference |

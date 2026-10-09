# ArcVault — Architecture

This document describes the system design, component responsibilities, data flow, concurrency model, security boundaries, and key design decisions in ArcVault.

---

## Table of Contents

- [System Overview](#system-overview)
- [Component Map](#component-map)
- [Component Details](#component-details)
  - [Backup Agent](#backup-agent)
  - [Backup Controller](#backup-controller)
  - [Backup Engine](#backup-engine)
  - [Desktop GUI](#desktop-gui)
- [Data Flow](#data-flow)
  - [Job Creation and Execution](#job-creation-and-execution)
  - [Live Log Streaming](#live-log-streaming)
  - [Agent Discovery and Health Checks](#agent-discovery-and-health-checks)
- [Concurrency Model](#concurrency-model)
- [Persistence](#persistence)
  - [SQLite Job Database](#sqlite-job-database)
  - [Log Files](#log-files)
- [Security Architecture](#security-architecture)
  - [Authentication Flow](#authentication-flow)
  - [Credential Encryption](#credential-encryption)
  - [TLS](#tls)
- [Crash Recovery](#crash-recovery)
- [Technology Stack](#technology-stack)
- [Key Design Decisions](#key-design-decisions)

---

## System Overview

ArcVault uses a hub-and-spoke model. One **controller** (the hub) manages many **agents** (the spokes). Each agent runs on a machine that needs to be backed up and performs the actual work; the controller provides a single dashboard for operators to configure jobs and observe results.

```
                        ┌──────────────────────────────────┐
                        │     Controller (Streamlit)        │
                        │     port 8501                     │
                        │                                   │
                        │  - job configuration UI           │
                        │  - agent health monitoring        │
                        │  - live log display               │
                        │  - JWT minting (RS256)            │
                        └────────────┬─────────────────────┘
                                     │ HTTPS + JWT / X-API-Key
                     ┌───────────────┼───────────────┐
                     ▼               ▼               ▼
             ┌──────────────┐ ┌──────────────┐ ┌──────────────┐
             │  Agent A     │ │  Agent B     │ │  Agent C     │
             │  port 8443   │ │  port 8443   │ │  port 8443   │
             └──────┬───────┘ └──────┬───────┘ └──────┬───────┘
                    │                │                │
                    ▼                ▼                ▼
               robocopy          robocopy          robocopy
               net use           net use           net use
               SQLite            SQLite            SQLite
```

The controller and agents are **decoupled** — agents operate independently, queue their own jobs, and persist state locally. The controller is purely a UI/orchestration layer; losing it does not interrupt running backups.

---

## Component Map

```
ArcVault/
├── apps/backup-agent/
│   ├── agent/
│   │   ├── app.py                    ← FastAPI application (HTTP layer)
│   │   ├── db.py                     ← SQLite persistence
│   │   ├── log_handler.py            ← Per-job rotating file logs
│   │   ├── run_agent.py              ← uvicorn bootstrap
│   │   ├── win_service.py            ← Windows Service wrapper
│   │   └── engine/
│   │       ├── config.py             ← Pydantic models
│   │       ├── config_loader.py      ← JSON/YAML + env-var loading
│   │       ├── credential_encryption.py  ← Fernet encryption
│   │       ├── robocopy.py           ← argv builder + exit code classifier
│   │       └── runner.py             ← Backup execution orchestrator
│   ├── gui/
│   │   └── main_window.py            ← PyQt5 desktop GUI
│   └── arcvault_unified.py           ← Unified entry point
└── apps/backup-control/
    └── controller/
        └── app.py                    ← Streamlit dashboard (single file)
```

---

## Component Details

### Backup Agent

**File:** `apps/backup-agent/agent/app.py`  
**Runtime:** FastAPI + uvicorn (ASGI), single worker process, multithreaded

The agent is the core of ArcVault. It is a thin HTTP layer over the backup engine:

- Validates incoming requests using Pydantic (via `BackupConfig`)
- Maintains an in-memory job cache backed by SQLite
- Dispatches each job to a background thread via FastAPI `BackgroundTasks`
- Controls concurrency with a `threading.Semaphore`
- Streams log output to clients over SSE while simultaneously writing to disk
- Exposes `/health` and `/info` for monitoring and auto-discovery

The agent is intentionally single-process/single-worker. Horizontal scaling is achieved by deploying multiple agents on different machines rather than scaling a single agent instance.

**Entry points:**

| Mode | Command |
|------|---------|
| Development CLI | `python agent/run_agent.py` |
| Windows Service | `python agent/win_service.py start` |
| Unified exe | `arcvault_unified.py` (GUI dispatch) |

---

### Backup Controller

**File:** `apps/backup-control/controller/app.py`  
**Runtime:** Streamlit (single-file application)

The controller is a stateless UI layer — it holds no authoritative job state of its own. All state lives in the agents. The controller:

- Reads the agent registry from `agents.json` (a simple `{name: url}` dict persisted to disk)
- Renders a job-configuration form and POSTs it to the selected agent
- Polls agent `/health` on every render cycle (with a manual "Refresh" button)
- Calls `/info` to detect URL changes and update `agents.json` automatically
- Follows SSE log streams and renders them in a text area
- Runs a 30-second reconciliation loop that fetches `/jobs` from all agents and surfaces state-change notifications

The controller uses a **sliding-refresh JWT cache**: a token is minted once and reused until less than 20% of its TTL remains, at which point a new token is minted transparently before the next request.

---

### Backup Engine

**Files:** `apps/backup-agent/agent/engine/`

A pure Python library with no HTTP concerns. The agent's `app.py` calls into it.

#### `config.py` — Data Models

Three Pydantic v2 models form the schema for the entire system:

```
BackupConfig
├── sources: List[str]          # validated non-empty, stripped
├── destination: str            # validated exists + writable (unless skipped)
├── options: BackupOptions      # robocopy flags with sane defaults
├── source_credentials: Credentials | None
└── destination_credentials: Credentials | None

BackupOptions
└── include_empty_dirs, verbose, report_extras,
    retries, wait_seconds, threads, copy_symlinks,
    remove_system_hidden

Credentials
├── username: str
├── password: str               # transparently encrypted via __getattribute__
└── domain: str | None
```

`Credentials` uses a `model_validator` to encrypt the password on construction and overrides `__getattribute__` to decrypt on access, making encryption transparent to all callers.

#### `config_loader.py` — Configuration Loading

Supports loading `BackupConfig` from:
- A Python dict (used by the HTTP endpoint)
- A JSON or YAML file path
- Environment variable overrides (`BACKUP_*`, double-underscore for nesting)

Example env override: `BACKUP_OPTIONS__THREADS=16` overrides `options.threads`.

#### `robocopy.py` — Command Builder

Translates a `BackupConfig` into one `robocopy` argv list per source:

```python
["robocopy", src, destination, "/E", "/V", "/X", "/R:3", "/W:10", "/MT:16", "/A-:SH"]
```

Also classifies robocopy exit codes into `("succeeded" | "failed", reason)` tuples.

#### `runner.py` — Execution Orchestrator

`run_backup(cfg, emit)` is the main entry point. It:

1. Builds robocopy commands from the config
2. Creates a timestamped log file in `{destination}/#backup_logs/`
3. Checks destination disk space — aborts if < 1 GB free
4. Verifies all source paths are accessible
5. Connects UNC shares with `net use` (exponential backoff: 1s → 2s → 4s, up to 3 attempts)
6. Spawns each robocopy subprocess and pipes stdout line-by-line through the `emit` callback
7. Disconnects UNC shares in a `finally` block regardless of outcome

The `emit` callback is the bridge between the engine and the HTTP layer — the agent passes `lambda s: _enqueue_log(job_id, s)` so every engine log line goes to both disk and the SSE stream.

---

### Desktop GUI

**Files:** `apps/backup-agent/gui/main_window.py`, `arcvault_unified.py`

A PyQt5 application (`ArcVaultMainWindow`) with four tabs used during initial setup and maintenance:

| Tab | Purpose |
|-----|---------|
| ⚙️ Service | Install, start, stop, and uninstall the Windows Service; configure host/port |
| 🔐 SSL Certificates | Generate a self-signed RSA certificate (server.crt, server.key, ca-bundle.crt) |
| 🔑 JWT Keys | Generate the RSA key pair for RS256 auth (controller_private.pem, controller_public.pem) |
| ℹ️ About | Version info |

`arcvault_unified.py` dispatches based on `sys.argv`:

```
arcvault.exe                 → launch PyQt5 GUI
arcvault.exe install         → install Windows Service
arcvault.exe --gen-cert      → CLI certificate generation (headless)
arcvault.exe --gen-jwt       → CLI JWT key generation (headless)
```

This file is the PyInstaller entry point — the entire agent stack bundles into a single `arcvault.exe`.

---

## Data Flow

### Job Creation and Execution

```
Controller UI
  │  POST /jobs  {sources, destination, options, credentials}
  ▼
app.py: create_job()
  ├── _require_token()           # authenticate
  ├── _check_source_conflicts()  # scan running/queued jobs for path overlap
  ├── db.save_job()              # persist initial state (queued) to SQLite
  ├── log_buffers[job_id] = Queue(maxsize=1000)  # init SSE buffer
  └── background.add_task(_run_backup, job_id, cfg)
        │
        ▼ (background thread)
  _run_backup()
  ├── job_semaphore.acquire()    # block if at MAX_CONCURRENT_JOBS
  ├── db: state → running
  ├── load_backup_config()       # validate config + apply env overrides
  ├── cfg.validate_destination_accessible()  # pre-flight check
  └── engine_run_backup(cfg, emit)
        ├── disk_space check
        ├── source path check
        ├── net use connect (with retry)
        ├── for each source:
        │     robocopy subprocess → line-by-line emit()
        └── net use disconnect
  ├── db: state → succeeded | failed
  ├── log_buffers[job_id].put(None)  # SSE sentinel → close stream
  ├── log_handler.close_job_logger()
  └── job_semaphore.release()
```

### Live Log Streaming

```
app.py: job_logs()  →  StreamingResponse(_iter_logs(job_id))
                                │
                                │  SSE: "data: <line>\n\n"
                                ▼
                          Controller UI
                          (Follow logs button)
```

The `_iter_logs` generator reads from the in-memory `Queue` until it receives the `None` sentinel. The queue is written to by `_enqueue_log`, which is called from every `emit()` invocation in the engine. Log lines are also written to disk first, so the SSE stream is best-effort (near capacity, oldest lines are evicted) while disk logs are authoritative.

### Agent Discovery and Health Checks

```
Controller startup
  └── _load_agents_from_disk() → agents.json → {name: url}

Every render cycle
  └── for each agent:
        GET {url}/health (timeout=5s)
        → 🟢 if r.ok, 🔴 otherwise

On "Refresh" click
  └── for each agent:
        GET {url}/info (timeout=2s)
        → if public_url changed: update agents.json

Every 30 seconds (reconciliation)
  └── for each agent:
        GET {url}/jobs
        → compare state vs session_state cache
        → surface notifications for state transitions
```

---

## Concurrency Model

The agent uses Python threads (not async) for backup execution, because robocopy is a blocking subprocess and the work is I/O-bound on the network/disk, not CPU-bound.

```
Main thread:  uvicorn ASGI event loop  (handles HTTP requests)
              │
              └── BackgroundTasks.add_task(_run_backup, ...)
                        │
                        ▼  (new thread per job, up to MAX_CONCURRENT_JOBS)
                  Thread pool (managed by threading.Semaphore)
                  ├── Job thread 1: robocopy subprocess
                  ├── Job thread 2: robocopy subprocess
                  └── Job thread 3: robocopy subprocess (default max)
```

**Semaphore behavior:**  
- `job_semaphore.acquire(blocking=False)` is tried first. If it fails (all slots full), the job is marked `queued` and `job_semaphore.acquire()` blocks the thread until a slot opens. This means the background thread exists but is parked — it does not consume CPU.

**Cache locking:**  
- `_jobs_cache` is a plain dict protected by `_jobs_cache_lock` (`threading.Lock`). All reads and writes go through `_get_job_from_cache`, `_update_job_cache`, and `_get_all_jobs`, which acquire the lock.

---

## Persistence

### SQLite Job Database

**Default path:** `C:/ProgramData/ArcVault/jobs.db`  
**Override:** `ARCVAULT_DB_PATH` environment variable

```sql
CREATE TABLE jobs (
    id          TEXT PRIMARY KEY,
    state       TEXT NOT NULL,        -- queued|running|succeeded|failed
    progress    INTEGER DEFAULT 0,
    error       TEXT,
    sources     TEXT,                 -- JSON array
    config      TEXT,                 -- JSON full config dict (for recovery)
    created_at  REAL NOT NULL,        -- Unix timestamp
    started_at  REAL,
    finished_at REAL,
    updated_at  REAL NOT NULL
);

CREATE INDEX idx_jobs_state      ON jobs(state);
CREATE INDEX idx_jobs_created_at ON jobs(created_at DESC);
```

The agent maintains an in-memory `_jobs_cache` dict for fast access and writes through to SQLite on every state change. On startup, the full job history is lazily loaded from SQLite on first access.

### Log Files

Each job writes to a rotating log file:

```
{destination}/
└── #backup_logs/
    ├── backup_20261009_143200.log     (active job)
    └── backup_20261008_091500.log     (completed job)
```

If no destination is known (config validation failed before a destination was set), logs fall back to:

```
C:/ProgramData/ArcVault/logs/<job_id>.log
```

Log rotation: 10 MB max per file, 3 backup files retained (`RotatingFileHandler`).

---

## Security Architecture

### Authentication Flow

```
            Controller                        Agent
               │                               │
               │  1. Read private key from     │
               │     CTRL_JWT_PRIVATE_KEY_PATH │
               │                               │
               │  2. Mint RS256 JWT            │
               │     iss: backup-controller    │
               │     aud: backup-agent         │
               │     exp: now + 15 min         │
               │     jti: random 8 bytes       │
               │                               │
               │  3. X-API-Key: <jwt>          │
               │ ────────────────────────────► │
               │                               │  4. Detect JWT (3 dots)
               │                               │  5. Load public key from
               │                               │     AGENT_JWT_PUBLIC_KEY_PATH
               │                               │  6. Verify RS256 signature
               │                               │  7. Validate iss, aud, exp
               │                               │  8. Accept or 401
```

**Token caching (controller side):** The minted JWT is cached in `_cached_token`. Before each request the controller checks the remaining TTL; if > 20% remains the cached token is reused. This avoids a key-read + RSA sign on every HTTP call.

**Fallback chain (agent side):**
1. If `AGENT_REQUIRE_JWT=true` → JWT only, no fallback
2. If JWT public key configured AND token looks like a JWT → validate as JWT
3. Otherwise → compare against `AGENT_TOKEN` (static token)

### Credential Encryption

UNC share credentials (`Credentials.password`) are encrypted with **Fernet** (AES-128-CBC + HMAC-SHA256) from the `cryptography` library.

**Key derivation:**
1. Primary: `ARCVAULT_ENCRYPTION_KEY` environment variable (base64-encoded Fernet key)
2. Fallback: PBKDF2HMAC-SHA256 derived from `socket.gethostname() + os.getlogin()` with a fixed salt

Encryption happens transparently in `Credentials._encrypt_password_on_input` (a `model_validator`). Decryption happens transparently in `Credentials.__getattribute__` whenever `.password` is accessed. No caller needs to handle encryption/decryption explicitly.

### TLS

TLS is terminated directly by uvicorn when `AGENT_CERTFILE` and `AGENT_KEYFILE` are set. The bundled GUI can generate self-signed certificates for internal deployments. For production, certificates generated by the GUI (or from a corporate CA) are placed at the configured paths.

The controller's SSL verification behavior is controlled by:
- `AGENT_VERIFY_SSL=false` — disable verification (for self-signed certs)
- `AGENT_CA_BUNDLE=/path/to/ca-bundle.crt` — provide a custom CA bundle

---

## Crash Recovery

On agent startup, `startup_event()` (FastAPI `on_event("startup")`) runs:

```python
active_jobs = db.load_active_jobs()   # state IN ('queued', 'running')
for job_id, job_data in active_jobs.items():
    job_data["state"] = "failed"
    job_data["error"] = "Service restarted - job interrupted"
    db.save_job(job_id, job_data)
```

This ensures no job is permanently stuck in `queued` or `running` after an unclean shutdown. The operator can see these failed jobs in the controller and re-submit them.

---

## Technology Stack

| Concern | Library / Tool | Version Constraint |
|---------|---------------|--------------------|
| Agent HTTP server | FastAPI | ≥ 0.110 |
| ASGI server | uvicorn[standard] | ≥ 0.22 |
| Controller UI | Streamlit | ≥ 1.31 |
| Data validation | Pydantic v2 | ≥ 2.5 |
| Authentication | PyJWT | ≥ 2.8 |
| Credential encryption | cryptography (Fernet) | ≥ 42.0 |
| Job persistence | sqlite3 (stdlib) | — |
| Windows Service | pywin32 | ≥ 306 |
| Desktop GUI | PyQt5 | ≥ 5.15 |
| HTTP client (controller) | requests | ≥ 2.31 |
| Packaging | PyInstaller | ≥ 6.0 |
| Backup engine | robocopy (Windows built-in) | — |
| Network share auth | net use (Windows built-in) | — |

---

## Key Design Decisions

**Why robocopy?**  
robocopy is battle-tested, ships with every version of Windows, handles VSS, retries, multi-threading, and UNC shares natively. Reimplementing that in Python would be fragile and slower.

**Why one agent per machine?**  
Agents are lightweight (< 30 MB RSS idle) and require no shared infrastructure. Each agent owns its SQLite database and log files locally, so a controller outage or network partition does not interrupt running backups.

**Why SQLite over an in-memory dict?**  
Early versions used a plain Python dict for job state. This meant all history was lost on restart and recovery was impossible. SQLite adds durability with zero deployment overhead — no separate database server required.

**Why threads instead of async for backup execution?**  
`subprocess.Popen` with line-by-line stdout reading is inherently blocking. While it could be wrapped in `asyncio.create_subprocess_exec`, threads are simpler, debuggable, and sufficient for the expected concurrency (≤ 10 simultaneous jobs). The semaphore gives a hard upper bound.

**Why SSE instead of WebSockets for log streaming?**  
SSE is unidirectional (server → client), simpler to implement over HTTP/1.1, and works through most reverse proxies without special configuration. Log streaming is read-only from the client's perspective, so WebSockets' bidirectionality is not needed.

**Why Streamlit for the controller?**  
A Streamlit app requires no separate frontend build step, no JavaScript, and minimal boilerplate. For an internal ops tool used by a small team, that tradeoff is acceptable. The controller is stateless by design, so Streamlit's re-render-on-interaction model fits naturally.

**Why dual-write logs (disk + queue)?**  
Disk logs are the authoritative record — they are never dropped and survive agent restarts. The in-memory queue is a convenience buffer for live streaming and is explicitly bounded (1 000 lines). If the queue fills up, the oldest lines are evicted, but the disk log remains complete.

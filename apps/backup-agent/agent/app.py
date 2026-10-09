import os
import time
import queue
import threading
import asyncio
import logging
from datetime import datetime, timezone
from uuid import uuid4
from typing import Dict, Iterator, Optional, List

from fastapi import (
    BackgroundTasks,
    FastAPI,
    Header,
    HTTPException,
)  # pyright: ignore[reportMissingImports]
from fastapi.responses import StreamingResponse  # pyright: ignore[reportMissingImports]
import socket
import jwt  # pyright: ignore[reportMissingImports]
from .engine.config import BackupConfig
from .engine.config_loader import load_backup_config, ConfigurationError
from .engine.runner import run_backup as engine_run_backup
from . import db
from . import log_handler


API_TOKEN_ENV_VAR_NAME = "AGENT_TOKEN"
JWT_PUBLIC_KEY_ENV = "AGENT_JWT_PUBLIC_KEY"
JWT_PUBLIC_KEY_PATH_ENV = "AGENT_JWT_PUBLIC_KEY_PATH"
JWT_ISSUER_ENV = "AGENT_JWT_ISSUER"
JWT_AUDIENCE_ENV = "AGENT_JWT_AUDIENCE"
JWT_LEEWAY_ENV = "AGENT_JWT_LEEWAY"
REQUIRE_JWT_ENV = "AGENT_REQUIRE_JWT"


def _get_api_token() -> str:
    token = os.getenv(API_TOKEN_ENV_VAR_NAME)
    if not token:
        raise RuntimeError(
            f"Security Error: {API_TOKEN_ENV_VAR_NAME} environment variable not set. "
            "Cannot run in production without authentication token."
        )
    return token


def _require_token(provided: Optional[str]) -> None:
    def _is_truthy_env(name: str) -> bool:
        v = (os.getenv(name, "") or "").strip().lower()
        return v in {"1", "true", "yes", "on"}

    # Enforce JWT-only when configured
    if _is_truthy_env(REQUIRE_JWT_ENV):
        if not provided or not _looks_like_jwt(provided):
            raise HTTPException(status_code=401, detail="JWT required")
        _verify_jwt_or_401(provided)
        return

    # If a JWT public key is configured and the token looks like a JWT, validate RS256
    if provided and _jwt_public_key_available() and _looks_like_jwt(provided):
        _verify_jwt_or_401(provided)
        return
    # Fallback: static token (dev mode or legacy)
    expected = _get_api_token()
    if provided != expected:
        raise HTTPException(status_code=401, detail="Unauthorized")


def _looks_like_jwt(token: str) -> bool:
    return token.count(".") == 2


def _jwt_public_key_available() -> bool:
    if os.getenv(JWT_PUBLIC_KEY_ENV):
        return True
    path = os.getenv(JWT_PUBLIC_KEY_PATH_ENV)
    return bool(path and os.path.exists(path))


def _load_jwt_public_key() -> Optional[str]:
    key_inline = os.getenv(JWT_PUBLIC_KEY_ENV)
    if key_inline:
        return key_inline
    path = os.getenv(JWT_PUBLIC_KEY_PATH_ENV)
    if path and os.path.exists(path):
        with open(path, "r", encoding="utf-8") as f:
            return f.read()
    return None


def _verify_jwt_or_401(token: str) -> None:
    public_key = _load_jwt_public_key()
    if not public_key:
        raise HTTPException(status_code=401, detail="JWT not configured")
    issuer = os.getenv(JWT_ISSUER_ENV, "backup-controller")
    audience = os.getenv(JWT_AUDIENCE_ENV, "backup-agent")
    try:
        leeway = int(os.getenv(JWT_LEEWAY_ENV, "60"))
    except ValueError:
        leeway = 60
    try:
        # Verify RS256 signature and claims
        jwt.decode(
            token,
            public_key,
            algorithms=["RS256"],
            options={"require": ["exp", "iat", "iss", "aud"]},
            issuer=issuer,
            audience=audience,
            leeway=leeway,
        )
    except jwt.PyJWTError as exc:  # type: ignore[attr-defined]
        raise HTTPException(status_code=401, detail=f"Invalid JWT: {exc}")


app = FastAPI(title="Backup Agent", version="0.1.0")


# Initialize database on startup
@app.on_event("startup")
async def startup_event():
    """Initialize database and recover any interrupted jobs."""
    db.init_db()
    
    # Recover jobs that were running when the service stopped
    active_jobs = db.load_active_jobs()
    for job_id, job_data in active_jobs.items():
        # Mark interrupted jobs as failed
        if job_data.get("state") in ["queued", "running"]:
            job_data["state"] = "failed"
            job_data["error"] = "Service restarted - job interrupted"
            job_data["finished_at"] = time.time()
            db.save_job(job_id, job_data)
            
            # Log the recovery action
            logger = log_handler.get_job_logger(
                job_id, 
                job_data.get("config", {}).get("destination")
            )
            logger.error("Job interrupted by service restart")
            log_handler.close_job_logger(job_id)


# Job state cache - loads from database on demand
# This provides fast access without hitting the database every time
_jobs_cache: Dict[str, Dict] = {}
_jobs_cache_lock = threading.Lock()


def _get_job_from_cache(job_id: str) -> Optional[Dict]:
    """Get job from cache or database."""
    with _jobs_cache_lock:
        if job_id in _jobs_cache:
            return _jobs_cache[job_id]
        # Load from database
        job_data = db.load_job(job_id)
        if job_data:
            _jobs_cache[job_id] = job_data
        return job_data


def _update_job_cache(job_id: str, updates: Dict) -> None:
    """Update job in cache and database."""
    with _jobs_cache_lock:
        if job_id not in _jobs_cache:
            _jobs_cache[job_id] = {}
        _jobs_cache[job_id].update(updates)
        db.save_job(job_id, _jobs_cache[job_id])


def _get_all_jobs() -> Dict[str, Dict]:
    """Get all jobs from cache/database."""
    with _jobs_cache_lock:
        # Load any missing jobs from database
        db_jobs = db.load_all_jobs()
        _jobs_cache.update(db_jobs)
        return _jobs_cache.copy()


# Job concurrency control
MAX_CONCURRENT_JOBS = int(os.getenv("AGENT_MAX_CONCURRENT_JOBS", "3"))
job_semaphore = threading.Semaphore(MAX_CONCURRENT_JOBS)


# Legacy log buffers for streaming - kept for compatibility but logs also go to disk
log_buffers: Dict[str, "queue.Queue[str | None]"] = {}


def _iter_logs(job_id: str) -> Iterator[bytes]:
    q: "queue.Queue[str | None]" = log_buffers[job_id]
    while True:
        line = q.get()
        if line is None:
            # Sentinel indicating job finished
            break
        # Server-Sent Events framing
        yield f"data: {line}\n\n".encode("utf-8")


@app.get("/health")
def health() -> Dict:
    # Public health endpoint without auth for simple reachability checks
    return {"status": "ok", "version": app.version}


def _enqueue_log(job_id: str, text: str, level: int = logging.INFO) -> None:
    """
    Log to both disk (persistent) and memory queue (for streaming).
    Disk logs are never dropped, memory queue keeps only recent logs for streaming.
    """
    # Always write to disk first (persistent, never dropped)
    log_handler.log_to_job(job_id, text, level)
    
    # Also put in queue for live streaming (if queue exists)
    if job_id in log_buffers:
        q = log_buffers[job_id]
        for line in text.splitlines() or [text]:
            # Keep only recent logs in memory queue for streaming
            if q.qsize() >= 950:  # Near the 1000 limit
                # Remove oldest entries to prevent memory growth
                for _ in range(100):
                    try:
                        q.get_nowait()
                    except queue.Empty:
                        break
            q.put(line)


def _run_backup(job_id: str, cfg_dict: Dict) -> None:
    """
    Execute a backup job with proper logging and state management.
    Logs are written to both disk (persistent) and memory queue (streaming).
    Job state is persisted to database.
    """
    # Get destination for log file location
    destination = cfg_dict.get("destination")
    
    # Initialize job logger with destination path (stores logs in destination/#backup_logs)
    job_logger = log_handler.get_job_logger(job_id, destination)
    
    # Wait for semaphore (this will queue jobs when at capacity)
    if not job_semaphore.acquire(blocking=False):
        _enqueue_log(job_id, f"Job {job_id} queued - waiting for available slot")
        _update_job_cache(job_id, {"state": "queued"})
        job_semaphore.acquire()  # Block until slot available
    
    try:
        # Check running job count
        all_jobs = _get_all_jobs()
        running_jobs = sum(1 for job in all_jobs.values() if job.get("state") == "running")
        _enqueue_log(job_id, f"Starting backup job {job_id} (slot {running_jobs + 1}/{MAX_CONCURRENT_JOBS})")
        
        # Update job state to running
        _update_job_cache(job_id, {
            "state": "running", 
            "started_at": time.time(), 
            "progress": 0
        })
        
        # Load and validate configuration
        try:
            cfg = load_backup_config(config_dict=cfg_dict, apply_env=True, validate=True)
        except ConfigurationError as e:
            _enqueue_log(job_id, f"Configuration validation failed: {e}", logging.ERROR)
            raise ValueError(f"Invalid configuration: {e}")
        
        # Pre-flight check: Verify destination is accessible
        accessible, error = cfg.validate_destination_accessible()
        if not accessible:
            _enqueue_log(job_id, f"Pre-flight check failed: {error}", logging.WARNING)
            _enqueue_log(job_id, "Attempting backup anyway (may fail if destination unavailable)", logging.WARNING)
        else:
            _enqueue_log(job_id, "Pre-flight check passed: Destination is accessible")
        
        # Run the backup
        final_state, reason = engine_run_backup(cfg, lambda s: _enqueue_log(job_id, s))
        
        # Update final state
        updates = {"state": final_state}
        if final_state != "succeeded":
            updates["error"] = reason
            _enqueue_log(job_id, f"Job failed: {reason}", logging.ERROR)
        else:
            _enqueue_log(job_id, "Job completed successfully")
        
        _update_job_cache(job_id, updates)
        
    except Exception as exc:  # pragma: no cover - simple MVP handling
        _update_job_cache(job_id, {
            "state": "failed",
            "error": str(exc)
        })
        _enqueue_log(job_id, f"ERROR: {exc}", logging.ERROR)
    finally:
        # Mark job as finished
        _update_job_cache(job_id, {"finished_at": time.time()})
        
        # Signal end of streaming logs
        if job_id in log_buffers:
            log_buffers[job_id].put(None)
        
        # Close the job logger to free resources
        log_handler.close_job_logger(job_id)
        
        # Release semaphore slot
        job_semaphore.release()


def _delayed_shutdown() -> None:
    # Give the HTTP response a moment to flush, then exit process
    time.sleep(0.25)
    os._exit(0)


@app.post("/shutdown")
def shutdown(
    x_api_key: Optional[str] = Header(default=None),
    authorization: Optional[str] = Header(default=None),
) -> Dict:
    token = x_api_key
    if (not token) and authorization and authorization.lower().startswith("bearer "):
        token = authorization.split(" ", 1)[1]
    _require_token(token)
    threading.Thread(target=_delayed_shutdown, daemon=True).start()
    return {"status": "shutting_down"}


def _check_source_conflicts(new_sources: List[str]) -> List[str]:
    """Check for overlapping source paths in running jobs"""
    conflicts = []
    all_jobs = _get_all_jobs()
    for job_id, job_meta in all_jobs.items():
        if job_meta.get("state") in ["queued", "running"]:
            existing_sources = job_meta.get("sources", [])
            for new_src in new_sources:
                for existing_src in existing_sources:
                    # Simple path overlap check
                    if (new_src.lower().startswith(existing_src.lower()) or 
                        existing_src.lower().startswith(new_src.lower())):
                        conflicts.append(f"Job {job_id[:8]} already using overlapping path: {existing_src}")
    return conflicts


@app.post("/jobs")
def create_job(
    cfg: Dict,
    background: BackgroundTasks,
    x_api_key: Optional[str] = Header(default=None),
    authorization: Optional[str] = Header(default=None),
) -> Dict:
    token = x_api_key
    if (not token) and authorization and authorization.lower().startswith("bearer "):
        token = authorization.split(" ", 1)[1]
    _require_token(token)
    
    # Check for source path conflicts
    sources = cfg.get("sources", [])
    conflicts = _check_source_conflicts(sources)
    if conflicts:
        raise HTTPException(status_code=409, detail={
            "error": "Source path conflicts detected", 
            "conflicts": conflicts
        })
    
    # Create job with persistent storage
    job_id = uuid4().hex
    job_data = {
        "state": "queued", 
        "progress": 0, 
        "error": None, 
        "sources": sources,
        "config": cfg,  # Store full config for recovery
        "created_at": time.time()
    }
    
    # Save to database
    db.save_job(job_id, job_data)
    
    # Update cache
    with _jobs_cache_lock:
        _jobs_cache[job_id] = job_data
    
    # Create log buffer for streaming
    log_buffers[job_id] = queue.Queue(maxsize=1000)
    
    # Start backup in background
    background.add_task(_run_backup, job_id, cfg)
    
    return {"job_id": job_id}


@app.get("/jobs")
def list_jobs(
    x_api_key: Optional[str] = Header(default=None),
    authorization: Optional[str] = Header(default=None),
) -> Dict:
    token = x_api_key
    if (not token) and authorization and authorization.lower().startswith("bearer "):
        token = authorization.split(" ", 1)[1]
    _require_token(token)
    
    # Get all jobs from database/cache
    all_jobs = _get_all_jobs()
    return {"jobs": [{"job_id": jid, **meta} for jid, meta in all_jobs.items()]}


@app.get("/jobs/{job_id}")
def job_status(
    job_id: str,
    x_api_key: Optional[str] = Header(default=None),
    authorization: Optional[str] = Header(default=None),
) -> Dict:
    token = x_api_key
    if (not token) and authorization and authorization.lower().startswith("bearer "):
        token = authorization.split(" ", 1)[1]
    _require_token(token)
    
    # Get job from cache/database
    job = _get_job_from_cache(job_id)
    if not job:
        raise HTTPException(status_code=404, detail="Not found")
    return {"job_id": job_id, **job}


@app.get("/jobs/{job_id}/logs")
def job_logs(
    job_id: str,
    x_api_key: Optional[str] = Header(default=None),
    authorization: Optional[str] = Header(default=None),
) -> StreamingResponse:
    token = x_api_key
    if (not token) and authorization and authorization.lower().startswith("bearer "):
        token = authorization.split(" ", 1)[1]
    _require_token(token)
    if job_id not in log_buffers:
        raise HTTPException(status_code=404, detail="Not found")
    return StreamingResponse(_iter_logs(job_id), media_type="text/event-stream")


def _detect_public_host() -> str:
    # Prefer DNS hostname; fallback to primary IPv4
    try:
        name = socket.getfqdn() or socket.gethostname()
        if name:
            return name
    except Exception:
        pass
    try:
        with socket.socket(socket.AF_INET, socket.SOCK_DGRAM) as s:
            s.connect(("8.8.8.8", 80))
            return s.getsockname()[0]
    except Exception:
        return "localhost"


@app.get("/info")
def info() -> Dict:
    # Public self-report (no secrets). Controller can use this to save the reachable URL
    host_override = os.getenv("AGENT_PUBLIC_HOST")
    url_override = os.getenv("AGENT_PUBLIC_URL")
    scheme = os.getenv("AGENT_PUBLIC_SCHEME", "https")
    host = host_override or _detect_public_host()
    public_url = url_override or f"{scheme}://{host}"
    
    # Get job counts from database
    job_counts = db.get_job_stats()
    
    return {
        "public_url": public_url,
        "hostname": socket.gethostname(),
        "version": app.version,
        "job_counts": job_counts,
        "max_concurrent_jobs": MAX_CONCURRENT_JOBS,
        "available_slots": job_semaphore._value,
    }


@app.post("/maintenance/cleanup")
def cleanup_old_data(
    days: int = 30,
    x_api_key: Optional[str] = Header(default=None),
    authorization: Optional[str] = Header(default=None),
) -> Dict:
    """
    Clean up old job records and log files.
    Removes completed/failed jobs and their logs older than specified days.
    """
    token = x_api_key
    if (not token) and authorization and authorization.lower().startswith("bearer "):
        token = authorization.split(" ", 1)[1]
    _require_token(token)
    
    # Clean up old jobs from database
    jobs_deleted = db.delete_old_jobs(days)
    
    # Clean up old log files
    logs_deleted = log_handler.cleanup_old_logs(days)
    
    # Clear deleted jobs from cache
    with _jobs_cache_lock:
        db_jobs = db.load_all_jobs()
        _jobs_cache.clear()
        _jobs_cache.update(db_jobs)
    
    return {
        "status": "cleanup_complete",
        "jobs_deleted": jobs_deleted,
        "log_files_deleted": logs_deleted,
        "days": days
    }


def is_port_available(port: int) -> bool:
    """Check if a port is available for binding."""
    with socket.socket(socket.AF_INET, socket.SOCK_STREAM) as s:
        try:
            s.bind(('0.0.0.0', port))
            return True
        except OSError:
            return False


# Optional: allow running directly for quick local testing
if __name__ == "__main__":  # pragma: no cover
    import uvicorn  # pyright: ignore[reportMissingImports]

    port = 8443
    if not is_port_available(port):
        raise RuntimeError(f"Port {port} already in use. Cannot start agent.")
    
    uvicorn.run("agent.app:app", host="0.0.0.0", port=port, reload=True)

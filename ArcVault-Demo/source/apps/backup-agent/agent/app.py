import os
import time
import queue
import threading
from datetime import datetime, timezone
from uuid import uuid4
from typing import Dict, Iterator, Optional

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
from .engine.runner import run_backup as engine_run_backup


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


# In-memory job state for MVP. Persist later under %ProgramData% if needed.
jobs: Dict[str, Dict] = {}
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


def _enqueue_log(q: "queue.Queue[str | None]", text: str) -> None:
    for line in text.splitlines() or [text]:
        q.put(line)


def _run_backup(job_id: str, cfg_dict: Dict) -> None:
    q = log_buffers[job_id]
    try:
        jobs[job_id].update(
            {"state": "running", "started_at": time.time(), "progress": 0}
        )
        _enqueue_log(q, f"Starting backup job {job_id}")
        cfg = BackupConfig.model_validate(cfg_dict)
        final_state, reason = engine_run_backup(cfg, lambda s: _enqueue_log(q, s))
        jobs[job_id]["state"] = final_state
        if final_state != "succeeded":
            jobs[job_id]["error"] = reason
    except Exception as exc:  # pragma: no cover - simple MVP handling
        jobs[job_id]["state"] = "failed"
        jobs[job_id]["error"] = str(exc)
        _enqueue_log(q, f"ERROR: {exc}")
    finally:
        jobs[job_id]["finished_at"] = time.time()
        q.put(None)


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
    job_id = uuid4().hex
    jobs[job_id] = {"state": "queued", "progress": 0, "error": None}
    log_buffers[job_id] = queue.Queue(maxsize=1000)
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
    return {"jobs": [{"job_id": jid, **meta} for jid, meta in jobs.items()]}


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
    job = jobs.get(job_id)
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
    return {
        "public_url": public_url,
        "hostname": socket.gethostname(),
        "version": app.version,
    }


# Optional: allow running directly for quick local testing
if __name__ == "__main__":  # pragma: no cover
    import uvicorn  # pyright: ignore[reportMissingImports]

    uvicorn.run("agent.app:app", host="0.0.0.0", port=8443, reload=True)

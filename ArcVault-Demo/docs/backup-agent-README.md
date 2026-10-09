# Backup Agent (FastAPI)

Lightweight per-machine agent that accepts backup jobs over HTTP and streams live logs.

## Quick start

```bash
python -m venv .venv
. .venv/Scripts/activate  # on Windows PowerShell: .venv\Scripts\Activate.ps1
pip install -e .
$env:AGENT_TOKEN = "dev-token"
uvicorn agent.app:app --host 0.0.0.0 --port 8443 --reload
```

Test:

```bash
# Create job
curl -X POST http://localhost:8443/jobs -H "X-API-Key: dev-token" -H "Content-Type: application/json" -d '{"sources":["C:/data"],"destination":"D:/backup"}'

# Stream logs
curl -N http://localhost:8443/jobs/<job_id>/logs -H "X-API-Key: dev-token"
```

## Notes
- SSE is used for log streaming (`text/event-stream`).
- Replace the `_simulate_backup` worker with real backup invocation.
- Configure as a Windows Service with NSSM.
- Auth in production:
  - Prefer RS256 JWTs. Set `AGENT_JWT_PUBLIC_KEY_PATH` (or `AGENT_JWT_PUBLIC_KEY`) on the agent and set `CTRL_JWT_PRIVATE_KEY_PATH` on the controller. The controller sends `X-API-Key: <JWT>` and the agent verifies.
  - To enforce JWT-only and disable static token fallback, set `AGENT_REQUIRE_JWT=true` on the agent.
  - If you must use a static token, set a long random `AGENT_TOKEN` on both sides and restrict network access.

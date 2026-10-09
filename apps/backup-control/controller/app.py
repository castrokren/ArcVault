import os
import time
import json
import socket
from urllib.parse import urlparse
from typing import Dict, List, Union, Optional, Tuple

import requests  # pyright: ignore[reportMissingModuleSource]
import jwt  # pyright: ignore[reportMissingImports]
from datetime import datetime, timedelta, timezone
import streamlit as st  # pyright: ignore[reportMissingImports]


st.set_page_config(page_title="ArcVault (Beta) Controller", layout="wide")

AGENTS_FILE = os.path.join(os.path.dirname(__file__), "agents.json")


def _load_agents_from_disk() -> Dict[str, str]:
    try:
        if os.path.exists(AGENTS_FILE):
            with open(AGENTS_FILE, "r", encoding="utf-8") as f:
                data = json.load(f)
                if isinstance(data, dict):
                    return {str(k): str(v) for k, v in data.items()}
    except Exception:
        pass
    return {"agent-local": os.getenv("AGENT_URL", "http://127.0.0.1:8443")}


def _save_agents_to_disk(agents: Dict[str, str]) -> None:
    try:
        with open(AGENTS_FILE, "w", encoding="utf-8") as f:
            json.dump(agents, f, indent=2)
    except Exception:
        pass


def _get_agents() -> Dict[str, str]:
    if "agents" not in st.session_state:
        st.session_state["agents"] = _load_agents_from_disk()
    return st.session_state["agents"]


def _api_headers() -> Dict[str, str]:
    # Prefer RS256 JWT if private key available; else fall back to static token
    pk = os.getenv("CTRL_JWT_PRIVATE_KEY")
    pk_path = os.getenv("CTRL_JWT_PRIVATE_KEY_PATH")
    if not pk and pk_path and os.path.exists(pk_path):
        with open(pk_path, "r", encoding="utf-8") as f:
            pk = f.read()
    if pk:
        return {"X-API-Key": _mint_access_token(pk)}
    token = os.getenv("AGENT_TOKEN")
    if not token:
        raise RuntimeError(
            "Security Error: AGENT_TOKEN environment variable not set. "
            "Cannot run in production without authentication token."
        )
    return {"X-API-Key": token}


_cached_token: Dict[str, str] = {"value": "", "exp": ""}


def _mint_access_token(private_key: str) -> str:
    # Sliding refresh: renew at 80% of TTL
    now = datetime.now(timezone.utc)
    if _cached_token.get("value") and _cached_token.get("exp"):
        try:
            exp_dt = datetime.fromisoformat(_cached_token["exp"])
            ttl = (exp_dt - now).total_seconds()
            if ttl > (15 * 60 * 0.2):  # more than 20% left
                return _cached_token["value"]
        except Exception:
            pass
    iss = os.getenv("CTRL_JWT_ISSUER", "backup-controller")
    aud = os.getenv("CTRL_JWT_AUDIENCE", "backup-agent")
    payload = {
        "iss": iss,
        "aud": aud,
        "iat": int(now.timestamp()),
        "exp": int((now + timedelta(minutes=15)).timestamp()),
        "jti": os.urandom(8).hex(),
        "sub": "controller",
    }
    token = jwt.encode(payload, private_key, algorithm="RS256")
    _cached_token["value"] = token
    _cached_token["exp"] = (now + timedelta(minutes=15)).isoformat()
    return token


def _display_host(url: str) -> str:
    parsed = urlparse(url)
    host = parsed.hostname or url
    if host in {"localhost", "127.0.0.1", "::1"}:
        try:
            return socket.gethostname()
        except Exception:
            return host
    return host


def _requests_verify() -> Union[str, bool]:
    v = (os.getenv("AGENT_VERIFY_SSL", "true") or "").lower()
    if v in ("0", "false", "no"):
        return False
    bundle = os.getenv("AGENT_CA_BUNDLE")
    if bundle and os.path.exists(bundle):
        return bundle
    return True


def _agent_request(
    base_url: str,
    method: str,
    path: str,
    *,
    timeout: int = 15,
    stream: bool = False,
    json: Optional[dict] = None,
    headers: Optional[Dict[str, str]] = None,
):
    # Compose URL and attempt with JWT/static header; on 401 retry with AGENT_TOKEN fallback
    url = f"{base_url.rstrip('/')}{path}"
    hdrs = dict(_api_headers())
    if headers:
        hdrs.update(headers)
    req = getattr(requests, method.lower())
    resp = req(url, timeout=timeout, stream=stream, json=json, headers=hdrs, verify=_requests_verify())
    if resp.status_code == 401:
        fallback = os.getenv("AGENT_TOKEN")
        if fallback:
            hdrs["X-API-Key"] = fallback
            resp = req(url, timeout=timeout, stream=stream, json=json, headers=hdrs, verify=_requests_verify())
    return resp


st.sidebar.header("Agents")
agents = _get_agents()
refresh_agents = st.sidebar.button("Refresh health/URLs")

def _agent_healthy(url: str) -> bool:
    try:
        r = requests.get(f"{url}/health", timeout=5, verify=_requests_verify())
        return r.ok
    except Exception:
        return False

# Health list
updates: Dict[str, str] = {}
for name, url in agents.items():
    status = "🟢" if _agent_healthy(url) else "🔴"
    if refresh_agents:
        try:
            r = requests.get(f"{url}/info", timeout=2, verify=_requests_verify())
            if r.ok and "public_url" in r.json():
                new_url = r.json()["public_url"].rstrip("/")
                if new_url and new_url != agents.get(name):
                    updates[name] = new_url
        except Exception:
            pass
    st.sidebar.write(f"{status} {_display_host(agents[name])} ({name})")

if updates:
    agents.update(updates)
    _save_agents_to_disk(agents)

# Add/remove agents
with st.sidebar.expander("Manage agents"):
    new_name = st.text_input("Name", key="agent_new_name")
    new_url = st.text_input("URL", key="agent_new_url", placeholder="http://host:8443")
    add_col1, add_col2 = st.columns(2)
    with add_col1:
        if st.button("Add agent"):
            if new_name and new_url:
                agents[new_name] = new_url
                _save_agents_to_disk(agents)
                st.success("Agent added")
            else:
                st.error("Name and URL are required")
    with add_col2:
        to_remove = st.selectbox("Remove", [""] + list(agents.keys()), key="agent_remove_select")
        if st.button("Remove selected") and to_remove:
            agents.pop(to_remove, None)
            _save_agents_to_disk(agents)
            st.success("Agent removed")

st.title("ArcVault (Beta) Control Panel")

agent_keys = list(agents.keys())
if not agent_keys:
    st.warning("No agents configured. Add one in 'Manage agents' on the left.")
    st.stop()

agent_name = st.selectbox("Agent", agent_keys, index=0)
if not agent_name:
    st.stop()
agent_url = agents.get(agent_name, "")
if not agent_url:
    st.warning("Selected agent has no URL configured.")
    st.stop()

st.subheader("Job Configuration")
sources_text = st.text_area("Sources (one per line)", value="")
destination = st.text_input("Destination path", value="")
copy_symlinks = st.checkbox("Copy symlinks (/SL)", value=False)
report_extras = st.checkbox("Report extras (/X)", value=True)
include_empty_dirs = st.checkbox("Include empty dirs (/E)", value=True)
verbose = st.checkbox("Verbose (/V)", value=True)
retries = st.number_input("Retries (/R:n)", min_value=0, max_value=1000, value=1)
wait_seconds = st.number_input("Wait seconds (/W:n)", min_value=0, max_value=60, value=5)
threads = st.number_input("Threads (/MT:n)", min_value=1, max_value=128, value=12)
remove_system_hidden = st.checkbox("Remove system + hidden (/A-:SH)", value=True)

with st.expander("UNC credentials (optional)"):
    src_user = st.text_input("Source username")
    src_pass = st.text_input("Source password", type="password")
    src_domain = st.text_input("Source domain")
    dst_user = st.text_input("Destination username")
    dst_pass = st.text_input("Destination password", type="password")
    dst_domain = st.text_input("Destination domain")

if st.button("Start Job"):
    cfg = {
        "sources": [s for s in (sources_text.splitlines()) if s.strip()],
        "destination": destination,
        "options": {
            "copy_symlinks": copy_symlinks,
            "report_extras": report_extras,
            "include_empty_dirs": include_empty_dirs,
            "verbose": verbose,
            "retries": int(retries),
            "wait_seconds": int(wait_seconds),
            "threads": int(threads),
            "remove_system_hidden": remove_system_hidden,
        },
    }
    if src_user or src_pass or src_domain:
        cfg["source_credentials"] = {"username": src_user, "password": src_pass, "domain": src_domain}
    if dst_user or dst_pass or dst_domain:
        cfg["destination_credentials"] = {"username": dst_user, "password": dst_pass, "domain": dst_domain}
    try:
        resp = _agent_request(agent_url, "post", "/jobs", json=cfg, timeout=15)
        resp.raise_for_status()
        st.session_state["current_job_id"] = resp.json()["job_id"]
        st.session_state["current_agent_url"] = agent_url
        st.success(f"Started job {st.session_state['current_job_id']}")
    except Exception as exc:
        st.error(f"Failed to start job: {exc}")

st.subheader("Agent Controls")
if st.button("Kill agent (shutdown)"):
    try:
        r = _agent_request(agent_url, "post", "/shutdown", timeout=5)
        if r.ok:
            st.success("Agent is shutting down")
        else:
            st.error(f"Shutdown failed: {r.status_code}")
    except Exception as exc:
        st.error(f"Shutdown error: {exc}")


job_id = st.session_state.get("current_job_id")
job_agent_url = st.session_state.get("current_agent_url", agent_url)

st.subheader("Live Logs")
placeholder = st.empty()
if st.button("Follow logs") and job_id:
    try:
        with _agent_request(job_agent_url, "get", f"/jobs/{job_id}/logs", stream=True, timeout=60) as r:
            lines: List[str] = []
            for raw in r.iter_lines():
                if not raw:
                    continue
                if raw.startswith(b"data:"):
                    lines.append(raw[5:].decode("utf-8"))
                    if len(lines) % 20 == 0:
                        placeholder.text("\n".join(lines[-400:]))
            placeholder.text("\n".join(lines[-400:]))
    except Exception as exc:
        st.error(f"Log stream error: {exc}")

st.subheader("Job Status")

# Auto-reconciliation - run every 30 seconds
if "last_reconciliation" not in st.session_state:
    st.session_state["last_reconciliation"] = 0

if time.time() - st.session_state["last_reconciliation"] > 30:
    st.session_state["last_reconciliation"] = time.time()
    try:
        # Reconcile job states with all agents
        for agent_name, agent_url in agents.items():
            try:
                remote_jobs = _agent_request(agent_url, "get", "/jobs", timeout=5).json()
                for remote_job in remote_jobs.get("jobs", []):
                    job_id = remote_job["job_id"]
                    # Update session state with latest from agent
                    if f"job_{job_id}_state" not in st.session_state:
                        st.session_state[f"job_{job_id}_state"] = remote_job
                    elif st.session_state[f"job_{job_id}_state"]["state"] != remote_job["state"]:
                        old_state = st.session_state[f"job_{job_id}_state"]["state"]
                        st.session_state[f"job_{job_id}_state"] = remote_job
                        
                        # Show notifications for state changes
                        if remote_job["state"] == "failed":
                            error_msg = remote_job.get("error", "Unknown error")
                            st.error(f"🚨 Job {job_id} FAILED: {error_msg}")
                        elif remote_job["state"] == "succeeded":
                            st.success(f"✅ Job {job_id} completed successfully")
                        elif remote_job["state"] == "running" and old_state == "queued":
                            st.info(f"🚀 Job {job_id} started running")
            except Exception:
                pass  # Agent unreachable, continue with others
    except Exception:
        pass  # Don't let reconciliation break the UI

if st.button("Refresh status") and job_id:
    try:
        s = _agent_request(job_agent_url, "get", f"/jobs/{job_id}", timeout=10).json()
        st.json(s)
        # Store in session state
        st.session_state[f"job_{job_id}_state"] = s
    except Exception as exc:
        st.error(f"Status error: {exc}")



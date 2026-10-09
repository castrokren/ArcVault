"""
Executable bootstrap for the Backup Agent.

Environment variables:
- AGENT_LISTEN_HOST (default: 0.0.0.0)
- AGENT_PORT (default: 8443)

JWT enforcement and key pinning are handled by the FastAPI app via:
- AGENT_REQUIRE_JWT=true
- AGENT_JWT_PUBLIC_KEY_PATH=... (or AGENT_JWT_PUBLIC_KEY)
- AGENT_JWT_ISSUER=backup-controller (override as needed)
- AGENT_JWT_AUDIENCE=backup-agent
"""

from __future__ import annotations

import os
import sys
from typing import Tuple


def _get_host_and_port() -> Tuple[str, int]:
    host = os.getenv("AGENT_LISTEN_HOST", os.getenv("LISTEN_HOST", "0.0.0.0"))
    port_str = os.getenv("AGENT_PORT", os.getenv("PORT", "8443"))
    try:
        port = int(port_str)
    except ValueError:
        port = 8443
    return host, port


def main() -> None:
    host, port = _get_host_and_port()
    # Import here to keep startup minimal and friendly to PyInstaller analysis
    from uvicorn import run as uvicorn_run  # type: ignore

    # Launch the FastAPI app
    uvicorn_run("agent.app:app", host=host, port=port)


if __name__ == "__main__":
    main()

import os
from typing import Optional


def _get_env(name: str, default: Optional[str] = None) -> Optional[str]:
    val = os.getenv(name)
    return val if val is not None and str(val).strip() != "" else default


def main() -> None:
    # Import lazily to keep startup clean for packaging
    import uvicorn  # type: ignore

    host = _get_env("AGENT_BIND_HOST", "0.0.0.0")
    port_str = _get_env("AGENT_BIND_PORT", "8443")
    try:
        port = int(port_str) if port_str else 8443
    except ValueError:
        port = 8443

    # Optional TLS: set AGENT_CERTFILE/AGENT_KEYFILE when terminating TLS here.
    certfile = _get_env("AGENT_CERTFILE")
    keyfile = _get_env("AGENT_KEYFILE")

    kwargs = {
        "host": host,
        "port": port,
        "log_level": "info",
        "factory": False,
        "workers": 1,
    }
    if certfile and keyfile:
        kwargs.update({"ssl_certfile": certfile, "ssl_keyfile": keyfile})

    # Run FastAPI app
    uvicorn.run("agent.app:app", **kwargs)


if __name__ == "__main__":
    main()



"""
Executable bootstrap for the Backup Agent.

Environment variables:
- AGENT_BIND_HOST (default: 0.0.0.0)
- AGENT_BIND_PORT (default: 8443)
- AGENT_CERTFILE (optional: path to SSL certificate file)
- AGENT_KEYFILE (optional: path to SSL private key file)

JWT enforcement and key pinning are handled by the FastAPI app via:
- AGENT_REQUIRE_JWT=true
- AGENT_JWT_PUBLIC_KEY_PATH=... (or AGENT_JWT_PUBLIC_KEY)
- AGENT_JWT_ISSUER=backup-controller (override as needed)
- AGENT_JWT_AUDIENCE=backup-agent
"""

from __future__ import annotations

import os
import socket
from typing import Optional


def _get_env(name: str, default: Optional[str] = None) -> Optional[str]:
    val = os.getenv(name)
    return val if val is not None and str(val).strip() != "" else default


def _is_port_available(port: int) -> bool:
    """Check if a port is available for binding."""
    with socket.socket(socket.AF_INET, socket.SOCK_STREAM) as s:
        try:
            s.bind(('0.0.0.0', port))
            return True
        except OSError:
            return False


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

    # Check port availability before starting
    if not _is_port_available(port):
        raise RuntimeError(f"Port {port} already in use. Cannot start agent.")

    # Run FastAPI app
    uvicorn.run("agent.app:app", **kwargs)


if __name__ == "__main__":
    main()



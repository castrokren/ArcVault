from __future__ import annotations

"""
Windows Service wrapper for the Backup Agent.

Build a one-file service EXE via PyInstaller, then install/start it with:
  RoboBackupAgentSvc.exe install
  RoboBackupAgentSvc.exe --startup auto start

Environment variables (read at service start):
- AGENT_LISTEN_HOST (default: 0.0.0.0)
- AGENT_PORT (default: 8443)
- AGENT_REQUIRE_JWT (recommended: true)
- AGENT_JWT_PUBLIC_KEY_PATH (path to controller public key)
- AGENT_JWT_ISSUER (issuer to pin)
"""

import os
import threading
import time
from typing import Tuple

import win32event  # type: ignore
import win32service  # type: ignore
import win32serviceutil  # type: ignore
import servicemanager  # type: ignore


def _get_host_and_port() -> Tuple[str, int]:
    host = os.getenv("AGENT_LISTEN_HOST", os.getenv("LISTEN_HOST", "0.0.0.0"))
    port_str = os.getenv("AGENT_PORT", os.getenv("PORT", "8443"))
    try:
        port = int(port_str)
    except ValueError:
        port = 8443
    return host, port


class ArcVaultAgentService(win32serviceutil.ServiceFramework):  # type: ignore[misc]
    _svc_name_ = "ArcVaultAgent"
    _svc_display_name_ = "ArcVault Agent Service"
    _svc_description_ = "Runs the ArcVault agent to execute backup jobs and stream logs."

    def __init__(self, args):
        win32serviceutil.ServiceFramework.__init__(self, args)
        self.stop_event = win32event.CreateEvent(None, 0, 0, None)
        self.server_thread: threading.Thread | None = None
        self.server = None

    def SvcStop(self):
        self.ReportServiceStatus(win32service.SERVICE_STOP_PENDING)
        try:
            # Signal uvicorn server to exit if available
            if self.server is not None:
                try:
                    self.server.should_exit = True  # type: ignore[attr-defined]
                except Exception:
                    pass
        finally:
            win32event.SetEvent(self.stop_event)

    def _run_server(self) -> None:
        # Defer import so PyInstaller can resolve dependencies cleanly
        import uvicorn  # type: ignore

        host, port = _get_host_and_port()
        config = uvicorn.Config("agent.app:app", host=host, port=port, log_level="info")
        self.server = uvicorn.Server(config)
        # Avoid installing signal handlers inside a service
        try:
            self.server.install_signal_handlers = lambda: None  # type: ignore[attr-defined]
        except Exception:
            pass
        self.server.run()

    def SvcDoRun(self):
        servicemanager.LogInfoMsg("ArcVaultAgent service starting")
        try:
            self.server_thread = threading.Thread(target=self._run_server, daemon=True)
            self.server_thread.start()
            # Wait until stop event is set
            while True:
                rc = win32event.WaitForSingleObject(self.stop_event, 500)
                if rc == win32event.WAIT_OBJECT_0:
                    break
                # Idle loop; could report heartbeat if desired
        finally:
            # Best-effort graceful shutdown
            if self.server is not None:
                try:
                    self.server.should_exit = True  # type: ignore[attr-defined]
                except Exception:
                    pass
            if self.server_thread is not None:
                # Give the server a moment to exit
                for _ in range(20):
                    if not self.server_thread.is_alive():
                        break
                    time.sleep(0.1)
        servicemanager.LogInfoMsg("ArcVaultAgent service stopped")


if __name__ == "__main__":
    win32serviceutil.HandleCommandLine(ArcVaultAgentService)  # type: ignore[misc]



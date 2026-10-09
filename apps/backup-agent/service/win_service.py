from __future__ import annotations

"""
Windows Service wrapper for the Backup Agent.

Build a one-file service EXE via PyInstaller, then install/start it with:
  RoboBackupAgentSvc.exe install
  RoboBackupAgentSvc.exe --startup auto start

Configuration (checked in order of priority):
1. service_config.json in the same directory as this script
2. Environment variables:
   - AGENT_LISTEN_HOST (default: 0.0.0.0)
   - AGENT_PORT (default: 8443)
   - AGENT_REQUIRE_JWT (recommended: true)
   - AGENT_JWT_PUBLIC_KEY_PATH (path to controller public key)
   - AGENT_JWT_ISSUER (issuer to pin)
3. Hardcoded defaults

Log files are written to: %TEMP%\arcvault_service.log
"""

import json
import os
import socket
import sys
import threading
import time
import traceback
from datetime import datetime
from pathlib import Path
from typing import Tuple

import win32event  # type: ignore
import win32service  # type: ignore
import win32serviceutil  # type: ignore
import servicemanager  # type: ignore


def _get_log_file() -> Path:
    """Get the path to the service log file."""
    temp_dir = os.environ.get('TEMP', 'C:/Windows/Temp')
    return Path(temp_dir) / 'arcvault_service.log'


def _log_to_file(message: str) -> None:
    """Write a message to the service log file."""
    try:
        log_file = _get_log_file()
        with open(log_file, 'a', encoding='utf-8') as f:
            timestamp = datetime.now().strftime('%Y-%m-%d %H:%M:%S')
            f.write(f"[{timestamp}] {message}\n")
    except Exception:
        # If we can't log to file, don't crash the service
        pass


def _is_port_available(port: int) -> bool:
    """Check if a port is available for binding."""
    with socket.socket(socket.AF_INET, socket.SOCK_STREAM) as s:
        try:
            s.bind(('0.0.0.0', port))
            return True
        except OSError:
            return False


def _get_host_and_port() -> Tuple[str, int]:
    """
    Get host and port configuration from:
    1. service_config.json file (if exists)
    2. Environment variables
    3. Hardcoded defaults
    """
    # Try to load from config file first
    try:
        config_path = Path(__file__).parent / "service_config.json"
        _log_to_file(f"Looking for config file at: {config_path}")
        
        if config_path.exists():
            with open(config_path, encoding='utf-8') as f:
                config = json.load(f)
            host = config.get("host", "0.0.0.0")
            port = config.get("port", 8443)
            _log_to_file(f"Loaded config from file: host={host}, port={port}")
            servicemanager.LogInfoMsg(f"Config loaded from {config_path}: {host}:{port}")
            return host, port
        else:
            _log_to_file(f"Config file not found at {config_path}, trying environment variables")
    except Exception as e:
        _log_to_file(f"Failed to read config file: {e}")
        servicemanager.LogWarningMsg(f"Failed to read config file: {e}")
    
    # Fallback to environment variables
    host = os.getenv("AGENT_LISTEN_HOST", os.getenv("LISTEN_HOST", "0.0.0.0"))
    port_str = os.getenv("AGENT_PORT", os.getenv("PORT", "8443"))
    try:
        port = int(port_str)
    except ValueError:
        port = 8443
    
    _log_to_file(f"Using configuration: host={host}, port={port} (from env/defaults)")
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
        """Run the uvicorn server in a separate thread."""
        try:
            _log_to_file("_run_server: Starting server thread")
            
            # Defer import so PyInstaller can resolve dependencies cleanly
            import uvicorn  # type: ignore
            import asyncio
            
            # Set up proper event loop for Windows service
            if sys.platform == 'win32':
                _log_to_file("_run_server: Setting Windows event loop policy")
                asyncio.set_event_loop_policy(asyncio.WindowsProactorEventLoopPolicy())

            host, port = _get_host_and_port()
            _log_to_file(f"_run_server: Creating uvicorn server on {host}:{port}")
            
            # Check port availability before starting
            if not _is_port_available(port):
                error_msg = f"Port {port} already in use. Cannot start agent."
                _log_to_file(f"ERROR: {error_msg}")
                servicemanager.LogErrorMsg(error_msg)
                raise RuntimeError(error_msg)
            
            # Configure uvicorn for Windows service (no stdout/stderr)
            config = uvicorn.Config(
                "agent.app:app", 
                host=host, 
                port=port, 
                log_level="info",
                use_colors=False,  # Disable colors since we have no stdout
                access_log=False   # Disable access log to avoid stdout issues
            )
            self.server = uvicorn.Server(config)
            
            # Avoid installing signal handlers inside a service
            try:
                self.server.install_signal_handlers = lambda: None  # type: ignore[attr-defined]
            except Exception:
                pass
            
            _log_to_file("_run_server: Starting uvicorn.run()")
            servicemanager.LogInfoMsg(f"Server starting on {host}:{port}")
            self.server.run()
            _log_to_file("_run_server: uvicorn.run() completed")
            
        except Exception as e:
            error_msg = f"Server thread crashed: {str(e)}\n{traceback.format_exc()}"
            _log_to_file(f"_run_server ERROR: {error_msg}")
            servicemanager.LogErrorMsg(error_msg)
            raise

    def SvcDoRun(self):
        """Main service loop with comprehensive logging and error handling."""
        try:
            # Initialize logging
            _log_to_file("\n" + "="*80)
            _log_to_file(f"=== ArcVault Agent Service Starting at {datetime.now()} ===")
            _log_to_file("="*80)
            
            servicemanager.LogInfoMsg("ArcVaultAgent service starting")
            
            # Log environment information
            _log_to_file(f"Python version: {sys.version}")
            _log_to_file(f"Working directory: {os.getcwd()}")
            _log_to_file(f"Script location: {__file__}")
            _log_to_file(f"Log file location: {_get_log_file()}")
            
            # Log configuration
            host, port = _get_host_and_port()
            _log_to_file(f"Server configuration: host={host}, port={port}")
            servicemanager.LogInfoMsg(f"Configuration: host={host}, port={port}")
            
            # Start the server thread
            _log_to_file("Creating server thread...")
            self.server_thread = threading.Thread(target=self._run_server, daemon=True)
            self.server_thread.start()
            _log_to_file("Server thread started successfully")
            
            # Report service is now RUNNING to Windows SCM
            # This is critical - without this, Windows thinks we're still starting
            self.ReportServiceStatus(win32service.SERVICE_RUNNING)
            _log_to_file("Reported SERVICE_RUNNING status to Windows")
            
            # Log to event log
            servicemanager.LogMsg(
                servicemanager.EVENTLOG_INFORMATION_TYPE,
                servicemanager.PYS_SERVICE_STARTED,
                (self._svc_name_, '')
            )
            
            # Wait until stop event is set
            _log_to_file("Entering main service loop, waiting for stop signal...")
            loop_count = 0
            while True:
                rc = win32event.WaitForSingleObject(self.stop_event, 5000)
                if rc == win32event.WAIT_OBJECT_0:
                    _log_to_file("Stop signal received")
                    break
                
                # Check if server thread is still alive
                loop_count += 1
                if loop_count % 12 == 0:  # Log every minute (12 * 5 seconds)
                    thread_alive = self.server_thread.is_alive() if self.server_thread else False
                    _log_to_file(f"Service heartbeat - Server thread alive: {thread_alive}")
                    
                    # If thread died unexpectedly, log error and stop service
                    if not thread_alive:
                        error_msg = "Server thread died unexpectedly!"
                        _log_to_file(f"CRITICAL: {error_msg}")
                        servicemanager.LogErrorMsg(error_msg)
                        break
                        
        except Exception as e:
            error_msg = f"Service crashed: {str(e)}\n{traceback.format_exc()}"
            _log_to_file(f"CRITICAL ERROR in SvcDoRun: {error_msg}")
            servicemanager.LogErrorMsg(error_msg)
            raise
            
        finally:
            _log_to_file("Beginning service shutdown...")
            
            # Best-effort graceful shutdown
            if self.server is not None:
                try:
                    _log_to_file("Signaling uvicorn server to exit...")
                    self.server.should_exit = True  # type: ignore[attr-defined]
                except Exception as e:
                    _log_to_file(f"Error signaling server exit: {e}")
                    
            if self.server_thread is not None:
                _log_to_file("Waiting for server thread to exit...")
                # Give the server a moment to exit
                for i in range(20):
                    if not self.server_thread.is_alive():
                        _log_to_file(f"Server thread exited after {i * 0.1:.1f} seconds")
                        break
                    time.sleep(0.1)
                else:
                    _log_to_file("Warning: Server thread did not exit gracefully")
                    
            _log_to_file("=== ArcVault Agent Service Stopped ===\n")
            servicemanager.LogInfoMsg("ArcVaultAgent service stopped")


if __name__ == "__main__":
    win32serviceutil.HandleCommandLine(ArcVaultAgentService)  # type: ignore[misc]

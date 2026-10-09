from __future__ import annotations

import os
import shutil
import subprocess
import time
from datetime import datetime
from typing import Callable, Iterable, Iterator, List, Tuple, Optional

from .config import BackupConfig
from .robocopy import build_robocopy_commands, classify_exit_code


LogSink = Callable[[str], None]


def _unc_share_root(path: str) -> Optional[str]:
    if path.startswith("\\\\"):
        parts = path.strip("\\").split("\\")
        if len(parts) >= 2:
            return f"\\\\{parts[0]}\\{parts[1]}"
    return None


def _net_use_connect(share: str, username: Optional[str], password: Optional[str], domain: Optional[str], emit: LogSink, max_retries: int = 3) -> None:
    """
    Connect to network share with retry logic and exponential backoff.
    
    Args:
        share: UNC path to the share
        username: Username for authentication
        password: Password for authentication
        domain: Domain for authentication
        emit: Logging function
        max_retries: Maximum number of connection attempts (default: 3)
    
    Raises:
        RuntimeError: If connection fails after all retry attempts
    """
    if not share:
        return
    user = None
    if username:
        user = f"{domain}\\{username}" if domain else username
    args = ["net", "use", share]
    if password:
        args.append(password)
    if user:
        args.append(f"/user:{user}")
    
    for attempt in range(max_retries):
        emit(f"NET USE connect {share} (attempt {attempt + 1}/{max_retries})")
        result = subprocess.run(args, capture_output=True, text=True)
        
        if result.returncode == 0:
            emit(f"✓ Connected to {share}")
            return
        
        # Log the error
        error_msg = result.stderr.strip() or result.stdout.strip() or "Unknown error"
        emit(f"⚠ Connection attempt {attempt + 1}/{max_retries} failed: {error_msg}")
        
        # If not the last attempt, wait before retrying
        if attempt < max_retries - 1:
            wait_time = 2 ** attempt  # Exponential backoff: 1s, 2s, 4s
            emit(f"Retrying in {wait_time} seconds...")
            time.sleep(wait_time)
    
    # All retries exhausted
    raise RuntimeError(f"Failed to connect to {share} after {max_retries} attempts")


def _net_use_disconnect(share: str, emit: LogSink) -> None:
    if not share:
        return
    args = ["net", "use", share, "/delete", "/y"]
    emit(f"NET USE disconnect {share}")
    subprocess.run(args, capture_output=True, text=True)


def run_backup(cfg: BackupConfig, emit: LogSink) -> Tuple[str, str]:
    """Execute robocopy for each source and emit logs in real-time.

    Returns (final_state, reason).
    """
    commands = build_robocopy_commands(cfg)
    src_share = _unc_share_root(cfg.sources[0]) if cfg.sources else None
    dst_share = _unc_share_root(cfg.destination)
    
    # Setup backup log file in destination
    backup_log_dir = os.path.join(cfg.destination, "#backup_logs")
    os.makedirs(backup_log_dir, exist_ok=True)
    timestamp = datetime.now().strftime("%Y%m%d_%H%M%S")
    log_file_path = os.path.join(backup_log_dir, f"backup_{timestamp}.log")
    
    def emit_and_log(message: str):
        """Emit to real-time stream and write to log file"""
        emit(message)
        try:
            with open(log_file_path, "a", encoding="utf-8") as f:
                f.write(f"[{datetime.now().isoformat()}] {message}\n")
        except Exception:
            pass  # Don't fail backup if logging fails
    
    emit_and_log(f"Backup started - Log file: {log_file_path}")
    
    # Basic resource monitoring
    try:
        # Check destination disk space
        total, used, free = shutil.disk_usage(cfg.destination)
        free_gb = free // (1024**3)
        emit_and_log(f"Destination disk space: {free_gb} GB free")
        
        if free_gb < 1:  # Less than 1GB free
            emit_and_log("WARNING: Low disk space on destination")
            return "failed", f"Insufficient disk space: only {free_gb} GB free"
        
        # Log source paths for verification
        for idx, source in enumerate(cfg.sources, 1):
            if os.path.exists(source):
                emit_and_log(f"Source {idx}: {source} - accessible")
            else:
                emit_and_log(f"ERROR: Source {idx}: {source} - not accessible")
                return "failed", f"Source path not accessible: {source}"
                
    except Exception as exc:
        emit_and_log(f"Warning: Resource check failed: {exc}")
    
    try:
        # Connect to network shares with retry logic
        try:
            if cfg.source_credentials and src_share:
                _net_use_connect(src_share, cfg.source_credentials.username, cfg.source_credentials.password, cfg.source_credentials.domain, emit_and_log)
            if cfg.destination_credentials and dst_share:
                _net_use_connect(dst_share, cfg.destination_credentials.username, cfg.destination_credentials.password, cfg.destination_credentials.domain, emit_and_log)
        except RuntimeError as exc:
            # Network share connection failed after retries
            emit_and_log(f"ERROR: Network share connection failed: {exc}")
            return "failed", str(exc)

        for idx, argv in enumerate(commands, start=1):
            emit_and_log(f"Executing robocopy {idx}/{len(commands)}: {' '.join(argv)}")
            proc = subprocess.Popen(argv, stdout=subprocess.PIPE, stderr=subprocess.STDOUT, text=True)
            assert proc.stdout is not None
            for line in iter(proc.stdout.readline, ''):
                if not line:
                    break
                emit_and_log(line.rstrip('\n'))
            rc = proc.wait()
            state, reason = classify_exit_code(rc)
            emit_and_log(f"robocopy finished with code {rc} -> {state}")
            if state != "succeeded":
                emit_and_log(f"Backup failed: {reason}")
                return state, reason
        emit_and_log("Backup completed successfully - all sources processed")
        return "succeeded", "all sources processed"
    finally:
        if src_share:
            _net_use_disconnect(src_share, emit)
        if dst_share:
            _net_use_disconnect(dst_share, emit)



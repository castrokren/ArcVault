"""
Disk-based logging handler for backup jobs.
Uses RotatingFileHandler to prevent log files from growing too large.
"""
import os
import logging
import threading
from logging.handlers import RotatingFileHandler
from typing import Dict, Optional, Tuple
from pathlib import Path


# Default logs directory - can be overridden via environment variable
DEFAULT_LOGS_DIR = os.path.join(
    os.getenv("PROGRAMDATA", "C:/ProgramData"),
    "ArcVault",
    "logs"
)

# Keep loggers and handlers in memory to reuse them
# Store (logger, destination) tuple to validate destination consistency
_job_loggers: Dict[str, Tuple[logging.Logger, Optional[str]]] = {}
_logger_lock = threading.Lock()


def get_logs_dir() -> str:
    """Get the logs directory path from environment or use default."""
    return os.getenv("ARCVAULT_LOGS_DIR", DEFAULT_LOGS_DIR)


def get_job_logger(job_id: str, destination: Optional[str] = None) -> logging.Logger:
    """
    Get or create a logger for a specific job.
    
    Thread-safe: Multiple threads can safely call this for the same job.
    
    Args:
        job_id: Unique job identifier
        destination: Optional destination path to create log in destination folder
        
    Returns:
        Logger instance for the job
    """
    with _logger_lock:
        # Check if logger already exists
        if job_id in _job_loggers:
            cached_logger, cached_dest = _job_loggers[job_id]
            
            # Validate destination matches (if both are specified)
            if destination and cached_dest and destination != cached_dest:
                # Log warning but return cached logger (don't recreate mid-job)
                cached_logger.warning(
                    f"Logger destination mismatch: cached={cached_dest}, requested={destination}. "
                    f"Using cached destination."
                )
            
            return cached_logger
        
        # Create new logger
        logger = logging.getLogger(f"job.{job_id}")
        logger.setLevel(logging.INFO)
        logger.propagate = False  # Don't propagate to root logger
        
        # Determine log file path
        if destination:
            # Store logs in destination folder under #backup_logs subdirectory
            log_dir = os.path.join(destination, "#backup_logs")
        else:
            # Store in default logs directory
            log_dir = get_logs_dir()
        
        # Ensure directory exists
        try:
            os.makedirs(log_dir, exist_ok=True)
        except (OSError, PermissionError):
            # Fallback to default logs directory if destination is unavailable
            log_dir = get_logs_dir()
            os.makedirs(log_dir, exist_ok=True)
        
        log_file = os.path.join(log_dir, f"{job_id}.log")
        
        # Create rotating file handler
        # Max 10MB per file, keep 3 backup files
        handler = RotatingFileHandler(
            log_file,
            maxBytes=10 * 1024 * 1024,  # 10MB
            backupCount=3,
            encoding="utf-8"
        )
        
        # Set format
        formatter = logging.Formatter(
            "%(asctime)s [%(levelname)s] %(message)s",
            datefmt="%Y-%m-%d %H:%M:%S"
        )
        handler.setFormatter(formatter)
        
        logger.addHandler(handler)
        
        # Cache logger with its destination
        _job_loggers[job_id] = (logger, destination)
        
        return logger


def log_to_job(job_id: str, message: str, level: int = logging.INFO) -> None:
    """
    Log a message to a job's log file.
    
    Args:
        job_id: Job identifier
        message: Message to log
        level: Logging level (default: INFO)
    """
    logger = get_job_logger(job_id)
    logger.log(level, message)


def get_job_log_path(job_id: str, destination: Optional[str] = None) -> str:
    """
    Get the path to a job's log file.
    
    Args:
        job_id: Job identifier
        destination: Optional destination path
        
    Returns:
        Path to the log file
    """
    if destination:
        log_dir = os.path.join(destination, "#backup_logs")
    else:
        log_dir = get_logs_dir()
    
    return os.path.join(log_dir, f"{job_id}.log")


def read_job_logs(job_id: str, destination: Optional[str] = None, tail_lines: int = 1000) -> list:
    """
    Read the last N lines from a job's log file.
    
    Args:
        job_id: Job identifier
        destination: Optional destination path
        tail_lines: Number of lines to read from the end (default: 1000)
        
    Returns:
        List of log lines
    """
    log_path = get_job_log_path(job_id, destination)
    
    if not os.path.exists(log_path):
        return []
    
    try:
        with open(log_path, "r", encoding="utf-8", errors="ignore") as f:
            lines = f.readlines()
            # Return last N lines
            return lines[-tail_lines:] if len(lines) > tail_lines else lines
    except Exception:
        return []


def follow_job_logs(job_id: str, destination: Optional[str] = None, start_pos: int = 0):
    """
    Generator that yields new log lines as they are written.
    Similar to 'tail -f' functionality.
    
    Args:
        job_id: Job identifier
        destination: Optional destination path
        start_pos: File position to start reading from
        
    Yields:
        New log lines as they are written
    """
    log_path = get_job_log_path(job_id, destination)
    
    # Wait for log file to be created
    import time
    max_wait = 30  # seconds
    waited = 0
    while not os.path.exists(log_path) and waited < max_wait:
        time.sleep(0.5)
        waited += 0.5
    
    if not os.path.exists(log_path):
        return
    
    try:
        with open(log_path, "r", encoding="utf-8", errors="ignore") as f:
            # Seek to start position
            f.seek(start_pos)
            
            while True:
                line = f.readline()
                if line:
                    yield line.rstrip("\n")
                else:
                    # Check if job is finished
                    # This check would be done by the caller
                    break
    except Exception:
        return


def close_job_logger(job_id: str) -> None:
    """
    Close and remove a job's logger.
    Call this when a job finishes to free resources.
    
    Thread-safe: Can be called from any thread.
    
    Args:
        job_id: Job identifier
    """
    with _logger_lock:
        if job_id in _job_loggers:
            logger, _ = _job_loggers[job_id]
            # Close all handlers
            for handler in logger.handlers[:]:
                handler.close()
                logger.removeHandler(handler)
            del _job_loggers[job_id]


def cleanup_old_logs(days: int = 30) -> int:
    """
    Delete log files older than specified days.
    
    Args:
        days: Delete logs older than this many days
        
    Returns:
        Number of log files deleted
    """
    import time
    
    log_dir = get_logs_dir()
    if not os.path.exists(log_dir):
        return 0
    
    cutoff = time.time() - (days * 24 * 60 * 60)
    deleted = 0
    
    for filename in os.listdir(log_dir):
        if not filename.endswith(".log"):
            continue
        
        filepath = os.path.join(log_dir, filename)
        try:
            if os.path.getmtime(filepath) < cutoff:
                os.remove(filepath)
                deleted += 1
                
                # Also remove rotated backups
                for i in range(1, 4):
                    backup_path = f"{filepath}.{i}"
                    if os.path.exists(backup_path):
                        os.remove(backup_path)
                        deleted += 1
        except Exception:
            continue
    
    return deleted


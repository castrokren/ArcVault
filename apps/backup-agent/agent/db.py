"""
Database module for persistent job storage.
Stores job metadata, states, and completion times in SQLite.
"""
import os
import json
import sqlite3
import time
from contextlib import contextmanager
from typing import Dict, List, Optional, Iterator


# Default database path - can be overridden via environment variable
DEFAULT_DB_PATH = os.path.join(
    os.getenv("PROGRAMDATA", "C:/ProgramData"),
    "ArcVault",
    "jobs.db"
)


def get_db_path() -> str:
    """Get the database path from environment or use default."""
    return os.getenv("ARCVAULT_DB_PATH", DEFAULT_DB_PATH)


@contextmanager
def get_db() -> Iterator[sqlite3.Connection]:
    """Context manager for database connections."""
    db_path = get_db_path()
    # Ensure directory exists
    os.makedirs(os.path.dirname(db_path), exist_ok=True)
    
    conn = sqlite3.connect(db_path)
    conn.row_factory = sqlite3.Row  # Enable column access by name
    try:
        yield conn
    finally:
        conn.close()


def init_db() -> None:
    """Initialize the database schema."""
    with get_db() as conn:
        conn.execute("""
            CREATE TABLE IF NOT EXISTS jobs (
                id TEXT PRIMARY KEY,
                state TEXT NOT NULL,
                progress INTEGER DEFAULT 0,
                error TEXT,
                sources TEXT,
                config TEXT,
                created_at REAL NOT NULL,
                started_at REAL,
                finished_at REAL,
                updated_at REAL NOT NULL
            )
        """)
        
        # Create index for faster queries by state and created_at
        conn.execute("""
            CREATE INDEX IF NOT EXISTS idx_jobs_state 
            ON jobs(state)
        """)
        conn.execute("""
            CREATE INDEX IF NOT EXISTS idx_jobs_created_at 
            ON jobs(created_at DESC)
        """)
        
        conn.commit()


def save_job(job_id: str, job_data: dict) -> None:
    """
    Save or update a job in the database.
    
    Args:
        job_id: Unique job identifier
        job_data: Job metadata dictionary
    """
    with get_db() as conn:
        now = time.time()
        
        # Extract fields from job_data
        state = job_data.get("state", "queued")
        progress = job_data.get("progress", 0)
        error = job_data.get("error")
        sources = json.dumps(job_data.get("sources", []))
        config = json.dumps(job_data.get("config", {}))
        created_at = job_data.get("created_at", now)
        started_at = job_data.get("started_at")
        finished_at = job_data.get("finished_at")
        
        conn.execute("""
            INSERT INTO jobs (
                id, state, progress, error, sources, config,
                created_at, started_at, finished_at, updated_at
            ) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
            ON CONFLICT(id) DO UPDATE SET
                state = excluded.state,
                progress = excluded.progress,
                error = excluded.error,
                started_at = excluded.started_at,
                finished_at = excluded.finished_at,
                updated_at = excluded.updated_at
        """, (
            job_id, state, progress, error, sources, config,
            created_at, started_at, finished_at, now
        ))
        conn.commit()


def load_job(job_id: str) -> Optional[Dict]:
    """
    Load a job from the database.
    
    Args:
        job_id: Job identifier
        
    Returns:
        Job data dictionary or None if not found
    """
    with get_db() as conn:
        cursor = conn.execute(
            "SELECT * FROM jobs WHERE id = ?",
            (job_id,)
        )
        row = cursor.fetchone()
        if not row:
            return None
        
        return _row_to_dict(row)


def load_all_jobs() -> Dict[str, Dict]:
    """
    Load all jobs from the database.
    
    Returns:
        Dictionary mapping job_id to job data
    """
    with get_db() as conn:
        cursor = conn.execute("SELECT * FROM jobs ORDER BY created_at DESC")
        jobs = {}
        for row in cursor.fetchall():
            job_data = _row_to_dict(row)
            jobs[row["id"]] = job_data
        return jobs


def load_active_jobs() -> Dict[str, Dict]:
    """
    Load only queued and running jobs from the database.
    Useful for recovery on restart.
    
    Returns:
        Dictionary mapping job_id to job data
    """
    with get_db() as conn:
        cursor = conn.execute(
            "SELECT * FROM jobs WHERE state IN ('queued', 'running') ORDER BY created_at"
        )
        jobs = {}
        for row in cursor.fetchall():
            job_data = _row_to_dict(row)
            jobs[row["id"]] = job_data
        return jobs


def delete_old_jobs(days: int = 30) -> int:
    """
    Delete jobs older than specified days.
    
    Args:
        days: Delete jobs older than this many days
        
    Returns:
        Number of jobs deleted
    """
    cutoff = time.time() - (days * 24 * 60 * 60)
    with get_db() as conn:
        cursor = conn.execute(
            "DELETE FROM jobs WHERE created_at < ? AND state IN ('succeeded', 'failed')",
            (cutoff,)
        )
        conn.commit()
        return cursor.rowcount


def get_job_stats() -> Dict:
    """
    Get statistics about jobs in the database.
    
    Returns:
        Dictionary with job counts by state
    """
    with get_db() as conn:
        cursor = conn.execute("""
            SELECT 
                COUNT(*) as total,
                SUM(CASE WHEN state = 'queued' THEN 1 ELSE 0 END) as queued,
                SUM(CASE WHEN state = 'running' THEN 1 ELSE 0 END) as running,
                SUM(CASE WHEN state = 'succeeded' THEN 1 ELSE 0 END) as succeeded,
                SUM(CASE WHEN state = 'failed' THEN 1 ELSE 0 END) as failed
            FROM jobs
        """)
        row = cursor.fetchone()
        return {
            "total": row["total"],
            "queued": row["queued"] or 0,
            "running": row["running"] or 0,
            "succeeded": row["succeeded"] or 0,
            "failed": row["failed"] or 0,
        }


def _row_to_dict(row: sqlite3.Row) -> Dict:
    """Convert a database row to a job data dictionary."""
    return {
        "state": row["state"],
        "progress": row["progress"],
        "error": row["error"],
        "sources": json.loads(row["sources"]) if row["sources"] else [],
        "config": json.loads(row["config"]) if row["config"] else {},
        "created_at": row["created_at"],
        "started_at": row["started_at"],
        "finished_at": row["finished_at"],
    }


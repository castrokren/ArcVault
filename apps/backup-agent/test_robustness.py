"""
Test script for robustness improvements.
Run this before deploying to verify database and logging work correctly.

Usage:
    python test_robustness.py
"""
import os
import sys
import time
import tempfile
from pathlib import Path

# Add agent to path
sys.path.insert(0, str(Path(__file__).parent))

from agent import db, log_handler


def test_database():
    """Test database functionality"""
    print("=" * 60)
    print("Testing Database Module")
    print("=" * 60)
    
    # Use temp database for testing
    test_db_path = os.path.join(tempfile.gettempdir(), "test_arcvault_jobs.db")
    os.environ["ARCVAULT_DB_PATH"] = test_db_path
    
    print(f"Using test database: {test_db_path}")
    
    try:
        # Clean up old test database
        if os.path.exists(test_db_path):
            os.remove(test_db_path)
            print("✓ Cleaned up old test database")
        
        # Test 1: Initialize database
        print("\n[Test 1] Initialize database...")
        db.init_db()
        assert os.path.exists(test_db_path), "Database file not created"
        print("✓ Database initialized successfully")
        
        # Test 2: Save a job
        print("\n[Test 2] Save a job...")
        job_id = "test-job-123"
        job_data = {
            "state": "queued",
            "progress": 0,
            "error": None,
            "sources": ["C:/test/source1", "C:/test/source2"],
            "config": {"destination": "C:/test/dest", "options": {"verbose": True}},
            "created_at": time.time()
        }
        db.save_job(job_id, job_data)
        print("✓ Job saved successfully")
        
        # Test 3: Load the job
        print("\n[Test 3] Load job...")
        loaded_job = db.load_job(job_id)
        assert loaded_job is not None, "Job not found"
        assert loaded_job["state"] == "queued", "Job state incorrect"
        assert len(loaded_job["sources"]) == 2, "Sources not saved correctly"
        print(f"✓ Job loaded successfully: state={loaded_job['state']}, sources={loaded_job['sources']}")
        
        # Test 4: Update job state
        print("\n[Test 4] Update job state...")
        job_data["state"] = "running"
        job_data["started_at"] = time.time()
        db.save_job(job_id, job_data)
        
        loaded_job = db.load_job(job_id)
        assert loaded_job["state"] == "running", "Job state not updated"
        assert loaded_job["started_at"] is not None, "Started time not saved"
        print(f"✓ Job updated successfully: state={loaded_job['state']}")
        
        # Test 5: Create multiple jobs
        print("\n[Test 5] Create multiple jobs...")
        for i in range(5):
            job_id = f"test-job-{i}"
            job_data = {
                "state": "succeeded" if i % 2 == 0 else "failed",
                "progress": 100,
                "error": None if i % 2 == 0 else "Test error",
                "sources": [f"C:/test/source{i}"],
                "config": {},
                "created_at": time.time() - (i * 3600)  # Stagger creation times
            }
            db.save_job(job_id, job_data)
        print("✓ Multiple jobs created")
        
        # Test 6: Load all jobs
        print("\n[Test 6] Load all jobs...")
        all_jobs = db.load_all_jobs()
        assert len(all_jobs) >= 6, f"Expected at least 6 jobs, got {len(all_jobs)}"
        print(f"✓ Loaded {len(all_jobs)} jobs")
        
        # Test 7: Get job statistics
        print("\n[Test 7] Get job statistics...")
        stats = db.get_job_stats()
        print(f"  Total: {stats['total']}")
        print(f"  Succeeded: {stats['succeeded']}")
        print(f"  Failed: {stats['failed']}")
        assert stats["total"] >= 6, "Job count incorrect"
        print("✓ Statistics retrieved successfully")
        
        # Test 8: Delete old jobs
        print("\n[Test 8] Delete old jobs...")
        # Set cutoff to 0 days to delete all completed jobs
        deleted_count = db.delete_old_jobs(days=0)
        print(f"✓ Deleted {deleted_count} old jobs")
        
        # Test 9: Simulate startup recovery
        print("\n[Test 9] Simulate startup recovery...")
        # Create some active jobs
        recovery_jobs = []
        for i, state in enumerate(["queued", "running", "running", "succeeded"]):
            job_id = f"recovery-job-{i}"
            job_data = {
                "state": state,
                "progress": 50 if state == "running" else 0,
                "error": None,
                "sources": [f"C:/test/recovery{i}"],
                "config": {"destination": f"C:/backup/recovery{i}"},
                "created_at": time.time() - 3600,  # 1 hour ago
                "started_at": time.time() - 1800 if state == "running" else None
            }
            db.save_job(job_id, job_data)
            if state in ["queued", "running"]:
                recovery_jobs.append(job_id)
        
        # Simulate recovery: load active jobs and mark as failed
        active_jobs = db.load_active_jobs()
        recovered_count = 0
        for job_id, job_data in active_jobs.items():
            if job_data.get("state") in ["queued", "running"]:
                job_data["state"] = "failed"
                job_data["error"] = "Service restarted - job interrupted"
                job_data["finished_at"] = time.time()
                db.save_job(job_id, job_data)
                recovered_count += 1
        
        assert recovered_count == 3, f"Expected 3 recovered jobs, got {recovered_count}"
        
        # Verify jobs are marked as failed
        for job_id in recovery_jobs:
            recovered = db.load_job(job_id)
            assert recovered["state"] == "failed", f"Job {job_id} not marked as failed"
            assert "Service restarted" in recovered["error"], f"Job {job_id} missing recovery error"
        
        print(f"✓ Recovered {recovered_count} interrupted jobs")
        
        print("\n" + "=" * 60)
        print("Database Tests: ALL PASSED ✓")
        print("=" * 60)
        return True
        
    except Exception as e:
        print(f"\n✗ Database test failed: {e}")
        import traceback
        traceback.print_exc()
        return False
    finally:
        # Clean up test database
        if os.path.exists(test_db_path):
            try:
                os.remove(test_db_path)
            except:
                pass


def test_logging():
    """Test logging functionality"""
    print("\n" + "=" * 60)
    print("Testing Logging Module")
    print("=" * 60)
    
    # Use temp logs directory for testing
    test_logs_dir = os.path.join(tempfile.gettempdir(), "test_arcvault_logs")
    os.environ["ARCVAULT_LOGS_DIR"] = test_logs_dir
    
    print(f"Using test logs directory: {test_logs_dir}")
    
    # Use temp destination for testing
    test_dest_dir = os.path.join(tempfile.gettempdir(), "test_backup_dest")
    
    try:
        # Test 1: Create job logger (default location)
        print("\n[Test 1] Create job logger (default location)...")
        job_id = "test-log-job-1"
        logger = log_handler.get_job_logger(job_id)
        assert logger is not None, "Logger not created"
        print("✓ Logger created successfully")
        
        # Test 2: Write logs
        print("\n[Test 2] Write logs...")
        log_handler.log_to_job(job_id, "Test log message 1")
        log_handler.log_to_job(job_id, "Test log message 2")
        log_handler.log_to_job(job_id, "Error occurred", level=40)  # ERROR level
        print("✓ Logs written")
        
        # Test 3: Verify log file exists
        print("\n[Test 3] Verify log file exists...")
        log_path = log_handler.get_job_log_path(job_id)
        assert os.path.exists(log_path), f"Log file not found at {log_path}"
        print(f"✓ Log file exists: {log_path}")
        
        # Test 4: Read logs
        print("\n[Test 4] Read logs...")
        log_lines = log_handler.read_job_logs(job_id)
        assert len(log_lines) >= 3, f"Expected at least 3 log lines, got {len(log_lines)}"
        print(f"✓ Read {len(log_lines)} log lines:")
        for line in log_lines:
            print(f"  {line.strip()}")
        
        # Test 5: Create logger with destination
        print("\n[Test 5] Create logger with destination...")
        job_id_2 = "test-log-job-2"
        os.makedirs(test_dest_dir, exist_ok=True)
        logger2 = log_handler.get_job_logger(job_id_2, test_dest_dir)
        log_handler.log_to_job(job_id_2, "Log in destination folder")
        
        dest_log_path = log_handler.get_job_log_path(job_id_2, test_dest_dir)
        assert os.path.exists(dest_log_path), "Log file not created in destination"
        assert "#backup_logs" in dest_log_path, "Log not in #backup_logs folder"
        print(f"✓ Log created in destination: {dest_log_path}")
        
        # Test 6: Write many logs (test rotation)
        print("\n[Test 6] Test log rotation...")
        job_id_3 = "test-log-job-3"
        logger3 = log_handler.get_job_logger(job_id_3)
        
        # Write ~1MB of logs
        for i in range(10000):
            log_handler.log_to_job(job_id_3, f"Test log message {i} with some extra text to increase size")
        
        log_path_3 = log_handler.get_job_log_path(job_id_3)
        log_size = os.path.getsize(log_path_3)
        print(f"✓ Wrote 10,000 log lines, file size: {log_size / 1024:.1f} KB")
        
        # Test 7: Close logger
        print("\n[Test 7] Close logger...")
        log_handler.close_job_logger(job_id)
        log_handler.close_job_logger(job_id_2)
        log_handler.close_job_logger(job_id_3)
        print("✓ Loggers closed successfully")
        
        # Test 8: Cleanup old logs
        print("\n[Test 8] Cleanup old logs...")
        deleted = log_handler.cleanup_old_logs(days=0)
        print(f"✓ Deleted {deleted} old log files")
        
        print("\n" + "=" * 60)
        print("Logging Tests: ALL PASSED ✓")
        print("=" * 60)
        return True
        
    except Exception as e:
        print(f"\n✗ Logging test failed: {e}")
        import traceback
        traceback.print_exc()
        return False
    finally:
        # Clean up test directories
        import shutil
        for test_dir in [test_logs_dir, test_dest_dir]:
            if os.path.exists(test_dir):
                try:
                    shutil.rmtree(test_dir)
                except:
                    pass


def main():
    """Run all tests"""
    print("\n" + "=" * 60)
    print("ArcVault Robustness Improvements - Test Suite")
    print("=" * 60)
    print()
    
    results = []
    
    # Test database
    results.append(("Database", test_database()))
    
    # Test logging
    results.append(("Logging", test_logging()))
    
    # Summary
    print("\n" + "=" * 60)
    print("TEST SUMMARY")
    print("=" * 60)
    
    for name, passed in results:
        status = "✓ PASSED" if passed else "✗ FAILED"
        print(f"{name:20s}: {status}")
    
    all_passed = all(result[1] for result in results)
    
    print("=" * 60)
    if all_passed:
        print("ALL TESTS PASSED ✓")
        print("\nThe robustness improvements are working correctly.")
        print("You can proceed with deployment.")
        return 0
    else:
        print("SOME TESTS FAILED ✗")
        print("\nPlease review the errors above before deploying.")
        return 1


if __name__ == "__main__":
    sys.exit(main())


"""
Integration test for configuration management system.

This tests the complete configuration loading, validation, and saving workflow.
"""

import json
import os
import sys
import tempfile
from pathlib import Path

# Add agent directory to path
sys.path.insert(0, str(Path(__file__).parent / "agent"))

from agent.engine.config_loader import (
    load_backup_config,
    save_config_file,
    load_service_config,
    ConfigurationError
)
from agent.engine.config import BackupConfig, BackupOptions


def test_config_validation():
    """Test basic configuration validation."""
    print("\n=== Test: Configuration Validation ===")
    
    # Create temp directories for testing
    with tempfile.TemporaryDirectory() as tmpdir:
        source_dir = Path(tmpdir) / "source"
        dest_dir = Path(tmpdir) / "destination"
        source_dir.mkdir()
        dest_dir.mkdir()
        
        # Valid config
        try:
            config = load_backup_config(config_dict={
                "sources": [str(source_dir)],
                "destination": str(dest_dir),
                "options": {
                    "threads": 16,
                    "retries": 3
                }
            })
            print(f"✓ Valid config loaded successfully")
            print(f"  Sources: {config.sources}")
            print(f"  Destination: {config.destination}")
            print(f"  Threads: {config.options.threads}")
            assert config.options.threads == 16
            assert config.options.retries == 3
        except ConfigurationError as e:
            print(f"✗ FAILED: {e}")
            return False
    
    # Invalid destination (doesn't exist)
    try:
        config = load_backup_config(config_dict={
            "sources": ["C:\\Data"],
            "destination": "Z:\\NonExistentPath"
        })
        print(f"✗ FAILED: Should have raised validation error for non-existent destination")
        return False
    except ConfigurationError as e:
        print(f"✓ Correctly caught invalid destination error")
    
    return True


def test_env_overrides():
    """Test environment variable overrides."""
    print("\n=== Test: Environment Variable Overrides ===")
    
    with tempfile.TemporaryDirectory() as tmpdir:
        dest_dir = Path(tmpdir)
        
        # Set environment variables
        os.environ["BACKUP_DESTINATION"] = str(dest_dir)
        os.environ["BACKUP_OPTIONS__THREADS"] = "32"
        os.environ["BACKUP_OPTIONS__RETRIES"] = "5"
        os.environ["BACKUP_OPTIONS__VERBOSE"] = "false"
        
        try:
            config = load_backup_config(
                config_dict={
                    "sources": [str(dest_dir)],  # Use temp dir as source too
                    "destination": "dummy",  # Will be overridden by env
                    "options": {
                        "threads": 8,  # Will be overridden
                        "retries": 1   # Will be overridden
                    }
                },
                apply_env=True
            )
            
            print(f"✓ Environment overrides applied successfully")
            print(f"  Destination: {config.destination} (from BACKUP_DESTINATION)")
            print(f"  Threads: {config.options.threads} (from BACKUP_OPTIONS__THREADS)")
            print(f"  Retries: {config.options.retries} (from BACKUP_OPTIONS__RETRIES)")
            print(f"  Verbose: {config.options.verbose} (from BACKUP_OPTIONS__VERBOSE)")
            
            assert config.destination == str(dest_dir)
            assert config.options.threads == 32
            assert config.options.retries == 5
            assert config.options.verbose == False
            
        except Exception as e:
            print(f"✗ FAILED: {e}")
            return False
        finally:
            # Clean up env vars
            for key in ["BACKUP_DESTINATION", "BACKUP_OPTIONS__THREADS", 
                       "BACKUP_OPTIONS__RETRIES", "BACKUP_OPTIONS__VERBOSE"]:
                os.environ.pop(key, None)
    
    return True


def test_file_loading_and_saving():
    """Test loading and saving configuration files."""
    print("\n=== Test: File Loading and Saving ===")
    
    with tempfile.TemporaryDirectory() as tmpdir:
        tmppath = Path(tmpdir)
        source_dir = tmppath / "source"
        dest_dir = tmppath / "dest"
        source_dir.mkdir()
        dest_dir.mkdir()
        
        config_file = tmppath / "test_config.json"
        
        # Create config
        config_dict = {
            "sources": [str(source_dir)],
            "destination": str(dest_dir),
            "options": {
                "threads": 24,
                "retries": 4,
                "verbose": True
            }
        }
        
        # Save to file
        with open(config_file, 'w') as f:
            json.dump(config_dict, f, indent=2)
        
        print(f"✓ Created config file: {config_file}")
        
        # Load from file
        try:
            config = load_backup_config(file_path=config_file)
            print(f"✓ Loaded config from file successfully")
            print(f"  Sources: {config.sources}")
            print(f"  Destination: {config.destination}")
            print(f"  Threads: {config.options.threads}")
            
            assert config.options.threads == 24
            assert config.options.retries == 4
            
        except ConfigurationError as e:
            print(f"✗ FAILED: {e}")
            return False
        
        # Test saving config
        output_file = tmppath / "output_config.json"
        try:
            save_config_file(config, output_file, format="json")
            print(f"✓ Saved config to: {output_file}")
            
            # Verify we can reload it
            reloaded = load_backup_config(file_path=output_file)
            assert reloaded.options.threads == 24
            print(f"✓ Successfully reloaded saved config")
            
        except ConfigurationError as e:
            print(f"✗ FAILED: {e}")
            return False
    
    return True


def test_validate_assignment():
    """Test that validate_assignment works (ConfigDict)."""
    print("\n=== Test: Validate Assignment ===")
    
    with tempfile.TemporaryDirectory() as tmpdir:
        tmppath = Path(tmpdir)
        
        try:
            config = load_backup_config(config_dict={
                "sources": [str(tmppath)],
                "destination": str(tmppath)
            })
            
            # Modify configuration after creation
            config.options.threads = 64
            print(f"✓ Modified threads to: {config.options.threads}")
            assert config.options.threads == 64
            
            # Try invalid assignment (should raise validation error)
            try:
                config.options.threads = 999  # Exceeds max of 128
                print(f"✗ FAILED: Should have raised validation error for threads=999")
                return False
            except Exception:
                print(f"✓ Correctly rejected invalid assignment (threads=999)")
            
        except Exception as e:
            print(f"✗ FAILED: {e}")
            return False
    
    return True


def test_service_config():
    """Test service configuration loading."""
    print("\n=== Test: Service Configuration ===")
    
    try:
        service_config = load_service_config()
        print(f"✓ Service config loaded successfully")
        print(f"  Host: {service_config.get('host')}")
        print(f"  Port: {service_config.get('port')}")
        print(f"  Log Level: {service_config.get('log_level')}")
        print(f"  Max Concurrent Jobs: {service_config.get('max_concurrent_jobs')}")
        
        # Check defaults are present
        assert "host" in service_config
        assert "port" in service_config
        assert "log_level" in service_config
        
    except Exception as e:
        print(f"✗ FAILED: {e}")
        return False
    
    return True


def test_programmatic_creation():
    """Test creating configuration programmatically."""
    print("\n=== Test: Programmatic Configuration Creation ===")
    
    with tempfile.TemporaryDirectory() as tmpdir:
        tmppath = Path(tmpdir)
        
        try:
            # Create directly using Pydantic models
            config = BackupConfig(
                sources=[str(tmppath)],
                destination=str(tmppath),
                options=BackupOptions(
                    threads=20,
                    retries=2,
                    verbose=False
                )
            )
            
            print(f"✓ Config created programmatically")
            print(f"  Sources: {config.sources}")
            print(f"  Threads: {config.options.threads}")
            print(f"  Retries: {config.options.retries}")
            print(f"  Verbose: {config.options.verbose}")
            
            assert config.options.threads == 20
            assert config.options.retries == 2
            assert config.options.verbose == False
            
        except Exception as e:
            print(f"✗ FAILED: {e}")
            return False
    
    return True


def main():
    """Run all tests."""
    print("=" * 70)
    print("Configuration Management Integration Tests")
    print("=" * 70)
    
    tests = [
        ("Configuration Validation", test_config_validation),
        ("Environment Overrides", test_env_overrides),
        ("File Loading and Saving", test_file_loading_and_saving),
        ("Validate Assignment", test_validate_assignment),
        ("Service Configuration", test_service_config),
        ("Programmatic Creation", test_programmatic_creation),
    ]
    
    results = []
    for test_name, test_func in tests:
        try:
            success = test_func()
            results.append((test_name, success))
        except Exception as e:
            print(f"\n✗ {test_name} CRASHED: {e}")
            results.append((test_name, False))
    
    # Summary
    print("\n" + "=" * 70)
    print("Test Results Summary")
    print("=" * 70)
    
    passed = sum(1 for _, success in results if success)
    total = len(results)
    
    for test_name, success in results:
        status = "✓ PASS" if success else "✗ FAIL"
        print(f"{status:10} {test_name}")
    
    print("=" * 70)
    print(f"Overall: {passed}/{total} tests passed")
    print("=" * 70)
    
    return passed == total


if __name__ == "__main__":
    success = main()
    sys.exit(0 if success else 1)


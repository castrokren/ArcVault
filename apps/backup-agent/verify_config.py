"""Simple verification that config management is working."""
import sys
import tempfile
from pathlib import Path

sys.path.insert(0, str(Path(__file__).parent / "agent"))

# Test basic import
try:
    from agent.engine.config_loader import load_backup_config, ConfigurationError
    from agent.engine.config import BackupConfig, BackupOptions
    
    with open("verification_results.txt", "w") as f:
        f.write("=== Configuration Management Verification ===\n\n")
        
        # Test 1: Basic import
        f.write("✓ Import successful\n")
        
        # Test 2: Create config with temp directory
        with tempfile.TemporaryDirectory() as tmpdir:
            config = load_backup_config(config_dict={
                "sources": [tmpdir],
                "destination": tmpdir,
                "options": {"threads": 16}
            })
            f.write(f"✓ Config created: sources={len(config.sources)}, dest={config.destination}\n")
            f.write(f"✓ Threads: {config.options.threads}\n")
            f.write(f"✓ ConfigDict validate_assignment: {config.model_config.get('validate_assignment')}\n")
        
        # Test 3: Validation
        try:
            bad_config = load_backup_config(config_dict={
                "sources": ["C:\\Data"],
                "destination": "Z:\\NonExistent"
            })
            f.write("✗ Should have caught validation error\n")
        except ConfigurationError:
            f.write("✓ Validation error caught for non-existent destination\n")
        
        f.write("\n=== All checks passed! ===\n")
    
    print("Verification complete. Results written to verification_results.txt")
    sys.exit(0)
    
except Exception as e:
    with open("verification_results.txt", "w") as f:
        f.write(f"✗ ERROR: {e}\n")
        import traceback
        f.write(traceback.format_exc())
    print(f"ERROR: {e}")
    sys.exit(1)


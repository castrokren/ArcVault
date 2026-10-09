"""
Example usage of the configuration loader module.

This demonstrates the various ways to load and validate backup configurations.
"""

import sys
from pathlib import Path

# Add parent directory to path for imports
sys.path.insert(0, str(Path(__file__).parent.parent.parent))

from agent.engine.config_loader import (
    load_backup_config,
    load_config_file,
    load_service_config,
    save_config_file,
    ConfigurationError
)
from agent.engine.config import BackupConfig, BackupOptions


def example_load_from_file():
    """Example: Load configuration from a JSON file."""
    print("\n=== Example 1: Load from JSON file ===")
    try:
        # Load configuration from file
        config = load_backup_config(file_path="example_backup_config.json")
        print(f"✓ Configuration loaded successfully")
        print(f"  Sources: {config.sources}")
        print(f"  Destination: {config.destination}")
        print(f"  Threads: {config.options.threads}")
        print(f"  Retries: {config.options.retries}")
    except ConfigurationError as e:
        print(f"✗ Failed to load configuration: {e}")


def example_load_from_dict():
    """Example: Load configuration from a dictionary."""
    print("\n=== Example 2: Load from dictionary ===")
    
    config_dict = {
        "sources": ["C:\\Data", "C:\\Projects"],
        "destination": "E:\\Backup",
        "options": {
            "threads": 8,
            "retries": 2,
            "verbose": True
        }
    }
    
    try:
        config = load_backup_config(config_dict=config_dict, validate=True)
        print(f"✓ Configuration validated successfully")
        print(f"  Sources: {config.sources}")
        print(f"  Destination: {config.destination}")
        print(f"  Threads: {config.options.threads}")
    except ConfigurationError as e:
        print(f"✗ Configuration validation failed: {e}")


def example_env_overrides():
    """Example: Using environment variable overrides."""
    print("\n=== Example 3: Environment variable overrides ===")
    
    import os
    
    # Set some environment variables
    os.environ["BACKUP_DESTINATION"] = "F:\\EnvBackup"
    os.environ["BACKUP_OPTIONS__THREADS"] = "32"
    os.environ["BACKUP_OPTIONS__RETRIES"] = "5"
    
    base_config = {
        "sources": ["C:\\Data"],
        "destination": "D:\\DefaultBackup",
        "options": {
            "threads": 8
        }
    }
    
    try:
        # Load with environment overrides
        config = load_backup_config(config_dict=base_config, apply_env=True)
        print(f"✓ Configuration with env overrides loaded")
        print(f"  Destination: {config.destination} (from env: BACKUP_DESTINATION)")
        print(f"  Threads: {config.options.threads} (from env: BACKUP_OPTIONS__THREADS)")
        print(f"  Retries: {config.options.retries} (from env: BACKUP_OPTIONS__RETRIES)")
    except ConfigurationError as e:
        print(f"✗ Failed: {e}")


def example_validation_errors():
    """Example: Handling validation errors."""
    print("\n=== Example 4: Validation errors ===")
    
    # Invalid config: missing destination
    invalid_config = {
        "sources": ["C:\\Data"]
        # destination is missing!
    }
    
    try:
        config = load_backup_config(config_dict=invalid_config)
        print("✗ Should have raised validation error")
    except ConfigurationError as e:
        print(f"✓ Validation caught error: {e}")
    
    # Invalid config: non-existent destination
    invalid_dest_config = {
        "sources": ["C:\\Data"],
        "destination": "Z:\\NonExistentPath"
    }
    
    try:
        config = load_backup_config(config_dict=invalid_dest_config)
        print("✗ Should have raised validation error for non-existent destination")
    except ConfigurationError as e:
        print(f"✓ Validation caught error: {e}")


def example_save_config():
    """Example: Save configuration to file."""
    print("\n=== Example 5: Save configuration ===")
    
    config_dict = {
        "sources": ["C:\\ImportantData"],
        "destination": "E:\\BackupArchive",
        "options": {
            "threads": 16,
            "retries": 3,
            "verbose": True
        }
    }
    
    try:
        # Create and validate config
        config = load_backup_config(config_dict=config_dict, validate=False)
        
        # Save to file
        output_file = "my_backup_config.json"
        save_config_file(config, output_file, format="json")
        print(f"✓ Configuration saved to {output_file}")
        
        # Read it back
        loaded = load_backup_config(file_path=output_file, validate=False)
        print(f"✓ Configuration reloaded successfully")
        print(f"  Sources: {loaded.sources}")
    except ConfigurationError as e:
        print(f"✗ Failed: {e}")


def example_service_config():
    """Example: Load service configuration."""
    print("\n=== Example 6: Service configuration ===")
    
    try:
        service_config = load_service_config()
        print(f"✓ Service configuration loaded")
        print(f"  Host: {service_config.get('host')}")
        print(f"  Port: {service_config.get('port')}")
        print(f"  Log Level: {service_config.get('log_level')}")
        print(f"  Max Concurrent Jobs: {service_config.get('max_concurrent_jobs')}")
    except Exception as e:
        print(f"✗ Failed: {e}")


def example_programmatic_config():
    """Example: Create configuration programmatically."""
    print("\n=== Example 7: Programmatic configuration ===")
    
    try:
        # Create using Pydantic models directly
        config = BackupConfig(
            sources=["C:\\MyData", "C:\\Projects"],
            destination="D:\\SafeBackup",
            options=BackupOptions(
                threads=24,
                retries=5,
                verbose=True,
                include_empty_dirs=True
            )
        )
        
        print(f"✓ Configuration created programmatically")
        print(f"  Sources: {config.sources}")
        print(f"  Destination: {config.destination}")
        print(f"  Threads: {config.options.threads}")
        
        # You can also modify after creation (because validate_assignment=True)
        config.options.threads = 32
        print(f"✓ Modified threads to: {config.options.threads}")
        
    except Exception as e:
        print(f"✗ Failed: {e}")


def main():
    """Run all examples."""
    print("=" * 70)
    print("Configuration Loader Examples")
    print("=" * 70)
    
    example_load_from_dict()
    example_env_overrides()
    example_validation_errors()
    example_programmatic_config()
    example_service_config()
    
    # These require actual files to exist
    # example_load_from_file()
    # example_save_config()
    
    print("\n" + "=" * 70)
    print("Examples completed!")
    print("=" * 70)


if __name__ == "__main__":
    main()


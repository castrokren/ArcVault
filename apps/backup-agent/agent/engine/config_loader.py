"""
Configuration file loading utilities for ArcVault Backup Agent.

This module centralizes all configuration loading logic with support for:
- JSON and YAML formats
- Environment variable overrides
- Default values
- Multiple configuration sources with priority order
- Robust error handling and validation
"""

import os
import json
import logging
from pathlib import Path
from typing import Dict, Any, Optional, Union, List

from .config import BackupConfig, BackupOptions, Credentials

logger = logging.getLogger(__name__)


class ConfigurationError(Exception):
    """Raised when configuration loading or validation fails."""
    pass


def load_json_config(file_path: Union[str, Path]) -> Dict[str, Any]:
    """
    Load configuration from a JSON file.
    
    Args:
        file_path: Path to the JSON configuration file
        
    Returns:
        Dictionary containing the configuration
        
    Raises:
        ConfigurationError: If file doesn't exist or JSON is invalid
    """
    path = Path(file_path)
    
    if not path.exists():
        raise ConfigurationError(f"Configuration file not found: {path}")
    
    if not path.is_file():
        raise ConfigurationError(f"Configuration path is not a file: {path}")
    
    try:
        with open(path, 'r', encoding='utf-8') as f:
            config = json.load(f)
        logger.info(f"Successfully loaded configuration from {path}")
        return config
    except json.JSONDecodeError as e:
        raise ConfigurationError(f"Invalid JSON in configuration file {path}: {e}")
    except Exception as e:
        raise ConfigurationError(f"Failed to read configuration file {path}: {e}")


def load_yaml_config(file_path: Union[str, Path]) -> Dict[str, Any]:
    """
    Load configuration from a YAML file.
    
    Args:
        file_path: Path to the YAML configuration file
        
    Returns:
        Dictionary containing the configuration
        
    Raises:
        ConfigurationError: If file doesn't exist, YAML is invalid, or PyYAML is not installed
    """
    path = Path(file_path)
    
    if not path.exists():
        raise ConfigurationError(f"Configuration file not found: {path}")
    
    if not path.is_file():
        raise ConfigurationError(f"Configuration path is not a file: {path}")
    
    try:
        import yaml
    except ImportError:
        raise ConfigurationError(
            "PyYAML is not installed. Install it with: pip install pyyaml"
        )
    
    try:
        with open(path, 'r', encoding='utf-8') as f:
            config = yaml.safe_load(f)
        logger.info(f"Successfully loaded configuration from {path}")
        return config if config else {}
    except yaml.YAMLError as e:
        raise ConfigurationError(f"Invalid YAML in configuration file {path}: {e}")
    except Exception as e:
        raise ConfigurationError(f"Failed to read configuration file {path}: {e}")


def load_config_file(file_path: Union[str, Path]) -> Dict[str, Any]:
    """
    Load configuration from a file (JSON or YAML, auto-detected by extension).
    
    Args:
        file_path: Path to the configuration file
        
    Returns:
        Dictionary containing the configuration
        
    Raises:
        ConfigurationError: If file format is unsupported or loading fails
    """
    path = Path(file_path)
    extension = path.suffix.lower()
    
    if extension == '.json':
        return load_json_config(path)
    elif extension in ('.yaml', '.yml'):
        return load_yaml_config(path)
    else:
        raise ConfigurationError(
            f"Unsupported configuration file format: {extension}. "
            "Supported formats: .json, .yaml, .yml"
        )


def apply_env_overrides(config: Dict[str, Any], prefix: str = "BACKUP_") -> Dict[str, Any]:
    """
    Apply environment variable overrides to configuration.
    
    Environment variables follow the pattern: {prefix}KEY=value
    Nested keys use double underscores: {prefix}OPTIONS__THREADS=8
    
    Args:
        config: Base configuration dictionary
        prefix: Prefix for environment variables (default: "BACKUP_")
        
    Returns:
        Configuration dictionary with environment overrides applied
        
    Example:
        BACKUP_DESTINATION=/mnt/backup
        BACKUP_OPTIONS__THREADS=16
        BACKUP_OPTIONS__RETRIES=3
    """
    result = config.copy()
    
    for env_key, env_value in os.environ.items():
        if not env_key.startswith(prefix):
            continue
        
        # Remove prefix and convert to lowercase
        key_path = env_key[len(prefix):].lower()
        
        # Handle nested keys (e.g., OPTIONS__THREADS -> options.threads)
        if '__' in key_path:
            parts = key_path.split('__')
            current = result
            
            # Navigate to the nested location
            for part in parts[:-1]:
                if part not in current:
                    current[part] = {}
                current = current[part]
            
            # Set the value (try to convert to appropriate type)
            final_key = parts[-1]
            current[final_key] = _parse_env_value(env_value)
        else:
            # Top-level key
            result[key_path] = _parse_env_value(env_value)
    
    return result


def _parse_env_value(value: str) -> Any:
    """
    Parse environment variable value to appropriate Python type.
    
    Attempts to convert strings to boolean, int, float, or list as appropriate.
    """
    # Boolean values
    if value.lower() in ('true', 'yes', '1', 'on'):
        return True
    if value.lower() in ('false', 'no', '0', 'off'):
        return False
    
    # Try integer
    try:
        return int(value)
    except ValueError:
        pass
    
    # Try float
    try:
        return float(value)
    except ValueError:
        pass
    
    # Check for comma-separated list
    if ',' in value:
        return [item.strip() for item in value.split(',')]
    
    # Return as string
    return value


def load_backup_config(
    file_path: Optional[Union[str, Path]] = None,
    config_dict: Optional[Dict[str, Any]] = None,
    apply_env: bool = True,
    validate: bool = True
) -> BackupConfig:
    """
    Load and validate a BackupConfig from various sources.
    
    Priority order (highest to lowest):
    1. Explicit config_dict parameter
    2. Configuration file (if file_path provided)
    3. Environment variables (if apply_env=True)
    
    Args:
        file_path: Optional path to configuration file
        config_dict: Optional pre-loaded configuration dictionary
        apply_env: Whether to apply environment variable overrides
        validate: Whether to validate the configuration (recommended)
        
    Returns:
        Validated BackupConfig instance
        
    Raises:
        ConfigurationError: If configuration is invalid or cannot be loaded
        
    Example:
        # Load from file
        config = load_backup_config("backup_job.json")
        
        # Load from dict with env overrides
        config = load_backup_config(config_dict={"sources": ["/data"], "destination": "/backup"})
        
        # Load from environment only
        config = load_backup_config()
    """
    # Start with empty config
    final_config: Dict[str, Any] = {}
    
    # Load from file if provided
    if file_path:
        try:
            final_config = load_config_file(file_path)
        except ConfigurationError as e:
            logger.error(f"Failed to load configuration file: {e}")
            raise
    
    # Override with explicit config_dict if provided
    if config_dict:
        final_config.update(config_dict)
    
    # Apply environment variable overrides
    if apply_env:
        final_config = apply_env_overrides(final_config)
    
    # Validate using Pydantic
    if validate:
        try:
            return BackupConfig.model_validate(final_config)
        except Exception as e:
            raise ConfigurationError(f"Configuration validation failed: {e}")
    else:
        # Create without validation (not recommended)
        return BackupConfig.model_construct(**final_config)


def load_service_config(
    config_file: str = "service_config.json",
    search_paths: Optional[List[Path]] = None
) -> Dict[str, Any]:
    """
    Load service-level configuration (host, port, etc.).
    
    Searches for configuration file in multiple locations:
    1. Custom search_paths (if provided)
    2. Current working directory
    3. Script/module directory
    4. Environment variables
    
    Args:
        config_file: Name of the service configuration file
        search_paths: Optional list of directories to search
        
    Returns:
        Service configuration dictionary with defaults applied
    """
    default_config = {
        "host": "0.0.0.0",
        "port": 8443,
        "log_level": "INFO",
        "max_concurrent_jobs": 5
    }
    
    # Define search locations
    if search_paths is None:
        search_paths = [
            Path.cwd(),
            Path(__file__).parent.parent,  # agent directory
        ]
    
    # Try to load from file
    for search_path in search_paths:
        config_path = search_path / config_file
        if config_path.exists():
            try:
                loaded_config = load_json_config(config_path)
                logger.info(f"Loaded service config from {config_path}")
                # Merge with defaults
                default_config.update(loaded_config)
                break
            except ConfigurationError as e:
                logger.warning(f"Failed to load service config from {config_path}: {e}")
    
    # Apply environment variable overrides for service config
    service_config = apply_env_overrides(default_config, prefix="AGENT_")
    
    # Also check legacy environment variables
    if "LISTEN_HOST" in os.environ:
        service_config["host"] = os.environ["LISTEN_HOST"]
    if "PORT" in os.environ:
        try:
            service_config["port"] = int(os.environ["PORT"])
        except ValueError:
            logger.warning(f"Invalid PORT value: {os.environ['PORT']}")
    
    return service_config


def save_config_file(
    config: Union[BackupConfig, Dict[str, Any]],
    file_path: Union[str, Path],
    format: str = "json"
) -> None:
    """
    Save configuration to a file.
    
    Args:
        config: Configuration to save (BackupConfig or dict)
        file_path: Destination file path
        format: Output format ("json" or "yaml")
        
    Raises:
        ConfigurationError: If saving fails
    """
    path = Path(file_path)
    
    # Convert BackupConfig to dict if needed
    if isinstance(config, BackupConfig):
        config_dict = config.model_dump(mode='json')
    else:
        config_dict = config
    
    try:
        if format.lower() == "json":
            with open(path, 'w', encoding='utf-8') as f:
                json.dump(config_dict, f, indent=2, ensure_ascii=False)
        elif format.lower() in ("yaml", "yml"):
            try:
                import yaml
            except ImportError:
                raise ConfigurationError("PyYAML not installed. Install with: pip install pyyaml")
            
            with open(path, 'w', encoding='utf-8') as f:
                yaml.safe_dump(config_dict, f, default_flow_style=False, allow_unicode=True)
        else:
            raise ConfigurationError(f"Unsupported format: {format}")
        
        logger.info(f"Successfully saved configuration to {path}")
    except Exception as e:
        raise ConfigurationError(f"Failed to save configuration to {path}: {e}")


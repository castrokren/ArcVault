# Configuration Management System

## Overview

The ArcVault Backup Agent uses a robust configuration management system built with Pydantic for validation and type safety. This document describes how to use the configuration system effectively.

## Key Features

- ✅ **Automatic Validation**: All configurations are validated using Pydantic models
- ✅ **Multiple Sources**: Load from JSON, YAML, dictionaries, or environment variables
- ✅ **Type Safety**: Strong typing with IDE support and autocomplete
- ✅ **Path Validation**: Validates that backup destinations exist and are writable
- ✅ **Secure Credentials**: Automatic encryption of passwords
- ✅ **Environment Overrides**: Override any configuration via environment variables
- ✅ **Assignment Validation**: Changes to config objects are validated in real-time

## Quick Start

### 1. Basic Usage - Load from Dictionary

```python
from agent.engine.config_loader import load_backup_config

config = load_backup_config(config_dict={
    "sources": ["C:\\Data", "C:\\Projects"],
    "destination": "E:\\Backup",
    "options": {
        "threads": 16,
        "retries": 3
    }
})

# Config is now validated and ready to use
print(f"Backing up {len(config.sources)} sources to {config.destination}")
```

### 2. Load from JSON File

```python
from agent.engine.config_loader import load_backup_config

# Load and validate from file
config = load_backup_config(file_path="backup_job.json")
```

### 3. Load from YAML File

```python
from agent.engine.config_loader import load_backup_config

# Works with .yaml or .yml files
config = load_backup_config(file_path="backup_job.yaml")
```

### 4. Environment Variable Overrides

```python
import os
from agent.engine.config_loader import load_backup_config

# Set environment variables
os.environ["BACKUP_DESTINATION"] = "F:\\Backup"
os.environ["BACKUP_OPTIONS__THREADS"] = "32"
os.environ["BACKUP_OPTIONS__RETRIES"] = "5"

# Load with env overrides applied
config = load_backup_config(
    config_dict={"sources": ["C:\\Data"], "destination": "D:\\Backup"},
    apply_env=True
)

# Destination will be F:\Backup (from env)
# Threads will be 32 (from env)
```

## Configuration Structure

### BackupConfig

Main configuration model for backup jobs:

```python
{
    "sources": ["C:\\Data", "C:\\Projects"],           # List of source paths
    "destination": "E:\\Backup",                       # Destination path
    "options": {                                       # Backup options (optional)
        "include_empty_dirs": true,                    # Copy empty directories
        "verbose": true,                               # Verbose output
        "report_extras": true,                         # Report extra files in destination
        "retries": 3,                                  # Number of retries (0-1000)
        "wait_seconds": 5,                             # Wait time between retries (0-60)
        "threads": 16,                                 # Number of threads (1-128)
        "copy_symlinks": false,                        # Copy symlink targets
        "remove_system_hidden": true                   # Remove system/hidden attributes
    },
    "source_credentials": {                            # Optional credentials for source
        "username": "user",
        "password": "pass",
        "domain": "WORKGROUP"
    },
    "destination_credentials": null                    # Optional credentials for destination
}
```

### BackupOptions

Detailed options for the backup operation:

| Field | Type | Default | Range | Description |
|-------|------|---------|-------|-------------|
| `include_empty_dirs` | bool | `true` | - | Copy empty directories |
| `verbose` | bool | `true` | - | Verbose logging |
| `report_extras` | bool | `true` | - | Report extra files in dest |
| `retries` | int | `1` | 0-1000 | Number of retry attempts |
| `wait_seconds` | int | `5` | 0-60 | Wait time between retries |
| `threads` | int | `12` | 1-128 | Multi-threading count |
| `copy_symlinks` | bool | `false` | - | Copy symlink targets |
| `remove_system_hidden` | bool | `true` | - | Remove system/hidden attrs |

## Validation Rules

### Sources
- ✅ Must be a non-empty list of strings
- ✅ Each source is trimmed of whitespace
- ✅ Empty strings are filtered out
- ❌ At least one valid source is required

### Destination
- ✅ Must be a non-empty string
- ✅ Trimmed of whitespace
- ✅ Must exist as a directory
- ✅ Must be writable (tested with temp file)
- ❌ Fails if path doesn't exist
- ❌ Fails if not writable

### Options
- ✅ All fields have sensible defaults
- ✅ Integer fields have min/max ranges enforced
- ✅ Invalid values are rejected at creation time

### Credentials
- ✅ Passwords are automatically encrypted
- ✅ Passwords are decrypted transparently when accessed
- ✅ Domain is optional

## Environment Variable Overrides

Environment variables follow this pattern:

```
BACKUP_<KEY>=<value>
BACKUP_<PARENT>__<CHILD>=<value>
```

### Examples

```bash
# Top-level keys
BACKUP_DESTINATION=E:\Backup
BACKUP_SOURCES=C:\Data,C:\Projects

# Nested keys (use double underscore)
BACKUP_OPTIONS__THREADS=32
BACKUP_OPTIONS__RETRIES=5
BACKUP_OPTIONS__VERBOSE=true

# Credentials
BACKUP_SOURCE_CREDENTIALS__USERNAME=admin
BACKUP_SOURCE_CREDENTIALS__PASSWORD=secret
BACKUP_SOURCE_CREDENTIALS__DOMAIN=CORP
```

### Type Conversion

Environment variables are automatically converted:

- `true`, `yes`, `1`, `on` → `True`
- `false`, `no`, `0`, `off` → `False`
- `123` → `123` (int)
- `3.14` → `3.14` (float)
- `a,b,c` → `["a", "b", "c"]` (list)
- Everything else → string

### Environment Variable Limitations & Edge Cases

While environment variables are convenient, they have some limitations:

#### 1. **Paths with Commas**

**Issue:** Commas are used to detect list values, which can cause problems with certain paths.

```bash
# ❌ This will be interpreted as a list
set BACKUP_DESTINATION=C:\Users\Smith, John\Backups
# Result: ['C:\Users\Smith', 'John\Backups']
```

**Solution:** Use a configuration file instead:

```json
{
  "destination": "C:\\Users\\Smith, John\\Backups"
}
```

**Note:** Paths with commas are extremely rare on Windows (invalid in most cases).

#### 2. **Leading Zeros in Numeric Values**

**Issue:** Numeric strings with leading zeros are converted to integers, losing the zeros.

```bash
# ❌ Leading zeros are lost
set JOB_ID=00123
# Result: 123 (integer)
```

**Solution:** Use strings in configuration files for values that need to preserve formatting:

```json
{
  "job_id": "00123"
}
```

**Note:** This typically only affects custom fields, not built-in backup configuration.

#### 3. **Complex Nested Structures**

**Issue:** Environment variables are not ideal for deeply nested or complex structures.

```bash
# ❌ Not practical for complex structures
set BACKUP_OPTIONS={"threads": 32, "verbose": true}
# Will be treated as a plain string, not parsed as JSON
```

**Solution:** Use JSON or YAML files for complex configurations:

```json
{
  "options": {
    "threads": 32,
    "verbose": true,
    "retries": 3,
    "wait_seconds": 5
  }
}
```

**Best Practice:** Use environment variables for simple overrides only (single values, not nested objects).

#### 4. **Special Characters in Values**

**Issue:** Some special characters in environment variables may require escaping.

```bash
# Windows PowerShell
$env:BACKUP_PASSWORD='P@ssw0rd!'

# Windows CMD (requires quotes)
set "BACKUP_PASSWORD=P@ssw0rd!"

# Bash/Linux
export BACKUP_PASSWORD='P@ssw0rd!'
```

**Recommendation:** For passwords and credentials, use configuration files with proper encryption.

### When to Use Environment Variables vs. Config Files

| Use Environment Variables For | Use Config Files For |
|-------------------------------|---------------------|
| Simple value overrides | Complete configurations |
| Environment-specific settings | Multiple backup jobs |
| CI/CD pipeline configuration | Complex nested structures |
| Temporary testing | Version-controlled configs |
| Single-value changes | Paths with special characters |
| Deployment automation | Team-shared configurations |

### Skipping Destination Validation

For CI/CD pipelines or scenarios where the destination is mounted later:

```bash
# Skip destination validation during config load
set BACKUP_SKIP_DESTINATION_CHECK=true
```

Then perform a pre-flight check before running the backup:

```python
config = load_backup_config(file_path="job.json")

# Later, when ready to run backup
accessible, error = config.validate_destination_accessible()
if not accessible:
    logger.error(f"Destination not ready: {error}")
    # Retry or defer backup
```

**Use cases:**
- Network shares mounted after service starts
- Cloud storage that needs authentication first
- Testing configurations without actual filesystem access

## API Reference

### `load_backup_config()`

Load and validate a BackupConfig from various sources.

```python
def load_backup_config(
    file_path: Optional[Union[str, Path]] = None,
    config_dict: Optional[Dict[str, Any]] = None,
    apply_env: bool = True,
    validate: bool = True
) -> BackupConfig
```

**Parameters:**
- `file_path`: Path to JSON/YAML config file
- `config_dict`: Dictionary with config data
- `apply_env`: Apply environment variable overrides
- `validate`: Validate the configuration (recommended)

**Returns:** Validated `BackupConfig` instance

**Raises:** `ConfigurationError` if loading/validation fails

**Priority Order:**
1. Explicit `config_dict` parameter (highest)
2. Configuration file (if `file_path` provided)
3. Environment variables (if `apply_env=True`)

### `load_config_file()`

Load configuration from a file (auto-detects JSON/YAML).

```python
def load_config_file(file_path: Union[str, Path]) -> Dict[str, Any]
```

### `load_service_config()`

Load service-level configuration (host, port, etc.).

```python
def load_service_config(
    config_file: str = "service_config.json",
    search_paths: Optional[List[Path]] = None
) -> Dict[str, Any]
```

**Returns:** Dictionary with service configuration and defaults

**Default Values:**
```python
{
    "host": "0.0.0.0",
    "port": 8443,
    "log_level": "INFO",
    "max_concurrent_jobs": 5
}
```

### `save_config_file()`

Save configuration to a file.

```python
def save_config_file(
    config: Union[BackupConfig, Dict[str, Any]],
    file_path: Union[str, Path],
    format: str = "json"
) -> None
```

**Parameters:**
- `config`: Configuration to save
- `file_path`: Destination file path
- `format`: Output format (`"json"` or `"yaml"`)

### `BackupConfig.validate_destination_accessible()`

Pre-flight check to verify destination is currently accessible.

```python
def validate_destination_accessible(
    self,
    raise_on_error: bool = False
) -> Tuple[bool, Optional[str]]
```

**Purpose:** Separate from initial validation, this method checks destination accessibility immediately before running a backup. Network shares and remote destinations can change state between configuration load time and backup execution time.

**Parameters:**
- `raise_on_error`: If `True`, raises `ValueError` on failure instead of returning `False`

**Returns:** 
- Tuple of `(is_accessible, error_message)`
- `(True, None)` if destination is accessible
- `(False, "error message")` if destination is not accessible

**Raises:**
- `ValueError`: If `raise_on_error=True` and destination is not accessible

**Example Usage:**

```python
# Load configuration (may skip validation with env var)
config = load_backup_config(file_path="job.json")

# Pre-flight check before backup
accessible, error = config.validate_destination_accessible()
if not accessible:
    logger.error(f"Destination not ready: {error}")
    # Implement retry logic or notify user
    schedule_retry()
else:
    # Safe to start backup
    run_backup(config)
```

**When to Use:**
- Before starting long-running backups
- In retry loops after transient failures
- When destination is a network share that might be unmounted
- After re-authentication with cloud storage
- In scheduled jobs that run hours after config load

**Comparison with Initial Validation:**

| Initial Validation | Pre-flight Check |
|--------------------|------------------|
| Runs at config load time | Runs before backup execution |
| Catches configuration errors | Catches runtime availability issues |
| Can be skipped with env var | Always checks current state |
| Fails fast on invalid config | Allows graceful retry logic |

## Error Handling

### ConfigurationError

All configuration-related errors raise `ConfigurationError`:

```python
from agent.engine.config_loader import load_backup_config, ConfigurationError

try:
    config = load_backup_config(file_path="backup.json")
except ConfigurationError as e:
    print(f"Configuration error: {e}")
    # Handle error appropriately
```

### Common Errors

1. **File Not Found**
   ```
   ConfigurationError: Configuration file not found: backup.json
   ```

2. **Invalid JSON/YAML**
   ```
   ConfigurationError: Invalid JSON in configuration file backup.json: ...
   ```

3. **Validation Failure**
   ```
   ConfigurationError: Configuration validation failed: 
   Destination does not exist: Z:\NonExistent
   ```

4. **Destination Not Writable**
   ```
   ConfigurationError: Configuration validation failed:
   Destination not writable: C:\Protected - [Errno 13] Permission denied
   ```

5. **Missing Required Fields**
   ```
   ConfigurationError: Configuration validation failed:
   Field required [type=missing, input_value={...}, input_type=dict]
   ```

## Best Practices

### 1. Always Validate

```python
# ✅ Good - validation enabled (default)
config = load_backup_config(config_dict=data)

# ❌ Bad - validation disabled (risky)
config = load_backup_config(config_dict=data, validate=False)
```

### 2. Use File-Based Configs for Jobs

```python
# ✅ Good - configuration in version control
config = load_backup_config(file_path="jobs/daily_backup.json")

# ❌ Bad - hardcoded configuration
config = load_backup_config(config_dict={
    "sources": ["C:\\Data"],
    "destination": "E:\\Backup"
})
```

### 3. Use Environment Variables for Deployment

```python
# ✅ Good - environment-specific overrides
# Production: BACKUP_DESTINATION=/mnt/backup-prod
# Staging: BACKUP_DESTINATION=/mnt/backup-staging
config = load_backup_config(file_path="job.json", apply_env=True)
```

### 4. Handle Errors Gracefully

```python
# ✅ Good - proper error handling
try:
    config = load_backup_config(file_path="job.json")
    run_backup(config)
except ConfigurationError as e:
    logger.error(f"Configuration error: {e}")
    send_alert(f"Backup failed: {e}")
    sys.exit(1)
```

### 5. Use Type Hints

```python
# ✅ Good - type hints for IDE support
from agent.engine.config import BackupConfig

def run_backup(config: BackupConfig) -> None:
    # IDE will provide autocomplete for config fields
    for source in config.sources:
        print(f"Backing up {source}")
```

## Migration Guide

### From Old Style (Direct Dict)

**Before:**
```python
cfg_dict = json.loads(config_json)
cfg = BackupConfig.model_validate(cfg_dict)
```

**After:**
```python
from agent.engine.config_loader import load_backup_config, ConfigurationError

try:
    cfg = load_backup_config(config_dict=json.loads(config_json))
except ConfigurationError as e:
    # Handle validation errors
    pass
```

### From File Loading

**Before:**
```python
with open("config.json") as f:
    cfg_dict = json.load(f)
cfg = BackupConfig.model_validate(cfg_dict)
```

**After:**
```python
from agent.engine.config_loader import load_backup_config

cfg = load_backup_config(file_path="config.json")
```

## Testing

### Unit Test Example

```python
import pytest
from agent.engine.config_loader import load_backup_config, ConfigurationError

def test_valid_config():
    config = load_backup_config(config_dict={
        "sources": ["C:\\Data"],
        "destination": "C:\\Temp"  # Must exist
    })
    assert len(config.sources) == 1
    assert config.destination == "C:\\Temp"

def test_invalid_destination():
    with pytest.raises(ConfigurationError):
        load_backup_config(config_dict={
            "sources": ["C:\\Data"],
            "destination": "Z:\\NonExistent"
        })

def test_env_overrides():
    import os
    os.environ["BACKUP_OPTIONS__THREADS"] = "32"
    
    config = load_backup_config(
        config_dict={
            "sources": ["C:\\Data"],
            "destination": "C:\\Temp"
        },
        apply_env=True
    )
    assert config.options.threads == 32
```

## Troubleshooting

### Issue: "Destination not writable"

**Cause:** The destination directory exists but the current user lacks write permissions.

**Solution:**
- Check directory permissions
- Run with appropriate privileges
- Choose a different destination

### Issue: Environment variables not applied

**Cause:** Wrong prefix or naming convention.

**Solution:**
- Use `BACKUP_` prefix (not `AGENT_`)
- Use double underscores for nested keys: `BACKUP_OPTIONS__THREADS`
- Check case sensitivity

### Issue: "Invalid JSON in configuration file"

**Cause:** Malformed JSON file.

**Solution:**
- Validate JSON using a linter (jsonlint.com)
- Check for trailing commas
- Ensure proper quote escaping

## See Also

- [config.py](./config.py) - Configuration models
- [config_loader.py](./config_loader.py) - Loading utilities
- [config_usage_example.py](./config_usage_example.py) - Usage examples
- [example_backup_config.json](./example_backup_config.json) - Example config file


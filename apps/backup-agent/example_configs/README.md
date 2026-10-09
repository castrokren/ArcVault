# Example Backup Configurations

This directory contains example configuration files for common backup scenarios. Copy and modify these templates for your specific needs.

## Available Templates

### 1. `simple_local_backup.json`
**Use Case:** Basic backup from one folder to a local drive

**Features:**
- Single source directory
- Local destination
- Standard options
- No credentials required

**Best For:**
- Personal document backups
- Single-folder backups
- Local disk-to-disk backups

---

### 2. `multiple_sources_backup.json`
**Use Case:** Back up multiple folders to one destination

**Features:**
- Multiple source directories
- Single destination
- Higher thread count for better performance
- Increased retry attempts

**Best For:**
- Comprehensive system backups
- Multiple project folders
- User profile backups (Documents, Pictures, etc.)

---

### 3. `network_share_backup.json`
**Use Case:** Backup to a network location (NAS, file server)

**Features:**
- Network destination (UNC path)
- Destination credentials
- Higher retry count for network reliability
- Moderate thread count to avoid network congestion

**Best For:**
- NAS backups
- File server backups
- Shared network storage
- Domain-joined computers

---

### 4. `high_performance_backup.json`
**Use Case:** Fast backups on high-speed storage

**Features:**
- Maximum threads (128) for best speed
- Minimal logging overhead
- Fast retry settings
- Optimized for throughput

**Best For:**
- Large file transfers
- SSD to SSD backups
- High-performance servers
- Time-critical backups

---

### 5. `conservative_reliable_backup.json`
**Use Case:** Backups over slow or unreliable networks

**Features:**
- High retry count (50 attempts)
- Long wait time between retries (30 seconds)
- Low thread count to avoid overwhelming network
- Verbose logging for troubleshooting

**Best For:**
- VPN connections
- Slow network links
- Unreliable Wi-Fi
- Remote office backups

---

### 6. `minimal_backup.json`
**Use Case:** Simplest possible configuration

**Features:**
- Only required fields (sources, destination)
- All options use defaults
- Easiest to understand

**Best For:**
- Quick testing
- Learning the configuration format
- Simple backup needs

---

## How to Use These Templates

### Step 1: Choose a Template

Pick the template that best matches your backup scenario.

### Step 2: Copy the Template

```bash
# Copy to your working directory
copy example_configs\simple_local_backup.json my_backup_config.json
```

### Step 3: Edit for Your Environment

Open the file and update:
- **sources**: Replace with your actual source directories
- **destination**: Replace with your backup destination
- **credentials**: Add username/password if backing up to network shares
- **options**: Adjust threads, retries, etc. as needed

### Step 4: Test the Configuration

```bash
# Load and validate without running
python -c "from agent.engine.config_loader import load_backup_config; load_backup_config(file_path='my_backup_config.json')"
```

### Step 5: Run Your Backup

Use the configuration with the backup agent API or CLI.

---

## Configuration Options Reference

### Required Fields

| Field | Type | Description |
|-------|------|-------------|
| `sources` | array | List of source directories to back up |
| `destination` | string | Destination directory for backups |

### Optional Fields

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `options` | object | (see below) | Backup behavior options |
| `source_credentials` | object | null | Credentials for source location |
| `destination_credentials` | object | null | Credentials for destination location |

### Options Object

| Field | Type | Default | Range | Description |
|-------|------|---------|-------|-------------|
| `include_empty_dirs` | boolean | true | - | Copy empty directories |
| `verbose` | boolean | true | - | Verbose logging |
| `report_extras` | boolean | true | - | Report extra files in destination |
| `retries` | integer | 1 | 0-1000 | Number of retry attempts |
| `wait_seconds` | integer | 5 | 0-60 | Wait time between retries |
| `threads` | integer | 12 | 1-128 | Multi-threading count |
| `copy_symlinks` | boolean | false | - | Copy symlink targets |
| `remove_system_hidden` | boolean | true | - | Remove system/hidden attributes |

### Credentials Object

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `username` | string | Yes | Username for authentication |
| `password` | string | Yes | Password (automatically encrypted) |
| `domain` | string | No | Domain/workgroup name |

---

## Path Format Guidelines

### Windows Paths

```json
"sources": ["C:\\Users\\Documents"],
"destination": "D:\\Backups"
```

**Note:** Use double backslashes (`\\`) in JSON to escape the backslash character.

### UNC Paths (Network Shares)

```json
"destination": "\\\\SERVER\\Share\\Folder"
```

**Note:** Use four backslashes (`\\\\`) at the start for UNC paths in JSON.

---

## Environment Variable Overrides

You can override any configuration value using environment variables:

```bash
# Override destination
set BACKUP_DESTINATION=E:\AlternateBackup

# Override nested options
set BACKUP_OPTIONS__THREADS=32
set BACKUP_OPTIONS__RETRIES=10
```

See `CONFIG_MANAGEMENT.md` for complete environment variable documentation.

---

## Tips and Best Practices

### 1. Start Small
Begin with `minimal_backup.json` and add options as needed.

### 2. Test First
Always test new configurations with a small source directory first.

### 3. Monitor Performance
Watch the first few backups to tune thread count and retry settings.

### 4. Secure Credentials
- Never commit configuration files with passwords to version control
- Consider using environment variables for passwords: `BACKUP_DESTINATION_CREDENTIALS__PASSWORD`

### 5. Document Your Changes
Add comments to your configurations (fields starting with `_` are ignored):

```json
{
  "_comment": "Production backup - runs daily at 2 AM",
  "_last_modified": "2025-12-02",
  "sources": ["C:\\Data"],
  "destination": "E:\\Backup"
}
```

### 6. Version Control
Keep your configurations in version control (without passwords):

```json
{
  "sources": ["C:\\Data"],
  "destination": "E:\\Backup",
  "destination_credentials": {
    "username": "backup_user",
    "password": "${BACKUP_PASSWORD}",
    "domain": "COMPANY"
  }
}
```

---

## Troubleshooting

### "Destination does not exist"
- Verify the destination path exists
- Check that drive letters are correct
- For network shares, ensure they're mounted/accessible

### "Destination not writable"
- Check folder permissions
- Ensure you have write access
- For network shares, verify credentials are correct

### "Configuration validation failed"
- Check JSON syntax (use a JSON validator)
- Verify all required fields are present
- Check that numeric values are in valid ranges

### Slow Performance
- Try increasing `threads` (up to 128)
- Reduce `retries` for faster failure detection
- Disable `verbose` and `report_extras`

### Network Failures
- Increase `retries` (try 20-50)
- Increase `wait_seconds` (try 15-30)
- Reduce `threads` (try 4-8)

---

## Getting Help

- **Full Documentation**: See `agent/engine/CONFIG_MANAGEMENT.md`
- **API Reference**: See API documentation in CONFIG_MANAGEMENT.md
- **Examples**: See `agent/engine/config_usage_example.py`

---

**Last Updated:** December 2, 2025
**Version:** 1.0


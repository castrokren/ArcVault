# Quick Start: Configuration Guide

Get started with ArcVault Backup Agent configurations in 5 minutes! This guide shows you how to quickly set up and run backup jobs.

## 🚀 30-Second Start

### 1. Copy a Template

```powershell
# Choose the template that fits your needs
copy example_configs\simple_local_backup.json my_backup.json
```

### 2. Edit for Your Paths

Open `my_backup.json` and update:
```json
{
  "sources": ["C:\\Your\\Source\\Folder"],
  "destination": "D:\\Your\\Backup\\Location"
}
```

### 3. Test It

```powershell
python -m agent.run_agent
# Then use the API to submit your backup job
```

Done! Your backup is configured.

---

## 📋 Which Template Should I Use?

| Your Situation | Use This Template | File Name |
|----------------|-------------------|-----------|
| Basic local backup | Simple Local Backup | `simple_local_backup.json` |
| Multiple folders to one place | Multiple Sources | `multiple_sources_backup.json` |
| Backup to network drive/NAS | Network Share | `network_share_backup.json` |
| Need maximum speed | High Performance | `high_performance_backup.json` |
| Slow/unreliable connection | Conservative/Reliable | `conservative_reliable_backup.json` |
| Just testing/learning | Minimal | `minimal_backup.json` |

---

## 🎯 Common Scenarios

### Scenario 1: Backup Documents to External Drive

**Goal:** Back up your Documents folder to an external USB drive every night.

**Template:** `simple_local_backup.json`

**Configuration:**
```json
{
  "sources": ["C:\\Users\\YourName\\Documents"],
  "destination": "E:\\Backups\\Documents",
  "options": {
    "threads": 12,
    "retries": 3
  }
}
```

**Why these settings:**
- Default threads (12) - Good balance for local drives
- 3 retries - Handles temporary file locks

---

### Scenario 2: Backup Multiple Folders to NAS

**Goal:** Back up Documents, Pictures, and Projects to a network NAS.

**Template:** `network_share_backup.json`

**Configuration:**
```json
{
  "sources": [
    "C:\\Users\\YourName\\Documents",
    "C:\\Users\\YourName\\Pictures",
    "C:\\Projects"
  ],
  "destination": "\\\\NAS\\Backups\\MyPC",
  "options": {
    "threads": 8,
    "retries": 10,
    "wait_seconds": 15
  },
  "destination_credentials": {
    "username": "nas_user",
    "password": "your_password",
    "domain": "WORKGROUP"
  }
}
```

**Why these settings:**
- 8 threads - Moderate, won't overwhelm network
- 10 retries - Network can be unreliable
- 15 second wait - Give network time to recover

---

### Scenario 3: Fast Backup Between SSDs

**Goal:** Mirror a project folder from one SSD to another as fast as possible.

**Template:** `high_performance_backup.json`

**Configuration:**
```json
{
  "sources": ["D:\\ActiveProjects"],
  "destination": "E:\\ProjectMirror",
  "options": {
    "threads": 128,
    "retries": 1,
    "verbose": false
  }
}
```

**Why these settings:**
- 128 threads - Maximum parallelism for SSDs
- 1 retry - Fast failure detection
- Verbose off - Reduces logging overhead

---

### Scenario 4: Backup Over VPN

**Goal:** Reliable backup to a remote location over VPN.

**Template:** `conservative_reliable_backup.json`

**Configuration:**
```json
{
  "sources": ["C:\\Data"],
  "destination": "\\\\RemoteServer\\Backups",
  "options": {
    "threads": 4,
    "retries": 50,
    "wait_seconds": 30
  },
  "destination_credentials": {
    "username": "vpn_user",
    "password": "secure_password",
    "domain": "COMPANY"
  }
}
```

**Why these settings:**
- 4 threads - Won't overwhelm VPN bandwidth
- 50 retries - VPN can disconnect/reconnect
- 30 second wait - Give VPN time to re-establish

---

## 🔧 Customizing Your Configuration

### Adjusting Performance

#### Speed Up Backups

```json
{
  "options": {
    "threads": 64,        // ↑ Increase threads
    "verbose": false,     // ↓ Reduce logging
    "report_extras": false
  }
}
```

#### Make More Reliable

```json
{
  "options": {
    "threads": 4,         // ↓ Reduce threads
    "retries": 20,        // ↑ More retries
    "wait_seconds": 20    // ↑ Longer wait
  }
}
```

### Adding Network Credentials

```json
{
  "destination_credentials": {
    "username": "your_username",
    "password": "your_password",
    "domain": "WORKGROUP"
  }
}
```

**Note:** Passwords are automatically encrypted when the configuration is loaded.

### Multiple Sources

```json
{
  "sources": [
    "C:\\Folder1",
    "C:\\Folder2",
    "D:\\Folder3",
    "E:\\Folder4"
  ]
}
```

**Tip:** Each source will be backed up sequentially to the destination.

---

## 🌍 Environment Variables

Override any setting without changing the config file:

### Windows PowerShell

```powershell
# Change destination
$env:BACKUP_DESTINATION = "F:\AlternateBackup"

# Change options
$env:BACKUP_OPTIONS__THREADS = "32"
$env:BACKUP_OPTIONS__RETRIES = "10"
```

### Windows CMD

```cmd
set BACKUP_DESTINATION=F:\AlternateBackup
set BACKUP_OPTIONS__THREADS=32
set BACKUP_OPTIONS__RETRIES=10
```

### Testing Different Settings

```powershell
# Test with 64 threads
$env:BACKUP_OPTIONS__THREADS = "64"
# Run backup

# Test with 16 threads
$env:BACKUP_OPTIONS__THREADS = "16"
# Run backup again
```

---

## ✅ Validation Checklist

Before running your backup, verify:

- [ ] Source paths exist and are readable
- [ ] Destination path exists and is writable
- [ ] Network paths are accessible (if using UNC paths)
- [ ] Credentials are correct (if using network shares)
- [ ] Enough disk space at destination
- [ ] Paths use double backslashes in JSON (`C:\\Data`)

### Quick Validation

```python
from agent.engine.config_loader import load_backup_config, ConfigurationError

try:
    config = load_backup_config(file_path="my_backup.json")
    print("✅ Configuration is valid!")
    print(f"Sources: {config.sources}")
    print(f"Destination: {config.destination}")
except ConfigurationError as e:
    print(f"❌ Configuration error: {e}")
```

---

## 🛠️ Troubleshooting

### Error: "Destination does not exist"

**Problem:** The destination folder doesn't exist.

**Solution:**
```powershell
# Create the destination folder
mkdir "D:\Backups"
```

---

### Error: "Destination not writable"

**Problem:** You don't have permission to write to the destination.

**Solutions:**
1. Check folder permissions
2. Run as administrator (if needed)
3. For network shares, verify credentials
4. Check if drive is full

---

### Error: "Invalid JSON"

**Problem:** JSON syntax error in your configuration file.

**Common Mistakes:**
```json
// ❌ Wrong - Trailing comma
{
  "sources": ["C:\\Data"],
  "destination": "D:\\Backup",  ← Remove this comma
}

// ✅ Correct - No trailing comma
{
  "sources": ["C:\\Data"],
  "destination": "D:\\Backup"
}
```

**Solution:** Use a JSON validator (jsonlint.com) to check your file.

---

### Backup is Slow

**Try:**
1. Increase threads: `"threads": 64`
2. Disable verbose: `"verbose": false`
3. Check network speed (for network backups)
4. Check if antivirus is scanning files

---

### Network Backup Keeps Failing

**Try:**
1. Increase retries: `"retries": 20`
2. Increase wait time: `"wait_seconds": 30`
3. Decrease threads: `"threads": 4`
4. Verify network connection is stable
5. Check credentials are correct

---

## 📚 Next Steps

### Learn More

- **Full Documentation:** `agent/engine/CONFIG_MANAGEMENT.md`
- **All Templates:** `example_configs/README.md`
- **API Reference:** See CONFIG_MANAGEMENT.md

### Advanced Topics

- Using environment variables for secrets
- Scheduling automated backups
- Monitoring backup job status
- Creating backup job templates

### Get Help

- Check the troubleshooting section above
- Review example configurations
- See CONFIG_MANAGEMENT.md for detailed documentation

---

## 🎓 Configuration Cheat Sheet

### Minimal Configuration
```json
{
  "sources": ["C:\\Data"],
  "destination": "D:\\Backup"
}
```

### With Options
```json
{
  "sources": ["C:\\Data"],
  "destination": "D:\\Backup",
  "options": {
    "threads": 16,
    "retries": 5
  }
}
```

### With Credentials
```json
{
  "sources": ["C:\\Data"],
  "destination": "\\\\NAS\\Backup",
  "destination_credentials": {
    "username": "user",
    "password": "pass",
    "domain": "WORKGROUP"
  }
}
```

### Full Configuration
```json
{
  "sources": ["C:\\Data", "C:\\Projects"],
  "destination": "\\\\NAS\\Backup",
  "options": {
    "include_empty_dirs": true,
    "verbose": true,
    "report_extras": true,
    "retries": 10,
    "wait_seconds": 15,
    "threads": 8,
    "copy_symlinks": false,
    "remove_system_hidden": true
  },
  "destination_credentials": {
    "username": "backup_user",
    "password": "secure_password",
    "domain": "COMPANY"
  }
}
```

---

## 🔐 Security Tips

1. **Don't commit passwords to version control**
   ```json
   {
     "destination_credentials": {
       "password": "${BACKUP_PASSWORD}"  // Use env var
     }
   }
   ```

2. **Use environment variables for sensitive data**
   ```powershell
   $env:BACKUP_DESTINATION_CREDENTIALS__PASSWORD = "secret"
   ```

3. **Secure your configuration files**
   ```powershell
   # Restrict access to config file
   icacls my_backup.json /inheritance:r /grant:r "$env:USERNAME:(R)"
   ```

4. **Passwords are automatically encrypted**
   - When loaded, passwords are encrypted in memory
   - Encryption happens transparently
   - No action needed on your part

---

**Last Updated:** December 2, 2025
**Version:** 1.0
**For:** ArcVault Backup Agent v2.0+

---

**Ready to start?** Choose a template from `example_configs/` and customize it for your needs!


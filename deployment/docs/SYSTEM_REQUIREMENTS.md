# 💻 System Requirements

Detailed system requirements for deploying ArcVault.

## Hardware Requirements

### Minimum Requirements

| Component | Specification |
|-----------|---------------|
| **CPU** | 2 cores @ 2.0 GHz |
| **RAM** | 4 GB |
| **Disk Space** | 10 GB free |
| **Network** | 100 Mbps |

**Use Cases:**
- Small deployments (1-5 backup jobs)
- Testing and development
- Low-frequency backups

### Recommended Requirements

| Component | Specification |
|-----------|---------------|
| **CPU** | 4+ cores @ 2.5+ GHz |
| **RAM** | 8+ GB |
| **Disk Space** | 50+ GB free |
| **Network** | 1 Gbps |

**Use Cases:**
- Production deployments
- Multiple concurrent backup jobs
- Large file transfers
- High-frequency backups

### Enterprise Requirements

| Component | Specification |
|-----------|---------------|
| **CPU** | 8+ cores @ 3.0+ GHz |
| **RAM** | 16+ GB |
| **Disk Space** | 500+ GB free (depends on backup volume) |
| **Network** | 10 Gbps |
| **Storage** | NVMe SSD for application, HDD/NAS for backups |

**Use Cases:**
- Large-scale deployments
- 24/7 operation
- Mission-critical backups
- Multiple agents

---

## Software Requirements

### Operating System

**Supported Windows Versions:**
- ✅ Windows 10 (20H2 or later)
- ✅ Windows 11 (all versions)
- ✅ Windows Server 2016
- ✅ Windows Server 2019
- ✅ Windows Server 2022

**Architecture:**
- ✅ x64 (64-bit) - Required
- ❌ x86 (32-bit) - Not supported

### PowerShell

- **Version**: PowerShell 5.1 or later
- **Execution Policy**: RemoteSigned or Unrestricted (for deployment scripts)

**Check PowerShell Version:**
```powershell
$PSVersionTable.PSVersion
```

**Set Execution Policy:**
```powershell
Set-ExecutionPolicy -ExecutionPolicy RemoteSigned -Scope LocalMachine
```

### Python

- **Version**: Python 3.9 or later (3.11+ recommended)
- **Distribution**: Official Python from python.org
- **Installation**: Can be installed automatically by deployment script

**Check Python Version:**
```powershell
python --version
```

**Installation Notes:**
- Must be added to PATH
- pip must be available
- virtualenv support required

### .NET Framework

- **Version**: .NET Framework 4.7.2 or later
- **Purpose**: Windows service management (pywin32)
- **Included with**: Windows 10/11, Server 2019+

**Check .NET Version:**
```powershell
Get-ItemProperty 'HKLM:\SOFTWARE\Microsoft\NET Framework Setup\NDP\v4\Full' | 
    Select-Object Version, Release
```

---

## Network Requirements

### Port Requirements

| Port | Protocol | Direction | Purpose |
|------|----------|-----------|---------|
| 8443 | TCP | Inbound | Agent API (default, configurable) |
| 443 | TCP | Outbound | External API calls (optional) |
| 80/443 | TCP | Outbound | Package downloads (during install) |

**Custom Ports:**
- Any unused port between 1024-65535
- Check availability: `netstat -ano | findstr :PORT`

### Firewall Requirements

**Windows Firewall:**
- Allow inbound connections on agent port
- Created automatically by deployment script (optional)

**Network Firewall:**
- Allow internal communication between agents and controller
- Allow outbound HTTPS for updates (optional)

**Example Firewall Rule:**
```powershell
New-NetFirewallRule `
    -DisplayName "ArcVault Agent" `
    -Direction Inbound `
    -Protocol TCP `
    -LocalPort 8443 `
    -Action Allow `
    -Profile Domain,Private
```

### TLS/SSL Requirements

- **TLS Version**: TLS 1.2 or later
- **Cipher Suites**: Modern cipher suites supported
- **Certificates**: Self-signed (internal) or trusted (external)

### Network Bandwidth

**Minimum:**
- 100 Mbps for small backup volumes
- Adequate for incremental backups

**Recommended:**
- 1 Gbps for production use
- 10 Gbps for large-scale deployments

**Bandwidth Calculation:**
```
Daily backup volume ÷ backup window = required bandwidth
Example: 100 GB ÷ 2 hours = ~115 Mbps minimum
```

---

## Storage Requirements

### Installation Storage

| Component | Size |
|-----------|------|
| Application Files | ~100 MB |
| Python Environment | ~500 MB |
| Logs (typical) | 10-100 MB/day |
| Configuration | <1 MB |
| **Total (typical)** | **1-2 GB** |

### Data Storage

**Backup Destination:**
- Depends on backup volume
- Plan for retention period
- Consider compression ratios

**Example Calculation:**
```
Source Data: 1 TB
Compression: 50% (typical)
Retention: 30 days
Daily Change: 5%

Storage Required:
- Full backup: 500 GB (compressed)
- Incremental (29 days): 500 GB × 5% × 29 = 725 GB
- Total: ~1.2 TB
```

### I/O Requirements

**Disk IOPS:**
- Minimum: 100 IOPS
- Recommended: 500+ IOPS
- Enterprise: 1000+ IOPS

**Disk Types:**
- **SSD**: Best for application files and active backups
- **HDD**: Acceptable for archive/long-term storage
- **NAS/SAN**: Supported for backup destinations

---

## Permission Requirements

### Installation Permissions

**Required During Installation:**
- ✅ Local Administrator rights
- ✅ Ability to install Windows services
- ✅ File system write access to Program Files
- ✅ Registry write access (for service)

### Runtime Permissions

**Service Account:**
- Read/write access to installation directory
- Read/write access to data directory
- Network access to backup destinations
- File system permissions for backup sources

**Recommended:**
- Run as dedicated service account
- Grant minimal necessary permissions
- Use NTFS permissions for access control

---

## Compatibility Matrix

### Python Packages

| Package | Minimum Version | Recommended Version |
|---------|----------------|---------------------|
| FastAPI | 0.68.0 | Latest stable |
| uvicorn | 0.15.0 | Latest stable |
| PyJWT | 2.0.0 | Latest stable |
| cryptography | 3.4.8 | Latest stable |
| PyQt5 | 5.15.0 | 5.15.9 |
| pywin32 | 300 | Latest stable |
| PyInstaller | 5.0.0 | Latest stable |

### Browser Support (for Web UI)

**Supported Browsers:**
- ✅ Chrome/Edge 90+
- ✅ Firefox 88+
- ✅ Safari 14+

### Storage Systems

**Supported:**
- ✅ Local NTFS volumes
- ✅ Windows network shares (SMB/CIFS)
- ✅ NAS devices (QNAP, Synology, etc.)
- ✅ SAN storage
- ✅ Cloud storage (via mounted drives)

**Not Tested:**
- ⚠️ FAT32/exFAT (not recommended)
- ⚠️ ReFS (should work, not extensively tested)
- ⚠️ Direct cloud API integration

---

## Performance Considerations

### CPU Scaling

**Backup Performance vs CPU:**
- 2 cores: ~50 MB/s throughput
- 4 cores: ~150 MB/s throughput
- 8 cores: ~300 MB/s throughput

*Actual performance varies based on disk I/O and compression*

### Memory Usage

**Typical Memory Usage:**
- Base service: 100-200 MB
- Per backup job: 50-100 MB
- GUI application: 150-300 MB

**Calculation:**
```
Base: 200 MB
Jobs: 100 MB × concurrent jobs
Buffer: 200 MB
Example (3 jobs): 200 + (100 × 3) + 200 = 700 MB
```

### Disk I/O

**I/O Patterns:**
- Primarily sequential reads/writes
- Burst activity during backups
- Low I/O during idle periods

**Optimization:**
- Use separate disks for source and destination
- Enable write caching (with UPS)
- Use RAID for redundancy

### Network Throughput

**Factors Affecting Throughput:**
- Network bandwidth
- Disk I/O speed
- CPU for compression/encryption
- File size and count
- Protocol overhead

**Typical Throughput:**
- 1 Gbps network: 80-100 MB/s
- 10 Gbps network: 400-800 MB/s

---

## Virtual Environment Support

### VMware

- ✅ ESXi 6.7+
- ✅ Workstation 15+
- ⚠️ Ensure VMware Tools installed
- ⚠️ Allocate adequate resources

### Hyper-V

- ✅ Windows Server 2016+
- ✅ Windows 10/11 Hyper-V
- ⚠️ Install Integration Services
- ⚠️ Use Generation 2 VMs

### Other Virtualization

- ✅ VirtualBox 6.0+
- ✅ KVM/QEMU (Windows guest)
- ⚠️ Performance may vary

**Recommendations:**
- Allocate at least 2 vCPUs
- Use thick-provisioned disks
- Enable paravirtualized I/O
- Snapshot before major changes

---

## Cloud Platform Support

### Azure

- ✅ Azure VMs (Windows)
- ✅ B-series, D-series recommended
- ⚠️ Consider egress costs for backups

### AWS

- ✅ EC2 instances (Windows)
- ✅ t3, m5 instance types recommended
- ⚠️ Use EBS volumes with adequate IOPS

### Google Cloud

- ✅ Compute Engine (Windows)
- ✅ n1, n2 machine types recommended
- ⚠️ Consider network pricing

---

## Dependency Checklist

Use this checklist to verify all requirements before installation:

### Pre-Installation

- [ ] Windows 10/11 or Server 2016+
- [ ] PowerShell 5.1+
- [ ] Administrator privileges
- [ ] 10+ GB free disk space
- [ ] Python 3.9+ (or will be installed)
- [ ] .NET Framework 4.7.2+
- [ ] Port 8443 available (or alternate)

### Network

- [ ] Firewall rules configured
- [ ] Network connectivity verified
- [ ] DNS resolution working
- [ ] Backup destinations accessible

### Storage

- [ ] Adequate disk space for backups
- [ ] NTFS file system (for install)
- [ ] Appropriate permissions
- [ ] Backup tested (read/write)

### Security

- [ ] SSL certificates planned
- [ ] JWT keys will be generated
- [ ] Service account planned (optional)
- [ ] Firewall rules documented

---

## Upgrading System

If your system doesn't meet requirements:

### Upgrade RAM
```powershell
# Check current RAM
Get-CimInstance Win32_PhysicalMemory | 
    Measure-Object -Property Capacity -Sum |
    ForEach-Object { [math]::Round($_.Sum / 1GB, 2) }
```

### Check Disk Space
```powershell
# Check available space
Get-PSDrive C | Select-Object Used, Free
```

### Upgrade Python
```powershell
# Download from: https://www.python.org/downloads/
# Or use chocolatey:
choco install python39 -y
```

### Upgrade PowerShell
```powershell
# PowerShell 7 (optional, 5.1 included with Windows 10+)
winget install Microsoft.PowerShell
```

---

## Validation Script

Run this script to validate system requirements:

```powershell
# Save as Test-SystemRequirements.ps1

Write-Host "`n=== System Requirements Check ===" -ForegroundColor Cyan

# OS Version
$os = Get-CimInstance Win32_OperatingSystem
Write-Host "`nOS: $($os.Caption) $($os.Version)"
if ($os.Version -ge "10.0.17763") {
    Write-Host "✓ OS version supported" -ForegroundColor Green
} else {
    Write-Host "✗ OS version too old" -ForegroundColor Red
}

# PowerShell Version
Write-Host "`nPowerShell: $($PSVersionTable.PSVersion)"
if ($PSVersionTable.PSVersion.Major -ge 5) {
    Write-Host "✓ PowerShell version OK" -ForegroundColor Green
} else {
    Write-Host "✗ PowerShell 5.1+ required" -ForegroundColor Red
}

# Python
try {
    $pythonVersion = & python --version 2>&1
    Write-Host "`nPython: $pythonVersion"
    if ($pythonVersion -match "3\.([9]|[1-9][0-9])\.") {
        Write-Host "✓ Python version OK" -ForegroundColor Green
    } else {
        Write-Host "⚠ Python 3.9+ recommended" -ForegroundColor Yellow
    }
} catch {
    Write-Host "`n⚠ Python not found (will be installed)" -ForegroundColor Yellow
}

# Disk Space
$drive = Get-PSDrive C
$freeGB = [math]::Round($drive.Free / 1GB, 2)
Write-Host "`nDisk Space: $freeGB GB free"
if ($freeGB -gt 10) {
    Write-Host "✓ Sufficient disk space" -ForegroundColor Green
} else {
    Write-Host "✗ Insufficient disk space (<10 GB)" -ForegroundColor Red
}

# RAM
$ram = (Get-CimInstance Win32_ComputerSystem).TotalPhysicalMemory / 1GB
Write-Host "`nRAM: $([math]::Round($ram, 2)) GB"
if ($ram -ge 4) {
    Write-Host "✓ Sufficient RAM" -ForegroundColor Green
} else {
    Write-Host "✗ Insufficient RAM (<4 GB)" -ForegroundColor Red
}

# Port availability
$port = 8443
$portAvailable = -not (Get-NetTCPConnection -LocalPort $port -ErrorAction SilentlyContinue)
Write-Host "`nPort $port`: $(if ($portAvailable) { 'Available' } else { 'In use' })"
if ($portAvailable) {
    Write-Host "✓ Port available" -ForegroundColor Green
} else {
    Write-Host "⚠ Port in use (can use alternate)" -ForegroundColor Yellow
}

Write-Host "`n=== End Check ===" -ForegroundColor Cyan
```

---

**Ready to install?** Proceed to [DEPLOYMENT_GUIDE.md](DEPLOYMENT_GUIDE.md)


# ⚡ Quick Start Guide

Get ArcVault up and running in minutes!

## 🎯 Prerequisites

- ✅ Windows 10/11 or Windows Server 2016+
- ✅ Administrator access
- ✅ 10 GB free disk space

## 🚀 Installation (3 Steps)

### Step 1: Extract Files

Extract the deployment package to a folder (e.g., `C:\Temp\ArcVault-Deployment`)

### Step 2: Run Deployment

Open PowerShell **as Administrator** and run:

```powershell
cd C:\Temp\ArcVault-Deployment\deployment
.\Deploy-ArcVault.ps1
```

**That's it!** The script will:
- ✅ Check system requirements
- ✅ Install Python (if needed)
- ✅ Deploy application files
- ✅ Configure Windows service
- ✅ Generate SSL certificates
- ✅ Start the service

### Step 3: Verify Installation

```powershell
.\Verify-ArcVault.ps1
```

You should see: **✓ VERIFICATION PASSED**

---

## 🎮 What's Next?

### Access the Application

**GUI Application:**
```powershell
& "C:\Program Files\ArcVault\arcvault.exe"
```

**API Endpoint:**
- URL: `http://localhost:8443`
- Test: Open in browser or use `curl`

### Check Service Status

```powershell
Get-Service ArcVaultAgent
```

Should show: **Status: Running**

### View Logs

```powershell
# Service log
Get-Content "$env:TEMP\arcvault_service.log" -Tail 20

# Application logs
Get-Content "C:\ProgramData\ArcVault\logs\agent.log" -Tail 20
```

---

## ⚙️ Configuration

### Change Port

Edit configuration:
```powershell
notepad "C:\ProgramData\ArcVault\service_config.json"
```

Change port:
```json
{
  "port": 9443
}
```

Restart service:
```powershell
Restart-Service ArcVaultAgent
```

### Configure Backup Jobs

Edit backup configuration:
```powershell
notepad "C:\ProgramData\ArcVault\backup_config.json"
```

Add backup job:
```json
{
  "name": "My Backup",
  "enabled": true,
  "sources": ["C:\\MyData"],
  "destination": "D:\\Backups",
  "schedule": {
    "type": "daily",
    "time": "02:00"
  }
}
```

---

## 🔄 Common Commands

### Service Management
```powershell
# Start service
Start-Service ArcVaultAgent

# Stop service
Stop-Service ArcVaultAgent

# Restart service
Restart-Service ArcVaultAgent

# Check status
Get-Service ArcVaultAgent
```

### Verification
```powershell
# Quick check
.\Verify-ArcVault.ps1

# Detailed check
.\Verify-ArcVault.ps1 -Detailed

# Export report
.\Verify-ArcVault.ps1 -Detailed -ExportReport "C:\Reports\arcvault.txt"
```

---

## 🔧 Troubleshooting

### Service Won't Start

```powershell
# Check log for errors
Get-Content "$env:TEMP\arcvault_service.log" -Tail 50

# Check port availability
Test-NetConnection -ComputerName localhost -Port 8443
```

### Port Already in Use

Install on different port:
```powershell
.\Deploy-ArcVault.ps1 -Port 9443
```

### Python Not Found

The deployment script will attempt to install Python automatically. If it fails:
1. Download Python 3.9+ from https://www.python.org/downloads/
2. Install with "Add Python to PATH" checked
3. Re-run deployment script

---

## 📚 Full Documentation

For detailed information:

- **[README.md](README.md)** - Complete deployment overview
- **[DEPLOYMENT_GUIDE.md](docs/DEPLOYMENT_GUIDE.md)** - Comprehensive deployment guide
- **[SYSTEM_REQUIREMENTS.md](docs/SYSTEM_REQUIREMENTS.md)** - System requirements
- **[TROUBLESHOOTING.md](docs/TROUBLESHOOTING.md)** - Troubleshooting guide

---

## 🆘 Need Help?

### Check Service Health

```powershell
# Download and run health check
# (Save the diagnostic script from TROUBLESHOOTING.md)
.\Check-ArcVaultHealth.ps1
```

### Collect Diagnostics

```powershell
# Create diagnostic bundle
$timestamp = Get-Date -Format "yyyyMMdd_HHmmss"
$bundlePath = "C:\Temp\ArcVault_Diagnostics_$timestamp"
New-Item -Path $bundlePath -ItemType Directory -Force

# Copy logs
Copy-Item "$env:TEMP\arcvault_service.log" $bundlePath -ErrorAction SilentlyContinue
Copy-Item "C:\ProgramData\ArcVault\logs\*" $bundlePath -Recurse -ErrorAction SilentlyContinue

# Compress
Compress-Archive -Path $bundlePath -DestinationPath "$bundlePath.zip"
Write-Host "Diagnostics: $bundlePath.zip"
```

---

## ✅ Success Indicators

Your installation is successful when:

✓ Service status is **Running**
```powershell
(Get-Service ArcVaultAgent).Status -eq "Running"
```

✓ Port is **listening**
```powershell
Test-NetConnection -ComputerName localhost -Port 8443
```

✓ No **errors** in logs
```powershell
!(Select-String -Path "$env:TEMP\arcvault_service.log" -Pattern "ERROR" -Quiet)
```

✓ Verification **passes**
```powershell
.\Verify-ArcVault.ps1
# Should show: ✓ VERIFICATION PASSED
```

---

## 🎉 Congratulations!

ArcVault is now installed and running. You can:

1. ✅ Configure backup jobs
2. ✅ Set up monitoring
3. ✅ Add more agents
4. ✅ Customize settings

**Happy backing up! 🚀**


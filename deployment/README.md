# 🚀 ArcVault Professional Deployment Package

**Production-ready deployment scripts for ArcVault backup system**

## 📋 Quick Start

### Prerequisites
- Windows 10/11 or Windows Server 2016+
- PowerShell 5.1 or later
- Administrator privileges
- Python 3.9+ (will be installed if missing)

### One-Line Installation
```powershell
# Run as Administrator
.\Deploy-ArcVault.ps1 -InstallPath "C:\Program Files\ArcVault" -Verbose
```

## 📦 What's Included

### Deployment Scripts
- **Deploy-ArcVault.ps1** - Main deployment script with automated installation
- **Verify-ArcVault.ps1** - Post-installation verification and health checks
- **Rollback-ArcVault.ps1** - Safe rollback to previous version
- **Upgrade-ArcVault.ps1** - Upgrade existing installation

### Configuration Files
- **default_service_config.json** - Service configuration template
- **default_backup_config.json** - Backup job configuration template
- **service.env** - Environment variables for service

### Documentation
- **DEPLOYMENT_GUIDE.md** - Comprehensive deployment instructions
- **SYSTEM_REQUIREMENTS.md** - System requirements and prerequisites
- **TROUBLESHOOTING.md** - Common issues and solutions

## 🎯 Deployment Options

### Option 1: Standard Installation (Recommended)
```powershell
.\Deploy-ArcVault.ps1
```
Installs to: `C:\Program Files\ArcVault`

### Option 2: Custom Installation Path
```powershell
.\Deploy-ArcVault.ps1 -InstallPath "D:\ArcVault"
```

### Option 3: Development Mode
```powershell
.\Deploy-ArcVault.ps1 -Development
```
Installs without service, for testing

### Option 4: Silent Installation
```powershell
.\Deploy-ArcVault.ps1 -Silent -LogPath "C:\Logs\arcvault_install.log"
```

## 📊 Installation Process

The deployment script will:

1. ✅ **Pre-flight Checks**
   - Verify administrator privileges
   - Check system requirements
   - Validate prerequisites

2. ✅ **Backup Existing Installation**
   - Create timestamped backup
   - Preserve configurations
   - Enable rollback capability

3. ✅ **Install Python Environment**
   - Install Python 3.9+ if missing
   - Create virtual environment
   - Install dependencies

4. ✅ **Deploy Application Files**
   - Copy files to installation directory
   - Set proper permissions
   - Create directory structure

5. ✅ **Configure Service**
   - Install Windows service
   - Configure startup settings
   - Set environment variables

6. ✅ **Initial Configuration**
   - Generate SSL certificates
   - Create JWT keys
   - Setup default configurations

7. ✅ **Verify Installation**
   - Health checks
   - Service status verification
   - Network connectivity tests

8. ✅ **Start Services**
   - Start ArcVault Agent service
   - Verify service is running
   - Log startup information

## 🔧 Post-Installation

### Verify Installation
```powershell
.\Verify-ArcVault.ps1
```

### Check Service Status
```powershell
Get-Service ArcVaultAgent
```

### View Service Logs
```powershell
Get-Content "$env:TEMP\arcvault_service.log" -Tail 50
```

### Access GUI
```powershell
& "C:\Program Files\ArcVault\arcvault.exe"
```

## 🔄 Upgrade Existing Installation

```powershell
# Upgrade to new version
.\Upgrade-ArcVault.ps1 -SourcePath ".\new_version"
```

## ⏮️ Rollback

If something goes wrong:
```powershell
# Rollback to previous version
.\Rollback-ArcVault.ps1
```

## 🛡️ Security Considerations

- **SSL Certificates**: Generated automatically for HTTPS
- **JWT Keys**: RS256 key pair created during installation
- **Service Account**: Runs as LocalSystem by default
- **File Permissions**: Restricted to Administrators group
- **Firewall**: Configure rules for agent port (default 8443)

## 📁 Installation Structure

```
C:\Program Files\ArcVault\
├── arcvault.exe                    # Main unified application
├── python\                         # Embedded Python runtime
├── venv\                          # Python virtual environment
├── logs\                          # Application logs
└── config\                        # Configuration files

C:\ProgramData\ArcVault\
├── service_config.json            # Service configuration
├── .env                           # Environment variables
├── keys\
│   ├── jwt_private.pem           # JWT private key
│   ├── jwt_public.pem            # JWT public key
│   ├── server.crt                # SSL certificate
│   └── server.key                # SSL private key
└── logs\
    └── #backup_logs\              # Backup operation logs
```

## 🐛 Troubleshooting

### Service Won't Start
```powershell
# Check service logs
Get-Content "$env:TEMP\arcvault_service.log"

# Verify port is available
Test-NetConnection -ComputerName localhost -Port 8443
```

### Port Already in Use
```powershell
# Find process using port
netstat -ano | findstr :8443

# Edit service configuration
notepad "C:\ProgramData\ArcVault\service_config.json"
```

### Configuration Issues
```powershell
# Re-run configuration wizard
.\Deploy-ArcVault.ps1 -ConfigureOnly
```

For more troubleshooting, see [TROUBLESHOOTING.md](docs/TROUBLESHOOTING.md)

## 📞 Support

- **Documentation**: See `docs/` folder
- **Logs**: Check `C:\ProgramData\ArcVault\logs\`
- **Service Logs**: Check `%TEMP%\arcvault_service.log`

## 📄 License

See LICENSE file in project root

---

**Built with ❤️ for enterprise backup management**


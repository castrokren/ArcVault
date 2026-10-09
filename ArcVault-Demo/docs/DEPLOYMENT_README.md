# ArcVault Production Deployment

This guide explains how to deploy ArcVault in production environments without requiring Python installation on target systems.

## 🚀 Quick Start

### One-Click Deployment
```bash
# Build everything (run once on development machine)
scripts\one_click_deploy.bat
```

This creates a complete deployment package with:
- ✅ Self-contained executables (no Python required)
- ✅ Automated installer
- ✅ Configuration wizard
- ✅ Service management tools
- ✅ Complete documentation

## 📦 Deployment Options

### Option 1: Single-File Installer (Recommended)
1. Copy `ArcVault-Installer.exe` to target server
2. Run as Administrator
3. Follow the setup wizard
4. Start services

### Option 2: ZIP Package
1. Copy `ArcVault-Deployment.zip` to target server
2. Extract to a folder
3. Run `ArcVault-Deploy.bat`
4. Configure and start services

### Option 3: Direct Copy
1. Copy the entire `deployment` folder to target server
2. Run `ArcVault-Deploy.bat`
3. Full manual control over installation

## 🛠️ What Gets Deployed

### Executables
- **ArcVaultAgent.exe** - Backup agent (standalone)
- **ArcVaultAgentService.exe** - Windows service version
- **ArcVaultController.exe** - Web control panel
- **ArcVaultKeyGen.exe** - JWT key generator

### Configuration
- **Environment templates** - Development, staging, production
- **Setup wizard** - Automated configuration
- **Service manager** - Windows service management
- **Documentation** - Complete setup guides

### Installation Structure
```
Program Files\ArcVault\
├── ArcVaultAgent.exe
├── ArcVaultAgentService.exe
├── ArcVaultController.exe
├── ArcVaultKeyGen.exe
└── ArcVault-Deploy.bat

ProgramData\ArcVault\
├── .env (configuration)
├── keys\ (SSL certificates, JWT keys)
└── logs\ (backup logs)
```

## 🔧 Installation Process

### 1. Run Installer
- Execute installer as Administrator
- Choose installation directory
- Files are installed to Program Files

### 2. Configuration Wizard
- Select environment (dev/staging/production)
- Enter server hostname
- JWT keys generated automatically
- SSL certificates configured

### 3. Service Management
- Install Windows Service
- Start/stop services
- Monitor service status
- View logs

## 🌐 Access Points

- **Agent API**: `https://your-server:8443`
- **Controller UI**: `http://your-server:8501`
- **Health Check**: `https://your-server:8443/health`

## 🔒 Security Features

### Production Mode
- ✅ JWT authentication with RSA keys
- ✅ SSL/TLS encryption
- ✅ Secure key storage
- ✅ Service isolation

### Development Mode
- ✅ Simple token authentication
- ✅ HTTP (no SSL required)
- ✅ Debug logging
- ✅ Localhost access

## 📋 System Requirements

### Target Server
- Windows 10/11 or Windows Server 2016+
- Administrator privileges for installation
- Network access for agent communication
- Ports 8443 (agent) and 8501 (controller) available

### Development Machine (for building)
- Python 3.9+
- PyInstaller
- Windows SDK (for service creation)

## 🚨 Troubleshooting

### Common Issues

**Service won't start**
- Check Windows Event Log
- Verify configuration in ProgramData\ArcVault\.env
- Ensure ports are not in use

**SSL Certificate Errors**
- Verify certificate files exist and are readable
- Check certificate validity and hostname match
- Ensure CA bundle is correct

**JWT Authentication Errors**
- Verify JWT keys are generated and accessible
- Check issuer and audience settings
- Ensure key permissions are correct

### Log Locations
- **Service Logs**: Windows Event Log (Application)
- **Backup Logs**: ProgramData\ArcVault\logs\
- **Configuration**: ProgramData\ArcVault\.env

## 🔄 Update Process

### Automatic Updates
- ArcVault includes built-in update checking
- Updates can be downloaded and installed automatically
- Service restart required after updates

### Manual Updates
1. Download new deployment package
2. Stop services
3. Replace executables
4. Restart services

## 📞 Support

### Documentation
- Complete setup guides in deployment package
- Troubleshooting documentation
- Security best practices

### Getting Help
- GitHub Issues: https://github.com/castrokren/arcvault-tool
- Check logs for error details
- Verify configuration settings

## 🎯 Best Practices

### Security
- Use JWT authentication in production
- Enable SSL/TLS for all communications
- Restrict network access with firewalls
- Regularly rotate JWT keys

### Monitoring
- Monitor service logs for errors
- Check backup job status regularly
- Verify network connectivity
- Monitor disk space for backup logs

### Maintenance
- Keep SSL certificates up to date
- Rotate JWT keys periodically
- Clean up old backup logs
- Monitor system resources

---

**Ready to deploy?** Run `scripts\one_click_deploy.bat` to create your deployment package!

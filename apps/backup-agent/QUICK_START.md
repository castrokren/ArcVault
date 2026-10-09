# ArcVault Unified - Quick Start Guide

## 🚀 Get Started in 3 Steps

### 1️⃣ Build It

```powershell
cd C:\Users\castrk05\Py_stuff\ArcVault\apps\backup-agent
.\build.ps1
```

Wait for build to complete. You'll get: `dist\ArcVault.exe`

### 2️⃣ Run It

```powershell
.\dist\ArcVault.exe
```

Or just double-click `ArcVault.exe`

### 3️⃣ Install Service

In the GUI window that opens:
1. Click **"⚙️ Service"** tab
2. Click **"Install Service"**
3. Click **"Start Service"**
4. Click **"Test Agent Connection"**

Done! ✅

## 📋 What You Can Do

### From the GUI:

**⚙️ Service Tab:**
- Install/Start/Stop/Uninstall Windows service
- Configure host and port
- Test agent connection
- Check service status

**🔐 SSL Certificates Tab:**
- Generate self-signed SSL certificates
- Configure hostname, key size, validity
- Choose output directory

**🔑 JWT Keys Tab:**
- Generate JWT authentication keys
- Configure key size
- Choose output directory

**ℹ️ About Tab:**
- Version information
- Feature list

### From Command Line:

```powershell
# Service management
ArcVault.exe install
ArcVault.exe start
ArcVault.exe stop
ArcVault.exe remove

# Generate SSL certificate
ArcVault.exe --gen-cert --hostname myserver.com --output C:\certs

# Generate JWT keys
ArcVault.exe --gen-jwt --output C:\keys
```

## 🎯 Common Tasks

### Task: Set up service for first time

```powershell
# Run GUI
.\ArcVault.exe

# In GUI:
# 1. Service tab → Configure port (e.g., 8443)
# 2. Click "Save Configuration"
# 3. Click "Install Service"
# 4. Click "Start Service"
# 5. Click "Test Agent Connection" to verify
```

### Task: Generate SSL certificates for my server

```powershell
# Option 1: Use GUI
.\ArcVault.exe
# Go to SSL Certificates tab, fill in details, click Generate

# Option 2: Use CLI
.\ArcVault.exe --gen-cert ^
    --hostname backup.mycompany.com ^
    --org "My Company" ^
    --country US ^
    --output C:\ArcVault\certs
```

### Task: Generate JWT keys for authentication

```powershell
# Option 1: Use GUI
.\ArcVault.exe
# Go to JWT Keys tab, click Generate

# Option 2: Use CLI
.\ArcVault.exe --gen-jwt --output C:\ArcVault\keys
```

### Task: Check if service is running

```powershell
# Option 1: Use GUI
.\ArcVault.exe
# Service tab → "Check Service Status"

# Option 2: Use Windows services
sc query ArcVaultAgent

# Option 3: Check if port is listening
netstat -an | findstr 8443
```

### Task: Troubleshoot service issues

```powershell
# View service log
notepad %TEMP%\arcvault_service.log

# Check Windows Event Log
eventvwr.msc
# Navigate to: Windows Logs → Application → Filter by "ArcVaultAgent"

# Test connection
curl http://localhost:8443/health
```

### Task: Change service port

```powershell
# 1. Stop service
.\ArcVault.exe stop

# 2. Run GUI
.\ArcVault.exe

# 3. In Service tab:
#    - Change port number
#    - Click "Save Configuration"

# 4. Start service
.\ArcVault.exe start
```

### Task: Distribute to other machines

```powershell
# Just copy the single file:
Copy-Item .\dist\ArcVault.exe \\other-machine\C$\Tools\

# On other machine, users can:
# 1. Double-click ArcVault.exe
# 2. Use GUI to install service
# 3. Done!
```

## 📁 Important Files

| File | Location | Purpose |
|------|----------|---------|
| `ArcVault.exe` | `dist\` | The main executable (everything) |
| `service_config.json` | `C:\Program Files (x86)\ArcVault\` | Service configuration |
| Service log | `%TEMP%\arcvault_service.log` | Service activity log |
| SSL certs | User choice (default: `keys\`) | Generated certificates |
| JWT keys | User choice (default: `keys\`) | Generated JWT keys |

## 🔧 Configuration

### Service Configuration File

Location: `C:\Program Files (x86)\ArcVault\service_config.json`

```json
{
  "host": "0.0.0.0",
  "port": 8443
}
```

Created automatically when you:
- Click "Save Configuration" in GUI, or
- Click "Install Service"

You can edit it manually if needed.

### Environment Variables (Optional)

Instead of config file, you can set:
- `AGENT_LISTEN_HOST` - Listen host (default: 0.0.0.0)
- `AGENT_PORT` - Listen port (default: 8443)
- `AGENT_REQUIRE_JWT` - Require JWT auth (true/false)
- `AGENT_JWT_PUBLIC_KEY_PATH` - Path to JWT public key

## ❓ Troubleshooting

### "Service failed to install"
→ Run as Administrator (right-click → Run as Administrator)

### "Port 8443 already in use"
→ Change port in GUI before installing, or stop the conflicting service

### "GUI won't start"
→ Run from command prompt to see error messages

### "Can't connect to agent"
→ Check if service is running: `sc query ArcVaultAgent`

### "Build failed"
→ Install dependencies: `pip install -r requirements_unified.txt`

## 📚 More Help

- **Full Documentation**: `README_UNIFIED.md`
- **Build Instructions**: `BUILD_UNIFIED.md`
- **Complete Summary**: `UNIFIED_APP_SUMMARY.md`

## 💡 Pro Tips

1. **Backup your keys**: Generated SSL/JWT keys should be backed up
2. **Use strong ports**: Don't use default ports in production
3. **Check logs**: When troubleshooting, always check the service log first
4. **Test first**: Use "Test Agent Connection" to verify service is responding
5. **One executable**: Distribute just `ArcVault.exe` - no dependencies needed!

---

**Need help? All documentation is in `apps/backup-agent/` directory.**


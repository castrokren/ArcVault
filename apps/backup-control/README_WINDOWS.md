# ArcVault Backup Controller - Windows Deployment Guide

## Overview

The ArcVault Backup Controller is a Streamlit-based web application that provides a centralized interface for managing multiple backup agents. It allows you to trigger backup jobs, monitor progress, and view live logs from remote agents across your network.

## 🚀 Quick Start (Windows)

### Prerequisites
- Windows 10/11 or Windows Server 2016+
- Python 3.9 or later
- PowerShell 5.1 or later
- Administrator privileges (for service installation)

### Option 1: Development Setup
```powershell
# Clone and setup
cd ArcVault/apps/backup-control

# Create virtual environment
python -m venv .venv
.\.venv\Scripts\Activate.ps1

# Install dependencies
pip install -e .

# Set basic configuration
$env:AGENT_TOKEN = "dev-token"  # Match your agents
$env:AGENT_URL = "http://localhost:8443"  # Default agent

# Run the controller
streamlit run controller/app.py
```

### Option 2: Production Deployment
```powershell
# Run the installer script
.\scripts\install_controller.ps1 -Port 8501 -Address "0.0.0.0"

# Or use the Python runner
python controller/run_controller.py
```

The controller will be available at: `http://localhost:8501`

## 📦 Installation Methods

### Method 1: Automated Installation (Recommended)

The installer script sets up JWT authentication and proper configuration:

```powershell
# Basic installation
.\scripts\install_controller.ps1

# Custom configuration
.\scripts\install_controller.ps1 -ConfigDir "C:\ArcVault\Config" -Port 8080 -Address "192.168.1.100"

# With custom JWT issuer
.\scripts\install_controller.ps1 -Issuer "my-backup-controller"
```

**What gets installed:**
- Configuration directory: `C:\ProgramData\BackupController\`
- JWT key pair: `controller_private.pem` and `controller_public.pem`
- Environment variables for runtime configuration

### Method 2: Manual Installation

1. **Install Python dependencies:**
   ```powershell
   pip install streamlit requests PyJWT cryptography
   ```

2. **Generate JWT keys (optional but recommended):**
   ```powershell
   # Create config directory
   New-Item -ItemType Directory -Force -Path "C:\ProgramData\BackupController"
   
   # Generate keys using Python
   python -c "
   from cryptography.hazmat.primitives.asymmetric import rsa
   from cryptography.hazmat.primitives import serialization
   from pathlib import Path
   
   key = rsa.generate_private_key(public_exponent=65537, key_size=3072)
   priv_path = Path('C:/ProgramData/BackupController/controller_private.pem')
   pub_path = Path('C:/ProgramData/BackupController/controller_public.pem')
   
   priv_path.write_bytes(key.private_bytes(
       encoding=serialization.Encoding.PEM,
       format=serialization.PrivateFormat.TraditionalOpenSSL,
       encryption_algorithm=serialization.NoEncryption()
   ))
   
   pub_path.write_bytes(key.public_key().public_bytes(
       encoding=serialization.Encoding.PEM,
       format=serialization.PublicFormat.SubjectPublicKeyInfo
   ))
   
   print(f'Private key: {priv_path}')
   print(f'Public key: {pub_path}')
   "
   ```

3. **Configure environment variables:**
   ```powershell
   $env:CTRL_JWT_PRIVATE_KEY_PATH = "C:\ProgramData\BackupController\controller_private.pem"
   $env:CTRL_JWT_ISSUER = "backup-controller"
   $env:CONTROLLER_PORT = "8501"
   $env:CONTROLLER_ADDRESS = "0.0.0.0"
   ```

4. **Run the controller:**
   ```powershell
   python controller/run_controller.py
   ```

### Method 3: Docker Deployment (Windows Containers)

```powershell
# Build the container
docker build -t arcvault-controller -f Dockerfile .

# Run with volume for configuration
docker run -d -p 8501:8501 -v C:\ProgramData\BackupController:/config arcvault-controller
```

## ⚙️ Configuration

### Environment Variables

| Variable | Description | Default | Required |
|----------|-------------|---------|----------|
| `CONTROLLER_PORT` | Web interface port | `8501` | No |
| `CONTROLLER_ADDRESS` | Bind address | `0.0.0.0` | No |
| `CTRL_JWT_PRIVATE_KEY_PATH` | JWT private key file | | No |
| `CTRL_JWT_PRIVATE_KEY` | JWT private key (inline) | | No |
| `CTRL_JWT_ISSUER` | JWT issuer name | `backup-controller` | No |
| `AGENT_TOKEN` | Fallback static token | `dev-token` | No |
| `AGENT_URL` | Default agent URL | `http://127.0.0.1:8443` | No |
| `STREAMLIT_HEADLESS` | Run without browser | `true` | No |

### Agent Configuration

The controller manages agents through a JSON file: `controller/agents.json`

```json
{
  "windows-server-01": "http://192.168.1.100:8443",
  "windows-server-02": "http://192.168.1.101:8443",
  "dev-machine": "http://localhost:8443"
}
```

### JWT Authentication Setup

For production deployments, use JWT authentication:

1. **Generate key pair** (done by installer):
   ```powershell
   .\scripts\install_controller.ps1
   ```

2. **Distribute public key to agents:**
   ```powershell
   # Copy public key to each agent machine
   Copy-Item "C:\ProgramData\BackupController\controller_public.pem" "\\agent-machine\C$\ProgramData\ArcVault\"
   ```

3. **Configure agents to use JWT:**
   ```powershell
   # On each agent machine
   $env:AGENT_REQUIRE_JWT = "true"
   $env:AGENT_JWT_PUBLIC_KEY_PATH = "C:\ProgramData\ArcVault\controller_public.pem"
   ```

## 🖥️ Web Interface

### Main Features

1. **Agent Management**
   - Add/remove agents
   - View agent status and information
   - Test agent connectivity

2. **Backup Job Control**
   - Create backup jobs with multiple sources
   - Configure backup options (mirror, subdirectories, etc.)
   - Monitor job progress in real-time

3. **Live Log Streaming**
   - Stream logs from running jobs
   - Server-Sent Events for real-time updates
   - Job history and status tracking

4. **Multi-Agent Support**
   - Manage multiple agents from one interface
   - Switch between agents easily
   - Bulk operations across agents

### Using the Interface

1. **Access the controller:**
   ```
   http://localhost:8501
   ```

2. **Add agents:**
   - Click "Manage Agents" in the sidebar
   - Enter agent name and URL
   - Test connectivity
   - Save configuration

3. **Create backup jobs:**
   - Select target agent
   - Enter source paths (one per line)
   - Enter destination path
   - Configure options
   - Click "Start Job"

4. **Monitor progress:**
   - View job status in real-time
   - Click "Follow logs" for live updates
   - Check job history

## 🔧 Management and Operations

### Starting/Stopping the Controller

```powershell
# Start controller
python controller/run_controller.py

# Or with custom configuration
$env:CONTROLLER_PORT = "8080"
$env:CONTROLLER_ADDRESS = "192.168.1.100"
python controller/run_controller.py

# Using Streamlit directly
streamlit run controller/app.py --server.port 8501 --server.address 0.0.0.0
```

### Service Installation (Windows Service)

Create a Windows Service for production deployment:

```powershell
# Create service wrapper script
$serviceScript = @"
import subprocess
import os
import sys

# Set environment variables
os.environ['CTRL_JWT_PRIVATE_KEY_PATH'] = r'C:\ProgramData\BackupController\controller_private.pem'
os.environ['CTRL_JWT_ISSUER'] = 'backup-controller'
os.environ['CONTROLLER_PORT'] = '8501'
os.environ['CONTROLLER_ADDRESS'] = '0.0.0.0'
os.environ['STREAMLIT_HEADLESS'] = 'true'

# Change to controller directory
os.chdir(r'$PWD')

# Run the controller
subprocess.run([sys.executable, 'controller/run_controller.py'])
"@

Set-Content -Path "run_controller_service.py" -Value $serviceScript

# Install as Windows Service using NSSM or similar
# nssm install "ArcVault Controller" python "C:\path\to\run_controller_service.py"
```

### Scheduled Task Alternative

```powershell
# Create scheduled task for auto-start
$action = New-ScheduledTaskAction -Execute "python" -Argument "controller/run_controller.py" -WorkingDirectory $PWD
$trigger = New-ScheduledTaskTrigger -AtStartup
$settings = New-ScheduledTaskSettingsSet -AllowStartIfOnBatteries -DontStopIfGoingOnBatteries -StartWhenAvailable
$principal = New-ScheduledTaskPrincipal -UserId "SYSTEM" -LogonType ServiceAccount -RunLevel Highest

Register-ScheduledTask -TaskName "ArcVault Controller" -Action $action -Trigger $trigger -Settings $settings -Principal $principal -Description "ArcVault Backup Controller Web Interface"
```

### Configuration Management

```powershell
# Backup configuration
Copy-Item -Path "C:\ProgramData\BackupController" -Destination "backup-location" -Recurse

# View current agents
Get-Content "controller/agents.json" | ConvertFrom-Json

# Add agent programmatically
$agents = Get-Content "controller/agents.json" | ConvertFrom-Json
$agents | Add-Member -NotePropertyName "new-agent" -NotePropertyValue "http://192.168.1.102:8443"
$agents | ConvertTo-Json | Set-Content "controller/agents.json"
```

## 🔍 Troubleshooting

### Common Issues

**Controller won't start:**
```powershell
# Check Python installation
python --version

# Check dependencies
pip list | findstr streamlit

# Test manual start
python -c "import streamlit; print('Streamlit OK')"

# Check port availability
netstat -ano | findstr :8501
```

**Can't connect to agents:**
```powershell
# Test agent connectivity
Invoke-WebRequest -Uri "http://agent-ip:8443/health" -UseBasicParsing

# Check authentication
$headers = @{"X-API-Key" = "your-token"}
Invoke-WebRequest -Uri "http://agent-ip:8443/info" -Headers $headers

# Verify JWT configuration
Test-Path "C:\ProgramData\BackupController\controller_private.pem"
```

**JWT authentication issues:**
```powershell
# Verify key pair exists
Test-Path "C:\ProgramData\BackupController\controller_private.pem"
Test-Path "C:\ProgramData\BackupController\controller_public.pem"

# Check key permissions
Get-Acl "C:\ProgramData\BackupController\controller_private.pem"

# Test JWT generation
python -c "
import jwt
import os
from datetime import datetime, timedelta

with open('C:/ProgramData/BackupController/controller_private.pem', 'r') as f:
    private_key = f.read()

payload = {
    'iss': 'backup-controller',
    'aud': 'backup-agent',
    'exp': datetime.utcnow() + timedelta(hours=1),
    'iat': datetime.utcnow()
}

token = jwt.encode(payload, private_key, algorithm='RS256')
print('JWT generated successfully')
print(f'Token length: {len(token)}')
"
```

**Web interface issues:**
```powershell
# Clear Streamlit cache
Remove-Item -Path "$env:USERPROFILE\.streamlit" -Recurse -Force -ErrorAction SilentlyContinue

# Check browser access
Start-Process "http://localhost:8501"

# Test with different browser/incognito mode
```

### Log Analysis

```powershell
# Streamlit logs are typically in console output
# Redirect to file for analysis
python controller/run_controller.py > controller.log 2>&1

# Monitor in real-time
Get-Content controller.log -Wait -Tail 10

# Search for errors
Select-String -Path controller.log -Pattern "ERROR|Exception|Failed"
```

### Network Diagnostics

```powershell
# Test controller accessibility
Test-NetConnection -ComputerName localhost -Port 8501

# Test from remote machine
Test-NetConnection -ComputerName controller-ip -Port 8501

# Check firewall rules
Get-NetFirewallRule -DisplayName "*8501*"

# Create firewall rule if needed
New-NetFirewallRule -DisplayName "ArcVault Controller" -Direction Inbound -Protocol TCP -LocalPort 8501 -Action Allow
```

## 🔒 Security Considerations

### Production Security

1. **Use HTTPS:**
   ```powershell
   # Configure Streamlit for HTTPS
   # Create .streamlit/config.toml
   New-Item -ItemType Directory -Force -Path ".streamlit"
   @"
   [server]
   sslCertFile = "C:/certs/controller.crt"
   sslKeyFile = "C:/certs/controller.key"
   "@ | Set-Content ".streamlit/config.toml"
   ```

2. **Secure JWT keys:**
   ```powershell
   # Restrict access to private key
   icacls "C:\ProgramData\BackupController\controller_private.pem" /inheritance:r
   icacls "C:\ProgramData\BackupController\controller_private.pem" /grant:r "Administrators:F" "SYSTEM:F"
   ```

3. **Network security:**
   ```powershell
   # Bind to specific interface
   $env:CONTROLLER_ADDRESS = "192.168.1.100"
   
   # Restrict firewall access
   New-NetFirewallRule -DisplayName "ArcVault Controller" -Direction Inbound -Protocol TCP -LocalPort 8501 -RemoteAddress "192.168.1.0/24" -Action Allow
   ```

4. **Authentication:**
   ```powershell
   # Use strong JWT keys (3072-bit RSA minimum)
   # Rotate keys periodically
   # Monitor access logs
   ```

### Access Control

The controller web interface has no built-in authentication. For production:

1. **Use a reverse proxy** (IIS, nginx) with authentication
2. **VPN access** to restrict network access
3. **Windows Authentication** via IIS integration
4. **Custom authentication** wrapper

Example IIS configuration:
```xml
<system.webServer>
    <security>
        <authentication>
            <windowsAuthentication enabled="true" />
            <anonymousAuthentication enabled="false" />
        </authentication>
    </security>
    <rewrite>
        <rules>
            <rule name="Streamlit" stopProcessing="true">
                <match url="(.*)" />
                <action type="Rewrite" url="http://localhost:8501/{R:1}" />
            </rule>
        </rules>
    </rewrite>
</system.webServer>
```

## 📊 Monitoring and Maintenance

### Health Monitoring

```powershell
# Create health check script
$healthCheck = @"
try {
    `$response = Invoke-WebRequest -Uri "http://localhost:8501" -TimeoutSec 10
    if (`$response.StatusCode -eq 200) {
        Write-Host "Controller is healthy"
        exit 0
    }
} catch {
    Write-Host "Controller health check failed: `$_"
    exit 1
}
"@

Set-Content -Path "controller-health-check.ps1" -Value $healthCheck
```

### Performance Monitoring

```powershell
# Monitor Python process
Get-Process python | Where-Object {$_.MainWindowTitle -like "*streamlit*"}

# Check memory usage
Get-Process python | Select-Object ProcessName, WorkingSet, VirtualMemorySize

# Monitor network connections
netstat -ano | findstr :8501
```

### Backup and Recovery

```powershell
# Backup controller configuration
$backupDir = "C:\Backups\ArcVault\$(Get-Date -Format 'yyyy-MM-dd')"
New-Item -ItemType Directory -Force -Path $backupDir

Copy-Item -Path "C:\ProgramData\BackupController" -Destination "$backupDir\config" -Recurse
Copy-Item -Path "controller\agents.json" -Destination "$backupDir\agents.json"

# Create restore script
@"
# Restore ArcVault Controller Configuration
Copy-Item -Path "$backupDir\config\*" -Destination "C:\ProgramData\BackupController" -Recurse -Force
Copy-Item -Path "$backupDir\agents.json" -Destination "controller\agents.json" -Force
"@ | Set-Content "$backupDir\restore.ps1"
```

## 🔄 Updates and Maintenance

### Updating the Controller

```powershell
# Stop controller (if running as service/task)
Stop-Process -Name python -Force

# Update code
git pull origin main

# Update dependencies
pip install -e . --upgrade

# Restart controller
python controller/run_controller.py
```

### Database Migration

The controller uses JSON files for configuration. For upgrades:

```powershell
# Backup current configuration
Copy-Item "controller\agents.json" "controller\agents.json.backup"

# Apply any configuration schema changes
# (handled automatically by the application)
```

## 🤝 Integration with Agents

### Adding New Agents

1. **Install agent** on target machine using the agent installer
2. **Note the agent URL** from installation output
3. **Add to controller** via web interface or JSON file:
   ```json
   {
     "new-agent-name": "http://agent-ip:8443"
   }
   ```
4. **Test connectivity** using the web interface

### Agent Authentication

**Static Token Method:**
```powershell
# Same token on controller and all agents
$env:AGENT_TOKEN = "shared-secure-token"
```

**JWT Method (Recommended):**
```powershell
# Controller generates JWT tokens
# Agents verify using public key
# More secure, supports token expiration
```

### Bulk Agent Management

```powershell
# Add multiple agents from CSV
$agents = Import-Csv "agents.csv"  # Name,URL columns
$agentConfig = @{}

foreach ($agent in $agents) {
    $agentConfig[$agent.Name] = $agent.URL
}

$agentConfig | ConvertTo-Json | Set-Content "controller\agents.json"
```

## 📝 Development

### Development Setup

```powershell
# Clone repository
git clone <repository>
cd ArcVault/apps/backup-control

# Setup development environment
python -m venv .venv
.\.venv\Scripts\Activate.ps1
pip install -e .

# Install development dependencies
pip install pytest black flake8

# Run in development mode
streamlit run controller/app.py --server.runOnSave true
```

### Building Executable

```powershell
# Install PyInstaller
pip install pyinstaller

# Build single-file executable
pyinstaller --onefile --name ArcVaultController controller/run_controller.py

# The executable will be in dist/ArcVaultController.exe
```

### API Integration

The controller can be extended with REST API endpoints:

```python
# Add to controller/app.py
import streamlit as st
from streamlit.web.server.websocket_headers import _get_websocket_headers

# Custom API endpoints can be added using Streamlit's session state
# and custom request handling
```

---

## 📞 Support

For issues and questions:

1. **Check logs** for error messages
2. **Verify network connectivity** to agents
3. **Test authentication** configuration
4. **Review firewall** and security settings

**Quick diagnostics:**
```powershell
# Test controller access
Invoke-WebRequest -Uri "http://localhost:8501"

# Test agent connectivity
Invoke-WebRequest -Uri "http://agent-ip:8443/health"

# Check configuration
Get-Content "controller\agents.json"
Test-Path "C:\ProgramData\BackupController\controller_private.pem"
```

**Common solutions:**
- Restart the controller service
- Check Python and dependency versions
- Verify JWT key permissions
- Test with different browsers
- Check Windows Firewall settings

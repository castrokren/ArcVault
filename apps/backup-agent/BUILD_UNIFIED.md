# Building ArcVault Unified Application

## Overview

This builds ONE executable (`ArcVault.exe`) that does everything:
- **GUI Mode** (default): Manage certificates, JWT keys, and service installation
- **Service Mode**: Run as Windows background service
- **CLI Mode**: Generate certs/JWT from command line

## Prerequisites

1. Python 3.8 or higher
2. Windows (for service functionality)

## Build Steps

### 1. Install Dependencies

```powershell
pip install -r requirements_unified.txt
```

### 2. Build with PyInstaller

```powershell
pyinstaller build_unified.spec
```

The executable will be created in `dist/ArcVault.exe`

## Usage

### GUI Mode (Default)

Just run the executable:
```powershell
.\ArcVault.exe
```

This opens the GUI where you can:
- Generate SSL certificates
- Generate JWT keys
- Install/manage the Windows service
- Configure service settings
- Test agent connectivity

### Service Installation

From the GUI, click "Install Service" button, or from command line:

```powershell
# Install service
.\ArcVault.exe install

# Start service
.\ArcVault.exe start

# Stop service
.\ArcVault.exe stop

# Uninstall service
.\ArcVault.exe remove
```

### CLI Mode

Generate certificates:
```powershell
.\ArcVault.exe --gen-cert --hostname example.com --output C:\certs
```

Generate JWT keys:
```powershell
.\ArcVault.exe --gen-jwt --output C:\keys
```

## Service Configuration

The service reads configuration from:
1. `C:\Program Files (x86)\ArcVault\service_config.json` (if installed)
2. Environment variables: `AGENT_LISTEN_HOST`, `AGENT_PORT`
3. Defaults: `0.0.0.0:8443`

Example `service_config.json`:
```json
{
  "host": "0.0.0.0",
  "port": 8443
}
```

## Distribution

Just distribute the single `ArcVault.exe` file. It contains everything needed:
- No separate scripts
- No configuration files (created on first run)
- No external dependencies

## Troubleshooting

### Service won't install
- Run as Administrator
- Check Windows Event Viewer for errors
- Check service log: `%TEMP%\arcvault_service.log`

### GUI won't start
- Make sure PyQt5 is properly bundled
- Check console output by running from command prompt

### Build errors
- Ensure all dependencies are installed
- Try: `pip install --upgrade pip setuptools`
- Clean build: `rmdir /s /q build dist`


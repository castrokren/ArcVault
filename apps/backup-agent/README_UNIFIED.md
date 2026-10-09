# ArcVault Unified Application

## What is This?

**ONE executable that does EVERYTHING.** No more scattered scripts!

`ArcVault.exe` is your all-in-one tool for:
- 🖥️ **GUI Management Interface** - Generate certs, manage services, configure everything
- ⚙️ **Windows Service** - Runs as background service for backups
- 🔐 **Certificate Generator** - Create SSL certificates from CLI
- 🔑 **JWT Key Generator** - Create JWT keys from CLI
- 🛠️ **Service Installer** - Installs itself as a Windows service

## Quick Start

### 1. Build the Application

```powershell
cd apps\backup-agent
.\build.ps1
```

This creates `dist\ArcVault.exe` - your single unified executable.

### 2. Run the GUI

Double-click `ArcVault.exe` or run:
```powershell
.\ArcVault.exe
```

The GUI lets you:
- Generate SSL certificates
- Generate JWT authentication keys
- Install the Windows service
- Start/stop/uninstall the service
- Configure service settings (host, port)
- Test agent connectivity

### 3. Install as Windows Service

From the GUI:
1. Go to "Service" tab
2. Configure host and port
3. Click "Install Service"
4. Click "Start Service"

Or from command line:
```powershell
.\ArcVault.exe install
.\ArcVault.exe start
```

## All Usage Modes

### GUI Mode (Default)
```powershell
# Just run it
.\ArcVault.exe
```
Opens the management interface.

### Service Commands
```powershell
# Install as Windows service
.\ArcVault.exe install

# Start the service
.\ArcVault.exe start

# Stop the service
.\ArcVault.exe stop

# Uninstall the service
.\ArcVault.exe remove

# Set service to auto-start
.\ArcVault.exe --startup auto update
```

### Certificate Generation (CLI)
```powershell
# Generate SSL certificate
.\ArcVault.exe --gen-cert --hostname myserver.com --output C:\certs

# With custom settings
.\ArcVault.exe --gen-cert ^
    --hostname myserver.com ^
    --key-size 4096 ^
    --days 730 ^
    --country US ^
    --org "My Company" ^
    --output C:\certs ^
    --force
```

### JWT Key Generation (CLI)
```powershell
# Generate JWT keys
.\ArcVault.exe --gen-jwt --output C:\keys

# With custom key size
.\ArcVault.exe --gen-jwt --key-size 4096 --output C:\keys --force
```

### Help
```powershell
.\ArcVault.exe --help
```

## Project Structure

```
apps/backup-agent/
├── arcvault_unified.py          # Main entry point - routes to GUI/Service/CLI
├── gui/
│   ├── __init__.py
│   └── main_window.py           # Complete PyQt5 GUI
├── service/
│   ├── __init__.py
│   └── win_service.py           # Windows Service implementation
├── cli/
│   ├── __init__.py
│   ├── cert_generator.py        # SSL certificate CLI tool
│   └── jwt_generator.py         # JWT key CLI tool
├── agent/                       # FastAPI backup agent (existing)
│   ├── app.py
│   └── engine/
├── build_unified.spec           # PyInstaller build configuration
├── build.ps1                    # Build script
├── requirements_unified.txt     # All dependencies
└── README_UNIFIED.md           # This file
```

## Configuration

### Service Configuration

When installed, the service looks for configuration in this order:

1. **Config file**: `C:\Program Files (x86)\ArcVault\service_config.json`
   ```json
   {
     "host": "0.0.0.0",
     "port": 8443
   }
   ```

2. **Environment variables**: `AGENT_LISTEN_HOST`, `AGENT_PORT`

3. **Defaults**: `0.0.0.0:8443`

You can create/edit the config file from the GUI using "Save Configuration" button.

### Agent Configuration

For JWT authentication and other agent settings, create environment variables:
- `AGENT_REQUIRE_JWT` - Set to "true" to require JWT tokens
- `AGENT_JWT_PUBLIC_KEY_PATH` - Path to public key file
- `AGENT_JWT_ISSUER` - JWT issuer to validate

## Service Logs

Service logs are written to:
- `%TEMP%\arcvault_service.log` - Service operations log
- `C:\Program Files (x86)\ArcVault\agent_stdout.log` - Service stdout (if using NSSM)
- `C:\Program Files (x86)\ArcVault\agent_stderr.log` - Service stderr (if using NSSM)

View logs from GUI using "Check Service Status" button.

## Development

### Running Without Building

You can run the Python script directly during development:

```powershell
# GUI mode
python arcvault_unified.py

# Generate cert
python arcvault_unified.py --gen-cert --hostname test.local

# Generate JWT
python arcvault_unified.py --gen-jwt
```

Note: Service installation only works with the compiled .exe

### Building

```powershell
# Install dependencies
pip install -r requirements_unified.txt

# Build
pyinstaller build_unified.spec

# Or use the build script
.\build.ps1
```

## Distribution

To distribute to users:

1. Build the application: `.\build.ps1`
2. Copy `dist\ArcVault.exe` to target machine
3. That's it! No installer needed, no dependencies, no scripts.

Users can:
- Double-click to open GUI
- Use GUI to generate certs/JWT
- Use GUI to install Windows service
- Everything is self-contained

## Troubleshooting

### GUI won't start
- Run from command prompt to see errors
- Ensure PyQt5 was bundled correctly in build
- Check if antivirus is blocking

### Service won't install
- Must run as Administrator for service installation
- Check Windows Event Viewer → Application logs
- View service log: `%TEMP%\arcvault_service.log`

### Service won't start
- Check port 8443 isn't already in use: `netstat -an | findstr 8443`
- Check service configuration file exists
- View service logs

### Build errors
```powershell
# Clean build
Remove-Item -Recurse -Force build, dist

# Upgrade tools
pip install --upgrade pip setuptools pyinstaller

# Rebuild
.\build.ps1
```

## Benefits of Unified Approach

✅ **Single executable** - No scattered PowerShell scripts
✅ **User-friendly** - GUI for non-technical users
✅ **Flexible** - CLI for automation and scripting
✅ **Self-contained** - Installs itself as service
✅ **No dependencies** - Everything bundled
✅ **Easy distribution** - Just one .exe file
✅ **Professional** - Modern GUI with progress tracking
✅ **Maintainable** - All code in one project

## Technical Details

- **Language**: Python 3.8+
- **GUI Framework**: PyQt5
- **Service Framework**: pywin32
- **Crypto Library**: cryptography
- **Agent Framework**: FastAPI + Uvicorn
- **Build Tool**: PyInstaller
- **Platform**: Windows (service functionality)

## License

Part of the ArcVault backup system.


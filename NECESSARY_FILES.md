# ArcVault Application - Necessary Files

This document lists all files required for the ArcVault application to run.

## 1. GUI Application (PyQt5 Version)

### Core Application File
- `scripts/arcvault_gui_pyqt5.py` - Main GUI application script

### Python Dependencies (install via pip)
- **PyQt5** >= 5.15.0 - GUI framework
- **cryptography** >= 3.4.8 - SSL certificate and JWT key generation
- **psutil** - System utilities (optional, for enhanced features)
- **pywin32** >= 300 - Windows service management (optional, for service features)

### External Executables (for full functionality)
- `nssm.exe` - Non-Sucking Service Manager (for Windows service installation)
  - Location: Project root or bundled with executable
  - Used for installing/managing the backup agent as a Windows service

### Optional Files (for installer functionality)
- `installers-exe/ArcVault-Client-Installer.exe` - NSIS installer for client deployment
- `apps/backup-agent/dist/ArcVault.exe` - Backup agent executable (for installation via GUI)

---

## 2. Backup Agent Application

### Core Application Files
- `apps/backup-agent/arcvault_unified.py` - Unified entry point
- `apps/backup-agent/agent/app.py` - FastAPI application
- `apps/backup-agent/agent/engine/config.py` - Configuration management
- `apps/backup-agent/agent/engine/runner.py` - Backup execution engine
- `apps/backup-agent/agent/engine/robocopy.py` - Robocopy integration
- `apps/backup-agent/service/win_service.py` - Windows service wrapper

### Configuration Files
- `apps/backup-agent/agent/service_config.json` - Service configuration
  - Can be created at runtime or placed in:
    - `C:/ProgramData/ArcVault/service_config.json` (preferred)
    - `C:/Program Files (x86)/ArcVault/service_config.json`
    - `C:/Program Files (x86)/ArcVault/config.json`

### Python Dependencies (install via pip)
- **fastapi** >= 0.68.0 - Web framework
- **uvicorn[standard]** >= 0.15.0 - ASGI server
- **PyJWT** >= 2.0.0 - JWT token handling
- **pydantic** >= 2.5 - Data validation
- **pywin32** >= 300 - Windows service support

### Build Configuration
- `apps/backup-agent/pyproject.toml` - Package configuration
- `apps/backup-agent/requirements_unified.txt` - Unified requirements
- `apps/backup-agent/build_unified.spec` - PyInstaller spec file

---

## 3. SSL Certificate Generation

### Required Libraries
- **cryptography** >= 3.4.8 - Certificate generation
  - Includes: x509, RSA key generation, serialization

### Generated Files (created at runtime)
- `server.crt` - SSL certificate
- `server.key` - Private key
- `ca-bundle.crt` - CA bundle (self-signed)

---

## 4. JWT Key Generation

### Required Libraries
- **cryptography** >= 3.4.8 - Key pair generation

### Generated Files (created at runtime)
- `controller_private.pem` - Private key (keep secure)
- `controller_public.pem` - Public key (distribute to clients)

---

## 5. Windows Service Installation

### Required Files
- `nssm.exe` - Service manager executable
- `ArcVault.exe` - Agent executable (built from backup-agent)

### Installation Locations
- **Executable**: `C:/Program Files (x86)/ArcVault/ArcVault.exe`
- **Configuration**: `C:/ProgramData/ArcVault/service_config.json`
- **Logs**: `C:/ProgramData/ArcVault/logs/` (created automatically)

---

## 6. Build Files (for creating executables)

### GUI Build Scripts
- `scripts/build_gui_pyqt5_executable.bat` - Build script for PyQt5 GUI
- `ArcVault-GUI-PyQt5.spec` - PyInstaller spec file (optional)

### Agent Build Scripts
- `apps/backup-agent/build.ps1` - PowerShell build script
- `apps/backup-agent/build_unified.spec` - PyInstaller spec file

### Build Dependencies
- **pyinstaller** >= 5.0.0 - Executable bundler

---

## 7. Runtime Requirements

### System Requirements
- **Python** >= 3.9 (if running from source)
- **Windows** 10/11 (for Windows service features)
- **Administrator privileges** (for service installation)

### Environment Variables (optional)
- `AGENT_TOKEN` - API authentication token
- `AGENT_JWT_PUBLIC_KEY` - JWT public key (inline)
- `AGENT_JWT_PUBLIC_KEY_PATH` - Path to JWT public key file
- `AGENT_JWT_ISSUER` - JWT issuer
- `AGENT_JWT_AUDIENCE` - JWT audience
- `AGENT_REQUIRE_JWT` - Require JWT authentication (true/false)

---

## 8. Minimal Standalone Deployment

### For GUI Application Only (standalone executable)
1. Build executable: `scripts/build_gui_pyqt5_executable.bat`
2. Output: `installers-exe/ArcVault-GUI-PyQt5.exe`
3. Bundle with executable (if needed):
   - `nssm.exe` (for service management)
   - `ArcVault.exe` (for agent installation)

### For Backup Agent Only (standalone executable)
1. Build executable: `apps/backup-agent/build.ps1`
2. Output: `apps/backup-agent/dist/ArcVault.exe`
3. Required at runtime:
   - `service_config.json` (created or provided)

---

## 9. File Structure Summary

```
ArcVault/
├── scripts/
│   └── arcvault_gui_pyqt5.py          # GUI application
├── apps/
│   └── backup-agent/
│       ├── arcvault_unified.py         # Unified entry point
│       ├── agent/
│       │   ├── app.py                  # FastAPI app
│       │   ├── engine/
│       │   │   ├── config.py           # Config management
│       │   │   ├── runner.py           # Backup runner
│       │   │   └── robocopy.py         # Robocopy integration
│       │   └── service_config.json     # Service config (template)
│       ├── service/
│       │   └── win_service.py          # Windows service
│       └── dist/
│           └── ArcVault.exe            # Built agent executable
├── nssm.exe                            # Service manager
└── installers-exe/
    ├── ArcVault-GUI-PyQt5.exe          # Built GUI executable
    └── ArcVault-Client-Installer.exe  # NSIS installer (optional)
```

---

## 10. Quick Start Checklist

### To run GUI from source:
- [ ] Python 3.9+ installed
- [ ] Install: `pip install PyQt5 cryptography`
- [ ] Run: `python scripts/arcvault_gui_pyqt5.py`

### To run agent from source:
- [ ] Python 3.9+ installed
- [ ] Install: `pip install -r apps/backup-agent/requirements_unified.txt`
- [ ] Create/configure `service_config.json`
- [ ] Run: `python apps/backup-agent/arcvault_unified.py`

### To deploy as executables:
- [ ] Build GUI: `scripts/build_gui_pyqt5_executable.bat`
- [ ] Build Agent: `apps/backup-agent/build.ps1`
- [ ] Ensure `nssm.exe` is available
- [ ] Deploy executables to target machines

---

## Notes

- **Backup logs** are written to `#backup_logs` subdirectory in the destination folder (per memory)
- Configuration files can be created at runtime if not present
- Most dependencies can be auto-installed if missing (with user permission)
- The GUI can function standalone for certificate/JWT generation without the agent
- Service installation requires administrator privileges on Windows


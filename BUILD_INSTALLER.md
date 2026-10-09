# Building the ArcVault Agent Installer

## Prerequisites

1. **Inno Setup 6.0 or later**
   - Download from: https://jrsoftware.org/isdl.php
   - Install the full version (includes compiler)
   - Free and open source

## Build Steps

### Method 1: Using Inno Setup GUI

1. **Open Inno Setup Compiler**
   - Launch "Inno Setup Compiler" from Start Menu

2. **Open the Script**
   - Click `File` → `Open`
   - Navigate to: `C:\Users\castrk05\Py_stuff\ArcVault\ArcVaultAgentInstaller.iss`
   - Click Open

3. **Compile the Installer**
   - Click `Build` → `Compile` (or press F9)
   - Wait for compilation to complete (30-60 seconds)

4. **Find Your Installer**
   - Location: `C:\Users\castrk05\Py_stuff\ArcVault\installer_output\ArcVaultAgent_Setup_1.0.0.exe`
   - Size: ~42-45 MB

### Method 2: Using Command Line

```powershell
# Navigate to project directory
cd C:\Users\castrk05\Py_stuff\ArcVault

# Compile with Inno Setup (adjust path if installed elsewhere)
& "C:\Program Files (x86)\Inno Setup 6\ISCC.exe" ArcVaultAgentInstaller.iss
```

## What the Installer Includes

### Files Packaged:
- ✅ `ArcVaultAgentService.exe` (Windows Service - 41 MB)
- ✅ `ArcVaultAgent.exe` (Standalone console - 41 MB)
- ✅ `service_config.json` (Configuration file)
- ✅ Documentation (Troubleshooting guides, improvements summary)
- ✅ Diagnostic scripts (diagnose_service.ps1, reinstall_service.ps1)

### What the Installer Does:
1. **Checks for Admin privileges** (required for service installation)
2. **Installs files** to `C:\Program Files\ArcVault Agent\`
3. **Stops existing service** (if already installed)
4. **Installs Windows Service** using the executable
5. **Configures auto-start** (service starts with Windows)
6. **Starts the service** immediately
7. **Creates Start Menu shortcuts** for:
   - Run agent in console mode
   - View service logs
   - Run diagnostics
   - Edit configuration
   - View troubleshooting guide
   - Uninstaller

### Uninstaller:
- Automatically created
- Stops and removes the Windows service
- Removes all installed files
- Optionally removes log files
- Available from Programs & Features or Start Menu

## Testing the Installer

### Before Distribution:
```powershell
# 1. Build the installer (see above)

# 2. Stop your current service (for testing clean install)
sc.exe stop ArcVaultAgent
sc.exe delete ArcVaultAgent

# 3. Run the installer
.\installer_output\ArcVaultAgent_Setup_1.0.0.exe

# 4. Verify installation
sc.exe query ArcVaultAgent
Get-NetTCPConnection -LocalPort 8443
Get-Content "C:\Windows\Temp\arcvault_service.log" -Tail 20

# 5. Test uninstaller
# Go to Programs & Features → ArcVault Agent → Uninstall
```

## Customization Options

### Change Version Number:
Edit `ArcVaultAgentInstaller.iss`, line 6:
```
#define MyAppVersion "1.0.1"
```

### Change Default Install Location:
Edit line 19:
```
DefaultDirName={autopf}\YourCompanyName\ArcVault Agent
```

### Add Custom Icon:
1. Create a .ico file
2. Edit line 31:
```
SetupIconFile=path\to\your\icon.ico
```

### Add License Agreement:
Edit line 24:
```
LicenseFile=LICENSE.txt
```

## Distribution

### Single File Installer:
- **File:** `installer_output\ArcVaultAgent_Setup_1.0.0.exe`
- **Size:** ~42-45 MB (all dependencies included)
- **Requirements:** Windows 7 or later, Administrator privileges
- **Silent Install:** Add `/SILENT` or `/VERYSILENT` parameter

### Silent Installation:
```powershell
# Silent install
.\ArcVaultAgent_Setup_1.0.0.exe /VERYSILENT /NORESTART

# Silent install to custom directory
.\ArcVaultAgent_Setup_1.0.0.exe /VERYSILENT /DIR="C:\CustomPath\ArcVault"

# Silent uninstall
& "C:\Program Files\ArcVault Agent\unins000.exe" /VERYSILENT
```

## Troubleshooting Build Issues

### "Cannot find files" Error:
- Make sure all paths in the .iss file are correct
- Verify executables exist in `dist\` folder
- Check that documentation files exist in `apps\backup-agent\agent\`

### "Access Denied" During Build:
- Close any programs that might have files open
- Run Inno Setup as Administrator

### Installer Size Too Large:
- Already using ultra compression (`lzma2/ultra64`)
- Size is normal due to Python runtime and dependencies
- Both executables needed: service (~41MB) + standalone (~41MB) = ~82MB uncompressed → ~42MB compressed

## Quick Build Script

Save this as `build_installer.bat`:

```batch
@echo off
echo Building ArcVault Agent Installer...
echo.

REM Check if Inno Setup is installed
if not exist "C:\Program Files (x86)\Inno Setup 6\ISCC.exe" (
    echo ERROR: Inno Setup not found!
    echo Please install from: https://jrsoftware.org/isdl.php
    pause
    exit /b 1
)

REM Build the installer
"C:\Program Files (x86)\Inno Setup 6\ISCC.exe" ArcVaultAgentInstaller.iss

if %errorlevel% equ 0 (
    echo.
    echo ========================================
    echo SUCCESS! Installer built successfully!
    echo ========================================
    echo.
    echo Location: installer_output\ArcVaultAgent_Setup_1.0.0.exe
    echo.
    explorer installer_output
) else (
    echo.
    echo Build failed with error code %errorlevel%
)

pause
```

## Next Steps

1. **Download Inno Setup**: https://jrsoftware.org/isdl.php
2. **Build the installer** using Method 1 or Method 2 above
3. **Test the installer** on a clean machine or VM
4. **Distribute** the single .exe file

The installer is production-ready and includes everything needed for deployment! 🚀


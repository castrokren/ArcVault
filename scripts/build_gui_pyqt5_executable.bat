@echo off
setlocal ENABLEDELAYEDEXPANSION

echo ========================================
echo ArcVault PyQt5 GUI Executable Builder
echo ========================================
echo.

REM Set paths
set "PROJECT_ROOT=%~dp0.."
set "GUI_DIR=%PROJECT_ROOT%\installers-exe"
set "VENV_PATH=%TEMP%\arcvault_pyqt5_venv"

pushd "%PROJECT_ROOT%"

REM Create virtual environment for GUI build
echo [1/5] Setting up build environment...
if not exist "%VENV_PATH%" (
    python -m venv "%VENV_PATH%"
)

REM Activate virtual environment
call "%VENV_PATH%\Scripts\activate"

REM Install dependencies
echo [2/5] Installing dependencies...
python -m pip install --upgrade pip
python -m pip install pyinstaller PyQt5 cryptography psutil pywin32

REM Clean previous builds
echo [3/5] Cleaning previous builds...
if exist "build" rmdir /s /q "build" 2>nul
if exist "dist" rmdir /s /q "dist" 2>nul
if exist "%GUI_DIR%\ArcVault-GUI-PyQt5.exe" del /q "%GUI_DIR%\ArcVault-GUI-PyQt5.exe" 2>nul

REM Build PyQt5 GUI executable
echo [4/5] Building PyQt5 GUI executable...

REM Check if nssm.exe exists and use absolute path
set "NSSM_ARG="
if exist "%PROJECT_ROOT%\nssm.exe" (
    echo Including nssm.exe in build...
    set NSSM_ARG=--add-data "%PROJECT_ROOT%\nssm.exe;."
) else (
    echo WARNING: nssm.exe not found in project root, skipping...
)

REM Check if agent executable exists and bundle it
set "AGENT_ARG="
if exist "%PROJECT_ROOT%\apps\backup-agent\dist\ArcVault.exe" (
    echo Including ArcVault.exe agent in build...
    set AGENT_ARG=--add-data "%PROJECT_ROOT%\apps\backup-agent\dist\ArcVault.exe;."
) else (
    echo WARNING: ArcVault.exe not found. Expected: apps\backup-agent\dist\ArcVault.exe
    echo Build the agent first using: cd apps\backup-agent ^&^& .\build.ps1
)

pyinstaller --noconfirm --onefile ^
  --name "ArcVault-GUI-PyQt5" ^
  --distpath "%GUI_DIR%" ^
  --workpath "%TEMP%\pyi_pyqt5_gui_%RANDOM%" ^
  --specpath "%TEMP%\pyi_pyqt5_gui_spec_%RANDOM%" ^
  --windowed ^
  %NSSM_ARG% ^
  %AGENT_ARG% ^
  --hidden-import=PyQt5 ^
  --hidden-import=PyQt5.QtWidgets ^
  --hidden-import=PyQt5.QtCore ^
  --hidden-import=PyQt5.QtGui ^
  --hidden-import=cryptography ^
  --hidden-import=psutil ^
  --hidden-import=win32event ^
  --hidden-import=win32api ^
  --hidden-import=winerror ^
  --hidden-import=cryptography.hazmat.primitives ^
  --hidden-import=cryptography.hazmat.primitives.asymmetric.rsa ^
  --hidden-import=cryptography.hazmat.primitives.hashes ^
  --hidden-import=cryptography.hazmat.primitives.serialization ^
  --hidden-import=cryptography.x509 ^
  --collect-all cryptography ^
  --collect-all PyQt5 ^
  "scripts\arcvault_gui_pyqt5.py"

if %ERRORLEVEL% NEQ 0 (
    echo ERROR: PyQt5 GUI executable build failed
    popd
    exit /b %ERRORLEVEL%
)

REM Verify build
if exist "%GUI_DIR%\ArcVault-GUI-PyQt5.exe" (
    echo.
    echo ========================================
    echo PyQt5 GUI Executable Build Complete
    echo ========================================
    echo.
    echo Created: %GUI_DIR%\ArcVault-GUI-PyQt5.exe
    echo.
    echo Features:
    echo - Professional PyQt5 interface with modern styling
    echo - SSL Certificate Generation with advanced options
    echo - JWT Key Generation with progress tracking
    echo - Client Installation Management
    echo - Windows Service Control with real-time feedback
    echo - Agent Configuration and connection testing
    echo - Threaded operations for responsive UI
    echo - Comprehensive logging and error handling
    echo.
    echo Usage:
    echo   Double-click ArcVault-GUI-PyQt5.exe to launch
    echo   or run from command line
    echo.
    
    REM Get file size
    echo File size:
    powershell -Command "(Get-Item '%GUI_DIR%\ArcVault-GUI-PyQt5.exe').Length / 1MB"
    echo MB
    echo.
) else (
    echo ERROR: PyQt5 GUI executable was not created
    popd
    exit /b 1
)

REM Test the executable
echo [5/5] Testing executable...
echo Testing if the executable launches correctly...
timeout /t 2 /nobreak >nul
echo Test complete. You can now launch the GUI manually.

REM Clean up build artifacts
rmdir /s /q "build" 2>nul
rmdir /s /q "dist" 2>nul

popd
echo.
echo ========================================
echo Build Complete!
echo ========================================
echo.
echo The new PyQt5 GUI provides:
echo - Modern, professional appearance
echo - Better performance and responsiveness  
echo - Advanced UI controls and styling
echo - Improved user experience
echo - Thread-safe operations
echo - Real-time progress indicators
echo.
echo Launch: %GUI_DIR%\ArcVault-GUI-PyQt5.exe

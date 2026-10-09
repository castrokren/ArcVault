@echo off
setlocal ENABLEDELAYEDEXPANSION

echo ========================================
echo ArcVault GUI Executable Builder
echo ========================================
echo.

REM Set paths
set "PROJECT_ROOT=%~dp0.."
set "GUI_DIR=%PROJECT_ROOT%\installers-exe"
set "VENV_PATH=%TEMP%\arcvault_gui_venv"

pushd "%PROJECT_ROOT%"

REM Create virtual environment for GUI build
echo [1/4] Setting up build environment...
if not exist "%VENV_PATH%" (
    python -m venv "%VENV_PATH%"
)

REM Activate virtual environment
call "%VENV_PATH%\Scripts\activate"

REM Install dependencies
echo [2/4] Installing dependencies...
python -m pip install --upgrade pip
python -m pip install pyinstaller cryptography

REM Clean previous builds
echo [3/4] Cleaning previous builds...
if exist "build" rmdir /s /q "build" 2>nul
if exist "dist" rmdir /s /q "dist" 2>nul
if exist "%GUI_DIR%\ArcVault-GUI.exe" del /q "%GUI_DIR%\ArcVault-GUI.exe" 2>nul

REM Build GUI executable
echo [4/4] Building GUI executable...
pyinstaller --noconfirm --onefile ^
  --name "ArcVault-GUI" ^
  --distpath "%GUI_DIR%" ^
  --workpath "%TEMP%\pyi_gui_%RANDOM%" ^
  --specpath "%TEMP%\pyi_gui_spec_%RANDOM%" ^
  --windowed ^
  --hidden-import=tkinter ^
  --hidden-import=tkinter.ttk ^
  --hidden-import=tkinter.filedialog ^
  --hidden-import=tkinter.messagebox ^
  --hidden-import=tkinter.scrolledtext ^
  --hidden-import=cryptography ^
  --hidden-import=cryptography.hazmat.primitives ^
  --hidden-import=cryptography.hazmat.primitives.asymmetric.rsa ^
  --hidden-import=cryptography.hazmat.primitives.hashes ^
  --hidden-import=cryptography.hazmat.primitives.serialization ^
  --hidden-import=cryptography.x509 ^
  --collect-all cryptography ^
  "scripts\arcvault_gui.py"

if %ERRORLEVEL% NEQ 0 (
    echo ERROR: GUI executable build failed
    popd
    exit /b %ERRORLEVEL%
)

REM Verify build
if exist "%GUI_DIR%\ArcVault-GUI.exe" (
    echo.
    echo ========================================
    echo GUI Executable Build Complete
    echo ========================================
    echo.
    echo Created: %GUI_DIR%\ArcVault-GUI.exe
    echo.
    echo Features:
    echo - SSL Certificate Generation
    echo - JWT Key Generation
    echo - Client Installation Management
    echo - Windows Service Control
    echo - Agent Configuration
    echo.
    echo Usage:
    echo   Double-click ArcVault-GUI.exe to launch
    echo   or run from command line
    echo.
) else (
    echo ERROR: GUI executable was not created
    popd
    exit /b 1
)

REM Clean up build artifacts
rmdir /s /q "build" 2>nul
rmdir /s /q "dist" 2>nul

popd
echo Build complete!

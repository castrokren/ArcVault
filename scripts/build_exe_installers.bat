@echo off
setlocal ENABLEDELAYEDEXPANSION

REM Build executable installers from .bat files
REM This converts all .bat files to .exe for easier deployment

echo ========================================
echo ArcVault EXE Installer Builder
echo ========================================
echo.

REM Determine repo root
set "SCRIPT_DIR=%~dp0"
pushd "%SCRIPT_DIR%\.."

REM Create exe installers directory (do not remove on rebuild to preserve outputs if later steps fail)
set "EXE_DIR=installers-exe"
if not exist "%EXE_DIR%" mkdir "%EXE_DIR%"

echo Building EXE installers...
echo.

REM Create virtual environment for building
set "PYTHON_EXE=C:\Users\castrk05\AppData\Local\Programs\Python\Python311\python.exe"
set "VENV_PATH=%TEMP%\arcvault_exe_venv"
if not exist "%VENV_PATH%" (
    "%PYTHON_EXE%" -m venv "%VENV_PATH%"
)
call "%VENV_PATH%\Scripts\activate"
python -m pip install --upgrade pip
python -m pip install pyinstaller cryptography

REM Build client EXE installer (NSIS)
echo [1/4] Building client EXE installer (NSIS)...
if not exist "deployment-client" (
    echo deployment-client not found. Building client package first...
    call "scripts\build_client_package.bat"
    if ERRORLEVEL 1 (
        echo ERROR: Client package build failed
        popd
        exit /b %ERRORLEVEL%
    )
)

if exist "deployment-client" (
    REM NSIS is available - proceed with build

    echo Generating NSIS script...
    if not exist "temp-exe-build" mkdir "temp-exe-build"
    del /q /f "temp-exe-build\ArcVault-Client-Installer.nsi" 2>nul
    del /q /f "installers-exe\ArcVault-Client-Installer.exe" 2>nul

    (
        echo ^^!include "MUI2.nsh"
        echo OutFile "%CD%\installers-exe\ArcVault-Client-Installer.exe"
        echo Name "ArcVault Client Installer"
        echo RequestExecutionLevel admin
        echo InstallDir "$PROGRAMFILES\ArcVault"
        echo BrandingText "ArcVault"
        echo ^^!insertmacro MUI_PAGE_WELCOME
        echo ^^!insertmacro MUI_PAGE_DIRECTORY
        echo ^^!insertmacro MUI_PAGE_INSTFILES
        echo ^^!insertmacro MUI_PAGE_FINISH
        echo ^^!insertmacro MUI_LANGUAGE "English"
        echo Section "Install"
        echo   SetOutPath "$INSTDIR"
        echo   File /r "%CD%\deployment-client\*"
        echo   ExecShell "open" "$INSTDIR\ArcVault-Agent.bat"
        echo SectionEnd
    ) > "temp-exe-build\ArcVault-Client-Installer.nsi"

    echo Building NSIS package...
    "C:\Program Files (x86)\NSIS\Bin\makensis.exe" /V2 "temp-exe-build\ArcVault-Client-Installer.nsi"

    if not exist "installers-exe\ArcVault-Client-Installer.exe" (
        echo ERROR: Failed to create ArcVault-Client-Installer.exe via NSIS
        popd
        exit /b 1
    )
    echo Client installer created: %EXE_DIR%\ArcVault-Client-Installer.exe
) else (
    echo ERROR: deployment-client still not found after build attempt.
    popd
    exit /b 1
)

REM Build server EXE installers
echo [2/4] Building server EXE installers...
if not exist "deployment-server" (
    echo deployment-server not found. Building server package first...
    call "scripts\build_server_package.bat"
    if ERRORLEVEL 1 (
        echo WARNING: Server package build failed. Skipping server installer.
        goto AFTER_SERVER
    )
)
if exist "deployment-server" (
    pyinstaller --noconfirm  --onefile ^
      --workpath "%TEMP%\pyi_server_%RANDOM%" ^
      --specpath "%TEMP%\pyi_server_spec_%RANDOM%" ^
      --name ArcVault-Server-Installer ^
      --distpath "%EXE_DIR%" ^
      --add-data "deployment-server;deployment-server" ^
      --hidden-import=subprocess ^
      --hidden-import=os ^
      --hidden-import=sys ^
      --console ^
      "scripts\server_installer_wrapper.py"
    
    if ERRORLEVEL 1 (
        echo WARNING: Server installer build failed. Continuing.
        goto AFTER_SERVER
    )
) else (
    echo WARNING: deployment-server still not found after build attempt. Skipping server installer.
)

:AFTER_SERVER

REM Build certificate generator
echo [3/4] Building certificate generator...
pyinstaller --noconfirm  --onefile ^
  --workpath "%TEMP%\pyi_cert_%RANDOM%" ^
  --specpath "%TEMP%\pyi_cert_spec_%RANDOM%" ^
  --name ArcVault-Cert-Generator ^
  --distpath "%EXE_DIR%" ^
  --collect-all cryptography ^
  --console ^
  "scripts\generate_self_signed_cert.py"

if ERRORLEVEL 1 (
    echo WARNING: Certificate generator build failed. Continuing.
)

REM Build JWT key generator
echo [4/4] Building JWT key generator...
pyinstaller --noconfirm  --onefile ^
  --workpath "%TEMP%\pyi_jwt_%RANDOM%" ^
  --specpath "%TEMP%\pyi_jwt_spec_%RANDOM%" ^
  --name ArcVault-JWT-Generator ^
  --distpath "%EXE_DIR%" ^
  --collect-all cryptography ^
  --console ^
  "scripts\generate_jwt_keys.py"

if ERRORLEVEL 1 (
    echo WARNING: JWT key generator build failed. Continuing.
)

REM Create README for EXE installers
(
echo ArcVault EXE Installers
echo =======================
echo.
echo This directory contains self-contained executable installers for ArcVault.
echo.
echo CONTENTS:
echo ---------
echo ArcVault-Client-Installer.exe    - Client package installer
echo ArcVault-Server-Installer.exe    - Server package installer
echo ArcVault-Cert-Generator.exe      - Self-signed certificate generator
echo ArcVault-JWT-Generator.exe       - JWT key pair generator
echo.
echo USAGE:
echo ------
echo 1. CLIENT INSTALLATION:
echo    - Run ArcVault-Client-Installer.exe on client machines
echo    - Follow the setup wizard
echo    - Configure connection to backup server
echo.
echo 2. SERVER INSTALLATION:
echo    - Run ArcVault-Server-Installer.exe on backup server
echo    - Follow the setup wizard
echo    - Generate JWT keys and certificates
echo.
echo 3. CERTIFICATE GENERATION:
echo    - Run ArcVault-Cert-Generator.exe to create self-signed certificates
echo    - Use for HTTPS communication without external certificates
echo.
echo 4. JWT KEY GENERATION:
echo    - Run ArcVault-JWT-Generator.exe to create JWT key pairs
echo    - Use for secure authentication between server and clients
echo.
echo FEATURES:
echo ---------
echo - Self-contained executables (no Python required)
echo - Automated setup wizards
echo - Self-signed certificate generation
echo - JWT authentication setup
echo - Windows service management
echo - Cross-platform server deployment (Docker)
echo.
echo SECURITY:
echo ---------
echo - Self-signed certificates for internal use
echo - JWT authentication with RSA keys
echo - Secure key storage with proper permissions
echo - HTTPS communication on port 443
echo.
echo SUPPORT:
echo --------
echo For issues and support, visit: https://github.com/castrokren/arcvault-tool
) > "%EXE_DIR%\README.txt"

echo.
echo ========================================
echo EXE Installers Complete!
echo ========================================
echo.
echo Created in: %EXE_DIR%\
echo.
echo Contents:
echo - ArcVault-Client-Installer.exe (Client package installer)
echo - ArcVault-Server-Installer.exe (Server package installer)
echo - ArcVault-Cert-Generator.exe (Certificate generator)
echo - ArcVault-JWT-Generator.exe (JWT key generator)
echo - README.txt (Documentation)
echo.
echo To deploy:
echo 1. Copy EXE files to target machines
echo 2. Run as Administrator
echo 3. Follow setup wizards
echo.
echo Package sizes:
for /f %%i in ('dir "%EXE_DIR%" /s /-c ^| find "File(s)"') do echo %%i bytes

popd
endlocal

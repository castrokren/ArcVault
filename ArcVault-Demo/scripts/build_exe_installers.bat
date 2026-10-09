@echo off
setlocal ENABLEDELAYEDEXPANSION

REM Build executable installers from .bat files
REM This converts all .bat files to .exe for easier deployment

echo ========================================
echo ArcVault EXE Installer Builder
echo ========================================
echo.

REM Determine repo root
set SCRIPT_DIR=%~dp0
pushd "%SCRIPT_DIR%\.."

REM Create exe installers directory
set EXE_DIR=installers-exe
if exist "%EXE_DIR%" (
    echo Removing existing EXE installers directory...
    rmdir /s /q "%EXE_DIR%"
)
mkdir "%EXE_DIR%"

echo Building EXE installers...
echo.

REM Create virtual environment for building
if not exist .venv_exe (
    python -m venv .venv_exe
)
call .venv_exe\Scripts\activate
python -m pip install --upgrade pip
python -m pip install pyinstaller

REM Build client EXE installers
echo [1/4] Building client EXE installers...
if exist "deployment-client" (
    pyinstaller --noconfirm --clean --onefile ^
      --name ArcVault-Client-Installer ^
      --distpath "%EXE_DIR%" ^
      --add-data "deployment-client;deployment-client" ^
      --hidden-import=subprocess ^
      --hidden-import=os ^
      --hidden-import=sys ^
      --console ^
      scripts\client_installer_wrapper.py
    
    if %ERRORLEVEL% NEQ 0 (
        echo ERROR: Client installer build failed
        popd
        exit /b %ERRORLEVEL%
    )
) else (
    echo WARNING: deployment-client not found. Run build_client_package.bat first.
)

REM Build server EXE installers
echo [2/4] Building server EXE installers...
if exist "deployment-server" (
    pyinstaller --noconfirm --clean --onefile ^
      --name ArcVault-Server-Installer ^
      --distpath "%EXE_DIR%" ^
      --add-data "deployment-server;deployment-server" ^
      --hidden-import=subprocess ^
      --hidden-import=os ^
      --hidden-import=sys ^
      --console ^
      scripts\server_installer_wrapper.py
    
    if %ERRORLEVEL% NEQ 0 (
        echo ERROR: Server installer build failed
        popd
        exit /b %ERRORLEVEL%
    )
) else (
    echo WARNING: deployment-server not found. Run build_server_package.bat first.
)

REM Build certificate generator
echo [3/4] Building certificate generator...
pyinstaller --noconfirm --clean --onefile ^
  --name ArcVault-Cert-Generator ^
  --distpath "%EXE_DIR%" ^
  --hidden-import=cryptography ^
  --hidden-import=cryptography.hazmat ^
  --hidden-import=cryptography.hazmat.primitives ^
  --hidden-import=cryptography.hazmat.primitives.asymmetric ^
  --hidden-import=cryptography.hazmat.primitives.serialization ^
  --hidden-import=cryptography.hazmat.primitives.hashes ^
  --hidden-import=cryptography.x509 ^
  --hidden-import=cryptography.x509.oid ^
  --console ^
  scripts\generate_self_signed_cert.py

if %ERRORLEVEL% NEQ 0 (
    echo ERROR: Certificate generator build failed
    popd
    exit /b %ERRORLEVEL%
)

REM Build JWT key generator
echo [4/4] Building JWT key generator...
pyinstaller --noconfirm --clean --onefile ^
  --name ArcVault-JWT-Generator ^
  --distpath "%EXE_DIR%" ^
  --hidden-import=cryptography ^
  --hidden-import=cryptography.hazmat ^
  --hidden-import=cryptography.hazmat.primitives ^
  --hidden-import=cryptography.hazmat.primitives.asymmetric ^
  --hidden-import=cryptography.hazmat.primitives.serialization ^
  --console ^
  scripts\generate_jwt_keys.py

if %ERRORLEVEL% NEQ 0 (
    echo ERROR: JWT key generator build failed
    popd
    exit /b %ERRORLEVEL%
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

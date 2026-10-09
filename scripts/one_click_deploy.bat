@echo off
setlocal ENABLEDELAYEDEXPANSION

REM One-Click Deployment Script for ArcVault
REM This script builds everything and creates a complete deployment package

echo ========================================
echo ArcVault One-Click Deployment Builder
echo ========================================
echo.
echo This script will:
echo 1. Build client package (for agents)
echo 2. Build server package (for controller)
echo 3. Create installers for both
echo 4. Generate documentation
echo.
echo Choose deployment type:
echo 1. Client Package (Windows agents only)
echo 2. Server Package (Cross-platform controller)
echo 3. Both Packages
echo 4. Unified Installer (Everything in one package)
echo 5. Demo Package (For sharing with friends)
echo.
set /p DEPLOY_CHOICE="Enter your choice (1-5): "
echo.

REM Check for Python
set "PYTHON_EXE=C:\Users\castrk05\AppData\Local\Programs\Python\Python311\python.exe"
"%PYTHON_EXE%" --version >nul 2>&1
if %ERRORLEVEL% NEQ 0 (
    echo ERROR: Python is not installed or not in PATH
    echo Please install Python 3.9+ and try again
    pause
    exit /b 1
)

REM Check for required tools
where pyinstaller >nul 2>&1
if %ERRORLEVEL% NEQ 0 (
    echo Installing PyInstaller...
    pip install pyinstaller
)

where iexpress >nul 2>&1
if %ERRORLEVEL% NEQ 0 (
    echo WARNING: IExpress not found. Will create batch installer instead.
    set USE_IEXPRESS=false
) else (
    set USE_IEXPRESS=true
)

echo.
echo Starting build process...
echo.

REM Step 1: Build packages based on choice
if "%DEPLOY_CHOICE%"=="1" (
    echo [1/4] Building client package...
    call "scripts\build_client_package.bat"
    if %ERRORLEVEL% NEQ 0 (
        echo ERROR: Client package build failed
        pause
        exit /b %ERRORLEVEL%
    )
) else if "%DEPLOY_CHOICE%"=="2" (
    echo [1/4] Building server package...
    call "scripts\build_server_package.bat"
    if %ERRORLEVEL% NEQ 0 (
        echo ERROR: Server package build failed
        pause
        exit /b %ERRORLEVEL%
    )
) else if "%DEPLOY_CHOICE%"=="3" (
    echo [1/4] Building client package...
    call "scripts\build_client_package.bat"
    if %ERRORLEVEL% NEQ 0 (
        echo ERROR: Client package build failed
        pause
        exit /b %ERRORLEVEL%
    )
    echo [1/4] Building server package...
    call "scripts\build_server_package.bat"
    if %ERRORLEVEL% NEQ 0 (
        echo ERROR: Server package build failed
        pause
        exit /b %ERRORLEVEL%
    )
) else if "%DEPLOY_CHOICE%"=="4" (
    echo [1/4] Building unified installer...
    call "scripts\build_unified_installer.bat"
    if %ERRORLEVEL% NEQ 0 (
        echo ERROR: Unified installer build failed
        pause
        exit /b %ERRORLEVEL%
    )
) else if "%DEPLOY_CHOICE%"=="5" (
    echo [1/4] Creating demo package...
    call "scripts\create_demo_package.bat"
    if %ERRORLEVEL% NEQ 0 (
        echo ERROR: Demo package creation failed
        pause
        exit /b %ERRORLEVEL%
    )
) else (
    echo ERROR: Invalid choice. Please run the script again.
    pause
    exit /b 1
)

REM Step 2: Create EXE installers (Windows only) - Skip for unified installer and demo
if not "%DEPLOY_CHOICE%"=="4" if not "%DEPLOY_CHOICE%"=="5" (
    echo [2/4] Creating EXE installers...
    call scripts\build_exe_installers.bat
    if %ERRORLEVEL% NEQ 0 (
        echo ERROR: EXE installer build failed
        pause
        exit /b %ERRORLEVEL%
    )
) else (
    echo [2/4] Skipping EXE installers (included in package)...
)

REM Step 3: Create deployment documentation
echo [3/4] Creating deployment documentation...

(
echo # ArcVault Production Deployment Guide
echo.
echo This guide explains how to deploy ArcVault in production environments.
echo.
echo ## Quick Deployment
echo.
echo ### Option 1: Single-File Installer
echo 1. Copy `ArcVault-Installer.exe` to your target server
echo 2. Run as Administrator
echo 3. Follow the setup wizard
echo.
echo ### Option 2: ZIP Package
echo 1. Copy `ArcVault-Deployment.zip` to your target server
echo 2. Extract to a folder
echo 3. Run `ArcVault-Deploy.bat`
echo.
echo ### Option 3: Direct Copy
echo 1. Copy the entire `deployment` folder to your target server
echo 2. Run `ArcVault-Deploy.bat`
echo.
echo ## System Requirements
echo.
echo - Windows 10/11 or Windows Server 2016+
echo - Administrator privileges for installation
echo - Network access for agent communication
echo - Ports 8443 (agent) and 8501 (controller) available
echo.
echo ## Installation Steps
echo.
echo 1. **Run Installer**
echo    - Execute the installer as Administrator
echo    - Choose installation directory (default: Program Files\ArcVault)
echo.
echo 2. **Configure Environment**
echo    - Run the setup wizard
echo    - Select environment type (development/staging/production)
echo    - Enter server hostname
echo    - JWT keys are generated automatically
echo.
echo 3. **SSL Setup (Production)**
echo    - Copy SSL certificates to ProgramData\ArcVault\keys\
echo    - Required files: agent.crt, agent.key, ca-bundle.crt
echo.
echo 4. **Start Services**
echo    - Use Service Manager to install Agent Service
echo    - Start Controller (interactive or background)
echo.
echo ## Access Points
echo.
echo - **Agent API**: https://your-server:8443
echo - **Controller UI**: http://your-server:8501
echo - **Health Check**: https://your-server:8443/health
echo.
echo ## Service Management
echo.
echo Use the Service Manager for:
echo - Installing/removing Windows Service
echo - Starting/stopping services
echo - Viewing service status
echo - Accessing logs
echo.
echo ## Troubleshooting
echo.
echo ### Common Issues
echo.
echo **Service won't start**
echo - Check Windows Event Log
echo - Verify configuration in ProgramData\ArcVault\.env
echo - Ensure ports are not in use
echo.
echo **SSL Certificate Errors**
echo - Verify certificate files exist and are readable
echo - Check certificate validity and hostname match
echo - Ensure CA bundle is correct
echo.
echo **JWT Authentication Errors**
echo - Verify JWT keys are generated and accessible
echo - Check issuer and audience settings
echo - Ensure key permissions are correct
echo.
echo **Network Connectivity Issues**
echo - Check firewall settings
echo - Verify agent URLs in controller configuration
echo - Test network connectivity between components
echo.
echo ### Log Locations
echo.
echo - **Service Logs**: Windows Event Log (Application)
echo - **Backup Logs**: ProgramData\ArcVault\logs\
echo - **Configuration**: ProgramData\ArcVault\.env
echo.
echo ## Security Considerations
echo.
echo - Use JWT authentication in production
echo - Enable SSL/TLS for all communications
echo - Restrict network access with firewalls
echo - Regularly rotate JWT keys
echo - Monitor service logs for security events
echo.
echo ## Support
echo.
echo For issues and support:
echo - GitHub: https://github.com/castrokren/arcvault-tool
echo - Documentation: See README files in deployment package
echo - Logs: Check Windows Event Log and backup logs
) > "DEPLOYMENT_GUIDE.md"

REM Step 4: Create summary
echo [4/4] Creating deployment summary...

echo.
echo ========================================
echo ArcVault Deployment Packages Complete!
echo ========================================
echo.
echo Created files:
echo.

if exist "deployment-client" (
    echo ✅ Client Package: deployment-client\
    echo    - ArcVaultAgent.exe (Backup agent)
    echo    - ArcVaultAgentService.exe (Windows service)
    echo    - Configuration and setup scripts
    echo    - Ready for client machine deployment
    echo.
)

if exist "deployment-server" (
    echo ✅ Server Package: deployment-server\
    echo    - ArcVaultController.exe (Web control panel)
    echo    - ArcVaultKeyGen.exe (JWT key generator)
    echo    - Configuration and setup scripts
    echo    - Ready for server deployment
    echo.
)

if exist "installers-exe\ArcVault-Client-Installer.exe" (
    echo ✅ Client Installer: installers-exe\ArcVault-Client-Installer.exe
    echo    - Self-contained EXE installer for client machines
    echo.
)

if exist "installers-exe\ArcVault-Server-Installer.exe" (
    echo ✅ Server Installer: installers-exe\ArcVault-Server-Installer.exe
    echo    - Self-contained EXE installer for backup server
    echo.
)

if exist "installers-exe\ArcVault-Cert-Generator.exe" (
    echo ✅ Certificate Generator: installers-exe\ArcVault-Cert-Generator.exe
    echo    - Self-signed certificate generator for HTTPS
    echo.
)

if exist "installers-exe\ArcVault-JWT-Generator.exe" (
    echo ✅ JWT Generator: installers-exe\ArcVault-JWT-Generator.exe
    echo    - JWT key pair generator for authentication
    echo.
)

if exist "DEPLOYMENT_GUIDE.md" (
    echo ✅ Documentation: DEPLOYMENT_GUIDE.md
    echo    - Complete deployment instructions
    echo    - Troubleshooting guide
    echo    - Security considerations
    echo.
)

echo Deployment Options:
echo ===================
echo.
if "%DEPLOY_CHOICE%"=="1" (
    echo CLIENT DEPLOYMENT:
    echo 1. Copy deployment-client folder to target client machine
    echo 2. Run ArcVault-Agent.bat
    echo 3. Follow setup wizard to configure connection to backup server
    echo 4. Start agent service
    echo.
) else if "%DEPLOY_CHOICE%"=="2" (
    echo SERVER DEPLOYMENT:
    echo 1. Copy deployment-server folder to target backup server
    echo 2. Run ArcVault-Server.bat
    echo 3. Follow setup wizard to configure server
    echo 4. Start controller and manage agents
    echo.
) else if "%DEPLOY_CHOICE%"=="3" (
    echo CLIENT DEPLOYMENT:
    echo 1. Copy deployment-client folder to target client machines
    echo 2. Run ArcVault-Agent.bat on each client
    echo 3. Configure connection to backup server
    echo 4. Start agent services
    echo.
    echo SERVER DEPLOYMENT:
    echo 1. Copy deployment-server folder to target backup server
    echo 2. Run ArcVault-Server.bat
    echo 3. Configure server and generate JWT keys
    echo 4. Start controller and add agents
    echo.
)

echo Package Sizes:
if exist "deployment-client" (
    for /f %%i in ('dir "deployment-client" /s /-c 2^>nul ^| find "File(s)"') do echo - Client Package: %%i bytes
)
if exist "deployment-server" (
    for /f %%i in ('dir "deployment-server" /s /-c 2^>nul ^| find "File(s)"') do echo - Server Package: %%i bytes
)
if exist "installers-exe" (
    for /f %%i in ('dir "installers-exe" /s /-c 2^>nul ^| find "File(s)"') do echo - EXE Installers: %%i bytes
)

echo.
echo Next Steps:
echo ===========
if "%DEPLOY_CHOICE%"=="1" (
    echo 1. Test client EXE installer on a clean Windows system
    echo 2. Deploy ArcVault-Client-Installer.exe to Windows client machines
    echo 3. Run installer and configure connection to backup server
    echo 4. Start agent services
) else if "%DEPLOY_CHOICE%"=="2" (
    echo 1. Test server deployment on target system
    echo 2. Deploy using Docker: scripts\docker_deploy.bat (Windows)
    echo 3. OR deploy using Docker: scripts\docker_deploy.sh (macOS/Linux)
    echo 4. Configure server environment
    echo 5. Start controller and add agents
) else if "%DEPLOY_CHOICE%"=="3" (
    echo 1. Test both deployments on clean systems
    echo 2. Deploy server using Docker (any OS)
    echo 3. Deploy ArcVault-Client-Installer.exe to Windows client machines
    echo 4. Run installers and configure all components
    echo 5. Start all services
) else if "%DEPLOY_CHOICE%"=="4" (
    echo 1. Copy installer-unified folder to target machine
    echo 2. Run ArcVault-Unified.bat as Administrator
    echo 3. Choose deployment type (Client/Server/Both)
    echo 4. Follow setup wizard to configure environment
    echo 5. Use manager.bat to manage services
) else if "%DEPLOY_CHOICE%"=="5" (
    echo 1. Send ArcVault-Demo.zip to your friend
    echo 2. Friend extracts the zip file
    echo 3. Friend runs setup_demo.bat (installs dependencies)
    echo 4. Friend runs run_demo.bat (starts demo)
    echo 5. Friend goes to http://localhost:8501 to test
)
echo.
echo For detailed instructions, see DEPLOYMENT_GUIDE.md
echo.

pause

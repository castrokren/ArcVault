@echo off
setlocal ENABLEDELAYEDEXPANSION

REM Build a unified ArcVault installer that includes everything
REM This creates a single installer that handles client, server, certificates, and JWT keys

echo ========================================
echo ArcVault Unified Installer Builder
echo ========================================
echo.

REM Determine repo root
set "SCRIPT_DIR=%~dp0"
pushd "%SCRIPT_DIR%\.."

REM Create unified installer directory
set "UNIFIED_DIR=installer-unified"
if exist "%UNIFIED_DIR%" (
    echo Removing existing unified installer directory...
    rmdir /s /q "%UNIFIED_DIR%"
)
mkdir "%UNIFIED_DIR%"

echo Building unified installer...
echo.

REM Create subdirectories
mkdir "%UNIFIED_DIR%\client"
mkdir "%UNIFIED_DIR%\server"
mkdir "%UNIFIED_DIR%\tools"
mkdir "%UNIFIED_DIR%\docker"
mkdir "%UNIFIED_DIR%\config"

REM Build all components
echo [1/6] Building all components...

REM Build Agent EXE
echo Building Agent executable...
set "PYTHON_EXE=C:\Users\castrk05\AppData\Local\Programs\Python\Python311\python.exe"
if not exist .venv_agent (
    "%PYTHON_EXE%" -m venv .venv_agent
)
call ".venv_agent\Scripts\activate"
python -m pip install --upgrade pip
python -m pip install pyinstaller pywin32
python -m pip install -e "apps\backup-agent"

pyinstaller --noconfirm --clean --onefile --name ArcVaultAgent --distpath "%UNIFIED_DIR%\client" "apps\backup-agent\agent\run_agent.py"

pyinstaller --noconfirm --clean --onefile --name ArcVaultAgentService --distpath "%UNIFIED_DIR%\client" "apps\backup-agent\agent\win_service.py"

REM Build Controller EXE
echo Building Controller executable...
if not exist .venv_controller (
    "%PYTHON_EXE%" -m venv .venv_controller
)
call ".venv_controller\Scripts\activate"
python -m pip install --upgrade pip
python -m pip install pyinstaller
python -m pip install -e "apps\backup-control"

pyinstaller --noconfirm --clean --onefile --name ArcVaultController --distpath "%UNIFIED_DIR%\server" "apps\backup-control\controller\run_controller.py" --add-data "apps\backup-control\controller;apps\backup-control\controller"

REM Build utility tools
echo Building utility tools...
pyinstaller --noconfirm --clean --onefile --name ArcVaultCertGen --distpath "%UNIFIED_DIR%\tools" "scripts\generate_self_signed_cert.py"

pyinstaller --noconfirm --clean --onefile --name ArcVaultJWTGen --distpath "%UNIFIED_DIR%\tools" "scripts\generate_jwt_keys.py"

REM Copy Docker files
echo [2/6] Copying Docker files...
copy "Dockerfile.server" "%UNIFIED_DIR%\docker\"
copy "docker-compose.yml" "%UNIFIED_DIR%\docker\"
mkdir "%UNIFIED_DIR%\docker\nginx" 2>nul
copy "nginx\nginx.conf" "%UNIFIED_DIR%\docker\nginx\" 2>nul

REM Copy configuration templates
echo [3/6] Copying configuration templates...
copy "env.templates\*.env" "%UNIFIED_DIR%\config\"

REM Create unified installer script
echo [4/6] Creating unified installer...

(
echo @echo off
echo setlocal ENABLEDELAYEDEXPANSION
echo.
echo echo ========================================
echo echo ArcVault Unified Installer
echo echo ========================================
echo echo.
echo echo This installer will set up ArcVault for your environment.
echo echo It can install:
echo echo - Windows backup agents ^(clients^)
echo echo - Cross-platform backup server ^(Docker^)
echo echo - SSL certificates and JWT keys
echo echo - Complete configuration
echo echo.
echo.
echo REM Check for administrator privileges
echo net session ^>nul 2^>^&1
echo if %%errorLevel%% == 0 ^(
echo     echo Running with administrator privileges...
echo ^) else ^(
echo     echo ERROR: This installer requires administrator privileges.
echo     echo Please run as Administrator.
echo     pause
echo     exit /b 1
echo ^)
echo.
echo REM Set installation directories
echo set INSTALL_DIR=%%ProgramFiles%%\ArcVault
echo set CONFIG_DIR=%%ProgramData%%\ArcVault
echo set DOCKER_DIR=%%ProgramData%%\ArcVault\docker
echo.
echo echo Installing ArcVault to: %%INSTALL_DIR%%
echo echo Configuration directory: %%CONFIG_DIR%%
echo echo Docker directory: %%DOCKER_DIR%%
echo.
echo REM Create directories
echo mkdir "%%INSTALL_DIR%%" 2^>nul
echo mkdir "%%CONFIG_DIR%%" 2^>nul
echo mkdir "%%CONFIG_DIR%%\keys" 2^>nul
echo mkdir "%%CONFIG_DIR%%\logs" 2^>nul
echo mkdir "%%CONFIG_DIR%%\config" 2^>nul
echo mkdir "%%DOCKER_DIR%%" 2^>nul
echo.
echo REM Copy all executables
echo echo Copying ArcVault components...
echo copy "client\ArcVaultAgent.exe" "%%INSTALL_DIR%%\"
echo copy "client\ArcVaultAgentService.exe" "%%INSTALL_DIR%%\"
echo copy "server\ArcVaultController.exe" "%%INSTALL_DIR%%\"
echo copy "tools\ArcVaultCertGen.exe" "%%INSTALL_DIR%%\"
echo copy "tools\ArcVaultJWTGen.exe" "%%INSTALL_DIR%%\"
echo.
echo REM Copy Docker files
echo echo Copying Docker configuration...
echo xcopy "docker\*" "%%DOCKER_DIR%%\" /E /I /Y
echo.
echo REM Copy configuration templates
echo copy "config\*.env" "%%CONFIG_DIR%%\config\"
echo.
echo REM Set permissions
echo echo Setting permissions...
echo icacls "%%INSTALL_DIR%%" /inheritance:r /grant:r "Administrators:F" "SYSTEM:F" /T
echo icacls "%%CONFIG_DIR%%" /inheritance:r /grant:r "Administrators:F" "SYSTEM:F" /T
echo.
echo echo.
echo echo ========================================
echo echo Installation Complete!
echo echo ========================================
echo echo.
echo echo Next steps:
echo echo 1. Run: %%INSTALL_DIR%%\ArcVaultSetup.exe
echo echo 2. Choose your deployment type
echo echo 3. Configure your environment
echo echo 4. Start services
echo echo.
echo pause
) > "%UNIFIED_DIR%\install.bat" 2>nul

REM Create unified setup wizard
echo [5/6] Creating unified setup wizard...

(
echo @echo off
echo setlocal ENABLEDELAYEDEXPANSION
echo.
echo echo ========================================
echo echo ArcVault Unified Setup Wizard
echo echo ========================================
echo echo.
echo echo This wizard will configure ArcVault for your environment.
echo echo.
echo.
echo set INSTALL_DIR=%%ProgramFiles%%\ArcVault
echo set CONFIG_DIR=%%ProgramData%%\ArcVault
echo set DOCKER_DIR=%%ProgramData%%\ArcVault\docker
echo.
echo REM Deployment type selection
echo :DEPLOY_SELECT
echo echo Select your deployment type:
echo echo 1. Client Only ^(Windows backup agent^)
echo echo 2. Server Only ^(Cross-platform backup server^)
echo echo 3. Both Client and Server ^(Complete setup^)
echo echo.
echo set /p DEPLOY_CHOICE="Enter your choice (1-3): "
echo.
echo if "%%DEPLOY_CHOICE%%"=="1" ^(
echo     set DEPLOY_TYPE=client
echo ^) else if "%%DEPLOY_CHOICE%%"=="2" ^(
echo     set DEPLOY_TYPE=server
echo ^) else if "%%DEPLOY_CHOICE%%"=="3" ^(
echo     set DEPLOY_TYPE=both
echo ^) else ^(
echo     echo Invalid choice. Please try again.
echo     goto DEPLOY_SELECT
echo ^)
echo.
echo echo Selected deployment: %%DEPLOY_TYPE%%
echo.
echo REM Environment selection
echo :ENV_SELECT
echo echo Select your environment:
echo echo 1. Development ^(local testing^)
echo echo 2. Staging ^(pre-production testing^)
echo echo 3. Production ^(live deployment^)
echo echo.
echo set /p ENV_CHOICE="Enter your choice (1-3): "
echo.
echo if "%%ENV_CHOICE%%"=="1" ^(
echo     set ENV_TYPE=development
echo     set ENV_FILE=development.env
echo ^) else if "%%ENV_CHOICE%%"=="2" ^(
echo     set ENV_TYPE=staging
echo     set ENV_FILE=staging.env
echo ^) else if "%%ENV_CHOICE%%"=="3" ^(
echo     set ENV_TYPE=production
echo     set ENV_FILE=production.env
echo ^) else ^(
echo     echo Invalid choice. Please try again.
echo     goto ENV_SELECT
echo ^)
echo.
echo echo Selected environment: %%ENV_TYPE%%
echo.
echo REM Copy environment file
echo copy "%%CONFIG_DIR%%\config\%%ENV_FILE%%" "%%CONFIG_DIR%%\.env"
echo.
echo REM Generate certificates and JWT keys
echo echo.
echo echo Generating security components...
echo.
echo REM Generate self-signed certificates
echo echo Generating self-signed certificates...
echo %%INSTALL_DIR%%\ArcVaultCertGen.exe "%%CONFIG_DIR%%\keys" "%%COMPUTERNAME%%"
echo.
echo REM Generate JWT keys for production/staging
echo if "%%ENV_TYPE%%"=="production" ^(
echo     echo Generating JWT keys for production...
echo     %%INSTALL_DIR%%\ArcVaultJWTGen.exe "%%CONFIG_DIR%%\keys"
echo ^) else if "%%ENV_TYPE%%"=="staging" ^(
echo     echo Generating JWT keys for staging...
echo     %%INSTALL_DIR%%\ArcVaultJWTGen.exe "%%CONFIG_DIR%%\keys"
echo ^)
echo.
echo REM Configure based on deployment type
echo if "%%DEPLOY_TYPE%%"=="client" goto CONFIGURE_CLIENT
echo if "%%DEPLOY_TYPE%%"=="server" goto CONFIGURE_SERVER
echo if "%%DEPLOY_TYPE%%"=="both" goto CONFIGURE_BOTH
echo.
echo :CONFIGURE_CLIENT
echo echo.
echo echo Configuring Windows Client...
echo echo.
echo set /p SERVER_URL="Enter backup server URL (e.g., https://backup-server:443): "
echo if "%%SERVER_URL%%"=="" ^(
echo     echo ERROR: Server URL is required.
echo     pause
echo     exit /b 1
echo ^)
echo.
echo echo AGENT_URL=%%SERVER_URL%% >> "%%CONFIG_DIR%%\.env"
echo echo AGENT_LISTEN_HOST=0.0.0.0 >> "%%CONFIG_DIR%%\.env"
echo echo AGENT_PORT=443 >> "%%CONFIG_DIR%%\.env"
echo echo AGENT_PUBLIC_HOST=%%COMPUTERNAME%% >> "%%CONFIG_DIR%%\.env"
echo echo AGENT_PUBLIC_URL=https://%%COMPUTERNAME%%:443 >> "%%CONFIG_DIR%%\.env"
echo echo AGENT_PUBLIC_SCHEME=https >> "%%CONFIG_DIR%%\.env"
echo echo AGENT_CERTFILE=%%CONFIG_DIR%%\keys\server.crt >> "%%CONFIG_DIR%%\.env"
echo echo AGENT_KEYFILE=%%CONFIG_DIR%%\keys\server.key >> "%%CONFIG_DIR%%\.env"
echo echo AGENT_VERIFY_SSL=false >> "%%CONFIG_DIR%%\.env"
echo echo AGENT_CA_BUNDLE=%%CONFIG_DIR%%\keys\ca-bundle.crt >> "%%CONFIG_DIR%%\.env"
echo.
echo echo Installing Windows Service...
echo %%INSTALL_DIR%%\ArcVaultAgentService.exe install
echo net start ArcVaultAgent
echo.
echo echo Client configuration complete!
echo echo Agent available at: https://%%COMPUTERNAME%%:443
echo goto END
echo.
echo :CONFIGURE_SERVER
echo echo.
echo echo Configuring Cross-Platform Server...
echo echo.
echo set /p SERVER_HOST="Enter server hostname (e.g., backup-server.company.com): "
echo if "%%SERVER_HOST%%"=="" set SERVER_HOST=%%COMPUTERNAME%%
echo.
echo set /p CONTROLLER_PORT="Controller port (default: 8501): "
echo if "%%CONTROLLER_PORT%%"=="" set CONTROLLER_PORT=8501
echo.
echo REM Update environment file
echo powershell -Command "^(Get-Content '%%CONFIG_DIR%%\.env'^) -replace 'your-controller-hostname.domain.com', '%%SERVER_HOST%%' ^| Set-Content '%%CONFIG_DIR%%\.env'"
echo powershell -Command "^(Get-Content '%%CONFIG_DIR%%\.env'^) -replace 'CONTROLLER_PORT=8501', 'CONTROLLER_PORT=%%CONTROLLER_PORT%%' ^| Set-Content '%%CONFIG_DIR%%\.env'"
echo.
echo echo Starting Docker server...
echo cd /d "%%DOCKER_DIR%%"
echo docker-compose up -d --build
echo.
echo echo Server configuration complete!
echo echo Web interface available at: http://%%SERVER_HOST%%:%%CONTROLLER_PORT%%
echo goto END
echo.
echo :CONFIGURE_BOTH
echo echo.
echo echo Configuring Complete Setup...
echo echo.
echo set /p SERVER_HOST="Enter server hostname (e.g., backup-server.company.com): "
echo if "%%SERVER_HOST%%"=="" set SERVER_HOST=%%COMPUTERNAME%%
echo.
echo set /p CONTROLLER_PORT="Controller port (default: 8501): "
echo if "%%CONTROLLER_PORT%%"=="" set CONTROLLER_PORT=8501
echo.
echo REM Update environment file
echo powershell -Command "^(Get-Content '%%CONFIG_DIR%%\.env'^) -replace 'your-controller-hostname.domain.com', '%%SERVER_HOST%%' ^| Set-Content '%%CONFIG_DIR%%\.env'"
echo powershell -Command "^(Get-Content '%%CONFIG_DIR%%\.env'^) -replace 'CONTROLLER_PORT=8501', 'CONTROLLER_PORT=%%CONTROLLER_PORT%%' ^| Set-Content '%%CONFIG_DIR%%\.env'"
echo.
echo echo AGENT_URL=https://%%SERVER_HOST%%:443 >> "%%CONFIG_DIR%%\.env"
echo echo AGENT_LISTEN_HOST=0.0.0.0 >> "%%CONFIG_DIR%%\.env"
echo echo AGENT_PORT=443 >> "%%CONFIG_DIR%%\.env"
echo echo AGENT_PUBLIC_HOST=%%COMPUTERNAME%% >> "%%CONFIG_DIR%%\.env"
echo echo AGENT_PUBLIC_URL=https://%%COMPUTERNAME%%:443 >> "%%CONFIG_DIR%%\.env"
echo echo AGENT_PUBLIC_SCHEME=https >> "%%CONFIG_DIR%%\.env"
echo echo AGENT_CERTFILE=%%CONFIG_DIR%%\keys\server.crt >> "%%CONFIG_DIR%%\.env"
echo echo AGENT_KEYFILE=%%CONFIG_DIR%%\keys\server.key >> "%%CONFIG_DIR%%\.env"
echo echo AGENT_VERIFY_SSL=false >> "%%CONFIG_DIR%%\.env"
echo echo AGENT_CA_BUNDLE=%%CONFIG_DIR%%\keys\ca-bundle.crt >> "%%CONFIG_DIR%%\.env"
echo.
echo echo Installing Windows Service...
echo %%INSTALL_DIR%%\ArcVaultAgentService.exe install
echo net start ArcVaultAgent
echo.
echo echo Starting Docker server...
echo cd /d "%%DOCKER_DIR%%"
echo docker-compose up -d --build
echo.
echo echo Complete setup configuration finished!
echo echo - Agent available at: https://%%COMPUTERNAME%%:443
echo echo - Web interface available at: http://%%SERVER_HOST%%:%%CONTROLLER_PORT%%
echo goto END
echo.
echo :END
echo echo.
echo echo ========================================
echo echo ArcVault Setup Complete!
echo echo ========================================
echo echo.
echo echo Deployment: %%DEPLOY_TYPE%%
echo echo Environment: %%ENV_TYPE%%
echo echo.
echo echo Configuration saved to: %%CONFIG_DIR%%\.env
echo echo Certificates: %%CONFIG_DIR%%\keys\
echo echo Logs: %%CONFIG_DIR%%\logs\
echo.
echo echo Management:
echo - Use Windows Services to manage the agent
echo - Use Docker commands to manage the server
echo - Access web interface to manage backup jobs
echo.
echo pause
) > "%UNIFIED_DIR%\setup.bat" 2>nul

REM Create unified manager
echo [6/6] Creating unified manager...

(
echo @echo off
echo setlocal ENABLEDELAYEDEXPANSION
echo.
echo echo ========================================
echo echo ArcVault Unified Manager
echo echo ========================================
echo echo.
echo.
echo set INSTALL_DIR=%%ProgramFiles%%\ArcVault
echo set CONFIG_DIR=%%ProgramData%%\ArcVault
echo set DOCKER_DIR=%%ProgramData%%\ArcVault\docker
echo.
echo :MENU
echo echo ArcVault Unified Manager
echo echo =======================
echo echo.
echo echo 1. Manage Windows Agent
echo echo 2. Manage Docker Server
echo echo 3. View Service Status
echo echo 4. View Logs
echo echo 5. Regenerate Certificates
echo echo 6. Regenerate JWT Keys
echo echo 7. Open Web Interface
echo echo 8. Exit
echo echo.
echo set /p CHOICE="Enter your choice (1-8): "
echo.
echo if "%%CHOICE%%"=="1" goto MANAGE_AGENT
echo if "%%CHOICE%%"=="2" goto MANAGE_SERVER
echo if "%%CHOICE%%"=="3" goto VIEW_STATUS
echo if "%%CHOICE%%"=="4" goto VIEW_LOGS
echo if "%%CHOICE%%"=="5" goto REGEN_CERTS
echo if "%%CHOICE%%"=="6" goto REGEN_JWT
echo if "%%CHOICE%%"=="7" goto OPEN_WEB
echo if "%%CHOICE%%"=="8" goto EXIT
echo echo Invalid choice. Please try again.
echo goto MENU
echo.
echo :MANAGE_AGENT
echo echo Windows Agent Management:
echo echo ========================
echo echo 1. Start Agent Service
echo echo 2. Stop Agent Service
echo echo 3. Restart Agent Service
echo echo 4. Uninstall Agent Service
echo echo 5. Back to main menu
echo echo.
echo set /p AGENT_CHOICE="Enter your choice (1-5): "
echo.
echo if "%%AGENT_CHOICE%%"=="1" net start ArcVaultAgent
echo if "%%AGENT_CHOICE%%"=="2" net stop ArcVaultAgent
echo if "%%AGENT_CHOICE%%"=="3" net stop ArcVaultAgent ^&^& net start ArcVaultAgent
echo if "%%AGENT_CHOICE%%"=="4" net stop ArcVaultAgent ^&^& %%INSTALL_DIR%%\ArcVaultAgentService.exe remove
echo if "%%AGENT_CHOICE%%"=="5" goto MENU
echo pause
echo goto MENU
echo.
echo :MANAGE_SERVER
echo echo Docker Server Management:
echo echo ========================
echo echo 1. Start Server
echo echo 2. Stop Server
echo echo 3. Restart Server
echo echo 4. Update Server
echo echo 5. View Server Logs
echo echo 6. Back to main menu
echo echo.
echo set /p SERVER_CHOICE="Enter your choice (1-6): "
echo.
echo if "%%SERVER_CHOICE%%"=="1" cd /d "%%DOCKER_DIR%%" ^&^& docker-compose up -d
echo if "%%SERVER_CHOICE%%"=="2" cd /d "%%DOCKER_DIR%%" ^&^& docker-compose down
echo if "%%SERVER_CHOICE%%"=="3" cd /d "%%DOCKER_DIR%%" ^&^& docker-compose restart
echo if "%%SERVER_CHOICE%%"=="4" cd /d "%%DOCKER_DIR%%" ^&^& docker-compose up -d --build
echo if "%%SERVER_CHOICE%%"=="5" cd /d "%%DOCKER_DIR%%" ^&^& docker-compose logs -f
echo if "%%SERVER_CHOICE%%"=="6" goto MENU
echo pause
echo goto MENU
echo.
echo :VIEW_STATUS
echo echo ArcVault Service Status:
echo echo =======================
echo echo.
echo echo Windows Agent:
echo sc query ArcVaultAgent
echo echo.
echo echo Docker Server:
echo cd /d "%%DOCKER_DIR%%"
echo docker-compose ps
echo echo.
echo echo Health Checks:
echo curl -s https://localhost:443/health 2^>nul ^|^| echo "Agent not responding"
echo curl -s http://localhost:8501 2^>nul ^|^| echo "Server not responding"
echo pause
echo goto MENU
echo.
echo :VIEW_LOGS
echo echo ArcVault Logs:
echo echo ==============
echo echo.
echo echo Windows Agent Logs:
echo echo ------------------
echo powershell -Command "Get-EventLog -LogName Application -Source 'ArcVault*' -Newest 10 ^| Format-Table TimeGenerated, EntryType, Message -Wrap"
echo echo.
echo echo Docker Server Logs:
echo echo ------------------
echo cd /d "%%DOCKER_DIR%%"
echo docker-compose logs --tail=20
echo echo.
echo echo Backup Logs:
echo echo -----------
echo if exist "%%CONFIG_DIR%%\logs" ^(
echo     dir "%%CONFIG_DIR%%\logs" /b
echo ^) else ^(
echo     echo No backup logs found.
echo ^)
echo pause
echo goto MENU
echo.
echo :REGEN_CERTS
echo echo Regenerating Certificates...
echo %%INSTALL_DIR%%\ArcVaultCertGen.exe "%%CONFIG_DIR%%\keys" "%%COMPUTERNAME%%"
echo echo Certificates regenerated.
echo pause
echo goto MENU
echo.
echo :REGEN_JWT
echo echo Regenerating JWT Keys...
echo %%INSTALL_DIR%%\ArcVaultJWTGen.exe "%%CONFIG_DIR%%\keys"
echo echo JWT keys regenerated.
echo pause
echo goto MENU
echo.
echo :OPEN_WEB
echo echo Opening web interface...
echo start http://localhost:8501
echo goto MENU
echo.
echo :EXIT
echo echo Goodbye!
echo exit /b 0
) > "%UNIFIED_DIR%\manager.bat" 2>nul

REM Create README
(
echo ArcVault Unified Installer
echo ==========================
echo.
echo This installer contains everything needed to deploy ArcVault:
echo - Windows backup agents
echo - Cross-platform backup server
echo - SSL certificates and JWT keys
echo - Complete configuration
echo.
echo CONTENTS:
echo ---------
echo client\
echo   ArcVaultAgent.exe          - Backup agent executable
echo   ArcVaultAgentService.exe   - Windows service version
echo.
echo server\
echo   ArcVaultController.exe     - Web control panel
echo.
echo tools\
echo   ArcVaultCertGen.exe        - Certificate generator
echo   ArcVaultJWTGen.exe         - JWT key generator
echo.
echo docker\
echo   Dockerfile.server          - Server container
echo   docker-compose.yml         - Multi-service orchestration
echo   nginx\nginx.conf           - HTTPS reverse proxy
echo.
echo config\
echo   development.env            - Development configuration
echo   staging.env               - Staging configuration  
echo   production.env            - Production configuration
echo.
echo DEPLOYMENT:
echo ----------
echo 1. Run install.bat as Administrator
echo 2. Run setup.bat to configure your environment
echo 3. Use manager.bat to manage services
echo.
echo FEATURES:
echo ---------
echo - Single installer for everything
echo - Automatic certificate generation
echo - Automatic JWT key generation
echo - Windows client support
echo - Cross-platform server support
echo - Docker deployment
echo - HTTPS communication
echo - Complete configuration
echo.
echo SUPPORT:
echo --------
echo For issues and support, visit: https://github.com/castrokren/arcvault-tool
) > "%UNIFIED_DIR%\README.txt" 2>nul

REM Create main launcher
(
echo @echo off
echo echo ArcVault Unified Deployment
echo echo ===========================
echo echo.
echo echo Choose an option:
echo echo 1. Install ArcVault
echo echo 2. Configure ArcVault  
echo echo 3. Manage Services
echo echo 4. View Documentation
echo echo 5. Exit
echo echo.
echo set /p CHOICE="Enter your choice (1-5): "
echo.
echo if "%%CHOICE%%"=="1" start install.bat
echo if "%%CHOICE%%"=="2" start setup.bat
echo if "%%CHOICE%%"=="3" start manager.bat
echo if "%%CHOICE%%"=="4" start README.txt
echo if "%%CHOICE%%"=="5" exit
echo.
echo Invalid choice. Please try again.
echo pause
echo goto :eof
) > "%UNIFIED_DIR%\ArcVault-Unified.bat" 2>nul

echo.
echo ========================================
echo Unified Installer Complete!
echo ========================================
echo.
echo Package created in: %UNIFIED_DIR%\
echo.
echo Contents:
echo - All client and server executables
echo - Certificate and JWT generators
echo - Docker configuration
echo - Complete setup wizard
echo - Unified management tools
echo - README.txt (Documentation)
echo.
echo To deploy:
echo 1. Copy the entire '%UNIFIED_DIR%' folder to target machine
echo 2. Run ArcVault-Unified.bat
echo 3. Follow the setup wizard
echo.
echo Package size:
set "TOTAL_SIZE="
for /f "tokens=3" %%s in ('dir /s "%UNIFIED_DIR%" ^| findstr /r /c:" [0-9][0-9]* File(s)"') do set "TOTAL_SIZE=%%s"
echo !TOTAL_SIZE! bytes

popd
endlocal

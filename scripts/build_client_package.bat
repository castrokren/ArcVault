@echo off
setlocal ENABLEDELAYEDEXPANSION

REM Build a lightweight client deployment package for ArcVault agents
REM This creates a minimal package for installing agents on client machines

echo ========================================
echo ArcVault Client Package Builder
echo ========================================
echo.

REM Determine repo root
set "SCRIPT_DIR=%~dp0"
pushd "%SCRIPT_DIR%\.."

REM Create client deployment directory
set "CLIENT_DIR=deployment-client"
if exist "%CLIENT_DIR%" (
    echo Removing existing client deployment directory...
    rmdir /s /q "%CLIENT_DIR%"
)
mkdir "%CLIENT_DIR%"

echo Building client package...
echo.

REM Create subdirectories
mkdir "%CLIENT_DIR%\agent"
mkdir "%CLIENT_DIR%\keys"
mkdir "%CLIENT_DIR%\config"

REM Build Agent EXE
echo [1/4] Building Agent executable...
set "PYTHON_EXE=C:\Users\castrk05\AppData\Local\Programs\Python\Python311\python.exe"
if not exist .venv_agent (
    "%PYTHON_EXE%" -m venv .venv_agent
)
call ".venv_agent\Scripts\activate"
python -m pip install --upgrade pip
python -m pip install pyinstaller pywin32
python -m pip install -e "apps\backup-agent"

pyinstaller --noconfirm --clean --onefile ^
  --name ArcVaultAgent ^
  --distpath "%CLIENT_DIR%\agent" ^
  "apps\backup-agent\agent\run_agent.py"

if %ERRORLEVEL% NEQ 0 (
    echo ERROR: Agent build failed
    popd
    exit /b %ERRORLEVEL%
)

REM Build Agent Service EXE
echo [2/4] Building Agent Service executable...
pyinstaller --noconfirm --clean --onefile ^
  --name ArcVaultAgentService ^
  --distpath "%CLIENT_DIR%\agent" ^
  --hidden-import=win32timezone ^
  --hidden-import=win32api ^
  --hidden-import=win32event ^
  --hidden-import=win32service ^
  --hidden-import=win32serviceutil ^
  --hidden-import=servicemanager ^
  "apps\backup-agent\agent\win_service.py"

if %ERRORLEVEL% NEQ 0 (
    echo ERROR: Agent Service build failed
    popd
    exit /b %ERRORLEVEL%
)

REM Copy client-specific configuration
echo [3/4] Copying client configuration...
copy "env.templates\production.env" "%CLIENT_DIR%\config\production.env"
copy "env.templates\staging.env" "%CLIENT_DIR%\config\staging.env"
copy "env.templates\development.env" "%CLIENT_DIR%\config\development.env"

REM Create client installer
echo [4/4] Creating client installer...

REM Create main client installer script
(
echo @echo off
echo setlocal ENABLEDELAYEDEXPANSION
echo.
echo echo ========================================
echo echo ArcVault Agent Installer
echo echo ========================================
echo echo.
echo echo This installer will install the ArcVault backup agent on this computer.
echo echo The agent will connect to your backup server to receive backup jobs.
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
echo REM Set installation directory
echo set INSTALL_DIR=%%ProgramFiles%%\ArcVault
echo set CONFIG_DIR=%%ProgramData%%\ArcVault
echo.
echo echo Installing ArcVault Agent to: %%INSTALL_DIR%%
echo echo Configuration directory: %%CONFIG_DIR%%
echo.
echo REM Create directories
echo mkdir "%%INSTALL_DIR%%" 2^>nul
echo mkdir "%%CONFIG_DIR%%" 2^>nul
echo mkdir "%%CONFIG_DIR%%\keys" 2^>nul
echo mkdir "%%CONFIG_DIR%%\logs" 2^>nul
echo.
echo REM Copy executables
echo echo Copying ArcVault Agent...
echo copy "agent\ArcVaultAgent.exe" "%%INSTALL_DIR%%\"
echo copy "agent\ArcVaultAgentService.exe" "%%INSTALL_DIR%%\"
echo.
echo REM Copy configuration templates
echo copy "config\*.env" "%%CONFIG_DIR%%\"
echo.
echo REM Set permissions
echo echo Setting permissions...
echo icacls "%%INSTALL_DIR%%" /inheritance:r /grant:r "Administrators:F" "SYSTEM:F" /T
echo icacls "%%CONFIG_DIR%%" /inheritance:r /grant:r "Administrators:F" "SYSTEM:F" /T
echo.
echo echo.
echo echo ========================================
echo echo Agent Installation Complete!
echo echo ========================================
echo echo.
echo echo Next steps:
echo echo 1. Run: %%INSTALL_DIR%%\ArcVaultAgentSetup.exe
echo echo 2. Configure connection to backup server
echo echo 3. Start the agent service
echo echo.
echo echo The agent will be available at: http://%%COMPUTERNAME%%:8443
echo echo.
echo pause
) > "%CLIENT_DIR%\install-agent.bat"

REM Create client setup wizard
(
echo @echo off
echo setlocal ENABLEDELAYEDEXPANSION
echo.
echo echo ========================================
echo echo ArcVault Agent Configuration
echo echo ========================================
echo echo.
echo echo This wizard will configure the ArcVault agent to connect to your backup server.
echo echo.
echo.
echo set INSTALL_DIR=%%ProgramFiles%%\ArcVault
echo set CONFIG_DIR=%%ProgramData%%\ArcVault
echo.
echo REM Server connection configuration
echo echo Configure connection to backup server:
echo echo.
echo set /p SERVER_URL="Enter backup server URL (e.g., https://backup-server.company.com:8443): "
echo if "%%SERVER_URL%%"=="" ^(
echo     echo ERROR: Server URL is required.
echo     pause
echo     exit /b 1
echo ^)
echo.
echo echo Select authentication method:
echo echo 1. Simple token ^(development/testing^)
echo echo 2. JWT authentication ^(production^)
echo echo.
echo set /p AUTH_CHOICE="Enter your choice (1-2): "
echo.
echo if "%%AUTH_CHOICE%%"=="1" ^(
echo     set /p AUTH_TOKEN="Enter authentication token: "
echo     if "%%AUTH_TOKEN%%"=="" set AUTH_TOKEN=dev-token-12345
echo     echo AGENT_TOKEN=%%AUTH_TOKEN%% > "%%CONFIG_DIR%%\.env"
echo     echo AGENT_REQUIRE_JWT=false >> "%%CONFIG_DIR%%\.env"
echo ^) else if "%%AUTH_CHOICE%%"=="2" ^(
echo     echo JWT authentication requires the server's public key.
echo     echo Please copy the server's public key to: %%CONFIG_DIR%%\keys\controller_public.pem
echo     echo.
echo     pause
echo     if not exist "%%CONFIG_DIR%%\keys\controller_public.pem" ^(
echo         echo ERROR: JWT public key not found.
echo         echo Please copy the key file and run setup again.
echo         pause
echo         exit /b 1
echo     ^)
echo     echo AGENT_REQUIRE_JWT=true > "%%CONFIG_DIR%%\.env"
echo     echo AGENT_JWT_PUBLIC_KEY_PATH=%%CONFIG_DIR%%\keys\controller_public.pem >> "%%CONFIG_DIR%%\.env"
echo     echo AGENT_JWT_ISSUER=backup-controller >> "%%CONFIG_DIR%%\.env"
echo     echo AGENT_JWT_AUDIENCE=backup-agent >> "%%CONFIG_DIR%%\.env"
echo ^) else ^(
echo     echo Invalid choice. Please try again.
echo     pause
echo     exit /b 1
echo ^)
echo.
echo REM Configure agent settings
echo echo.
echo echo Configure agent settings:
echo echo.
echo set /p AGENT_PORT="Agent port (default: 8443): "
echo if "%%AGENT_PORT%%"=="" set AGENT_PORT=8443
echo.
echo set /p AGENT_HOST="Agent hostname (default: %%COMPUTERNAME%%): "
echo if "%%AGENT_HOST%%"=="" set AGENT_HOST=%%COMPUTERNAME%%
echo.
echo REM Update environment file
echo echo AGENT_LISTEN_HOST=0.0.0.0 >> "%%CONFIG_DIR%%\.env"
echo echo AGENT_PORT=%%AGENT_PORT%% >> "%%CONFIG_DIR%%\.env"
echo echo AGENT_PUBLIC_HOST=%%AGENT_HOST%% >> "%%CONFIG_DIR%%\.env"
echo echo AGENT_PUBLIC_URL=http://%%AGENT_HOST%%:%%AGENT_PORT%% >> "%%CONFIG_DIR%%\.env"
echo echo AGENT_PUBLIC_SCHEME=http >> "%%CONFIG_DIR%%\.env"
echo.
echo REM SSL configuration
echo echo.
echo echo SSL Configuration:
echo echo 1. HTTP ^(development^)
echo echo 2. HTTPS ^(production^)
echo echo.
echo set /p SSL_CHOICE="Enter your choice (1-2): "
echo.
echo if "%%SSL_CHOICE%%"=="2" ^(
echo     echo HTTPS requires SSL certificates.
echo     echo Please copy your certificates to: %%CONFIG_DIR%%\keys\
echo     echo Required files: agent.crt, agent.key, ca-bundle.crt
echo     echo.
echo     pause
echo     echo AGENT_CERTFILE=%%CONFIG_DIR%%\keys\agent.crt >> "%%CONFIG_DIR%%\.env"
echo     echo AGENT_KEYFILE=%%CONFIG_DIR%%\keys\agent.key >> "%%CONFIG_DIR%%\.env"
echo     echo AGENT_VERIFY_SSL=true >> "%%CONFIG_DIR%%\.env"
echo     echo AGENT_CA_BUNDLE=%%CONFIG_DIR%%\keys\ca-bundle.crt >> "%%CONFIG_DIR%%\.env"
echo ^) else ^(
echo     echo AGENT_VERIFY_SSL=false >> "%%CONFIG_DIR%%\.env"
echo ^)
echo.
echo echo.
echo echo ========================================
echo echo Agent Configuration Complete!
echo echo ========================================
echo echo.
echo echo Configuration saved to: %%CONFIG_DIR%%\.env
echo echo.
echo echo Agent will be available at: http://%%AGENT_HOST%%:%%AGENT_PORT%%
echo echo.
echo echo Next steps:
echo echo 1. Install the agent service
echo echo 2. Start the service
echo echo 3. Register this agent with your backup server
echo echo.
echo pause
) > "%CLIENT_DIR%\setup-agent.bat"

REM Create client service manager
(
echo @echo off
echo setlocal ENABLEDELAYEDEXPANSION
echo.
echo echo ========================================
echo echo ArcVault Agent Service Manager
echo echo ========================================
echo echo.
echo.
echo set INSTALL_DIR=%%ProgramFiles%%\ArcVault
echo set CONFIG_DIR=%%ProgramData%%\ArcVault
echo.
echo :MENU
echo echo ArcVault Agent Service Manager
echo echo ===============================
echo echo.
echo echo 1. Install Agent Service
echo echo 2. Start Agent Service
echo echo 3. Stop Agent Service
echo echo 4. Uninstall Agent Service
echo echo 5. View Service Status
echo echo 6. View Agent Logs
echo echo 7. Test Agent Connection
echo echo 8. Exit
echo echo.
echo set /p CHOICE="Enter your choice (1-8): "
echo.
echo if "%%CHOICE%%"=="1" goto INSTALL_SERVICE
echo if "%%CHOICE%%"=="2" goto START_SERVICE
echo if "%%CHOICE%%"=="3" goto STOP_SERVICE
echo if "%%CHOICE%%"=="4" goto UNINSTALL_SERVICE
echo if "%%CHOICE%%"=="5" goto VIEW_STATUS
echo if "%%CHOICE%%"=="6" goto VIEW_LOGS
echo if "%%CHOICE%%"=="7" goto TEST_CONNECTION
echo if "%%CHOICE%%"=="8" goto EXIT
echo echo Invalid choice. Please try again.
echo goto MENU
echo.
echo :INSTALL_SERVICE
echo echo Installing ArcVault Agent Service...
echo "%%INSTALL_DIR%%\ArcVaultAgentService.exe" install
echo if %%ERRORLEVEL%% == 0 ^(
echo     echo Service installed successfully.
echo ^) else ^(
echo     echo Failed to install service.
echo ^)
echo pause
echo goto MENU
echo.
echo :START_SERVICE
echo echo Starting ArcVault Agent Service...
echo net start ArcVaultAgent
echo if %%ERRORLEVEL%% == 0 ^(
echo     echo Service started successfully.
echo ^) else ^(
echo     echo Failed to start service.
echo ^)
echo pause
echo goto MENU
echo.
echo :STOP_SERVICE
echo echo Stopping ArcVault Agent Service...
echo net stop ArcVaultAgent
echo if %%ERRORLEVEL%% == 0 ^(
echo     echo Service stopped successfully.
echo ^) else ^(
echo     echo Failed to stop service.
echo ^)
echo pause
echo goto MENU
echo.
echo :UNINSTALL_SERVICE
echo echo Uninstalling ArcVault Agent Service...
echo net stop ArcVaultAgent 2^>nul
echo "%%INSTALL_DIR%%\ArcVaultAgentService.exe" remove
echo if %%ERRORLEVEL%% == 0 ^(
echo     echo Service uninstalled successfully.
echo ^) else ^(
echo     echo Failed to uninstall service.
echo ^)
echo pause
echo goto MENU
echo.
echo :VIEW_STATUS
echo echo ArcVault Agent Service Status:
echo echo ==============================
echo sc query ArcVaultAgent
echo echo.
echo echo Agent Health Check:
echo curl -s http://localhost:8443/health 2^>nul ^|^| echo "Agent not responding"
echo pause
echo goto MENU
echo.
echo :VIEW_LOGS
echo echo ArcVault Agent Logs:
echo echo ====================
echo echo.
echo echo Service Logs:
echo echo ------------
echo powershell -Command "Get-EventLog -LogName Application -Source 'ArcVault*' -Newest 10 ^| Format-Table TimeGenerated, EntryType, Message -Wrap"
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
echo :TEST_CONNECTION
echo echo Testing Agent Connection:
echo echo =========================
echo echo.
echo echo Testing local agent...
echo curl -s http://localhost:8443/health
echo echo.
echo echo Testing agent info...
echo curl -s http://localhost:8443/info
echo echo.
echo pause
echo goto MENU
echo.
echo :EXIT
echo echo Goodbye!
echo exit /b 0
) > "%CLIENT_DIR%\agent-manager.bat"

REM Create client README
(
echo ArcVault Agent - Client Deployment Package
echo ==========================================
echo.
echo This package contains everything needed to install the ArcVault backup agent
echo on client machines.
echo.
echo CONTENTS:
echo ---------
echo agent\
echo   ArcVaultAgent.exe          - Backup agent executable
echo   ArcVaultAgentService.exe   - Windows service version
echo.
echo config\
echo   development.env            - Development configuration
echo   staging.env               - Staging configuration  
echo   production.env            - Production configuration
echo.
echo keys\                        - Directory for SSL certificates and JWT keys
echo.
echo DEPLOYMENT SCRIPTS:
echo ------------------
echo install-agent.bat           - Main installer ^(run as Administrator^)
echo setup-agent.bat             - Configuration wizard
echo agent-manager.bat           - Service management tool
echo.
echo QUICK START:
echo -----------
echo 1. Run install-agent.bat as Administrator
echo 2. Run setup-agent.bat to configure connection to backup server
echo 3. Use agent-manager.bat to start the service
echo.
echo DETAILED INSTRUCTIONS:
echo ---------------------
echo 1. INSTALLATION:
echo    - Copy this entire folder to the target client machine
echo    - Run install-agent.bat as Administrator
echo    - This installs the agent to Program Files
echo.
echo 2. CONFIGURATION:
echo    - Run setup-agent.bat to configure the agent
echo    - Enter your backup server URL
echo    - Choose authentication method (token or JWT)
echo    - Configure SSL if needed
echo.
echo 3. AUTHENTICATION:
echo    - Simple Token: Enter the token provided by your administrator
echo    - JWT: Copy the server's public key to keys\controller_public.pem
echo.
echo 4. SSL CERTIFICATES ^(if using HTTPS^):
echo    - Copy your SSL certificates to ProgramData\ArcVault\keys\
echo    - Required files: agent.crt, agent.key, ca-bundle.crt
echo.
echo 5. START SERVICE:
echo    - Use agent-manager.bat to manage the service
echo    - Install and start the Agent Service
echo    - The agent will be available at http://computer-name:8443
echo.
echo 6. REGISTER WITH SERVER:
echo    - Add this agent to your backup server's agent list
echo    - Use the agent's URL: http://computer-name:8443
echo.
echo ACCESS:
echo ------
echo - Agent API: http://computer-name:8443
echo - Health Check: http://computer-name:8443/health
echo - Agent Info: http://computer-name:8443/info
echo.
echo TROUBLESHOOTING:
echo ---------------
echo - Check service status using agent-manager.bat
echo - View logs in ProgramData\ArcVault\logs\
echo - Ensure firewall allows port 8443
echo - Verify connection to backup server
echo.
echo SUPPORT:
echo --------
echo For issues and support, visit: https://github.com/castrokren/arcvault-tool
) > "%CLIENT_DIR%\README.txt"

REM Create a simple launcher
(
echo @echo off
echo echo ArcVault Agent - Client Installation
echo echo ====================================
echo echo.
echo echo Choose an option:
echo echo 1. Install Agent
echo echo 2. Configure Agent  
echo echo 3. Manage Agent Service
echo echo 4. View Documentation
echo echo 5. Exit
echo echo.
echo set /p CHOICE="Enter your choice (1-5): "
echo.
echo if "%%CHOICE%%"=="1" start install-agent.bat
echo if "%%CHOICE%%"=="2" start setup-agent.bat
echo if "%%CHOICE%%"=="3" start agent-manager.bat
echo if "%%CHOICE%%"=="4" start README.txt
echo if "%%CHOICE%%"=="5" exit
echo.
echo Invalid choice. Please try again.
echo pause
echo goto :eof
) > "%CLIENT_DIR%\ArcVault-Agent.bat"

echo.
echo ========================================
echo Client Package Complete!
echo ========================================
echo.
echo Package created in: %CLIENT_DIR%\
echo.
echo Contents:
echo - ArcVaultAgent.exe (Agent executable)
echo - ArcVaultAgentService.exe (Windows Service)
echo - install-agent.bat (Main installer)
echo - setup-agent.bat (Configuration wizard)
echo - agent-manager.bat (Service management)
echo - ArcVault-Agent.bat (Main launcher)
echo - Configuration templates
echo - README.txt (Documentation)
echo.
echo To deploy on client machines:
echo 1. Copy the entire '%CLIENT_DIR%' folder to target client
echo 2. Run ArcVault-Agent.bat
echo 3. Follow the setup wizard
echo.
echo Package size: 
for /f %%i in ('dir "%CLIENT_DIR%" /s /-c ^| find "File(s)"') do echo %%i bytes

popd
endlocal

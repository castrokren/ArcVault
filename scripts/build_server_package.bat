@echo off
setlocal ENABLEDELAYEDEXPANSION

REM Build a server deployment package for ArcVault controller
REM This creates a package for installing the controller on the backup server

echo ========================================
echo ArcVault Server Package Builder
echo ========================================
echo.

REM Determine repo root
set "SCRIPT_DIR=%~dp0"
pushd "%SCRIPT_DIR%\.."

REM Create server deployment directory
set "SERVER_DIR=deployment-server"
if exist "%SERVER_DIR%" (
    echo Removing existing server deployment directory...
    rmdir /s /q "%SERVER_DIR%"
)
mkdir "%SERVER_DIR%"

echo Building server package...
echo.

REM Create subdirectories
mkdir "%SERVER_DIR%\controller"
mkdir "%SERVER_DIR%\keys"
mkdir "%SERVER_DIR%\config"
mkdir "%SERVER_DIR%\logs"

REM Build Controller EXE
echo [1/5] Building Controller executable...
set "PYTHON_EXE=C:\Users\castrk05\AppData\Local\Programs\Python\Python311\python.exe"
if not exist .venv_controller (
    "%PYTHON_EXE%" -m venv .venv_controller
)
call ".venv_controller\Scripts\activate"
python -m pip install --upgrade pip
python -m pip install pyinstaller
python -m pip install -e "apps\backup-control"

pyinstaller --noconfirm --clean --onefile ^
  --name ArcVaultController ^
  --distpath "%SERVER_DIR%\controller" ^
  "apps\backup-control\controller\run_controller.py" ^
  --add-data "apps\backup-control\controller;apps\backup-control\controller"

if %ERRORLEVEL% NEQ 0 (
    echo ERROR: Controller build failed
    popd
    exit /b %ERRORLEVEL%
)

REM Build JWT Key Generator
echo [2/5] Building JWT Key Generator...
pyinstaller --noconfirm --clean --onefile ^
  --name ArcVaultKeyGen ^
  --distpath "%SERVER_DIR%" ^
  "scripts\generate_jwt_keys.py"

if %ERRORLEVEL% NEQ 0 (
    echo ERROR: Key generator build failed
    popd
    exit /b %ERRORLEVEL%
)

REM Copy server-specific configuration
echo [3/5] Copying server configuration...
copy "env.templates\production.env" "%SERVER_DIR%\config\production.env"
copy "env.templates\staging.env" "%SERVER_DIR%\config\staging.env"
copy "env.templates\development.env" "%SERVER_DIR%\config\development.env"

REM Create server installer
echo [4/5] Creating server installer...

REM Create main server installer script
(
echo @echo off
echo setlocal ENABLEDELAYEDEXPANSION
echo.
echo echo ========================================
echo echo ArcVault Server Installer
echo echo ========================================
echo echo.
echo echo This installer will install the ArcVault backup server on this computer.
echo echo The server will manage backup agents and provide a web interface.
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
echo echo Installing ArcVault Server to: %%INSTALL_DIR%%
echo echo Configuration directory: %%CONFIG_DIR%%
echo.
echo REM Create directories
echo mkdir "%%INSTALL_DIR%%" 2^>nul
echo mkdir "%%CONFIG_DIR%%" 2^>nul
echo mkdir "%%CONFIG_DIR%%\keys" 2^>nul
echo mkdir "%%CONFIG_DIR%%\logs" 2^>nul
echo.
echo REM Copy executables
echo echo Copying ArcVault Server components...
echo copy "controller\ArcVaultController.exe" "%%INSTALL_DIR%%\"
echo copy "ArcVaultKeyGen.exe" "%%INSTALL_DIR%%\"
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
echo echo Server Installation Complete!
echo echo ========================================
echo echo.
echo echo Next steps:
echo echo 1. Run: %%INSTALL_DIR%%\ArcVaultServerSetup.exe
echo echo 2. Configure your server environment
echo echo 3. Generate JWT keys for agent authentication
echo echo 4. Start the controller
echo echo.
echo echo The controller will be available at: http://%%COMPUTERNAME%%:8501
echo echo.
echo pause
) > "%SERVER_DIR%\install-server.bat"

REM Create server setup wizard
(
echo @echo off
echo setlocal ENABLEDELAYEDEXPANSION
echo.
echo echo ========================================
echo echo ArcVault Server Configuration
echo echo ========================================
echo echo.
echo echo This wizard will configure the ArcVault backup server.
echo echo.
echo.
echo set INSTALL_DIR=%%ProgramFiles%%\ArcVault
echo set CONFIG_DIR=%%ProgramData%%\ArcVault
echo.
echo REM Environment selection
echo :ENV_SELECT
echo echo Select your server environment:
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
echo copy "%%CONFIG_DIR%%\%%ENV_FILE%%" "%%CONFIG_DIR%%\.env"
echo.
echo REM Generate JWT keys for production/staging
echo if "%%ENV_TYPE%%"=="production" ^(
echo     echo Generating JWT keys for production...
echo     %%INSTALL_DIR%%\ArcVaultKeyGen.exe "%%CONFIG_DIR%%\keys"
echo ^) else if "%%ENV_TYPE%%"=="staging" ^(
echo     echo Generating JWT keys for staging...
echo     %%INSTALL_DIR%%\ArcVaultKeyGen.exe "%%CONFIG_DIR%%\keys"
echo ^)
echo.
echo REM Configure server settings
echo echo.
echo echo Configure server settings:
echo echo.
echo set /p SERVER_HOST="Enter server hostname (e.g., backup-server.company.com): "
echo if "%%SERVER_HOST%%"=="" set SERVER_HOST=%%COMPUTERNAME%%
echo.
echo set /p CONTROLLER_PORT="Controller port (default: 8501): "
echo if "%%CONTROLLER_PORT%%"=="" set CONTROLLER_PORT=8501
echo.
echo REM Update environment file with server settings
echo powershell -Command "^(Get-Content '%%CONFIG_DIR%%\.env'^) -replace 'your-controller-hostname.domain.com', '%%SERVER_HOST%%' ^| Set-Content '%%CONFIG_DIR%%\.env'"
echo powershell -Command "^(Get-Content '%%CONFIG_DIR%%\.env'^) -replace 'CONTROLLER_PORT=8501', 'CONTROLLER_PORT=%%CONTROLLER_PORT%%' ^| Set-Content '%%CONFIG_DIR%%\.env'"
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
echo     echo Required files: server.crt, server.key, ca-bundle.crt
echo     echo.
echo     pause
echo     echo CONTROLLER_CERTFILE=%%CONFIG_DIR%%\keys\server.crt >> "%%CONFIG_DIR%%\.env"
echo     echo CONTROLLER_KEYFILE=%%CONFIG_DIR%%\keys\server.key >> "%%CONFIG_DIR%%\.env"
echo     echo CONTROLLER_VERIFY_SSL=true >> "%%CONFIG_DIR%%\.env"
echo     echo CONTROLLER_CA_BUNDLE=%%CONFIG_DIR%%\keys\ca-bundle.crt >> "%%CONFIG_DIR%%\.env"
echo ^) else ^(
echo     echo CONTROLLER_VERIFY_SSL=false >> "%%CONFIG_DIR%%\.env"
echo ^)
echo.
echo REM Agent management
echo echo.
echo echo Agent Management:
echo echo.
echo echo The server will manage backup agents. You can add agents through the web interface.
echo echo Default agent URL: http://localhost:8443
echo echo.
echo set /p DEFAULT_AGENT="Enter default agent URL (or press Enter for localhost): "
echo if "%%DEFAULT_AGENT%%"=="" set DEFAULT_AGENT=http://localhost:8443
echo echo AGENT_URL=%%DEFAULT_AGENT%% >> "%%CONFIG_DIR%%\.env"
echo.
echo echo.
echo echo ========================================
echo echo Server Configuration Complete!
echo echo ========================================
echo echo.
echo echo Environment: %%ENV_TYPE%%
echo echo Hostname: %%SERVER_HOST%%
echo echo Controller Port: %%CONTROLLER_PORT%%
echo echo Default Agent: %%DEFAULT_AGENT%%
echo echo.
echo echo Configuration saved to: %%CONFIG_DIR%%\.env
echo echo.
echo echo Next steps:
echo echo 1. Start the controller
echo echo 2. Access the web interface at: http://%%SERVER_HOST%%:%%CONTROLLER_PORT%%
echo echo 3. Add backup agents through the web interface
echo echo 4. Configure backup jobs
echo echo.
echo pause
) > "%SERVER_DIR%\setup-server.bat"

REM Create server manager
(
echo @echo off
echo setlocal ENABLEDELAYEDEXPANSION
echo.
echo echo ========================================
echo echo ArcVault Server Manager
echo echo ========================================
echo echo.
echo.
echo set INSTALL_DIR=%%ProgramFiles%%\ArcVault
echo set CONFIG_DIR=%%ProgramData%%\ArcVault
echo.
echo :MENU
echo echo ArcVault Server Manager
echo echo =======================
echo echo.
echo echo 1. Start Controller ^(Interactive^)
echo echo 2. Start Controller ^(Background^)
echo echo 3. Stop Controller
echo echo 4. View Controller Status
echo echo 5. View Server Logs
echo echo 6. Manage JWT Keys
echo echo 7. Test Agent Connections
echo echo 8. Open Web Interface
echo echo 9. Exit
echo echo.
echo set /p CHOICE="Enter your choice (1-9): "
echo.
echo if "%%CHOICE%%"=="1" goto START_CONTROLLER
echo if "%%CHOICE%%"=="2" goto START_CONTROLLER_BG
echo if "%%CHOICE%%"=="3" goto STOP_CONTROLLER
echo if "%%CHOICE%%"=="4" goto VIEW_STATUS
echo if "%%CHOICE%%"=="5" goto VIEW_LOGS
echo if "%%CHOICE%%"=="6" goto MANAGE_KEYS
echo if "%%CHOICE%%"=="7" goto TEST_AGENTS
echo if "%%CHOICE%%"=="8" goto OPEN_WEB
echo if "%%CHOICE%%"=="9" goto EXIT
echo echo Invalid choice. Please try again.
echo goto MENU
echo.
echo :START_CONTROLLER
echo echo Starting ArcVault Controller...
echo cd /d "%%INSTALL_DIR%%"
echo %%INSTALL_DIR%%\ArcVaultController.exe
echo goto MENU
echo.
echo :START_CONTROLLER_BG
echo echo Starting ArcVault Controller in background...
echo start "" "%%INSTALL_DIR%%\ArcVaultController.exe"
echo echo Controller started in background.
echo pause
echo goto MENU
echo.
echo :STOP_CONTROLLER
echo echo Stopping ArcVault Controller...
echo taskkill /f /im ArcVaultController.exe 2^>nul
echo if %%ERRORLEVEL%% == 0 ^(
echo     echo Controller stopped successfully.
echo ^) else ^(
echo     echo Controller was not running.
echo ^)
echo pause
echo goto MENU
echo.
echo :VIEW_STATUS
echo echo ArcVault Server Status:
echo echo ======================
echo echo.
echo echo Controller Process:
echo tasklist /fi "imagename eq ArcVaultController.exe" 2^>nul ^| find "ArcVaultController.exe" ^>nul
echo if %%ERRORLEVEL%% == 0 ^(
echo     echo ✅ Controller is running
echo ^) else ^(
echo     echo ❌ Controller is not running
echo ^)
echo echo.
echo echo Web Interface:
echo curl -s http://localhost:8501 2^>nul ^|^| echo "Web interface not responding"
echo pause
echo goto MENU
echo.
echo :VIEW_LOGS
echo echo ArcVault Server Logs:
echo echo ====================
echo echo.
echo echo Controller Logs:
echo echo ---------------
echo if exist "%%CONFIG_DIR%%\logs" ^(
echo     dir "%%CONFIG_DIR%%\logs" /b
echo ^) else ^(
echo     echo No controller logs found.
echo ^)
echo echo.
echo echo System Logs:
echo echo -----------
echo powershell -Command "Get-EventLog -LogName Application -Source 'ArcVault*' -Newest 10 ^| Format-Table TimeGenerated, EntryType, Message -Wrap"
echo pause
echo goto MENU
echo.
echo :MANAGE_KEYS
echo echo JWT Key Management:
echo echo ==================
echo echo.
echo echo 1. Generate new JWT keys
echo echo 2. View existing keys
echo echo 3. Copy public key for agents
echo echo 4. Back to main menu
echo echo.
echo set /p KEY_CHOICE="Enter your choice (1-4): "
echo.
echo if "%%KEY_CHOICE%%"=="1" ^(
echo     echo Generating new JWT keys...
echo     %%INSTALL_DIR%%\ArcVaultKeyGen.exe "%%CONFIG_DIR%%\keys"
echo ^) else if "%%KEY_CHOICE%%"=="2" ^(
echo     echo Existing JWT keys:
echo     dir "%%CONFIG_DIR%%\keys\*.pem" /b 2^>nul ^|^| echo "No JWT keys found"
echo ^) else if "%%KEY_CHOICE%%"=="3" ^(
echo     echo Copying public key for agents...
echo     if exist "%%CONFIG_DIR%%\keys\controller_public.pem" ^(
echo         copy "%%CONFIG_DIR%%\keys\controller_public.pem" "%%CONFIG_DIR%%\keys\controller_public_for_agents.pem"
echo         echo Public key copied to: %%CONFIG_DIR%%\keys\controller_public_for_agents.pem
echo         echo Distribute this file to all agent machines.
echo     ^) else ^(
echo         echo No public key found. Generate keys first.
echo     ^)
echo ^) else if "%%KEY_CHOICE%%"=="4" ^(
echo     goto MENU
echo ^) else ^(
echo     echo Invalid choice.
echo ^)
echo pause
echo goto MENU
echo.
echo :TEST_AGENTS
echo echo Testing Agent Connections:
echo echo ==========================
echo echo.
echo echo Testing default agent...
echo curl -s http://localhost:8443/health 2^>nul ^|^| echo "Default agent not responding"
echo echo.
echo echo Testing agent info...
echo curl -s http://localhost:8443/info 2^>nul ^|^| echo "Default agent info not available"
echo echo.
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
) > "%SERVER_DIR%\server-manager.bat"

REM Create server README
(
echo ArcVault Server - Backup Controller Deployment Package
echo ======================================================
echo.
echo This package contains everything needed to install the ArcVault backup server
echo on your backup management machine.
echo.
echo CONTENTS:
echo ---------
echo controller\
echo   ArcVaultController.exe     - Web control panel
echo.
echo ArcVaultKeyGen.exe           - JWT key generator
echo.
echo config\
echo   development.env            - Development configuration
echo   staging.env               - Staging configuration  
echo   production.env            - Production configuration
echo.
echo keys\                        - Directory for SSL certificates and JWT keys
echo logs\                        - Directory for server logs
echo.
echo DEPLOYMENT SCRIPTS:
echo ------------------
echo install-server.bat          - Main installer ^(run as Administrator^)
echo setup-server.bat            - Configuration wizard
echo server-manager.bat          - Server management tool
echo.
echo QUICK START:
echo -----------
echo 1. Run install-server.bat as Administrator
echo 2. Run setup-server.bat to configure your server
echo 3. Use server-manager.bat to start the controller
echo 4. Access the web interface to manage agents
echo.
echo DETAILED INSTRUCTIONS:
echo ---------------------
echo 1. INSTALLATION:
echo    - Copy this entire folder to your backup server
echo    - Run install-server.bat as Administrator
echo    - This installs the server to Program Files
echo.
echo 2. CONFIGURATION:
echo    - Run setup-server.bat to configure the server
echo    - Choose environment (development/staging/production)
echo    - Enter server hostname and port
echo    - JWT keys are generated automatically for production/staging
echo.
echo 3. SSL CERTIFICATES ^(if using HTTPS^):
echo    - Copy your SSL certificates to ProgramData\ArcVault\keys\
echo    - Required files: server.crt, server.key, ca-bundle.crt
echo.
echo 4. START CONTROLLER:
echo    - Use server-manager.bat to start the controller
echo    - The web interface will be available at http://server:8501
echo.
echo 5. MANAGE AGENTS:
echo    - Access the web interface
echo    - Add backup agents through the interface
echo    - Configure backup jobs
echo    - Monitor backup progress
echo.
echo ACCESS:
echo ------
echo - Web Interface: http://server:8501
echo - Controller API: http://server:8501/api
echo.
echo AGENT MANAGEMENT:
echo ----------------
echo 1. Install agents on client machines using the client package
echo 2. Add agents to the server through the web interface
echo 3. Configure backup jobs for each agent
echo 4. Monitor backup progress and logs
echo.
echo JWT KEY MANAGEMENT:
echo ------------------
echo 1. Generate JWT keys using the key generator
echo 2. Distribute the public key to all agent machines
echo 3. Keep the private key secure on the server
echo 4. Rotate keys periodically for security
echo.
echo TROUBLESHOOTING:
echo ---------------
echo - Check controller status using server-manager.bat
echo - View logs in ProgramData\ArcVault\logs\
echo - Ensure firewall allows port 8501
echo - Verify agent connections through the web interface
echo.
echo SUPPORT:
echo --------
echo For issues and support, visit: https://github.com/castrokren/arcvault-tool
) > "%SERVER_DIR%\README.txt"

REM Create a simple launcher
(
echo @echo off
echo echo ArcVault Server - Backup Controller
echo echo ==================================
echo echo.
echo echo Choose an option:
echo echo 1. Install Server
echo echo 2. Configure Server  
echo echo 3. Manage Server
echo echo 4. View Documentation
echo echo 5. Exit
echo echo.
echo set /p CHOICE="Enter your choice (1-5): "
echo.
echo if "%%CHOICE%%"=="1" start install-server.bat
echo if "%%CHOICE%%"=="2" start setup-server.bat
echo if "%%CHOICE%%"=="3" start server-manager.bat
echo if "%%CHOICE%%"=="4" start README.txt
echo if "%%CHOICE%%"=="5" exit
echo.
echo Invalid choice. Please try again.
echo pause
echo goto :eof
) > "%SERVER_DIR%\ArcVault-Server.bat"

echo.
echo ========================================
echo Server Package Complete!
echo ========================================
echo.
echo Package created in: %SERVER_DIR%\
echo.
echo Contents:
echo - ArcVaultController.exe (Web control panel)
echo - ArcVaultKeyGen.exe (JWT key generator)
echo - install-server.bat (Main installer)
echo - setup-server.bat (Configuration wizard)
echo - server-manager.bat (Server management)
echo - ArcVault-Server.bat (Main launcher)
echo - Configuration templates
echo - README.txt (Documentation)
echo.
echo To deploy on backup server:
echo 1. Copy the entire '%SERVER_DIR%' folder to target server
echo 2. Run ArcVault-Server.bat
echo 3. Follow the setup wizard
echo.
echo Package size: 
for /f %%i in ('dir "%SERVER_DIR%" /s /-c ^| find "File(s)"') do echo %%i bytes

popd
endlocal

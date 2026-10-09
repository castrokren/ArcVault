@echo off
setlocal ENABLEDELAYEDEXPANSION

REM Build a complete self-contained deployment package for ArcVault
REM This creates a single folder with everything needed for production deployment

echo ========================================
echo ArcVault Production Deployment Builder
echo ========================================
echo.

REM Determine repo root
set SCRIPT_DIR=%~dp0
pushd "%SCRIPT_DIR%\.."

REM Create deployment directory
set DEPLOY_DIR=deployment
if exist "%DEPLOY_DIR%" (
    echo Removing existing deployment directory...
    rmdir /s /q "%DEPLOY_DIR%"
)
mkdir "%DEPLOY_DIR%"

echo Building deployment package...
echo.

REM Create subdirectories
mkdir "%DEPLOY_DIR%\agent"
mkdir "%DEPLOY_DIR%\controller"
mkdir "%DEPLOY_DIR%\keys"
mkdir "%DEPLOY_DIR%\logs"
mkdir "%DEPLOY_DIR%\config"

REM Build Agent EXE
echo [1/6] Building Agent executable...
if not exist .venv_agent (
    python -m venv .venv_agent
)
call .venv_agent\Scripts\activate
python -m pip install --upgrade pip
python -m pip install pyinstaller pywin32
python -m pip install -e "apps\backup-agent"

pyinstaller --noconfirm --clean --onefile ^
  --name ArcVaultAgent ^
  --distpath "%DEPLOY_DIR%\agent" ^
  apps\backup-agent\agent\run_agent.py

if %ERRORLEVEL% NEQ 0 (
    echo ERROR: Agent build failed
    popd
    exit /b %ERRORLEVEL%
)

REM Build Agent Service EXE
echo [2/6] Building Agent Service executable...
pyinstaller --noconfirm --clean --onefile ^
  --name ArcVaultAgentService ^
  --distpath "%DEPLOY_DIR%\agent" ^
  apps\backup-agent\agent\win_service.py

if %ERRORLEVEL% NEQ 0 (
    echo ERROR: Agent Service build failed
    popd
    exit /b %ERRORLEVEL%
)

REM Build Controller EXE
echo [3/6] Building Controller executable...
if not exist .venv_controller (
    python -m venv .venv_controller
)
call .venv_controller\Scripts\activate
python -m pip install --upgrade pip
python -m pip install pyinstaller
python -m pip install -e "apps\backup-control"

pyinstaller --noconfirm --clean --onefile ^
  --name ArcVaultController ^
  --distpath "%DEPLOY_DIR%\controller" ^
  apps\backup-control\controller\run_controller.py ^
  --add-data "apps\backup-control\controller;apps\backup-control\controller"

if %ERRORLEVEL% NEQ 0 (
    echo ERROR: Controller build failed
    popd
    exit /b %ERRORLEVEL%
)

REM Copy configuration files
echo [4/6] Copying configuration files...
copy "env.templates\production.env" "%DEPLOY_DIR%\config\production.env"
copy "env.templates\staging.env" "%DEPLOY_DIR%\config\staging.env"
copy "env.templates\development.env" "%DEPLOY_DIR%\config\development.env"

REM Create deployment scripts
echo [5/6] Creating deployment scripts...

REM Create main installer script
(
echo @echo off
echo setlocal ENABLEDELAYEDEXPANSION
echo.
echo echo ========================================
echo echo ArcVault Production Installer
echo echo ========================================
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
echo echo Installing ArcVault to: %%INSTALL_DIR%%
echo echo Configuration directory: %%CONFIG_DIR%%
echo.
echo REM Create directories
echo mkdir "%%INSTALL_DIR%%" 2^>nul
echo mkdir "%%CONFIG_DIR%%" 2^>nul
echo mkdir "%%CONFIG_DIR%%\keys" 2^>nul
echo mkdir "%%CONFIG_DIR%%\logs" 2^>nul
echo.
echo REM Copy executables
echo echo Copying ArcVault components...
echo copy "agent\ArcVaultAgent.exe" "%%INSTALL_DIR%%\"
echo copy "agent\ArcVaultAgentService.exe" "%%INSTALL_DIR%%\"
echo copy "controller\ArcVaultController.exe" "%%INSTALL_DIR%%\"
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
echo echo Installation Complete!
echo echo ========================================
echo echo.
echo echo Next steps:
echo echo 1. Run: %%INSTALL_DIR%%\ArcVaultSetup.exe
echo echo 2. Configure your environment
echo echo 3. Start the services
echo echo.
echo pause
) > "%DEPLOY_DIR%\install.bat"

REM Create setup wizard
(
echo @echo off
echo setlocal ENABLEDELAYEDEXPANSION
echo.
echo echo ========================================
echo echo ArcVault Configuration Wizard
echo echo ========================================
echo echo.
echo.
echo set INSTALL_DIR=%%ProgramFiles%%\ArcVault
echo set CONFIG_DIR=%%ProgramData%%\ArcVault
echo.
echo echo This wizard will help you configure ArcVault for your environment.
echo echo.
echo.
echo REM Environment selection
echo :ENV_SELECT
echo echo Select your deployment environment:
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
echo     %%INSTALL_DIR%%\ArcVaultKeyGen.exe
echo ^) else if "%%ENV_TYPE%%"=="staging" ^(
echo     echo Generating JWT keys for staging...
echo     %%INSTALL_DIR%%\ArcVaultKeyGen.exe
echo ^)
echo.
echo REM Configure hostname
echo echo.
echo echo Configure your hostname:
echo set /p HOSTNAME="Enter your server hostname (e.g., backup-server.company.com): "
echo if "%%HOSTNAME%%"=="" set HOSTNAME=%%COMPUTERNAME%%
echo.
echo REM Update environment file with hostname
echo powershell -Command "^(Get-Content '%%CONFIG_DIR%%\.env'^) -replace 'your-agent-hostname.domain.com', '%%HOSTNAME%%' ^| Set-Content '%%CONFIG_DIR%%\.env'"
echo.
echo echo.
echo echo ========================================
echo echo Configuration Complete!
echo echo ========================================
echo echo.
echo echo Environment: %%ENV_TYPE%%
echo echo Hostname: %%HOSTNAME%%
echo echo.
echo echo Next steps:
echo echo 1. Review configuration: %%CONFIG_DIR%%\.env
echo echo 2. Add SSL certificates to: %%CONFIG_DIR%%\keys\
echo echo 3. Start services using the Service Manager
echo echo.
echo pause
) > "%DEPLOY_DIR%\setup.bat"

REM Create key generator executable
echo [6/6] Building Key Generator executable...
pyinstaller --noconfirm --clean --onefile ^
  --name ArcVaultKeyGen ^
  --distpath "%DEPLOY_DIR%" ^
  scripts\generate_jwt_keys.py

REM Create service manager
(
echo @echo off
echo setlocal ENABLEDELAYEDEXPANSION
echo.
echo echo ========================================
echo echo ArcVault Service Manager
echo echo ========================================
echo echo.
echo.
echo set INSTALL_DIR=%%ProgramFiles%%\ArcVault
echo set CONFIG_DIR=%%ProgramData%%\ArcVault
echo.
echo :MENU
echo echo ArcVault Service Manager
echo echo ========================
echo echo.
echo echo 1. Install Agent Service
echo echo 2. Start Agent Service
echo echo 3. Stop Agent Service
echo echo 4. Uninstall Agent Service
echo echo 5. Start Controller ^(Interactive^)
echo echo 6. Start Controller ^(Background^)
echo echo 7. View Service Status
echo echo 8. View Logs
echo echo 9. Exit
echo echo.
echo set /p CHOICE="Enter your choice (1-9): "
echo.
echo if "%%CHOICE%%"=="1" goto INSTALL_SERVICE
echo if "%%CHOICE%%"=="2" goto START_SERVICE
echo if "%%CHOICE%%"=="3" goto STOP_SERVICE
echo if "%%CHOICE%%"=="4" goto UNINSTALL_SERVICE
echo if "%%CHOICE%%"=="5" goto START_CONTROLLER
echo if "%%CHOICE%%"=="6" goto START_CONTROLLER_BG
echo if "%%CHOICE%%"=="7" goto VIEW_STATUS
echo if "%%CHOICE%%"=="8" goto VIEW_LOGS
echo if "%%CHOICE%%"=="9" goto EXIT
echo echo Invalid choice. Please try again.
echo goto MENU
echo.
echo :INSTALL_SERVICE
echo echo Installing ArcVault Agent Service...
echo %%INSTALL_DIR%%\ArcVaultAgentService.exe install
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
echo %%INSTALL_DIR%%\ArcVaultAgentService.exe remove
echo if %%ERRORLEVEL%% == 0 ^(
echo     echo Service uninstalled successfully.
echo ^) else ^(
echo     echo Failed to uninstall service.
echo ^)
echo pause
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
echo :VIEW_STATUS
echo echo ArcVault Service Status:
echo echo ========================
echo sc query ArcVaultAgent
echo echo.
echo echo Agent Health Check:
echo curl -s http://localhost:8443/health 2^>nul ^|^| echo "Agent not responding"
echo pause
echo goto MENU
echo.
echo :VIEW_LOGS
echo echo ArcVault Logs:
echo echo ==============
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
echo :EXIT
echo echo Goodbye!
echo exit /b 0
) > "%DEPLOY_DIR%\service-manager.bat"

REM Create README for deployment
(
echo ArcVault Production Deployment Package
echo ======================================
echo.
echo This package contains everything needed to deploy ArcVault in production.
echo.
echo CONTENTS:
echo ---------
echo agent\
echo   ArcVaultAgent.exe          - Backup agent executable
echo   ArcVaultAgentService.exe   - Windows service version
echo.
echo controller\
echo   ArcVaultController.exe     - Web control panel
echo.
echo config\
echo   development.env            - Development configuration
echo   staging.env               - Staging configuration  
echo   production.env            - Production configuration
echo.
echo keys\                        - Directory for SSL certificates and JWT keys
echo logs\                        - Directory for backup logs
echo.
echo DEPLOYMENT SCRIPTS:
echo ------------------
echo install.bat                 - Main installer ^(run as Administrator^)
echo setup.bat                   - Configuration wizard
echo service-manager.bat         - Service management tool
echo ArcVaultKeyGen.exe          - JWT key generator
echo.
echo QUICK START:
echo -----------
echo 1. Run install.bat as Administrator
echo 2. Run setup.bat to configure your environment
echo 3. Use service-manager.bat to start services
echo.
echo DETAILED INSTRUCTIONS:
echo ---------------------
echo 1. INSTALLATION:
echo    - Copy this entire folder to the target server
echo    - Run install.bat as Administrator
echo    - This installs ArcVault to Program Files
echo.
echo 2. CONFIGURATION:
echo    - Run setup.bat to configure your environment
echo    - Choose development, staging, or production
echo    - Enter your server hostname
echo    - JWT keys are generated automatically for production/staging
echo.
echo 3. SSL CERTIFICATES ^(Production only^):
echo    - Copy your SSL certificates to ProgramData\ArcVault\keys\
echo    - Required files: agent.crt, agent.key, ca-bundle.crt
echo.
echo 4. START SERVICES:
echo    - Use service-manager.bat to manage services
echo    - Install and start the Agent Service
echo    - Start the Controller ^(interactive or background^)
echo.
echo 5. ACCESS:
echo    - Agent API: https://your-server:8443
echo    - Controller UI: http://your-server:8501
echo.
echo TROUBLESHOOTING:
echo ---------------
echo - Check service status using service-manager.bat
echo - View logs in ProgramData\ArcVault\logs\
echo - Ensure firewall allows ports 8443 and 8501
echo - Verify SSL certificates are valid and accessible
echo.
echo SUPPORT:
echo --------
echo For issues and support, visit: https://github.com/castrokren/arcvault-tool
) > "%DEPLOY_DIR%\README.txt"

REM Create a simple launcher
(
echo @echo off
echo echo ArcVault Production Deployment
echo echo ==============================
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
echo if "%%CHOICE%%"=="3" start service-manager.bat
echo if "%%CHOICE%%"=="4" start README.txt
echo if "%%CHOICE%%"=="5" exit
echo.
echo Invalid choice. Please try again.
echo pause
echo goto :eof
) > "%DEPLOY_DIR%\ArcVault-Deploy.bat"

echo.
echo ========================================
echo Deployment Package Complete!
echo ========================================
echo.
echo Package created in: %DEPLOY_DIR%\
echo.
echo Contents:
echo - ArcVaultAgent.exe (Agent executable)
echo - ArcVaultAgentService.exe (Windows Service)
echo - ArcVaultController.exe (Web interface)
echo - ArcVaultKeyGen.exe (Key generator)
echo - install.bat (Main installer)
echo - setup.bat (Configuration wizard)
echo - service-manager.bat (Service management)
echo - ArcVault-Deploy.bat (Main launcher)
echo - Configuration templates
echo - README.txt (Documentation)
echo.
echo To deploy:
echo 1. Copy the entire '%DEPLOY_DIR%' folder to target server
echo 2. Run ArcVault-Deploy.bat
echo 3. Follow the setup wizard
echo.
echo Package size: 
for /f %%i in ('dir "%DEPLOY_DIR%" /s /-c ^| find "File(s)"') do echo %%i bytes

popd
endlocal

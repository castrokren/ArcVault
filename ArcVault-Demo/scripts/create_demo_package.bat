@echo off
setlocal ENABLEDELAYEDEXPANSION

REM Create a demo package for sharing with friends
REM This includes everything needed to run and test ArcVault

echo ========================================
echo ArcVault Demo Package Creator
echo ========================================
echo.

REM Determine repo root
set SCRIPT_DIR=%~dp0
pushd "%SCRIPT_DIR%\.."

REM Create demo package directory
set DEMO_DIR=ArcVault-Demo
if exist "%DEMO_DIR%" (
    echo Removing existing demo package...
    rmdir /s /q "%DEMO_DIR%"
)
mkdir "%DEMO_DIR%"

echo Creating demo package for your friend...
echo.

REM Create subdirectories
mkdir "%DEMO_DIR%\source"
mkdir "%DEMO_DIR%\deployment"
mkdir "%DEMO_DIR%\docs"
mkdir "%DEMO_DIR%\scripts"

REM Copy source code
echo [1/6] Copying source code...
xcopy "apps" "%DEMO_DIR%\source\apps\" /E /I /Y
copy "version_info.txt" "%DEMO_DIR%\source\"
copy "update_checker.py" "%DEMO_DIR%\source\"
copy "update_gui.py" "%DEMO_DIR%\source\"

REM Copy all scripts
echo [2/6] Copying scripts...
xcopy "scripts" "%DEMO_DIR%\scripts\" /E /I /Y
xcopy "env.templates" "%DEMO_DIR%\scripts\env.templates\" /E /I /Y

REM Copy Docker files
echo [3/6] Copying Docker configuration...
copy "Dockerfile.server" "%DEMO_DIR%\"
copy "docker-compose.yml" "%DEMO_DIR%\"
if exist "nginx" (
    xcopy "nginx" "%DEMO_DIR%\nginx\" /E /I /Y
)

REM Copy documentation
echo [4/6] Copying documentation...
copy "README.md" "%DEMO_DIR%\docs\" 2>nul
copy "DEPLOYMENT_*.md" "%DEMO_DIR%\docs\" 2>nul
copy "apps\backup-agent\README.md" "%DEMO_DIR%\docs\backup-agent-README.md" 2>nul
copy "apps\backup-control\README.md" "%DEMO_DIR%\docs\backup-control-README.md" 2>nul

REM Create demo setup script
echo [5/6] Creating demo setup script...

(
echo @echo off
echo setlocal ENABLEDELAYEDEXPANSION
echo.
echo echo ========================================
echo echo ArcVault Demo Setup
echo echo ========================================
echo echo.
echo echo This will set up ArcVault for testing and demonstration.
echo echo.
echo echo What this demo includes:
echo echo - Backup Agent ^(FastAPI server^)
echo echo - Backup Controller ^(Streamlit web interface^)
echo echo - Robocopy integration for Windows
echo echo - JWT authentication
echo echo - Self-signed certificates
echo echo - Docker deployment option
echo echo.
echo echo.
echo REM Check for Python
echo python --version ^>nul 2^>^&1
echo if %%errorLevel%% NEQ 0 ^(
echo     echo ERROR: Python is not installed or not in PATH.
echo     echo Please install Python 3.8+ and try again.
echo     echo Download from: https://www.python.org/downloads/
echo     pause
echo     exit /b 1
echo ^)
echo.
echo echo Python found. Checking version...
echo python --version
echo.
echo REM Create virtual environment
echo echo Creating virtual environment...
echo python -m venv .venv
echo if %%errorLevel%% NEQ 0 ^(
echo     echo ERROR: Failed to create virtual environment.
echo     pause
echo     exit /b 1
echo ^)
echo.
echo REM Activate virtual environment
echo echo Activating virtual environment...
echo call .venv\Scripts\activate
echo.
echo REM Install dependencies
echo echo Installing dependencies...
echo python -m pip install --upgrade pip
echo python -m pip install -e "source\apps\backup-agent"
echo python -m pip install -e "source\apps\backup-control"
echo.
echo REM Generate certificates and keys
echo echo Generating demo certificates and keys...
echo mkdir keys 2^>nul
echo python scripts\generate_self_signed_cert.py keys localhost
echo python scripts\generate_jwt_keys.py keys
echo.
echo REM Create demo environment file
echo echo Creating demo environment configuration...
echo copy "scripts\env.templates\development.env" ".env"
echo.
echo REM Update environment for demo
echo echo Updating environment for demo...
echo powershell -Command "^(Get-Content '.env'^) -replace 'your-controller-hostname.domain.com', 'localhost' ^| Set-Content '.env'"
echo powershell -Command "^(Get-Content '.env'^) -replace 'AGENT_CERTFILE=keys/server.crt', 'AGENT_CERTFILE=keys/server.crt' ^| Set-Content '.env'"
echo powershell -Command "^(Get-Content '.env'^) -replace 'AGENT_KEYFILE=keys/server.key', 'AGENT_KEYFILE=keys/server.key' ^| Set-Content '.env'"
echo powershell -Command "^(Get-Content '.env'^) -replace 'AGENT_CA_BUNDLE=keys/ca-bundle.crt', 'AGENT_CA_BUNDLE=keys/ca-bundle.crt' ^| Set-Content '.env'"
echo.
echo echo.
echo echo ========================================
echo echo Demo Setup Complete!
echo echo ========================================
echo echo.
echo echo Next steps:
echo echo 1. Start the backup agent: python source\apps\backup-agent\agent\run_agent.py
echo echo 2. Start the controller: python source\apps\backup-control\controller\run_controller.py
echo echo 3. Open web interface: http://localhost:8501
echo echo 4. Add agent: http://localhost:8443
echo echo.
echo echo Or use the quick start script: run_demo.bat
echo echo.
echo pause
) > "%DEMO_DIR%\setup_demo.bat"

REM Create quick start script
echo [6/6] Creating quick start script...

(
echo @echo off
echo setlocal ENABLEDELAYEDEXPANSION
echo.
echo echo ========================================
echo echo ArcVault Demo - Quick Start
echo echo ========================================
echo echo.
echo echo Starting ArcVault demo...
echo echo.
echo.
echo REM Check if setup was run
echo if not exist ".venv" ^(
echo     echo ERROR: Demo setup not run yet.
echo     echo Please run setup_demo.bat first.
echo     pause
echo     exit /b 1
echo ^)
echo.
echo REM Activate virtual environment
echo call .venv\Scripts\activate
echo.
echo REM Start services
echo echo Starting backup agent...
echo start "ArcVault Agent" cmd /k "python source\apps\backup-agent\agent\run_agent.py"
echo.
echo echo Waiting for agent to start...
echo timeout /t 3 /nobreak ^>nul
echo.
echo echo Starting backup controller...
echo start "ArcVault Controller" cmd /k "python source\apps\backup-control\controller\run_controller.py"
echo.
echo echo Waiting for controller to start...
echo timeout /t 5 /nobreak ^>nul
echo.
echo echo Opening web interface...
echo start http://localhost:8501
echo.
echo echo.
echo echo ========================================
echo echo ArcVault Demo Running!
echo echo ========================================
echo echo.
echo echo Services started:
echo echo - Backup Agent: http://localhost:8443
echo echo - Web Interface: http://localhost:8501
echo echo.
echo echo To test:
echo echo 1. Go to http://localhost:8501
echo echo 2. Add agent: http://localhost:8443
echo echo 3. Create a backup job
echo echo 4. Monitor progress
echo echo.
echo echo Press any key to stop all services...
echo pause
echo.
echo echo Stopping services...
echo taskkill /f /im python.exe 2^>nul
echo echo Demo stopped.
echo pause
) > "%DEMO_DIR%\run_demo.bat"

REM Create README for friend
(
echo ArcVault Demo Package
echo ====================
echo.
echo Hi! This is a demo of ArcVault, a backup system I've been working on.
echo.
echo WHAT IS ARCVault?
echo ================
echo ArcVault is a backup system that consists of:
echo - Backup Agent: Runs on client machines, performs backups using Robocopy
echo - Backup Controller: Web interface for managing backup jobs and agents
echo - JWT Authentication: Secure communication between components
echo - Self-signed Certificates: HTTPS communication without external certificates
echo - Docker Support: Cross-platform server deployment
echo.
echo QUICK START
echo ===========
echo 1. Run setup_demo.bat (installs dependencies and configures environment)
echo 2. Run run_demo.bat (starts all services and opens web interface)
echo 3. Go to http://localhost:8501 to see the web interface
echo 4. Add agent at http://localhost:8443
echo 5. Create a backup job and test it
echo.
echo REQUIREMENTS
echo ============
echo - Windows 10/11 or Windows Server 2016+
echo - Python 3.8+ (will be installed in virtual environment)
echo - Administrator privileges (for Robocopy and service management)
echo.
echo WHAT TO TEST
echo ============
echo 1. Web Interface: http://localhost:8501
echo    - Add backup agents
echo    - Create backup jobs
echo    - Monitor backup progress
echo    - View backup logs
echo.
echo 2. Agent API: http://localhost:8443
echo    - Health check: http://localhost:8443/health
echo    - Agent info: http://localhost:8443/info
echo    - Job status: http://localhost:8443/jobs/{job_id}
echo.
echo 3. Backup Jobs
echo    - Create jobs with different source/destination paths
echo    - Test with local folders
echo    - Test with UNC network paths
echo    - Monitor progress and logs
echo.
echo 4. Authentication
echo    - JWT token generation and validation
echo    - API key authentication
echo    - Secure communication over HTTPS
echo.
echo 5. Docker Deployment (Optional)
echo    - docker-compose up -d
echo    - Test cross-platform server deployment
echo.
echo FEATURES TO EXPLORE
echo ===================
echo - Robocopy Integration: Windows-native backup tool
echo - JWT Authentication: Production-ready security
echo - Self-signed Certificates: HTTPS without external CAs
echo - Docker Support: Cross-platform deployment
echo - Web Interface: Streamlit-based control panel
echo - Real-time Logs: Server-sent events for live updates
echo - Service Management: Windows service integration
echo - Configuration Management: Environment-based settings
echo.
echo FILES TO LOOK AT
echo ================
echo - source/apps/backup-agent/agent/app.py: FastAPI agent server
echo - source/apps/backup-control/controller/app.py: Streamlit web interface
echo - source/apps/backup-agent/agent/engine/robocopy.py: Robocopy integration
echo - scripts/generate_self_signed_cert.py: Certificate generation
echo - scripts/generate_jwt_keys.py: JWT key generation
echo - docker-compose.yml: Docker deployment configuration
echo.
echo DEPLOYMENT OPTIONS
echo ==================
echo 1. Development: Python virtual environment (this demo)
echo 2. Production: EXE installers (no Python required)
echo 3. Docker: Containerized deployment
echo 4. Unified: Single installer with everything
echo.
echo FEEDBACK WANTED
echo ===============
echo I'd love your thoughts on:
echo - Code structure and organization
echo - User interface and experience
echo - Security implementation
echo - Deployment approach
echo - Feature suggestions
echo - Bug reports
echo - Performance improvements
echo.
echo CONTACT
echo =======
echo Feel free to reach out with any questions or suggestions!
echo.
echo Enjoy testing ArcVault! 🚀
) > "%DEMO_DIR%\README.txt"

REM Create a simple test script
(
echo @echo off
echo echo ArcVault Demo Test Script
echo echo ========================
echo echo.
echo echo Testing ArcVault components...
echo echo.
echo.
echo REM Test agent health
echo echo Testing agent health...
echo curl -s http://localhost:8443/health 2^>nul
echo if %%errorLevel%% == 0 ^(
echo     echo ✅ Agent is running
echo ^) else ^(
echo     echo ❌ Agent is not responding
echo ^)
echo.
echo REM Test controller
echo echo Testing controller...
echo curl -s http://localhost:8501 2^>nul
echo if %%errorLevel%% == 0 ^(
echo     echo ✅ Controller is running
echo ^) else ^(
echo     echo ❌ Controller is not responding
echo ^)
echo.
echo REM Test agent info
echo echo Testing agent info...
echo curl -s http://localhost:8443/info 2^>nul
echo if %%errorLevel%% == 0 ^(
echo     echo ✅ Agent info endpoint working
echo ^) else ^(
echo     echo ❌ Agent info endpoint not working
echo ^)
echo.
echo echo.
echo echo Test complete!
echo pause
) > "%DEMO_DIR%\test_demo.bat"

REM Create a zip file
echo Creating zip file...
powershell -Command "Compress-Archive -Path '%DEMO_DIR%\*' -DestinationPath 'ArcVault-Demo.zip' -Force"

echo.
echo ========================================
echo Demo Package Complete!
echo ========================================
echo.
echo Package created: ArcVault-Demo.zip
echo.
echo Contents:
echo - Complete source code
echo - All scripts and configuration
echo - Docker deployment files
echo - Documentation
echo - Demo setup and run scripts
echo - Test scripts
echo.
echo To share with your friend:
echo 1. Send them ArcVault-Demo.zip
echo 2. They extract it and run setup_demo.bat
echo 3. They run run_demo.bat to start the demo
echo 4. They go to http://localhost:8501 to test
echo.
echo Package size:
for /f %%i in ('dir "ArcVault-Demo.zip" /-c ^| find "File(s)"') do echo %%i bytes

popd
endlocal

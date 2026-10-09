@echo off
setlocal ENABLEDELAYEDEXPANSION

echo ========================================
echo ArcVault Demo Setup
echo ========================================
echo.
echo This will set up ArcVault for testing and demonstration.
echo.
echo What this demo includes:
echo - Backup Agent (FastAPI server)
echo - Backup Controller (Streamlit web interface)
echo - Robocopy integration for Windows
echo - JWT authentication
echo - Self-signed certificates
echo - Docker deployment option
echo.
echo.
REM Check for Python
python --version >nul 2>&1
if %errorLevel% NEQ 0 (
    echo ERROR: Python is not installed or not in PATH.
    echo Please install Python 3.8+ and try again.
    echo Download from: https://www.python.org/downloads/
    pause
    exit /b 1
)

echo Python found. Checking version...
python --version

REM Create virtual environment
echo Creating virtual environment...
python -m venv .venv
if %errorLevel% NEQ 0 (
    echo ERROR: Failed to create virtual environment.
    pause
    exit /b 1
)

REM Activate virtual environment
echo Activating virtual environment...
call .venv\Scripts\activate

REM Install dependencies
echo Installing dependencies...
python -m pip install --upgrade pip
python -m pip install -e "source\apps\backup-agent"
python -m pip install -e "source\apps\backup-control"

REM Generate certificates and keys
echo Generating demo certificates and keys...
mkdir keys 2>nul
python scripts\generate_self_signed_cert.py keys localhost
python scripts\generate_jwt_keys.py keys

REM Create demo environment file
echo Creating demo environment configuration...
copy "scripts\env.templates\development.env" ".env"

REM Update environment for demo
echo Updating environment for demo...
powershell -Command "^(Get-Content '.env'^) -replace 'your-controller-hostname.domain.com', 'localhost' ^| Set-Content '.env'"
powershell -Command "^(Get-Content '.env'^) -replace 'AGENT_CERTFILE=keys/server.crt', 'AGENT_CERTFILE=keys/server.crt' ^| Set-Content '.env'"
powershell -Command "^(Get-Content '.env'^) -replace 'AGENT_KEYFILE=keys/server.key', 'AGENT_KEYFILE=keys/server.key' ^| Set-Content '.env'"
powershell -Command "^(Get-Content '.env'^) -replace 'AGENT_CA_BUNDLE=keys/ca-bundle.crt', 'AGENT_CA_BUNDLE=keys/ca-bundle.crt' ^| Set-Content '.env'"

echo.
echo ========================================
echo Demo Setup Complete
echo ========================================
echo.
echo Next steps:
echo 1. Start the backup agent: python source\apps\backup-agent\agent\run_agent.py
echo 2. Start the controller: python source\apps\backup-control\controller\run_controller.py
echo 3. Open web interface: http://localhost:8501
echo 4. Add agent: http://localhost:8443
echo.
echo Or use the quick start script: run_demo.bat
echo.
pause

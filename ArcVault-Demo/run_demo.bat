@echo off
setlocal ENABLEDELAYEDEXPANSION

echo ========================================
echo ArcVault Demo - Quick Start
echo ========================================
echo.
echo Starting ArcVault demo...
echo.

REM Check if setup was run
if not exist ".venv" (
    echo ERROR: Demo setup not run yet.
    echo Please run setup_demo.bat first.
    pause
    exit /b 1
)

REM Activate virtual environment
call .venv\Scripts\activate

REM Start services
echo Starting backup agent...
start "ArcVault Agent" cmd /k "python source\apps\backup-agent\agent\run_agent.py"

echo Waiting for agent to start...
timeout /t 3 /nobreak >nul

echo Starting backup controller...
start "ArcVault Controller" cmd /k "python source\apps\backup-control\controller\run_controller.py"

echo Waiting for controller to start...
timeout /t 5 /nobreak >nul

echo Opening web interface...
start http://localhost:8501

echo.
echo ========================================
echo ArcVault Demo Running
echo ========================================
echo.
echo Services started:
echo - Backup Agent: http://localhost:8443
echo - Web Interface: http://localhost:8501
echo.
echo To test:
echo 1. Go to http://localhost:8501
echo 2. Add agent: http://localhost:8443
echo 3. Create a backup job
echo 4. Monitor progress
echo.
echo Press any key to stop all services...
pause

echo Stopping services...
taskkill /f /im python.exe 2>nul
echo Demo stopped.
pause

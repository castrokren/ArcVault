@echo off
setlocal ENABLEDELAYEDEXPANSION

REM Build a Windows Service EXE for the backup agent using PyInstaller.

set SCRIPT_DIR=%~dp0
pushd "%SCRIPT_DIR%\..\..\.."

if not exist .venv_agent_svc (
    python -m venv .venv_agent_svc
)
call .venv_agent_svc\Scripts\activate

python -m pip install --upgrade pip
python -m pip install pyinstaller pywin32
python -m pip install -e "apps\backup-agent"

pyinstaller --noconfirm --clean --onefile ^
  --name ArcVaultAgentSvc ^
  apps\backup-agent\agent\win_service.py

if %ERRORLEVEL% NEQ 0 (
  echo PyInstaller build failed.
  popd
  exit /b %ERRORLEVEL%
)

echo.
echo Build complete. Output: dist\ArcVaultAgentSvc.exe
echo Install with: ArcVaultAgentSvc.exe install  and then  ArcVaultAgentSvc.exe --startup auto start
echo.

popd
endlocal



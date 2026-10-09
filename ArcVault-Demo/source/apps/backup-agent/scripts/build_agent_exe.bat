@echo off
setlocal ENABLEDELAYEDEXPANSION

REM Build a self-contained Windows EXE for the backup agent using PyInstaller.
REM This script can be run from any directory.

REM Determine repo root (two levels up from this script file)
set SCRIPT_DIR=%~dp0
pushd "%SCRIPT_DIR%\..\..\.."

REM Create and activate an isolated virtual environment for building
if not exist .venv_agent (
    python -m venv .venv_agent
)
call .venv_agent\Scripts\activate

python -m pip install --upgrade pip
python -m pip install pyinstaller
python -m pip install -e "apps\backup-agent"

REM Build single-file EXE
pyinstaller --noconfirm --clean --onefile ^
  --name ArcVaultAgent ^
  apps\backup-agent\agent\run_agent.py

if %ERRORLEVEL% NEQ 0 (
  echo PyInstaller build failed.
  popd
  exit /b %ERRORLEVEL%
)

echo.
echo Build complete. Output: dist\ArcVaultAgent.exe
echo Distribute the public key and set AGENT_REQUIRE_JWT=true on the target hosts.
echo.

popd
endlocal

@echo off
setlocal ENABLEDELAYEDEXPANSION

REM Build a self-contained Windows EXE for the agent using PyInstaller.
REM This script can be run from any directory.

set SCRIPT_DIR=%~dp0
pushd "%SCRIPT_DIR%\..\.."

if not exist .venv_agent (
    python -m venv .venv_agent
)
call .venv_agent\Scripts\activate

python -m pip install --upgrade pip
python -m pip install pyinstaller
python -m pip install -e "apps\backup-agent"

REM Build single-file EXE. Bundle the agent package directory as data so
REM runtime assets (if any later) are included.
pyinstaller --noconfirm --clean --onefile ^
  --name ArcVaultAgent ^
  apps\backup-agent\agent\run_agent.py ^
  --add-data "apps\backup-agent\agent;apps\backup-agent\agent"

if %ERRORLEVEL% NEQ 0 (
  echo PyInstaller build failed.
  popd
  exit /b %ERRORLEVEL%
)

echo.
echo Build complete. Output: dist\ArcVaultAgent.exe
echo Consider signing the executable using your signing script.
echo.

popd
endlocal



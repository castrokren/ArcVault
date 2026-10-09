@echo off
setlocal ENABLEDELAYEDEXPANSION

REM Build a self-contained Windows EXE for the controller using PyInstaller.
REM This script can be run from any directory.

REM Determine repo root (two levels up from this script file)
set SCRIPT_DIR=%~dp0
pushd "%SCRIPT_DIR%\..\..\.."

REM Create and activate an isolated virtual environment
if not exist .venv_controller (
    python -m venv .venv_controller
)
call .venv_controller\Scripts\activate

python -m pip install --upgrade pip
python -m pip install pyinstaller
python -m pip install -e "apps\backup-control"

REM Build single-file EXE. Bundle the controller directory as data so the
REM Streamlit app (app.py, agents.json) is available at runtime.
pyinstaller --noconfirm --clean --onefile ^
  --name ArcVaultController ^
  apps\backup-control\controller\run_controller.py ^
  --add-data "apps\backup-control\controller;apps\backup-control\controller"

if %ERRORLEVEL% NEQ 0 (
  echo PyInstaller build failed.
  popd
  exit /b %ERRORLEVEL%
)

echo.
echo Build complete. Output: dist\ArcVaultController.exe
echo You can sign the executable with your existing signing scripts if desired.
echo.

popd
endlocal



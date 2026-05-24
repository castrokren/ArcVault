@echo off
:: ============================================================
::  ArcVault 2.0 — Build Installer EXE
::  Run this once from C:\Projects\ArcVault2.0\deployment\
::  Requires: Python 3.8+ installed and on PATH
:: ============================================================
setlocal
set SCRIPT=arcvault_installer.py
set OUT_NAME=ArcVault-Installer

echo.
echo  ArcVault 2.0 ^| Build Installer EXE
echo  =====================================
echo.

python --version >nul 2>&1
if errorlevel 1 (
    echo  [ERROR] Python not found on PATH.
    echo  Download from https://python.org ^(check "Add Python to PATH"^).
    pause & exit /b 1
)

echo  [1/3]  Installing PyInstaller...
pip install pyinstaller --quiet
if errorlevel 1 ( echo  [ERROR] pip install failed. & pause & exit /b 1 )

echo  [2/3]  Compiling %SCRIPT% ^to %OUT_NAME%.exe ...
pyinstaller ^
    --onefile ^
    --noconsole ^
    --name "%OUT_NAME%" ^
    --distpath "%CD%" ^
    --workpath "%TEMP%\arcvault_pybuild" ^
    --specpath "%TEMP%" ^
    %SCRIPT%
if errorlevel 1 ( echo  [ERROR] PyInstaller failed. & pause & exit /b 1 )

echo  [3/3]  Cleaning up...
if exist "%TEMP%\arcvault_pybuild" rd /s /q "%TEMP%\arcvault_pybuild" 2>nul
if exist "%TEMP%\%OUT_NAME%.spec"  del /q "%TEMP%\%OUT_NAME%.spec"  2>nul

echo.
echo  Done!  Distribute this file:
echo.
echo    %CD%\%OUT_NAME%.exe
echo.
echo  Users just double-click it. The exe requests admin rights,
echo  downloads the ArcVault binary, and installs the service automatically.
echo.
pause

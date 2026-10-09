@echo off
echo ========================================
echo ArcVault GUI Admin Rebuild Script
echo ========================================
echo.

REM Check if running as administrator
net session >nul 2>&1
if %errorlevel% neq 0 (
    echo ERROR: This script must be run as Administrator!
    echo Right-click this script and select "Run as administrator"
    echo.
    pause
    exit /b 1
)

echo Running with administrator privileges...
echo.

echo [1/4] Killing all ArcVault GUI processes...
taskkill /f /im "ArcVault-GUI-PyQt5.exe" >nul 2>&1
taskkill /f /im "ArcVault-GUI.exe" >nul 2>&1
wmic process where "name like '%%ArcVault-GUI%%'" delete >nul 2>&1

echo [2/4] Removing locked executable files...
del /f /q "installers-exe\ArcVault-GUI-PyQt5.exe" >nul 2>&1
del /f /q "installers-exe\ArcVault-GUI.exe" >nul 2>&1

echo [3/4] Waiting for file system to release locks...
timeout /t 3 /nobreak >nul

echo [4/4] Rebuilding PyQt5 GUI...
call "scripts\build_gui_pyqt5_executable.bat"

echo.
echo ========================================
echo Rebuild complete!
echo ========================================

REM Check if the new executable was created
if exist "installers-exe\ArcVault-GUI-PyQt5.exe" (
    echo SUCCESS: New ArcVault-GUI-PyQt5.exe created successfully!
    echo File size:
    dir "installers-exe\ArcVault-GUI-PyQt5.exe" | findstr "ArcVault-GUI-PyQt5.exe"
) else (
    echo ERROR: Failed to create new executable.
    echo Check the build output above for errors.
)

echo.
pause

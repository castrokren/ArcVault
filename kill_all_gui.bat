@echo off
echo ========================================
echo ArcVault GUI Process Cleanup Utility
echo ========================================
echo.

REM Check if running as administrator
net session >nul 2>&1
if %errorlevel% neq 0 (
    echo This script requires administrator privileges to kill all processes.
    echo Please run as administrator for complete cleanup.
    echo.
    echo Attempting standard cleanup...
) else (
    echo Running with administrator privileges - full cleanup possible.
)

echo.
echo Killing all ArcVault GUI processes...

REM Kill all PyQt5 GUI instances
echo - Terminating PyQt5 GUI instances...
taskkill /f /im "ArcVault-GUI-PyQt5.exe" 2>nul
if %errorlevel% equ 0 (
    echo   SUCCESS: PyQt5 instances terminated
) else (
    echo   INFO: No PyQt5 instances found or access denied
)

REM Kill all tkinter GUI instances  
echo - Terminating tkinter GUI instances...
taskkill /f /im "ArcVault-GUI.exe" 2>nul
if %errorlevel% equ 0 (
    echo   SUCCESS: Tkinter instances terminated
) else (
    echo   INFO: No tkinter instances found or access denied
)

REM Kill any Python processes running the GUI scripts
echo - Terminating Python GUI processes...
taskkill /f /fi "WINDOWTITLE eq *arcvault_gui*" 2>nul

REM Try to kill by process name pattern (if running as admin)
net session >nul 2>&1
if %errorlevel% equ 0 (
    echo - Attempting advanced cleanup...
    wmic process where "name like '%%ArcVault-GUI%%'" delete 2>nul
)

echo.
echo ========================================
echo Remaining ArcVault processes:
echo ========================================
tasklist | findstr "ArcVault" 2>nul
if %errorlevel% neq 0 (
    echo No ArcVault processes found - cleanup successful!
) else (
    echo.
    echo Some processes may still be running due to access restrictions.
    echo If processes persist, try:
    echo 1. Restart your computer
    echo 2. Run this script as Administrator
    echo 3. Use Task Manager to end processes manually
)

echo.
echo ========================================
echo Cleanup complete!
echo ========================================
pause

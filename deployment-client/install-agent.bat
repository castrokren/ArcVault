@echo off
setlocal ENABLEDELAYEDEXPANSION

echo ========================================
echo ArcVault Agent Installer
echo ========================================
echo.
echo This installer will install the ArcVault backup agent on this computer.
echo The agent will connect to your backup server to receive backup jobs.
echo.

REM Check for administrator privileges
net session >nul 2>&1
if %errorLevel% == 0 (
    echo Running with administrator privileges...
) else (
    echo ERROR: This installer requires administrator privileges.
    echo Please run as Administrator.
    pause
    exit /b 1
)

REM Set installation directory
set INSTALL_DIR=%ProgramFiles%\ArcVault
set CONFIG_DIR=%ProgramData%\ArcVault

echo Installing ArcVault Agent to: %INSTALL_DIR%
echo Configuration directory: %CONFIG_DIR%

REM Create directories
mkdir "%INSTALL_DIR%" 2>nul
mkdir "%CONFIG_DIR%" 2>nul
mkdir "%CONFIG_DIR%\keys" 2>nul
mkdir "%CONFIG_DIR%\logs" 2>nul

REM Copy executables
echo Copying ArcVault Agent...
copy "agent\ArcVaultAgent.exe" "%INSTALL_DIR%\"
copy "agent\ArcVaultAgentService.exe" "%INSTALL_DIR%\"

REM Copy configuration templates
copy "config\*.env" "%CONFIG_DIR%\"

REM Set permissions
echo Setting permissions...
icacls "%INSTALL_DIR%" /inheritance:r /grant:r "Administrators:F" "SYSTEM:F" /T
icacls "%CONFIG_DIR%" /inheritance:r /grant:r "Administrators:F" "SYSTEM:F" /T

echo.
echo ========================================
echo Agent Installation Complete
echo ========================================
echo.
echo Next steps:
echo 1. Run: %INSTALL_DIR%\ArcVaultAgentSetup.exe
echo 2. Configure connection to backup server
echo 3. Start the agent service
echo.
echo The agent will be available at: http://%COMPUTERNAME%:8443
echo.
pause

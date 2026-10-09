@echo off
setlocal ENABLEDELAYEDEXPANSION

echo ========================================
echo ArcVault Agent Service Manager
echo ========================================
echo.

set INSTALL_DIR=%ProgramFiles%\ArcVault
set CONFIG_DIR=%ProgramData%\ArcVault

:MENU
echo ArcVault Agent Service Manager
echo ===============================
echo.
echo 1. Install Agent Service
echo 2. Start Agent Service
echo 3. Stop Agent Service
echo 4. Uninstall Agent Service
echo 5. View Service Status
echo 6. View Agent Logs
echo 7. Test Agent Connection
echo 8. Exit
echo.
set /p CHOICE="Enter your choice (1-8): "

if "%CHOICE%"=="1" goto INSTALL_SERVICE
if "%CHOICE%"=="2" goto START_SERVICE
if "%CHOICE%"=="3" goto STOP_SERVICE
if "%CHOICE%"=="4" goto UNINSTALL_SERVICE
if "%CHOICE%"=="5" goto VIEW_STATUS
if "%CHOICE%"=="6" goto VIEW_LOGS
if "%CHOICE%"=="7" goto TEST_CONNECTION
if "%CHOICE%"=="8" goto EXIT
echo Invalid choice. Please try again.
goto MENU

:INSTALL_SERVICE
echo Installing ArcVault Agent Service...
"%INSTALL_DIR%\ArcVaultAgentService.exe" install
if %ERRORLEVEL% == 0 (
    echo Service installed successfully.
) else (
    echo Failed to install service.
)
pause
goto MENU

:START_SERVICE
echo Starting ArcVault Agent Service...
net start ArcVaultAgent
if %ERRORLEVEL% == 0 (
    echo Service started successfully.
) else (
    echo Failed to start service.
)
pause
goto MENU

:STOP_SERVICE
echo Stopping ArcVault Agent Service...
net stop ArcVaultAgent
if %ERRORLEVEL% == 0 (
    echo Service stopped successfully.
) else (
    echo Failed to stop service.
)
pause
goto MENU

:UNINSTALL_SERVICE
echo Uninstalling ArcVault Agent Service...
net stop ArcVaultAgent 2>nul
"%INSTALL_DIR%\ArcVaultAgentService.exe" remove
if %ERRORLEVEL% == 0 (
    echo Service uninstalled successfully.
) else (
    echo Failed to uninstall service.
)
pause
goto MENU

:VIEW_STATUS
echo ArcVault Agent Service Status:
echo ==============================
sc query ArcVaultAgent
echo.
echo Agent Health Check:
curl -s http://localhost:8443/health 2>nul || echo "Agent not responding"
pause
goto MENU

:VIEW_LOGS
echo ArcVault Agent Logs:
echo ====================
echo.
echo Service Logs:
echo ------------
powershell -Command "Get-EventLog -LogName Application -Source 'ArcVault*' -Newest 10 ^| Format-Table TimeGenerated, EntryType, Message -Wrap"
echo.
echo Backup Logs:
echo -----------
if exist "%CONFIG_DIR%\logs" (
    dir "%CONFIG_DIR%\logs" /b
) else (
    echo No backup logs found.
)
pause
goto MENU

:TEST_CONNECTION
echo Testing Agent Connection:
echo =========================
echo.
echo Testing local agent...
curl -s http://localhost:8443/health
echo.
echo Testing agent info...
curl -s http://localhost:8443/info
echo.
pause
goto MENU

:EXIT
echo Goodbye
exit /b 0

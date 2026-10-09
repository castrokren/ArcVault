@echo off
setlocal ENABLEDELAYEDEXPANSION

echo ========================================
echo ArcVault Agent Configuration
echo ========================================
echo.
echo This wizard will configure the ArcVault agent to connect to your backup server.
echo.

set INSTALL_DIR=%ProgramFiles%\ArcVault
set CONFIG_DIR=%ProgramData%\ArcVault

REM Server connection configuration
echo Configure connection to backup server:
echo.
set /p SERVER_URL="Enter backup server URL (e.g., https://backup-server.company.com:8443): "
if "%SERVER_URL%"=="" (
    echo ERROR: Server URL is required.
    pause
    exit /b 1
)

echo Select authentication method:
echo 1. Simple token (development/testing)
echo 2. JWT authentication (production)
echo.
set /p AUTH_CHOICE="Enter your choice (1-2): "

if "%AUTH_CHOICE%"=="1" (
    set /p AUTH_TOKEN="Enter authentication token: "
    if "%AUTH_TOKEN%"=="" set AUTH_TOKEN=dev-token-12345
) else if "%AUTH_CHOICE%"=="2" (
    echo JWT authentication requires the server's public key.
    echo Please copy the server's public key to: %CONFIG_DIR%\keys\controller_public.pem
    echo.
    pause
    if not exist "%CONFIG_DIR%\keys\controller_public.pem" (
        echo ERROR: JWT public key not found.
        echo Please copy the key file and run setup again.
        pause
        exit /b 1
    )
) else (
    echo Invalid choice. Please try again.
    pause
    exit /b 1
)

REM Configure agent settings
echo.
echo Configure agent settings:
echo.
set /p AGENT_PORT="Agent port (default: 8443): "
if "%AGENT_PORT%"=="" set AGENT_PORT=8443

set /p AGENT_HOST="Agent hostname (default: %COMPUTERNAME%): "
if "%AGENT_HOST%"=="" set AGENT_HOST=%COMPUTERNAME%

REM Update environment file

REM SSL configuration
echo.
echo SSL Configuration:
echo 1. HTTP (development)
echo 2. HTTPS (production)
echo.
set /p SSL_CHOICE="Enter your choice (1-2): "

if "%SSL_CHOICE%"=="2" (
    echo HTTPS requires SSL certificates.
    echo Please copy your certificates to: %CONFIG_DIR%\keys\
    echo Required files: agent.crt, agent.key, ca-bundle.crt
    echo.
    pause
) else (
)

echo.
echo ========================================
echo Agent Configuration Complete
echo ========================================
echo.
echo Configuration saved to: %CONFIG_DIR%\.env
echo.
echo Agent will be available at: http://%AGENT_HOST%:%AGENT_PORT%
echo.
echo Next steps:
echo 1. Install the agent service
echo 2. Start the service
echo 3. Register this agent with your backup server
echo.
pause

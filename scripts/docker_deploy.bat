@echo off
setlocal ENABLEDELAYEDEXPANSION

REM ArcVault Docker Deployment Script for Windows
REM Deploys ArcVault server using Docker for cross-platform support

echo ========================================
echo ArcVault Docker Deployment
echo ========================================
echo.

REM Check if Docker is installed
docker --version >nul 2>&1
if %ERRORLEVEL% NEQ 0 (
    echo ERROR: Docker is not installed.
    echo Please install Docker Desktop and try again.
    pause
    exit /b 1
)

REM Check if Docker Compose is installed
docker-compose --version >nul 2>&1
if %ERRORLEVEL% NEQ 0 (
    echo ERROR: Docker Compose is not installed.
    echo Please install Docker Compose and try again.
    pause
    exit /b 1
)

REM Create data directories
echo Creating data directories...
if not exist "data" mkdir data
if not exist "data\keys" mkdir data\keys
if not exist "data\logs" mkdir data\logs
if not exist "data\config" mkdir data\config
if not exist "nginx" mkdir nginx

REM Copy configuration templates
echo Copying configuration templates...
copy "env.templates\production.env" "data\config\production.env"
copy "env.templates\staging.env" "data\config\staging.env"
copy "env.templates\development.env" "data\config\development.env"

REM Generate self-signed certificates if they don't exist
if not exist "data\keys\server.crt" (
    echo Generating self-signed certificates...
    
    REM Check if we have the certificate generator
    if exist "scripts\generate_self_signed_cert.py" (
        python scripts\generate_self_signed_cert.py data\keys localhost
    ) else (
        echo WARNING: Certificate generator not found.
        echo You may need to generate certificates manually.
    )
)

REM Generate JWT keys if they don't exist
if not exist "data\keys\controller_private.pem" (
    echo Generating JWT keys...
    
    REM Check if we have the JWT key generator
    if exist "scripts\generate_jwt_keys.py" (
        python scripts\generate_jwt_keys.py data\keys
    ) else (
        echo WARNING: JWT key generator not found.
        echo You may need to generate JWT keys manually.
    )
)

REM Set up environment file
echo Setting up environment configuration...
(
echo # ArcVault Docker Configuration
echo CONTROLLER_PORT=8501
echo CONTROLLER_ADDRESS=0.0.0.0
echo STREAMLIT_HEADLESS=1
echo AGENT_VERIFY_SSL=false
echo AGENT_REQUIRE_JWT=true
echo AGENT_JWT_PUBLIC_KEY_PATH=/app/keys/controller_public.pem
echo AGENT_JWT_ISSUER=backup-controller
echo AGENT_JWT_AUDIENCE=backup-agent
echo CTRL_JWT_PRIVATE_KEY_PATH=/app/keys/controller_private.pem
echo CTRL_JWT_ISSUER=backup-controller
echo CTRL_JWT_AUDIENCE=backup-agent
) > "data\config\.env"

REM Build and start services
echo Building and starting ArcVault server...
docker-compose up -d --build

if %ERRORLEVEL% NEQ 0 (
    echo ERROR: Failed to build or start Docker services.
    echo Check Docker is running and try again.
    pause
    exit /b %ERRORLEVEL%
)

REM Wait for service to be ready
echo Waiting for service to be ready...
timeout /t 10 /nobreak >nul

REM Check if service is running
docker-compose ps | findstr "arcvault-server.*Up" >nul
if %ERRORLEVEL% == 0 (
    echo.
    echo ========================================
    echo ArcVault Server Deployment Complete!
    echo ========================================
    echo.
    echo ✅ ArcVault server is running!
    echo.
    echo Access points:
    echo - Web Interface: http://localhost:8501
    echo - With Nginx (HTTPS): https://localhost:443 (if enabled)
    echo.
    echo Management commands:
    echo - View logs: docker-compose logs -f
    echo - Stop service: docker-compose down
    echo - Restart service: docker-compose restart
    echo - Update service: docker-compose up -d --build
    echo.
    echo Configuration files:
    echo - Environment: data\config\.env
    echo - JWT Keys: data\keys\
    echo - Logs: data\logs\
    echo.
    echo To enable HTTPS with Nginx:
    echo docker-compose --profile with-nginx up -d
    echo.
) else (
    echo ❌ Failed to start ArcVault server.
    echo Check logs with: docker-compose logs
    pause
    exit /b 1
)

pause

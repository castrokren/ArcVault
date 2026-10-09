@echo off
echo ArcVault Demo Test Script
echo ========================
echo.
echo Testing ArcVault components...
echo.

REM Test agent health
echo Testing agent health...
curl -s http://localhost:8443/health 2>nul
if %errorLevel% == 0 (
    echo ✅ Agent is running
) else (
    echo ❌ Agent is not responding
)

REM Test controller
echo Testing controller...
curl -s http://localhost:8501 2>nul
if %errorLevel% == 0 (
    echo ✅ Controller is running
) else (
    echo ❌ Controller is not responding
)

REM Test agent info
echo Testing agent info...
curl -s http://localhost:8443/info 2>nul
if %errorLevel% == 0 (
    echo ✅ Agent info endpoint working
) else (
    echo ❌ Agent info endpoint not working
)

echo.
echo Test complete
pause

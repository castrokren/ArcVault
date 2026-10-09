@echo off
echo ArcVault Agent - Client Installation
echo ====================================
echo.
echo Choose an option:
echo 1. Install Agent
echo 2. Configure Agent  
echo 3. Manage Agent Service
echo 4. View Documentation
echo 5. Exit
echo.
set /p CHOICE="Enter your choice (1-5): "

if "%CHOICE%"=="1" start install-agent.bat
if "%CHOICE%"=="2" start setup-agent.bat
if "%CHOICE%"=="3" start agent-manager.bat
if "%CHOICE%"=="4" start README.txt
if "%CHOICE%"=="5" exit

Invalid choice. Please try again.
pause
goto :eof

@echo off
REM Run Phase 22 Week 2 failure injection tests

cd /d "%~dp0"
echo Running failure injection tests...
echo.

go test -v -run TestAgent
echo.
echo.

echo Running network timeout scenario test...
go test -v -run TestNetworkTimeout
echo.
echo.

echo Running coordinator crash recovery test...
go test -v -run TestCoordinatorCrashRecovery
echo.
echo.

echo Running database lock test...
go test -v -run TestDatabaseLockUnderLoad
echo.
echo.

echo All failure injection tests complete!
pause

@echo off
cd /d C:\Projects\ArcVault2.0
echo === BUILD COORDINATOR ===
go build ./coordinator/...
if %ERRORLEVEL% neq 0 (echo COORDINATOR BUILD FAILED && exit /b 1)
echo Coordinator build OK

echo === BUILD AGENT ===
go build ./agent/...
if %ERRORLEVEL% neq 0 (echo AGENT BUILD FAILED && exit /b 1)
echo Agent build OK

echo === RUN TESTS ===
go test ./... -count=1 -timeout 120s 2>&1
if %ERRORLEVEL% neq 0 (echo TESTS FAILED && pause && exit /b 1)
echo ALL TESTS PASSED
pause

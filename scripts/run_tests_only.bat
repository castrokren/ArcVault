@echo off
cd /d C:\Projects\ArcVault2.0
go test ./... -count=1 -timeout 120s > test_output.log 2>&1
echo %ERRORLEVEL% > test_exit_code.txt

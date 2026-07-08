@echo off
REM Run Phase 22 load harness tests

cd /d "%~dp0"
echo Running arcvault-test harness tests...
echo.

go test -v

echo.
echo Tests complete. Running basic load test...
echo.

go run . load --agents=10 --jobs-per-agent=20 --job-duration=100 --output=test_report.json

echo.
echo Load test complete. Report saved to test_report.json
pause

@echo off
REM Phase 22 Week 4: Full Test Suite Execution & Performance Validation
REM This script runs the complete test suite at multiple load levels

cd /d "%~dp0"
setlocal enabledelayedexpansion

echo.
echo ============================================
echo Phase 22 Week 4: Full Test Suite Validation
echo ============================================
echo.

REM Create reports directory
if not exist reports mkdir reports

echo Running baseline unit tests...
go test -v > reports\unit_tests.log 2>&1
if %ERRORLEVEL% NEQ 0 (
    echo WARNING: Some unit tests failed (see unit_tests.log)
    echo Continuing with load tests...
) else (
    echo ✓ Unit tests passed
)

echo.
echo ============================================
echo Load Test 1: Baseline (10 agents, 50 jobs)
echo ============================================
go run . load ^
  --agents=10 ^
  --jobs-per-agent=50 ^
  --job-duration=100 ^
  --output=reports\baseline_10_agents.json

echo.
echo ============================================
echo Load Test 2: Medium (50 agents, 20 jobs)
echo ============================================
go run . load ^
  --agents=50 ^
  --jobs-per-agent=20 ^
  --job-duration=100 ^
  --output=reports\medium_50_agents.json

echo.
echo ============================================
echo Load Test 3: High (100 agents, 10 jobs)
echo ============================================
go run . load ^
  --agents=100 ^
  --jobs-per-agent=10 ^
  --job-duration=100 ^
  --output=reports\high_100_agents.json

echo.
echo ============================================
echo Failure Scenarios: Agent Disconnect
echo ============================================
go run . load ^
  --agents=20 ^
  --jobs-per-agent=15 ^
  --job-duration=500 ^
  --failure-rate=0.2 ^
  --failure-type=disconnect ^
  --output=reports\failure_disconnect.json

echo.
echo ============================================
echo Performance Summary
echo ============================================
echo.
echo Baseline (10 agents):
powershell -Command "Get-Content reports\baseline_10_agents.json | ConvertFrom-Json | Select-Object -Property agents_spawned, jobs_created, jobs_completed, throughput_jobs_per_second, latency_p50_ms, latency_p95_ms, peak_memory_mb | Format-Table"

echo.
echo Medium Load (50 agents):
powershell -Command "Get-Content reports\medium_50_agents.json | ConvertFrom-Json | Select-Object -property agents_spawned, jobs_created, jobs_completed, throughput_jobs_per_second, latency_p50_ms, latency_p95_ms, peak_memory_mb | Format-Table"

echo.
echo High Load (100 agents):
powershell -Command "Get-Content reports\high_100_agents.json | ConvertFrom-Json | Select-Object -property agents_spawned, jobs_created, jobs_completed, throughput_jobs_per_second, latency_p50_ms, latency_p95_ms, peak_memory_mb | Format-Table"

echo.
echo ============================================
echo Results Summary
echo ============================================
echo All reports saved to: reports\
echo.
echo Files:
echo   - baseline_10_agents.json
echo   - medium_50_agents.json
echo   - high_100_agents.json
echo   - failure_disconnect.json
echo   - unit_tests.log
echo.
echo Week 4 test suite complete!
pause

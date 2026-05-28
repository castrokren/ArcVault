@echo off
cd /d C:\Projects\ArcVault2.0
echo === BUILD COORDINATOR === > build_test_output.log 2>&1
go build ./coordinator/... >> build_test_output.log 2>&1
if %ERRORLEVEL% neq 0 (
  echo COORDINATOR BUILD FAILED >> build_test_output.log
  echo DONE_WITH_ERROR >> build_test_output.log
  exit /b 1
)
echo Coordinator build OK >> build_test_output.log

echo === BUILD AGENT === >> build_test_output.log 2>&1
go build ./agent/... >> build_test_output.log 2>&1
if %ERRORLEVEL% neq 0 (
  echo AGENT BUILD FAILED >> build_test_output.log
  echo DONE_WITH_ERROR >> build_test_output.log
  exit /b 1
)
echo Agent build OK >> build_test_output.log

echo === RUN TESTS === >> build_test_output.log 2>&1
go test ./... -count=1 -timeout 120s >> build_test_output.log 2>&1
if %ERRORLEVEL% neq 0 (
  echo TESTS FAILED >> build_test_output.log
  echo DONE_WITH_ERROR >> build_test_output.log
  exit /b 1
)
echo ALL TESTS PASSED >> build_test_output.log
echo DONE_SUCCESS >> build_test_output.log

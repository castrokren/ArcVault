@echo off
cd /d C:\Projects\ArcVault2.0

(
echo === BUILD COORDINATOR ===
go build ./coordinator/...
echo BUILD_COORD_EXIT=%ERRORLEVEL%
echo === BUILD AGENT ===
go build ./agent/...
echo BUILD_AGENT_EXIT=%ERRORLEVEL%
echo === RUN TESTS ===
go test ./... -count=1 -timeout 120s -v
echo TEST_EXIT=%ERRORLEVEL%
echo SCRIPT_DONE
) > build_test_output2.log 2>&1

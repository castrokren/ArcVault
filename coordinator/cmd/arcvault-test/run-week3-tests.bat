@echo off
REM Run Phase 22 Week 3 integration, edge case, and recovery scenario tests

cd /d "%~dp0"
echo Running Week 3: Integration Tests, Edge Cases, and Recovery Scenarios
echo.
echo ============================================
echo Integration Tests (Multi-agent workflows)
echo ============================================
go test -v -run TestMultiAgent

echo.
echo ============================================
echo Edge Case Tests (Boundary conditions)
echo ============================================
go test -v -run TestLarge
go test -v -run TestHighFile
go test -v -run TestPermission
go test -v -run TestLongJob
go test -v -run TestConcurrent
go test -v -run TestRapid
go test -v -run TestMixed

echo.
echo ============================================
echo Recovery Scenario Tests
echo ============================================
go test -v -run TestJobRecovery
go test -v -run TestMultipleAgent
go test -v -run TestStaggered
go test -v -run TestDatabase
go test -v -run TestProgress
go test -v -run TestCoordinator

echo.
echo All Week 3 tests complete!
pause

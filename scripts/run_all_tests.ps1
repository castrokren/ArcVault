# Comprehensive test run for Step 5 refactor verification
# Tests all affected packages: db, business, server

Set-Location C:\Projects\ArcVault2.0

Write-Host "========================================" -ForegroundColor Green
Write-Host "ArcVault Step 5 Refactor Test Suite" -ForegroundColor Green
Write-Host "========================================" -ForegroundColor Green
Write-Host ""

# Test 1: DB layer tests (queries, job operations)
Write-Host "TEST 1: Database Layer (coordinator/db)" -ForegroundColor Cyan
Write-Host "Testing DB interfaces, job operations, group queries..." -ForegroundColor Yellow
go test ./coordinator/db -v 2>&1 | Tee-Object -FilePath test_db.txt
$db_status = $LASTEXITCODE
Write-Host ""

# Test 2: Business logic tests (service layer)
Write-Host "TEST 2: Business Layer (coordinator/business)" -ForegroundColor Cyan
Write-Host "Testing JobService, agents, groups..." -ForegroundColor Yellow
go test ./coordinator/business -v 2>&1 | Tee-Object -FilePath test_business.txt
$business_status = $LASTEXITCODE
Write-Host ""

# Test 3: Server/Handler tests (API layer)
Write-Host "TEST 3: Server Layer (coordinator/server)" -ForegroundColor Cyan
Write-Host "Testing handlers, job create/delete, group dispatch..." -ForegroundColor Yellow
go test ./coordinator/server -v 2>&1 | Tee-Object -FilePath test_server.txt
$server_status = $LASTEXITCODE
Write-Host ""

# Test 4: Full integration (all packages)
Write-Host "TEST 4: Full Integration (all coordinator packages)" -ForegroundColor Cyan
Write-Host "Running comprehensive test suite..." -ForegroundColor Yellow
go test ./coordinator/... -v 2>&1 | Tee-Object -FilePath test_full.txt
$full_status = $LASTEXITCODE
Write-Host ""

# Summary
Write-Host "========================================" -ForegroundColor Green
Write-Host "TEST EXECUTION SUMMARY" -ForegroundColor Green
Write-Host "========================================" -ForegroundColor Green
Write-Host ""

if ($db_status -eq 0) {
    Write-Host "✓ DB Layer Tests: PASSED" -ForegroundColor Green
} else {
    Write-Host "✗ DB Layer Tests: FAILED" -ForegroundColor Red
}

if ($business_status -eq 0) {
    Write-Host "✓ Business Layer Tests: PASSED" -ForegroundColor Green
} else {
    Write-Host "✗ Business Layer Tests: FAILED" -ForegroundColor Red
}

if ($server_status -eq 0) {
    Write-Host "✓ Server Layer Tests: PASSED" -ForegroundColor Green
} else {
    Write-Host "✗ Server Layer Tests: FAILED" -ForegroundColor Red
}

if ($full_status -eq 0) {
    Write-Host "✓ Full Integration Tests: PASSED" -ForegroundColor Green
} else {
    Write-Host "✗ Full Integration Tests: FAILED" -ForegroundColor Red
}

Write-Host ""
Write-Host "Overall Result: " -NoNewline
if ($db_status -eq 0 -and $business_status -eq 0 -and $server_status -eq 0 -and $full_status -eq 0) {
    Write-Host "ALL TESTS PASSED ✓" -ForegroundColor Green
    exit 0
} else {
    Write-Host "SOME TESTS FAILED ✗" -ForegroundColor Red
    exit 1
}

# run_all_error_handling_tests.ps1
# Master test runner for all error handling tests

<#
.SYNOPSIS
    Master test suite for ArcVault error handling improvements

.DESCRIPTION
    Runs comprehensive tests for both error handling improvements:
    1. Port conflict detection
    2. Network share retry logic with exponential backoff

.PARAMETER SkipPortTest
    Skip the port conflict detection tests

.PARAMETER SkipNetworkTest
    Skip the network retry logic tests

.PARAMETER SkipCertificateCheck
    Skip SSL certificate validation (for test environments)

.EXAMPLE
    .\run_all_error_handling_tests.ps1
    
.EXAMPLE
    .\run_all_error_handling_tests.ps1 -SkipCertificateCheck
#>

param(
    [switch]$SkipPortTest,
    [switch]$SkipNetworkTest,
    [switch]$SkipCertificateCheck
)

# Script configuration
$ErrorActionPreference = "Continue"
$scriptPath = Split-Path -Parent $MyInvocation.MyCommand.Path

Write-Host "================================================================" -ForegroundColor Cyan
Write-Host "   ArcVault Backup Agent - Error Handling Test Suite          " -ForegroundColor Cyan
Write-Host "                                                               " -ForegroundColor Cyan
Write-Host "   This suite validates:                                       " -ForegroundColor Cyan
Write-Host "   1. Port conflict detection                                  " -ForegroundColor Cyan
Write-Host "   2. Network share retry logic with exponential backoff       " -ForegroundColor Cyan
Write-Host "================================================================" -ForegroundColor Cyan
Write-Host ""

$totalPassed = 0
$totalFailed = 0
$testResults = @()

# Check prerequisites
Write-Host "=== Prerequisites Check ===" -ForegroundColor Cyan

# Check if running as Administrator (needed for some tests)
$isAdmin = ([Security.Principal.WindowsPrincipal] [Security.Principal.WindowsIdentity]::GetCurrent()).IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)
if ($isAdmin) {
    Write-Host "[OK] Running as Administrator" -ForegroundColor Green
} else {
    Write-Host "[WARN] Not running as Administrator - some tests may fail" -ForegroundColor Yellow
    Write-Host "       Recommend running: Start-Process powershell -Verb RunAs" -ForegroundColor Gray
}

# Check if agent token is set (for network tests)
if ($env:AGENT_TOKEN) {
    Write-Host "[OK] AGENT_TOKEN environment variable is set" -ForegroundColor Green
} else {
    Write-Host "[WARN] AGENT_TOKEN not set - network tests may fail" -ForegroundColor Yellow
    Write-Host "       Set with: `$env:AGENT_TOKEN = 'your-token-here'" -ForegroundColor Gray
}

Write-Host ""

# Test 1: Port Conflict Detection
if (-not $SkipPortTest) {
    Write-Host "================================================================" -ForegroundColor Cyan
    Write-Host "  TEST SUITE 1: Port Conflict Detection" -ForegroundColor Cyan
    Write-Host "================================================================" -ForegroundColor Cyan
    Write-Host ""
    
    $portTestScript = Join-Path $scriptPath "test_port_conflict.ps1"
    
    if (Test-Path $portTestScript) {
        try {
            & $portTestScript
            $portTestResult = $LASTEXITCODE
            
            if ($portTestResult -eq 0) {
                Write-Host "[PASS] Port Conflict Test Suite: PASSED" -ForegroundColor Green
                $totalPassed++
                $testResults += @{ Name = "Port Conflict Detection"; Status = "PASSED" }
            } else {
                Write-Host "[FAIL] Port Conflict Test Suite: FAILED" -ForegroundColor Red
                $totalFailed++
                $testResults += @{ Name = "Port Conflict Detection"; Status = "FAILED" }
            }
        } catch {
            Write-Host "[ERROR] Port Conflict Test Suite: ERROR" -ForegroundColor Red
            Write-Host "        $($_.Exception.Message)" -ForegroundColor Gray
            $totalFailed++
            $testResults += @{ Name = "Port Conflict Detection"; Status = "ERROR" }
        }
    } else {
        Write-Host "[FAIL] Port conflict test script not found: $portTestScript" -ForegroundColor Red
        $totalFailed++
        $testResults += @{ Name = "Port Conflict Detection"; Status = "NOT FOUND" }
    }
    
    Write-Host ""
} else {
    Write-Host "[SKIP] Skipping Port Conflict Detection tests" -ForegroundColor Gray
    Write-Host ""
}

# Test 2: Network Share Retry Logic
if (-not $SkipNetworkTest) {
    Write-Host "================================================================" -ForegroundColor Cyan
    Write-Host "  TEST SUITE 2: Network Share Retry Logic" -ForegroundColor Cyan
    Write-Host "================================================================" -ForegroundColor Cyan
    Write-Host ""
    
    $networkTestScript = Join-Path $scriptPath "test_network_retry.ps1"
    
    if (Test-Path $networkTestScript) {
        try {
            $params = @{}
            if ($SkipCertificateCheck) {
                $params['SkipCertificateCheck'] = $true
            }
            
            & $networkTestScript @params
            $networkTestResult = $LASTEXITCODE
            
            if ($networkTestResult -eq 0) {
                Write-Host "[PASS] Network Retry Test Suite: PASSED" -ForegroundColor Green
                $totalPassed++
                $testResults += @{ Name = "Network Share Retry Logic"; Status = "PASSED" }
            } else {
                Write-Host "[FAIL] Network Retry Test Suite: FAILED" -ForegroundColor Red
                $totalFailed++
                $testResults += @{ Name = "Network Share Retry Logic"; Status = "FAILED" }
            }
        } catch {
            Write-Host "[ERROR] Network Retry Test Suite: ERROR" -ForegroundColor Red
            Write-Host "        $($_.Exception.Message)" -ForegroundColor Gray
            $totalFailed++
            $testResults += @{ Name = "Network Share Retry Logic"; Status = "ERROR" }
        }
    } else {
        Write-Host "[FAIL] Network retry test script not found: $networkTestScript" -ForegroundColor Red
        $totalFailed++
        $testResults += @{ Name = "Network Share Retry Logic"; Status = "NOT FOUND" }
    }
    
    Write-Host ""
} else {
    Write-Host "[SKIP] Skipping Network Share Retry Logic tests" -ForegroundColor Gray
    Write-Host ""
}

# Final Summary
Write-Host "================================================================" -ForegroundColor Cyan
Write-Host "                    FINAL TEST SUMMARY                         " -ForegroundColor Cyan
Write-Host "================================================================" -ForegroundColor Cyan
Write-Host ""

$testResults | ForEach-Object {
    $statusColor = switch ($_.Status) {
        "PASSED" { "Green" }
        "FAILED" { "Red" }
        "ERROR" { "Red" }
        "NOT FOUND" { "Yellow" }
        default { "Gray" }
    }
    
    $statusSymbol = switch ($_.Status) {
        "PASSED" { "[PASS]" }
        "FAILED" { "[FAIL]" }
        "ERROR" { "[ERROR]" }
        "NOT FOUND" { "[NOTFOUND]" }
        default { "[?]" }
    }
    
    Write-Host "  $statusSymbol $($_.Name): " -NoNewline
    Write-Host $_.Status -ForegroundColor $statusColor
}

Write-Host ""
Write-Host "Test Suites Passed: $totalPassed" -ForegroundColor Green
Write-Host "Test Suites Failed: $totalFailed" -ForegroundColor $(if ($totalFailed -eq 0) { "Green" } else { "Red" })
Write-Host ""

if ($totalFailed -eq 0) {
    Write-Host "================================================================" -ForegroundColor Green
    Write-Host "  ALL ERROR HANDLING TESTS PASSED!                            " -ForegroundColor Green
    Write-Host "                                                               " -ForegroundColor Green
    Write-Host "  Your error handling implementations are working correctly!   " -ForegroundColor Green
    Write-Host "  - Port conflict detection: OK                                " -ForegroundColor Green
    Write-Host "  - Network retry logic: OK                                    " -ForegroundColor Green
    Write-Host "================================================================" -ForegroundColor Green
    exit 0
} else {
    Write-Host "================================================================" -ForegroundColor Red
    Write-Host "  SOME TESTS FAILED                                           " -ForegroundColor Red
    Write-Host "                                                               " -ForegroundColor Red
    Write-Host "  Review the detailed output above to diagnose issues.         " -ForegroundColor Red
    Write-Host "================================================================" -ForegroundColor Red
    exit 1
}
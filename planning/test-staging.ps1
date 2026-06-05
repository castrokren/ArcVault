# ArcVault Staging Environment Test Suite
# Tests all path authentication functionality in staging

param(
    [string]$CoordinatorUrl = "http://localhost:8081",
    [string]$AdminToken = "staging-test-token-abc123xyz"
)

$testsPassed = 0
$testsFailed = 0
$credentials = @()

function Write-TestHeader {
    param([string]$Title)
    Write-Host "`n" + ("=" * 60) -ForegroundColor Cyan
    Write-Host $Title -ForegroundColor Cyan
    Write-Host ("=" * 60) -ForegroundColor Cyan
}

function Test-Pass {
    param([string]$Message)
    Write-Host "✅ $Message" -ForegroundColor Green
    $script:testsPassed++
}

function Test-Fail {
    param([string]$Message)
    Write-Host "❌ $Message" -ForegroundColor Red
    $script:testsFailed++
}

# TEST 1: Coordinator Connectivity
Write-TestHeader "TEST 1: Coordinator Connectivity"

try {
    $response = Invoke-WebRequest "$CoordinatorUrl/api/version" -ErrorAction Stop
    if ($response.StatusCode -eq 200) {
        Test-Pass "Coordinator responding on $CoordinatorUrl"
    } else {
        Test-Fail "Unexpected status code: $($response.StatusCode)"
    }
} catch {
    Test-Fail "Cannot connect to coordinator: $_"
}

# TEST 2: Create Credential - SMB
Write-TestHeader "TEST 2: Create SMB Credential"

$smbCredBody = @{
    name = "staging-smb-$(Get-Random)"
    type = "SMB"
    data = @{
        host = "staging.local"
        username = "testuser"
        password = "testpass123"
    }
} | ConvertTo-Json

try {
    $response = Invoke-WebRequest `
        -Uri "$CoordinatorUrl/api/credential-profiles" `
        -Method POST `
        -Headers @{ Authorization = "Bearer $AdminToken" } `
        -Body $smbCredBody `
        -ContentType "application/json" `
        -ErrorAction Stop

    if ($response.StatusCode -eq 201) {
        $credData = $response.Content | ConvertFrom-Json
        $credentials += $credData.id
        Test-Pass "SMB credential created: $($credData.name)"
    } else {
        Test-Fail "Unexpected status: $($response.StatusCode)"
    }
} catch {
    Test-Fail "Failed to create SMB credential: $_"
}

# TEST 3: Create Credential - SSH
Write-TestHeader "TEST 3: Create SSH Credential"

$sshCredBody = @{
    name = "staging-ssh-$(Get-Random)"
    type = "SSH"
    data = @{
        key = "-----BEGIN OPENSSH PRIVATE KEY-----`nMIIEpAIBAAKCAQEA0z7T...`n-----END OPENSSH PRIVATE KEY-----"
    }
} | ConvertTo-Json

try {
    $response = Invoke-WebRequest `
        -Uri "$CoordinatorUrl/api/credential-profiles" `
        -Method POST `
        -Headers @{ Authorization = "Bearer $AdminToken" } `
        -Body $sshCredBody `
        -ContentType "application/json" `
        -ErrorAction Stop

    if ($response.StatusCode -eq 201) {
        $credData = $response.Content | ConvertFrom-Json
        $credentials += $credData.id
        Test-Pass "SSH credential created: $($credData.name)"
    } else {
        Test-Fail "Unexpected status: $($response.StatusCode)"
    }
} catch {
    Test-Fail "Failed to create SSH credential: $_"
}

# TEST 4: List Credentials
Write-TestHeader "TEST 4: List Credentials"

try {
    $response = Invoke-WebRequest `
        -Uri "$CoordinatorUrl/api/credential-profiles" `
        -Headers @{ Authorization = "Bearer $AdminToken" } `
        -ErrorAction Stop

    if ($response.StatusCode -eq 200) {
        $creds = $response.Content | ConvertFrom-Json
        if ($creds.Count -gt 0) {
            Test-Pass "Retrieved $($creds.Count) credential(s)"
            # Verify no encrypted data in response
            $hasEncryptedData = $creds | Where-Object { $_.encrypted_data -ne $null }
            if ($null -eq $hasEncryptedData) {
                Test-Pass "Credentials list does not expose encrypted data (security OK)"
            } else {
                Test-Fail "WARNING: encrypted_data found in list response"
            }
        } else {
            Test-Fail "No credentials returned"
        }
    } else {
        Test-Fail "Unexpected status: $($response.StatusCode)"
    }
} catch {
    Test-Fail "Failed to list credentials: $_"
}

# TEST 5: Delete Non-Referenced Credential
Write-TestHeader "TEST 5: Delete Credential (No References)"

if ($credentials.Count -gt 0) {
    $credToDelete = $credentials[0]
    try {
        $response = Invoke-WebRequest `
            -Uri "$CoordinatorUrl/api/credential-profiles/$credToDelete" `
            -Method DELETE `
            -Headers @{ Authorization = "Bearer $AdminToken" } `
            -ErrorAction Stop

        if ($response.StatusCode -eq 204) {
            Test-Pass "Credential deleted successfully"
            $credentials = $credentials | Where-Object { $_ -ne $credToDelete }
        } else {
            Test-Fail "Unexpected status: $($response.StatusCode)"
        }
    } catch {
        Test-Fail "Failed to delete credential: $_"
    }
} else {
    Test-Fail "No credentials available to test delete"
}

# TEST 6: Error Handling - Missing Auth
Write-TestHeader "TEST 6: Error Handling - Missing Auth"

try {
    $response = Invoke-WebRequest `
        -Uri "$CoordinatorUrl/api/credential-profiles" `
        -ErrorAction Stop
    Test-Fail "Request should fail without authorization"
} catch {
    if ($_.Exception.Response.StatusCode -eq 401) {
        Test-Pass "Correctly rejected unauthorized request (401)"
    } else {
        Test-Fail "Unexpected error: $($_.Exception.Response.StatusCode)"
    }
}

# TEST 7: Error Handling - Invalid Token
Write-TestHeader "TEST 7: Error Handling - Invalid Token"

try {
    $response = Invoke-WebRequest `
        -Uri "$CoordinatorUrl/api/credential-profiles" `
        -Headers @{ Authorization = "Bearer invalid-token" } `
        -ErrorAction Stop
    Test-Fail "Request should fail with invalid token"
} catch {
    if ($_.Exception.Response.StatusCode -eq 401 -or $_.Exception.Response.StatusCode -eq 400) {
        Test-Pass "Correctly rejected invalid token"
    } else {
        Test-Fail "Unexpected error: $($_.Exception.Response.StatusCode)"
    }
}

# SUMMARY
Write-TestHeader "STAGING TEST SUMMARY"
$totalTests = $testsPassed + $testsFailed
Write-Host "Passed: $testsPassed/$totalTests" -ForegroundColor Green
Write-Host "Failed: $testsFailed/$totalTests" -ForegroundColor $(if ($testsFailed -gt 0) { "Red" } else { "Green" })

if ($testsFailed -eq 0) {
    Write-Host "`n✅ All staging tests passed! Ready for production deployment." -ForegroundColor Green
    exit 0
} else {
    Write-Host "`n❌ Some tests failed. Review above and fix before production." -ForegroundColor Red
    exit 1
}

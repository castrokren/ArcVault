# Run tests and save output to file
$outputFile = "$PSScriptRoot\test_results.txt"
Write-Host "Running tests and saving output to: $outputFile"

# Capture all output
& "$PSScriptRoot\run_all_error_handling_tests.ps1" -SkipCertificateCheck *>&1 | Tee-Object -FilePath $outputFile

Write-Host "`n`nTest results have been saved to: $outputFile"
Write-Host "Exit code: $LASTEXITCODE"


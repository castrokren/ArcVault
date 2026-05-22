Set-Location C:\Projects\ArcVault2.0
go test ./coordinator/server/ -v 2>&1 | Tee-Object -FilePath test_output.txt
Write-Host "TEST EXECUTION COMPLETE"

Set-Location C:\Projects\ArcVault2.0

Write-Host "=== BUILD COORDINATOR ===" -ForegroundColor Cyan
go build ./coordinator/...
if ($LASTEXITCODE -ne 0) { Write-Host "COORDINATOR BUILD FAILED" -ForegroundColor Red; exit 1 }
Write-Host "Coordinator build OK" -ForegroundColor Green

Write-Host "=== BUILD AGENT ===" -ForegroundColor Cyan
go build ./agent/...
if ($LASTEXITCODE -ne 0) { Write-Host "AGENT BUILD FAILED" -ForegroundColor Red; exit 1 }
Write-Host "Agent build OK" -ForegroundColor Green

Write-Host "=== RUN TESTS ===" -ForegroundColor Cyan
go test ./... -count=1 -timeout 120s
if ($LASTEXITCODE -ne 0) { Write-Host "TESTS FAILED" -ForegroundColor Red; exit 1 }
Write-Host "ALL TESTS PASSED" -ForegroundColor Green

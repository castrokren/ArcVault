
$out = @()
$out += "=== SERVICE STATE ==="
$out += (sc.exe query arcvault-coordinator) -join "`n"
$out += ""
$out += "=== PORT 443 BINDING ==="
$out += (netstat -ano | findstr ":443") -join "`n"
$out += ""
$out += "=== COORDINATOR EXE ==="
$exePath = "C:\ArcVault\coordinator.exe"
if (Test-Path $exePath) {
    $out += "EXISTS: " + (Get-Item $exePath).LastWriteTime
    $out += "SIZE:   " + (Get-Item $exePath).Length + " bytes"
} else {
    $out += "MISSING: $exePath"
}
$out += ""
$out += "=== CONFIG.JSON ==="
$cfgPath = "C:\ArcVault\config.json"
if (Test-Path $cfgPath) {
    $out += Get-Content $cfgPath -Raw
} else {
    $out += "MISSING: $cfgPath"
}
$out += ""
$out += "=== CERT FILES ==="
$out += "cert.pem: " + (Test-Path "C:\ArcVault\cert.pem")
$out += "key.pem:  " + (Test-Path "C:\ArcVault\key.pem")
$out += ""
$out += "=== HEALTH CHECK ==="
[System.Net.ServicePointManager]::ServerCertificateValidationCallback = { $true }
try {
    $r = Invoke-RestMethod "https://localhost/health" -TimeoutSec 5
    $out += "https://localhost/health: " + ($r | ConvertTo-Json)
} catch {
    $out += "https://localhost/health FAILED: " + $_.Exception.Message
}
try {
    $r2 = Invoke-RestMethod "https://localhost:443/health" -TimeoutSec 5
    $out += "https://localhost:443/health: " + ($r2 | ConvertTo-Json)
} catch {
    $out += "https://localhost:443/health FAILED: " + $_.Exception.Message
}
$out += ""
$out += "=== EVENT LOG (last 5 arcvault entries) ==="
try {
    $evts = Get-WinEvent -LogName Application -MaxEvents 100 -ErrorAction SilentlyContinue |
        Where-Object { $_.ProviderName -match "arcvault" } |
        Select-Object -First 5
    foreach ($e in $evts) { $out += $e.TimeCreated.ToString() + " [" + $e.LevelDisplayName + "] " + $e.Message }
} catch { $out += "Event log error: $_" }
$out += ""
$out += "=== DONE ==="
$logPath = "C:\Projects\ArcVault2.0\diagnose_output.txt"
$out | Out-File $logPath -Encoding UTF8
Write-Host "Output written to $logPath"
$out | ForEach-Object { Write-Host $_ }

#Requires -Version 5.0
param(
    [string]$SourcePath = "C:\Projects\ArcVault2.0",
    [string]$Tag = "v1.0.0"
)

$ErrorActionPreference = "Stop"

Write-Host ""
Write-Host " ArcVault - Build and Upload $Tag Assets" -ForegroundColor Cyan
Write-Host " ==========================================" -ForegroundColor Cyan
Write-Host ""

# -- Validate source repo ------------------------------------------------------

if (-not (Test-Path "$SourcePath\go.mod")) {
    Write-Host " ERROR: go.mod not found in: $SourcePath" -ForegroundColor Red
    exit 1
}

# -- Preflight checks ----------------------------------------------------------

if (-not (Get-Command go -ErrorAction SilentlyContinue)) {
    Write-Host " ERROR: 'go' not found. Install from https://go.dev/dl/" -ForegroundColor Red
    exit 1
}
if (-not (Get-Command gh -ErrorAction SilentlyContinue)) {
    Write-Host " ERROR: GitHub CLI not found." -ForegroundColor Red
    Write-Host " Install: winget install GitHub.cli  then run: gh auth login" -ForegroundColor DarkGray
    exit 1
}

Write-Host " Source repo : $SourcePath"
Write-Host " Release tag : $Tag"
Write-Host " Target      : Windows 64-bit (amd64)"
Write-Host ""

# -- Build ---------------------------------------------------------------------

$DistDir = Join-Path $SourcePath "dist-release"
New-Item -ItemType Directory -Force -Path $DistDir | Out-Null

Set-Location $SourcePath

$env:GOOS   = "windows"
$env:GOARCH = "amd64"

Write-Host "[1/4]  Building binaries..." -ForegroundColor Yellow

Write-Host "   Building coordinator..." -ForegroundColor DarkGray
& go build -ldflags="-s -w" -o "$DistDir\coordinator-windows-amd64.exe" .\coordinator
if ($LASTEXITCODE -ne 0) {
    Write-Host " ERROR: coordinator build failed." -ForegroundColor Red
    exit 1
}
Write-Host "   OK  coordinator-windows-amd64.exe" -ForegroundColor Green

Write-Host "   Building agent..." -ForegroundColor DarkGray
& go build -ldflags="-s -w" -o "$DistDir\agent-windows-amd64.exe" .\agent
if ($LASTEXITCODE -ne 0) {
    Write-Host " ERROR: agent build failed." -ForegroundColor Red
    exit 1
}
Write-Host "   OK  agent-windows-amd64.exe" -ForegroundColor Green

Remove-Item Env:\GOOS, Env:\GOARCH -ErrorAction SilentlyContinue

# -- Generate checksums --------------------------------------------------
Write-Host ""
Write-Host "[2/4]  Generating SHA256SUMS..." -ForegroundColor Yellow

$checksumFile = Join-Path $DistDir "SHA256SUMS"
Get-ChildItem $DistDir -Filter "*.exe" | ForEach-Object {
    $hash = (Get-FileHash -Algorithm SHA256 $_.FullName).Hash.ToLower()
    "$hash  $($_.Name)" | Out-File -FilePath $checksumFile -Append -Encoding ascii
}
Write-Host "   OK  SHA256SUMS generated" -ForegroundColor Green

# -- Upload --------------------------------------------------------------------

Write-Host ""
Write-Host "[3/4]  Uploading to GitHub Release $Tag..." -ForegroundColor Yellow

$files = Get-ChildItem $DistDir -Filter "*" | Select-Object -ExpandProperty FullName

foreach ($f in $files) {
    Write-Host "   Uploading $(Split-Path $f -Leaf)..." -ForegroundColor DarkGray
    & gh release upload $Tag $f --clobber
    if ($LASTEXITCODE -ne 0) {
        Write-Host " ERROR: Upload failed. Run 'gh auth login' if not authenticated." -ForegroundColor Red
        exit 1
    }
}

Remove-Item -Recurse -Force $DistDir

# -- Done ----------------------------------------------------------------------

Write-Host ""
Write-Host "[4/4]  Done!" -ForegroundColor Green
Write-Host ""
Write-Host "  https://github.com/castrokren/ArcVault/releases/tag/$Tag" -ForegroundColor Cyan
Write-Host "  The installer will now download successfully." -ForegroundColor White
Write-Host ""

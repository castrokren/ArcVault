param(
    [string]$ConfigDir = 'C:\ProgramData\BackupController',
    [int]$Port = 8501,
    [string]$Address = '0.0.0.0',
    [string]$Issuer = 'backup-controller'
)

$ErrorActionPreference = 'Stop'

# Ensure config dir exists
New-Item -ItemType Directory -Force -Path $ConfigDir | Out-Null
$priv = Join-Path $ConfigDir 'controller_private.pem'
$pub = Join-Path $ConfigDir 'controller_public.pem'

function Write-KeyPairPython {
@"
from cryptography.hazmat.primitives.asymmetric import rsa
from cryptography.hazmat.primitives import serialization
from pathlib import Path
import sys
d = Path(r"$ConfigDir")
priv = d / "controller_private.pem"
pub = d / "controller_public.pem"
if not priv.exists() or not pub.exists():
    key = rsa.generate_private_key(public_exponent=65537, key_size=3072)
    priv.write_bytes(key.private_bytes(encoding=serialization.Encoding.PEM,
        format=serialization.PrivateFormat.TraditionalOpenSSL,
        encryption_algorithm=serialization.NoEncryption()))
    pub.write_bytes(key.public_key().public_bytes(encoding=serialization.Encoding.PEM,
        format=serialization.PublicFormat.SubjectPublicKeyInfo))
print(str(priv))
print(str(pub))
"@
}

# Generate keypair if missing (Python-based, no OpenSSL dependency)
if (-not (Test-Path $priv) -or -not (Test-Path $pub)) {
  $py = Write-KeyPairPython
  $tempPy = Join-Path $env:TEMP 'write_keys.py'
  Set-Content -Path $tempPy -Value $py -Encoding ASCII
  & python $tempPy | Out-Null
  Remove-Item $tempPy -ErrorAction SilentlyContinue
}

# Lock down private key ACLs
icacls $priv /inheritance:r | Out-Null
icacls $priv /grant:r "Administrators:F" "SYSTEM:F" | Out-Null

# Export environment variables for controller runtime
$env:CTRL_JWT_PRIVATE_KEY_PATH = $priv
$env:CTRL_JWT_ISSUER = $Issuer
$env:CONTROLLER_PORT = "$Port"
$env:CONTROLLER_ADDRESS = $Address

Write-Host "Controller installed/initialized."
Write-Host "Private key: $priv"
Write-Host "Public key:  $pub"
Write-Host "Issuer:      $Issuer"
Write-Host "Address:     ${Address}:$Port"
Write-Host "Set agents to use the public key above."



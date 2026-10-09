#!/usr/bin/env python3
"""
Generate self-signed SSL certificates for ArcVault.

This script creates self-signed certificates for HTTPS communication
without requiring external certificate authorities.

Usage:
    python scripts/generate_self_signed_cert.py [output_directory] [hostname]
    
Default output directory: keys/
Default hostname: localhost
"""

import os
import sys
import argparse
from pathlib import Path
from datetime import datetime, timedelta
from cryptography import x509
from cryptography.x509.oid import NameOID, ExtendedKeyUsageOID
from cryptography.hazmat.primitives import hashes, serialization
from cryptography.hazmat.primitives.asymmetric import rsa
from cryptography.hazmat.backends import default_backend


def generate_private_key(key_size: int = 2048):
    """Generate RSA private key."""
    return rsa.generate_private_key(
        public_exponent=65537,
        key_size=key_size,
        backend=default_backend()
    )


def create_self_signed_certificate(private_key, hostname: str, days_valid: int = 365):
    """Create a self-signed certificate."""
    subject = issuer = x509.Name([
        x509.NameAttribute(NameOID.COUNTRY_NAME, "US"),
        x509.NameAttribute(NameOID.STATE_OR_PROVINCE_NAME, "New York"),
        x509.NameAttribute(NameOID.LOCALITY_NAME, "New York"),
        x509.NameAttribute(NameOID.ORGANIZATION_NAME, "ArcVault"),
        x509.NameAttribute(NameOID.COMMON_NAME, hostname),
    ])
    
    cert = x509.CertificateBuilder().subject_name(
        subject
    ).issuer_name(
        issuer
    ).public_key(
        private_key.public_key()
    ).serial_number(
        x509.random_serial_number()
    ).not_valid_before(
        datetime.utcnow()
    ).not_valid_after(
        datetime.utcnow() + timedelta(days=days_valid)
    ).add_extension(
        x509.SubjectAlternativeName([
            x509.DNSName(hostname),
            x509.DNSName("localhost"),
            x509.IPAddress(ipaddress.IPv4Address("127.0.0.1")),
            x509.IPAddress(ipaddress.IPv6Address("::1")),
        ]),
        critical=False,
    ).add_extension(
        x509.KeyUsage(
            key_cert_sign=False,
            crl_sign=False,
            digital_signature=True,
            content_commitment=False,
            key_encipherment=True,
            data_encipherment=False,
            key_agreement=False,
            encipher_only=False,
            decipher_only=False,
        ),
        critical=True,
    ).add_extension(
        x509.ExtendedKeyUsage([
            ExtendedKeyUsageOID.SERVER_AUTH,
            ExtendedKeyUsageOID.CLIENT_AUTH,
        ]),
        critical=True,
    ).sign(private_key, hashes.SHA256(), default_backend())
    
    return cert


def save_private_key(private_key, filepath: Path):
    """Save private key in PEM format."""
    pem = private_key.private_bytes(
        encoding=serialization.Encoding.PEM,
        format=serialization.PrivateFormat.PKCS8,
        encryption_algorithm=serialization.NoEncryption()
    )
    with open(filepath, 'wb') as f:
        f.write(pem)
    
    # Set restrictive permissions (owner read/write only)
    os.chmod(filepath, 0o600)


def save_certificate(certificate, filepath: Path):
    """Save certificate in PEM format."""
    pem = certificate.public_bytes(serialization.Encoding.PEM)
    with open(filepath, 'wb') as f:
        f.write(pem)
    
    # Set readable permissions
    os.chmod(filepath, 0o644)


def create_ca_bundle(certificate, filepath: Path):
    """Create a CA bundle with the self-signed certificate."""
    # For self-signed certificates, the CA bundle is just the certificate itself
    save_certificate(certificate, filepath)


def main():
    parser = argparse.ArgumentParser(description='Generate self-signed SSL certificates for ArcVault')
    parser.add_argument('output_dir', nargs='?', default='keys', 
                       help='Output directory for certificates (default: keys/)')
    parser.add_argument('hostname', nargs='?', default='localhost',
                       help='Hostname for the certificate (default: localhost)')
    parser.add_argument('--key-size', type=int, default=2048,
                       help='RSA key size in bits (default: 2048)')
    parser.add_argument('--days', type=int, default=365,
                       help='Certificate validity in days (default: 365)')
    parser.add_argument('--force', action='store_true',
                       help='Overwrite existing certificates')
    
    args = parser.parse_args()
    
    # Create output directory
    output_dir = Path(args.output_dir)
    output_dir.mkdir(exist_ok=True)
    
    # Certificate file paths
    private_key_path = output_dir / 'server.key'
    certificate_path = output_dir / 'server.crt'
    ca_bundle_path = output_dir / 'ca-bundle.crt'
    
    # Check if certificates already exist
    if not args.force and (private_key_path.exists() or certificate_path.exists()):
        print(f"Error: Certificates already exist in {output_dir}")
        print("Use --force to overwrite existing certificates")
        sys.exit(1)
    
    try:
        print(f"Generating self-signed certificate for {args.hostname}...")
        print(f"Key size: {args.key_size} bits")
        print(f"Validity: {args.days} days")
        
        # Generate private key
        private_key = generate_private_key(args.key_size)
        
        # Create self-signed certificate
        certificate = create_self_signed_certificate(
            private_key, 
            args.hostname, 
            args.days
        )
        
        # Save files
        print(f"Saving private key to {private_key_path}")
        save_private_key(private_key, private_key_path)
        
        print(f"Saving certificate to {certificate_path}")
        save_certificate(certificate, certificate_path)
        
        print(f"Saving CA bundle to {ca_bundle_path}")
        create_ca_bundle(certificate, ca_bundle_path)
        
        print("\n✅ Self-signed certificate generated successfully!")
        print(f"\n📁 Certificates saved to: {output_dir.absolute()}")
        print(f"🔐 Private key: {private_key_path.name}")
        print(f"🔓 Certificate: {certificate_path.name}")
        print(f"📜 CA bundle: {ca_bundle_path.name}")
        
        print(f"\n📋 Certificate details:")
        print(f"   Subject: {certificate.subject}")
        print(f"   Issuer: {certificate.issuer}")
        print(f"   Valid from: {certificate.not_valid_before}")
        print(f"   Valid until: {certificate.not_valid_after}")
        print(f"   Serial number: {certificate.serial_number}")
        
        print(f"\n🔍 Certificate fingerprint:")
        try:
            fingerprint = certificate.fingerprint(hashes.SHA256())
            print(f"   SHA-256: {fingerprint.hex()}")
        except Exception:
            pass
        
        print(f"\n⚠️  Security Notes:")
        print(f"   - This is a self-signed certificate")
        print(f"   - Browsers will show security warnings")
        print(f"   - Suitable for internal/development use")
        print(f"   - For production, consider using a trusted CA")
        
        print(f"\n📋 Next steps:")
        print(f"1. Update your .env files with certificate paths")
        print(f"2. Set AGENT_CERTFILE={certificate_path}")
        print(f"3. Set AGENT_KEYFILE={private_key_path}")
        print(f"4. Set AGENT_CA_BUNDLE={ca_bundle_path}")
        print(f"5. Set AGENT_VERIFY_SSL=true")
        
    except Exception as e:
        print(f"Error generating certificates: {e}")
        sys.exit(1)


if __name__ == '__main__':
    main()

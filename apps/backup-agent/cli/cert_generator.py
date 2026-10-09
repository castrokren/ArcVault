"""
SSL Certificate Generator - CLI Tool

Generate self-signed SSL certificates from command line.
"""

import argparse
import sys
from pathlib import Path
from datetime import datetime, timedelta

try:
    from cryptography import x509
    from cryptography.x509.oid import NameOID
    from cryptography.hazmat.primitives import hashes, serialization
    from cryptography.hazmat.primitives.asymmetric import rsa
    import ipaddress
except ImportError:
    print("ERROR: cryptography library not installed")
    print("Install with: pip install cryptography")
    sys.exit(1)


def generate_certificate(output_dir, hostname, key_size, days, country, org, force=False):
    """Generate SSL certificate"""
    try:
        print(f"Generating SSL certificate...")
        print(f"  Hostname: {hostname}")
        print(f"  Key Size: {key_size}")
        print(f"  Valid for: {days} days")
        print(f"  Organization: {org} ({country})")
        print()
        
        # Create output directory
        output_dir = Path(output_dir)
        output_dir.mkdir(parents=True, exist_ok=True)
        
        # Check for existing files
        cert_file = output_dir / "server.crt"
        key_file = output_dir / "server.key"
        ca_file = output_dir / "ca-bundle.crt"
        
        if not force and any(f.exists() for f in [cert_file, key_file, ca_file]):
            print("ERROR: Certificate files already exist!")
            print("Use --force to overwrite")
            return False
        
        print("Generating RSA key pair...")
        
        # Generate private key
        private_key = rsa.generate_private_key(
            public_exponent=65537,
            key_size=key_size,
        )
        
        print("Creating certificate...")
        
        # Create certificate
        subject = issuer = x509.Name([
            x509.NameAttribute(NameOID.COUNTRY_NAME, country),
            x509.NameAttribute(NameOID.ORGANIZATION_NAME, org),
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
            datetime.utcnow() + timedelta(days=days)
        ).add_extension(
            x509.SubjectAlternativeName([
                x509.DNSName(hostname),
                x509.DNSName("localhost"),
                x509.IPAddress(ipaddress.IPv4Address("127.0.0.1")),
            ]),
            critical=False,
        ).sign(private_key, hashes.SHA256())
        
        print("Saving certificate files...")
        
        # Save private key
        with open(key_file, "wb") as f:
            f.write(private_key.private_bytes(
                encoding=serialization.Encoding.PEM,
                format=serialization.PrivateFormat.PKCS8,
                encryption_algorithm=serialization.NoEncryption()
            ))
        
        # Save certificate
        with open(cert_file, "wb") as f:
            f.write(cert.public_bytes(serialization.Encoding.PEM))
        
        # Create CA bundle (same as cert for self-signed)
        with open(ca_file, "wb") as f:
            f.write(cert.public_bytes(serialization.Encoding.PEM))
        
        print()
        print("SUCCESS! SSL certificate generated:")
        print(f"  Output directory: {output_dir.absolute()}")
        print(f"  Private key: {key_file.name}")
        print(f"  Certificate: {cert_file.name}")
        print(f"  CA bundle: {ca_file.name}")
        print()
        
        return True
        
    except Exception as e:
        print(f"ERROR: {e}")
        return False


def main():
    """Main entry point for CLI"""
    parser = argparse.ArgumentParser(description="Generate self-signed SSL certificates")
    
    parser.add_argument("--output", "-o", default="keys",
                       help="Output directory (default: keys)")
    parser.add_argument("--hostname", default="localhost",
                       help="Hostname for certificate (default: localhost)")
    parser.add_argument("--key-size", type=int, default=2048, choices=[2048, 3072, 4096],
                       help="RSA key size (default: 2048)")
    parser.add_argument("--days", type=int, default=365,
                       help="Certificate validity in days (default: 365)")
    parser.add_argument("--country", default="US",
                       help="Country code, 2 letters (default: US)")
    parser.add_argument("--org", default="ArcVault",
                       help="Organization name (default: ArcVault)")
    parser.add_argument("--force", "-f", action="store_true",
                       help="Overwrite existing certificates")
    
    args = parser.parse_args()
    
    if len(args.country) != 2:
        print("ERROR: Country code must be exactly 2 letters")
        sys.exit(1)
    
    success = generate_certificate(
        args.output, args.hostname, args.key_size, args.days,
        args.country.upper(), args.org, args.force
    )
    
    sys.exit(0 if success else 1)


if __name__ == "__main__":
    main()


#!/usr/bin/env python3
"""
Generate JWT RSA key pairs for ArcVault authentication.

This script creates:
- controller_private.pem (for the controller to sign JWTs)
- controller_public.pem (for agents to verify JWTs)

Usage:
    python scripts/generate_jwt_keys.py [output_directory]
    
Default output directory: keys/
"""

import os
import sys
import argparse
from pathlib import Path
from cryptography.hazmat.primitives import serialization
from cryptography.hazmat.primitives.asymmetric import rsa
from cryptography.hazmat.backends import default_backend


def generate_rsa_keypair(key_size: int = 2048):
    """Generate RSA key pair for JWT signing/verification."""
    private_key = rsa.generate_private_key(
        public_exponent=65537,
        key_size=key_size,
        backend=default_backend()
    )
    public_key = private_key.public_key()
    return private_key, public_key


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


def save_public_key(public_key, filepath: Path):
    """Save public key in PEM format."""
    pem = public_key.public_bytes(
        encoding=serialization.Encoding.PEM,
        format=serialization.PublicFormat.SubjectPublicKeyInfo
    )
    with open(filepath, 'wb') as f:
        f.write(pem)
    
    # Set readable permissions
    os.chmod(filepath, 0o644)


def main():
    parser = argparse.ArgumentParser(description='Generate JWT RSA key pairs for ArcVault')
    parser.add_argument('output_dir', nargs='?', default='keys', 
                       help='Output directory for keys (default: keys/)')
    parser.add_argument('--key-size', type=int, default=2048,
                       help='RSA key size in bits (default: 2048)')
    parser.add_argument('--force', action='store_true',
                       help='Overwrite existing keys')
    
    args = parser.parse_args()
    
    # Create output directory
    output_dir = Path(args.output_dir)
    output_dir.mkdir(exist_ok=True)
    
    # Key file paths
    private_key_path = output_dir / 'controller_private.pem'
    public_key_path = output_dir / 'controller_public.pem'
    
    # Check if keys already exist
    if not args.force and (private_key_path.exists() or public_key_path.exists()):
        print(f"Error: Keys already exist in {output_dir}")
        print("Use --force to overwrite existing keys")
        sys.exit(1)
    
    try:
        print(f"Generating {args.key_size}-bit RSA key pair...")
        private_key, public_key = generate_rsa_keypair(args.key_size)
        
        print(f"Saving private key to {private_key_path}")
        save_private_key(private_key, private_key_path)
        
        print(f"Saving public key to {public_key_path}")
        save_public_key(public_key, public_key_path)
        
        print("\n✅ JWT key pair generated successfully!")
        print(f"\n📁 Keys saved to: {output_dir.absolute()}")
        print(f"🔐 Private key: {private_key_path.name} (keep secure!)")
        print(f"🔓 Public key: {public_key_path.name} (can be shared)")
        
        print("\n📋 Next steps:")
        print("1. Copy controller_public.pem to all agent machines")
        print("2. Keep controller_private.pem secure on controller machine")
        print("3. Update your .env files with the key paths")
        print("4. Set AGENT_REQUIRE_JWT=true in production")
        
        # Show key fingerprints for verification
        print(f"\n🔍 Key fingerprints:")
        try:
            from cryptography.hazmat.primitives import hashes
            private_fingerprint = private_key.private_bytes(
                encoding=serialization.Encoding.DER,
                format=serialization.PrivateFormat.PKCS8,
                encryption_algorithm=serialization.NoEncryption()
            )
            public_fingerprint = public_key.public_bytes(
                encoding=serialization.Encoding.DER,
                format=serialization.PublicFormat.SubjectPublicKeyInfo
            )
            
            import hashlib
            private_hash = hashlib.sha256(private_fingerprint).hexdigest()[:16]
            public_hash = hashlib.sha256(public_fingerprint).hexdigest()[:16]
            
            print(f"   Private key: {private_hash}")
            print(f"   Public key:  {public_hash}")
        except Exception:
            pass
            
    except Exception as e:
        print(f"Error generating keys: {e}")
        sys.exit(1)


if __name__ == '__main__':
    main()

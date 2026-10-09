"""
JWT Key Generator - CLI Tool

Generate RSA key pairs for JWT authentication from command line.
"""

import argparse
import sys
from pathlib import Path

try:
    from cryptography.hazmat.primitives import serialization
    from cryptography.hazmat.primitives.asymmetric import rsa
except ImportError:
    print("ERROR: cryptography library not installed")
    print("Install with: pip install cryptography")
    sys.exit(1)


def generate_jwt_keys(output_dir, key_size, force=False):
    """Generate JWT key pair"""
    try:
        print(f"Generating JWT key pair...")
        print(f"  Key Size: {key_size}")
        print()
        
        # Create output directory
        output_dir = Path(output_dir)
        output_dir.mkdir(parents=True, exist_ok=True)
        
        # Check for existing files
        private_file = output_dir / "controller_private.pem"
        public_file = output_dir / "controller_public.pem"
        
        if not force and any(f.exists() for f in [private_file, public_file]):
            print("ERROR: JWT key files already exist!")
            print("Use --force to overwrite")
            return False
        
        print("Generating RSA key pair...")
        
        # Generate private key
        private_key = rsa.generate_private_key(
            public_exponent=65537,
            key_size=key_size,
        )
        
        # Get public key
        public_key = private_key.public_key()
        
        print("Saving key files...")
        
        # Save private key
        with open(private_file, "wb") as f:
            f.write(private_key.private_bytes(
                encoding=serialization.Encoding.PEM,
                format=serialization.PrivateFormat.PKCS8,
                encryption_algorithm=serialization.NoEncryption()
            ))
        
        # Save public key
        with open(public_file, "wb") as f:
            f.write(public_key.public_bytes(
                encoding=serialization.Encoding.PEM,
                format=serialization.PublicFormat.SubjectPublicKeyInfo
            ))
        
        print()
        print("SUCCESS! JWT key pair generated:")
        print(f"  Output directory: {output_dir.absolute()}")
        print(f"  Private key: {private_file.name} (keep secure!)")
        print(f"  Public key: {public_file.name} (distribute to clients)")
        print()
        print("Next steps:")
        print("  1. Keep the private key secure on your server")
        print("  2. Copy the public key to all client machines")
        print("  3. Configure server to use the private key")
        print("  4. Configure clients to use the public key")
        print()
        
        return True
        
    except Exception as e:
        print(f"ERROR: {e}")
        return False


def main():
    """Main entry point for CLI"""
    parser = argparse.ArgumentParser(description="Generate RSA key pairs for JWT authentication")
    
    parser.add_argument("--output", "-o", default="keys",
                       help="Output directory (default: keys)")
    parser.add_argument("--key-size", type=int, default=2048, choices=[2048, 3072, 4096],
                       help="RSA key size (default: 2048)")
    parser.add_argument("--force", "-f", action="store_true",
                       help="Overwrite existing keys")
    
    args = parser.parse_args()
    
    success = generate_jwt_keys(args.output, args.key_size, args.force)
    
    sys.exit(0 if success else 1)


if __name__ == "__main__":
    main()


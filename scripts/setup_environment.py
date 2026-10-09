#!/usr/bin/env python3
"""
ArcVault Environment Setup Script

This script helps you set up the proper environment configuration for ArcVault.
It can:
- Copy environment templates
- Generate JWT keys
- Validate configuration
- Create directory structure

Usage:
    python scripts/setup_environment.py [command] [options]
    
Commands:
    init [env]     - Initialize environment (dev/staging/prod)
    keys           - Generate JWT key pairs
    validate       - Validate current configuration
    help           - Show this help
"""

import os
import sys
import shutil
import argparse
from pathlib import Path
from typing import Optional


class ArcVaultSetup:
    def __init__(self, project_root: Optional[Path] = None):
        self.project_root = project_root or Path(__file__).parent.parent
        self.templates_dir = self.project_root / 'env.templates'
        self.keys_dir = self.project_root / 'keys'
        
    def init_environment(self, env_type: str = 'development'):
        """Initialize environment configuration."""
        env_types = ['development', 'staging', 'production']
        if env_type not in env_types:
            print(f"Error: Environment must be one of: {', '.join(env_types)}")
            return False
            
        template_file = self.templates_dir / f'{env_type}.env'
        if not template_file.exists():
            print(f"Error: Template file not found: {template_file}")
            return False
            
        env_file = self.project_root / '.env'
        
        if env_file.exists():
            response = input(f".env already exists. Overwrite? (y/N): ")
            if response.lower() != 'y':
                print("Setup cancelled.")
                return False
                
        try:
            shutil.copy2(template_file, env_file)
            print(f"✅ Environment initialized for {env_type}")
            print(f"📁 Configuration copied to: {env_file}")
            
            # Create keys directory if it doesn't exist
            self.keys_dir.mkdir(exist_ok=True)
            
            print(f"\n📋 Next steps:")
            if env_type == 'development':
                print("1. Review .env file and adjust settings as needed")
                print("2. Run 'python scripts/setup_environment.py keys' to generate JWT keys (optional)")
                print("3. Start the agent: cd apps/backup-agent && python -m agent.app")
                print("4. Start the controller: cd apps/backup-control && streamlit run controller/app.py")
            else:
                print("1. Review .env file and update hostnames/URLs")
                print("2. Run 'python scripts/setup_environment.py keys' to generate JWT keys")
                print("3. Set up SSL certificates in keys/ directory")
                print("4. Deploy using the build scripts")
                
            return True
            
        except Exception as e:
            print(f"Error initializing environment: {e}")
            return False
    
    def generate_keys(self):
        """Generate JWT key pairs."""
        try:
            # Import and run the key generation script
            key_script = self.project_root / 'scripts' / 'generate_jwt_keys.py'
            if not key_script.exists():
                print(f"Error: Key generation script not found: {key_script}")
                return False
                
            # Run the key generation script
            import subprocess
            result = subprocess.run([
                sys.executable, str(key_script), str(self.keys_dir)
            ], capture_output=True, text=True)
            
            if result.returncode == 0:
                print(result.stdout)
                return True
            else:
                print(f"Error generating keys: {result.stderr}")
                return False
                
        except Exception as e:
            print(f"Error running key generation: {e}")
            return False
    
    def validate_configuration(self):
        """Validate current configuration."""
        print("🔍 Validating ArcVault configuration...")
        
        issues = []
        
        # Check for .env file
        env_file = self.project_root / '.env'
        if not env_file.exists():
            issues.append("❌ .env file not found - run 'init' command first")
        else:
            print("✅ .env file found")
            
        # Check for keys directory
        if not self.keys_dir.exists():
            issues.append("❌ keys/ directory not found")
        else:
            print("✅ keys/ directory found")
            
        # Check for JWT keys
        private_key = self.keys_dir / 'controller_private.pem'
        public_key = self.keys_dir / 'controller_public.pem'
        
        if not private_key.exists():
            issues.append("❌ JWT private key not found - run 'keys' command")
        else:
            print("✅ JWT private key found")
            
        if not public_key.exists():
            issues.append("❌ JWT public key not found - run 'keys' command")
        else:
            print("✅ JWT public key found")
            
        # Check environment variables
        if env_file.exists():
            self._validate_env_file(env_file, issues)
            
        # Summary
        if issues:
            print(f"\n⚠️  Found {len(issues)} issues:")
            for issue in issues:
                print(f"   {issue}")
            return False
        else:
            print("\n✅ Configuration looks good!")
            return True
    
    def _validate_env_file(self, env_file: Path, issues: list):
        """Validate .env file contents."""
        try:
            with open(env_file, 'r') as f:
                content = f.read()
                
            # Check for required variables
            required_vars = [
                'AGENT_LISTEN_HOST',
                'AGENT_PORT', 
                'CONTROLLER_PORT',
                'AGENT_TOKEN'
            ]
            
            for var in required_vars:
                if f'{var}=' not in content:
                    issues.append(f"❌ Missing required variable: {var}")
                    
            # Check for JWT configuration
            if 'AGENT_REQUIRE_JWT=true' in content:
                if 'AGENT_JWT_PUBLIC_KEY_PATH=' not in content:
                    issues.append("❌ JWT required but public key path not set")
                if 'CTRL_JWT_PRIVATE_KEY_PATH=' not in content:
                    issues.append("❌ JWT required but private key path not set")
                    
        except Exception as e:
            issues.append(f"❌ Error reading .env file: {e}")
    
    def show_help(self):
        """Show detailed help information."""
        print("""
🔧 ArcVault Environment Setup

This script helps you configure ArcVault for different environments.

COMMANDS:
  init [env]     Initialize environment configuration
                 Options: development, staging, production
                 
  keys           Generate JWT RSA key pairs for secure authentication
                 
  validate       Check current configuration for issues
                 
  help           Show this help message

EXAMPLES:
  # Set up for development
  python scripts/setup_environment.py init development
  
  # Set up for production
  python scripts/setup_environment.py init production
  python scripts/setup_environment.py keys
  
  # Check configuration
  python scripts/setup_environment.py validate

ENVIRONMENT TYPES:
  development    - Local development with simple token auth
  staging        - Test environment with JWT auth (optional SSL)
  production     - Production with JWT auth and SSL required

For more information, see the README.md file.
        """)


def main():
    parser = argparse.ArgumentParser(description='ArcVault Environment Setup')
    parser.add_argument('command', nargs='?', default='help',
                       choices=['init', 'keys', 'validate', 'help'],
                       help='Command to run')
    parser.add_argument('env_type', nargs='?', default='development',
                       choices=['development', 'staging', 'production'],
                       help='Environment type for init command')
    
    args = parser.parse_args()
    
    setup = ArcVaultSetup()
    
    if args.command == 'init':
        success = setup.init_environment(args.env_type)
        sys.exit(0 if success else 1)
        
    elif args.command == 'keys':
        success = setup.generate_keys()
        sys.exit(0 if success else 1)
        
    elif args.command == 'validate':
        success = setup.validate_configuration()
        sys.exit(0 if success else 1)
        
    elif args.command == 'help':
        setup.show_help()
        sys.exit(0)


if __name__ == '__main__':
    main()

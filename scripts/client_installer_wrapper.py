#!/usr/bin/env python3
"""
Wrapper script for ArcVault Client Installer.
This script extracts the client package and runs the installation.
"""

import os
import sys
import tempfile
import shutil
import subprocess
from pathlib import Path


def get_resource_path(relative_path):
    """Get the path to a resource file."""
    try:
        # PyInstaller creates a temp folder and stores path in _MEIPASS
        base_path = sys._MEIPASS
    except Exception:
        base_path = os.path.abspath(".")
    
    return os.path.join(base_path, relative_path)


def main():
    print("ArcVault Client Installer")
    print("========================")
    print()
    
    # Check for administrator privileges
    try:
        import ctypes
        if not ctypes.windll.shell32.IsUserAnAdmin():
            print("ERROR: This installer requires administrator privileges.")
            print("Please run as Administrator.")
            input("Press Enter to exit...")
            sys.exit(1)
    except Exception:
        print("WARNING: Could not verify administrator privileges.")
        print("Please ensure you are running as Administrator.")
        input("Press Enter to continue...")
    
    # Extract client package
    print("Extracting client package...")
    
    try:
        # Get the path to the embedded client package
        client_package_path = get_resource_path("deployment-client")
        
        if not os.path.exists(client_package_path):
            print("ERROR: Client package not found in installer.")
            input("Press Enter to exit...")
            sys.exit(1)
        
        # Create temporary directory for extraction
        temp_dir = tempfile.mkdtemp(prefix="arcvault_client_")
        print(f"Extracting to: {temp_dir}")
        
        # Copy client package to temp directory
        shutil.copytree(client_package_path, os.path.join(temp_dir, "deployment-client"))
        
        # Change to the extracted directory
        extracted_path = os.path.join(temp_dir, "deployment-client")
        os.chdir(extracted_path)
        
        print("Client package extracted successfully!")
        print()
        
        # Run the client installer
        print("Starting ArcVault Client Setup...")
        print()
        
        # Run the main client launcher
        launcher_script = "ArcVault-Agent.bat"
        if os.path.exists(launcher_script):
            subprocess.run([launcher_script], shell=True)
        else:
            print("ERROR: Client launcher not found.")
            input("Press Enter to exit...")
            sys.exit(1)
        
    except Exception as e:
        print(f"ERROR: Failed to extract client package: {e}")
        input("Press Enter to exit...")
        sys.exit(1)
    
    finally:
        # Clean up temporary directory
        try:
            if 'temp_dir' in locals() and os.path.exists(temp_dir):
                shutil.rmtree(temp_dir)
        except Exception:
            pass


if __name__ == '__main__':
    main()

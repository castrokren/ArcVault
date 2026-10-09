#!/usr/bin/env python3
"""
ArcVault Unified Application

One application that does everything:
- GUI mode (default): Manage certificates, JWT keys, service installation
- Service mode: Run as Windows service
- CLI mode: Generate certs/JWT from command line
- Service management: Install/uninstall/start/stop itself

Usage:
    arcvault.exe                          # Launch GUI
    arcvault.exe install                  # Install as Windows service
    arcvault.exe start                    # Start service
    arcvault.exe stop                     # Stop service
    arcvault.exe remove                   # Uninstall service
    arcvault.exe --gen-cert               # Generate SSL certificate
    arcvault.exe --gen-jwt                # Generate JWT keys
"""

import sys
import os
import argparse
from pathlib import Path

# Determine mode based on arguments
def get_mode():
    """Determine what mode to run in"""
    if len(sys.argv) > 1:
        arg = sys.argv[1].lower()
        
        # Windows service commands
        if arg in ['install', 'remove', 'start', 'stop', 'restart', 'update', 'debug']:
            return 'service_command'
        
        # CLI commands
        if arg.startswith('--gen-cert'):
            return 'gen_cert'
        if arg.startswith('--gen-jwt'):
            return 'gen_jwt'
        
        # Help
        if arg in ['-h', '--help', 'help']:
            return 'help'
    
    # Default to GUI
    return 'gui'


def show_help():
    """Show help message"""
    print(__doc__)
    sys.exit(0)


def run_gui():
    """Run the GUI application"""
    try:
        from PyQt5.QtWidgets import QApplication, QMainWindow, QMessageBox
        from PyQt5.QtCore import Qt
    except ImportError:
        print("PyQt5 not installed. Installing...")
        import subprocess
        subprocess.check_call([sys.executable, "-m", "pip", "install", "PyQt5"])
        from PyQt5.QtWidgets import QApplication, QMainWindow, QMessageBox
        from PyQt5.QtCore import Qt
    
    from gui.main_window import ArcVaultMainWindow
    
    app = QApplication(sys.argv)
    app.setApplicationName("ArcVault Management Suite")
    app.setApplicationVersion("3.0.0")
    
    window = ArcVaultMainWindow()
    window.show()
    
    sys.exit(app.exec_())


def run_service_command():
    """Handle Windows service installation/management commands"""
    try:
        import win32serviceutil
        from service.win_service import ArcVaultAgentService
        
        # Let pywin32 handle the command
        win32serviceutil.HandleCommandLine(ArcVaultAgentService)
    except ImportError:
        print("ERROR: Windows service dependencies not installed.")
        print("Install with: pip install pywin32")
        sys.exit(1)


def run_cert_generator():
    """Run certificate generation from CLI"""
    # Remove the --gen-cert flag before passing to argparse
    sys.argv = [sys.argv[0]] + [arg for arg in sys.argv[1:] if arg != '--gen-cert']
    
    from cli.cert_generator import main as cert_main
    cert_main()


def run_jwt_generator():
    """Run JWT key generation from CLI"""
    # Remove the --gen-jwt flag before passing to argparse
    sys.argv = [sys.argv[0]] + [arg for arg in sys.argv[1:] if arg != '--gen-jwt']
    
    from cli.jwt_generator import main as jwt_main
    jwt_main()


def main():
    """Main entry point - route to appropriate mode"""
    mode = get_mode()
    
    if mode == 'help':
        show_help()
    elif mode == 'gui':
        run_gui()
    elif mode == 'service_command':
        run_service_command()
    elif mode == 'gen_cert':
        run_cert_generator()
    elif mode == 'gen_jwt':
        run_jwt_generator()
    else:
        show_help()


if __name__ == "__main__":
    main()

